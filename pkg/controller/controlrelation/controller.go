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

package controlrelation

import (
	"context"
	"time"

	"google.golang.org/grpc/connectivity"

	"github.com/onosproject/onos-config/pkg/controller/utils"

	"github.com/onosproject/onos-lib-go/pkg/errors"

	topoapi "github.com/onosproject/onos-api/go/onos/topo"

	"github.com/onosproject/onos-config/pkg/southbound/gnmi"
	"github.com/onosproject/onos-config/pkg/store/topo"
	"github.com/onosproject/onos-lib-go/pkg/controller"
	"github.com/onosproject/onos-lib-go/pkg/logging"
)

var log = logging.GetLogger("controller", "controlrelation")

const (
	defaultTimeout = 30 * time.Second
)

// NewController returns a new control relation  controller
func NewController(topo topo.Store, conns gnmi.ConnManager) *controller.Controller {
	c := controller.NewController("controlrelation")

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

// Reconciler reconciles control relations
type Reconciler struct {
	conns gnmi.ConnManager
	topo  topo.Store
}

// Reconcile reconciles control relations
func (r *Reconciler) Reconcile(id controller.ID) (controller.Result, error) {
	ctx, cancel := context.WithTimeout(context.Background(), defaultTimeout)
	defer cancel()

	targetID := id.Value.(topoapi.ID)
	log.Infof("Reconciling control relation for target %s", targetID)
	if ok, err := r.reconcileControlRelation(ctx, targetID); err != nil {
		return controller.Result{}, err
	} else if ok {
		return controller.Result{}, nil
	}
	return controller.Result{}, nil

}

func (r *Reconciler) reconcileControlRelation(ctx context.Context, targetID topoapi.ID) (bool, error) {
	conn, err := r.conns.Get(ctx, targetID)
	if err != nil {
		if !errors.IsNotFound(err) {
			log.Warnf("Failed to reconcile control relation for target %s, %v", targetID, err)
			return false, err
		}
		// if the connection is not found, find the associated control relation and delete it from topo
		controlRelationFilter := getControlRelationFilter(utils.GetOnosConfigID(), targetID)
		controlRelations, err := r.topo.List(ctx, controlRelationFilter)
		if err != nil {
			log.Warnf("Failed to reconcile control relation for target %s, %v", targetID, err)
			return false, err
		}

		return r.reconcileDeleteE2ControlRelation(ctx, &controlRelations[0])

	}

	currentState := conn.GetClientConn().GetState()
	if currentState == connectivity.Ready {
		log.Infof("Creating control relation for connection %s", conn.ID())
		// creates a control relation
		object := &topoapi.Object{
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
		err = r.topo.Create(ctx, object)
		if err != nil {
			if !errors.IsAlreadyExists(err) {
				log.Warnf("Failed to reconcile control relation for connection %s, %v", conn.ID(), err)
				return false, err
			}
			return false, nil
		}
	} else if currentState == connectivity.TransientFailure || currentState == connectivity.Shutdown {
		controlRelation, err := r.topo.Get(ctx, topoapi.ID(conn.ID()))
		if err != nil {
			if !errors.IsNotFound(err) {
				log.Warnf("Failed to reconcile control relation for connection %s, %v", conn.ID(), err)
				return false, err
			}
			return false, nil
		}
		return r.reconcileDeleteE2ControlRelation(ctx, controlRelation)
	} else if currentState == connectivity.Idle {
		conn.GetClientConn().Connect()
		return false, nil
	}

	return true, nil
}

func (r *Reconciler) reconcileDeleteE2ControlRelation(ctx context.Context, object *topoapi.Object) (bool, error) {
	err := r.topo.Delete(ctx, object)
	if err != nil {
		if !errors.IsNotFound(err) {
			log.Warnf("Failed to reconcile deleting the control relation '%s,  %v", object.ID, err)
			return false, err
		}
		return false, nil
	}
	return true, nil
}

func getControlRelationFilter(srcID, targetID topoapi.ID) *topoapi.Filters {
	controlRelationFilter := &topoapi.Filters{
		RelationFilter: &topoapi.RelationFilter{
			SrcId:        string(srcID),
			RelationKind: topoapi.CONTROLS,
			TargetId:     string(targetID)},
	}

	return controlRelationFilter
}
