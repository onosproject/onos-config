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
	proposalstore "github.com/onosproject/onos-config/pkg/store/proposal"
	topostore "github.com/onosproject/onos-config/pkg/store/topo"
	"github.com/onosproject/onos-lib-go/pkg/controller"
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
		var test testCase
		assert.NoError(t, json.Unmarshal(scanner.Bytes(), &test))
		name := fmt.Sprintf("Model-%d", i)
		t.Run(name, func(t *testing.T) {
			testReconciler(t, test)
		})
	}
}

const testTarget = "test"

func testReconciler(t *testing.T, trace testCase) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	conns := gnmi.NewMockConnManager(ctrl)
	topo := topostore.NewMockStore(ctrl)
	pluginRegistry := pluginregistry.NewMockPluginRegistry(ctrl)

	conns.EXPECT().Get(gomock.Any(), gomock.Eq(gnmi.ConnID(testTarget))).
		DoAndReturn(func(ctx context.Context, id gnmi.ConnID) (gnmi.Conn, bool) {
			if trace.Init.Nodes[trace.Context.Node].Connected {
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
				Type:    "test",
				Version: "1",
				Address: "localhost:1234",
			}
			if err := obj.SetAspect(&configurable); err != nil {
				return nil, err
			}
			return obj, nil
		}).AnyTimes()

	for nodeID, node := range trace.Init.Nodes {
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
				if trace.Init.Mastership.Master == nodeID {
					mastership := topoapi.MastershipState{
						Term:   uint64(trace.Init.Mastership.Term),
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

	proposals, err := proposalstore.NewAtomixStore(cluster)
	assert.NoError(t, err)

	for i, proposalState := range trace.Init.Proposals {
		index := Index(i + 1)
		proposalID := proposalstore.NewID(testTarget, configapi.Index(index))
		proposal := &configapi.Proposal{
			ID:       proposalID,
			TargetID: testTarget,
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
		case ProposalPhaseApply:
			assert.NotNil(t, proposal.Status.Phases.Apply)
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
		proposals:      proposals,
		pluginRegistry: pluginRegistry,
	}

	proposalID := proposalstore.NewID(testTarget, configapi.Index(trace.Context.Index))
	_, err = reconciler.Reconcile(controller.NewID(proposalID))
	assert.NoError(t, err)

	for i, proposalState := range trace.Next.Proposals {
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

type testCase struct {
	Context testContext `json:"context"`
	Init    testState   `json:"init"`
	Next    testState   `json:"next"`
}

type testContext struct {
	Node  NodeID `json:"node"`
	Index Index  `json:"index"`
}

type testState struct {
	Proposals     []ProposalState      `json:"proposals"`
	Configuration ConfigurationState   `json:"configuration"`
	Mastership    MastershipState      `json:"mastership"`
	Nodes         map[NodeID]NodeState `json:"nodes"`
	Target        TargetState          `json:"target"`
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
	Index int    `json:"index"`
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
