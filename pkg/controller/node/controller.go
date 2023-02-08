// SPDX-FileCopyrightText: 2020-present Open Networking Foundation <info@opennetworking.org>
//
// SPDX-License-Identifier: Apache-2.0

package node

import (
	"context"
	"time"

	gogotypes "github.com/gogo/protobuf/types"

	"github.com/onosproject/onos-lib-go/pkg/errors"

	topoapi "github.com/onosproject/onos-api/go/onos/topo"

	"github.com/onosproject/onos-config/pkg/controller/utils"
	"github.com/onosproject/onos-config/pkg/store/topo"
	"github.com/onosproject/onos-lib-go/pkg/controller"
	"github.com/onosproject/onos-lib-go/pkg/logging"
)

const (
	defaultTimeout            = 30 * time.Second
	defaultExpirationDuration = 30 * time.Second
	defaultGRPCPort           = 5150
	defaultGNMIPort           = 9339
)

var log = logging.GetLogger("controller", "node")

// NewController returns a new node controller
func NewController(topo topo.Store) *controller.Controller {
	c := controller.NewController("node")
	c.Watch(&TopoWatcher{
		topo: topo,
	})

	c.Reconcile(&Reconciler{
		topo: topo,
	})

	return c
}

// Reconciler is a onos-config node reconciler
type Reconciler struct {
	topo topo.Store
}

func (r *Reconciler) createOnosConfigEntity(ctx context.Context, onosConfigID topoapi.ID) error {
	log.Debugf("Creating onos-config entity %s", onosConfigID)
	object := &topoapi.Object{
		ID:   utils.GetOnosConfigID(),
		Type: topoapi.Object_ENTITY,
		Obj: &topoapi.Object_Entity{
			Entity: &topoapi.Entity{
				KindID: topoapi.ONOS_CONFIG,
			},
		},
		Aspects: make(map[string]*gogotypes.Any),
		Labels:  map[string]string{},
	}

	expiration := time.Now().Add(defaultExpirationDuration)
	leaseAspect := &topoapi.Lease{
		Expiration: &expiration,
	}

	err := object.SetAspect(leaseAspect)
	if err != nil {
		return err
	}

	err = r.topo.Create(ctx, object)
	if err != nil && !errors.IsAlreadyExists(err) {
		log.Infof("Creating onos-config entity %s failed: %v", onosConfigID, err)
		return err
	}

	return nil

}

// Reconcile reconciles the onos-config entities
func (r *Reconciler) Reconcile(id controller.ID) (controller.Result, error) {
	ctx, cancel := context.WithTimeout(context.Background(), defaultTimeout)
	defer cancel()

	onosConfigID := id.Value.(topoapi.ID)
	log.Debugf("Reconciling onos-config entity with ID: %s", onosConfigID)
	object, err := r.topo.Get(ctx, onosConfigID)
	if err == nil {
		//  Reconciles an onos-config entity that’s not local so the controller should requeue
		//  it for the lease expiration time and delete the entity if the lease has not been renewed
		if onosConfigID != utils.GetOnosConfigID() {
			lease := &topoapi.Lease{}
			_ = object.GetAspect(lease)

			// Check if the the lease is expired
			if lease.Expiration.Before(time.Now()) {
				log.Debugf("Deleting the expired lease for onos-config with ID: %s", onosConfigID)
				err := r.topo.Delete(ctx, object)
				if !errors.IsNotFound(err) {
					return controller.Result{}, err
				}
				return controller.Result{}, nil
			}

			// Requeue the object to be reconciled at the expiration time
			return controller.Result{
				RequeueAfter: time.Until(*lease.Expiration),
			}, nil
		}

		// Renew the lease If this is the onos-config entity for the local node
		if onosConfigID == utils.GetOnosConfigID() {
			lease := &topoapi.Lease{}

			err := object.GetAspect(lease)
			if err != nil {
				return controller.Result{}, err
			}

			remainingTime := time.Until(*lease.GetExpiration())
			// If the remaining time of lease is more than  half the lease duration, no need to renew the lease
			// schedule the next renewal
			if remainingTime > defaultExpirationDuration/2 {
				log.Debugf("No need to renew the lease for %s, the remaining lease time is %v seconds", onosConfigID, remainingTime)
				return controller.Result{
					RequeueAfter: time.Until(lease.Expiration.Add(defaultExpirationDuration / 2 * -1)),
				}, nil
			}

			// Renew the release to trigger the reconciler
			log.Debugf("Renew the lease for onos-config with ID: %s", onosConfigID)
			expiration := time.Now().Add(defaultExpirationDuration)
			lease = &topoapi.Lease{
				Expiration: &expiration,
			}

			err = object.SetAspect(lease)
			if err != nil {
				return controller.Result{}, err
			}
			err = r.topo.Update(ctx, object)
			if !errors.IsNotFound(err) {
				return controller.Result{}, err
			}
			return controller.Result{}, nil
		}

	} else if !errors.IsNotFound(err) {
		log.Infof("Renewing onos-config entity lease failed for onos-config with ID %s: %v", onosConfigID, err)
		return controller.Result{}, err
	}

	// Create the onos-config entity
	if err := r.createOnosConfigEntity(ctx, onosConfigID); err != nil {
		log.Infof("Creating onos-config entity with ID %s failed: %v", onosConfigID, err)
		return controller.Result{}, err
	}

	return controller.Result{}, nil

}
