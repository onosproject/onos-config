// Copyright 2019-present Open Networking Foundation.
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

package controller

import (
	leadershipstore "github.com/onosproject/onos-config/pkg/store/leadership"
	"github.com/onosproject/onos-lib-go/pkg/controller"
	"sync"
)

// LeadershipActivator is an Activator for activating a controller on leadership
// The LeadershipActivator listens for leadership changes in the leadership store. When the local node
// becomes the leader, the controller is activated. If the local node loses leadership, the controller
// is deactivated. This can ensure only a single controller processes requests in a cluster.
type LeadershipActivator struct {
	Store leadershipstore.Store
	ch    chan leadershipstore.Leadership
	mu    sync.Mutex
}

// Start starts the activator
func (a *LeadershipActivator) Start(ch chan<- bool) error {
	a.mu.Lock()
	a.ch = make(chan leadershipstore.Leadership)
	a.mu.Unlock()

	if err := a.Store.Watch(a.ch); err != nil {
		return err
	}

	go func() {
		if leader, err := a.Store.IsLeader(); err == nil && leader {
			ch <- true
		} else {
			ch <- false
		}

		for leadership := range a.ch {
			if leadership.Leader == a.Store.NodeID() {
				ch <- true
			} else {
				ch <- false
			}
		}
	}()
	return nil
}

// Stop stops the activator
func (a *LeadershipActivator) Stop() {
	a.mu.Lock()
	close(a.ch)
	a.mu.Unlock()
}

var _ controller.Activator = &LeadershipActivator{}
