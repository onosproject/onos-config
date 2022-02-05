// Copyright 2022-present Open Networking Foundation.
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

package proposal

import (
	"context"
	configurationstore "github.com/onosproject/onos-config/pkg/store/configuration"
	proposalstore "github.com/onosproject/onos-config/pkg/store/proposal"
	"sync"

	configapi "github.com/onosproject/onos-api/go/onos/config/v2"

	"github.com/onosproject/onos-lib-go/pkg/controller"
)

const queueSize = 100

// ProposalWatcher proposal store watcher
type ProposalWatcher struct {
	proposals proposalstore.Store
	cancel    context.CancelFunc
	mu        sync.Mutex
}

// Start starts the watcher
func (w *ProposalWatcher) Start(ch chan<- controller.ID) error {
	w.mu.Lock()
	defer w.mu.Unlock()
	if w.cancel != nil {
		return nil
	}

	eventCh := make(chan configapi.ProposalEvent, queueSize)
	ctx, cancel := context.WithCancel(context.Background())

	err := w.proposals.Watch(ctx, eventCh, proposalstore.WithReplay())
	if err != nil {
		cancel()
		return err
	}
	w.cancel = cancel
	go func() {
		for event := range eventCh {
			ch <- controller.NewID(event.Proposal.ID)
		}
	}()
	return nil
}

// Stop stops the watcher
func (w *ProposalWatcher) Stop() {
	w.mu.Lock()
	if w.cancel != nil {
		w.cancel()
		w.cancel = nil
	}
	w.mu.Unlock()
}

// ConfigurationWatcher configuration store watcher
type ConfigurationWatcher struct {
	configurations configurationstore.Store
	cancel         context.CancelFunc
	mu             sync.Mutex
}

// Start starts the watcher
func (w *ConfigurationWatcher) Start(ch chan<- controller.ID) error {
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
			ch <- controller.NewID(proposalstore.NewID(event.Configuration.TargetID, event.Configuration.Index))
			ch <- controller.NewID(proposalstore.NewID(event.Configuration.TargetID, event.Configuration.Status.Applied.Index))
		}
	}()
	return nil
}

// Stop stops the watcher
func (w *ConfigurationWatcher) Stop() {
	w.mu.Lock()
	if w.cancel != nil {
		w.cancel()
		w.cancel = nil
	}
	w.mu.Unlock()
}
