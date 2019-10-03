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
		activator:   &UnconditionalActivator{},
		partitioner: &UnaryPartitioner{},
		watchers:    make([]Watcher, 0),
		partitions:  make(map[PartitionKey]chan types.ID),
	}
}

// Controller is the request controller
type Controller struct {
	mu          sync.RWMutex
	activator   Activator
	partitioner WorkPartitioner
	filter      Filter
	watchers    []Watcher
	reconciler  Reconciler
	partitions  map[PartitionKey]chan types.ID
}

// Activate sets an activator for the controller
func (c *Controller) Activate(activator Activator) {
	c.mu.Lock()
	c.activator = activator
	c.mu.Unlock()
}

// Partition partitions work among multiple goroutines for the controller
func (c *Controller) Partition(partitioner WorkPartitioner) {
	c.mu.Lock()
	c.partitioner = partitioner
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
	go c.processEvents(ch)
}

// deactivate deactivates the controller
func (c *Controller) deactivate() {
	for _, watcher := range c.watchers {
		watcher.Stop()
	}
}

// processEvents processes the events from the given channel
func (c *Controller) processEvents(ch chan types.ID) {
	c.mu.RLock()
	filter := c.filter
	partitioner := c.partitioner
	c.mu.RUnlock()
	for id := range ch {
		// Ensure the event is applicable to this controller
		if filter == nil || filter.Accept(id) {
			c.partition(id, partitioner)
		}
	}
}

// partition writes the given request to a partition
func (c *Controller) partition(id types.ID, partitioner WorkPartitioner) {
	iteration := 1
	for {
		// Get the partition key for the object ID
		key, err := partitioner.Partition(id)
		if err != nil {
			time.Sleep(time.Duration(iteration*2) * time.Millisecond)
		} else {
			// Get or create a partition channel for the partition key
			c.mu.RLock()
			partition, ok := c.partitions[key]
			if !ok {
				c.mu.RUnlock()
				c.mu.Lock()
				partition, ok = c.partitions[key]
				if !ok {
					partition = make(chan types.ID)
					c.partitions[key] = partition
					go c.processRequests(partition)
				}
				c.mu.Unlock()
			} else {
				c.mu.RUnlock()
			}
			partition <- id
			return
		}
		iteration++
	}
}

// processRequests processes requests from the given channel
func (c *Controller) processRequests(ch chan types.ID) {
	c.mu.RLock()
	reconciler := c.reconciler
	c.mu.RUnlock()

	for id := range ch {
		// Reconcile the request. If the reconciliation is not successful, requeue the request to be processed
		// after the remaining enqueued events.
		succeeded := c.reconcile(id, reconciler)
		if !succeeded {
			go c.requeueRequest(ch, id)
		}
	}
}

// requeueRequest requeues the given request
func (c *Controller) requeueRequest(ch chan types.ID, id types.ID) {
	ch <- id
}

// reconcile reconciles the given request ID until complete
func (c *Controller) reconcile(id types.ID, reconciler Reconciler) bool {
	iteration := 1
	for {
		// Reconcile the request. If an error occurs, use exponential backoff to retry in order.
		// Otherwise, return the result.
		succeeded, err := reconciler.Reconcile(id)
		if err != nil {
			time.Sleep(time.Duration(iteration*2) * time.Millisecond)
		} else {
			return succeeded
		}
		iteration++
	}
}
