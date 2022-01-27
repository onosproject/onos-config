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

package configuration

import (
	"context"
	"github.com/onosproject/onos-config/pkg/southbound/gnmi"
	"sync"

	configapi "github.com/onosproject/onos-api/go/onos/config/v2"

	"github.com/onosproject/onos-config/pkg/store/configuration"

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
				err = event.Object.GetAspect(&topoapi.Configurable{})
				if err == nil {
					// TODO: Get target type and version from topo entity
					ch <- controller.NewID(configuration.NewID(configapi.TargetID(event.Object.ID), "", ""))
				}
			} else if relation, ok := event.Object.Obj.(*topoapi.Object_Relation); ok && event.Object.GetKind().Name == topoapi.CONTROLS {
				// TODO: Get target type and version from topo entity
				ch <- controller.NewID(configuration.NewID(configapi.TargetID(relation.Relation.TgtEntityID), "", ""))
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

// ConnWatcher connection watcher
type ConnWatcher struct {
	conns  gnmi.ConnManager
	topo   topo.Store
	cancel context.CancelFunc
	mu     sync.Mutex
}

// Start starts the connection watcher
func (w *ConnWatcher) Start(ch chan<- controller.ID) error {
	w.mu.Lock()
	defer w.mu.Unlock()
	if w.cancel != nil {
		return nil
	}

	eventCh := make(chan gnmi.ConnEvent, queueSize)
	ctx, cancel := context.WithCancel(context.Background())

	err := w.conns.Watch(ctx, eventCh)
	if err != nil {
		cancel()
		return err
	}
	w.cancel = cancel
	go func() {
		for event := range eventCh {
			if relation, err := w.topo.Get(ctx, topoapi.ID(event.Conn.ID())); err == nil {
				// TODO: Get target type and version from topo entity
				ch <- controller.NewID(configuration.NewID(configapi.TargetID(relation.GetRelation().TgtEntityID), "", ""))
			}
		}
	}()

	return nil
}

// Stop stops the connection watcher
func (w *ConnWatcher) Stop() {
	w.mu.Lock()
	if w.cancel != nil {
		w.cancel()
		w.cancel = nil
	}
	w.mu.Unlock()
}
