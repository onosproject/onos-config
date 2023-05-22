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

	eventCh := make(chan configapi.ProposalEvent)
	err = store2.Watch(context.Background(), eventCh)
	assert.NoError(t, err)

	target := configapi.Target{
		ID:      "target-1",
		Type:    "foo",
		Version: "1",
	}

	proposal1 := &configapi.Proposal{
		ObjectMeta: configapi.ObjectMeta{
			Key: "proposal-1",
		},
		ID: configapi.ProposalID{
			Target: target,
		},
		Values: map[string]*configapi.PathValue{
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

	proposal2 := &configapi.Proposal{
		ObjectMeta: configapi.ObjectMeta{
			Key: "proposal-2",
		},
		ID: configapi.ProposalID{
			Target: target,
		},
		Values: map[string]*configapi.PathValue{
			"foo": {
				Deleted: true,
			},
		},
	}

	// Create a new proposal
	err = store1.Create(context.TODO(), proposal1)
	assert.NoError(t, err)
	assert.Equal(t, "proposal-1", proposal1.Key)
	assert.Equal(t, configapi.Index(1), proposal1.ID.Index)
	assert.NotEqual(t, configapi.Revision(0), proposal1.Revision)

	// Get the proposal
	proposal1, err = store2.GetKey(context.TODO(), target, "proposal-1")
	assert.NoError(t, err)
	assert.NotNil(t, proposal1)
	assert.Equal(t, "proposal-1", proposal1.Key)
	assert.Equal(t, configapi.Index(1), proposal1.ID.Index)
	assert.NotEqual(t, configapi.Revision(0), proposal1.Revision)

	proposalEvent := nextProposal(t, eventCh)
	assert.Equal(t, "proposal-1", proposalEvent.Key)

	// Create another proposal
	err = store2.Create(context.TODO(), proposal2)
	assert.NoError(t, err)
	assert.Equal(t, "proposal-2", proposal2.Key)
	assert.Equal(t, configapi.Index(2), proposal2.ID.Index)
	assert.NotEqual(t, configapi.Revision(0), proposal2.Revision)

	proposalEvent = nextProposal(t, eventCh)
	assert.Equal(t, "proposal-2", proposalEvent.Key)

	// Watch events for a specific proposal
	proposalCh := make(chan configapi.ProposalEvent)
	err = store1.Watch(context.TODO(), proposalCh, WithProposalID(proposal2.ID))
	assert.NoError(t, err)

	// Update one of the proposals
	revision := proposal2.Revision
	err = store1.Update(context.TODO(), proposal2)
	assert.NoError(t, err)
	assert.NotEqual(t, revision, proposal2.Revision)

	proposalEvent = nextProposal(t, eventCh)
	assert.Equal(t, "proposal-2", proposalEvent.Key)

	event := nextEvent(t, proposalCh)
	assert.Equal(t, proposal2.ID, event.Proposal.ID)

	// Read and then update the proposal
	proposal2, err = store2.GetKey(context.TODO(), target, "proposal-2")
	assert.NoError(t, err)
	assert.NotNil(t, proposal2)
	now := time.Now()
	proposal2.Status.Change = &configapi.ProposalChangeStatus{}
	proposal2.Status.Change.Commit.Start = &now
	revision = proposal2.Revision
	err = store1.Update(context.TODO(), proposal2)
	assert.NoError(t, err)
	assert.NotEqual(t, revision, proposal2.Revision)

	proposalEvent = nextProposal(t, eventCh)
	assert.Equal(t, "proposal-2", proposalEvent.Key)

	event = nextEvent(t, proposalCh)
	assert.Equal(t, proposal2.ID, event.Proposal.ID)

	// Verify that concurrent updates fail
	proposal11, err := store1.GetKey(context.TODO(), target, "proposal-1")
	assert.NoError(t, err)
	proposal12, err := store2.GetKey(context.TODO(), target, "proposal-1")
	assert.NoError(t, err)

	proposal11.Status.Change = &configapi.ProposalChangeStatus{}
	proposal11.Status.Change.Commit.Start = &now
	err = store1.Update(context.TODO(), proposal11)
	assert.NoError(t, err)

	proposal11.Status.Change = &configapi.ProposalChangeStatus{}
	proposal11.Status.Change.Commit.Start = &now
	err = store2.Update(context.TODO(), proposal12)
	assert.Error(t, err)

	proposalEvent = nextProposal(t, eventCh)
	assert.Equal(t, "proposal-1", proposalEvent.Key)

	// List the proposals
	proposals, err := store1.List(context.TODO())
	assert.NoError(t, err)
	assert.Equal(t, 2, len(proposals))

	proposal := &configapi.Proposal{
		ObjectMeta: configapi.ObjectMeta{
			Key: "proposal-3",
		},
		ID: configapi.ProposalID{
			Target: target,
		},
		Values: map[string]*configapi.PathValue{
			"foo": {
				Value: configapi.TypedValue{
					Bytes: []byte("Hello world!"),
					Type:  configapi.ValueType_STRING,
				},
			},
		},
	}

	err = store1.Create(context.TODO(), proposal)
	assert.NoError(t, err)

	proposal = &configapi.Proposal{
		ObjectMeta: configapi.ObjectMeta{
			Key: "proposal-4",
		},
		ID: configapi.ProposalID{
			Target: target,
		},
		Values: map[string]*configapi.PathValue{
			"bar": {
				Value: configapi.TypedValue{
					Bytes: []byte("Hello world!"),
					Type:  configapi.ValueType_STRING,
				},
			},
		},
	}

	err = store1.Create(context.TODO(), proposal)
	assert.NoError(t, err)

	eventCh = make(chan configapi.ProposalEvent)
	err = store1.Watch(context.TODO(), eventCh, WithReplay())
	assert.NoError(t, err)

	proposal = nextProposal(t, eventCh)
	assert.Equal(t, configapi.Index(1), proposal.ID.Index)
	proposal = nextProposal(t, eventCh)
	assert.Equal(t, configapi.Index(2), proposal.ID.Index)
	proposal = nextProposal(t, eventCh)
	assert.Equal(t, configapi.Index(3), proposal.ID.Index)
	proposal = nextProposal(t, eventCh)
	assert.Equal(t, configapi.Index(4), proposal.ID.Index)
}

func nextEvent(t *testing.T, ch chan configapi.ProposalEvent) *configapi.ProposalEvent {
	select {
	case e := <-ch:
		return &e
	case <-time.After(5 * time.Second):
		t.FailNow()
	}
	return nil
}

func nextProposal(t *testing.T, ch chan configapi.ProposalEvent) *configapi.Proposal {
	return &nextEvent(t, ch).Proposal
}
