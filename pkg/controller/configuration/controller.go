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
	"bytes"
	"context"
	"time"

	topoapi "github.com/onosproject/onos-api/go/onos/topo"

	"github.com/onosproject/onos-lib-go/pkg/errors"

	configapi "github.com/onosproject/onos-api/go/onos/config/v2"
	utilsv2 "github.com/onosproject/onos-config/pkg/utils/values/v2"

	"github.com/onosproject/onos-lib-go/pkg/controller"
	gpb "github.com/openconfig/gnmi/proto/gnmi"

	jsonvaluesv2 "github.com/onosproject/onos-config/pkg/modelregistry/jsonvalues/v2"
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
	log.Infof("Reconciling configuration %v", config)
	if config.Status.State != configapi.ConfigurationState_CONFIGURATION_PENDING {
		log.Debugf("Skipping Reconciliation of configuration %s, configuration state is: %s", configurationID, config.Status.GetState())
		return controller.Result{}, nil
	}

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
			configValues, err := jsonvaluesv2.DecomposeJSONWithPaths("", update.GetVal().GetJsonVal(), nil, nil)
			if err != nil {
				return false, err
			}
			currentConfigValues = append(currentConfigValues, configValues...)
		}
	}

	desiredConfigValues := config.Values
	var setRequestValues []*configapi.PathValue
	for _, desiredConfigValue := range desiredConfigValues {
		for _, currentConfigValue := range currentConfigValues {
			if desiredConfigValue.Path == currentConfigValue.Path &&
				!bytes.Equal(desiredConfigValue.Value.GetBytes(), currentConfigValue.Value.GetBytes()) {
				setRequestValues = append(setRequestValues, currentConfigValue)

			}
		}
	}
	setRequest, err := utilsv2.PathValuesToGnmiChange(setRequestValues)
	if err != nil {
		return false, err
	}
	setResponse, err := conn.Set(ctx, setRequest)
	if err != nil {
		return false, err
	}

	log.Debugf("Reconciling configuration, Set response '%v' is received for configuration: %s", setResponse, config.ID)
	return false, nil
}
