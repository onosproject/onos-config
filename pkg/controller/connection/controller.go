// SPDX-FileCopyrightText: 2020-present Open Networking Foundation <info@opennetworking.org>
//
// SPDX-License-Identifier: Apache-2.0

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
	log.Debugf("Reconciling Conn '%s'", connID)
	conn, ok := r.conns.Get(ctx, connID)
	if !ok {
		return r.deleteRelation(ctx, connID)
	}
	return r.createRelation(ctx, conn)
}

func (r *Reconciler) createRelation(ctx context.Context, conn gnmi.Conn) (controller.Result, error) {
	_, err := r.topo.Get(ctx, topoapi.ID(conn.ID()))
	if err != nil {
		if !errors.IsNotFound(err) {
			log.Errorf("Failed reconciling Conn '%s'", conn.ID(), err)
			return controller.Result{}, err
		}
		log.Infof("Creating CONTROLS relation '%s'", conn.ID())
		relation := &topoapi.Object{
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
				log.Errorf("Failed creating CONTROLS relation '%s'", conn.ID(), err)
				return controller.Result{}, err
			}
			log.Warnf("Failed creating CONTROLS relation '%s'", conn.ID(), err)
			return controller.Result{}, nil
		}
	}
	return controller.Result{}, nil
}

func (r *Reconciler) deleteRelation(ctx context.Context, connID gnmi.ConnID) (controller.Result, error) {
	relation, err := r.topo.Get(ctx, topoapi.ID(connID))
	if err != nil {
		if !errors.IsNotFound(err) {
			log.Errorf("Failed reconciling Conn '%s'", connID, err)
			return controller.Result{}, err
		}
		return controller.Result{}, nil
	}
	log.Infof("Deleting CONTROLS relation '%s'", connID)
	err = r.topo.Delete(ctx, relation)
	if err != nil {
		if !errors.IsNotFound(err) {
			log.Errorf("Failed deleting CONTROLS relation '%s'", connID, err)
			return controller.Result{}, err
		}
		log.Warnf("Failed deleting CONTROLS relation '%s'", connID, err)
		return controller.Result{}, nil
	}
	return controller.Result{}, nil
}
