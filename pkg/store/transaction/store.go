// Copyright 2020-present Open Networking Foundation.
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
// limitations under the License

package transaction

import (
	"io"
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
	io.Closer

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

	// Delete deletes a transaction
	Delete(ctx context.Context, transaction *configapi.Transaction) error

	// List lists transactions
	List(ctx context.Context) ([]*configapi.Transaction, error)

	// Watch watches the network configuration store for changes
	Watch(ctx context.Context, ch chan<- configapi.TransactionEvent, opts ...WatchOption) error
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

func (s *transactionStore) GetByIndex(ctx context.Context, index configapi.Index) (*configapi.Transaction, error) {
	entry, err := s.transactions.GetIndex(ctx, indexedmap.Index(index))
	if err != nil {
		return nil, errors.FromAtomix(err)
	}
	return decodeTransaction(*entry)
}

func (s *transactionStore) GetPrev(ctx context.Context, index configapi.Index) (*configapi.Transaction, error) {
	entry, err := s.transactions.PrevEntry(ctx, indexedmap.Index(index))
	if err != nil {
		return nil, errors.FromAtomix(err)
	}
	return decodeTransaction(*entry)
}

func (s *transactionStore) GetNext(ctx context.Context, index configapi.Index) (*configapi.Transaction, error) {
	entry, err := s.transactions.NextEntry(ctx, indexedmap.Index(index))
	if err != nil {
		return nil, errors.FromAtomix(err)
	}
	return decodeTransaction(*entry)
}

func (s *transactionStore) Create(ctx context.Context, transaction *configapi.Transaction) error {
	if transaction.ID == "" {
		transaction.ID = newTransactionID()
	}
	if transaction.Revision != 0 {
		return errors.NewInvalid("not a new object")
	}

	bytes, err := proto.Marshal(transaction)
	if err != nil {
		return errors.NewInvalid("transaction encoding failed: %v", err)
	}

	entry, err := s.transactions.Append(ctx, string(transaction.ID), bytes)
	if err != nil {
		return errors.FromAtomix(err)
	}

	transaction.Index = configapi.Index(entry.Index)
	transaction.Revision = configapi.Revision(entry.Revision)
	return nil
}

func (s *transactionStore) Update(ctx context.Context, transaction *configapi.Transaction) error {
	if transaction.Revision == 0 {
		return errors.NewInvalid("not a stored object")
	}

	bytes, err := proto.Marshal(transaction)
	if err != nil {
		return errors.NewInvalid("change encoding failed: %v", err)
	}

	entry, err := s.transactions.Set(ctx, indexedmap.Index(transaction.Index), string(transaction.ID), bytes, indexedmap.IfMatch(meta.NewRevision(meta.Revision(transaction.Revision))))
	if err != nil {
		return errors.FromAtomix(err)
	}

	transaction.Revision = configapi.Revision(entry.Revision)
	return nil
}

func (s *transactionStore) Delete(ctx context.Context, transaction *configapi.Transaction) error {
	if transaction.Revision == 0 {
		return errors.NewInvalid("not a stored object")
	}

	_, err := s.transactions.RemoveIndex(ctx, indexedmap.Index(transaction.Index), indexedmap.IfMatch(meta.NewRevision(meta.Revision(transaction.Revision))))
	if err != nil {
		return errors.FromAtomix(err)
	}

	transaction.Revision = 0
	return nil
}

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
func (s *transactionStore) Close() error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	return s.transactions.Close(ctx)
}

func decodeTransaction(entry indexedmap.Entry) (*configapi.Transaction, error) {
	transaction := &configapi.Transaction{}
	if err := proto.Unmarshal(entry.Value, transaction); err != nil {
		return nil, errors.NewInvalid("transaction decoding failed: %v", err)
	}
	transaction.ID = configapi.TransactionID(entry.Key)
	transaction.Index = configapi.Index(entry.Index)
	transaction.Revision = configapi.Revision(entry.Revision)
	return transaction, nil
}

// newTransactionID creates a new transaction ID
func newTransactionID() configapi.TransactionID {
	newUUID := configapi.NewUUID()
	return configapi.TransactionID(newUUID.String())
}

var _ Store = &transactionStore{}
