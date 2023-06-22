package transaction

import (
	"context"
	"fmt"
	configv2 "github.com/onosproject/onos-api/go/onos/config/v2"
	topoapi "github.com/onosproject/onos-api/go/onos/topo"
	"github.com/onosproject/onos-config/internal/controller/model"
	pluginregistryinternal "github.com/onosproject/onos-config/internal/pluginregistry"
	gnmiinternal "github.com/onosproject/onos-config/internal/southbound/gnmi"
	configurationstoreinternal "github.com/onosproject/onos-config/internal/store/configuration"
	topostoreinternal "github.com/onosproject/onos-config/internal/store/topo"
	transactionstoreinternal "github.com/onosproject/onos-config/internal/store/transaction"
	gnmisb "github.com/onosproject/onos-config/pkg/southbound/gnmi"
	"github.com/onosproject/onos-lib-go/pkg/errors"
	gnmi "github.com/openconfig/gnmi/proto/gnmi"
	"testing"

	"github.com/golang/mock/gomock"
	configapi "github.com/onosproject/onos-api/go/onos/config/v3"
	"github.com/onosproject/onos-lib-go/pkg/controller"
	"github.com/stretchr/testify/assert"
)

const (
	testTargetID                              = "test"
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
	Transactions  model.Transactions      `json:"transactions"`
	Configuration *model.Configuration    `json:"configuration"`
	Mastership    model.Mastership        `json:"mastership"`
	Conns         model.Conns             `json:"conns"`
	Target        *model.Target           `json:"target"`
	Event         *model.TransactionEvent `json:"event"`
}

func testReconciler(t *testing.T, testCase model.TestCase[TestState, TestContext]) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	target := configapi.Target{
		ID:      testTargetID,
		Type:    testTargetType,
		Version: testTargetVersion,
	}

	transactions := transactionstoreinternal.NewMockStore(ctrl)
	transactionStates := make(map[configapi.TransactionID]*configapi.Transaction)
	for _, transactionState := range testCase.State.Transactions {
		transactionID := configapi.TransactionID{
			Target: target,
			Index:  configapi.Index(transactionState.Index),
		}
		transaction := newTransaction(transactionID, transactionState)
		transactionStates[transactionID] = transaction
	}

	transactions.EXPECT().
		Get(gomock.Any(), gomock.Any()).
		DoAndReturn(func(ctx context.Context, transactionID configapi.TransactionID) (*configapi.Transaction, error) {
			transaction, ok := transactionStates[transactionID]
			if !ok {
				return nil, errors.NewNotFound("transaction not found")
			}
			return transaction, nil
		}).
		AnyTimes()

	configurations := configurationstoreinternal.NewMockStore(ctrl)
	configurationID := configapi.ConfigurationID{
		Target: target,
	}
	configuration := newConfiguration(configurationID, *testCase.State.Configuration, testCase.State.Mastership)
	configurations.EXPECT().
		Get(gomock.Any(), gomock.Eq(configurationID)).
		Return(configuration, nil).
		AnyTimes()

	topo := topostoreinternal.NewMockStore(ctrl)
	topo.EXPECT().Get(gomock.Any(), gomock.Eq(topoapi.ID(testTargetID))).
		DoAndReturn(func(ctx context.Context, id topoapi.ID) (*topoapi.Object, error) {
			obj := &topoapi.Object{
				ID:   testTargetID,
				Type: topoapi.Object_ENTITY,
				Obj: &topoapi.Object_Entity{
					Entity: &topoapi.Entity{
						KindID: "target",
					},
				},
			}
			configurable := topoapi.Configurable{
				Type:    string(testTargetType),
				Version: string(testTargetVersion),
				Address: "localhost:1234",
			}
			if err := obj.SetAspect(&configurable); err != nil {
				return nil, err
			}
			return obj, nil
		}).AnyTimes()

	conns := gnmiinternal.NewMockConnManager(ctrl)
	for nodeID := range testCase.State.Conns {
		topo.EXPECT().Get(gomock.Any(), gomock.Eq(gomock.Eq(topoapi.ID(nodeID)))).
			DoAndReturn(func(ctx context.Context, id topoapi.ID) (*topoapi.Object, error) {
				obj := &topoapi.Object{
					ID:   topoapi.ID(nodeID),
					Type: topoapi.Object_ENTITY,
					Obj: &topoapi.Object_Entity{
						Entity: &topoapi.Entity{
							KindID: topoapi.ONOS_CONFIG,
						},
					},
				}
				if testCase.State.Mastership.Master == nodeID {
					mastership := topoapi.MastershipState{
						Term:   uint64(testCase.State.Mastership.Term),
						NodeId: fmt.Sprintf("%s-%s", nodeID, testTargetID),
					}
					if err := obj.SetAspect(&mastership); err != nil {
						return nil, err
					}
				}
				return obj, nil
			}).AnyTimes()
	}

	plugins := pluginregistryinternal.NewMockPluginRegistry(ctrl)

	if testCase.Transitions.Event != nil {
		switch testCase.Transitions.Event.Event {
		case model.CommitEvent:
			switch testCase.Transitions.Event.Phase {
			case model.TransactionPhaseChange:
				switch testCase.Transitions.Event.Status {
				case model.TransactionInProgress:
					configurations.EXPECT().
						UpdateStatus(gomock.Any(), gomock.Any()).
						DoAndReturn(func(ctx context.Context, configuration *configapi.Configuration) error {
							testConfiguration(t, configuration, *testCase.Transitions.Configuration)
							return nil
						})

					transactionState, ok := testCase.Transitions.Transactions[testCase.Context.Index]
					if ok {
						transactions.EXPECT().
							UpdateStatus(gomock.Any(), gomock.Any()).
							DoAndReturn(func(ctx context.Context, transaction *configapi.Transaction) error {
								testTransaction(t, transaction, transactionState)
								return nil
							})
					} else {
						transactions.EXPECT().
							UpdateStatus(gomock.Any(), gomock.Any()).Return(errors.NewConflict("test conflict"))
					}
				case model.TransactionComplete:
					plugin := pluginregistryinternal.NewMockModelPlugin(ctrl)
					plugin.EXPECT().
						Validate(gomock.Any(), gomock.Any()).
						Return(nil).
						AnyTimes()
					plugins.EXPECT().
						GetPlugin(gomock.Eq(configv2.TargetType(testTargetType)), gomock.Eq(configv2.TargetVersion(testTargetVersion))).
						Return(plugin, true).
						AnyTimes()

					configurations.EXPECT().
						UpdateStatus(gomock.Any(), gomock.Any()).
						DoAndReturn(func(ctx context.Context, configuration *configapi.Configuration) error {
							testConfiguration(t, configuration, *testCase.Transitions.Configuration)
							return nil
						})

					transactionState, ok := testCase.Transitions.Transactions[testCase.Context.Index]
					if ok {
						transactions.EXPECT().
							UpdateStatus(gomock.Any(), gomock.Any()).
							DoAndReturn(func(ctx context.Context, transaction *configapi.Transaction) error {
								testTransaction(t, transaction, transactionState)
								return nil
							})
					} else {
						transactions.EXPECT().
							UpdateStatus(gomock.Any(), gomock.Any()).Return(errors.NewConflict("test conflict"))
					}
				case model.TransactionFailed:
					plugin := pluginregistryinternal.NewMockModelPlugin(ctrl)
					plugin.EXPECT().
						Validate(gomock.Any(), gomock.Any()).
						Return(errors.NewInvalid("validation error")).
						AnyTimes()
					plugins.EXPECT().
						GetPlugin(gomock.Eq(configv2.TargetType(testTargetType)), gomock.Eq(configv2.TargetVersion(testTargetVersion))).
						Return(plugin, true).
						AnyTimes()

					transactionState, ok := testCase.Transitions.Transactions[testCase.Context.Index]
					assert.True(t, ok)
					transactions.EXPECT().
						UpdateStatus(gomock.Any(), gomock.Any()).
						DoAndReturn(func(ctx context.Context, transaction *configapi.Transaction) error {
							testTransaction(t, transaction, transactionState)
							return nil
						})

					if testCase.Transitions.Configuration != nil {
						configurations.EXPECT().
							UpdateStatus(gomock.Any(), gomock.Any()).
							DoAndReturn(func(ctx context.Context, configuration *configapi.Configuration) error {
								testConfiguration(t, configuration, *testCase.Transitions.Configuration)
								return nil
							})
					} else {
						configurations.EXPECT().
							UpdateStatus(gomock.Any(), gomock.Any()).
							Return(errors.NewConflict("test conflict"))
					}
				}
			case model.TransactionPhaseRollback:
				configurations.EXPECT().
					UpdateStatus(gomock.Any(), gomock.Any()).
					DoAndReturn(func(ctx context.Context, configuration *configapi.Configuration) error {
						testConfiguration(t, configuration, *testCase.Transitions.Configuration)
						return nil
					})

				transactionState, ok := testCase.Transitions.Transactions[testCase.Context.Index]
				if ok {
					transactions.EXPECT().
						UpdateStatus(gomock.Any(), gomock.Any()).
						DoAndReturn(func(ctx context.Context, transaction *configapi.Transaction) error {
							testTransaction(t, transaction, transactionState)
							return nil
						})
				} else {
					transactions.EXPECT().
						UpdateStatus(gomock.Any(), gomock.Any()).Return(errors.NewConflict("test conflict"))
				}
			}
		case model.ApplyEvent:
			switch testCase.Transitions.Event.Status {
			case model.TransactionComplete:
				topo.EXPECT().
					Get(gomock.Any(), gomock.Eq(topoapi.ID(fmt.Sprintf("%s-%s", testCase.Context.NodeID, testTargetID)))).
					DoAndReturn(func(ctx context.Context, id topoapi.ID) (*topoapi.Object, error) {
						return &topoapi.Object{
							ID:   topoapi.ID(fmt.Sprintf("%s-%s", testCase.Context.NodeID, testTargetID)),
							Type: topoapi.Object_RELATION,
							Obj: &topoapi.Object_Relation{
								Relation: &topoapi.Relation{
									KindID:      topoapi.CONTROLS,
									SrcEntityID: topoapi.ID(testCase.Context.NodeID),
									TgtEntityID: testTargetID,
								},
							},
						}, nil
					})

				conns.EXPECT().
					Get(gomock.Any(), gomock.Eq(gnmisb.ConnID(fmt.Sprintf("%s-%s", testCase.Context.NodeID, testTargetID)))).
					DoAndReturn(func(ctx context.Context, id gnmisb.ConnID) (gnmisb.Conn, bool) {
						conn := gnmiinternal.NewMockConn(ctrl)
						conn.EXPECT().ID().Return(id).AnyTimes()
						conn.EXPECT().
							Set(gomock.Any(), gomock.Any()).
							Return(&gnmi.SetResponse{}, nil)
						return conn, true
					})

				if testCase.Transitions.Configuration != nil {
					configurations.EXPECT().
						UpdateStatus(gomock.Any(), gomock.Any()).
						DoAndReturn(func(ctx context.Context, configuration *configapi.Configuration) error {
							testConfiguration(t, configuration, *testCase.Transitions.Configuration)
							return nil
						})
				}

				transactionState, ok := testCase.Transitions.Transactions[testCase.Context.Index]
				if ok {
					transactions.EXPECT().
						UpdateStatus(gomock.Any(), gomock.Any()).
						DoAndReturn(func(ctx context.Context, transaction *configapi.Transaction) error {
							testTransaction(t, transaction, transactionState)
							return nil
						})
				} else {
					transactions.EXPECT().
						UpdateStatus(gomock.Any(), gomock.Any()).Return(errors.NewConflict("test conflict"))
				}
			case model.TransactionFailed:
				topo.EXPECT().
					Get(gomock.Any(), gomock.Eq(topoapi.ID(fmt.Sprintf("%s-%s", testCase.Context.NodeID, testTargetID)))).
					DoAndReturn(func(ctx context.Context, id topoapi.ID) (*topoapi.Object, error) {
						return &topoapi.Object{
							ID:   topoapi.ID(fmt.Sprintf("%s-%s", testCase.Context.NodeID, testTargetID)),
							Type: topoapi.Object_RELATION,
							Obj: &topoapi.Object_Relation{
								Relation: &topoapi.Relation{
									KindID:      topoapi.CONTROLS,
									SrcEntityID: topoapi.ID(testCase.Context.NodeID),
									TgtEntityID: testTargetID,
								},
							},
						}, nil
					}).AnyTimes()

				conns.EXPECT().
					Get(gomock.Any(), gomock.Eq(gnmisb.ConnID(fmt.Sprintf("%s-%s", testCase.Context.NodeID, testTargetID)))).
					DoAndReturn(func(ctx context.Context, id gnmisb.ConnID) (gnmisb.Conn, bool) {
						conn := gnmiinternal.NewMockConn(ctrl)
						conn.EXPECT().ID().Return(id).AnyTimes()
						conn.EXPECT().
							Set(gomock.Any(), gomock.Any()).
							Return(nil, errors.Status(errors.NewInternal("internal error")).Err())
						return conn, true
					}).AnyTimes()

				transactionState, ok := testCase.Transitions.Transactions[testCase.Context.Index]
				assert.True(t, ok)
				transactions.EXPECT().
					UpdateStatus(gomock.Any(), gomock.Any()).
					DoAndReturn(func(ctx context.Context, transaction *configapi.Transaction) error {
						testTransaction(t, transaction, transactionState)
						return nil
					})

				if testCase.Transitions.Configuration != nil {
					configurations.EXPECT().
						UpdateStatus(gomock.Any(), gomock.Any()).
						DoAndReturn(func(ctx context.Context, configuration *configapi.Configuration) error {
							testConfiguration(t, configuration, *testCase.Transitions.Configuration)
							return nil
						})
				} else {
					configurations.EXPECT().
						UpdateStatus(gomock.Any(), gomock.Any()).
						Return(errors.NewConflict("test conflict"))
				}
			default:
				if testCase.Transitions.Configuration != nil {
					configurations.EXPECT().
						UpdateStatus(gomock.Any(), gomock.Any()).
						DoAndReturn(func(ctx context.Context, configuration *configapi.Configuration) error {
							testConfiguration(t, configuration, *testCase.Transitions.Configuration)
							return nil
						})
				} else {
					configurations.EXPECT().
						UpdateStatus(gomock.Any(), gomock.Any()).
						Return(errors.NewConflict("test conflict"))
				}

				transactionState, ok := testCase.Transitions.Transactions[testCase.Context.Index]
				if ok {
					transactions.EXPECT().
						UpdateStatus(gomock.Any(), gomock.Any()).
						DoAndReturn(func(ctx context.Context, transaction *configapi.Transaction) error {
							testTransaction(t, transaction, transactionState)
							return nil
						})
				} else {
					transactions.EXPECT().
						UpdateStatus(gomock.Any(), gomock.Any()).Return(errors.NewConflict("test conflict"))
				}
			}
		}
	} else {
		if testCase.Transitions.Configuration != nil {
			configurations.EXPECT().
				UpdateStatus(gomock.Any(), gomock.Any()).
				DoAndReturn(func(ctx context.Context, configuration *configapi.Configuration) error {
					testConfiguration(t, configuration, *testCase.Transitions.Configuration)
					return nil
				})
		}

		for _, state := range testCase.Transitions.Transactions {
			transactionState := state
			transactions.EXPECT().
				UpdateStatus(gomock.Any(), gomock.Any()).
				DoAndReturn(func(ctx context.Context, transaction *configapi.Transaction) error {
					testTransaction(t, transaction, transactionState)
					return nil
				})
		}
	}

	reconciler := &Reconciler{
		nodeID:         configapi.NodeID(testCase.Context.NodeID),
		transactions:   transactions,
		configurations: configurations,
		conns:          conns,
		topo:           topo,
		plugins:        plugins,
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

	values := make(map[string]configapi.PathValue)
	for path, value := range state.Change.Values {
		values[path] = configapi.PathValue{
			Path: path,
			Value: configapi.TypedValue{
				Type:  configapi.ValueType_STRING,
				Bytes: []byte(value),
			},
		}
	}
	transaction.Values = values

	switch state.Phase {
	case model.TransactionPhaseChange:
		transaction.Status.Phase = configapi.TransactionStatus_CHANGE
		transaction.Status.Change = configapi.TransactionChangeStatus{
			Ordinal: configapi.Ordinal(state.Change.Ordinal),
			Commit:  newTransactionPhaseStatus(state.Change.Commit),
			Apply:   newTransactionPhaseStatus(state.Change.Apply),
		}
	case model.TransactionPhaseRollback:
		transaction.Status.Phase = configapi.TransactionStatus_ROLLBACK
		transaction.Status.Change = configapi.TransactionChangeStatus{
			Ordinal: configapi.Ordinal(state.Change.Ordinal),
			Commit:  newTransactionPhaseStatus(state.Change.Commit),
			Apply:   newTransactionPhaseStatus(state.Change.Apply),
		}
		transaction.Status.Rollback = configapi.TransactionRollbackStatus{
			Ordinal: configapi.Ordinal(state.Rollback.Ordinal),
			Commit:  newTransactionPhaseStatus(state.Rollback.Commit),
			Apply:   newTransactionPhaseStatus(state.Rollback.Apply),
			Index:   configapi.Index(state.Rollback.Index),
		}
		rollbackValues := make(map[string]configapi.PathValue)
		for path, value := range state.Rollback.Values {
			rollbackValues[path] = configapi.PathValue{
				Path: path,
				Value: configapi.TypedValue{
					Type:  configapi.ValueType_STRING,
					Bytes: []byte(value),
				},
			}
		}
		transaction.Status.Rollback.Values = rollbackValues
	}
	return transaction
}

func newTransactionPhaseStatus(state *model.TransactionState) *configapi.TransactionPhaseStatus {
	if state == nil {
		return nil
	}

	var status configapi.TransactionPhaseStatus
	switch *state {
	case model.TransactionPending:
		status.State = configapi.TransactionPhaseStatus_PENDING
	case model.TransactionInProgress:
		status.State = configapi.TransactionPhaseStatus_IN_PROGRESS
		status.Start = now()
	case model.TransactionComplete:
		status.State = configapi.TransactionPhaseStatus_COMPLETE
		status.Start = now()
		status.End = now()
	case model.TransactionAborted:
		status.State = configapi.TransactionPhaseStatus_ABORTED
		status.Start = now()
		status.End = now()
	case model.TransactionCanceled:
		status.State = configapi.TransactionPhaseStatus_CANCELED
		status.Start = now()
		status.End = now()
	case model.TransactionFailed:
		status.State = configapi.TransactionPhaseStatus_FAILED
		status.Start = now()
		status.End = now()
	}
	return &status
}

func testTransaction(t *testing.T, transaction *configapi.Transaction, state model.Transaction) {
	assert.Equal(t, state.Index, model.Index(transaction.ID.Index))

	assert.Equal(t, state.Change.Ordinal, model.Ordinal(transaction.Status.Change.Ordinal))
	testTransactionState(t, transaction.Status.Change.Commit, state.Change.Commit)
	testTransactionState(t, transaction.Status.Change.Apply, state.Change.Apply)

	assert.Equal(t, state.Rollback.Ordinal, model.Ordinal(transaction.Status.Rollback.Ordinal))
	testTransactionState(t, transaction.Status.Rollback.Commit, state.Rollback.Commit)
	testTransactionState(t, transaction.Status.Rollback.Apply, state.Rollback.Apply)
}

func testTransactionState(t *testing.T, phase *configapi.TransactionPhaseStatus, state *model.TransactionState) bool {
	if state == nil {
		return assert.Nil(t, phase)
	}
	switch *state {
	case model.TransactionPending:
		return assert.Equal(t, configapi.TransactionPhaseStatus_PENDING, phase.State)
	case model.TransactionInProgress:
		return assert.Equal(t, configapi.TransactionPhaseStatus_IN_PROGRESS, phase.State)
	case model.TransactionComplete:
		return assert.Equal(t, configapi.TransactionPhaseStatus_COMPLETE, phase.State)
	case model.TransactionAborted:
		return assert.Equal(t, configapi.TransactionPhaseStatus_ABORTED, phase.State)
	case model.TransactionCanceled:
		return assert.Equal(t, configapi.TransactionPhaseStatus_CANCELED, phase.State)
	case model.TransactionFailed:
		return assert.Equal(t, configapi.TransactionPhaseStatus_FAILED, phase.State)
	}
	return false
}

func newConfiguration(configurationID configapi.ConfigurationID, state model.Configuration, mastership model.Mastership) *configapi.Configuration {
	configuration := &configapi.Configuration{
		ObjectMeta: configapi.ObjectMeta{
			Key: configapi.NewUUID().String(),
		},
		ID: configurationID,
		Committed: configapi.CommittedConfiguration{
			Index:    configapi.Index(state.Committed.Index),
			Ordinal:  configapi.Ordinal(state.Committed.Ordinal),
			Revision: configapi.Revision(state.Committed.Revision),
			Target:   configapi.Index(state.Committed.Target),
			Change:   configapi.Index(state.Committed.Change),
			Values:   newPathValues(state.Committed.Values),
		},
		Applied: configapi.AppliedConfiguration{
			Index:    configapi.Index(state.Applied.Index),
			Ordinal:  configapi.Ordinal(state.Applied.Ordinal),
			Revision: configapi.Revision(state.Applied.Revision),
			Target:   configapi.Index(state.Applied.Target),
			Term:     configapi.MastershipTerm(state.Term),
			Values:   newPathValues(state.Applied.Values),
		},
	}

	if mastership.Master != "" {
		configuration.Status.Mastership = &configapi.MastershipStatus{
			Master: configapi.NodeID(fmt.Sprintf("%s-%s", mastership.Master, testTargetID)),
			Term:   configapi.MastershipTerm(mastership.Term),
		}
	}

	switch state.State {
	case model.ConfigurationPending:
		configuration.Status.State = configapi.ConfigurationStatus_SYNCHRONIZING
	case model.ConfigurationComplete:
		configuration.Status.State = configapi.ConfigurationStatus_SYNCHRONIZED
	}
	return configuration
}

func testConfiguration(t *testing.T, configuration *configapi.Configuration, state model.Configuration) {
	assert.Equal(t, configapi.Index(state.Committed.Index), configuration.Committed.Index)
	assert.Equal(t, configapi.Ordinal(state.Committed.Ordinal), configuration.Committed.Ordinal)
	assert.Equal(t, configapi.Revision(state.Committed.Revision), configuration.Committed.Revision)
	assert.Equal(t, configapi.Index(state.Committed.Target), configuration.Committed.Target)
	assert.Equal(t, configapi.Index(state.Committed.Change), configuration.Committed.Change)

	for key, value := range state.Committed.Values {
		assert.Equal(t, value, string(configuration.Committed.Values[key].Value.Bytes))
		assert.False(t, configuration.Committed.Values[key].Deleted)
	}

	assert.Equal(t, configapi.Index(state.Applied.Index), configuration.Applied.Index)
	assert.Equal(t, configapi.Ordinal(state.Applied.Ordinal), configuration.Applied.Ordinal)
	assert.Equal(t, configapi.Revision(state.Applied.Revision), configuration.Applied.Revision)
	assert.Equal(t, configapi.Index(state.Applied.Target), configuration.Applied.Target)

	for key, value := range state.Applied.Values {
		assert.Equal(t, value, string(configuration.Applied.Values[key].Value.Bytes))
		assert.False(t, configuration.Applied.Values[key].Deleted)
	}
}

func newPathValues(state model.Values) map[string]configapi.PathValue {
	values := make(map[string]configapi.PathValue)
	for path, value := range state {
		pathValue := configapi.PathValue{
			Path: path,
		}
		if value != "" {
			pathValue.Value = configapi.TypedValue{
				Bytes: []byte(value),
				Type:  configapi.ValueType_STRING,
			}
		} else {
			pathValue.Deleted = true
		}
		values[path] = pathValue
	}
	return values
}
