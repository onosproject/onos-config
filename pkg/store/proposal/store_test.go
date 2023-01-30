// SPDX-FileCopyrightText: 2020-present Open Networking Foundation <info@opennetworking.org>
//
// SPDX-License-Identifier: Apache-2.0

package proposal

import (
	"context"
	"github.com/atomix/go-sdk/pkg/test"
	"testing"
	"time"

	configapi "github.com/onosproject/onos-api/go/onos/config/v2"
	"github.com/stretchr/testify/assert"
)

func TestProposalStore(t *testing.T) {
	cluster := test.NewClient()
	defer cluster.Close()

	store1, err := NewAtomixStore(cluster)
	assert.NoError(t, err)

	store2, err := NewAtomixStore(cluster)
	assert.NoError(t, err)

	target1 := configapi.TargetID("target-1")
	target2 := configapi.TargetID("target-2")

	ch := make(chan configapi.ProposalEvent)
	err = store2.Watch(context.Background(), ch)
	assert.NoError(t, err)

	target1ConfigValues := make(map[string]*configapi.PathValue)
	target1ConfigValues["/foo"] = &configapi.PathValue{
		Value: configapi.TypedValue{
			Bytes: []byte("Hello world!"),
			Type:  configapi.ValueType_STRING,
		},
	}

	target1Config := &configapi.Proposal{
		ID:               configapi.ProposalID(target1),
		TargetID:         target1,
		TransactionIndex: 1,
		Details: &configapi.Proposal_Change{
			Change: &configapi.ChangeProposal{
				Values: target1ConfigValues,
			},
		},
	}

	target2ConfigValues := make(map[string]*configapi.PathValue)
	target2ConfigValues["/foo"] = &configapi.PathValue{
		Value: configapi.TypedValue{
			Bytes: []byte("Hello world again!"),
			Type:  configapi.ValueType_STRING,
		},
	}
	target2Config := &configapi.Proposal{
		ID:               configapi.ProposalID(target2),
		TargetID:         target2,
		TransactionIndex: 1,
		Details: &configapi.Proposal_Change{
			Change: &configapi.ChangeProposal{
				Values: target2ConfigValues,
			},
		},
	}

	err = store1.Create(context.TODO(), target1Config)
	assert.NoError(t, err)
	assert.Equal(t, configapi.ProposalID(target1), target1Config.ID)
	assert.NotEqual(t, configapi.Revision(0), target1Config.Revision)

	err = store2.Create(context.TODO(), target2Config)
	assert.NoError(t, err)
	assert.Equal(t, configapi.ProposalID(target2), target2Config.ID)
	assert.NotEqual(t, configapi.Revision(0), target2Config.Revision)

	// Get the proposal
	target1Config, err = store2.Get(context.TODO(), configapi.ProposalID(target1))
	assert.NoError(t, err)
	assert.NotNil(t, target1Config)
	assert.Equal(t, configapi.ProposalID(target1), target1Config.ID)
	assert.NotEqual(t, configapi.Revision(0), target1Config.Revision)

	// Verify events were received for the proposals
	proposalEvent := nextEvent(t, ch)
	assert.NotNil(t, proposalEvent)
	proposalEvent = nextEvent(t, ch)
	assert.NotNil(t, proposalEvent)

	// Watch events for a specific proposal
	proposalCh := make(chan configapi.ProposalEvent)
	err = store1.Watch(context.TODO(), proposalCh, WithProposalID(target2Config.ID))
	assert.NoError(t, err)

	// Update one of the proposals
	revision := target2Config.Revision
	err = store1.Update(context.TODO(), target2Config)
	assert.NoError(t, err)
	assert.NotEqual(t, revision, target2Config.Revision)

	event := <-proposalCh
	assert.Equal(t, target2Config.ID, event.Proposal.ID)

	// Lists proposals
	proposalList, err := store1.List(context.TODO())
	assert.NoError(t, err)
	assert.Equal(t, 2, len(proposalList))

	// Read and then update the proposal
	target2Config, err = store2.Get(context.TODO(), configapi.ProposalID(target2))
	assert.NoError(t, err)
	assert.NotNil(t, target2Config)
	now := time.Now()
	target2Config.Status.Phases.Initialize = &configapi.ProposalInitializePhase{
		ProposalPhaseStatus: configapi.ProposalPhaseStatus{
			Start: &now,
		},
	}
	revision = target2Config.Revision
	err = store1.Update(context.TODO(), target2Config)
	assert.NoError(t, err)
	assert.NotEqual(t, revision, target2Config.Revision)

	event = <-proposalCh
	assert.Equal(t, target2Config.ID, event.Proposal.ID)

	// Verify that concurrent updates fail
	target1Config11, err := store1.Get(context.TODO(), configapi.ProposalID(target1))
	assert.NoError(t, err)
	target1Config12, err := store2.Get(context.TODO(), configapi.ProposalID(target1))
	assert.NoError(t, err)

	target1Config11.Status.Phases.Initialize = &configapi.ProposalInitializePhase{
		ProposalPhaseStatus: configapi.ProposalPhaseStatus{
			Start: &now,
		},
	}
	err = store1.Update(context.TODO(), target1Config11)
	assert.NoError(t, err)

	target1Config12.Status.Phases.Initialize = &configapi.ProposalInitializePhase{
		ProposalPhaseStatus: configapi.ProposalPhaseStatus{
			Start: &now,
		},
	}
	err = store2.Update(context.TODO(), target1Config12)
	assert.Error(t, err)

	// Verify events were received again
	proposalEvent = nextEvent(t, ch)
	assert.NotNil(t, proposalEvent)
	proposalEvent = nextEvent(t, ch)
	assert.NotNil(t, proposalEvent)
	proposalEvent = nextEvent(t, ch)
	assert.NotNil(t, proposalEvent)

	// Checks list of proposal after deleting a proposal
	proposalList, err = store2.List(context.TODO())
	assert.NoError(t, err)
	assert.Equal(t, 2, len(proposalList))

	err = store1.Close(context.TODO())
	assert.NoError(t, err)

	err = store2.Close(context.TODO())
	assert.NoError(t, err)

}

func nextEvent(t *testing.T, ch chan configapi.ProposalEvent) *configapi.Proposal {
	select {
	case c := <-ch:
		return &c.Proposal
	case <-time.After(5 * time.Second):
		t.FailNow()
	}
	return nil
}
