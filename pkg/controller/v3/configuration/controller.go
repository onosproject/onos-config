// SPDX-FileCopyrightText: 2020-present Open Networking Foundation <info@opennetworking.org>
//
// SPDX-License-Identifier: Apache-2.0

package configuration

import (
	"context"
	"github.com/onosproject/onos-config/pkg/utils/v3/values"
	"github.com/openconfig/gnmi/proto/gnmi_ext"
	"time"

	controllerutils "github.com/onosproject/onos-config/pkg/controller/utils"

	topoapi "github.com/onosproject/onos-api/go/onos/topo"

	"github.com/onosproject/onos-lib-go/pkg/errors"

	configapi "github.com/onosproject/onos-api/go/onos/config/v3"
	"github.com/onosproject/onos-config/pkg/southbound/gnmi"
	"github.com/onosproject/onos-config/pkg/store/topo"
	configurationstore "github.com/onosproject/onos-config/pkg/store/v3/configuration"
	"github.com/onosproject/onos-lib-go/pkg/controller"
	"github.com/onosproject/onos-lib-go/pkg/logging"
)

var log = logging.GetLogger("controller", "configuration")

const (
	defaultTimeout = 30 * time.Second
)

// NewController returns a configuration controller
func NewController(topo topo.Store, conns gnmi.ConnManager, configurations configurationstore.Store) *controller.Controller {
	c := controller.NewController("configuration")
	c.Watch(&Watcher{
		configurations: configurations,
	})
	c.Watch(&TopoWatcher{
		topo: topo,
	})
	c.Reconcile(&Reconciler{
		conns:          conns,
		topo:           topo,
		configurations: configurations,
	})
	return c
}

// Reconciler reconciles configurations
type Reconciler struct {
	conns          gnmi.ConnManager
	topo           topo.Store
	configurations configurationstore.Store
}

// Reconcile reconciles target configurations
func (r *Reconciler) Reconcile(id controller.ID) (controller.Result, error) {
	ctx, cancel := context.WithTimeout(context.Background(), defaultTimeout)
	defer cancel()

	configurationID := id.Value.(configapi.ConfigurationID)
	config, err := r.configurations.Get(ctx, configurationID)
	if err != nil {
		if !errors.IsNotFound(err) {
			log.Warnf("Failed to reconcile Configuration '%s'", configurationID, err)
			return controller.Result{}, err
		}
		log.Debugf("Configuration '%s' not found", configurationID)
		return controller.Result{}, nil
	}

	log.Debugf("Reconciling Configuration '%s'", config.ID)
	log.Debug(config)
	return r.reconcileConfiguration(ctx, config)
}

func (r *Reconciler) reconcileConfiguration(ctx context.Context, config *configapi.Configuration) (controller.Result, error) {
	// Get the target entity from topo
	target, err := r.topo.Get(ctx, topoapi.ID(config.ID.Target.ID))
	if err != nil {
		if !errors.IsNotFound(err) {
			log.Errorf("Failed fetching target Entity '%s' from topo", config.ID.Target.ID, err)
			return controller.Result{}, err
		}
		log.Debugf("Target entity '%s' not found", config.ID.Target.ID)
		return controller.Result{}, nil
	}

	// Get the target configurable configuration
	configurable := topoapi.Configurable{}
	_ = target.GetAspect(&configurable)

	// If the target is persistent, mark the configuration PERSISTED.
	if configurable.Persistent {
		if config.Status.State != configapi.ConfigurationStatus_PERSISTED {
			log.Infof("Skipping synchronization of Configuration '%s': target is persistent", config.ID)
			config.Status.State = configapi.ConfigurationStatus_PERSISTED
			if err := r.updateConfigurationStatus(ctx, config); err != nil {
				return controller.Result{}, err
			}
			return controller.Result{}, nil
		}
		return controller.Result{}, nil
	}

	// If the configuration is not SYNCHRONIZING, skip synchronization.
	if config.Status.State != configapi.ConfigurationStatus_SYNCHRONIZING {
		// If the configuration term is greater than the applied term, set the state to SYNCHRONIZING
		if config.Status.Mastership.Term > config.Applied.Term {
			log.Infof("Configuration '%s' mastership term has increased, synchronizing...", config.ID)
			config.Status.State = configapi.ConfigurationStatus_SYNCHRONIZING
			if err := r.updateConfigurationStatus(ctx, config); err != nil {
				return controller.Result{}, err
			}
			return controller.Result{}, nil
		}
		return controller.Result{}, nil
	}

	// If the master ID is not set, skip reconciliation.
	if config.Status.Mastership.Master == "" {
		log.Debugf("No master for target '%s'", config.ID.Target.ID)
		return controller.Result{}, nil
	}

	// If the applied index is 0, skip applying changes.
	if config.Applied.Index == 0 {
		log.Infof("Skipping synchronization of Configuration '%s': no applied changes to synchronize", config.ID)
		config.Status.State = configapi.ConfigurationStatus_SYNCHRONIZED
		config.Applied.Term = config.Status.Mastership.Term
		if err := r.updateConfigurationStatus(ctx, config); err != nil {
			return controller.Result{}, err
		}
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
		log.Warnf("Master relation not found for target '%s'", config.ID.Target.ID)
		return controller.Result{}, nil
	}
	if relation.GetRelation().SrcEntityID != controllerutils.GetOnosConfigID() {
		log.Debugf("Not the master for target '%s'", config.ID.Target.ID)
		return controller.Result{}, nil
	}

	// Get the master connection
	conn, ok := r.conns.Get(ctx, gnmi.ConnID(relation.ID))
	if !ok {
		log.Warnf("Connection not found for target '%s'", config.ID.Target.ID)
		return controller.Result{}, nil
	}

	indexedPathValues := make(map[configapi.Index][]configapi.PathValue)
	if config.Applied.Values != nil {
		for _, appliedValue := range config.Applied.Values {
			indexedPathValues[appliedValue.Index] = append(indexedPathValues[appliedValue.Index], appliedValue)
		}
	}
	log.Infof("Updating %d paths on target '%s'", len(indexedPathValues), config.ID.Target.ID)
	for transactionIndex, pathValues := range indexedPathValues {
		// Create a gNMI set request
		log.Debugw("Creating Set request for changes in transaction", "TransactionIndex", transactionIndex)
		setRequest, err := values.PathValuesToGnmiChange(pathValues, config.ID.Target.ID)
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
			// The gNMI Set request can be denied if this master has been superseded by a master in a later term.
			// Rather than reverting to the STALE state now, wait for this node to see the mastership state change
			// to avoid flapping between states while the system converges.
			if errors.IsForbidden(err) {
				log.Warnf("Configuration '%s' mastership superseded for term %d", config.ID, config.Status.Mastership.Term)
				return controller.Result{}, nil
			}
			log.Errorf("Failed sending SetRequest %+v", setRequest, err)
			return controller.Result{}, err
		}
		log.Debugf("Received SetResponse %+v", setResponse)
	}

	// Update the configuration state and path statuses
	log.Infof("Configuration '%s' synchronization complete", config.ID)
	config.Applied.Term = config.Status.Mastership.Term
	config.Status.State = configapi.ConfigurationStatus_SYNCHRONIZED
	if err := r.updateConfigurationStatus(ctx, config); err != nil {
		return controller.Result{}, err
	}
	return controller.Result{}, nil
}

func (r *Reconciler) updateConfigurationStatus(ctx context.Context, configuration *configapi.Configuration) error {
	log.Debug(configuration.Status)
	err := r.configurations.UpdateStatus(ctx, configuration)
	if err != nil {
		if !errors.IsNotFound(err) && !errors.IsConflict(err) {
			log.Errorf("Failed updating Proposal '%s' status", configuration.ID, err)
			return err
		}
		log.Warnf("Write conflict updating Proposal '%s' status", configuration.ID, err)
		return nil
	}
	return nil
}
