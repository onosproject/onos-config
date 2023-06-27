// SPDX-FileCopyrightText: 2020-present Open Networking Foundation <info@opennetworking.org>
//
// SPDX-License-Identifier: Apache-2.0

package transaction

import (
	"github.com/atomix/go-sdk/pkg/primitive"
	"github.com/atomix/go-sdk/pkg/primitive/indexedmap"
	"github.com/atomix/go-sdk/pkg/types"
	"github.com/google/uuid"
	"io"
	"sync"
	"time"

	"github.com/onosproject/onos-lib-go/pkg/errors"

	"golang.org/x/net/context"

	configapi "github.com/onosproject/onos-api/go/onos/config/v2"

	"github.com/onosproject/onos-lib-go/pkg/logging"
)

var log = logging.GetLogger()

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
func NewAtomixStore(client primitive.Client) (Store, error) {
	transactions, err := indexedmap.NewBuilder[configapi.TransactionID, *configapi.Transaction](client, "transactions").
		Tag("onos-config", "transaction").
		Codec(types.Proto[*configapi.Transaction](&configapi.Transaction{})).
		Get(context.Background())
	if err != nil {
		return nil, errors.FromAtomix(err)
	}
	store := &transactionStore{
		transactions: transactions,
		watchers:     make(map[uuid.UUID]chan<- configapi.TransactionEvent),
		idWatchers:   make(map[configapi.TransactionID]map[uuid.UUID]chan<- configapi.TransactionEvent),
	}
	if err := store.open(); err != nil {
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

type transactionStore struct {
	transactions indexedmap.IndexedMap[configapi.TransactionID, *configapi.Transaction]
	watchers     map[uuid.UUID]chan<- configapi.TransactionEvent
	idWatchers   map[configapi.TransactionID]map[uuid.UUID]chan<- configapi.TransactionEvent
	mu           sync.RWMutex
}

func (s *transactionStore) open() error {
	events, err := s.transactions.Events(context.Background())
	if err != nil {
		return err
	}
	go func() {
		for {
			event, err := events.Next()
			if err == io.EOF {
				break
			}
			if err != nil {
				log.Error(err)
				continue
			}

			var transactionEvent configapi.TransactionEvent
			switch e := event.(type) {
			case *indexedmap.Inserted[configapi.TransactionID, *configapi.Transaction]:
				transaction := e.Entry.Value
				transaction.Index = configapi.Index(e.Entry.Index)
				transaction.Version = uint64(e.Entry.Version)
				transactionEvent = configapi.TransactionEvent{
					Type:        configapi.TransactionEvent_CREATED,
					Transaction: *transaction,
				}
			case *indexedmap.Updated[configapi.TransactionID, *configapi.Transaction]:
				transaction := e.Entry.Value
				transaction.Index = configapi.Index(e.Entry.Index)
				transaction.Version = uint64(e.Entry.Version)
				transactionEvent = configapi.TransactionEvent{
					Type:        configapi.TransactionEvent_UPDATED,
					Transaction: *transaction,
				}
			case *indexedmap.Removed[configapi.TransactionID, *configapi.Transaction]:
				transaction := e.Entry.Value
				transaction.Index = configapi.Index(e.Entry.Index)
				transaction.Version = uint64(e.Entry.Version)
				transactionEvent = configapi.TransactionEvent{
					Type:        configapi.TransactionEvent_DELETED,
					Transaction: *transaction,
				}
			}

			var watchers []chan<- configapi.TransactionEvent
			s.mu.RLock()
			for _, ch := range s.watchers {
				watchers = append(watchers, ch)
			}
			idWatchers, ok := s.idWatchers[transactionEvent.Transaction.ID]
			if ok {
				for _, ch := range idWatchers {
					watchers = append(watchers, ch)
				}
			}
			s.mu.RUnlock()

			for _, ch := range watchers {
				ch <- transactionEvent
			}
		}
	}()
	return nil
}

// Get gets a transaction
func (s *transactionStore) Get(ctx context.Context, id configapi.TransactionID) (*configapi.Transaction, error) {
	// If the transaction is not already in the cache, get it from the underlying primitive.
	entry, err := s.transactions.Get(ctx, id)
	if err != nil {
		return nil, errors.FromAtomix(err)
	}
	transaction := entry.Value
	transaction.Index = configapi.Index(entry.Index)
	transaction.Version = uint64(entry.Version)
	return transaction, nil
}

// GetByIndex gets a transaction by index
func (s *transactionStore) GetByIndex(ctx context.Context, index configapi.Index) (*configapi.Transaction, error) {
	// If the transaction is not already in the cache, get it from the underlying primitive.
	entry, err := s.transactions.GetIndex(ctx, indexedmap.Index(index))
	if err != nil {
		return nil, errors.FromAtomix(err)
	}
	transaction := entry.Value
	transaction.Index = configapi.Index(entry.Index)
	transaction.Version = uint64(entry.Version)
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

	// Append a new entry to the transaction log.
	entry, err := s.transactions.Append(ctx, transaction.ID, transaction)
	if err != nil {
		return errors.FromAtomix(err)
	}
	transaction.Index = configapi.Index(entry.Index)
	transaction.Version = uint64(entry.Version)
	return nil
}

// Update updates an existing transaction
func (s *transactionStore) Update(ctx context.Context, transaction *configapi.Transaction) error {
	if transaction.Revision == 0 {
		return errors.NewInvalid("transaction must contain a revision on update")
	}
	if transaction.Version == 0 {
		return errors.NewInvalid("transaction must contain a version on update")
	}
	transaction.Revision++
	transaction.Updated = time.Now()

	// Update the entry in the transaction log.
	entry, err := s.transactions.Update(ctx, transaction.ID, transaction, indexedmap.IfVersion(primitive.Version(transaction.Version)))
	if err != nil {
		return errors.FromAtomix(err)
	}
	transaction.Index = configapi.Index(entry.Index)
	transaction.Version = uint64(entry.Version)
	return nil
}

// UpdateStatus updates an existing transaction status
func (s *transactionStore) UpdateStatus(ctx context.Context, transaction *configapi.Transaction) error {
	if transaction.Revision == 0 {
		return errors.NewInvalid("transaction must contain a revision on update")
	}
	if transaction.Version == 0 {
		return errors.NewInvalid("transaction must contain a version on update")
	}
	transaction.Updated = time.Now()

	// Update the entry in the transaction log.
	entry, err := s.transactions.Update(ctx, transaction.ID, transaction, indexedmap.IfVersion(primitive.Version(transaction.Version)))
	if err != nil {
		return errors.FromAtomix(err)
	}
	transaction.Index = configapi.Index(entry.Index)
	transaction.Version = uint64(entry.Version)
	return nil
}

// List lists transactions
func (s *transactionStore) List(ctx context.Context) ([]*configapi.Transaction, error) {
	stream, err := s.transactions.List(ctx)
	if err != nil {
		return nil, errors.FromAtomix(err)
	}

	var transactions []*configapi.Transaction
	for {
		entry, err := stream.Next()
		if err == io.EOF {
			return transactions, nil
		}
		if err != nil {
			log.Error(err)
			return nil, err
		}
		transaction := entry.Value
		transaction.Version = uint64(entry.Version)
		transaction.Index = configapi.Index(entry.Index)
		transactions = append(transactions, transaction)
	}
}

func (s *transactionStore) Watch(ctx context.Context, ch chan<- configapi.TransactionEvent, opts ...WatchOption) error {
	var options watchOptions
	for _, opt := range opts {
		opt.apply(&options)
	}

	id := uuid.New()
	eventCh := make(chan configapi.TransactionEvent)
	s.mu.Lock()
	if options.transactionID != "" {
		watchers, ok := s.idWatchers[options.transactionID]
		if !ok {
			watchers = make(map[uuid.UUID]chan<- configapi.TransactionEvent)
			s.idWatchers[options.transactionID] = watchers
		}
		watchers[id] = eventCh
	} else {
		s.watchers[id] = eventCh
	}
	s.mu.Unlock()

	go func() {
		defer func() {
			s.mu.Lock()
			if options.transactionID != "" {
				watchers, ok := s.idWatchers[options.transactionID]
				if ok {
					delete(watchers, id)
					if len(watchers) == 0 {
						delete(s.idWatchers, options.transactionID)
					}
				}
			} else {
				delete(s.watchers, id)
			}
			s.mu.Unlock()
		}()

		if options.replay {
			if options.transactionID != "" {
				entry, err := s.transactions.Get(ctx, options.transactionID)
				if err != nil {
					err = errors.FromAtomix(err)
					if !errors.IsNotFound(err) {
						log.Error(err)
					}
				} else {
					transaction := entry.Value
					transaction.Index = configapi.Index(entry.Index)
					transaction.Version = uint64(entry.Version)
					if ctx.Err() != nil {
						close(ch)
						return
					}
					ch <- configapi.TransactionEvent{
						Type:        configapi.TransactionEvent_REPLAYED,
						Transaction: *transaction,
					}
				}
			} else {
				entries, err := s.transactions.List(ctx)
				if err != nil {
					log.Error(err)
					close(ch)
					return
				}
				for {
					entry, err := entries.Next()
					if err == io.EOF {
						break
					}
					if err != nil {
						log.Error(err)
						continue
					}
					if ctx.Err() != nil {
						close(ch)
						return
					}
					transaction := entry.Value
					transaction.Index = configapi.Index(entry.Index)
					transaction.Version = uint64(entry.Version)
					ch <- configapi.TransactionEvent{
						Type:        configapi.TransactionEvent_REPLAYED,
						Transaction: *transaction,
					}
				}
			}
		}

		for {
			select {
			case event := <-eventCh:
				ch <- event
			case <-ctx.Done():
				close(ch)
				go func() {
					for range eventCh {
					}
				}()
				return
			}
		}
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

// newTransactionID creates a new transaction ID
func newTransactionID() configapi.TransactionID {
	newUUID := configapi.NewUUID()
	return configapi.TransactionID(newUUID.String())
}

var _ Store = &transactionStore{}
