// SPDX-FileCopyrightText: 2020-present Open Networking Foundation <info@opennetworking.org>
//
// SPDX-License-Identifier: Apache-2.0

package mastership

import (
	"context"
	configapi "github.com/onosproject/onos-api/go/onos/config/v2"
	"github.com/onosproject/onos-config/pkg/store/configuration"
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
		for event := range eventCh {
			log.Debugf("Received topo event '%s'", event.Object.ID)
			if relation, ok := event.Object.Obj.(*topoapi.Object_Relation); ok &&
				relation.Relation.KindID == topoapi.CONTROLS {
				srcEntity, err := w.topo.Get(ctx, relation.Relation.SrcEntityID)
				if err != nil {
					log.Warn(err)
				} else if srcEntity.GetEntity().KindID == topoapi.ONOS_CONFIG {
					ch <- controller.NewID(relation.Relation.TgtEntityID)
				}

			}
			if _, ok := event.Object.Obj.(*topoapi.Object_Entity); ok {
				// If the entity object has configurable aspect then the controller
				// can make a connection to it
				err = event.Object.GetAspect(&topoapi.Configurable{})
				if err == nil {
					ch <- controller.NewID(event.Object.ID)
				}
			}

		}
		close(ch)
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
	configurations configuration.Store
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

	err := w.configurations.Watch(ctx, eventCh, configuration.WithReplay())
	if err != nil {
		cancel()
		return err
	}
	w.cancel = cancel
	go func() {
		for event := range eventCh {
			ch <- controller.NewID(topoapi.ID(event.Configuration.TargetID))
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
