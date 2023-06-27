// SPDX-FileCopyrightText: 2020-present Open Networking Foundation <info@opennetworking.org>
//
// SPDX-License-Identifier: Apache-2.0

package mastership

import (
	"context"
	configapi "github.com/onosproject/onos-api/go/onos/config/v3"
	"github.com/onosproject/onos-config/pkg/controller/utils"
	configurationstore "github.com/onosproject/onos-config/pkg/store/v3/configuration"
	"math/rand"
	"time"

	"github.com/onosproject/onos-lib-go/pkg/errors"

	topoapi "github.com/onosproject/onos-api/go/onos/topo"
	"github.com/onosproject/onos-config/pkg/store/topo"
	"github.com/onosproject/onos-lib-go/pkg/controller"

	"github.com/onosproject/onos-lib-go/pkg/logging"
)

const defaultTimeout = 30 * time.Second

var log = logging.GetLogger("controller", "mastership")

// NewController returns a new mastership controller
func NewController(topo topo.Store, configurations configurationstore.Store) *controller.Controller {
	c := controller.NewController("mastership")
	c.Watch(&TopoWatcher{
		topo: topo,
	})
	c.Watch(&ConfigurationStoreWatcher{
		configurations: configurations,
	})
	c.Reconcile(&Reconciler{
		topo:           topo,
		configurations: configurations,
	})
	return c
}

// Reconciler is mastership reconciler
type Reconciler struct {
	topo           topo.Store
	configurations configurationstore.Store
}

// Reconcile reconciles the mastership state for a gnmi target
func (r *Reconciler) Reconcile(id controller.ID) (controller.Result, error) {
	ctx, cancel := context.WithTimeout(context.Background(), defaultTimeout)
	defer cancel()

	configID := id.Value.(configapi.ConfigurationID)
	config, err := r.configurations.Get(ctx, configID)
	if err != nil {
		if errors.IsNotFound(err) {
			return controller.Result{}, nil
		}
		return controller.Result{}, err
	}

	// List the objects in the topo store
	objects, err := r.topo.List(ctx, &topoapi.Filters{
		RelationFilter: &topoapi.RelationFilter{
			RelationKind: topoapi.CONTROLS,
			Scope:        topoapi.RelationFilterScope_RELATIONS_ONLY,
			SrcId:        string(utils.GetOnosConfigID()),
		},
	})
	if err != nil {
		log.Warnf("Updating MastershipState for Configuration '%s' failed: %v", config.ID, err)
		return controller.Result{}, err
	}
	targetRelations := make(map[topoapi.ID]topoapi.Object)
	for _, object := range objects {
		if object.GetRelation().TgtEntityID == topoapi.ID(config.ID.Target.ID) {
			targetRelations[object.ID] = object
		}
	}

	if _, ok := targetRelations[topoapi.ID(config.Status.Mastership.Master)]; !ok {
		if len(targetRelations) == 0 {
			if config.Status.Mastership.Master == "" {
				return controller.Result{}, nil
			}
			log.Infof("Master in term %d resigned for Configuration '%s'", config.Status.Mastership.Term, config.ID)
			config.Status.Mastership.Master = ""
		} else {
			// Select a random master to assign to the gnmi target
			relations := make([]topoapi.Object, 0, len(targetRelations))
			for _, targetRelation := range targetRelations {
				relations = append(relations, targetRelation)
			}
			relation := relations[rand.Intn(len(relations))]

			// Increment the mastership term and assign the selected master
			config.Status.Mastership.Term++
			config.Status.Mastership.Master = configapi.NodeID(relation.ID)
			log.Infof("Elected new master '%s' in term %d for Configuration '%s'", config.Status.Mastership.Master, config.Status.Mastership.Term, config.ID)
		}

		// Update the Configuration status
		err = r.configurations.UpdateStatus(ctx, config)
		if err != nil {
			if !errors.IsNotFound(err) && !errors.IsConflict(err) {
				log.Warnf("Updating mastership for Configuration '%s' failed: %v", config.ID, err)
				return controller.Result{}, err
			}
			return controller.Result{}, nil
		}
		return controller.Result{}, nil
	}
	return controller.Result{}, nil
}
