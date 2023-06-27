// SPDX-FileCopyrightText: 2020-present Open Networking Foundation <info@opennetworking.org>
//
// SPDX-License-Identifier: Apache-2.0

package proposal

import (
	"context"
	configurationstore "github.com/onosproject/onos-config/pkg/store/v2/configuration"
	proposalstore "github.com/onosproject/onos-config/pkg/store/v2/proposal"
	"sync"

	configapi "github.com/onosproject/onos-api/go/onos/config/v2"

	"github.com/onosproject/onos-lib-go/pkg/controller"
)

const queueSize = 100

// Watcher proposal store watcher
type Watcher struct {
	proposals proposalstore.Store
	cancel    context.CancelFunc
	mu        sync.Mutex
}

// Start starts the watcher
func (w *Watcher) Start(ch chan<- controller.ID) error {
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
func (w *Watcher) Stop() {
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
