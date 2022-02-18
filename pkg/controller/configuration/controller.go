// Copyright 2021-present Open Networking Foundation.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package configuration

import (
	"context"
	"github.com/openconfig/gnmi/proto/gnmi_ext"
	"time"

	controllerutils "github.com/onosproject/onos-config/pkg/controller/utils"

	topoapi "github.com/onosproject/onos-api/go/onos/topo"

	"github.com/onosproject/onos-lib-go/pkg/errors"

	configapi "github.com/onosproject/onos-api/go/onos/config/v2"
	utilsv2 "github.com/onosproject/onos-config/pkg/utils/values/v2"

	"github.com/onosproject/onos-config/pkg/southbound/gnmi"
	"github.com/onosproject/onos-config/pkg/store/configuration"
	"github.com/onosproject/onos-config/pkg/store/topo"
	"github.com/onosproject/onos-lib-go/pkg/controller"
	"github.com/onosproject/onos-lib-go/pkg/logging"
)

var log = logging.GetLogger("controller", "configuration")

const (
	defaultTimeout = 30 * time.Second
)

// NewController returns a configuration controller
func NewController(topo topo.Store, conns gnmi.ConnManager, configurations configuration.Store) *controller.Controller {
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
	configurations configuration.Store
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

	log.Infof("Reconciling Configuration '%s'", config.ID)
	log.Debug(config)
	return r.reconcileConfiguration(ctx, config)
}

func (r *Reconciler) reconcileConfiguration(ctx context.Context, config *configapi.Configuration) (controller.Result, error) {
	// Get the target entity from topo
	target, err := r.topo.Get(ctx, topoapi.ID(config.TargetID))
	if err != nil {
		if !errors.IsNotFound(err) {
			log.Errorf("Failed fetching target Entity '%s' from topo", config.TargetID, err)
			return controller.Result{}, err
		}
		log.Debugf("Target entity '%s' not found", config.TargetID)

		// If the target was deleted after a master was already elected for it,
		// SOUND THE ALARM! and revert back to the SYNCHRONIZING state.
		log.Errorf("Mastership state lost for target '%s'. Future configuration changes may not be applicable!")
		config.Status.State = configapi.ConfigurationStatus_UNKNOWN
		config.Status.Mastership.Master = ""
		config.Status.Mastership.Term = 0
		if err := r.updateConfigurationStatus(ctx, config); err != nil {
			return controller.Result{}, err
		}
		return controller.Result{}, nil
	}

	// Get the target configurable configuration
	configurable := topoapi.Configurable{}
	_ = target.GetAspect(&configurable)

	// Get the target mastership state
	mastership := topoapi.MastershipState{}
	_ = target.GetAspect(&mastership)
	mastershipTerm := configapi.MastershipTerm(mastership.Term)

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

	// If the mastership term has changed, update the configuration and mark it SYNCHRONIZING for the next term.
	if mastershipTerm > config.Status.Mastership.Term {
		log.Infof("Synchronizing Configuration '%s'", config.ID)
		config.Status.State = configapi.ConfigurationStatus_SYNCHRONIZING
		config.Status.Mastership.Master = mastership.NodeId
		config.Status.Mastership.Term = mastershipTerm
		if err := r.updateConfigurationStatus(ctx, config); err != nil {
			return controller.Result{}, err
		}
		return controller.Result{}, nil
	}

	// If the configuration is not SYNCHRONIZING, skip synchronization.
	if config.Status.State != configapi.ConfigurationStatus_SYNCHRONIZING {
		return controller.Result{}, nil
	}

	// If the master node ID is not set, skip reconciliation.
	if mastership.NodeId == "" {
		log.Debugf("No master for target '%s'", config.TargetID)
		return controller.Result{}, nil
	}

	// If the applied index is 0, skip applying changes.
	if config.Status.Applied.Index == 0 {
		log.Infof("Skipping synchronization of Configuration '%s': no applied changes to synchronize", config.ID)
		config.Status.State = configapi.ConfigurationStatus_SYNCHRONIZED
		if err := r.updateConfigurationStatus(ctx, config); err != nil {
			return controller.Result{}, err
		}
		return controller.Result{}, nil
	}

	// If we've made it this far, we know there's a master relation.
	// Get the relation and check whether this node is the source
	relation, err := r.topo.Get(ctx, topoapi.ID(mastership.NodeId))
	if err != nil {
		if !errors.IsNotFound(err) {
			log.Errorf("Failed fetching master Relation '%s' from topo", mastership.NodeId, err)
			return controller.Result{}, err
		}
		log.Warnf("Master relation not found for target '%s'", config.TargetID)
		return controller.Result{}, nil
	}
	if relation.GetRelation().SrcEntityID != controllerutils.GetOnosConfigID() {
		log.Debugf("Not the master for target '%s'", config.TargetID)
		return controller.Result{}, nil
	}

	// Get the master connection
	conn, ok := r.conns.Get(ctx, gnmi.ConnID(relation.ID))
	if !ok {
		log.Warnf("Connection not found for target '%s'", config.TargetID)
		return controller.Result{}, nil
	}

	pathValues := make([]*configapi.PathValue, 0, len(config.Status.Applied.Values))
	if config.Status.Applied.Values != nil {
		for _, appliedValue := range config.Status.Applied.Values {
			pathValues = append(pathValues, appliedValue)
		}
	}
	log.Infof("Updating %d paths on target '%s'", len(pathValues), config.TargetID)

	// Create a gNMI set request
	setRequest, err := utilsv2.PathValuesToGnmiChange(pathValues)
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
					Low: uint64(mastershipTerm),
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
			log.Warnf("Configuration '%s' mastership superseded for term %d", config.ID, mastershipTerm)
			return controller.Result{}, nil
		}
		log.Errorf("Failed sending SetRequest %+v", setRequest, err)
		return controller.Result{}, err
	}
	log.Debugf("Received SetResponse %+v", setResponse)

	// Update the configuration state and path statuses
	log.Infof("Configuration '%s' synchronization complete", config.ID)
	config.Status.Applied.Mastership.Master = mastership.NodeId
	config.Status.Applied.Mastership.Term = mastershipTerm
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
