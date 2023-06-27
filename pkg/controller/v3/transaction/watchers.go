// SPDX-FileCopyrightText: 2020-present Open Networking Foundation <info@opennetworking.org>
//
// SPDX-License-Identifier: Apache-2.0

package transaction

import (
	"context"
	configurationstore "github.com/onosproject/onos-config/pkg/store/v3/configuration"
	transactionstore "github.com/onosproject/onos-config/pkg/store/v3/transaction"
	"sync"

	configapi "github.com/onosproject/onos-api/go/onos/config/v3"

	"github.com/onosproject/onos-lib-go/pkg/controller"
)

const queueSize = 100

// Watcher transaction store watcher
type Watcher struct {
	transactions transactionstore.Store
	cancel       context.CancelFunc
	mu           sync.Mutex
}

// Start starts the watcher
func (w *Watcher) Start(ch chan<- controller.ID) error {
	w.mu.Lock()
	defer w.mu.Unlock()
	if w.cancel != nil {
		return nil
	}

	eventCh := make(chan configapi.TransactionEvent, queueSize)
	ctx, cancel := context.WithCancel(context.Background())

	err := w.transactions.Watch(ctx, eventCh, transactionstore.WithReplay())
	if err != nil {
		cancel()
		return err
	}
	w.cancel = cancel
	go func() {
		for event := range eventCh {
			ch <- controller.NewID(event.Transaction.ID)
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
			ch <- controller.NewID(configapi.TransactionID{
				Target: event.Configuration.ID.Target,
				Index:  event.Configuration.Committed.Target,
			})
			ch <- controller.NewID(configapi.TransactionID{
				Target: event.Configuration.ID.Target,
				Index:  event.Configuration.Applied.Target,
			})
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
