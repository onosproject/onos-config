package proposal

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"github.com/atomix/go-sdk/pkg/test"
	"github.com/golang/mock/gomock"
	configapi "github.com/onosproject/onos-api/go/onos/config/v2"
	topoapi "github.com/onosproject/onos-api/go/onos/topo"
	"github.com/onosproject/onos-config/pkg/pluginregistry"
	"github.com/onosproject/onos-config/pkg/southbound/gnmi"
	configurationstore "github.com/onosproject/onos-config/pkg/store/configuration"
	proposalstore "github.com/onosproject/onos-config/pkg/store/proposal"
	topostore "github.com/onosproject/onos-config/pkg/store/topo"
	"github.com/onosproject/onos-lib-go/pkg/controller"
	"github.com/onosproject/onos-lib-go/pkg/errors"
	"github.com/stretchr/testify/assert"
	"os"
	"testing"
)

func TestReconciler(t *testing.T) {
	file, err := os.Open("testdata/model/Model.log")
	assert.NoError(t, err)

	i := 0
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		i++
		var test TestCase
		assert.NoError(t, json.Unmarshal(scanner.Bytes(), &test))
		if test.Next.Target == nil {
			test.Next.Target = test.Init.Target
		}
		if len(test.Next.Nodes) == 0 {
			test.Next.Nodes = test.Init.Nodes
		}
		if test.Next.Mastership == nil {
			test.Next.Mastership = test.Init.Mastership
		}
		if test.Next.Configuration == nil {
			test.Next.Configuration = test.Init.Configuration
		}
		if len(test.Next.Proposals) == 0 {
			test.Next.Proposals = test.Init.Proposals
		}
		name := fmt.Sprintf("Model-%d", i)
		t.Run(name, func(t *testing.T) {
			testReconciler(t, test)
		})
	}
}

const (
	testTarget                                = "test"
	testTargetType    configapi.TargetType    = "test"
	testTargetVersion configapi.TargetVersion = "1"
)

func testReconciler(t *testing.T, testCase TestCase) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	conns := gnmi.NewMockConnManager(ctrl)
	topo := topostore.NewMockStore(ctrl)
	pluginRegistry := pluginregistry.NewMockPluginRegistry(ctrl)

	conns.EXPECT().Get(gomock.Any(), gomock.Eq(gnmi.ConnID(testTarget))).
		DoAndReturn(func(ctx context.Context, id gnmi.ConnID) (gnmi.Conn, bool) {
			if testCase.Init.Nodes[testCase.Context.Node].Connected {
				conn := gnmi.NewMockConn(ctrl)
				conn.EXPECT().ID().Return(id).AnyTimes()
				return conn, true
			}
			return nil, false
		}).AnyTimes()

	topo.EXPECT().Get(gomock.Any(), gomock.Eq(topoapi.ID(testTarget))).
		DoAndReturn(func(ctx context.Context, id topoapi.ID) (*topoapi.Object, error) {
			obj := &topoapi.Object{
				ID:   testTarget,
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

	for nodeID, node := range testCase.Init.Nodes {
		topo.EXPECT().Get(gomock.Any(), gomock.Eq(topoapi.ID(nodeID))).
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
				if testCase.Init.Mastership.Master == nodeID {
					mastership := topoapi.MastershipState{
						Term:   uint64(testCase.Init.Mastership.Term),
						NodeId: fmt.Sprintf("%s-%s", nodeID, testTarget),
					}
					if err := obj.SetAspect(&mastership); err != nil {
						return nil, err
					}
				}
				return obj, nil
			}).AnyTimes()

		if node.Connected {
			topo.EXPECT().Get(gomock.Any(), gomock.Eq(topoapi.ID(fmt.Sprintf("%s-%s", nodeID, testTarget)))).
				DoAndReturn(func(ctx context.Context, id topoapi.ID) (*topoapi.Object, error) {
					return &topoapi.Object{
						ID:   topoapi.ID(nodeID),
						Type: topoapi.Object_RELATION,
						Obj: &topoapi.Object_Relation{
							Relation: &topoapi.Relation{
								KindID: topoapi.CONTROLS,
							},
						},
					}, nil
				}).AnyTimes()
		}
	}
	cluster := test.NewClient()
	defer cluster.Close()

	configurations, err := configurationstore.NewAtomixStore(cluster)
	assert.NoError(t, err)

	configurationID := configurationstore.NewID(testTarget, testTargetType, testTargetVersion)
	configuration := &configapi.Configuration{
		ID:       configurationID,
		TargetID: testTarget,
		Values:   make(map[string]*configapi.PathValue),
		Status: configapi.ConfigurationStatus{
			Mastership: configapi.MastershipInfo{
				Master: string(testCase.Init.Mastership.Master),
				Term:   configapi.MastershipTerm(testCase.Init.Mastership.Term),
			},
			Committed: configapi.CommittedConfigurationStatus{
				Index: configapi.Index(testCase.Init.Configuration.Committed.Index),
			},
			Applied: configapi.AppliedConfigurationStatus{
				Index: configapi.Index(testCase.Init.Configuration.Applied.Index),
				Mastership: configapi.MastershipInfo{
					Term: configapi.MastershipTerm(testCase.Init.Configuration.Applied.Term),
				},
			},
		},
	}
	switch testCase.Init.Configuration.Status {
	case ConfigurationStatusInProgress:
		configuration.Status.State = configapi.ConfigurationStatus_SYNCHRONIZING
	case ConfigurationStatusComplete:
		configuration.Status.State = configapi.ConfigurationStatus_SYNCHRONIZED
	}
	for key, value := range testCase.Init.Configuration.Committed.Values {
		pathValue := &configapi.PathValue{
			Path:  key,
			Index: configapi.Index(value.Index),
		}
		if value.Value != nil {
			pathValue.Value = configapi.TypedValue{
				Bytes: []byte(*value.Value),
				Type:  configapi.ValueType_STRING,
			}
		} else {
			pathValue.Deleted = true
		}
		configuration.Values[key] = pathValue
	}
	for key, value := range testCase.Init.Configuration.Applied.Values {
		pathValue := &configapi.PathValue{
			Path:  key,
			Index: configapi.Index(value.Index),
		}
		if value.Value != nil {
			pathValue.Value = configapi.TypedValue{
				Bytes: []byte(*value.Value),
				Type:  configapi.ValueType_STRING,
			}
		} else {
			pathValue.Deleted = true
		}
		configuration.Status.Applied.Values[key] = pathValue
	}
	assert.NoError(t, configurations.Create(context.TODO(), configuration))

	proposals, err := proposalstore.NewAtomixStore(cluster)
	assert.NoError(t, err)

	for i, proposalState := range testCase.Init.Proposals {
		index := Index(i + 1)
		proposalID := proposalstore.NewID(testTarget, configapi.Index(index))
		proposal := &configapi.Proposal{
			ID:               proposalID,
			TransactionIndex: configapi.Index(index),
			TargetID:         testTarget,
			TargetTypeVersion: configapi.TargetTypeVersion{
				TargetType:    testTargetType,
				TargetVersion: testTargetVersion,
			},
		}

		proposal.Status.RollbackIndex = configapi.Index(proposalState.Rollback.Index)
		rollbackValues := make(map[string]configapi.PathValue)
		for key, value := range proposalState.Rollback.Values {
			pathValue := configapi.PathValue{
				Path:  key,
				Index: configapi.Index(proposalState.Rollback.Index),
			}
			if value.Value != nil {
				pathValue.Value = configapi.TypedValue{
					Bytes: []byte(*value.Value),
					Type:  configapi.ValueType_STRING,
				}
			} else {
				pathValue.Deleted = true
			}
			rollbackValues[key] = pathValue
		}

		switch proposalState.Type {
		case ProposalTypeChange:
			pathValues := make(map[string]*configapi.PathValue)
			for key, value := range proposalState.Change.Values {
				pathValue := &configapi.PathValue{
					Path:  key,
					Index: configapi.Index(index),
				}
				if value.Value != nil {
					pathValue.Value = configapi.TypedValue{
						Bytes: []byte(*value.Value),
						Type:  configapi.ValueType_STRING,
					}
				} else {
					pathValue.Deleted = true
				}
				pathValues[key] = pathValue
			}
			proposal.Details = &configapi.Proposal_Change{
				Change: &configapi.ChangeProposal{
					Values: pathValues,
				},
			}
		case ProposalTypeRollback:
			proposal.Details = &configapi.Proposal_Rollback{
				Rollback: &configapi.RollbackProposal{
					RollbackIndex: configapi.Index(proposalState.Rollback.Index),
				},
			}
		}

		switch proposalState.Phase {
		case ProposalPhaseCommit:
			proposal.Status.Phases.Commit = &configapi.ProposalCommitPhase{}
			switch proposalState.Status {
			case ProposalStatusInProgress:
				proposal.Status.Phases.Commit.State = configapi.ProposalCommitPhase_COMMITTING
			case ProposalStatusComplete:
				proposal.Status.Phases.Commit.State = configapi.ProposalCommitPhase_COMMITTED
			case ProposalStatusFailed:
				// TODO
			}

			// If the next state is failed, mock the plugin registry to fail validation.
			if testCase.Next.Proposals[i].Status == ProposalStatusFailed {
				plugin := pluginregistry.NewMockModelPlugin(ctrl)
				plugin.EXPECT().Validate(gomock.Any(), gomock.Any()).Return(errors.NewInvalid("validation error")).AnyTimes()
				pluginRegistry.EXPECT().GetPlugin(gomock.Eq(testTargetType), gomock.Eq(testTargetVersion)).Return(plugin, true).AnyTimes()
			} else {
				plugin := pluginregistry.NewMockModelPlugin(ctrl)
				plugin.EXPECT().Validate(gomock.Any(), gomock.Any()).Return(nil).AnyTimes()
				pluginRegistry.EXPECT().GetPlugin(gomock.Eq(testTargetType), gomock.Eq(testTargetVersion)).Return(plugin, true).AnyTimes()
			}
		case ProposalPhaseApply:
			proposal.Status.Phases.Validate = &configapi.ProposalValidatePhase{
				State: configapi.ProposalValidatePhase_VALIDATED,
			}
			proposal.Status.Phases.Commit = &configapi.ProposalCommitPhase{
				State: configapi.ProposalCommitPhase_COMMITTED,
			}
			proposal.Status.Phases.Apply = &configapi.ProposalApplyPhase{}
			switch proposalState.Status {
			case ProposalStatusInProgress:
				proposal.Status.Phases.Apply.State = configapi.ProposalApplyPhase_APPLYING
			case ProposalStatusComplete:
				proposal.Status.Phases.Apply.State = configapi.ProposalApplyPhase_APPLIED
			case ProposalStatusFailed:
				proposal.Status.Phases.Apply.State = configapi.ProposalApplyPhase_FAILED
			}
		}
		assert.NoError(t, proposals.Create(context.TODO(), proposal))
	}

	reconciler := &Reconciler{
		conns:          conns,
		topo:           topo,
		configurations: configurations,
		proposals:      proposals,
		pluginRegistry: pluginRegistry,
	}

	proposalID := proposalstore.NewID(testTarget, configapi.Index(testCase.Context.Index))
	_, err = reconciler.Reconcile(controller.NewID(proposalID))
	assert.NoError(t, err)

	configuration, err = configurations.Get(context.TODO(), configurationID)
	assert.NoError(t, err)

	assert.Equal(t, configapi.Index(testCase.Next.Configuration.Committed.Index), configuration.Index)
	for key, value := range testCase.Next.Configuration.Committed.Values {
		assert.Equal(t, configapi.Index(value.Index), configuration.Values[key].Index)
		if value.Value != nil {
			assert.Equal(t, string(*value.Value), string(configuration.Values[key].Value.Bytes))
		} else {
			assert.True(t, configuration.Values[key].Deleted)
		}
	}

	assert.Equal(t, configapi.MastershipTerm(testCase.Next.Configuration.Applied.Term), configuration.Status.Applied.Mastership.Term)
	for key, value := range testCase.Next.Configuration.Applied.Values {
		assert.Equal(t, configapi.Index(value.Index), configuration.Status.Applied.Values[key].Index)
		if value.Value != nil {
			assert.Equal(t, string(*value.Value), string(configuration.Status.Applied.Values[key].Value.Bytes))
		} else {
			assert.True(t, configuration.Status.Applied.Values[key].Deleted)
		}
	}

	switch testCase.Next.Configuration.Status {
	case ConfigurationStatusInProgress:
		assert.Equal(t, configapi.ConfigurationStatus_SYNCHRONIZING, configuration.Status.State)
	case ConfigurationStatusComplete:
		assert.Equal(t, configapi.ConfigurationStatus_SYNCHRONIZED, configuration.Status.State)
	}

	for i, proposalState := range testCase.Next.Proposals {
		index := Index(i + 1)
		proposalID := proposalstore.NewID(testTarget, configapi.Index(index))
		proposal, err := proposals.Get(context.TODO(), proposalID)
		assert.NoError(t, err)

		switch proposalState.Phase {
		case ProposalPhaseCommit:
			assert.NotNil(t, proposal.Status.Phases.Commit)
			switch proposalState.Status {
			case ProposalStatusInProgress:
				assert.Equal(t, configapi.ProposalCommitPhase_COMMITTING, proposal.Status.Phases.Commit.State)
			case ProposalStatusComplete:
				assert.Equal(t, configapi.ProposalCommitPhase_COMMITTED, proposal.Status.Phases.Commit.State)
			case ProposalStatusFailed:
				// TODO
			}
		case ProposalPhaseApply:
			assert.NotNil(t, proposal.Status.Phases.Apply)
			switch proposalState.Status {
			case ProposalStatusInProgress:
				assert.Equal(t, configapi.ProposalApplyPhase_APPLYING, proposal.Status.Phases.Apply.State)
			case ProposalStatusComplete:
				assert.Equal(t, configapi.ProposalApplyPhase_APPLIED, proposal.Status.Phases.Apply.State)
			case ProposalStatusFailed:
				assert.Equal(t, configapi.ProposalApplyPhase_FAILED, proposal.Status.Phases.Apply.State)
			}
		}
	}
}

type TestCase struct {
	Context TestContext `json:"context"`
	Init    TestState   `json:"init"`
	Next    TestState   `json:"next"`
}

type TestContext struct {
	Node  NodeID `json:"node"`
	Index Index  `json:"index"`
}

type TestState struct {
	Proposals     []ProposalState      `json:"proposals"`
	Configuration *ConfigurationState  `json:"configuration"`
	Mastership    *MastershipState     `json:"mastership"`
	Nodes         map[NodeID]NodeState `json:"nodes"`
	Target        *TargetState         `json:"target"`
}

type Index uint64

type Revision uint64

type Term uint64

type NodeID string

type NodeState struct {
	Incarnation int  `json:"incarnation"`
	Connected   bool `json:"connected"`
}

type TargetState struct {
	Incarnation int    `json:"incarnation"`
	Values      Values `json:"values"`
	Running     bool   `json:"running"`
}

type ProposalPhase string

const (
	ProposalPhaseCommit ProposalPhase = "Commit"
	ProposalPhaseApply  ProposalPhase = "Apply"
)

type ProposalStatus string

const (
	ProposalStatusInProgress ProposalStatus = "InProgress"
	ProposalStatusComplete   ProposalStatus = "Complete"
	ProposalStatusFailed     ProposalStatus = "Failed"
)

type ProposalType string

const (
	ProposalTypeChange   ProposalType = "Change"
	ProposalTypeRollback ProposalType = "Rollback"
)

type ProposalState struct {
	Phase    ProposalPhase  `json:"phase"`
	Status   ProposalStatus `json:"state"`
	Type     ProposalType   `json:"type"`
	Change   ChangeState    `json:"change"`
	Rollback RollbackState  `json:"rollback"`
}

type ChangeState struct {
	Index  Index  `json:"index"`
	Values Values `json:"values"`
}

type RollbackState struct {
	Index  Index  `json:"index"`
	Values Values `json:"values"`
}

type Values map[string]ValueState

func (s *Values) UnmarshalJSON(data []byte) error {
	var list []any
	if err := json.Unmarshal(data, &list); err != nil {
		values := make(map[string]ValueState)
		if err := json.Unmarshal(data, &values); err != nil {
			return err
		}
		state := make(Values)
		for key, value := range values {
			state[key] = value
		}
		*s = state
	}
	return nil
}

const nilString = "<nil>"

type ValueState struct {
	Index Index  `json:"index"`
	Value *Value `json:"value"`
}

type Value string

func (s *Value) UnmarshalJSON(data []byte) error {
	var value string
	if err := json.Unmarshal(data, &value); err != nil {
		return err
	}
	if value != nilString {
		*s = Value(value)
	}
	return nil
}

type ConfigurationState struct {
	Status    ConfigurationStatus         `json:"state"`
	Committed ConfigurationCommittedState `json:"committed"`
	Applied   ConfigurationAppliedState   `json:"applied"`
}

type ConfigurationCommittedState struct {
	Index    Index    `json:"index"`
	Revision Revision `json:"revision"`
	Term     Term     `json:"term"`
	Values   Values   `json:"values"`
}

type ConfigurationAppliedState struct {
	Index    Index    `json:"index"`
	Revision Revision `json:"revision"`
	Term     Term     `json:"term"`
	Values   Values   `json:"values"`
}

type ConfigurationStatus string

const (
	ConfigurationStatusInProgress ConfigurationStatus = "InProgress"
	ConfigurationStatusComplete   ConfigurationStatus = "Complete"
)

type MastershipState struct {
	Term   Term   `json:"term"`
	Master NodeID `json:"master"`
}
