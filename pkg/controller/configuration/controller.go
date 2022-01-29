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
	"time"

	controllerutils "github.com/onosproject/onos-config/pkg/controller/utils"

	"github.com/onosproject/onos-config/pkg/pluginregistry"

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
func NewController(topo topo.Store, conns gnmi.ConnManager, configurations configuration.Store, pluginRegistry *pluginregistry.PluginRegistry) *controller.Controller {
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
		pluginRegistry: pluginRegistry,
	})
	return c
}

// Reconciler reconciles configurations
type Reconciler struct {
	conns          gnmi.ConnManager
	topo           topo.Store
	configurations configuration.Store
	pluginRegistry *pluginregistry.PluginRegistry
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

	if ok, err := r.reconcileConfiguration(ctx, config); err != nil {
		return controller.Result{}, err
	} else if ok {
		return controller.Result{}, nil
	}
	return controller.Result{}, nil
}

func (r *Reconciler) reconcileConfiguration(ctx context.Context, config *configapi.Configuration) (bool, error) {
	// Get the target entity from topo
	target, err := r.topo.Get(ctx, topoapi.ID(config.TargetID))
	if err != nil {
		if !errors.IsNotFound(err) {
			log.Errorf("Failed fetching target Entity '%s' from topo", config.TargetID, err)
			return false, err
		}
		log.Debugf("Target entity '%s' not found", config.TargetID)

		// If the target was deleted after a master was already elected for it,
		// SOUND THE ALARM! and revert back to the PENDING state.
		log.Errorf("Mastership state lost for target '%s'. Future configuration changes may not be applicable!")
		config.Status.State = configapi.ConfigurationState_CONFIGURATION_PENDING
		config.Status.MastershipState.Term = 0
		log.Debug(config.Status)
		err := r.configurations.UpdateStatus(ctx, config)
		if err != nil {
			if !errors.IsNotFound(err) && !errors.IsConflict(err) {
				log.Errorf("Failed updating Configuration '%s' status", config.ID, err)
				return false, err
			}
			log.Warnf("Write conflict updating Configuration '%s' status", config.ID, err)
			return false, nil
		}
		return true, nil
	}

	// Get the target mastership state
	mastership := topoapi.MastershipState{}
	_ = target.GetAspect(&mastership)
	mastershipTerm := configapi.MastershipTerm(mastership.Term)

	// Reconcile the PENDING and STALE states in which the control plane configuration
	// is potentially out-of-sync with the target.
	switch config.Status.State {
	// If the configuration is STALE, clear the configured paths and process the
	// configuration again once a new master is assigned.
	case configapi.ConfigurationState_CONFIGURATION_PENDING,
		configapi.ConfigurationState_CONFIGURATION_STALE:
		// If a new master has been assigned, clean the configuration state from
		// the prior term and reconcile the configuration for the new term.
		if mastershipTerm > config.Status.MastershipState.Term {
			log.Infof("Synchronizing Configuration '%s' for new mastership term %d", config.ID, mastershipTerm)
			config.Status.State = configapi.ConfigurationState_CONFIGURATION_UPDATING
			config.Status.MastershipState.Term = mastershipTerm
			config.Status.Paths = nil
			log.Debug(config.Status)
			err := r.configurations.UpdateStatus(ctx, config)
			if err != nil {
				if !errors.IsNotFound(err) && !errors.IsConflict(err) {
					log.Errorf("Failed updating Configuration '%s' status", config.ID, err)
					return false, err
				}
				log.Warnf("Write conflict updating Configuration '%s' status", config.ID, err)
				return false, nil
			}
			return true, nil
		}
		return false, nil
	default:
		// If the configuration is not STALE and mastership has changed, set the
		// configuration state to STALE to force reconciliation of all paths.
		if mastershipTerm > config.Status.MastershipState.Term || mastership.NodeId == "" {
			log.Infof("Mastership term expired; Configuration '%s' may be stale", config.ID)
			config.Status.State = configapi.ConfigurationState_CONFIGURATION_STALE
			log.Debug(config.Status)
			err := r.configurations.UpdateStatus(ctx, config)
			if err != nil {
				if !errors.IsNotFound(err) && !errors.IsConflict(err) {
					log.Errorf("Failed updating Configuration '%s' status", config.ID, err)
					return false, err
				}
				log.Warnf("Write conflict updating Configuration '%s' status", config.ID, err)
				return false, nil
			}
			return true, nil
		}

		// If the revision has changed, trigger propagation of new updates if necessary.
		if config.Revision > config.Status.Revision {
			log.Infof("Updating Configuration '%s' to revision %d", config.ID, config.Revision)
			config.Status.State = configapi.ConfigurationState_CONFIGURATION_UPDATING
			config.Status.Revision = config.Revision
			log.Debug(config.Status)
			err := r.configurations.UpdateStatus(ctx, config)
			if err != nil {
				if !errors.IsNotFound(err) && !errors.IsConflict(err) {
					log.Errorf("Failed updating Configuration '%s' status", config.ID, err)
					return false, err
				}
				log.Warnf("Write conflict updating Configuration '%s' status", config.ID, err)
				return false, nil
			}
			return true, nil
		}
	}

	// The remainder of the algorithm reconciles the state on the target. Ignore completed configurations.
	if config.Status.State == configapi.ConfigurationState_CONFIGURATION_COMPLETE ||
		config.Status.State == configapi.ConfigurationState_CONFIGURATION_FAILED {
		return false, nil
	}

	// If we've made it this far, we know there's a master relation.
	// Get the relation and check whether this node is the source
	relation, err := r.topo.Get(ctx, topoapi.ID(mastership.NodeId))
	if err != nil {
		if !errors.IsNotFound(err) {
			log.Errorf("Failed fetching master Relation '%s' from topo", mastership.NodeId, err)
			return false, err
		}
		log.Warnf("Master relation not found for target '%s'", config.TargetID)
		return false, nil
	}
	if relation.GetRelation().SrcEntityID != controllerutils.GetOnosConfigID() {
		log.Debugf("Not the master for target '%s'", config.TargetID)
		return false, nil
	}

	// Get the master connection
	conn, ok := r.conns.Get(ctx, gnmi.ConnID(relation.ID))
	if !ok {
		log.Warnf("Connection not found for target '%s'", config.TargetID)
		return false, nil
	}

	// While in the UPDATING state, push pending updates to the target via gNMI Set.
	// Construct the set of path/value changes from the configuration status
	if config.Status.Paths == nil {
		config.Status.Paths = make(map[string]*configapi.PathStatus)
	}
	pathValues := make([]*configapi.PathValue, 0, len(config.Status.Paths))
	pathUpdates := make(map[string]configapi.Index)
	for path, pathValue := range config.Values {
		if pathValue.Index > 0 || pathValue.Deleted {
			pathStatus, ok := config.Status.Paths[path]
			if !ok || pathStatus.Index != pathValue.Index {
				pathValues = append(pathValues, pathValue)
				pathUpdates[path] = pathValue.Index
			}
		}
	}
	log.Infof("Updating %d paths on target '%s'", len(pathValues), config.TargetID)

	// Create a gNMI set request
	setRequest, err := utilsv2.PathValuesToGnmiChange(pathValues)
	if err != nil {
		log.Errorf("Failed constructing Set request for Configuration '%s'", config.ID, err)
		config.Status.State = configapi.ConfigurationState_CONFIGURATION_FAILED
		config.Status.Revision = config.Revision
		log.Debug(config.Status)
		err = r.configurations.UpdateStatus(ctx, config)
		if err != nil {
			if !errors.IsConflict(err) && !errors.IsNotFound(err) {
				log.Errorf("Failed updating Configuration '%s' status", config.ID, err)
				return false, err
			}
			log.Warnf("Write conflict updating Configuration '%s' status", config.ID, err)
			return false, nil
		}
		return true, nil
	}

	// Execute the set request
	log.Debugf("Sending SetRequest %+v", setRequest)
	setResponse, err := conn.Set(ctx, setRequest)
	if err != nil {
		log.Errorf("Failed sending SetRequest %+v", setRequest, err)
		return false, err
	}
	log.Debugf("Received SetResponse %+v", setResponse)

	// Update the configuration state and path statuses
	log.Infof("Finalizing Configuration '%s' status", config.ID)
	for path, index := range pathUpdates {
		config.Status.Paths[path] = &configapi.PathStatus{
			Index: index,
		}
	}
	config.Status.State = configapi.ConfigurationState_CONFIGURATION_COMPLETE
	config.Status.Revision = config.Revision
	log.Debug(config.Status)
	err = r.configurations.UpdateStatus(ctx, config)
	if err != nil {
		if !errors.IsConflict(err) && !errors.IsNotFound(err) {
			log.Errorf("Failed updating Configuration '%s' status", config.ID, err)
			return false, err
		}
		log.Warnf("Write conflict updating Configuration '%s' status", config.ID, err)
		return false, nil
	}
	return true, nil
}
