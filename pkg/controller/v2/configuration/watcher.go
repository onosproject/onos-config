// SPDX-FileCopyrightText: 2020-present Open Networking Foundation <info@opennetworking.org>
//
// SPDX-License-Identifier: Apache-2.0

package configuration

import (
	"context"
	"sync"

	configapi "github.com/onosproject/onos-api/go/onos/config/v2"

	"github.com/onosproject/onos-config/pkg/store/v2/configuration"

	topoapi "github.com/onosproject/onos-api/go/onos/topo"
	"github.com/onosproject/onos-config/pkg/store/topo"
	"github.com/onosproject/onos-lib-go/pkg/controller"
)

const queueSize = 100

// Watcher configuration store watcher
type Watcher struct {
	configurations configuration.Store
	cancel         context.CancelFunc
	mu             sync.Mutex
}

// Start starts the watcher
func (w *Watcher) Start(ch chan<- controller.ID) error {
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
			ch <- controller.NewID(event.Configuration.ID)
		}
	}()
	return nil
}

// Stop stops the watcher
func (w *Watcher) Stop() {
	w.mu.Lock()
	if w.cancel != nil {
		w.cancel()
		w.cancel = nil
	}
	w.mu.Unlock()
}

// TopoWatcher topology watcher
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
			if _, ok := event.Object.Obj.(*topoapi.Object_Entity); ok {
				configurable := &topoapi.Configurable{}
				if err = event.Object.GetAspect(configurable); err == nil {
					ch <- controller.NewID(configuration.NewID(configapi.TargetID(event.Object.ID),
						configapi.TargetType(configurable.Type), configapi.TargetVersion(configurable.Version)))
				}
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
