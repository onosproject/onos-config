// SPDX-FileCopyrightText: 2020-present Open Networking Foundation <info@opennetworking.org>
//
// SPDX-License-Identifier: Apache-2.0

package proposal

import (
	"context"
	"fmt"
	"strings"
	"time"

	topoapi "github.com/onosproject/onos-api/go/onos/topo"
	controllerutils "github.com/onosproject/onos-config/pkg/controller/utils"
	proposalstore "github.com/onosproject/onos-config/pkg/store/v2/proposal"
	pathutils "github.com/onosproject/onos-config/pkg/utils/path"
	"github.com/onosproject/onos-config/pkg/utils/v2/tree"
	utilsv2 "github.com/onosproject/onos-config/pkg/utils/v2/values"
	"github.com/openconfig/gnmi/proto/gnmi_ext"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/onosproject/onos-config/pkg/pluginregistry"

	"github.com/onosproject/onos-lib-go/pkg/errors"

	configapi "github.com/onosproject/onos-api/go/onos/config/v2"
	"github.com/onosproject/onos-config/pkg/southbound/gnmi"
	"github.com/onosproject/onos-config/pkg/store/topo"
	"github.com/onosproject/onos-config/pkg/store/v2/configuration"
	"github.com/onosproject/onos-lib-go/pkg/controller"
	"github.com/onosproject/onos-lib-go/pkg/logging"
	gpb "github.com/openconfig/gnmi/proto/gnmi"
)

var log = logging.GetLogger("controller", "proposal")

const (
	defaultTimeout = 30 * time.Second
)

// NewController returns a proposal controller
func NewController(topo topo.Store, conns gnmi.ConnManager, proposals proposalstore.Store, configurations configuration.Store, pluginRegistry pluginregistry.PluginRegistry) *controller.Controller {
	c := controller.NewController("proposal")
	c.Watch(&Watcher{
		proposals: proposals,
	})
	c.Watch(&ConfigurationWatcher{
		configurations: configurations,
	})
	c.Partition(&Partitioner{})
	c.Reconcile(&Reconciler{
		conns:          conns,
		topo:           topo,
		proposals:      proposals,
		configurations: configurations,
		pluginRegistry: pluginRegistry,
	})
	return c
}

// Partitioner is a proposal partitioner
type Partitioner struct{}

// Partition partitions proposals by target ID
func (p *Partitioner) Partition(id controller.ID) (controller.PartitionKey, error) {
	proposalID := string(id.Value.(configapi.ProposalID))
	targetID := proposalID[:strings.LastIndex(proposalID, "-")]
	return controller.PartitionKey(targetID), nil
}

// Reconciler reconciles proposals
type Reconciler struct {
	conns          gnmi.ConnManager
	topo           topo.Store
	proposals      proposalstore.Store
	configurations configuration.Store
	pluginRegistry pluginregistry.PluginRegistry
}

// Reconcile reconciles target proposals
func (r *Reconciler) Reconcile(id controller.ID) (controller.Result, error) {
	ctx, cancel := context.WithTimeout(context.Background(), defaultTimeout)
	defer cancel()

	proposalID := id.Value.(configapi.ProposalID)
	proposal, err := r.proposals.Get(ctx, proposalID)
	if err != nil {
		if !errors.IsNotFound(err) {
			log.Warnf("Failed to reconcile Proposal '%s'", proposalID, err)
			return controller.Result{}, err
		}
		log.Debugf("Proposal '%s' not found", proposalID)
		return controller.Result{}, nil
	}

	log.Debugf("Reconciling Proposal '%s'", proposal.ID)
	log.Debug(proposal)
	return r.reconcileProposal(ctx, proposal)
}

func (r *Reconciler) reconcileProposal(ctx context.Context, proposal *configapi.Proposal) (controller.Result, error) {
	if proposal.Status.Phases.Apply != nil {
		return r.reconcileApply(ctx, proposal)
	} else if proposal.Status.Phases.Abort != nil {
		return r.reconcileAbort(ctx, proposal)
	} else if proposal.Status.Phases.Commit != nil {
		return r.reconcileCommit(ctx, proposal)
	} else if proposal.Status.Phases.Validate != nil {
		return r.reconcileValidate(ctx, proposal)
	} else if proposal.Status.Phases.Initialize != nil {
		return r.reconcileInitialize(ctx, proposal)
	} else {
		log.Infof("Initializing Proposal '%s'", proposal.ID)
		proposal.Status.Phases.Initialize = &configapi.ProposalInitializePhase{
			ProposalPhaseStatus: configapi.ProposalPhaseStatus{
				Start: getCurrentTimestamp(),
			},
		}
		if err := r.updateProposalStatus(ctx, proposal); err != nil {
			return controller.Result{}, err
		}
		return controller.Result{}, nil
	}
}

func (r *Reconciler) reconcileInitialize(ctx context.Context, proposal *configapi.Proposal) (controller.Result, error) {
	switch proposal.Status.Phases.Initialize.State {
	case configapi.ProposalInitializePhase_INITIALIZING:
		log.Infof("Initializing Transaction %d Proposal to target '%s'", proposal.TransactionIndex, proposal.TargetID)
		configID := configuration.NewID(proposal.TargetID, proposal.TargetType, proposal.TargetVersion)
		config, err := r.configurations.Get(ctx, configID)
		if err != nil {
			if !errors.IsNotFound(err) {
				log.Errorf("Failed reconciling Transaction %d Proposal to target '%s'", proposal.TransactionIndex, proposal.TargetID, err)
				return controller.Result{}, err
			}

			log.Infof("Creating Configuration for target '%s'", proposal.TargetID)
			config = &configapi.Configuration{
				ID:       configID,
				TargetID: proposal.TargetID,
				Status: configapi.ConfigurationStatus{
					Proposed: configapi.ProposedConfigurationStatus{
						Index: proposal.TransactionIndex,
					},
				},
			}
			err := r.configurations.Create(ctx, config)
			if err != nil {
				if !errors.IsAlreadyExists(err) {
					log.Errorf("Failed reconciling Transaction %d Proposal to target '%s'", proposal.TransactionIndex, proposal.TargetID, err)
					return controller.Result{}, err
				}
			}
			return controller.Result{
				Requeue: controller.NewID(proposal.ID),
			}, nil
		}

		if config.Status.Proposed.Index < proposal.TransactionIndex {
			if config.Status.Proposed.Index > 0 {
				prevProposalID := proposalstore.NewID(config.TargetID, config.Status.Proposed.Index)
				prevProposal, err := r.proposals.Get(ctx, prevProposalID)
				if err != nil {
					if !errors.IsNotFound(err) {
						log.Errorf("Failed reconciling Transaction %d Proposal to target '%s'", proposal.TransactionIndex, proposal.TargetID, err)
						return controller.Result{}, err
					}
				} else {
					log.Infof("Linking Transaction %d with Transaction %d", proposal.TransactionIndex, config.Status.Proposed.Index)
					if prevProposal.Status.NextIndex == 0 {
						prevProposal.Status.NextIndex = proposal.TransactionIndex
						if err := r.updateProposalStatus(ctx, prevProposal); err != nil {
							return controller.Result{}, err
						}
						return controller.Result{
							Requeue: controller.NewID(proposal.ID),
						}, nil
					}
					if proposal.Status.PrevIndex == 0 {
						proposal.Status.PrevIndex = config.Status.Proposed.Index
						if err := r.updateProposalStatus(ctx, proposal); err != nil {
							return controller.Result{}, err
						}
						return controller.Result{
							Requeue: controller.NewID(proposal.ID),
						}, nil
					}
				}
			}

			log.Infof("Updating Configuration '%s' status", proposal.TargetID)
			config.Status.Proposed.Index = proposal.TransactionIndex
			err := r.configurations.UpdateStatus(ctx, config)
			if err != nil {
				log.Warnf("Failed reconciling Transaction %d Proposal to target '%s'", proposal.TransactionIndex, proposal.TargetID, err)
				return controller.Result{}, err
			}
			return controller.Result{
				Requeue: controller.NewID(proposal.ID),
			}, nil
		}

		log.Infof("Transaction %d Proposal to target '%s' initialized", proposal.TransactionIndex, proposal.TargetID)
		proposal.Status.Phases.Initialize.State = configapi.ProposalInitializePhase_INITIALIZED
		proposal.Status.Phases.Initialize.End = getCurrentTimestamp()
		if err := r.updateProposalStatus(ctx, proposal); err != nil {
			return controller.Result{}, err
		}
		return controller.Result{}, nil
	default:
		return controller.Result{}, nil
	}
}

func (r *Reconciler) reconcileValidate(ctx context.Context, proposal *configapi.Proposal) (controller.Result, error) {
	switch proposal.Status.Phases.Validate.State {
	case configapi.ProposalValidatePhase_VALIDATING:
		configID := configuration.NewID(proposal.TargetID, proposal.TargetType, proposal.TargetVersion)
		config, err := r.configurations.Get(ctx, configID)
		if err != nil {
			if !errors.IsNotFound(err) {
				log.Errorf("Failed reconciling Transaction %d Proposal to target '%s'", proposal.TransactionIndex, proposal.TargetID, err)
				return controller.Result{}, err
			}
			return controller.Result{}, nil
		}

		// If the previous proposal has not yet been committed, wait for it.
		if proposal.Status.PrevIndex != 0 && config.Status.Committed.Index != proposal.Status.PrevIndex {
			log.Infof("Transaction %d Proposal to target '%s' waiting for Transaction %d Proposal to be committed", proposal.TransactionIndex, proposal.TargetID, proposal.Status.PrevIndex)
			return controller.Result{Requeue: controller.NewID(proposalstore.NewID(proposal.TargetID, proposal.Status.PrevIndex))}, nil
		}

		log.Infof("Validating Transaction %d Proposal to target '%s'", proposal.TransactionIndex, proposal.TargetID)

		var rollbackIndex configapi.Index
		var rollbackValues map[string]*configapi.PathValue

		changeValues := make(map[string]*configapi.PathValue)
		if config.Values != nil {
			for path, pathValue := range config.Values {
				changeValues[path] = pathValue
			}
		}

		modelPlugin, ok := r.pluginRegistry.GetPlugin(proposal.TargetType, proposal.TargetVersion)
		if !ok {
			proposal.Status.Phases.Validate.State = configapi.ProposalValidatePhase_FAILED
			proposal.Status.Phases.Validate.Failure = &configapi.Failure{
				Type:        configapi.Failure_INVALID,
				Description: fmt.Sprintf("model plugin '%s/%s' not found", proposal.TargetType, proposal.TargetVersion),
			}
			proposal.Status.Phases.Validate.End = getCurrentTimestamp()
			if err := r.updateProposalStatus(ctx, proposal); err != nil {
				return controller.Result{}, err
			}
			return controller.Result{}, nil
		}

		switch details := proposal.Details.(type) {
		case *configapi.Proposal_Change:
			rollbackIndex = config.Index
			rollbackValues = make(map[string]*configapi.PathValue)
			for path, changeValue := range details.Change.Values {
				deletedParentPath, deletedParentValue := applyChangeToConfig(changeValues, path, changeValue)
				if deletedParentValue != nil {
					rollbackValues[deletedParentPath] = deletedParentValue
				}
				if configValue, ok := config.Values[path]; ok {
					rollbackValues[path] = configValue
				} else {
					rollbackValues[path] = &configapi.PathValue{
						Path:    path,
						Deleted: true,
					}
				}
			}
		case *configapi.Proposal_Rollback:
			if config.Index != details.Rollback.RollbackIndex {
				err := errors.NewForbidden("proposal %d is not the latest change to target '%s'", details.Rollback.RollbackIndex, proposal.TargetID)
				log.Warnf("Transaction %d Proposal to target '%s' is invalid", proposal.TransactionIndex, proposal.TargetID, err)
				proposal.Status.Phases.Validate.State = configapi.ProposalValidatePhase_FAILED
				proposal.Status.Phases.Validate.Failure = &configapi.Failure{
					Type:        configapi.Failure_FORBIDDEN,
					Description: err.Error(),
				}
				proposal.Status.Phases.Validate.End = getCurrentTimestamp()
				if err := r.updateProposalStatus(ctx, proposal); err != nil {
					return controller.Result{}, err
				}
				return controller.Result{}, nil
			}

			targetProposalID := proposalstore.NewID(proposal.TargetID, details.Rollback.RollbackIndex)
			targetProposal, err := r.proposals.Get(ctx, targetProposalID)
			if err != nil {
				if !errors.IsNotFound(err) {
					log.Errorf("Failed reconciling Transaction %d Proposal to target '%s'", proposal.TransactionIndex, proposal.TargetID, err)
					return controller.Result{}, err
				}
				err := errors.NewForbidden("proposal %d not found for target '%s'", details.Rollback.RollbackIndex, proposal.TargetID)
				log.Warnf("Transaction %d Proposal to target '%s' is invalid", proposal.TransactionIndex, proposal.TargetID, err)
				proposal.Status.Phases.Validate.State = configapi.ProposalValidatePhase_FAILED
				proposal.Status.Phases.Validate.Failure = &configapi.Failure{
					Type:        configapi.Failure_NOT_FOUND,
					Description: err.Error(),
				}
				proposal.Status.Phases.Validate.End = getCurrentTimestamp()
				if err := r.updateProposalStatus(ctx, proposal); err != nil {
					return controller.Result{}, err
				}
				return controller.Result{}, nil
			}

			switch targetProposal.Details.(type) {
			case *configapi.Proposal_Change:
				for path, rollbackValue := range targetProposal.Status.RollbackValues {
					changeValues[path] = rollbackValue
				}
				rollbackIndex = targetProposal.Status.RollbackIndex
				rollbackValues = targetProposal.Status.RollbackValues
			case *configapi.Proposal_Rollback:
				err := errors.NewForbidden("proposal %d is not a valid change to target '%s'", details.Rollback.RollbackIndex, proposal.TargetID)
				log.Warnf("Transaction %d Proposal to target '%s' is invalid", proposal.TransactionIndex, proposal.TargetID, err)
				proposal.Status.Phases.Validate.State = configapi.ProposalValidatePhase_FAILED
				proposal.Status.Phases.Validate.Failure = &configapi.Failure{
					Type:        configapi.Failure_FORBIDDEN,
					Description: err.Error(),
				}
				proposal.Status.Phases.Validate.End = getCurrentTimestamp()
				if err := r.updateProposalStatus(ctx, proposal); err != nil {
					return controller.Result{}, err
				}
				return controller.Result{}, nil
			}
		}

		values := make([]*configapi.PathValue, 0, len(changeValues))
		for _, changeValue := range changeValues {
			values = append(values, changeValue)
		}

		jsonTree, err := tree.BuildTree(values, true)
		if err != nil {
			return controller.Result{}, err
		}

		// If validation fails any target, mark the Proposal FAILED.
		err = modelPlugin.Validate(ctx, jsonTree)
		if err != nil {
			log.Warnf("Failed validating Proposal '%s'", proposal.ID, err)
			proposal.Status.Phases.Validate.State = configapi.ProposalValidatePhase_FAILED
			proposal.Status.Phases.Validate.Failure = &configapi.Failure{
				Type:        configapi.Failure_INVALID,
				Description: err.Error(),
			}
			proposal.Status.Phases.Validate.End = getCurrentTimestamp()
			if err := r.updateProposalStatus(ctx, proposal); err != nil {
				return controller.Result{}, err
			}
			return controller.Result{}, nil
		}

		// If validation is successful, mark the Proposal VALIDATED.
		log.Infof("Transaction %d Proposal to target '%s' validated", proposal.TransactionIndex, proposal.TargetID)
		proposal.Status.RollbackIndex = rollbackIndex
		proposal.Status.RollbackValues = rollbackValues
		proposal.Status.Phases.Validate.State = configapi.ProposalValidatePhase_VALIDATED
		proposal.Status.Phases.Validate.End = getCurrentTimestamp()
		if err := r.updateProposalStatus(ctx, proposal); err != nil {
			return controller.Result{}, err
		}
		return controller.Result{}, nil
	default:
		return controller.Result{}, nil
	}
}

func (r *Reconciler) reconcileAbort(ctx context.Context, proposal *configapi.Proposal) (controller.Result, error) {
	switch proposal.Status.Phases.Abort.State {
	case configapi.ProposalAbortPhase_ABORTING:
		configID := configuration.NewID(proposal.TargetID, proposal.TargetType, proposal.TargetVersion)
		config, err := r.configurations.Get(ctx, configID)
		if err != nil {
			if !errors.IsNotFound(err) {
				log.Errorf("Failed reconciling Transaction %d Proposal to target '%s'", proposal.TransactionIndex, proposal.TargetID, err)
				return controller.Result{}, err
			}
			return controller.Result{}, nil
		}

		if config.Status.Committed.Index == proposal.Status.PrevIndex &&
			config.Status.Applied.Index == proposal.Status.PrevIndex {
			config.Status.Committed.Index = proposal.TransactionIndex
			config.Status.Applied.Index = proposal.TransactionIndex
			if err := r.configurations.UpdateStatus(ctx, config); err != nil {
				log.Warnf("Failed reconciling Transaction %d Proposal to target '%s'", proposal.TransactionIndex, proposal.TargetID, err)
				return controller.Result{}, err
			}
			proposal.Status.Phases.Abort.End = getCurrentTimestamp()
			proposal.Status.Phases.Abort.State = configapi.ProposalAbortPhase_ABORTED
			if err := r.updateProposalStatus(ctx, proposal); err != nil {
				return controller.Result{}, err
			}
		} else if config.Status.Committed.Index == proposal.Status.PrevIndex {
			config.Status.Committed.Index = proposal.TransactionIndex
			if err := r.configurations.UpdateStatus(ctx, config); err != nil {
				log.Warnf("Failed reconciling Transaction %d Proposal to target '%s'", proposal.TransactionIndex, proposal.TargetID, err)
				return controller.Result{}, err
			}
		} else if config.Status.Applied.Index == proposal.Status.PrevIndex &&
			config.Status.Committed.Index >= proposal.TransactionIndex {
			config.Status.Committed.Index = proposal.TransactionIndex
			if err := r.configurations.UpdateStatus(ctx, config); err != nil {
				log.Warnf("Failed reconciling Transaction %d Proposal to target '%s'", proposal.TransactionIndex, proposal.TargetID, err)
				return controller.Result{}, err
			}
			proposal.Status.Phases.Abort.End = getCurrentTimestamp()
			proposal.Status.Phases.Abort.State = configapi.ProposalAbortPhase_ABORTED
			if err := r.updateProposalStatus(ctx, proposal); err != nil {
				return controller.Result{}, err
			}
			return controller.Result{}, nil
		}

	}
	return controller.Result{}, nil
}

func (r *Reconciler) reconcileCommit(ctx context.Context, proposal *configapi.Proposal) (controller.Result, error) {
	switch proposal.Status.Phases.Commit.State {
	case configapi.ProposalCommitPhase_COMMITTING:
		configID := configuration.NewID(proposal.TargetID, proposal.TargetType, proposal.TargetVersion)
		config, err := r.configurations.Get(ctx, configID)
		if err != nil {
			if !errors.IsNotFound(err) {
				log.Errorf("Failed reconciling Transaction %d Proposal to target '%s'", proposal.TransactionIndex, proposal.TargetID, err)
				return controller.Result{}, err
			}
			return controller.Result{}, nil
		}

		if config.Status.Committed.Index == proposal.Status.PrevIndex {
			var changeValues map[string]*configapi.PathValue
			switch details := proposal.Details.(type) {
			case *configapi.Proposal_Change:
				config.Index = proposal.TransactionIndex
				changeValues = details.Change.Values
			case *configapi.Proposal_Rollback:
				config.Index = proposal.Status.RollbackIndex
				changeValues = proposal.Status.RollbackValues
			}
			log.Infof("Committing %d changes at index %d to Configuration '%s'", len(changeValues), config.Index, config.ID)
			if config.Values == nil {
				config.Values = make(map[string]*configapi.PathValue)
			}
			updatedChangeValues := controllerutils.AddDeleteChildren(proposal.TransactionIndex, changeValues, config.Values)
			for path, updatedChangeValue := range updatedChangeValues {
				_, _ = applyChangeToConfig(config.Values, path, updatedChangeValue)
			}
			config.Status.Committed.Index = proposal.TransactionIndex
			err = r.configurations.Update(ctx, config)
			if err != nil {
				log.Warnf("Failed reconciling Transaction %d Proposal to target '%s'", proposal.TransactionIndex, proposal.TargetID, err)
				return controller.Result{}, err
			}
		}

		log.Infof("Committed Proposal '%s'", proposal.ID)
		proposal.Status.Phases.Commit.State = configapi.ProposalCommitPhase_COMMITTED
		proposal.Status.Phases.Commit.End = getCurrentTimestamp()
		if err := r.updateProposalStatus(ctx, proposal); err != nil {
			return controller.Result{}, err
		}
		return controller.Result{}, nil
	case configapi.ProposalCommitPhase_COMMITTED:
		if proposal.Status.NextIndex != 0 {
			return controller.Result{
				Requeue: controller.NewID(proposalstore.NewID(proposal.TargetID, proposal.Status.NextIndex)),
			}, nil
		}
		return controller.Result{}, nil
	default:
		return controller.Result{}, nil
	}
}

func applyChangeToConfig(values map[string]*configapi.PathValue, path string, value *configapi.PathValue) (string, *configapi.PathValue) {
	values[path] = value

	// Walk up the path and make sure that there are no parents marked as deleted in the given map, if so, remove them
	parent := pathutils.GetParentPath(path)
	for parent != "" {
		if v := values[parent]; v != nil && v.Deleted {
			// Delete the parent marked as deleted and return its path and value
			delete(values, parent)
			return parent, v
		}
		parent = pathutils.GetParentPath(parent)
	}
	return "", nil
}

func (r *Reconciler) reconcileApply(ctx context.Context, proposal *configapi.Proposal) (controller.Result, error) {
	switch proposal.Status.Phases.Apply.State {
	case configapi.ProposalApplyPhase_APPLYING:
		log.Infof("Applying Transaction %d Proposal to target '%s'", proposal.TransactionIndex, proposal.TargetID)
		configID := configuration.NewID(proposal.TargetID, proposal.TargetType, proposal.TargetVersion)
		config, err := r.configurations.Get(ctx, configID)
		if err != nil {
			if !errors.IsNotFound(err) {
				log.Errorf("Failed reconciling Transaction %d Proposal to target '%s'", proposal.TransactionIndex, proposal.TargetID, err)
				return controller.Result{}, err
			}
			return controller.Result{}, nil
		}

		// If the proposal has already been committed to the configuration, update the proposal state and return.
		if config.Status.Applied.Index >= proposal.TransactionIndex {
			log.Infof("Applied Proposal '%s'", proposal.ID)
			proposal.Status.Phases.Apply.State = configapi.ProposalApplyPhase_APPLIED
			proposal.Status.Phases.Apply.Term = config.Status.Applied.Mastership.Term
			proposal.Status.Phases.Apply.End = getCurrentTimestamp()
			if err := r.updateProposalStatus(ctx, proposal); err != nil {
				return controller.Result{}, err
			}
			return controller.Result{}, nil
		}

		// If the previous proposal has not yet been applied, wait for it.
		if proposal.Status.PrevIndex != 0 && config.Status.Applied.Index != proposal.Status.PrevIndex {
			log.Infof("Transaction %d Proposal to target '%s' waiting for Transaction %d Proposal to be applied", proposal.TransactionIndex, proposal.TargetID, proposal.Status.PrevIndex)
			return controller.Result{Requeue: controller.NewID(proposalstore.NewID(proposal.TargetID, proposal.Status.PrevIndex))}, nil
		}

		// If the configuration is synchronizing, wait for it to complete.
		if config.Status.State == configapi.ConfigurationStatus_SYNCHRONIZING {
			log.Infof("Waiting for synchronization of Configuration to target '%s'", proposal.TargetID)
			return controller.Result{}, nil
		}

		// Get the target entity from topo
		target, err := r.topo.Get(ctx, topoapi.ID(proposal.TargetID))
		if err != nil {
			if !errors.IsNotFound(err) {
				log.Errorf("Failed fetching target Entity '%s' from topo", proposal.TargetID, err)
				return controller.Result{}, err
			}
			log.Debugf("Target entity '%s' not found", proposal.TargetID)
			return controller.Result{}, nil
		}

		// If the configuration is in an old term, wait for synchronization.
		if config.Status.Applied.Mastership.Term < config.Status.Mastership.Term {
			log.Infof("Waiting for synchronization of Configuration to target '%s'", proposal.TargetID)
			return controller.Result{}, nil
		}

		// If the master node ID is not set, skip reconciliation.
		if config.Status.Mastership.Master == "" {
			log.Debugf("No master for target '%s'", proposal.TargetID)
			return controller.Result{}, nil
		}

		// If we've made it this far, we know there's a master relation.
		// Get the relation and check whether this node is the source
		relation, err := r.topo.Get(ctx, topoapi.ID(config.Status.Mastership.Master))
		if err != nil {
			if !errors.IsNotFound(err) {
				log.Errorf("Failed fetching master Relation '%s' from topo", config.Status.Mastership.Master, err)
				return controller.Result{}, err
			}
			log.Warnf("Master relation not found for target '%s'", proposal.TargetID)
			return controller.Result{}, nil
		}
		if relation.GetRelation().SrcEntityID != controllerutils.GetOnosConfigID() {
			log.Debugf("Not the master for target '%s'", proposal.TargetID)
			return controller.Result{}, nil
		}

		// Get the master connection
		conn, ok := r.conns.Get(ctx, gnmi.ConnID(relation.ID))
		if !ok {
			log.Warnf("Connection not found for target '%s'", proposal.TargetID)
			return controller.Result{}, nil
		}

		configurable := &topoapi.Configurable{}
		_ = target.GetAspect(configurable)

		if configurable.ValidateCapabilities {
			capabilityResponse, err := conn.Capabilities(ctx, &gpb.CapabilityRequest{})
			if err != nil {
				log.Warnf("Cannot retrieve capabilities of the target %s", proposal.TargetID)
				return controller.Result{}, err
			}
			targetDataModels := capabilityResponse.SupportedModels
			modelPlugin, ok := r.pluginRegistry.GetPlugin(proposal.TargetType, proposal.TargetVersion)
			if !ok {
				proposal.Status.Phases.Apply.State = configapi.ProposalApplyPhase_FAILED
				proposal.Status.Phases.Apply.Failure = &configapi.Failure{
					Type:        configapi.Failure_INVALID,
					Description: fmt.Sprintf("model plugin '%s/%s' not found", proposal.TargetType, proposal.TargetVersion),
				}
				proposal.Status.Phases.Apply.End = getCurrentTimestamp()
				if err := r.updateProposalStatus(ctx, proposal); err != nil {
					return controller.Result{}, err
				}
				return controller.Result{}, nil
			}
			pluginCapability := modelPlugin.Capabilities(ctx)
			pluginDataModels := pluginCapability.SupportedModels

			if !isModelDataCompatible(pluginDataModels, targetDataModels) {
				err = errors.NewNotSupported("plugin data models %v are not supported in the target %s", pluginDataModels, proposal.TargetID)
				log.Warn(err)
				return controller.Result{}, err
			}
		}

		// Get the set of changes. If the Proposal is a change, use the change values.
		// If the proposal is a rollback, use the rollback values.
		var changeValues map[string]*configapi.PathValue
		switch details := proposal.Details.(type) {
		case *configapi.Proposal_Change:
			changeValues = details.Change.Values
		case *configapi.Proposal_Rollback:
			changeValues = proposal.Status.RollbackValues
		}

		updatedChangeValues := controllerutils.AddDeleteChildren(proposal.TransactionIndex, changeValues, config.Values)
		// Create a list of PathValue pairs from which to construct a gNMI Set for the Proposal.
		pathValues := make([]*configapi.PathValue, 0, len(updatedChangeValues))
		for _, changeValue := range updatedChangeValues {
			pathValues = append(pathValues, changeValue)
		}
		pathValues = tree.PrunePathValues(pathValues, true)

		log.Infof("Updating %d paths on target '%s'", len(pathValues), config.TargetID)

		// Create a gNMI set request
		setRequest, err := utilsv2.PathValuesToGnmiChange(pathValues, proposal.TargetID)
		if err != nil {
			log.Errorf("Failed constructing SetRequest for Configuration '%s'", config.ID, err)
			return controller.Result{}, nil
		}

		// Add the master arbitration extension to provide concurrency control for multi-node controllers.
		setRequest.Extension = append(setRequest.Extension, &gnmi_ext.Extension{
			Ext: &gnmi_ext.Extension_MasterArbitration{
				MasterArbitration: &gnmi_ext.MasterArbitration{
					Role: &gnmi_ext.Role{
						Id: "onos-config",
					},
					ElectionId: &gnmi_ext.Uint128{
						Low: uint64(config.Status.Mastership.Term),
					},
				},
			},
		})

		// Execute the set request
		log.Debugf("Sending SetRequest %+v", setRequest)
		setResponse, err := conn.Set(ctx, setRequest)
		if err != nil {
			code := status.Code(err)
			switch code {
			case codes.Unavailable, codes.Canceled, codes.DeadlineExceeded:
				log.Errorf("Failed sending SetRequest %+v", setRequest, err)
				return controller.Result{}, err
			case codes.PermissionDenied:
				// The gNMI Set request can be denied if this master has been superseded by a master in a later term.
				// Rather than reverting to the STALE state now, wait for this node to see the mastership state change
				// to avoid flapping between states while the system converges.
				log.Warnf("Configuration '%s' mastership superseded for term %d", config.ID, config.Status.Mastership.Term)
				return controller.Result{}, nil
			default:
				var failureType configapi.Failure_Type
				switch code {
				case codes.Unknown:
					failureType = configapi.Failure_UNKNOWN
				case codes.Canceled:
					failureType = configapi.Failure_CANCELED
				case codes.NotFound:
					failureType = configapi.Failure_NOT_FOUND
				case codes.AlreadyExists:
					failureType = configapi.Failure_ALREADY_EXISTS
				case codes.Unauthenticated:
					failureType = configapi.Failure_UNAUTHORIZED
				case codes.PermissionDenied:
					failureType = configapi.Failure_FORBIDDEN
				case codes.FailedPrecondition:
					failureType = configapi.Failure_CONFLICT
				case codes.InvalidArgument:
					failureType = configapi.Failure_INVALID
				case codes.Unavailable:
					failureType = configapi.Failure_UNAVAILABLE
				case codes.Unimplemented:
					failureType = configapi.Failure_NOT_SUPPORTED
				case codes.DeadlineExceeded:
					failureType = configapi.Failure_TIMEOUT
				case codes.Internal:
					failureType = configapi.Failure_INTERNAL
				}

				// Update the Configuration's applied index to indicate this Proposal was applied even though it failed.
				log.Infof("Updating applied index for Configuration '%s' to %d in term %d", config.ID, proposal.TransactionIndex, config.Status.Mastership.Term)
				config.Status.Applied.Index = proposal.TransactionIndex
				if err := r.configurations.UpdateStatus(ctx, config); err != nil {
					log.Warnf("Failed reconciling Transaction %d Proposal to target '%s'", proposal.TransactionIndex, proposal.TargetID, err)
					return controller.Result{}, err
				}

				// Add the failure to the proposal's apply phase state.
				log.Warnf("Failed applying Proposal '%s'", proposal.ID, err)
				proposal.Status.Phases.Apply.State = configapi.ProposalApplyPhase_FAILED
				proposal.Status.Phases.Apply.Failure = &configapi.Failure{
					Type:        failureType,
					Description: err.Error(),
				}
				proposal.Status.Phases.Apply.Term = config.Status.Mastership.Term
				proposal.Status.Phases.Apply.End = getCurrentTimestamp()
				if err := r.updateProposalStatus(ctx, proposal); err != nil {
					return controller.Result{}, err
				}
				return controller.Result{}, nil
			}
		}
		log.Debugf("Received SetResponse %+v", setResponse)

		// Update the Configuration's applied index to indicate this Proposal was applied.
		log.Infof("Updating applied index for Configuration '%s' to %d in term %d", config.ID, proposal.TransactionIndex, config.Status.Mastership.Term)
		config.Status.Applied.Index = proposal.TransactionIndex
		if config.Status.Applied.Values == nil {
			config.Status.Applied.Values = make(map[string]*configapi.PathValue)
		}
		for path, changeValue := range updatedChangeValues {
			config.Status.Applied.Values[path] = changeValue
		}

		if err := r.configurations.UpdateStatus(ctx, config); err != nil {
			log.Warnf("Failed reconciling Transaction %d Proposal to target '%s'", proposal.TransactionIndex, proposal.TargetID, err)
			return controller.Result{}, err
		}

		// Update the proposal state to APPLIED.
		log.Infof("Applied Proposal '%s'", proposal.ID)
		proposal.Status.Phases.Apply.State = configapi.ProposalApplyPhase_APPLIED
		proposal.Status.Phases.Apply.Term = config.Status.Mastership.Term
		proposal.Status.Phases.Apply.End = getCurrentTimestamp()
		if err := r.updateProposalStatus(ctx, proposal); err != nil {
			return controller.Result{}, err
		}
		return controller.Result{}, nil
	case configapi.ProposalApplyPhase_APPLIED:
		if proposal.Status.NextIndex != 0 {
			return controller.Result{
				Requeue: controller.NewID(proposalstore.NewID(proposal.TargetID, proposal.Status.NextIndex)),
			}, nil
		}
		return controller.Result{}, nil
	default:
		return controller.Result{}, nil
	}
}

func (r *Reconciler) updateProposalStatus(ctx context.Context, proposal *configapi.Proposal) error {
	log.Debug(proposal.Status)
	err := r.proposals.UpdateStatus(ctx, proposal)
	if err != nil {
		if !errors.IsNotFound(err) && !errors.IsConflict(err) {
			log.Errorf("Failed updating Proposal '%s' status", proposal.ID, err)
			return err
		}
		log.Warnf("Write conflict updating Proposal '%s' status", proposal.ID, err)
		return nil
	}
	return nil
}

func getCurrentTimestamp() *time.Time {
	t := time.Now()
	return &t
}

func isModelDataCompatible(pluginDataModels []*gpb.ModelData, targetDataModels []*gpb.ModelData) bool {
	if len(pluginDataModels) > len(targetDataModels) {
		return false
	}
	for _, pluginDataModel := range pluginDataModels {
		modelDataCompatible := false
		for _, targetDataModel := range targetDataModels {
			if pluginDataModel.Name == targetDataModel.Name && pluginDataModel.Version == targetDataModel.Version && pluginDataModel.Organization == targetDataModel.Organization {
				modelDataCompatible = true
			}
		}
		if !modelDataCompatible {
			return false
		}
	}
	return true
}
