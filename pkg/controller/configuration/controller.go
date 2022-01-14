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

	"github.com/onosproject/onos-config/pkg/utils"

	"github.com/onosproject/onos-config/pkg/pluginregistry"

	topoapi "github.com/onosproject/onos-api/go/onos/topo"

	"github.com/onosproject/onos-lib-go/pkg/errors"

	configapi "github.com/onosproject/onos-api/go/onos/config/v2"
	utilsv2 "github.com/onosproject/onos-config/pkg/utils/values/v2"

	"github.com/onosproject/onos-lib-go/pkg/controller"
	gpb "github.com/openconfig/gnmi/proto/gnmi"

	"github.com/onosproject/onos-config/pkg/southbound/gnmi"
	"github.com/onosproject/onos-config/pkg/store/configuration"
	"github.com/onosproject/onos-config/pkg/store/topo"
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
		topo:           topo,
		configurations: configurations,
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
	config, err := r.configurations.Get(ctx, configapi.TargetID(configurationID))
	if err != nil {
		if !errors.IsNotFound(err) {
			log.Warnf("Failed to reconcile configuration %s, %s", configurationID, err)
			return controller.Result{}, err
		}
		log.Debugf("Configuration %s not found", configurationID)
		return controller.Result{}, nil
	}

	log.Infof("Reconciling configuration %v", config)
	if ok, err := r.reconcileConfiguration(ctx, config); err != nil {
		log.Warnf("Failed to reconcile configuration: %s, %s", configurationID, err)
		return controller.Result{}, err
	} else if ok {
		return controller.Result{}, nil
	}

	return controller.Result{}, nil

}

func (r *Reconciler) reconcileConfiguration(ctx context.Context, config *configapi.Configuration) (bool, error) {

	target, err := r.topo.Get(ctx, topoapi.ID(config.TargetID))
	if err != nil {
		if !errors.IsNotFound(err) {
			return false, err
		}
		return false, nil
	}

	mastership := topoapi.MastershipState{}
	_ = target.GetAspect(&mastership)
	targetMastershipTerm := configapi.MastershipTerm(mastership.Term)

	// If no master is found, skip reconciliation
	if targetMastershipTerm == 0 {
		log.Warnf("Mastership state not found for target '%s'", config.TargetID)
		return false, nil
	}

	targetMasterRelation, err := r.topo.Get(ctx, topoapi.ID(mastership.NodeId))
	if err != nil {
		if !errors.IsNotFound(err) {
			return false, err
		}
		return false, nil
	}

	targetMasterInstanceID := targetMasterRelation.GetRelation().SrcEntityID
	localInstanceID := controllerutils.GetOnosConfigID()

	// If this node is not the master, skip reconciliation of the configuration
	if localInstanceID != targetMasterInstanceID {
		log.Debugf("Skipping Reconciliation of configuration '%s', not the master for target '%s'", config.ID, config.TargetID)
		return false, nil
	}

	conn, err := r.conns.Get(ctx, topoapi.ID(config.TargetID))
	if err != nil {
		if !errors.IsNotFound(err) {
			return false, err
		}
		log.Warnf("Reconciling configuration '%s': connection not found for target %s", config.ID, config.TargetID, err)
		return false, nil
	}

	if config.Status.MastershipState.Term < targetMastershipTerm &&
		config.Status.State != configapi.ConfigurationState_CONFIGURATION_PENDING {
		config.Status.State = configapi.ConfigurationState_CONFIGURATION_PENDING
		err = r.configurations.Update(ctx, config)
		if err != nil {
			if !errors.IsConflict(err) && !errors.IsNotFound(err) {
				return false, err
			}
			return false, nil
		}
		return true, nil
	}

	if config.Status.State != configapi.ConfigurationState_CONFIGURATION_PENDING {
		log.Debugf("Skipping Reconciliation of configuration '%s', its configuration state is: %s", config.TargetID, config.Status.GetState())
		return false, nil
	}

	// If the configuration's mastership term is less than the current mastership term,
	// assume the target may have restarted/reconnected and perform a full reconciliation
	// of the target configuration from the root path.
	var setRequestChanges []*configapi.PathValue
	if config.Status.MastershipState.Term < targetMastershipTerm {
		rootPath := &gpb.Path{Elem: make([]*gpb.PathElem, 0)}
		getRootReq := &gpb.GetRequest{
			Path:     []*gpb.Path{rootPath},
			Encoding: gpb.Encoding_JSON_IETF,
		}
		root, err := conn.Get(ctx, getRootReq)
		if err != nil {
			return false, err
		}

		if len(root.Notification) == 0 {
			return false, errors.NewInvalid("notification list is empty")
		}

		var currentConfigValues []*configapi.PathValue
		for _, notification := range root.Notification {
			for _, update := range notification.Update {
				modelName := utils.ToModelNameV2(config.TargetType, config.TargetVersion)
				modelPlugin, ok := r.pluginRegistry.GetPlugin(modelName)
				if !ok {
					return false, err
				}
				configValues, err := modelPlugin.GetPathValues(ctx, "", update.GetVal().GetJsonIetfVal())
				if err != nil {
					return false, err
				}
				currentConfigValues = append(currentConfigValues, configValues...)
			}
		}
		log.Debugf("Current target %s config values: %v", config.TargetID, currentConfigValues)
		currentConfigValuesMap := make(map[string]*configapi.PathValue, len(currentConfigValues))

		for _, configValue := range currentConfigValues {
			currentConfigValuesMap[configValue.Path] = configValue
		}

		desiredConfigValues := config.Values
		for _, desiredConfigValue := range desiredConfigValues {
			if currentConfigValue, ok := currentConfigValuesMap[desiredConfigValue.Path]; ok {
				if desiredConfigValue.Path == currentConfigValue.Path {
					setRequestChanges = append(setRequestChanges, desiredConfigValue)
				}
			}
		}
		//If the Configuration's transaction index is greater than the target index,
		// reconcile the configuration with the target.
	} else if config.Status.TransactionIndex > config.Status.SyncIndex &&
		targetMastershipTerm == config.Status.MastershipState.Term {
		desiredConfigValues := config.Values
		for _, desiredConfigValue := range desiredConfigValues {
			// Perform partial reconciliation of target configuration (update only paths that have changed)
			if desiredConfigValue.Index > config.Status.SyncIndex {
				setRequestChanges = append(setRequestChanges, desiredConfigValue)
			}
		}
	}

	setRequest, err := utilsv2.PathValuesToGnmiChange(setRequestChanges)
	if err != nil {
		return false, err
	}
	log.Debugf("Reconciling configuration; Set request is created for configuration %s: %v", config.ID, setRequest)
	setResponse, err := conn.Set(ctx, setRequest)
	if err != nil {
		return false, err
	}
	// Once the target has been updated,
	// update the target index to match the reconciled transaction index.
	log.Debugf("Reconciling configuration %s: set response is received %v", config.ID, setResponse)
	config.Status.State = configapi.ConfigurationState_CONFIGURATION_COMPLETE
	config.Status.MastershipState.Term = targetMastershipTerm
	config.Status.SyncIndex = config.Status.TransactionIndex

	err = r.configurations.Update(ctx, config)
	if err != nil {
		if !errors.IsConflict(err) && !errors.IsNotFound(err) {
			return false, err
		}
		return false, nil
	}
	log.Infof("Reconciling configuration %s is complete", config.ID)

	return true, nil
}
