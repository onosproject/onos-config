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

// Reconcile reconciles configurations
func (r *Reconciler) Reconcile(id controller.ID) (controller.Result, error) {
	ctx, cancel := context.WithTimeout(context.Background(), defaultTimeout)
	defer cancel()

	configurationID := id.Value.(configapi.ConfigurationID)
	config, err := r.configurations.Get(ctx, configurationID)
	if err != nil {
		if !errors.IsNotFound(err) {
			log.Warnf("Failed to reconcile configuration %s, %s", configurationID, err)
			return controller.Result{}, err
		}
		log.Debugf("Configuration %s not found", configurationID)
		return controller.Result{}, nil
	}
	if config.Status.State != configapi.ConfigurationState_CONFIGURATION_PENDING {
		log.Debugf("Skipping Reconciliation of configuration '%s', its configuration state is: %s", configurationID, config.Status.GetState())
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
	conn, err := r.conns.Get(ctx, topoapi.ID(config.TargetID))
	if err != nil {
		if !errors.IsNotFound(err) {
			return false, err
		}
		return false, nil
	}

	rootPath := &gpb.Path{Elem: make([]*gpb.PathElem, 0)}
	getRootReq := &gpb.GetRequest{
		Path:     []*gpb.Path{rootPath},
		Encoding: gpb.Encoding_JSON,
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
			configValues, err := modelPlugin.GetPathValues(ctx, "", update.GetVal().GetJsonVal())
			if err != nil {
				return false, err
			}

			log.Debugf("Notification update in json format: %+v", update.GetVal().GetJsonVal())
			currentConfigValues = append(currentConfigValues, configValues...)
		}
	}
	log.Debugf("Current config values: %v", currentConfigValues)

	desiredConfigValues := config.Values
	var setRequestChanges []*configapi.PathValue
	for _, desiredConfigValue := range desiredConfigValues {
		for _, currentConfigValue := range currentConfigValues {
			if desiredConfigValue.Path == currentConfigValue.Path {
				setRequestChanges = append(setRequestChanges, desiredConfigValue)
			}
		}
	}
	log.Debugf("Set request changes: %v", setRequestChanges)
	setRequest, err := utilsv2.PathValuesToGnmiChange(setRequestChanges)
	if err != nil {
		return false, err
	}
	log.Debugf("Set request is created for configuration %s: %v", config.ID, setRequest)
	setResponse, err := conn.Set(ctx, setRequest)
	if err != nil {
		return false, err
	}
	log.Debugf("Reconciling configuration %s: set response is received %v", config.ID, setResponse)

	config.Status.State = configapi.ConfigurationState_CONFIGURATION_COMPLETE
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
