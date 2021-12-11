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
	"sync"

	"google.golang.org/grpc/connectivity"

	topoapi "github.com/onosproject/onos-api/go/onos/topo"
	"github.com/onosproject/onos-config/pkg/controller/utils"

	"github.com/onosproject/onos-config/pkg/store/topo"
	"github.com/onosproject/onos-lib-go/pkg/controller"

	"github.com/onosproject/onos-config/pkg/southbound/gnmi"
)

const queueSize = 100

// ConnWatcher is a gnmi connection watcher
type ConnWatcher struct {
	conns  gnmi.ConnManager
	cancel context.CancelFunc
	mu     sync.Mutex
	connCh chan gnmi.ConnEvent
}

// Start starts the connection watcher
func (c *ConnWatcher) Start(ch chan<- controller.ID) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.cancel != nil {
		return nil
	}

	c.connCh = make(chan gnmi.ConnEvent, queueSize)
	ctx, cancel := context.WithCancel(context.Background())
	err := c.conns.Watch(ctx, c.connCh)
	if err != nil {
		cancel()
		return err
	}
	c.cancel = cancel

	go func() {
		for connEvent := range c.connCh {
			conn := connEvent.Conn
			log.Infof("Received gNMI Connection event for connection '%s'", connEvent.EventType)
			switch connEvent.EventType {
			case gnmi.Connected:
				log.Infof("Test here 0")
				ch <- controller.NewID(connEvent.Conn.TargetID())
				connStateCh := make(chan connectivity.State)
				err = conn.WatchConnState(ctx, connStateCh)
				go func() {
					for connState := range connStateCh {
						log.Debugf("Received gNMI Connection state event for connection '%s'", connState.String())
						ch <- controller.NewID(connEvent.Conn.TargetID())
					}
				}()
			}

		}
		close(ch)
	}()
	return nil
}

// Stop stops the connection watcher
func (c *ConnWatcher) Stop() {
	c.mu.Lock()
	if c.cancel != nil {
		c.cancel()
		c.cancel = nil
	}
	c.mu.Unlock()
}

// TopoWatcher is a topology watcher
type TopoWatcher struct {
	topo   topo.Store
	conns  gnmi.ConnManager
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
			conns, err := w.conns.List(ctx)
			if err != nil {
				log.Warnf("cannot retrieve the list conns %s", err)
				continue
			}
			if relation, ok := event.Object.Obj.(*topoapi.Object_Relation); ok {
				if relation.Relation.KindID == topoapi.CONTROLS {
					log.Debugf("Received control relation event: %+v", event.Object.ID)
					for _, conn := range conns {
						if conn.ID() == gnmi.ConnID(event.Object.ID) {
							ch <- controller.NewID(conn.TargetID())
						}
					}
				}
			}
			if entity, ok := event.Object.Obj.(*topoapi.Object_Entity); ok &&
				entity.Entity.KindID == topoapi.ONOS_CONFIG && event.Type == topoapi.EventType_REMOVED {
				log.Debugf("Received onos-config topo event '%s'", event.Object.ID)
				controlRelationSrcIDFilter := &topoapi.Filters{
					RelationFilter: &topoapi.RelationFilter{
						RelationKind: topoapi.CONTROLS,
						SrcId:        string(utils.GetOnosConfigID()),
					},
				}

				relations, err := w.topo.List(ctx, controlRelationSrcIDFilter)
				if err != nil {
					log.Warn(err)
					continue
				}

				for _, relation := range relations {
					ch <- controller.NewID(relation.GetRelation().GetTgtEntityID())
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
