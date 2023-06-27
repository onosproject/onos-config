// SPDX-FileCopyrightText: 2020-present Open Networking Foundation <info@opennetworking.org>
//
// SPDX-License-Identifier: Apache-2.0

package transaction

import (
	"context"
	proposalstore "github.com/onosproject/onos-config/pkg/store/v2/proposal"
	transactionstore "github.com/onosproject/onos-config/pkg/store/v2/transaction"
	"sync"

	configapi "github.com/onosproject/onos-api/go/onos/config/v2"

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
			ch <- controller.NewID(event.Transaction.Index)
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
			ch <- controller.NewID(event.Proposal.TransactionIndex)
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
