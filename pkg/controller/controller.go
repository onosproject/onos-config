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
	"container/list"
	"github.com/onosproject/onos-config/pkg/types"
	"sync"
	"time"
)

// Watcher is implemented by controllers to implement watching for specific events
type Watcher interface {
	// Start starts watching for events
	Start(ch chan<- types.ID) error

	// Stop stops watching for events
	Stop()
}

// Reconciler reconciles objects
type Reconciler interface {
	// Reconcile is called to reconcile the state of an object
	Reconcile(types.ID) (bool, error)
}

// NewController creates a new controller
func NewController() *Controller {
	return &Controller{
		activator:  &AlwaysActivator{},
		watchers:   make([]Watcher, 0),
		retryQueue: list.New(),
	}
}

// Controller is the request controller
type Controller struct {
	mu         sync.Mutex
	activator  Activator
	filter     Filter
	watchers   []Watcher
	reconciler Reconciler
	retryQueue *list.List
}

// Activate sets an activator for the controller
func (c *Controller) Activate(activator Activator) {
	c.mu.Lock()
	c.activator = activator
	c.mu.Unlock()
}

// Filter sets a filter for the controller
func (c *Controller) Filter(filter Filter) {
	c.mu.Lock()
	c.filter = filter
	c.mu.Unlock()
}

// Watch adds a watcher to the controller
func (c *Controller) Watch(watcher Watcher) {
	c.mu.Lock()
	c.watchers = append(c.watchers, watcher)
	c.mu.Unlock()
}

// Reconcile sets the reconciler for the controller
func (c *Controller) Reconcile(reconciler Reconciler) {
	c.mu.Lock()
	c.reconciler = reconciler
	c.mu.Unlock()
}

// Start starts the request controller
func (c *Controller) Start() error {
	ch := make(chan bool)
	if err := c.activator.Start(ch); err != nil {
		return err
	}
	go func() {
		for activate := range ch {
			if activate {
				c.activate()
			} else {
				c.deactivate()
			}
		}
	}()
	return nil
}

// activate activates the controller
func (c *Controller) activate() {
	ch := make(chan types.ID)
	for _, watcher := range c.watchers {
		go func(watcher Watcher) {
			_ = watcher.Start(ch)
		}(watcher)
	}
	go c.process(ch)
}

// deactivate deactivates the controller
func (c *Controller) deactivate() {
	for _, watcher := range c.watchers {
		watcher.Stop()
	}
}

// process processes the events from the given channel
func (c *Controller) process(ch chan types.ID) {
	c.mu.Lock()
	filter := c.filter
	c.mu.Unlock()
	for id := range ch {
		if filter == nil || filter.Accept(id) {
			c.reconcile(id)
		}
	}
}

// retry retries the given list of requests
func (c *Controller) retry(requests *list.List) {
	element := requests.Front()
	for element != nil {
		c.reconcile(element.Value.(types.ID))
	}
}

// reconcile reconciles the given request ID until complete
func (c *Controller) reconcile(id types.ID) {
	c.mu.Lock()
	reconciler := c.reconciler
	c.mu.Unlock()

	iteration := 1
	for {
		c.mu.Lock()

		// First, check if any requests are pending in the retry queue
		if c.retryQueue.Len() > 0 {
			retries := c.retryQueue
			c.retryQueue = list.New()
			c.mu.Unlock()
			c.retry(retries)
			c.mu.Lock()
		}

		succeeded, err := reconciler.Reconcile(id)
		if err != nil {
			c.mu.Unlock()
			time.Sleep(time.Duration(iteration*2) * time.Millisecond)
		} else if !succeeded {
			c.retryQueue.PushBack(id)
			c.mu.Unlock()
			return
		} else {
			c.mu.Unlock()
		}
		iteration++
	}
}
