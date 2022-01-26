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
	"github.com/atomix/atomix-go-client/pkg/atomix"
	"time"

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

	// GetPrev gets the previous network change by index
	GetPrev(ctx context.Context, index configapi.Index) (*configapi.Transaction, error)

	// GetNext gets the next transaction by index
	GetNext(ctx context.Context, index configapi.Index) (*configapi.Transaction, error)

	// Create creates a new transaction
	Create(ctx context.Context, transaction *configapi.Transaction) error

	// Update updates an existing transaction
	Update(ctx context.Context, transaction *configapi.Transaction) error

	// UpdateStatus updates the status of an existing transaction
	UpdateStatus(ctx context.Context, transaction *configapi.Transaction) error

	// Delete deletes a transaction
	Delete(ctx context.Context, transaction *configapi.Transaction) error

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
	return &transactionStore{
		transactions: transactions,
	}, nil
}

// WatchOption is a configuration option for Watch calls
type WatchOption interface {
	apply([]indexedmap.WatchOption) []indexedmap.WatchOption
}

// watchReplyOption is an option to replay events on watch
type watchReplayOption struct {
}

func (o watchReplayOption) apply(opts []indexedmap.WatchOption) []indexedmap.WatchOption {
	return append(opts, indexedmap.WithReplay())
}

// WithReplay returns a WatchOption that replays past changes
func WithReplay() WatchOption {
	return watchReplayOption{}
}

type watchIDOption struct {
	id configapi.TransactionID
}

func (o watchIDOption) apply(opts []indexedmap.WatchOption) []indexedmap.WatchOption {
	return append(opts, indexedmap.WithFilter(indexedmap.Filter{
		Key: string(o.id),
	}))
}

// WithTransactionID returns a Watch option that watches for transactions based on a  given transaction ID
func WithTransactionID(id configapi.TransactionID) WatchOption {
	return watchIDOption{id: id}
}

type transactionStore struct {
	transactions indexedmap.IndexedMap
}

// Get gets a transaction
func (s *transactionStore) Get(ctx context.Context, id configapi.TransactionID) (*configapi.Transaction, error) {
	entry, err := s.transactions.Get(ctx, string(id))
	if err != nil {
		return nil, errors.FromAtomix(err)
	}
	return decodeTransaction(*entry)
}

// GetByIndex gets a transaction by index
func (s *transactionStore) GetByIndex(ctx context.Context, index configapi.Index) (*configapi.Transaction, error) {
	entry, err := s.transactions.GetIndex(ctx, indexedmap.Index(index))
	if err != nil {
		return nil, errors.FromAtomix(err)
	}
	return decodeTransaction(*entry)
}

// GetPrev gets the previous network change by index
func (s *transactionStore) GetPrev(ctx context.Context, index configapi.Index) (*configapi.Transaction, error) {
	entry, err := s.transactions.PrevEntry(ctx, indexedmap.Index(index))
	if err != nil {
		return nil, errors.FromAtomix(err)
	}
	return decodeTransaction(*entry)
}

// GetNext gets the next transaction by index
func (s *transactionStore) GetNext(ctx context.Context, index configapi.Index) (*configapi.Transaction, error) {
	entry, err := s.transactions.NextEntry(ctx, indexedmap.Index(index))
	if err != nil {
		return nil, errors.FromAtomix(err)
	}
	return decodeTransaction(*entry)
}

// Create creates a new transaction
func (s *transactionStore) Create(ctx context.Context, transaction *configapi.Transaction) error {
	if transaction.ID == "" {
		transaction.ID = newTransactionID()
	}
	if transaction.Revision != 0 {
		return errors.NewInvalid("not a new object")
	}
	transaction.Revision = 1
	transaction.Created = time.Now()
	transaction.Updated = time.Now()

	bytes, err := proto.Marshal(transaction)
	if err != nil {
		return errors.NewInvalid("transaction encoding failed: %v", err)
	}

	entry, err := s.transactions.Append(ctx, string(transaction.ID), bytes)
	if err != nil {
		return errors.FromAtomix(err)
	}

	transaction.Index = configapi.Index(entry.Index)
	transaction.Version = uint64(entry.Revision)
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

	bytes, err := proto.Marshal(transaction)
	if err != nil {
		return errors.NewInvalid("change encoding failed: %v", err)
	}

	entry, err := s.transactions.Set(ctx, indexedmap.Index(transaction.Index), string(transaction.ID), bytes, indexedmap.IfMatch(meta.NewRevision(meta.Revision(transaction.Version))))
	if err != nil {
		return errors.FromAtomix(err)
	}

	transaction.Version = uint64(entry.Revision)
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
	transaction.Revision++
	transaction.Updated = time.Now()

	bytes, err := proto.Marshal(transaction)
	if err != nil {
		return errors.NewInvalid("change encoding failed: %v", err)
	}

	entry, err := s.transactions.Set(ctx, indexedmap.Index(transaction.Index), string(transaction.ID), bytes, indexedmap.IfMatch(meta.NewRevision(meta.Revision(transaction.Version))))
	if err != nil {
		return errors.FromAtomix(err)
	}

	transaction.Version = uint64(entry.Revision)
	return nil
}

// Delete deletes a transaction
func (s *transactionStore) Delete(ctx context.Context, transaction *configapi.Transaction) error {
	if transaction.Version == 0 {
		return errors.NewInvalid("transaction must contain a version on delete")
	}

	if transaction.Deleted == nil {
		log.Debugf("Updating transaction %s", transaction.ID)
		t := time.Now()
		transaction.Deleted = &t
		bytes, err := proto.Marshal(transaction)
		if err != nil {
			return errors.NewInvalid("transaction encoding failed: %v", err)
		}
		entry, err := s.transactions.Set(ctx, indexedmap.Index(transaction.Index), string(transaction.ID), bytes, indexedmap.IfMatch(meta.NewRevision(meta.Revision(transaction.Version))))
		if err != nil {
			return errors.FromAtomix(err)
		}
		transaction.Version = uint64(entry.Revision)
	} else {
		log.Debugf("Deleting transaction %s", transaction.ID)
		_, err := s.transactions.RemoveIndex(ctx, indexedmap.Index(transaction.Index), indexedmap.IfMatch(meta.NewRevision(meta.Revision(transaction.Version))))
		if err != nil {
			log.Warnf("Failed to delete transaction %s: %s", transaction.ID, err)
			return errors.FromAtomix(err)
		}
		transaction.Version = 0
	}
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
		if transaction, err := decodeTransaction(entry); err == nil {
			transactions = append(transactions, transaction)
		}
	}
	return transactions, nil
}

// Watch watches the transaction store  for changes
func (s *transactionStore) Watch(ctx context.Context, ch chan<- configapi.TransactionEvent, opts ...WatchOption) error {
	watchOpts := make([]indexedmap.WatchOption, 0)
	for _, opt := range opts {
		watchOpts = opt.apply(watchOpts)
	}

	indexMapCh := make(chan indexedmap.Event)
	if err := s.transactions.Watch(ctx, indexMapCh, watchOpts...); err != nil {
		return errors.FromAtomix(err)
	}

	go func() {
		defer close(ch)
		for event := range indexMapCh {
			if transaction, err := decodeTransaction(event.Entry); err == nil {
				var eventType configapi.TransactionEventType
				switch event.Type {
				case indexedmap.EventReplay:
					eventType = configapi.TransactionEventType_TRANSACTION_EVENT_UNKNOWN
				case indexedmap.EventInsert:
					eventType = configapi.TransactionEventType_TRANSACTION_CREATED
				case indexedmap.EventRemove:
					eventType = configapi.TransactionEventType_TRANSACTION_DELETED
				case indexedmap.EventUpdate:
					eventType = configapi.TransactionEventType_TRANSACTION_UPDATED
				default:
					eventType = configapi.TransactionEventType_TRANSACTION_UPDATED
				}
				ch <- configapi.TransactionEvent{
					Type:        eventType,
					Transaction: *transaction,
				}
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

func decodeTransaction(entry indexedmap.Entry) (*configapi.Transaction, error) {
	transaction := &configapi.Transaction{}
	if err := proto.Unmarshal(entry.Value, transaction); err != nil {
		return nil, errors.NewInvalid("transaction decoding failed: %v", err)
	}
	transaction.ID = configapi.TransactionID(entry.Key)
	transaction.Index = configapi.Index(entry.Index)
	transaction.Version = uint64(entry.Revision)
	return transaction, nil
}

// newTransactionID creates a new transaction ID
func newTransactionID() configapi.TransactionID {
	newUUID := configapi.NewUUID()
	return configapi.TransactionID(newUUID.String())
}

var _ Store = &transactionStore{}
