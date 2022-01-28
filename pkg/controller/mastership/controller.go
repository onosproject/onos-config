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

package mastership

import (
	"context"
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
func NewController(topo topo.Store) *controller.Controller {
	c := controller.NewController("mastership")
	c.Watch(&TopoWatcher{
		topo: topo,
	})

	c.Reconcile(&Reconciler{
		topo: topo,
	})
	return c
}

// Reconciler is mastership reconciler
type Reconciler struct {
	topo topo.Store
}

// Reconcile reconciles the mastership state for a gnmi target
func (r *Reconciler) Reconcile(id controller.ID) (controller.Result, error) {
	ctx, cancel := context.WithTimeout(context.Background(), defaultTimeout)
	defer cancel()

	targetID := id.Value.(topoapi.ID)
	log.Infof("Reconciling mastership election for the gNMI target  %s", targetID)
	targetEntity, err := r.topo.Get(ctx, targetID)
	if err != nil {
		if errors.IsNotFound(err) {
			return controller.Result{}, nil
		}
		log.Warnf("Failed to reconcile mastership election for the gNMI target with ID %s: %s", targetEntity.ID, err)
		return controller.Result{}, err
	}

	// List the objects in the topo store
	objects, err := r.topo.List(ctx, &topoapi.Filters{
		KindFilter: &topoapi.Filter{
			Filter: &topoapi.Filter_Equal_{
				Equal_: &topoapi.EqualFilter{
					Value: topoapi.CONTROLS,
				},
			},
		},
	})
	if err != nil {
		log.Warnf("Updating MastershipState for target '%s' failed: %v", targetEntity.GetID(), err)
		return controller.Result{}, err
	}
	controlsRelations := make(map[topoapi.ID]topoapi.Object)
	for _, object := range objects {
		if object.GetRelation().TgtEntityID == targetID {
			controlsRelations[object.ID] = object
		}
	}

	mastership := &topoapi.MastershipState{}
	_ = targetEntity.GetAspect(mastership)
	if _, ok := controlsRelations[topoapi.ID(mastership.NodeId)]; !ok {
		log.Debugf("Updating MastershipState for the gNMI target '%s'", targetEntity.GetID())
		if len(controlsRelations) == 0 {
			if mastership.NodeId == "" {
				log.Warnf("No controls relations found for target entity '%s'", targetEntity.GetID())
			}
			mastership.NodeId = ""
		} else {
			// Select a random master to assign to the gnmi target
			relations := make([]topoapi.Object, 0, len(controlsRelations))
			for _, targetRelation := range relations {
				relations = append(relations, targetRelation)
			}
			relation := relations[rand.Intn(len(relations))]

			// Increment the mastership term and assign the selected master
			mastership.Term++
			mastership.NodeId = string(relation.ID)
		}

		err = targetEntity.SetAspect(mastership)
		if err != nil {
			log.Warnf("Updating MastershipState for gNMI target '%s' failed: %v", targetEntity.GetID(), err)
			return controller.Result{}, err
		}

		// Update the gNMI target entity
		err = r.topo.Update(ctx, targetEntity)
		if err != nil {
			if !errors.IsNotFound(err) {
				log.Warnf("Updating MastershipState for gNMI target '%s' failed: %v", targetEntity.GetID(), err)
				return controller.Result{}, err
			}
			return controller.Result{}, nil
		}
		return controller.Result{}, nil
	}
	return controller.Result{}, nil
}
