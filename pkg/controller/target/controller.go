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

package target

import (
	"context"
	"time"

	"github.com/onosproject/onos-lib-go/pkg/errors"

	topoapi "github.com/onosproject/onos-api/go/onos/topo"
	"github.com/onosproject/onos-config/pkg/southbound/gnmi"
	"github.com/onosproject/onos-config/pkg/store/topo"
	"github.com/onosproject/onos-lib-go/pkg/controller"
	"github.com/onosproject/onos-lib-go/pkg/logging"
)

var log = logging.GetLogger("controller", "target")

const (
	defaultTimeout = 30 * time.Second
)

// NewController returns a new gNMI connection  controller
func NewController(topo topo.Store, conns gnmi.ConnManager) *controller.Controller {
	c := controller.NewController("connection")
	c.Watch(&TopoWatcher{
		topo: topo,
	})
	c.Reconcile(&Reconciler{
		conns: conns,
		topo:  topo,
	})
	return c
}

// Reconciler reconciles gNMI connections
type Reconciler struct {
	conns gnmi.ConnManager
	topo  topo.Store
}

// Reconcile reconciles a connection for a gnmi target
func (r *Reconciler) Reconcile(id controller.ID) (controller.Result, error) {
	ctx, cancel := context.WithTimeout(context.Background(), defaultTimeout)
	defer cancel()

	targetID := id.Value.(topoapi.ID)
	log.Infof("Reconciling Target '%s'", targetID)
	target, err := r.topo.Get(ctx, targetID)
	if err != nil {
		if !errors.IsNotFound(err) {
			log.Errorf("Failed reconciling Target '%s'", targetID, err)
			return controller.Result{}, err
		}
		return r.disconnect(ctx, targetID)
	}
	return r.connect(ctx, target)
}

func (r *Reconciler) connect(ctx context.Context, target *topoapi.Object) (controller.Result, error) {
	log.Info("Connecting to Target '%s'", target.ID)
	if err := r.conns.Connect(ctx, target); err != nil {
		if !errors.IsAlreadyExists(err) {
			log.Errorf("Failed connecting to Target '%s'", target.ID, err)
			return controller.Result{}, err
		}
		log.Warnf("Failed connecting to Target '%s'", target.ID, err)
		return controller.Result{}, nil
	}
	return controller.Result{}, nil
}

func (r *Reconciler) disconnect(ctx context.Context, targetID topoapi.ID) (controller.Result, error) {
	log.Info("Disconnecting from Target '%s'", targetID)
	if err := r.conns.Disconnect(ctx, targetID); err != nil {
		if !errors.IsNotFound(err) {
			log.Errorf("Failed disconnecting from Target '%s'", targetID, err)
			return controller.Result{}, err
		}
		log.Warnf("Failed disconnecting from Target '%s'", targetID, err)
		return controller.Result{}, nil
	}
	return controller.Result{}, nil
}
