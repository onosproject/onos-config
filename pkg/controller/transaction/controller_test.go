package transaction

import (
	"context"
	"github.com/onosproject/onos-config/internal/controller/model"
	configurationstoreinternal "github.com/onosproject/onos-config/internal/store/configuration"
	proposalstoreinternal "github.com/onosproject/onos-config/internal/store/proposal"
	transactionstoreinternal "github.com/onosproject/onos-config/internal/store/transaction"
	"github.com/onosproject/onos-lib-go/pkg/errors"
	"testing"

	"github.com/golang/mock/gomock"
	configapi "github.com/onosproject/onos-api/go/onos/config/v2"
	"github.com/onosproject/onos-lib-go/pkg/controller"
	"github.com/stretchr/testify/assert"
)

const (
	testTarget                                = "test"
	testTargetType    configapi.TargetType    = "test"
	testTargetVersion configapi.TargetVersion = "1"
)

func TestReconciler(t *testing.T) {
	model.RunTests[TestState, TestContext](t, "testdata/ModelTestCases.tar.gz", testReconciler)
}

type TestContext struct {
	NodeID model.NodeID `json:"node"`
	Index  model.Index  `json:"index"`
}

type TestState struct {
	Transactions  model.Transactions   `json:"transactions"`
	Proposals     model.Proposals      `json:"proposals"`
	Configuration *model.Configuration `json:"configuration"`
}

func testReconciler(t *testing.T, testCase model.TestCase[TestState, TestContext]) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	target := configapi.Target{
		ID:      testTarget,
		Type:    testTargetType,
		Version: testTargetVersion,
	}

	transactionStore := transactionstoreinternal.NewMockStore(ctrl)
	transactions := make(map[configapi.TransactionID]*configapi.Transaction)
	proposalTransactions := make(map[model.Index]*configapi.Transaction)
	for _, transactionState := range testCase.State.Transactions {
		transactionID := configapi.TransactionID{
			Target: target,
			Index:  configapi.Index(transactionState.Index),
		}
		transaction := newTransaction(transactionID, transactionState)
		transactions[transactionID] = transaction
		switch transactionState.Type {
		case model.TransactionTypeChange:
			proposalTransactions[transactionState.Proposal] = transaction
		}
	}

	transactionStore.EXPECT().
		Get(gomock.Any(), gomock.Any()).
		DoAndReturn(func(ctx context.Context, transactionID configapi.TransactionID) (*configapi.Transaction, error) {
			transaction, ok := transactions[transactionID]
			if !ok {
				return nil, errors.NewNotFound("transaction not found")
			}
			return transaction, nil
		}).
		AnyTimes()

	for _, state := range testCase.Transitions.Transactions {
		transactionState := state
		transactionStore.EXPECT().
			UpdateStatus(gomock.Any(), gomock.Any()).
			DoAndReturn(func(ctx context.Context, transaction *configapi.Transaction) error {
				testTransaction(t, transaction, transactionState)
				return nil
			})
	}

	proposalStore := proposalstoreinternal.NewMockStore(ctrl)
	proposals := make(map[configapi.ProposalID]*configapi.Proposal)
	proposalKeys := make(map[string]*configapi.Proposal)
	for _, proposalState := range testCase.State.Proposals {
		transaction, ok := proposalTransactions[proposalState.Index]
		if !assert.True(t, ok) {
			return
		}
		proposalID := configapi.ProposalID{
			Target: target,
			Index:  configapi.Index(proposalState.Index),
		}
		proposal := newProposal(proposalID, transaction.Key, proposalState)
		proposals[proposalID] = proposal
		proposalKeys[transaction.Key] = proposal
	}

	proposalStore.EXPECT().
		Get(gomock.Any(), gomock.Any()).
		DoAndReturn(func(ctx context.Context, proposalID configapi.ProposalID) (*configapi.Proposal, error) {
			proposal, ok := proposals[proposalID]
			if !ok {
				return nil, errors.NewNotFound("proposal not found")
			}
			return proposal, nil
		}).
		AnyTimes()

	proposalStore.EXPECT().
		GetKey(gomock.Any(), gomock.Eq(target), gomock.Any()).
		DoAndReturn(func(ctx context.Context, target configapi.Target, key string) (*configapi.Proposal, error) {
			proposal, ok := proposalKeys[key]
			if !ok {
				return nil, errors.NewNotFound("proposal not found")
			}
			return proposal, nil
		}).
		AnyTimes()

	for _, state := range testCase.Transitions.Proposals {
		proposalState := state
		proposalID := configapi.ProposalID{
			Target: target,
			Index:  configapi.Index(proposalState.Index),
		}
		if _, ok := proposals[proposalID]; !ok {
			proposalStore.EXPECT().Create(gomock.Any(), gomock.Any()).
				DoAndReturn(func(ctx context.Context, proposal *configapi.Proposal) error {
					proposal.ID.Index = configapi.Index(proposalState.Index)
					testProposal(t, proposal, proposalState)
					return nil
				})
		} else {
			proposalStore.EXPECT().UpdateStatus(gomock.Any(), gomock.Any()).
				DoAndReturn(func(ctx context.Context, proposal *configapi.Proposal) error {
					testProposal(t, proposal, proposalState)
					return nil
				})
		}
	}

	configurationStore := configurationstoreinternal.NewMockStore(ctrl)
	configurationID := configapi.ConfigurationID{
		Target: target,
	}
	configuration := newConfiguration(configurationID, *testCase.State.Configuration)
	configurationStore.EXPECT().
		Get(gomock.Any(), gomock.Eq(configurationID)).
		Return(configuration, nil).
		AnyTimes()

	reconciler := &Reconciler{
		transactions:   transactionStore,
		proposals:      proposalStore,
		configurations: configurationStore,
	}

	transactionID := configapi.TransactionID{
		Target: target,
		Index:  configapi.Index(testCase.Context.Index),
	}
	_, err := reconciler.Reconcile(controller.NewID(transactionID))
	assert.NoError(t, err)
}

func newTransaction(transactionID configapi.TransactionID, state model.Transaction) *configapi.Transaction {
	transaction := &configapi.Transaction{
		ObjectMeta: configapi.ObjectMeta{
			Key: configapi.NewUUID().String(),
		},
		ID: transactionID,
	}

	switch state.Type {
	case model.TransactionTypeChange:
		values := make(map[string]*configapi.PathValue)
		for path, value := range state.Values {
			values[path] = &configapi.PathValue{
				Path: path,
				Value: configapi.TypedValue{
					Type:  configapi.ValueType_STRING,
					Bytes: []byte(value),
				},
			}
		}
		transaction.Details = &configapi.Transaction_Change{
			Change: &configapi.TransactionChange{
				Values: values,
			},
		}
	case model.TransactionTypeRollback:
		transaction.Details = &configapi.Transaction_Rollback{
			Rollback: &configapi.TransactionRollback{
				Index: configapi.Index(state.Proposal),
			},
		}
	}

	transaction.Status.Initialize = newTransactionPhaseStatus(state.Init)
	transaction.Status.Commit = newTransactionPhaseStatus(state.Commit)
	transaction.Status.Apply = newTransactionPhaseStatus(state.Apply)
	return transaction
}

func newTransactionPhaseStatus(state model.TransactionStatus) configapi.TransactionPhaseStatus {
	var status configapi.TransactionPhaseStatus
	switch state {
	case model.TransactionStatusPending:
		status.State = configapi.TransactionPhaseStatus_PENDING
	case model.TransactionStatusInProgress:
		status.State = configapi.TransactionPhaseStatus_IN_PROGRESS
		status.Start = getCurrentTimestamp()
	case model.TransactionStatusComplete:
		status.State = configapi.TransactionPhaseStatus_COMPLETE
		status.Start = getCurrentTimestamp()
		status.End = getCurrentTimestamp()
	case model.TransactionStatusAborted:
		status.State = configapi.TransactionPhaseStatus_ABORTED
		status.Start = getCurrentTimestamp()
		status.End = getCurrentTimestamp()
	case model.TransactionStatusFailed:
		status.State = configapi.TransactionPhaseStatus_FAILED
		status.Start = getCurrentTimestamp()
		status.End = getCurrentTimestamp()
	}
	return status
}

func testTransaction(t *testing.T, transaction *configapi.Transaction, state model.Transaction) {
	assert.Equal(t, state.Index, model.Index(transaction.ID.Index))
	testTransactionPhase(t, transaction.Status.Initialize, state.Init)
	testTransactionPhase(t, transaction.Status.Commit, state.Commit)
	testTransactionPhase(t, transaction.Status.Apply, state.Apply)
}

func testTransactionPhase(t *testing.T, phase configapi.TransactionPhaseStatus, status model.TransactionStatus) {
	switch status {
	case model.TransactionStatusPending:
		assert.Equal(t, configapi.TransactionPhaseStatus_PENDING, phase.State)
	case model.TransactionStatusInProgress:
		assert.Equal(t, configapi.TransactionPhaseStatus_IN_PROGRESS, phase.State)
	case model.TransactionStatusComplete:
		assert.Equal(t, configapi.TransactionPhaseStatus_COMPLETE, phase.State)
	case model.TransactionStatusAborted:
		assert.Equal(t, configapi.TransactionPhaseStatus_ABORTED, phase.State)
	case model.TransactionStatusFailed:
		assert.Equal(t, configapi.TransactionPhaseStatus_FAILED, phase.State)
	}
}

func newConfiguration(configurationID configapi.ConfigurationID, state model.Configuration) *configapi.Configuration {
	configuration := &configapi.Configuration{
		ObjectMeta: configapi.ObjectMeta{
			Key: configapi.NewUUID().String(),
		},
		ID:     configurationID,
		Index:  configapi.Index(state.Committed.Revision),
		Values: newPathValues(state.Committed.Values),
	}

	configuration.Status.Committed.Index = configapi.Index(state.Committed.Index)
	configuration.Status.Applied.Index = configapi.Index(state.Applied.Index)
	configuration.Status.Applied.Values = newPathValues(state.Applied.Values)
	return configuration
}

func newPathValues(state model.ValueStates) map[string]*configapi.PathValue {
	values := make(map[string]*configapi.PathValue)
	for path, value := range state {
		pathValue := &configapi.PathValue{
			Path:  path,
			Index: configapi.Index(value.Index),
		}
		if value.Value != "" {
			pathValue.Value = configapi.TypedValue{
				Bytes: []byte(value.Value),
				Type:  configapi.ValueType_STRING,
			}
		} else {
			pathValue.Deleted = true
		}
		values[path] = pathValue
	}
	return values
}

func newProposal(proposalID configapi.ProposalID, key string, state model.Proposal) *configapi.Proposal {
	proposal := &configapi.Proposal{
		ObjectMeta: configapi.ObjectMeta{
			Key: key,
		},
		ID:     proposalID,
		Values: newPathValues(state.Change.Values),
	}

	proposal.Status.Change = configapi.ProposalChangeStatus{
		Commit: newProposalPhase(state.Change.Commit),
		Apply:  newProposalPhase(state.Change.Apply),
	}

	proposal.Status.Rollback = configapi.ProposalRollbackStatus{
		Index:  configapi.Index(state.Rollback.Revision),
		Values: newPathValues(state.Rollback.Values),
		Commit: newProposalPhase(state.Rollback.Commit),
		Apply:  newProposalPhase(state.Rollback.Apply),
	}
	return proposal
}

func newProposalPhase(state model.ProposalStatus) configapi.ProposalPhaseStatus {
	var status configapi.ProposalPhaseStatus
	switch state {
	case model.ProposalStatusPending:
		status.State = configapi.ProposalPhaseStatus_PENDING
	case model.ProposalStatusInProgress:
		status.State = configapi.ProposalPhaseStatus_IN_PROGRESS
		status.Start = getCurrentTimestamp()
	case model.ProposalStatusComplete:
		status.State = configapi.ProposalPhaseStatus_COMPLETE
		status.Start = getCurrentTimestamp()
		status.End = getCurrentTimestamp()
	case model.ProposalStatusAborted:
		status.State = configapi.ProposalPhaseStatus_ABORTED
		status.Start = getCurrentTimestamp()
		status.End = getCurrentTimestamp()
	case model.ProposalStatusFailed:
		status.State = configapi.ProposalPhaseStatus_FAILED
		status.Start = getCurrentTimestamp()
		status.End = getCurrentTimestamp()
	}
	return status
}

func testProposal(t *testing.T, proposal *configapi.Proposal, state model.Proposal) {
	assert.Equal(t, state.Index, model.Index(proposal.ID.Index))
	testProposalChange(t, proposal.Status.Change, state.Change)
	testProposalRollback(t, proposal.Status.Rollback, state.Rollback)
}

func testProposalChange(t *testing.T, change configapi.ProposalChangeStatus, state model.Change) {
	testProposalPhase(t, change.Commit, state.Commit)
	testProposalPhase(t, change.Apply, state.Apply)
}

func testProposalRollback(t *testing.T, rollback configapi.ProposalRollbackStatus, state model.Rollback) {
	assert.Equal(t, configapi.Index(state.Revision), rollback.Index)
	assert.Equal(t, len(state.Values), len(rollback.Values))
	for key, value := range state.Values {
		if assert.NotNil(t, rollback.Values[key]) {
			assert.Equal(t, configapi.Index(value.Index), rollback.Values[key].Index)
			if value.Value == "" {
				assert.True(t, rollback.Values[key].Deleted)
			} else {
				assert.False(t, rollback.Values[key].Deleted)
				assert.Equal(t, string(value.Value), string(rollback.Values[key].Value.Bytes))
			}
			assert.Equal(t, configapi.Index(value.Index), rollback.Values[key].Index)
		}
	}
	testProposalPhase(t, rollback.Commit, state.Commit)
	testProposalPhase(t, rollback.Apply, state.Apply)
}

func testProposalPhase(t *testing.T, status configapi.ProposalPhaseStatus, state model.ProposalStatus) {
	t.Helper()
	switch state {
	case model.ProposalStatusPending:
		assert.Equal(t, configapi.ProposalPhaseStatus_PENDING, status.State)
	case model.ProposalStatusInProgress:
		assert.Equal(t, configapi.ProposalPhaseStatus_IN_PROGRESS, status.State)
	case model.ProposalStatusComplete:
		assert.Equal(t, configapi.ProposalPhaseStatus_COMPLETE, status.State)
	case model.ProposalStatusAborted:
		assert.Equal(t, configapi.ProposalPhaseStatus_ABORTED, status.State)
	case model.ProposalStatusFailed:
		assert.Equal(t, configapi.ProposalPhaseStatus_FAILED, status.State)
	}
}
