// SPDX-FileCopyrightText: 2020-present Open Networking Foundation <info@opennetworking.org>
//
// SPDX-License-Identifier: Apache-2.0

package transaction

import (
	"context"
	"github.com/atomix/go-sdk/pkg/test"
	"testing"
	"time"

	configapi "github.com/onosproject/onos-api/go/onos/config/v3"

	"github.com/stretchr/testify/assert"
)

func TestTransactionStore(t *testing.T) {
	cluster := test.NewClient()
	defer cluster.Close()

	store1, err := NewAtomixStore(cluster)
	assert.NoError(t, err)

	store2, err := NewAtomixStore(cluster)
	assert.NoError(t, err)

	eventCh := make(chan configapi.TransactionEvent)
	err = store2.Watch(context.Background(), eventCh)
	assert.NoError(t, err)

	target := configapi.Target{
		ID:      "target-1",
		Type:    "foo",
		Version: "1",
	}

	transaction1 := &configapi.Transaction{
		ObjectMeta: configapi.ObjectMeta{
			Key: "transaction-1",
		},
		ID: configapi.TransactionID{
			Target: target,
		},
		Values: map[string]configapi.PathValue{
			"foo": {
				Value: configapi.TypedValue{
					Bytes: []byte("Hello world!"),
					Type:  configapi.ValueType_STRING,
				},
			},
			"bar": {
				Value: configapi.TypedValue{
					Bytes: []byte("Hello world again!"),
					Type:  configapi.ValueType_STRING,
				},
			},
		},
	}

	transaction2 := &configapi.Transaction{
		ObjectMeta: configapi.ObjectMeta{
			Key: "transaction-2",
		},
		ID: configapi.TransactionID{
			Target: target,
		},
		Values: map[string]configapi.PathValue{
			"foo": {
				Deleted: true,
			},
		},
	}

	// Create a new transaction
	err = store1.Create(context.TODO(), transaction1)
	assert.NoError(t, err)
	assert.Equal(t, "transaction-1", transaction1.Key)
	assert.Equal(t, configapi.Index(1), transaction1.ID.Index)
	assert.NotEqual(t, configapi.Revision(0), transaction1.Revision)

	// Get the transaction
	transaction1, err = store2.GetKey(context.TODO(), target, "transaction-1")
	assert.NoError(t, err)
	assert.NotNil(t, transaction1)
	assert.Equal(t, "transaction-1", transaction1.Key)
	assert.Equal(t, configapi.Index(1), transaction1.ID.Index)
	assert.NotEqual(t, configapi.Revision(0), transaction1.Revision)

	transactionEvent := nextTransaction(t, eventCh)
	assert.Equal(t, "transaction-1", transactionEvent.Key)

	// Create another transaction
	err = store2.Create(context.TODO(), transaction2)
	assert.NoError(t, err)
	assert.Equal(t, "transaction-2", transaction2.Key)
	assert.Equal(t, configapi.Index(2), transaction2.ID.Index)
	assert.NotEqual(t, configapi.Revision(0), transaction2.Revision)

	transactionEvent = nextTransaction(t, eventCh)
	assert.Equal(t, "transaction-2", transactionEvent.Key)

	// Watch events for a specific transaction
	transactionCh := make(chan configapi.TransactionEvent)
	err = store1.Watch(context.TODO(), transactionCh, WithTransactionID(transaction2.ID))
	assert.NoError(t, err)

	// Update one of the transactions
	revision := transaction2.Revision
	err = store1.Update(context.TODO(), transaction2)
	assert.NoError(t, err)
	assert.NotEqual(t, revision, transaction2.Revision)

	transactionEvent = nextTransaction(t, eventCh)
	assert.Equal(t, "transaction-2", transactionEvent.Key)

	event := nextEvent(t, transactionCh)
	assert.Equal(t, transaction2.ID, event.Transaction.ID)

	// Read and then update the transaction
	transaction2, err = store2.GetKey(context.TODO(), target, "transaction-2")
	assert.NoError(t, err)
	assert.NotNil(t, transaction2)

	transaction2.Status.Change.Ordinal = 1
	revision = transaction2.Revision
	err = store1.Update(context.TODO(), transaction2)
	assert.NoError(t, err)
	assert.NotEqual(t, revision, transaction2.Revision)

	transactionEvent = nextTransaction(t, eventCh)
	assert.Equal(t, "transaction-2", transactionEvent.Key)

	event = nextEvent(t, transactionCh)
	assert.Equal(t, transaction2.ID, event.Transaction.ID)

	// Verify that concurrent updates fail
	transaction11, err := store1.GetKey(context.TODO(), target, "transaction-1")
	assert.NoError(t, err)
	transaction12, err := store2.GetKey(context.TODO(), target, "transaction-1")
	assert.NoError(t, err)

	transaction11.Status.Change.Ordinal = 2
	err = store1.Update(context.TODO(), transaction11)
	assert.NoError(t, err)

	transaction12.Status.Change.Ordinal = 3
	err = store2.Update(context.TODO(), transaction12)
	assert.Error(t, err)

	transactionEvent = nextTransaction(t, eventCh)
	assert.Equal(t, "transaction-1", transactionEvent.Key)

	// List the transactions
	transactions, err := store1.List(context.TODO())
	assert.NoError(t, err)
	assert.Equal(t, 2, len(transactions))

	transaction := &configapi.Transaction{
		ObjectMeta: configapi.ObjectMeta{
			Key: "transaction-3",
		},
		ID: configapi.TransactionID{
			Target: target,
		},
		Values: map[string]configapi.PathValue{
			"foo": {
				Value: configapi.TypedValue{
					Bytes: []byte("Hello world!"),
					Type:  configapi.ValueType_STRING,
				},
			},
		},
	}

	err = store1.Create(context.TODO(), transaction)
	assert.NoError(t, err)

	transaction = &configapi.Transaction{
		ObjectMeta: configapi.ObjectMeta{
			Key: "transaction-4",
		},
		ID: configapi.TransactionID{
			Target: target,
		},
		Values: map[string]configapi.PathValue{
			"bar": {
				Value: configapi.TypedValue{
					Bytes: []byte("Hello world!"),
					Type:  configapi.ValueType_STRING,
				},
			},
		},
	}

	err = store1.Create(context.TODO(), transaction)
	assert.NoError(t, err)

	eventCh = make(chan configapi.TransactionEvent)
	err = store1.Watch(context.TODO(), eventCh, WithReplay())
	assert.NoError(t, err)

	transaction = nextTransaction(t, eventCh)
	assert.Equal(t, configapi.Index(1), transaction.ID.Index)
	transaction = nextTransaction(t, eventCh)
	assert.Equal(t, configapi.Index(2), transaction.ID.Index)
	transaction = nextTransaction(t, eventCh)
	assert.Equal(t, configapi.Index(3), transaction.ID.Index)
	transaction = nextTransaction(t, eventCh)
	assert.Equal(t, configapi.Index(4), transaction.ID.Index)
}

func nextEvent(t *testing.T, ch chan configapi.TransactionEvent) *configapi.TransactionEvent {
	select {
	case e := <-ch:
		return &e
	case <-time.After(5 * time.Second):
		t.FailNow()
	}
	return nil
}

func nextTransaction(t *testing.T, ch chan configapi.TransactionEvent) *configapi.Transaction {
	return &nextEvent(t, ch).Transaction
}
