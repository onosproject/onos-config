// SPDX-FileCopyrightText: 2020-present Open Networking Foundation <info@opennetworking.org>
//
// SPDX-License-Identifier: Apache-2.0

package mastership

import (
	"context"
	configapi "github.com/onosproject/onos-api/go/onos/config/v3"
	configurationstore "github.com/onosproject/onos-config/pkg/store/v3/configuration"
	"github.com/onosproject/onos-lib-go/pkg/errors"
	"sync"

	"github.com/onosproject/onos-config/pkg/store/topo"

	topoapi "github.com/onosproject/onos-api/go/onos/topo"
	"github.com/onosproject/onos-lib-go/pkg/controller"
)

const queueSize = 100

// TopoWatcher is a topology watcher
type TopoWatcher struct {
	topo   topo.Store
	cancel context.CancelFunc
	mu     sync.Mutex
}

// Start starts the topo store watcher
func (w *TopoWatcher) Start(ch chan<- controller.ID) error {
	w.mu.Lock()
	defer w.mu.Unlock()
	if w.cancel != nil {
		return nil
	}

	eventCh := make(chan topoapi.Event, queueSize)
	ctx, cancel := context.WithCancel(context.Background())

	err := w.topo.Watch(ctx, eventCh, nil)
	if err != nil {
		cancel()
		return err
	}
	w.cancel = cancel

	go func() {
		defer close(ch)
		for event := range eventCh {
			log.Debugf("Received topo event '%s'", event.Object.ID)
			switch e := event.Object.Obj.(type) {
			// If the event is a relation change and the source is ONOS_CONFIG kind, enqueue the target
			// Configuration to be reconciled.
			case *topoapi.Object_Relation:
				// Get the source entity
				srcEntity, err := w.topo.Get(ctx, e.Relation.SrcEntityID)
				if err != nil {
					if !errors.IsNotFound(err) {
						log.Warn(err)
					}
					continue
				}

				// Check that the source is ONOS_CONFIG kind
				if srcEntity.GetEntity().KindID != topoapi.ONOS_CONFIG {
					continue
				}

				// Get the target entity
				tgtEntity, err := w.topo.Get(ctx, e.Relation.TgtEntityID)
				if err != nil {
					if !errors.IsNotFound(err) {
						log.Warn(err)
					}
					continue
				}

				// Check that the target has the Configurable aspect
				configurable := &topoapi.Configurable{}
				if err := tgtEntity.GetAspect(configurable); err != nil {
					continue
				}

				// Enqueue the associated Configuration for reconciliation
				ch <- controller.NewID(configapi.ConfigurationID{
					Target: configapi.Target{
						ID:      configapi.TargetID(tgtEntity.ID),
						Type:    configapi.TargetType(configurable.Type),
						Version: configapi.TargetVersion(configurable.Version),
					},
				})

			// If the event is an entity change and the entity has a Configurable aspect, enqueue the
			// associated Configuration for reconciliation.
			case *topoapi.Object_Entity:
				// Check that the entity has the Configurable aspect
				configurable := &topoapi.Configurable{}
				if err := event.Object.GetAspect(configurable); err != nil {
					continue
				}

				// Enqueue the associated Configuration for reconciliation
				ch <- controller.NewID(configapi.ConfigurationID{
					Target: configapi.Target{
						ID:      configapi.TargetID(event.Object.ID),
						Type:    configapi.TargetType(configurable.Type),
						Version: configapi.TargetVersion(configurable.Version),
					},
				})
			}
		}
	}()

	return nil
}

// Stop stops the topology watcher
func (w *TopoWatcher) Stop() {
	w.mu.Lock()
	if w.cancel != nil {
		w.cancel()
		w.cancel = nil
	}
	w.mu.Unlock()
}

// ConfigurationStoreWatcher configuration store watcher
type ConfigurationStoreWatcher struct {
	configurations configurationstore.Store
	cancel         context.CancelFunc
	mu             sync.Mutex
}

// Start starts the watcher
func (w *ConfigurationStoreWatcher) Start(ch chan<- controller.ID) error {
	w.mu.Lock()
	defer w.mu.Unlock()
	if w.cancel != nil {
		return nil
	}

	eventCh := make(chan configapi.ConfigurationEvent, queueSize)
	ctx, cancel := context.WithCancel(context.Background())

	err := w.configurations.Watch(ctx, eventCh, configurationstore.WithReplay())
	if err != nil {
		cancel()
		return err
	}
	w.cancel = cancel
	go func() {
		for event := range eventCh {
			ch <- controller.NewID(event.Configuration.ID)
		}
	}()
	return nil
}

// Stop stops the watcher
func (w *ConfigurationStoreWatcher) Stop() {
	w.mu.Lock()
	if w.cancel != nil {
		w.cancel()
		w.cancel = nil
	}
	w.mu.Unlock()
}
