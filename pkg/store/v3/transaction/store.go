// SPDX-FileCopyrightText: 2020-present Open Networking Foundation <info@opennetworking.org>
//
// SPDX-License-Identifier: Apache-2.0

package transaction

import (
	"fmt"
	"github.com/atomix/go-sdk/pkg/primitive"
	"github.com/atomix/go-sdk/pkg/primitive/indexedmap"
	_map "github.com/atomix/go-sdk/pkg/primitive/map"
	"github.com/atomix/go-sdk/pkg/types"
	"github.com/google/uuid"
	"io"
	"sync"
	"time"

	"github.com/onosproject/onos-lib-go/pkg/errors"

	"golang.org/x/net/context"

	configapi "github.com/onosproject/onos-api/go/onos/config/v3"

	"github.com/onosproject/onos-lib-go/pkg/logging"
)

var log = logging.GetLogger()

// Store transaction store interface
type Store interface {
	// Get gets a transaction
	Get(ctx context.Context, id configapi.TransactionID) (*configapi.Transaction, error)

	// GetKey gets a transaction by its key
	GetKey(ctx context.Context, target configapi.Target, key string) (*configapi.Transaction, error)

	// Create creates a new transaction
	Create(ctx context.Context, transaction *configapi.Transaction) error

	// Update updates an existing transaction
	Update(ctx context.Context, transaction *configapi.Transaction) error

	// UpdateStatus updates an existing transaction status
	UpdateStatus(ctx context.Context, transaction *configapi.Transaction) error

	// List lists transactions
	List(ctx context.Context) ([]configapi.Transaction, error)

	// Watch watches the transaction store  for changes
	Watch(ctx context.Context, ch chan<- configapi.TransactionEvent, opts ...WatchOption) error

	// Close closes the transaction store
	Close(ctx context.Context) error
}

// NewAtomixStore returns a new persistent Store
func NewAtomixStore(client primitive.Client) (Store, error) {
	targets, err := _map.NewBuilder[string, *configapi.Target](client, "transaction-targets").
		Tag("onos-config", "transaction").
		Codec(types.Proto[*configapi.Target](&configapi.Target{})).
		Get(context.Background())
	if err != nil {
		return nil, errors.FromAtomix(err)
	}
	store := &transactionStore{
		client:     client,
		targets:    targets,
		watchers:   make(map[uuid.UUID]chan<- configapi.TransactionEvent),
		idWatchers: make(map[configapi.TransactionID]map[uuid.UUID]chan<- configapi.TransactionEvent),
	}
	if err := store.open(); err != nil {
		return nil, err
	}
	return store, nil
}

type watchOptions struct {
	TransactionID configapi.TransactionID
	Replay        bool
}

// WatchOption is a configuration option for Watch calls
type WatchOption interface {
	apply(*watchOptions)
}

func newWatchOption(f func(*watchOptions)) WatchOption {
	return funcWatchOption{
		f: f,
	}
}

type funcWatchOption struct {
	f func(*watchOptions)
}

func (o funcWatchOption) apply(options *watchOptions) {
	o.f(options)
}

// WithReplay returns a WatchOption that replays past changes
func WithReplay() WatchOption {
	return newWatchOption(func(options *watchOptions) {
		options.Replay = true
	})
}

// WithTransactionID returns a Watch option that watches for transactions based on a  given transaction ID
func WithTransactionID(id configapi.TransactionID) WatchOption {
	return newWatchOption(func(options *watchOptions) {
		options.TransactionID = id
	})
}

type transactionStore struct {
	client     primitive.Client
	targets    _map.Map[string, *configapi.Target]
	logs       sync.Map
	watchers   map[uuid.UUID]chan<- configapi.TransactionEvent
	idWatchers map[configapi.TransactionID]map[uuid.UUID]chan<- configapi.TransactionEvent
	mu         sync.RWMutex
}

func (s *transactionStore) open() error {
	events, err := s.targets.Watch(context.Background())
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

			_, err = s.newTransactions(context.Background(), *event.Value)
			if err != nil {
				log.Error(err)
			}
		}
	}()

	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()

	entries, err := s.targets.List(context.Background())
	if err != nil {
		return errors.FromAtomix(err)
	}

	wg := &sync.WaitGroup{}
	errCh := make(chan error)
	for {
		entry, err := entries.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}

		wg.Add(1)
		go func() {
			defer wg.Done()
			_, err := s.newTransactions(ctx, *entry.Value)
			if err != nil {
				errCh <- err
				return
			}
		}()
	}

	go func() {
		wg.Wait()
		close(errCh)
	}()
	return <-errCh
}

func (s *transactionStore) getTransactions(ctx context.Context, target configapi.Target) (indexedmap.IndexedMap[string, *configapi.Transaction], error) {
	log, ok := s.logs.Load(target)
	if ok {
		return log.(indexedmap.IndexedMap[string, *configapi.Transaction]), nil
	}

	key := fmt.Sprintf("%s-%s-%s", target.ID, target.Type, target.Version)
	_, err := s.targets.Put(ctx, key, &target)
	if err != nil {
		err = errors.FromAtomix(err)
		if !errors.IsAlreadyExists(err) {
			return nil, err
		}
	}
	return s.newTransactions(ctx, target)
}

func (s *transactionStore) newTransactions(ctx context.Context, target configapi.Target) (indexedmap.IndexedMap[string, *configapi.Transaction], error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	log, ok := s.logs.Load(target)
	if ok {
		return log.(indexedmap.IndexedMap[string, *configapi.Transaction]), nil
	}

	name := fmt.Sprintf("transactions-%s-%s-%s", target.ID, target.Type, target.Version)
	transactions, err := indexedmap.NewBuilder[string, *configapi.Transaction](s.client, name).
		Tag("onos-config", "transaction").
		Codec(types.Proto[*configapi.Transaction](&configapi.Transaction{})).
		Get(ctx)
	if err != nil {
		return nil, errors.FromAtomix(err)
	}

	if err := s.watch(target, transactions); err != nil {
		return nil, err
	}

	s.logs.Store(target, transactions)
	return transactions, nil
}

func (s *transactionStore) watch(target configapi.Target, transactions indexedmap.IndexedMap[string, *configapi.Transaction]) error {
	events, err := transactions.Events(context.Background())
	if err != nil {
		return errors.FromAtomix(err)
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
			case *indexedmap.Inserted[string, *configapi.Transaction]:
				transaction := e.Entry.Value
				transaction.ID = configapi.TransactionID{
					Target: target,
					Index:  configapi.Index(e.Entry.Index),
				}
				transaction.Version = uint64(e.Entry.Version)
				transactionEvent = configapi.TransactionEvent{
					Type:        configapi.TransactionEvent_CREATED,
					Transaction: *transaction,
				}
			case *indexedmap.Updated[string, *configapi.Transaction]:
				transaction := e.Entry.Value
				transaction.ID = configapi.TransactionID{
					Target: target,
					Index:  configapi.Index(e.Entry.Index),
				}
				transaction.Version = uint64(e.Entry.Version)
				transactionEvent = configapi.TransactionEvent{
					Type:        configapi.TransactionEvent_UPDATED,
					Transaction: *transaction,
				}
			case *indexedmap.Removed[string, *configapi.Transaction]:
				transaction := e.Entry.Value
				transaction.ID = configapi.TransactionID{
					Target: target,
					Index:  configapi.Index(e.Entry.Index),
				}
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
func (s *transactionStore) Get(ctx context.Context, transactionID configapi.TransactionID) (*configapi.Transaction, error) {
	// If the transaction is not already in the cache, get it from the underlying primitive.
	transactions, err := s.getTransactions(ctx, transactionID.Target)
	if err != nil {
		return nil, err
	}
	entry, err := transactions.GetIndex(ctx, indexedmap.Index(transactionID.Index))
	if err != nil {
		return nil, errors.FromAtomix(err)
	}
	transaction := entry.Value
	transaction.ID.Index = configapi.Index(entry.Index)
	transaction.Version = uint64(entry.Version)
	return transaction, nil
}

// GetKey gets a transaction by its key
func (s *transactionStore) GetKey(ctx context.Context, target configapi.Target, key string) (*configapi.Transaction, error) {
	// If the transaction is not already in the cache, get it from the underlying primitive.
	transactions, err := s.getTransactions(ctx, target)
	if err != nil {
		return nil, err
	}
	entry, err := transactions.Get(ctx, key)
	if err != nil {
		return nil, errors.FromAtomix(err)
	}
	transaction := entry.Value
	transaction.ID.Index = configapi.Index(entry.Index)
	transaction.Version = uint64(entry.Version)
	return transaction, nil
}

// Create creates a new transaction
func (s *transactionStore) Create(ctx context.Context, transaction *configapi.Transaction) error {
	if transaction.Key == "" {
		transaction.Key = configapi.NewUUID().String()
	}
	if transaction.ID.Target.ID == "" {
		return errors.NewInvalid("transaction target ID is required")
	}
	if transaction.ID.Target.Type == "" {
		return errors.NewInvalid("transaction target Type is required")
	}
	if transaction.ID.Target.Version == "" {
		return errors.NewInvalid("transaction target version is required")
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
	transactions, err := s.getTransactions(ctx, transaction.ID.Target)
	if err != nil {
		return err
	}
	entry, err := transactions.Append(ctx, transaction.Key, transaction)
	if err != nil {
		return errors.FromAtomix(err)
	}
	transaction.ID.Index = configapi.Index(entry.Index)
	transaction.Version = uint64(entry.Version)
	return nil
}

// Update updates an existing transaction
func (s *transactionStore) Update(ctx context.Context, transaction *configapi.Transaction) error {
	if transaction.Key == "" {
		return errors.NewInvalid("transaction key is required")
	}
	if transaction.ID.Target.ID == "" {
		return errors.NewInvalid("transaction target ID is required")
	}
	if transaction.ID.Target.Type == "" {
		return errors.NewInvalid("transaction target Type is required")
	}
	if transaction.ID.Target.Version == "" {
		return errors.NewInvalid("transaction target version is required")
	}
	if transaction.Revision == 0 {
		return errors.NewInvalid("transaction must contain a revision on update")
	}
	if transaction.Version == 0 {
		return errors.NewInvalid("transaction must contain a version on update")
	}
	transaction.Revision++
	transaction.Updated = time.Now()

	// Update the entry in the transaction log.
	transactions, err := s.getTransactions(ctx, transaction.ID.Target)
	if err != nil {
		return err
	}
	entry, err := transactions.Update(ctx, transaction.Key, transaction, indexedmap.IfVersion(primitive.Version(transaction.Version)))
	if err != nil {
		return errors.FromAtomix(err)
	}
	transaction.ID.Index = configapi.Index(entry.Index)
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
	transactions, err := s.getTransactions(ctx, transaction.ID.Target)
	if err != nil {
		return err
	}
	entry, err := transactions.Update(ctx, transaction.Key, transaction, indexedmap.IfVersion(primitive.Version(transaction.Version)))
	if err != nil {
		return errors.FromAtomix(err)
	}
	transaction.ID.Index = configapi.Index(entry.Index)
	transaction.Version = uint64(entry.Version)
	return nil
}

// List lists transactions
func (s *transactionStore) List(ctx context.Context) ([]configapi.Transaction, error) {
	targets, err := s.targets.List(context.Background())
	if err != nil {
		return nil, errors.FromAtomix(err)
	}

	var transactions []configapi.Transaction
	for {
		entry, err := targets.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, err
		}

		log, err := s.getTransactions(ctx, *entry.Value)
		if err != nil {
			return nil, err
		}

		stream, err := log.List(ctx)
		if err != nil {
			return nil, errors.FromAtomix(err)
		}

		for {
			entry, err := stream.Next()
			if err == io.EOF {
				return transactions, nil
			}
			if err != nil {
				return nil, err
			}
			transaction := entry.Value
			transaction.Version = uint64(entry.Version)
			transaction.ID.Index = configapi.Index(entry.Index)
			transactions = append(transactions, *transaction)
		}
	}
	return transactions, nil
}

func (s *transactionStore) Watch(ctx context.Context, ch chan<- configapi.TransactionEvent, opts ...WatchOption) error {
	var options watchOptions
	for _, opt := range opts {
		opt.apply(&options)
	}

	id := uuid.New()
	eventCh := make(chan configapi.TransactionEvent)
	s.mu.Lock()
	if options.TransactionID.Index > 0 {
		watchers, ok := s.idWatchers[options.TransactionID]
		if !ok {
			watchers = make(map[uuid.UUID]chan<- configapi.TransactionEvent)
			s.idWatchers[options.TransactionID] = watchers
		}
		watchers[id] = eventCh
	} else {
		s.watchers[id] = eventCh
	}
	s.mu.Unlock()

	go func() {
		defer func() {
			s.mu.Lock()
			if options.TransactionID.Index > 0 {
				watchers, ok := s.idWatchers[options.TransactionID]
				if ok {
					delete(watchers, id)
					if len(watchers) == 0 {
						delete(s.idWatchers, options.TransactionID)
					}
				}
			} else {
				delete(s.watchers, id)
			}
			s.mu.Unlock()
		}()

		defer close(ch)

		if options.Replay {
			if options.TransactionID.Index > 0 {
				transactions, err := s.getTransactions(ctx, options.TransactionID.Target)
				if err != nil {
					log.Error(err)
					return
				}

				entry, err := transactions.GetIndex(ctx, indexedmap.Index(options.TransactionID.Index))
				if err != nil {
					err = errors.FromAtomix(err)
					if !errors.IsNotFound(err) {
						log.Error(err)
					}
				} else {
					transaction := entry.Value
					transaction.ID.Index = configapi.Index(entry.Index)
					transaction.Version = uint64(entry.Version)
					if ctx.Err() != nil {
						return
					}
					ch <- configapi.TransactionEvent{
						Type:        configapi.TransactionEvent_REPLAYED,
						Transaction: *transaction,
					}
				}
			} else {
				targets, err := s.targets.List(context.Background())
				if err != nil {
					log.Error(err)
					return
				}

				for {
					entry, err := targets.Next()
					if err == io.EOF {
						break
					}
					if err != nil {
						log.Error(err)
						continue
					}

					transactions, err := s.getTransactions(ctx, *entry.Value)
					if err != nil {
						log.Error(err)
						close(ch)
						return
					}

					stream, err := transactions.List(ctx)
					if err != nil {
						log.Error(err)
						return
					}

					for {
						entry, err := stream.Next()
						if err == io.EOF {
							break
						}
						if err != nil {
							log.Error(err)
							continue
						}
						transaction := entry.Value
						transaction.Version = uint64(entry.Version)
						transaction.ID.Index = configapi.Index(entry.Index)
						ch <- configapi.TransactionEvent{
							Type:        configapi.TransactionEvent_REPLAYED,
							Transaction: *transaction,
						}
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
	wg := &sync.WaitGroup{}
	s.logs.Range(func(key, value any) bool {
		wg.Add(1)
		go func() {
			_ = value.(indexedmap.IndexedMap[string, *configapi.Transaction]).Close(ctx)
		}()
		return true
	})
	wg.Wait()
	_ = s.targets.Close(ctx)
	return nil
}

var _ Store = &transactionStore{}
