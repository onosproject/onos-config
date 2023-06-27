// SPDX-FileCopyrightText: 2020-present Open Networking Foundation <info@opennetworking.org>
//
// SPDX-License-Identifier: Apache-2.0

package transaction

import (
	"context"
	"fmt"
	configv2 "github.com/onosproject/onos-api/go/onos/config/v2"
	topoapi "github.com/onosproject/onos-api/go/onos/topo"
	"github.com/onosproject/onos-config/internal/controller/model"
	pluginmock "github.com/onosproject/onos-config/internal/pluginregistry"
	gnmimock "github.com/onosproject/onos-config/internal/southbound/gnmi"
	topomock "github.com/onosproject/onos-config/internal/store/topo"
	configurationmock "github.com/onosproject/onos-config/internal/store/v3/configuration"
	transactionmock "github.com/onosproject/onos-config/internal/store/v3/transaction"
	gnmisb "github.com/onosproject/onos-config/pkg/southbound/gnmi"
	"github.com/onosproject/onos-lib-go/pkg/errors"
	"github.com/openconfig/gnmi/proto/gnmi"
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

var target = configapi.Target{
	ID:      testTargetID,
	Type:    testTargetType,
	Version: testTargetVersion,
}

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

	runner := &testRunner{
		TestCase: testCase,
	}
	runner.run(t)
}

type testRunner struct {
	model.TestCase[TestState, TestContext]
	ctrl           *gomock.Controller
	transactions   *transactionmock.MockStore
	configurations *configurationmock.MockStore
	conns          *gnmimock.MockConnManager
	topo           *topomock.MockStore
	plugins        *pluginmock.MockPluginRegistry
}

func (r *testRunner) run(t *testing.T) {
	reconciler := r.setup(t)
	defer r.teardown()

	transactionID := configapi.TransactionID{
		Target: target,
		Index:  configapi.Index(r.Context.Index),
	}

	_, err := reconciler.Reconcile(controller.NewID(transactionID))
	assert.NoError(t, err)
}

func (r *testRunner) setup(t *testing.T) *Reconciler {
	r.ctrl = gomock.NewController(t)
	r.setupState()
	r.mockTransitions(t)
	return &Reconciler{
		nodeID:         configapi.NodeID(r.Context.NodeID),
		transactions:   r.transactions,
		configurations: r.configurations,
		conns:          r.conns,
		topo:           r.topo,
		plugins:        r.plugins,
	}
}

func (r *testRunner) setupState() {
	r.transactions = transactionmock.NewMockStore(r.ctrl)
	transactionStates := make(map[configapi.TransactionID]*configapi.Transaction)
	for _, transactionState := range r.State.Transactions {
		transactionID := configapi.TransactionID{
			Target: target,
			Index:  configapi.Index(transactionState.Index),
		}
		transaction := newTransaction(transactionID, transactionState)
		transactionStates[transactionID] = transaction
	}

	r.transactions.EXPECT().
		Get(gomock.Any(), gomock.Any()).
		DoAndReturn(func(ctx context.Context, transactionID configapi.TransactionID) (*configapi.Transaction, error) {
			transaction, ok := transactionStates[transactionID]
			if !ok {
				return nil, errors.NewNotFound("transaction not found")
			}
			return transaction, nil
		}).
		AnyTimes()

	r.configurations = configurationmock.NewMockStore(r.ctrl)
	configurationID := configapi.ConfigurationID{
		Target: target,
	}
	configuration := newConfiguration(configurationID, *r.State.Configuration, r.State.Mastership)
	r.configurations.EXPECT().
		Get(gomock.Any(), gomock.Eq(configurationID)).
		Return(configuration, nil).
		AnyTimes()

	r.topo = topomock.NewMockStore(r.ctrl)
	r.topo.EXPECT().Get(gomock.Any(), gomock.Eq(topoapi.ID(testTargetID))).
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

	r.conns = gnmimock.NewMockConnManager(r.ctrl)
	for nodeID := range r.State.Conns {
		r.topo.EXPECT().Get(gomock.Any(), gomock.Eq(gomock.Eq(topoapi.ID(nodeID)))).
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
				if r.State.Mastership.Master == nodeID {
					mastership := topoapi.MastershipState{
						Term:   uint64(r.State.Mastership.Term),
						NodeId: fmt.Sprintf("%s-%s", nodeID, testTargetID),
					}
					if err := obj.SetAspect(&mastership); err != nil {
						return nil, err
					}
				}
				return obj, nil
			}).AnyTimes()
	}

	r.plugins = pluginmock.NewMockPluginRegistry(r.ctrl)
}

func (r *testRunner) mockTransitions(t *testing.T) {
	if r.Transitions.Event != nil {
		switch r.Transitions.Event.Event {
		case model.CommitEvent:
			switch r.Transitions.Event.Phase {
			case model.TransactionPhaseChange:
				switch r.Transitions.Event.Status {
				case model.TransactionInProgress:
					r.configurations.EXPECT().
						UpdateStatus(gomock.Any(), gomock.Any()).
						DoAndReturn(func(ctx context.Context, configuration *configapi.Configuration) error {
							testConfiguration(t, configuration, *r.Transitions.Configuration)
							return nil
						})

					transactionState, ok := r.Transitions.Transactions[r.Context.Index]
					if ok {
						r.transactions.EXPECT().
							UpdateStatus(gomock.Any(), gomock.Any()).
							DoAndReturn(func(ctx context.Context, transaction *configapi.Transaction) error {
								testTransaction(t, transaction, transactionState)
								return nil
							})
					} else {
						r.transactions.EXPECT().
							UpdateStatus(gomock.Any(), gomock.Any()).Return(errors.NewConflict("test conflict"))
					}
				case model.TransactionComplete:
					plugin := pluginmock.NewMockModelPlugin(r.ctrl)
					plugin.EXPECT().
						Validate(gomock.Any(), gomock.Any()).
						Return(nil).
						AnyTimes()
					r.plugins.EXPECT().
						GetPlugin(gomock.Eq(configv2.TargetType(testTargetType)), gomock.Eq(configv2.TargetVersion(testTargetVersion))).
						Return(plugin, true).
						AnyTimes()

					r.configurations.EXPECT().
						UpdateStatus(gomock.Any(), gomock.Any()).
						DoAndReturn(func(ctx context.Context, configuration *configapi.Configuration) error {
							testConfiguration(t, configuration, *r.Transitions.Configuration)
							return nil
						})

					transactionState, ok := r.Transitions.Transactions[r.Context.Index]
					if ok {
						r.transactions.EXPECT().
							UpdateStatus(gomock.Any(), gomock.Any()).
							DoAndReturn(func(ctx context.Context, transaction *configapi.Transaction) error {
								testTransaction(t, transaction, transactionState)
								return nil
							})
					} else {
						r.transactions.EXPECT().
							UpdateStatus(gomock.Any(), gomock.Any()).Return(errors.NewConflict("test conflict"))
					}
				case model.TransactionFailed:
					plugin := pluginmock.NewMockModelPlugin(r.ctrl)
					plugin.EXPECT().
						Validate(gomock.Any(), gomock.Any()).
						Return(errors.NewInvalid("validation error")).
						AnyTimes()
					r.plugins.EXPECT().
						GetPlugin(gomock.Eq(configv2.TargetType(testTargetType)), gomock.Eq(configv2.TargetVersion(testTargetVersion))).
						Return(plugin, true).
						AnyTimes()

					transactionState, ok := r.Transitions.Transactions[r.Context.Index]
					assert.True(t, ok)
					r.transactions.EXPECT().
						UpdateStatus(gomock.Any(), gomock.Any()).
						DoAndReturn(func(ctx context.Context, transaction *configapi.Transaction) error {
							testTransaction(t, transaction, transactionState)
							return nil
						})

					if r.Transitions.Configuration != nil {
						r.configurations.EXPECT().
							UpdateStatus(gomock.Any(), gomock.Any()).
							DoAndReturn(func(ctx context.Context, configuration *configapi.Configuration) error {
								testConfiguration(t, configuration, *r.Transitions.Configuration)
								return nil
							})
					} else {
						r.configurations.EXPECT().
							UpdateStatus(gomock.Any(), gomock.Any()).
							Return(errors.NewConflict("test conflict"))
					}
				}
			case model.TransactionPhaseRollback:
				r.configurations.EXPECT().
					UpdateStatus(gomock.Any(), gomock.Any()).
					DoAndReturn(func(ctx context.Context, configuration *configapi.Configuration) error {
						testConfiguration(t, configuration, *r.Transitions.Configuration)
						return nil
					})

				transactionState, ok := r.Transitions.Transactions[r.Context.Index]
				if ok {
					r.transactions.EXPECT().
						UpdateStatus(gomock.Any(), gomock.Any()).
						DoAndReturn(func(ctx context.Context, transaction *configapi.Transaction) error {
							testTransaction(t, transaction, transactionState)
							return nil
						})
				} else {
					r.transactions.EXPECT().
						UpdateStatus(gomock.Any(), gomock.Any()).Return(errors.NewConflict("test conflict"))
				}
			}
		case model.ApplyEvent:
			switch r.Transitions.Event.Status {
			case model.TransactionComplete:
				r.topo.EXPECT().
					Get(gomock.Any(), gomock.Eq(topoapi.ID(fmt.Sprintf("%s-%s", r.Context.NodeID, testTargetID)))).
					DoAndReturn(func(ctx context.Context, id topoapi.ID) (*topoapi.Object, error) {
						return &topoapi.Object{
							ID:   topoapi.ID(fmt.Sprintf("%s-%s", r.Context.NodeID, testTargetID)),
							Type: topoapi.Object_RELATION,
							Obj: &topoapi.Object_Relation{
								Relation: &topoapi.Relation{
									KindID:      topoapi.CONTROLS,
									SrcEntityID: topoapi.ID(r.Context.NodeID),
									TgtEntityID: testTargetID,
								},
							},
						}, nil
					})

				r.conns.EXPECT().
					Get(gomock.Any(), gomock.Eq(gnmisb.ConnID(fmt.Sprintf("%s-%s", r.Context.NodeID, testTargetID)))).
					DoAndReturn(func(ctx context.Context, id gnmisb.ConnID) (gnmisb.Conn, bool) {
						conn := gnmimock.NewMockConn(r.ctrl)
						conn.EXPECT().ID().Return(id).AnyTimes()
						conn.EXPECT().
							Set(gomock.Any(), gomock.Any()).
							Return(&gnmi.SetResponse{}, nil)
						return conn, true
					})

				if r.Transitions.Configuration != nil {
					r.configurations.EXPECT().
						UpdateStatus(gomock.Any(), gomock.Any()).
						DoAndReturn(func(ctx context.Context, configuration *configapi.Configuration) error {
							testConfiguration(t, configuration, *r.Transitions.Configuration)
							return nil
						})
				}

				transactionState, ok := r.Transitions.Transactions[r.Context.Index]
				if ok {
					r.transactions.EXPECT().
						UpdateStatus(gomock.Any(), gomock.Any()).
						DoAndReturn(func(ctx context.Context, transaction *configapi.Transaction) error {
							testTransaction(t, transaction, transactionState)
							return nil
						})
				} else {
					r.transactions.EXPECT().
						UpdateStatus(gomock.Any(), gomock.Any()).Return(errors.NewConflict("test conflict"))
				}
			case model.TransactionFailed:
				r.topo.EXPECT().
					Get(gomock.Any(), gomock.Eq(topoapi.ID(fmt.Sprintf("%s-%s", r.Context.NodeID, testTargetID)))).
					DoAndReturn(func(ctx context.Context, id topoapi.ID) (*topoapi.Object, error) {
						return &topoapi.Object{
							ID:   topoapi.ID(fmt.Sprintf("%s-%s", r.Context.NodeID, testTargetID)),
							Type: topoapi.Object_RELATION,
							Obj: &topoapi.Object_Relation{
								Relation: &topoapi.Relation{
									KindID:      topoapi.CONTROLS,
									SrcEntityID: topoapi.ID(r.Context.NodeID),
									TgtEntityID: testTargetID,
								},
							},
						}, nil
					}).AnyTimes()

				r.conns.EXPECT().
					Get(gomock.Any(), gomock.Eq(gnmisb.ConnID(fmt.Sprintf("%s-%s", r.Context.NodeID, testTargetID)))).
					DoAndReturn(func(ctx context.Context, id gnmisb.ConnID) (gnmisb.Conn, bool) {
						conn := gnmimock.NewMockConn(r.ctrl)
						conn.EXPECT().ID().Return(id).AnyTimes()
						conn.EXPECT().
							Set(gomock.Any(), gomock.Any()).
							Return(nil, errors.Status(errors.NewInternal("internal error")).Err())
						return conn, true
					}).AnyTimes()

				transactionState, ok := r.Transitions.Transactions[r.Context.Index]
				assert.True(t, ok)
				r.transactions.EXPECT().
					UpdateStatus(gomock.Any(), gomock.Any()).
					DoAndReturn(func(ctx context.Context, transaction *configapi.Transaction) error {
						testTransaction(t, transaction, transactionState)
						return nil
					})

				if r.Transitions.Configuration != nil {
					r.configurations.EXPECT().
						UpdateStatus(gomock.Any(), gomock.Any()).
						DoAndReturn(func(ctx context.Context, configuration *configapi.Configuration) error {
							testConfiguration(t, configuration, *r.Transitions.Configuration)
							return nil
						})
				} else {
					r.configurations.EXPECT().
						UpdateStatus(gomock.Any(), gomock.Any()).
						Return(errors.NewConflict("test conflict"))
				}
			default:
				if r.Transitions.Configuration != nil {
					r.configurations.EXPECT().
						UpdateStatus(gomock.Any(), gomock.Any()).
						DoAndReturn(func(ctx context.Context, configuration *configapi.Configuration) error {
							testConfiguration(t, configuration, *r.Transitions.Configuration)
							return nil
						})
				} else {
					r.configurations.EXPECT().
						UpdateStatus(gomock.Any(), gomock.Any()).
						Return(errors.NewConflict("test conflict"))
				}

				transactionState, ok := r.Transitions.Transactions[r.Context.Index]
				if ok {
					r.transactions.EXPECT().
						UpdateStatus(gomock.Any(), gomock.Any()).
						DoAndReturn(func(ctx context.Context, transaction *configapi.Transaction) error {
							testTransaction(t, transaction, transactionState)
							return nil
						})
				} else {
					r.transactions.EXPECT().
						UpdateStatus(gomock.Any(), gomock.Any()).Return(errors.NewConflict("test conflict"))
				}
			}
		}
	} else {
		if r.Transitions.Configuration != nil {
			r.configurations.EXPECT().
				UpdateStatus(gomock.Any(), gomock.Any()).
				DoAndReturn(func(ctx context.Context, configuration *configapi.Configuration) error {
					testConfiguration(t, configuration, *r.Transitions.Configuration)
					return nil
				})
		}

		for _, state := range r.Transitions.Transactions {
			transactionState := state
			r.transactions.EXPECT().
				UpdateStatus(gomock.Any(), gomock.Any()).
				DoAndReturn(func(ctx context.Context, transaction *configapi.Transaction) error {
					testTransaction(t, transaction, transactionState)
					return nil
				})
		}
	}
}

func (r *testRunner) teardown() {
	r.ctrl.Finish()
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
		}
	}

	rollbackIndex := configapi.Index(state.Rollback.Index)
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
	transaction.Status.Rollback.Index = rollbackIndex
	transaction.Status.Rollback.Values = rollbackValues
	return transaction
}

func newTransactionPhaseStatus(status *model.TransactionStatus) *configapi.TransactionPhaseStatus {
	if status == nil {
		return nil
	}

	var phase configapi.TransactionPhaseStatus
	switch *status {
	case model.TransactionPending:
		phase.State = configapi.TransactionPhaseStatus_PENDING
	case model.TransactionInProgress:
		phase.State = configapi.TransactionPhaseStatus_IN_PROGRESS
		phase.Start = now()
	case model.TransactionComplete:
		phase.State = configapi.TransactionPhaseStatus_COMPLETE
		phase.Start = now()
		phase.End = now()
	case model.TransactionAborted:
		phase.State = configapi.TransactionPhaseStatus_ABORTED
		phase.Start = now()
		phase.End = now()
	case model.TransactionCanceled:
		phase.State = configapi.TransactionPhaseStatus_CANCELED
		phase.Start = now()
		phase.End = now()
	case model.TransactionFailed:
		phase.State = configapi.TransactionPhaseStatus_FAILED
		phase.Start = now()
		phase.End = now()
	}
	return &phase
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

func testTransactionState(t *testing.T, phase *configapi.TransactionPhaseStatus, status *model.TransactionStatus) bool {
	if status == nil {
		return assert.Nil(t, phase)
	}
	switch *status {
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

func TestCascadingDeleteAlgorithm(t *testing.T) {
	// defining store here
	var store = make(map[string]configapi.PathValue)
	// should be deleted
	store["/x/y/z"] = configapi.PathValue{
		Path: "/x/y/z",
		Value: configapi.TypedValue{
			Bytes:    []byte{0xFF, 0xFF, 0xFF},
			Type:     configapi.ValueType_BYTES,
			TypeOpts: make([]int32, 0),
		},
		Deleted: true,
	}
	// should be kept
	store["/x/y/y"] = configapi.PathValue{
		Path: "/x/y/y",
		Value: configapi.TypedValue{
			Bytes:    []byte{0xFF, 0xFF, 0xFF},
			Type:     configapi.ValueType_BYTES,
			TypeOpts: make([]int32, 0),
		},
		Deleted: false,
	}
	// should be cascadingly deleted
	store["/x/y/z/w"] = configapi.PathValue{
		Path: "/x/y/z/w",
		Value: configapi.TypedValue{
			Bytes:    []byte{0xFF, 0xFF, 0xFF},
			Type:     configapi.ValueType_BYTES,
			TypeOpts: make([]int32, 0),
		},
		Deleted: false,
	}
	// should be cascadingly deleted
	store["/x/y/z/w1"] = configapi.PathValue{
		Path: "/x/y/z/w1",
		Value: configapi.TypedValue{
			Bytes:    []byte{0xFF, 0xFF, 0xFF},
			Type:     configapi.ValueType_BYTES,
			TypeOpts: make([]int32, 0),
		},
		Deleted: false,
	}
	// should be cascadingly deleted
	store["/x/y/y1"] = configapi.PathValue{
		Path: "/x/y/y1",
		Value: configapi.TypedValue{
			Bytes:    []byte{0xFF, 0xFF, 0xFF},
			Type:     configapi.ValueType_BYTES,
			TypeOpts: make([]int32, 0),
		},
		Deleted: false,
	}
	// should be deleted
	store["/x/y/y2"] = configapi.PathValue{
		Path: "/x/y/y2",
		Value: configapi.TypedValue{
			Bytes:    []byte{0xFF, 0xFF, 0xFF},
			Type:     configapi.ValueType_BYTES,
			TypeOpts: make([]int32, 0),
		},
		Deleted: true,
	}
	log.Infof("store is\n%v", store)

	// defining change values here
	var changeValues = make(map[string]configapi.PathValue)
	changeValues["/x/y/z"] = configapi.PathValue{
		Path: "/x/y/z",
		Value: configapi.TypedValue{
			Bytes:    []byte{0xFF, 0xFF, 0xFF},
			Type:     configapi.ValueType_BYTES,
			TypeOpts: make([]int32, 0),
		},
		Deleted: true,
	}
	changeValues["/x/y/y1"] = configapi.PathValue{
		Path: "/x/y/y1",
		Value: configapi.TypedValue{
			Bytes:    []byte{0xFF, 0xFF, 0xFF},
			Type:     configapi.ValueType_BYTES,
			TypeOpts: make([]int32, 0),
		},
		Deleted: false,
	}
	changeValues["/x/y/y2"] = configapi.PathValue{
		Path: "/x/y/y2",
		Value: configapi.TypedValue{
			Bytes:    []byte{0xFF, 0xFF, 0xFF},
			Type:     configapi.ValueType_BYTES,
			TypeOpts: make([]int32, 0),
		},
		Deleted: true,
	}
	log.Infof("changeValues is\n%v", changeValues)

	// cascading delete algorithm
	updatedChangeValues := addDeleteChildren(1, changeValues, store)
	log.Infof("updChangeValues is\n%v", updatedChangeValues)
	assert.Equal(t, 5, len(updatedChangeValues))
	log.Infof("updChangeValues has %d PathValues to delete", len(updatedChangeValues))
	assert.Equal(t, updatedChangeValues["/x/y/z"].Path, store["/x/y/z"].Path)
	assert.Equal(t, updatedChangeValues["/x/y/z"].Value.Type.String(), store["/x/y/z"].Value.Type.String())
	assert.Equal(t, updatedChangeValues["/x/y/y1"].Path, store["/x/y/y1"].Path)
	assert.Equal(t, updatedChangeValues["/x/y/y1"].Value.Type.String(), store["/x/y/y1"].Value.Type.String())
	assert.Equal(t, updatedChangeValues["/x/y/y2"].Path, store["/x/y/y2"].Path)
	assert.Equal(t, updatedChangeValues["/x/y/y2"].Value.Type.String(), store["/x/y/y2"].Value.Type.String())
	assert.Equal(t, updatedChangeValues["/x/y/z/w"].Path, store["/x/y/z/w"].Path)
	assert.Equal(t, updatedChangeValues["/x/y/z/w"].Value.Type.String(), store["/x/y/z/w"].Value.Type.String())
	assert.Equal(t, updatedChangeValues["/x/y/z/w1"].Path, store["/x/y/z/w1"].Path)
	assert.Equal(t, updatedChangeValues["/x/y/z/w1"].Value.Type.String(), store["/x/y/z/w1"].Value.Type.String())
}
