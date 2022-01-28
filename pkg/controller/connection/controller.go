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
	"github.com/onosproject/onos-config/pkg/controller/utils"
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

	connID := id.Value.(gnmi.ConnID)
	conn, ok := r.conns.Get(ctx, connID)
	if !ok {
		return r.deleteRelation(ctx, connID)
	}
	return r.createRelation(ctx, conn)
}

func (r *Reconciler) createRelation(ctx context.Context, conn gnmi.Conn) (controller.Result, error) {
	relation, err := r.topo.Get(ctx, topoapi.ID(conn.ID()))
	if err != nil {
		if !errors.IsNotFound(err) {
			return controller.Result{}, err
		}
		relation = &topoapi.Object{
			ID:   topoapi.ID(conn.ID()),
			Type: topoapi.Object_RELATION,
			Obj: &topoapi.Object_Relation{
				Relation: &topoapi.Relation{
					KindID:      topoapi.CONTROLS,
					SrcEntityID: utils.GetOnosConfigID(),
					TgtEntityID: conn.TargetID(),
				},
			},
		}
		err = r.topo.Create(ctx, relation)
		if err != nil {
			if !errors.IsAlreadyExists(err) {
				return controller.Result{}, err
			}
			return controller.Result{}, nil
		}
	}
	return controller.Result{}, nil
}

func (r *Reconciler) deleteRelation(ctx context.Context, connID gnmi.ConnID) (controller.Result, error) {
	relation, err := r.topo.Get(ctx, topoapi.ID(connID))
	if err != nil {
		if !errors.IsNotFound(err) {
			return controller.Result{}, err
		}
		return controller.Result{}, nil
	}
	err = r.topo.Delete(ctx, relation)
	if err != nil {
		if !errors.IsNotFound(err) {
			return controller.Result{}, err
		}
	}
	return controller.Result{}, nil
}
