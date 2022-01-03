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

package connection

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

var log = logging.GetLogger("controller", "connection")

const (
	defaultTimeout = 30 * time.Second
)

// NewController returns a new gNMI connection  controller
func NewController(topo topo.Store, conns gnmi.ConnManager) *controller.Controller {
	c := controller.NewController("connection")

	c.Watch(&ConnWatcher{
		conns: conns,
	})

	c.Watch(&TopoWatcher{
		topo:  topo,
		conns: conns,
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
	log.Infof("Reconciling connection for target %s", targetID)
	if ok, err := r.reconcileConnection(ctx, targetID); err != nil {
		return controller.Result{}, err
	} else if ok {
		return controller.Result{}, nil
	}

	return controller.Result{}, nil

}

func (r *Reconciler) reconcileConnection(ctx context.Context, targetID topoapi.ID) (bool, error) {
	// Checks the target does exist before opening a new connection
	// if the target does not exist, but we have a connection in the list of current
	// connections then the connection should be closed and removed from the list of connections, if an error occurs then it retires
	target, err := r.topo.Get(ctx, targetID)
	if err != nil {
		if !errors.IsNotFound(err) {
			log.Warnf("Failed to reconcile connection for target '%s': %v", targetID, err)
			return false, err
		}
		_, err := r.conns.Get(ctx, targetID)
		if err != nil {
			if !errors.IsNotFound(err) {
				log.Warnf("Failed to reconcile connection for target '%s': %v ", targetID, err)
				return false, err
			}
			return false, nil
		}
		// Close the connection associated with a target
		err = r.conns.Close(ctx, targetID)
		if err != nil {
			log.Warnf("Failed to reconcile connection for target '%s': %v", targetID, err)
			return false, err
		}
		return true, nil
	}

	// Opens a new connection to the target
	conn, err := r.conns.Connect(ctx, target)
	if err != nil {
		if !errors.IsAlreadyExists(err) {
			log.Warnf("Failed to open connection for target '%s', %v", targetID, err)
			return false, err
		}
		return false, nil
	}
	log.Infof("Connection %s for target %s is established successfully", targetID, conn.ID())
	return true, nil
}
