// Copyright 2019-present Open Networking Foundation.
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
	"testing"
	"time"

	"github.com/onosproject/onos-lib-go/pkg/errors"

	configapi "github.com/onosproject/onos-api/go/onos/config/v2"

	"github.com/atomix/atomix-go-client/pkg/atomix/test"
	"github.com/atomix/atomix-go-client/pkg/atomix/test/rsm"
	"github.com/stretchr/testify/assert"
)

func TestTransactionStore(t *testing.T) {
	test := test.NewTest(
		rsm.NewProtocol(),
		test.WithReplicas(1),
		test.WithPartitions(1),
	)
	assert.NoError(t, test.Start())
	defer test.Stop()

	client1, err := test.NewClient("node-1")
	assert.NoError(t, err)

	client2, err := test.NewClient("node-2")
	assert.NoError(t, err)

	store1, err := NewAtomixStore(client1)
	assert.NoError(t, err)

	store2, err := NewAtomixStore(client2)
	assert.NoError(t, err)

	target1 := configapi.TargetID("target-1")
	target2 := configapi.TargetID("target-2")

	ch := make(chan configapi.TransactionEvent)
	err = store2.Watch(context.Background(), ch)
	assert.NoError(t, err)

	transaction1 := &configapi.Transaction{
		ID: "transaction-1",
		Changes: []*configapi.Change{
			{

				TargetID:      target1,
				TargetVersion: "1.0.0",
				Values: []*configapi.ChangeValue{
					{
						Path: "foo",
						Value: &configapi.TypedValue{
							Bytes: []byte("Hello world!"),
							Type:  configapi.ValueType_STRING,
						},
					},
					{
						Path: "bar",
						Value: &configapi.TypedValue{
							Bytes: []byte("Hello world again!"),
							Type:  configapi.ValueType_STRING,
						},
					},
				},
			},
			{
				TargetID: target2,
				Values: []*configapi.ChangeValue{
					{
						Path: "baz",
						Value: &configapi.TypedValue{
							Bytes: []byte("Goodbye world!"),
							Type:  configapi.ValueType_STRING,
						},
					},
				},
			},
		},
	}

	transaction2 := &configapi.Transaction{
		ID: "transaction-2",
		Changes: []*configapi.Change{
			{
				TargetID: target1,
				Values: []*configapi.ChangeValue{
					{
						Path:    "foo",
						Removed: true,
					},
				},
			},
		},
	}

	// Create a new transaction
	err = store1.Create(context.TODO(), transaction1)
	assert.NoError(t, err)
	assert.Equal(t, configapi.TransactionID("transaction-1"), transaction1.ID)
	assert.Equal(t, configapi.Index(1), transaction1.Index)
	assert.NotEqual(t, configapi.Revision(0), transaction1.Revision)

	// Get the transaction
	transaction1, err = store2.Get(context.TODO(), "transaction-1")
	assert.NoError(t, err)
	assert.NotNil(t, transaction1)
	assert.Equal(t, configapi.TransactionID("transaction-1"), transaction1.ID)
	assert.Equal(t, configapi.Index(1), transaction1.Index)
	assert.NotEqual(t, configapi.Revision(0), transaction1.Revision)

	// Create another transaction
	err = store2.Create(context.TODO(), transaction2)
	assert.NoError(t, err)
	assert.Equal(t, configapi.TransactionID("transaction-2"), transaction2.ID)
	assert.Equal(t, configapi.Index(2), transaction2.Index)
	assert.NotEqual(t, configapi.Revision(0), transaction2.Revision)

	// Verify events were received for the transactions
	transactionEvent := nextEvent(t, ch)
	assert.Equal(t, configapi.TransactionID("transaction-1"), transactionEvent.ID)
	transactionEvent = nextEvent(t, ch)
	assert.Equal(t, configapi.TransactionID("transaction-2"), transactionEvent.ID)

	// Watch events for a specific transaction
	transactionCh := make(chan configapi.TransactionEvent)
	err = store1.Watch(context.TODO(), transactionCh, WithTransactionID(transaction2.ID))
	assert.NoError(t, err)

	// Update one of the transactions
	revision := transaction2.Revision
	err = store1.Update(context.TODO(), transaction2)
	assert.NoError(t, err)
	assert.NotEqual(t, revision, transaction2.Revision)

	event := <-transactionCh
	assert.Equal(t, transaction2.ID, event.Transaction.ID)
	assert.Equal(t, transaction2.Revision, event.Transaction.Revision)

	// Get previous and next transactions
	transaction1, err = store2.Get(context.TODO(), "transaction-1")
	assert.NoError(t, err)
	assert.NotNil(t, transaction1)
	transaction2, err = store2.GetNext(context.TODO(), transaction1.Index)
	assert.NoError(t, err)
	assert.NotNil(t, transaction2)
	assert.Equal(t, configapi.Index(2), transaction2.Index)

	transaction2, err = store2.Get(context.TODO(), "transaction-2")
	assert.NoError(t, err)
	assert.NotNil(t, transaction2)
	transaction1, err = store2.GetPrev(context.TODO(), transaction2.Index)
	assert.NoError(t, err)
	assert.NotNil(t, transaction1)
	assert.Equal(t, configapi.Index(1), transaction1.Index)

	// Read and then update the transaction
	transaction2, err = store2.Get(context.TODO(), "transaction-2")
	assert.NoError(t, err)
	assert.NotNil(t, transaction2)
	transaction2.Status.State = configapi.TransactionState_TRANSACTION_COMPLETE
	revision = transaction2.Revision
	err = store1.Update(context.TODO(), transaction2)
	assert.NoError(t, err)
	assert.NotEqual(t, revision, transaction2.Revision)

	event = <-transactionCh
	assert.Equal(t, transaction2.ID, event.Transaction.ID)
	assert.Equal(t, transaction2.Revision, event.Transaction.Revision)

	// Verify that concurrent updates fail
	transaction11, err := store1.Get(context.TODO(), "transaction-1")
	assert.NoError(t, err)
	transaction12, err := store2.Get(context.TODO(), "transaction-1")
	assert.NoError(t, err)

	transaction11.Status.State = configapi.TransactionState_TRANSACTION_COMPLETE
	err = store1.Update(context.TODO(), transaction11)
	assert.NoError(t, err)

	transaction12.Status.State = configapi.TransactionState_TRANSACTION_FAILED
	err = store2.Update(context.TODO(), transaction12)
	assert.Error(t, err)

	// Verify events were received again
	transactionEvent = nextEvent(t, ch)
	assert.Equal(t, configapi.TransactionID("transaction-2"), transactionEvent.ID)
	transactionEvent = nextEvent(t, ch)
	assert.Equal(t, configapi.TransactionID("transaction-2"), transactionEvent.ID)
	transactionEvent = nextEvent(t, ch)
	assert.Equal(t, configapi.TransactionID("transaction-1"), transactionEvent.ID)

	// List the transactions
	transactions, err := store1.List(context.TODO())
	assert.NoError(t, err)
	assert.Equal(t, 2, len(transactions))

	// Delete a transaction
	err = store1.Delete(context.TODO(), transaction2)
	assert.NoError(t, err)
	transaction, err := store2.Get(context.TODO(), "transaction-2")
	assert.Error(t, err)
	assert.True(t, errors.IsNotFound(err))
	assert.Nil(t, transaction)

	event = <-transactionCh
	assert.Equal(t, transaction2.ID, event.Transaction.ID)
	assert.Equal(t, configapi.TransactionEventType_TRANSACTION_DELETED, event.Type)

	transaction = &configapi.Transaction{
		ID: "transaction-3",
		Changes: []*configapi.Change{
			{
				TargetID: target1,
				Values: []*configapi.ChangeValue{
					{
						Path: "foo",
						Value: &configapi.TypedValue{
							Bytes: []byte("Hello world!"),
							Type:  configapi.ValueType_STRING,
						},
					},
				},
			},
		},
	}

	err = store1.Create(context.TODO(), transaction)
	assert.NoError(t, err)

	transaction = &configapi.Transaction{
		ID: "transaction-4",
		Changes: []*configapi.Change{
			{
				TargetID: target2,
				Values: []*configapi.ChangeValue{
					{
						Path: "bar",
						Value: &configapi.TypedValue{
							Bytes: []byte("Hello world!"),
							Type:  configapi.ValueType_STRING,
						},
					},
				},
			},
		},
	}

	err = store1.Create(context.TODO(), transaction)
	assert.NoError(t, err)

	ch = make(chan configapi.TransactionEvent)
	err = store1.Watch(context.TODO(), ch, WithReplay())
	assert.NoError(t, err)

	transaction = nextEvent(t, ch)
	assert.Equal(t, configapi.Index(1), transaction.Index)
	transaction = nextEvent(t, ch)
	assert.Equal(t, configapi.Index(3), transaction.Index)
	transaction = nextEvent(t, ch)
	assert.Equal(t, configapi.Index(4), transaction.Index)
}

func nextEvent(t *testing.T, ch chan configapi.TransactionEvent) *configapi.Transaction {
	select {
	case c := <-ch:
		return &c.Transaction
	case <-time.After(5 * time.Second):
		t.FailNow()
	}
	return nil
}
