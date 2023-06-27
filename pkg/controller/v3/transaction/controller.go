// SPDX-FileCopyrightText: 2020-present Open Networking Foundation <info@opennetworking.org>
//
// SPDX-License-Identifier: Apache-2.0

package transaction

import (
	"context"
	"fmt"
	configv2 "github.com/onosproject/onos-api/go/onos/config/v2"
	topoapi "github.com/onosproject/onos-api/go/onos/topo"
	"github.com/onosproject/onos-config/pkg/pluginregistry"
	"github.com/onosproject/onos-config/pkg/southbound/gnmi"
	"github.com/onosproject/onos-config/pkg/store/topo"
	configurationstore "github.com/onosproject/onos-config/pkg/store/v3/configuration"
	pathutils "github.com/onosproject/onos-config/pkg/utils/path"
	"github.com/onosproject/onos-config/pkg/utils/v3/tree"
	valuesutil "github.com/onosproject/onos-config/pkg/utils/v3/values"
	gpb "github.com/openconfig/gnmi/proto/gnmi"
	"github.com/openconfig/gnmi/proto/gnmi_ext"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"strings"
	"time"

	transactionstore "github.com/onosproject/onos-config/pkg/store/v3/transaction"

	"github.com/onosproject/onos-lib-go/pkg/errors"

	configapi "github.com/onosproject/onos-api/go/onos/config/v3"
	"github.com/onosproject/onos-lib-go/pkg/controller"
	"github.com/onosproject/onos-lib-go/pkg/logging"
)

var log = logging.GetLogger("controller", "transaction")

const (
	defaultTimeout = 30 * time.Second
)

// NewController returns a transaction controller
func NewController(nodeID configapi.NodeID, transactions transactionstore.Store, configurations configurationstore.Store,
	conns gnmi.ConnManager, topo topo.Store, plugins pluginregistry.PluginRegistry) *controller.Controller {
	c := controller.NewController("transaction")
	c.Watch(&Watcher{
		transactions: transactions,
	})
	c.Watch(&ConfigurationWatcher{
		configurations: configurations,
	})
	c.Reconcile(&Reconciler{
		nodeID:         nodeID,
		transactions:   transactions,
		configurations: configurations,
		conns:          conns,
		topo:           topo,
		plugins:        plugins,
	})
	return c
}

// Reconciler reconciles transactions
type Reconciler struct {
	nodeID         configapi.NodeID
	transactions   transactionstore.Store
	configurations configurationstore.Store
	conns          gnmi.ConnManager
	topo           topo.Store
	plugins        pluginregistry.PluginRegistry
}

// Reconcile reconciles target transactions
func (r *Reconciler) Reconcile(id controller.ID) (controller.Result, error) {
	ctx, cancel := context.WithTimeout(context.Background(), defaultTimeout)
	defer cancel()

	transactionID := id.Value.(configapi.TransactionID)
	transaction, err := r.transactions.Get(ctx, transactionID)
	if err != nil {
		if !errors.IsNotFound(err) {
			log.Warnf("Failed to reconcile Transaction %s", transactionID, err)
			return controller.Result{}, err
		}
		log.Debugf("Transaction %s not found", transactionID)
		return controller.Result{}, nil
	}

	configurationID := configapi.ConfigurationID{
		Target: transactionID.Target,
	}
	configuration, err := r.configurations.Get(ctx, configurationID)
	if err != nil {
		if !errors.IsNotFound(err) {
			log.Errorf("Failed reconciling Transaction %s to target '%s'", transactionID.Index, transactionID.Target, err)
			return controller.Result{}, err
		}
		return controller.Result{}, nil
	}

	log.Debugf("Reconciling Transaction %s", transactionID)
	log.Debug(transaction)
	return r.reconcileTransaction(ctx, transaction, configuration)
}

func (r *Reconciler) reconcileTransaction(ctx context.Context, transaction *configapi.Transaction, configuration *configapi.Configuration) (controller.Result, error) {
	switch transaction.Status.Phase {
	case configapi.TransactionStatus_CHANGE:
		if result, ok, err := r.reconcileChange(ctx, transaction, configuration); err != nil {
			return controller.Result{}, err
		} else if ok {
			return result, nil
		}
	case configapi.TransactionStatus_ROLLBACK:
		if result, ok, err := r.reconcileRollback(ctx, transaction, configuration); err != nil {
			return controller.Result{}, err
		} else if ok {
			return result, nil
		}
	}
	return controller.Result{}, nil
}

func (r *Reconciler) reconcileChange(ctx context.Context, transaction *configapi.Transaction, configuration *configapi.Configuration) (controller.Result, bool, error) {
	if transaction.Status.Phase != configapi.TransactionStatus_CHANGE {
		return controller.Result{}, false, nil
	}
	if result, ok, err := r.commitChange(ctx, transaction, configuration); err != nil {
		return controller.Result{}, false, err
	} else if ok {
		return result, true, nil
	}
	if result, ok, err := r.applyChange(ctx, transaction, configuration); err != nil {
		return controller.Result{}, false, err
	} else if ok {
		return result, true, nil
	}
	return controller.Result{}, false, nil
}

func (r *Reconciler) commitChange(ctx context.Context, transaction *configapi.Transaction, configuration *configapi.Configuration) (controller.Result, bool, error) {
	if transaction.Status.Change.Commit == nil {
		return controller.Result{}, false, nil
	}
	switch transaction.Status.Change.Commit.State {
	case configapi.TransactionPhaseStatus_PENDING:
		if configuration.Committed.Change != transaction.ID.Index-1 {
			return controller.Result{}, false, nil
		}

		if configuration.Committed.Target != transaction.ID.Index {
			if configuration.Committed.Index != configuration.Committed.Target {
				return controller.Result{}, false, nil
			}

			prevTransactionID := configapi.TransactionID{
				Target: transaction.ID.Target,
				Index:  configuration.Committed.Index,
			}
			prevTransaction, err := r.transactions.Get(ctx, prevTransactionID)
			if err != nil {
				if !errors.IsNotFound(err) {
					return controller.Result{}, false, err
				}
			} else if configuration.Committed.Target == configuration.Committed.Index &&
				prevTransaction.Status.Change.Commit.State <= configapi.TransactionPhaseStatus_IN_PROGRESS {
				return controller.Result{}, false, nil
			} else if configuration.Committed.Target < configuration.Committed.Index &&
				prevTransaction.Status.Rollback.Commit.State <= configapi.TransactionPhaseStatus_IN_PROGRESS {
				return controller.Result{}, false, nil
			}

			configuration.Committed.Target = transaction.ID.Index
			if err := r.updateConfigurationStatus(ctx, configuration); err != nil {
				return controller.Result{}, false, err
			}
		}

		changeValues := make(map[string]configapi.PathValue)
		if configuration.Committed.Values != nil {
			for path, pathValue := range configuration.Committed.Values {
				changeValues[path] = pathValue
			}
		}

		rollbackIndex := configapi.Index(configuration.Committed.Revision)
		rollbackValues := make(map[string]configapi.PathValue)
		for path, changeValue := range transaction.Values {
			deletedParentPath, ok, deletedParentValue := applyChangeToConfig(changeValues, path, changeValue)
			if ok {
				rollbackValues[deletedParentPath] = deletedParentValue
			}
			if configValue, ok := configuration.Committed.Values[path]; ok {
				rollbackValues[path] = configValue
			} else {
				rollbackValues[path] = configapi.PathValue{
					Path:    path,
					Deleted: true,
				}
			}
		}
		transaction.Status.Rollback.Index = rollbackIndex
		transaction.Status.Rollback.Values = rollbackValues

		transaction.Status.Change.Commit.Start = now()
		transaction.Status.Change.Commit.State = configapi.TransactionPhaseStatus_IN_PROGRESS
		if err := r.updateTransactionStatus(ctx, transaction); err != nil {
			return controller.Result{}, false, err
		}
		return controller.Result{}, true, nil
	case configapi.TransactionPhaseStatus_IN_PROGRESS:
		if configuration.Committed.Change == transaction.ID.Index {
			transaction.Status.Change.Commit.State = configapi.TransactionPhaseStatus_COMPLETE
			transaction.Status.Change.Ordinal = configuration.Committed.Ordinal
			transaction.Status.Change.Commit.End = now()
			if err := r.updateTransactionStatus(ctx, transaction); err != nil {
				return controller.Result{}, false, err
			}
			return controller.Result{
				Requeue: controller.NewID(configapi.TransactionID{
					Target: transaction.ID.Target,
					Index:  transaction.ID.Index + 1,
				}),
			}, true, nil
		}

		changeValues := make(map[string]configapi.PathValue)
		if configuration.Committed.Values != nil {
			for path, pathValue := range configuration.Committed.Values {
				changeValues[path] = pathValue
			}
		}

		for path, changeValue := range transaction.Values {
			_, _, _ = applyChangeToConfig(changeValues, path, changeValue)
		}

		values := make([]configapi.PathValue, 0, len(changeValues))
		for _, changeValue := range changeValues {
			values = append(values, changeValue)
		}

		jsonTree, err := tree.BuildTree(values, true)
		if err != nil {
			return controller.Result{}, false, err
		}

		modelPlugin, ok := r.plugins.GetPlugin(
			configv2.TargetType(transaction.ID.Target.Type),
			configv2.TargetVersion(transaction.ID.Target.Version))
		if !ok {
			log.Warnf("Failed validating target '%s' Transaction %d", transaction.ID.Target.ID, transaction.ID.Index, err)
			transaction.Status.Change.Commit.State = configapi.TransactionPhaseStatus_FAILED
			transaction.Status.Change.Commit.Failure = &configapi.Failure{
				Type:        configapi.Failure_INVALID,
				Description: fmt.Sprintf("model plugin '%s/%s' not found", transaction.ID.Target.Type, transaction.ID.Target.Version),
			}
			transaction.Status.Change.Commit.End = now()
			transaction.Status.Change.Apply.State = configapi.TransactionPhaseStatus_CANCELED
			if err := r.updateTransactionStatus(ctx, transaction); err != nil {
				return controller.Result{}, false, err
			}

			configuration.Committed.Index = transaction.ID.Index
			configuration.Committed.Change = transaction.ID.Index
			if err := r.updateConfigurationStatus(ctx, configuration); err != nil {
				return controller.Result{}, false, err
			}
			return controller.Result{}, true, nil
		}

		// If validation fails any target, mark the Proposal FAILED.
		err = modelPlugin.Validate(ctx, jsonTree)
		if err != nil {
			log.Warnf("Failed validating target '%s' Transaction %d", transaction.ID.Target.ID, transaction.ID.Index, err)
			transaction.Status.Change.Commit.State = configapi.TransactionPhaseStatus_FAILED
			transaction.Status.Change.Commit.Failure = &configapi.Failure{
				Type:        configapi.Failure_INVALID,
				Description: err.Error(),
			}
			transaction.Status.Change.Commit.End = now()
			transaction.Status.Change.Apply.State = configapi.TransactionPhaseStatus_CANCELED
			if err := r.updateTransactionStatus(ctx, transaction); err != nil {
				return controller.Result{}, false, err
			}

			configuration.Committed.Index = transaction.ID.Index
			configuration.Committed.Change = transaction.ID.Index
			if err := r.updateConfigurationStatus(ctx, configuration); err != nil {
				return controller.Result{}, false, err
			}
			return controller.Result{}, true, nil
		}

		// If validation is successful, mark the Proposal VALIDATED.
		log.Infof("Transaction %d to target '%s' validated", transaction.ID.Index, transaction.ID.Target.ID)

		configuration.Committed.Index = transaction.ID.Index
		configuration.Committed.Change = transaction.ID.Index
		configuration.Committed.Revision = configapi.Revision(transaction.ID.Index)
		configuration.Committed.Ordinal = configuration.Committed.Ordinal + 1
		for path, value := range transaction.Values {
			configuration.Committed.Values[path] = value
		}
		if err := r.updateConfigurationStatus(ctx, configuration); err != nil {
			return controller.Result{}, false, err
		}

		transaction.Status.Change.Commit.State = configapi.TransactionPhaseStatus_COMPLETE
		transaction.Status.Change.Ordinal = configuration.Committed.Ordinal
		transaction.Status.Change.Commit.End = now()
		if err := r.updateTransactionStatus(ctx, transaction); err != nil {
			return controller.Result{}, false, err
		}
		return controller.Result{
			Requeue: controller.NewID(configapi.TransactionID{
				Target: transaction.ID.Target,
				Index:  transaction.ID.Index + 1,
			}),
		}, true, nil
	case configapi.TransactionPhaseStatus_FAILED:
		if configuration.Committed.Change < transaction.ID.Index {
			configuration.Committed.Index = transaction.ID.Index
			configuration.Committed.Change = transaction.ID.Index
			if err := r.updateConfigurationStatus(ctx, configuration); err != nil {
				return controller.Result{}, false, err
			}
			return controller.Result{}, true, nil
		}
	}
	return controller.Result{}, false, nil
}

func (r *Reconciler) applyChange(ctx context.Context, transaction *configapi.Transaction, configuration *configapi.Configuration) (controller.Result, bool, error) {
	if transaction.Status.Change.Commit == nil || transaction.Status.Change.Apply == nil {
		return controller.Result{}, false, nil
	}
	if transaction.Status.Change.Commit.State != configapi.TransactionPhaseStatus_COMPLETE {
		return controller.Result{}, false, nil
	}
	switch transaction.Status.Change.Apply.State {
	case configapi.TransactionPhaseStatus_PENDING:
		if configuration.Applied.Ordinal != transaction.Status.Change.Ordinal-1 {
			return controller.Result{}, false, nil
		}

		// A failure occurred after updating the configuration target index but before the transaction status could be updated.
		if configuration.Applied.Target == transaction.ID.Index {
			transaction.Status.Change.Apply.State = configapi.TransactionPhaseStatus_IN_PROGRESS
			transaction.Status.Change.Apply.Start = now()
			if err := r.updateTransactionStatus(ctx, transaction); err != nil {
				return controller.Result{}, false, err
			}
			return controller.Result{}, true, nil
		}

		prevTransactionID := configapi.TransactionID{
			Target: transaction.ID.Target,
			Index:  configuration.Applied.Index,
		}
		prevTransaction, err := r.transactions.Get(ctx, prevTransactionID)
		if err != nil {
			if !errors.IsNotFound(err) {
				return controller.Result{}, false, err
			}
		} else if configuration.Applied.Target == configuration.Applied.Index &&
			prevTransaction.Status.Change.Apply.State <= configapi.TransactionPhaseStatus_IN_PROGRESS {
			return controller.Result{}, false, nil
		} else if configuration.Applied.Target < configuration.Applied.Index &&
			prevTransaction.Status.Rollback.Apply.State <= configapi.TransactionPhaseStatus_IN_PROGRESS {
			return controller.Result{}, false, nil
		}

		if configuration.Applied.Revision < configapi.Revision(transaction.Status.Rollback.Index) {
			transaction.Status.Change.Apply.State = configapi.TransactionPhaseStatus_ABORTED
			transaction.Status.Change.Apply.Start = now()
			transaction.Status.Change.Apply.End = now()
			if err := r.updateTransactionStatus(ctx, transaction); err != nil {
				return controller.Result{}, false, err
			}

			configuration.Applied.Target = transaction.ID.Index
			configuration.Applied.Index = transaction.ID.Index
			configuration.Applied.Ordinal = transaction.Status.Change.Ordinal
			if err := r.updateConfigurationStatus(ctx, configuration); err != nil {
				return controller.Result{}, false, err
			}
			return controller.Result{}, true, nil
		}

		configuration.Applied.Target = transaction.ID.Index
		if err := r.updateConfigurationStatus(ctx, configuration); err != nil {
			return controller.Result{}, false, err
		}

		transaction.Status.Change.Apply.State = configapi.TransactionPhaseStatus_IN_PROGRESS
		transaction.Status.Change.Apply.Start = now()
		if err := r.updateTransactionStatus(ctx, transaction); err != nil {
			return controller.Result{}, false, err
		}
		return controller.Result{}, true, nil
	case configapi.TransactionPhaseStatus_IN_PROGRESS:
		// A failure occurred between the configuration update and the transaction update.
		if configuration.Applied.Ordinal == transaction.Status.Change.Ordinal &&
			configuration.Applied.Revision == configapi.Revision(transaction.ID.Index) {
			transaction.Status.Change.Apply.State = configapi.TransactionPhaseStatus_COMPLETE
			transaction.Status.Change.Apply.End = now()
			log.Infof("Applied Transaction %d to '%s'", transaction.ID.Index, transaction.ID.Target.ID)
			if err := r.updateTransactionStatus(ctx, transaction); err != nil {
				return controller.Result{}, false, err
			}
			return controller.Result{
				Requeue: controller.NewID(configapi.TransactionID{
					Target: transaction.ID.Target,
					Index:  transaction.ID.Index + 1,
				}),
			}, true, nil
		}

		values := addDeleteChildren(transaction.ID.Index, transaction.Values, configuration.Committed.Values)
		if ok, err := r.applyValues(ctx, transaction, configuration, values); !ok {
			return controller.Result{}, false, err
		} else if err != nil {
			code := status.Code(err)
			switch code {
			case codes.Unavailable, codes.Canceled, codes.DeadlineExceeded:
				return controller.Result{}, false, err
			case codes.PermissionDenied:
				// The gNMI Set request can be denied if this master has been superseded by a master in a later term.
				// Rather than reverting to the STALE state now, wait for this node to see the mastership state change
				// to avoid flapping between states while the system converges.
				log.Warnf("Configuration '%s' mastership superseded for term %d", configuration.ID, configuration.Applied.Term)
				return controller.Result{}, false, nil
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

				// Add the failure to the proposal's apply phase state.
				log.Warnf("Failed applying Transaction %d to '%s'", transaction.ID.Index, transaction.ID.Target.ID, err)
				transaction.Status.Change.Apply.State = configapi.TransactionPhaseStatus_FAILED
				transaction.Status.Change.Apply.Failure = &configapi.Failure{
					Type:        failureType,
					Description: err.Error(),
				}
				transaction.Status.Change.Apply.End = now()
				if err := r.updateTransactionStatus(ctx, transaction); err != nil {
					return controller.Result{}, false, err
				}

				// Update the Configuration's applied index to indicate this Transaction was applied even though it failed.
				log.Infof("Updating applied index for Configuration '%s' to %d in term %d", configuration.ID, transaction.ID.Index, configuration.Applied.Term)
				configuration.Applied.Index = transaction.ID.Index
				configuration.Applied.Ordinal = transaction.Status.Change.Ordinal
				if err := r.updateConfigurationStatus(ctx, configuration); err != nil {
					return controller.Result{}, false, err
				}
				return controller.Result{}, true, nil
			}
		}

		log.Infof("Updating applied index for Configuration '%s' to %d in term %d", configuration.ID, transaction.ID.Index, configuration.Applied.Term)
		configuration.Applied.Index = transaction.ID.Index
		configuration.Applied.Ordinal = transaction.Status.Change.Ordinal
		configuration.Applied.Revision = configapi.Revision(transaction.ID.Index)
		if configuration.Applied.Values == nil {
			configuration.Applied.Values = make(map[string]configapi.PathValue)
		}
		for path, changeValue := range values {
			configuration.Applied.Values[path] = changeValue
		}

		if err := r.updateConfigurationStatus(ctx, configuration); err != nil {
			log.Warnf("Failed reconciling Proposal %d to target '%s'", transaction.ID.Index, transaction.ID.Target.ID, err)
			return controller.Result{}, false, err
		}

		// Update the proposal state to APPLIED.
		log.Infof("Applied Transaction %d to '%s'", transaction.ID.Index, transaction.ID.Target.ID)
		transaction.Status.Change.Apply.State = configapi.TransactionPhaseStatus_COMPLETE
		transaction.Status.Change.Apply.End = now()
		if err := r.updateTransactionStatus(ctx, transaction); err != nil {
			return controller.Result{}, false, err
		}
		return controller.Result{
			Requeue: controller.NewID(configapi.TransactionID{
				Target: transaction.ID.Target,
				Index:  transaction.ID.Index + 1,
			}),
		}, true, nil
	case configapi.TransactionPhaseStatus_ABORTED, configapi.TransactionPhaseStatus_FAILED:
		if configuration.Applied.Ordinal < transaction.Status.Change.Ordinal {
			configuration.Applied.Index = transaction.ID.Index
			configuration.Applied.Target = transaction.ID.Index
			configuration.Applied.Ordinal = transaction.Status.Change.Ordinal
			if err := r.updateConfigurationStatus(ctx, configuration); err != nil {
				return controller.Result{}, false, err
			}
			return controller.Result{}, true, nil
		}
	}
	return controller.Result{}, false, nil
}

func (r *Reconciler) reconcileRollback(ctx context.Context, transaction *configapi.Transaction, configuration *configapi.Configuration) (controller.Result, bool, error) {
	if transaction.Status.Phase != configapi.TransactionStatus_ROLLBACK {
		return controller.Result{}, false, nil
	}
	if result, ok, err := r.commitRollback(ctx, transaction, configuration); err != nil {
		return controller.Result{}, false, err
	} else if ok {
		return result, true, nil
	}
	if result, ok, err := r.applyRollback(ctx, transaction, configuration); err != nil {
		return controller.Result{}, false, err
	} else if ok {
		return result, true, nil
	}
	return controller.Result{}, false, nil
}

func (r *Reconciler) commitRollback(ctx context.Context, transaction *configapi.Transaction, configuration *configapi.Configuration) (controller.Result, bool, error) {
	if transaction.Status.Rollback.Commit == nil {
		return controller.Result{}, false, nil
	}
	switch transaction.Status.Rollback.Commit.State {
	case configapi.TransactionPhaseStatus_PENDING:
		if configuration.Committed.Revision != configapi.Revision(transaction.ID.Index) {
			return controller.Result{}, false, nil
		}

		if configuration.Committed.Target == transaction.ID.Index {
			if configuration.Committed.Index != configuration.Committed.Target {
				return controller.Result{}, false, nil
			}

			prevTransactionID := configapi.TransactionID{
				Target: transaction.ID.Target,
				Index:  configuration.Committed.Index,
			}
			prevTransaction, err := r.transactions.Get(ctx, prevTransactionID)
			if err != nil {
				if !errors.IsNotFound(err) {
					return controller.Result{}, false, err
				}
			} else if configuration.Committed.Index == transaction.ID.Index &&
				prevTransaction.Status.Change.Commit.State != configapi.TransactionPhaseStatus_COMPLETE {
				return controller.Result{}, false, nil
			} else if configuration.Committed.Index > transaction.ID.Index &&
				prevTransaction.Status.Rollback.Commit.State != configapi.TransactionPhaseStatus_COMPLETE {
				return controller.Result{}, false, nil
			}

			configuration.Committed.Target = transaction.Status.Rollback.Index
			if err := r.updateConfigurationStatus(ctx, configuration); err != nil {
				return controller.Result{}, false, nil
			}
		}

		if configuration.Committed.Target == transaction.Status.Rollback.Index {
			transaction.Status.Rollback.Commit.State = configapi.TransactionPhaseStatus_IN_PROGRESS
			transaction.Status.Rollback.Commit.Start = now()
			if err := r.updateTransactionStatus(ctx, transaction); err != nil {
				return controller.Result{}, false, err
			}
			return controller.Result{}, true, nil
		}
		return controller.Result{}, false, nil
	case configapi.TransactionPhaseStatus_IN_PROGRESS:
		if configuration.Committed.Revision == configapi.Revision(transaction.ID.Index) {
			for path, value := range transaction.Status.Rollback.Values {
				configuration.Committed.Values[path] = value
			}
			configuration.Committed.Index = transaction.ID.Index
			configuration.Committed.Ordinal = configuration.Committed.Ordinal + 1
			configuration.Committed.Revision = configapi.Revision(transaction.Status.Rollback.Index)
			if err := r.updateConfigurationStatus(ctx, configuration); err != nil {
				return controller.Result{}, false, err
			}
		}

		transaction.Status.Rollback.Ordinal = configuration.Committed.Ordinal
		transaction.Status.Rollback.Commit.State = configapi.TransactionPhaseStatus_COMPLETE
		transaction.Status.Rollback.Commit.End = now()
		if err := r.updateTransactionStatus(ctx, transaction); err != nil {
			return controller.Result{}, false, err
		}
		return controller.Result{}, true, nil
	}
	return controller.Result{}, false, nil
}

func (r *Reconciler) applyRollback(ctx context.Context, transaction *configapi.Transaction, configuration *configapi.Configuration) (controller.Result, bool, error) {
	if transaction.Status.Rollback.Commit == nil || transaction.Status.Rollback.Apply == nil {
		return controller.Result{}, false, nil
	}

	// Wait for the rollback to be committed first.
	if transaction.Status.Rollback.Commit.State != configapi.TransactionPhaseStatus_COMPLETE {
		return controller.Result{}, false, nil
	}

	switch transaction.Status.Rollback.Apply.State {
	case configapi.TransactionPhaseStatus_PENDING:
		// The change must be completed before the rollback can be applied. If the change apply is
		// pending, it must be aborted. If the change is in progress, it must be failed to ensure
		// the rollback will be applied to the target in the event of a race or failure in which the
		// target was updated but the transaction status was not. This enables users to effectively
		// cancel hanging (in progress) transactions by rolling them back.
		switch transaction.Status.Change.Apply.State {
		case configapi.TransactionPhaseStatus_PENDING:
			// If the change is pending apply, abort the apply phase. This must be done only once the
			// prior transaction phase has been applied to ensure aborts still occur sequentially
			// within the transaction log.
			if configuration.Applied.Ordinal == transaction.Status.Change.Ordinal-1 &&
				configuration.Applied.Target != transaction.ID.Index {
				prevTransactionID := configapi.TransactionID{
					Target: transaction.ID.Target,
					Index:  configuration.Applied.Index,
				}
				prevTransaction, err := r.transactions.Get(ctx, prevTransactionID)
				if err != nil {
					if !errors.IsNotFound(err) {
						return controller.Result{}, false, err
					}
				} else if configuration.Applied.Target == configuration.Applied.Index &&
					prevTransaction.Status.Change.Apply.State < configapi.TransactionPhaseStatus_COMPLETE {
					return controller.Result{}, false, nil
				} else if configuration.Applied.Target < configuration.Applied.Index &&
					prevTransaction.Status.Rollback.Apply.State < configapi.TransactionPhaseStatus_COMPLETE {
					return controller.Result{}, false, nil
				}

				transaction.Status.Change.Apply.State = configapi.TransactionPhaseStatus_ABORTED
				transaction.Status.Change.Apply.End = now()
				if err := r.updateTransactionStatus(ctx, transaction); err != nil {
					return controller.Result{}, false, err
				}

				configuration.Applied.Target = transaction.ID.Index
				configuration.Applied.Index = transaction.ID.Index
				configuration.Applied.Ordinal = transaction.Status.Change.Ordinal
				if err := r.updateConfigurationStatus(ctx, configuration); err != nil {
					return controller.Result{}, false, err
				}
				return controller.Result{}, true, nil
			}
			return controller.Result{}, false, nil
		case configapi.TransactionPhaseStatus_IN_PROGRESS:
			// If the change apply is IN_PROGRESS, fail the apply phase and update the applied
			// configuration status to unblock later transactions.
			if configuration.Applied.Ordinal != transaction.Status.Change.Ordinal {
				transaction.Status.Change.Apply.State = configapi.TransactionPhaseStatus_FAILED
				transaction.Status.Change.Apply.Failure = &configapi.Failure{
					Type:        configapi.Failure_CANCELED,
					Description: "canceled pending rollback",
				}
				transaction.Status.Change.Apply.End = now()
				if err := r.updateTransactionStatus(ctx, transaction); err != nil {
					return controller.Result{}, false, err
				}

				configuration.Applied.Target = transaction.ID.Index
				configuration.Applied.Index = transaction.ID.Index
				configuration.Applied.Ordinal = transaction.Status.Change.Ordinal
				if err := r.updateConfigurationStatus(ctx, configuration); err != nil {
					return controller.Result{}, false, err
				}
				return controller.Result{}, true, nil
			}
			return controller.Result{}, false, nil
		case configapi.TransactionPhaseStatus_ABORTED, configapi.TransactionPhaseStatus_FAILED:
			// If the change apply has been marked aborted or failed, ensure the applied configuration
			// status is updated. This is necessary in the event a failure occurs after aborting/failing
			// the transaction but before the configuration status could be updated.
			if configuration.Applied.Ordinal < transaction.Status.Change.Ordinal {
				configuration.Applied.Target = transaction.ID.Index
				configuration.Applied.Index = transaction.ID.Index
				configuration.Applied.Ordinal = transaction.Status.Change.Ordinal
				if err := r.updateConfigurationStatus(ctx, configuration); err != nil {
					return controller.Result{}, false, err
				}
				return controller.Result{}, true, nil
			}
		}

		// The rollback cannot be applied until the prior ordinal has been applied (can be a change or rollback).
		if configuration.Applied.Ordinal != transaction.Status.Rollback.Ordinal-1 {
			return controller.Result{}, false, nil
		}

		// If the applied target was already updated, this indicates a failure occurred before the transaction
		// status could be updated. Move the rollback apply phase to IN_PROGRESS.
		if configuration.Applied.Target == transaction.Status.Rollback.Index {
			transaction.Status.Rollback.Apply.State = configapi.TransactionPhaseStatus_IN_PROGRESS
			transaction.Status.Rollback.Apply.Start = now()
			if err := r.updateTransactionStatus(ctx, transaction); err != nil {
				return controller.Result{}, false, err
			}
			return controller.Result{}, true, nil
		}

		// Verify the prior transaction/phase (change or rollback) has been applied before proceeding.
		prevTransactionID := configapi.TransactionID{
			Target: transaction.ID.Target,
			Index:  configuration.Applied.Index,
		}
		prevTransaction, err := r.transactions.Get(ctx, prevTransactionID)
		if err != nil {
			if !errors.IsNotFound(err) {
				return controller.Result{}, false, err
			}
		} else if configuration.Applied.Index == transaction.ID.Index &&
			prevTransaction.Status.Change.Apply.State < configapi.TransactionPhaseStatus_COMPLETE {
			return controller.Result{}, false, nil
		} else if configuration.Applied.Index > transaction.ID.Index &&
			prevTransaction.Status.Rollback.Apply.State < configapi.TransactionPhaseStatus_COMPLETE {
			return controller.Result{}, false, nil
		}

		// Update the applied target index and mark the rollback apply phase IN_PROGRESS.
		configuration.Applied.Target = transaction.Status.Rollback.Index
		if err := r.updateConfigurationStatus(ctx, configuration); err != nil {
			return controller.Result{}, false, err
		}

		transaction.Status.Rollback.Apply.State = configapi.TransactionPhaseStatus_IN_PROGRESS
		transaction.Status.Rollback.Apply.Start = now()
		if err := r.updateTransactionStatus(ctx, transaction); err != nil {
			return controller.Result{}, false, err
		}
		return controller.Result{}, true, nil
	case configapi.TransactionPhaseStatus_IN_PROGRESS:
		// A failure occurred between the configuration update and the transaction update.
		if configuration.Applied.Ordinal == transaction.Status.Rollback.Ordinal &&
			configuration.Applied.Revision == configapi.Revision(transaction.Status.Rollback.Index) {
			transaction.Status.Rollback.Apply.State = configapi.TransactionPhaseStatus_COMPLETE
			transaction.Status.Rollback.Apply.End = now()
			log.Infof("Applied Transaction %d rollback to '%s'", transaction.ID.Index, transaction.ID.Target.ID)
			if err := r.updateTransactionStatus(ctx, transaction); err != nil {
				return controller.Result{}, false, err
			}
			return controller.Result{
				Requeue: controller.NewID(configapi.TransactionID{
					Target: transaction.ID.Target,
					Index:  transaction.ID.Index + 1,
				}),
			}, true, nil
		}

		values := addDeleteChildren(transaction.ID.Index, transaction.Status.Rollback.Values, configuration.Committed.Values)
		if ok, err := r.applyValues(ctx, transaction, configuration, values); !ok {
			return controller.Result{}, false, err
		} else if err != nil {
			code := status.Code(err)
			switch code {
			case codes.Unavailable, codes.Canceled, codes.DeadlineExceeded:
				return controller.Result{}, false, err
			case codes.PermissionDenied:
				// The gNMI Set request can be denied if this master has been superseded by a master in a later term.
				// Rather than reverting to the STALE state now, wait for this node to see the mastership state change
				// to avoid flapping between states while the system converges.
				log.Warnf("Configuration '%s' mastership superseded for term %d", configuration.ID, configuration.Applied.Term)
				return controller.Result{}, false, nil
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

				// Update the Configuration's applied index to indicate this Transaction was applied even though it failed.
				log.Infof("Updating applied index for Configuration '%s' to %d in term %d", configuration.ID, transaction.ID.Index, configuration.Applied.Term)
				configuration.Applied.Index = transaction.ID.Index
				configuration.Applied.Ordinal = transaction.Status.Rollback.Ordinal
				if err := r.updateConfigurationStatus(ctx, configuration); err != nil {
					log.Warnf("Failed reconciling Proposal %d to target '%s'", transaction.ID.Index, transaction.ID.Target.ID, err)
					return controller.Result{}, false, err
				}

				// Add the failure to the proposal's apply phase state.
				log.Warnf("Failed applying Transaction %d to '%s'", transaction.ID.Index, transaction.ID.Target.ID, err)
				transaction.Status.Rollback.Apply.State = configapi.TransactionPhaseStatus_FAILED
				transaction.Status.Rollback.Apply.Failure = &configapi.Failure{
					Type:        failureType,
					Description: err.Error(),
				}
				transaction.Status.Rollback.Apply.End = now()
				if err := r.updateTransactionStatus(ctx, transaction); err != nil {
					return controller.Result{}, false, err
				}
				return controller.Result{}, true, nil
			}
		}

		log.Infof("Updating applied index for Configuration '%s' to %d in term %d", configuration.ID, transaction.ID.Index, configuration.Applied.Term)
		configuration.Applied.Index = transaction.ID.Index
		configuration.Applied.Ordinal = transaction.Status.Rollback.Ordinal
		configuration.Applied.Revision = configapi.Revision(transaction.Status.Rollback.Index)
		if configuration.Applied.Values == nil {
			configuration.Applied.Values = make(map[string]configapi.PathValue)
		}
		for path, changeValue := range values {
			configuration.Applied.Values[path] = changeValue
		}

		if err := r.updateConfigurationStatus(ctx, configuration); err != nil {
			log.Warnf("Failed reconciling Proposal %d to target '%s'", transaction.ID.Index, transaction.ID.Target.ID, err)
			return controller.Result{}, false, err
		}

		// Update the proposal state to APPLIED.
		log.Infof("Applied Transaction %d to '%s'", transaction.ID.Index, transaction.ID.Target.ID)
		transaction.Status.Rollback.Apply.State = configapi.TransactionPhaseStatus_COMPLETE
		transaction.Status.Rollback.Apply.End = now()
		if err := r.updateTransactionStatus(ctx, transaction); err != nil {
			return controller.Result{}, false, err
		}
		return controller.Result{
			Requeue: controller.NewID(configapi.TransactionID{
				Target: transaction.ID.Target,
				Index:  transaction.ID.Index + 1,
			}),
		}, true, nil
	}
	return controller.Result{}, false, nil
}

func (r *Reconciler) applyValues(ctx context.Context, transaction *configapi.Transaction, configuration *configapi.Configuration, values map[string]configapi.PathValue) (bool, error) {
	// If the configuration is synchronizing, wait for it to complete.
	if configuration.Status.State == configapi.ConfigurationStatus_SYNCHRONIZING {
		log.Infof("Waiting for synchronization of Configuration to target '%s'", transaction.ID.Target.ID)
		return false, nil
	}

	// Get the target entity from topo
	target, err := r.topo.Get(ctx, topoapi.ID(transaction.ID.Target.ID))
	if err != nil {
		if !errors.IsNotFound(err) {
			log.Errorf("Failed fetching target Entity '%s' from topo", transaction.ID.Target.ID, err)
			return false, err
		}
		log.Debugf("Target entity '%s' not found", transaction.ID.Target.ID)
		return false, nil
	}

	// If the configuration is in an old term, wait for synchronization.
	if configuration.Applied.Term < configuration.Status.Mastership.Term {
		log.Infof("Waiting for synchronization of Configuration to target '%s'", transaction.ID.Target.ID)
		return false, nil
	}

	// If the master node ID is not set, skip reconciliation.
	if configuration.Status.Mastership.Master == "" {
		log.Debugf("No master for target '%s'", transaction.ID.Target.ID)
		return false, nil
	}

	// If we've made it this far, we know there's a master relation.
	// Get the relation and check whether this node is the source
	relation, err := r.topo.Get(ctx, topoapi.ID(configuration.Status.Mastership.Master))
	if err != nil {
		if !errors.IsNotFound(err) {
			log.Errorf("Failed fetching master Relation '%s' from topo", configuration.Status.Mastership.Master, err)
			return false, err
		}
		log.Warnf("Master relation not found for target '%s'", transaction.ID.Target.ID)
		return false, nil
	}
	if relation.GetRelation().SrcEntityID != topoapi.ID(r.nodeID) {
		log.Debugf("Not the master for target '%s'", transaction.ID.Target.ID)
		return false, nil
	}

	// Get the master connection
	conn, ok := r.conns.Get(ctx, gnmi.ConnID(relation.ID))
	if !ok {
		log.Warnf("Connection not found for target '%s'", transaction.ID.Target.ID)
		return false, nil
	}

	configurable := &topoapi.Configurable{}
	_ = target.GetAspect(configurable)

	if configurable.ValidateCapabilities {
		capabilityResponse, err := conn.Capabilities(ctx, &gpb.CapabilityRequest{})
		if err != nil {
			log.Warnf("Cannot retrieve capabilities of the target %s", transaction.ID.Target.ID)
			return false, err
		}
		targetDataModels := capabilityResponse.SupportedModels
		modelPlugin, ok := r.plugins.GetPlugin(
			configv2.TargetType(transaction.ID.Target.Type),
			configv2.TargetVersion(transaction.ID.Target.Version))
		if !ok {
			transaction.Status.Rollback.Apply.State = configapi.TransactionPhaseStatus_FAILED
			transaction.Status.Rollback.Apply.Failure = &configapi.Failure{
				Type:        configapi.Failure_INVALID,
				Description: fmt.Sprintf("model plugin '%s/%s' not found", transaction.ID.Target.Type, transaction.ID.Target.Version),
			}
			transaction.Status.Rollback.Apply.End = now()
			if err := r.updateTransactionStatus(ctx, transaction); err != nil {
				return false, err
			}
			return false, nil
		}
		pluginCapability := modelPlugin.Capabilities(ctx)
		pluginDataModels := pluginCapability.SupportedModels

		if !isModelDataCompatible(pluginDataModels, targetDataModels) {
			err = errors.NewNotSupported("plugin data models %v are not supported in the target %s", pluginDataModels, transaction.ID.Target.ID)
			log.Warn(err)
			return false, err
		}
	}

	// Create a list of PathValue pairs from which to construct a gNMI Set for the Proposal.
	pathValues := make([]configapi.PathValue, 0, len(values))
	for _, changeValue := range values {
		pathValues = append(pathValues, changeValue)
	}
	pathValues = tree.PrunePathValues(pathValues, true)

	log.Infof("Updating %d paths on target '%s'", len(pathValues), configuration.ID.Target.ID)

	// Create a gNMI set request
	setRequest, err := valuesutil.PathValuesToGnmiChange(pathValues, transaction.ID.Target.ID)
	if err != nil {
		log.Errorf("Failed constructing SetRequest for Configuration '%s'", configuration.ID, err)
		return false, nil
	}

	// Add the master arbitration extension to provide concurrency control for multi-node controllers.
	setRequest.Extension = append(setRequest.Extension, &gnmi_ext.Extension{
		Ext: &gnmi_ext.Extension_MasterArbitration{
			MasterArbitration: &gnmi_ext.MasterArbitration{
				Role: &gnmi_ext.Role{
					Id: "onos-config",
				},
				ElectionId: &gnmi_ext.Uint128{
					Low: uint64(configuration.Applied.Term),
				},
			},
		},
	})

	// Execute the set request
	log.Debugf("Sending SetRequest %+v", setRequest)
	setResponse, err := conn.Set(ctx, setRequest)
	if err != nil {
		log.Warnf("Failed sending SetRequest %+v", setRequest, err)
		return true, err
	}
	log.Debugf("Received SetResponse %+v", setResponse)
	return true, nil
}

func applyChangeToConfig(values map[string]configapi.PathValue, path string, value configapi.PathValue) (string, bool, configapi.PathValue) {
	values[path] = value

	// Walk up the path and make sure that there are no parents marked as deleted in the given map, if so, remove them
	parent := pathutils.GetParentPath(path)
	for parent != "" {
		if v, ok := values[parent]; ok && v.Deleted {
			// Delete the parent marked as deleted and return its path and value
			delete(values, parent)
			return parent, true, v
		}
		parent = pathutils.GetParentPath(parent)
	}
	return "", false, configapi.PathValue{}
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

func (r *Reconciler) updateTransactionStatus(ctx context.Context, transaction *configapi.Transaction) error {
	log.Debug(transaction.Status)
	err := r.transactions.UpdateStatus(ctx, transaction)
	if err != nil {
		if !errors.IsNotFound(err) && !errors.IsConflict(err) {
			log.Errorf("Failed updating Transaction %s status", transaction.ID, err)
			return err
		}
		log.Warnf("Write conflict updating Transaction %s status", transaction.ID, err)
		return nil
	}
	return nil
}

func (r *Reconciler) updateConfigurationStatus(ctx context.Context, configuration *configapi.Configuration) error {
	log.Debug(configuration.Status)
	err := r.configurations.UpdateStatus(ctx, configuration)
	if err != nil {
		if !errors.IsNotFound(err) && !errors.IsConflict(err) {
			log.Errorf("Failed updating Configuration '%s' status", configuration.ID, err)
			return err
		}
		log.Warnf("Write conflict updating Configuration '%s' status", configuration.ID, err)
		return nil
	}
	return nil
}

func now() *time.Time {
	t := time.Now()
	return &t
}

// addDeleteChildren adds all children of the intermediate path which is required to be deleted
func addDeleteChildren(index configapi.Index, changeValues map[string]configapi.PathValue, configStore map[string]configapi.PathValue) map[string]configapi.PathValue {
	// defining new changeValues map, where we will include old changeValues map and new pathValues to be cascading deleted
	var updChangeValues = make(map[string]configapi.PathValue)
	for _, changeValue := range changeValues {
		// if this pathValue has to be deleted, then we need to search for all children of this pathValue
		if changeValue.Deleted {
			for _, value := range configStore {
				if strings.HasPrefix(value.Path, changeValue.Path) && !strings.EqualFold(value.Path, changeValue.Path) {
					value.Index = index
					value.Deleted = true
					updChangeValues[value.Path] = value
				}
			}
			// overwriting itself in the store, we want the latest value (changeValue variable)
			updChangeValues[changeValue.Path] = changeValue
		} else {
			updChangeValues[changeValue.Path] = changeValue
		}
	}
	return updChangeValues
}
