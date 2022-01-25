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

package transaction

import (
	"context"
	"sync"

	"github.com/onosproject/onos-config/pkg/store/configuration"

	configapi "github.com/onosproject/onos-api/go/onos/config/v2"
	"github.com/onosproject/onos-lib-go/pkg/controller"

	"github.com/onosproject/onos-config/pkg/store/transaction"
)

const queueSize = 100

// ConfigurationWatcher configuration store watcher
type ConfigurationWatcher struct {
	configurations configuration.Store
	transactions   transaction.Store
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

	err := w.configurations.Watch(ctx, eventCh, configuration.WithReplay())
	if err != nil {
		cancel()
		return err
	}
	w.cancel = cancel
	go func() {
		for event := range eventCh {
			configTransactionIndex := event.Configuration.Status.TransactionIndex
			configSyncIndex := event.Configuration.Status.SyncIndex
			configTransaction, err := w.transactions.GetByIndex(ctx, configTransactionIndex)
			if err != nil {
				log.Warn(err)
				continue
			}
			configSyncTransaction, err := w.transactions.GetByIndex(ctx, configSyncIndex)
			if err != nil {
				log.Warn(err)
				continue
			}

			ch <- controller.NewID(configTransaction.ID)
			ch <- controller.NewID(configSyncTransaction.ID)
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

// Watcher transaction watcher changes
type Watcher struct {
	transactions transaction.Store
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

	err := w.transactions.Watch(ctx, eventCh, transaction.WithReplay())
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

// Stop stops the transaction watcher
func (w *Watcher) Stop() {
	w.mu.Lock()
	if w.cancel != nil {
		w.cancel()
		w.cancel = nil
	}
	w.mu.Unlock()
}
