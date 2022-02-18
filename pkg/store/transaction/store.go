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
	"github.com/google/uuid"
	"sync"
	"time"

	"github.com/atomix/atomix-go-client/pkg/atomix"

	"github.com/atomix/atomix-go-framework/pkg/atomix/meta"
	"github.com/golang/protobuf/proto"

	"github.com/onosproject/onos-lib-go/pkg/errors"

	"golang.org/x/net/context"

	configapi "github.com/onosproject/onos-api/go/onos/config/v2"

	"github.com/atomix/atomix-go-client/pkg/atomix/indexedmap"
	"github.com/onosproject/onos-lib-go/pkg/logging"
)

var log = logging.GetLogger("store", "transaction")

// Store transaction store interface
type Store interface {
	// Get gets a transaction
	Get(ctx context.Context, id configapi.TransactionID) (*configapi.Transaction, error)

	// GetByIndex gets a transaction by index
	GetByIndex(ctx context.Context, index configapi.Index) (*configapi.Transaction, error)

	// Create creates a new transaction
	Create(ctx context.Context, transaction *configapi.Transaction) error

	// Update updates an existing transaction
	Update(ctx context.Context, transaction *configapi.Transaction) error

	// UpdateStatus updates the status of an existing transaction
	UpdateStatus(ctx context.Context, transaction *configapi.Transaction) error

	// List lists transactions
	List(ctx context.Context) ([]*configapi.Transaction, error)

	// Watch watches the transaction store  for changes
	Watch(ctx context.Context, ch chan<- configapi.TransactionEvent, opts ...WatchOption) error

	// Close closes the transaction store
	Close(ctx context.Context) error
}

// NewAtomixStore returns a new persistent Store
func NewAtomixStore(client atomix.Client) (Store, error) {
	transactions, err := client.GetIndexedMap(context.Background(), "onos-config-transactions")
	if err != nil {
		return nil, errors.FromAtomix(err)
	}
	store := &transactionStore{
		transactions: transactions,
		cacheIDs:     make(map[configapi.TransactionID]*cacheEntry),
		cacheIndexes: make(map[configapi.Index]*cacheEntry),
		watchers:     make(map[uuid.UUID]chan<- configapi.TransactionEvent),
		eventCh:      make(chan configapi.TransactionEvent, 1000),
	}
	if err := store.open(context.Background()); err != nil {
		return nil, err
	}
	return store, nil
}

type watchOptions struct {
	transactionID configapi.TransactionID
	replay        bool
}

// WatchOption is a configuration option for Watch calls
type WatchOption interface {
	apply(*watchOptions)
}

// watchReplyOption is an option to replay events on watch
type watchReplayOption struct {
}

func (o watchReplayOption) apply(options *watchOptions) {
	options.replay = true
}

// WithReplay returns a WatchOption that replays past changes
func WithReplay() WatchOption {
	return watchReplayOption{}
}

type watchIDOption struct {
	id configapi.TransactionID
}

func (o watchIDOption) apply(options *watchOptions) {
	options.transactionID = o.id
}

// WithTransactionID returns a Watch option that watches for transactions based on a  given transaction ID
func WithTransactionID(id configapi.TransactionID) WatchOption {
	return watchIDOption{id: id}
}

type cacheEntry struct {
	*indexedmap.Entry
	prev *cacheEntry
	next *cacheEntry
}

type transactionStore struct {
	transactions indexedmap.IndexedMap
	cacheIDs     map[configapi.TransactionID]*cacheEntry
	cacheIndexes map[configapi.Index]*cacheEntry
	firstEntry   *cacheEntry
	cacheMu      sync.RWMutex
	watchers     map[uuid.UUID]chan<- configapi.TransactionEvent
	watchersMu   sync.RWMutex
	eventCh      chan configapi.TransactionEvent
}

func (s *transactionStore) open(ctx context.Context) error {
	ch := make(chan indexedmap.Event)
	if err := s.transactions.Watch(ctx, ch, indexedmap.WithReplay()); err != nil {
		return err
	}
	go func() {
		for event := range ch {
			entry := event.Entry
			s.updateCache(&entry)
		}
	}()
	go s.processEvents()
	return nil
}

func (s *transactionStore) publishEvent(event configapi.TransactionEvent) {
	s.eventCh <- event
}

func (s *transactionStore) processEvents() {
	for event := range s.eventCh {
		s.watchersMu.RLock()
		for _, watcher := range s.watchers {
			watcher <- event
		}
		s.watchersMu.RUnlock()
	}
}

func (s *transactionStore) updateCache(updateEntry *indexedmap.Entry) {
	transactionID := configapi.TransactionID(updateEntry.Key)
	transactionIndex := configapi.Index(updateEntry.Index)

	// Use a double-checked lock when updating the cache.
	// First, check for a more recent version of the transaction already in the cache.
	s.cacheMu.RLock()
	entry, ok := s.cacheIDs[transactionID]
	s.cacheMu.RUnlock()
	if ok && entry.Revision >= updateEntry.Revision {
		return
	}

	// The cache needs to be updated. Acquire a write lock and check once again
	// for a more recent version of the transaction.
	s.cacheMu.Lock()
	defer s.cacheMu.Unlock()
	entry, ok = s.cacheIDs[transactionID]
	if !ok {
		newEntry := &cacheEntry{
			Entry: updateEntry,
		}
		if s.firstEntry == nil || newEntry.Index < s.firstEntry.Index {
			s.firstEntry = newEntry
		}
		if prevEntry, ok := s.cacheIndexes[transactionIndex-1]; ok {
			newEntry.prev = prevEntry
			prevEntry.next = newEntry
		}
		if nextEntry, ok := s.cacheIndexes[transactionIndex+1]; ok {
			newEntry.next = nextEntry
			nextEntry.prev = newEntry
		}
		s.cacheIDs[transactionID] = newEntry
		s.cacheIndexes[transactionIndex] = newEntry

		var transaction configapi.Transaction
		if err := decodeTransaction(updateEntry, &transaction); err != nil {
			log.Error(err)
		} else {
			s.eventCh <- configapi.TransactionEvent{
				Type:        configapi.TransactionEvent_CREATED,
				Transaction: transaction,
			}
		}
	} else if updateEntry.Revision > entry.Revision {
		// Add the transaction to the ID and index caches and publish an event.
		newEntry := &cacheEntry{
			Entry: updateEntry,
			prev:  entry.prev,
			next:  entry.next,
		}
		if newEntry.prev != nil {
			newEntry.prev.next = newEntry
		} else {
			s.firstEntry = newEntry
		}
		if newEntry.next != nil {
			newEntry.next.prev = newEntry
		}
		s.cacheIDs[transactionID] = newEntry
		s.cacheIndexes[transactionIndex] = newEntry

		var transaction configapi.Transaction
		if err := decodeTransaction(updateEntry, &transaction); err != nil {
			log.Error(err)
		} else {
			s.eventCh <- configapi.TransactionEvent{
				Type:        configapi.TransactionEvent_UPDATED,
				Transaction: transaction,
			}
		}
	}
}

// Get gets a transaction
func (s *transactionStore) Get(ctx context.Context, id configapi.TransactionID) (*configapi.Transaction, error) {
	// Check the ID cache for the latest version of the transaction.
	s.cacheMu.RLock()
	cachedEntry, ok := s.cacheIDs[id]
	s.cacheMu.RUnlock()
	if ok {
		transaction := &configapi.Transaction{}
		if err := decodeTransaction(cachedEntry.Entry, transaction); err != nil {
			return nil, errors.NewInvalid("proposal decoding failed: %v", err)
		}
		return transaction, nil
	}

	// If the transaction is not already in the cache, get it from the underlying primitive.
	entry, err := s.transactions.Get(ctx, string(id))
	if err != nil {
		return nil, errors.FromAtomix(err)
	}

	// Decode the transaction bytes.
	transaction := &configapi.Transaction{}
	if err := decodeTransaction(entry, transaction); err != nil {
		return nil, errors.NewInvalid("transaction decoding failed: %v", err)
	}

	// Update the cache before returning the transaction.
	s.updateCache(entry)
	return transaction, nil
}

// GetByIndex gets a transaction by index
func (s *transactionStore) GetByIndex(ctx context.Context, index configapi.Index) (*configapi.Transaction, error) {
	// Check the index cache for the latest version of the transaction.
	s.cacheMu.RLock()
	cachedEntry, ok := s.cacheIndexes[index]
	s.cacheMu.RUnlock()
	if ok {
		transaction := &configapi.Transaction{}
		if err := decodeTransaction(cachedEntry.Entry, transaction); err != nil {
			return nil, errors.NewInvalid("proposal decoding failed: %v", err)
		}
		return transaction, nil
	}

	// If the transaction is not already in the cache, get it from the underlying primitive.
	entry, err := s.transactions.GetIndex(ctx, indexedmap.Index(index))
	if err != nil {
		return nil, errors.FromAtomix(err)
	}

	// Decode the transaction bytes.
	transaction := &configapi.Transaction{}
	if err := decodeTransaction(entry, transaction); err != nil {
		return nil, errors.NewInvalid("transaction decoding failed: %v", err)
	}

	// Update the cache before returning the transaction.
	s.updateCache(entry)
	return transaction, nil
}

// Create creates a new transaction
func (s *transactionStore) Create(ctx context.Context, transaction *configapi.Transaction) error {
	if transaction.ID == "" {
		transaction.ID = newTransactionID()
	}
	if transaction.Version != 0 {
		return errors.NewInvalid("not a new object")
	}
	if transaction.Revision != 0 {
		return errors.NewInvalid("not a new object")
	}
	transaction.Revision = 1
	transaction.Created = time.Now()
	transaction.Updated = time.Now()

	// Encode the transaction bytes.
	bytes, err := proto.Marshal(transaction)
	if err != nil {
		return errors.NewInvalid("transaction encoding failed: %v", err)
	}

	// Append a new entry to the transaction log.
	entry, err := s.transactions.Append(ctx, string(transaction.ID), bytes)
	if err != nil {
		return errors.FromAtomix(err)
	}

	// Decode the transaction from the returned entry bytes.
	if err := decodeTransaction(entry, transaction); err != nil {
		return errors.NewInvalid("transaction decoding failed: %v", err)
	}

	// Update the cache.
	s.updateCache(entry)
	return nil
}

// Update updates an existing transaction
func (s *transactionStore) Update(ctx context.Context, transaction *configapi.Transaction) error {
	if transaction.Revision == 0 {
		return errors.NewInvalid("configuration must contain a revision on update")
	}
	if transaction.Version == 0 {
		return errors.NewInvalid("configuration must contain a version on update")
	}
	transaction.Revision++
	transaction.Updated = time.Now()

	// Encode the transaction bytes.
	bytes, err := proto.Marshal(transaction)
	if err != nil {
		return errors.NewInvalid("change encoding failed: %v", err)
	}

	// Update the entry in the transaction log.
	entry, err := s.transactions.Set(ctx, indexedmap.Index(transaction.Index), string(transaction.ID), bytes, indexedmap.IfMatch(meta.NewRevision(meta.Revision(transaction.Version))))
	if err != nil {
		return errors.FromAtomix(err)
	}

	// Decode the transaction from the returned entry bytes.
	if err := decodeTransaction(entry, transaction); err != nil {
		return errors.NewInvalid("transaction decoding failed: %v", err)
	}

	// Update the cache.
	s.updateCache(entry)
	return nil
}

// UpdateStatus updates an existing transaction status
func (s *transactionStore) UpdateStatus(ctx context.Context, transaction *configapi.Transaction) error {
	if transaction.Revision == 0 {
		return errors.NewInvalid("configuration must contain a revision on update")
	}
	if transaction.Version == 0 {
		return errors.NewInvalid("configuration must contain a version on update")
	}
	transaction.Updated = time.Now()

	// Encode the transaction bytes.
	bytes, err := proto.Marshal(transaction)
	if err != nil {
		return errors.NewInvalid("change encoding failed: %v", err)
	}

	// Update the entry in the transaction log.
	entry, err := s.transactions.Set(ctx, indexedmap.Index(transaction.Index), string(transaction.ID), bytes, indexedmap.IfMatch(meta.NewRevision(meta.Revision(transaction.Version))))
	if err != nil {
		return errors.FromAtomix(err)
	}

	// Decode the transaction from the returned entry bytes.
	if err := decodeTransaction(entry, transaction); err != nil {
		return errors.NewInvalid("transaction decoding failed: %v", err)
	}

	// Update the cache.
	s.updateCache(entry)
	return nil
}

// List lists transactions
func (s *transactionStore) List(ctx context.Context) ([]*configapi.Transaction, error) {
	indexMapCh := make(chan indexedmap.Entry)
	if err := s.transactions.Entries(ctx, indexMapCh); err != nil {
		return nil, errors.FromAtomix(err)
	}

	transactions := make([]*configapi.Transaction, 0)
	for entry := range indexMapCh {
		transaction := &configapi.Transaction{}
		if err := decodeTransaction(&entry, transaction); err != nil {
			log.Error(err)
		} else {
			transactions = append(transactions, transaction)
		}
	}
	return transactions, nil
}

// Watch watches the transaction store  for changes
func (s *transactionStore) Watch(ctx context.Context, ch chan<- configapi.TransactionEvent, opts ...WatchOption) error {
	var options watchOptions
	for _, opt := range opts {
		opt.apply(&options)
	}

	watchCh := make(chan configapi.TransactionEvent, 10)
	id := uuid.New()
	s.watchersMu.Lock()
	s.watchers[id] = watchCh
	s.watchersMu.Unlock()

	var replay []configapi.TransactionEvent
	if options.replay {
		if options.transactionID == "" {
			s.cacheMu.RLock()
			replay = make([]configapi.TransactionEvent, 0, len(s.cacheIDs))
			entry := s.firstEntry
			for entry != nil {
				var transaction configapi.Transaction
				if err := decodeTransaction(entry.Entry, &transaction); err != nil {
					log.Error(err)
				} else {
					replay = append(replay, configapi.TransactionEvent{
						Type:        configapi.TransactionEvent_REPLAYED,
						Transaction: transaction,
					})
				}
				entry = entry.next
			}
			s.cacheMu.RUnlock()
		} else {
			s.cacheMu.RLock()
			entry, ok := s.cacheIDs[options.transactionID]
			if ok {
				var transaction configapi.Transaction
				if err := decodeTransaction(entry.Entry, &transaction); err != nil {
					log.Error(err)
				} else {
					replay = []configapi.TransactionEvent{
						{
							Type:        configapi.TransactionEvent_REPLAYED,
							Transaction: transaction,
						},
					}
				}
			}
			s.cacheMu.RUnlock()
		}
	}

	go func() {
		defer close(ch)
		for _, event := range replay {
			ch <- event
		}
		for event := range watchCh {
			if options.transactionID == "" || event.Transaction.ID == options.transactionID {
				ch <- event
			}
		}
	}()

	go func() {
		<-ctx.Done()
		s.watchersMu.Lock()
		delete(s.watchers, id)
		s.watchersMu.Unlock()
		close(watchCh)
	}()
	return nil
}

// Close closes the store
func (s *transactionStore) Close(ctx context.Context) error {
	err := s.transactions.Close(ctx)
	if err != nil {
		return errors.FromAtomix(err)
	}
	return nil
}

func decodeTransaction(entry *indexedmap.Entry, transaction *configapi.Transaction) error {
	if err := proto.Unmarshal(entry.Value, transaction); err != nil {
		return err
	}
	transaction.ID = configapi.TransactionID(entry.Key)
	transaction.Index = configapi.Index(entry.Index)
	transaction.Version = uint64(entry.Revision)
	return nil
}

// newTransactionID creates a new transaction ID
func newTransactionID() configapi.TransactionID {
	newUUID := configapi.NewUUID()
	return configapi.TransactionID(newUUID.String())
}

var _ Store = &transactionStore{}
