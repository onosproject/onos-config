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
	"github.com/onosproject/onos-config/pkg/store/cluster"
	leadershipstore "github.com/onosproject/onos-config/pkg/store/leadership"
	"sync"
)

// Activator is an interface for controlling the activation of a controller
type Activator interface {
	// Start starts the activator
	Start(ch chan<- bool) error

	// Stop stops the activator
	Stop()
}

// UnconditionalActivator activates controllers on all nodes
type UnconditionalActivator struct {
}

// Start starts the activator
func (a *UnconditionalActivator) Start(ch chan<- bool) error {
	ch <- true
	return nil
}

// Stop stops the activator
func (a *UnconditionalActivator) Stop() {

}

var _ Activator = &UnconditionalActivator{}

// LeadershipActivator is an Activator for activating a controller on leadership
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
		for leadership := range a.ch {
			if leadership.Leader == cluster.GetNodeID() {
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

var _ Activator = &LeadershipActivator{}
