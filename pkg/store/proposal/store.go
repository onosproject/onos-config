// Copyright 2022-present Open Networking Foundation.
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

package proposal

import (
	"context"
	"fmt"
	"time"

	"github.com/atomix/atomix-go-framework/pkg/atomix/meta"

	"github.com/golang/protobuf/proto"

	configapi "github.com/onosproject/onos-api/go/onos/config/v2"

	"github.com/onosproject/onos-lib-go/pkg/logging"

	"github.com/atomix/atomix-go-client/pkg/atomix"
	_map "github.com/atomix/atomix-go-client/pkg/atomix/map"
	"github.com/onosproject/onos-lib-go/pkg/errors"
)

var log = logging.GetLogger("store", "proposal")

// NewID returns a new Proposal ID for the given target/type/version
func NewID(targetID configapi.TargetID, index configapi.Index) configapi.ProposalID {
	return configapi.ProposalID(fmt.Sprintf("%s-%d", targetID, index))
}

// Store proposal store interface
type Store interface {
	// Get gets the proposal intended for a given target ID
	Get(ctx context.Context, id configapi.ProposalID) (*configapi.Proposal, error)

	// Create creates a proposal
	Create(ctx context.Context, proposal *configapi.Proposal) error

	// Update updates a proposal
	Update(ctx context.Context, proposal *configapi.Proposal) error

	// Delete deletes a proposal
	Delete(ctx context.Context, proposal *configapi.Proposal) error

	// List lists all the proposal
	List(ctx context.Context) ([]*configapi.Proposal, error)

	// Watch watches proposal changes
	Watch(ctx context.Context, ch chan<- configapi.ProposalEvent, opts ...WatchOption) error

	// UpdateStatus updates a proposal status
	UpdateStatus(ctx context.Context, proposal *configapi.Proposal) error

	Close(ctx context.Context) error
}

// NewAtomixStore returns a new persistent Store
func NewAtomixStore(client atomix.Client) (Store, error) {
	proposals, err := client.GetMap(context.Background(), "onos-config-proposals")
	if err != nil {
		return nil, errors.FromAtomix(err)
	}
	return &proposalStore{
		proposals: proposals,
	}, nil
}

type proposalStore struct {
	proposals _map.Map
}

// WatchOption is a proposal option for Watch calls
type WatchOption interface {
	apply([]_map.WatchOption) []_map.WatchOption
}

// watchReplyOption is an option to replay events on watch
type watchReplayOption struct {
}

func (o watchReplayOption) apply(opts []_map.WatchOption) []_map.WatchOption {
	return append(opts, _map.WithReplay())
}

// WithReplay returns a WatchOption that replays past changes
func WithReplay() WatchOption {
	return watchReplayOption{}
}

type watchIDOption struct {
	id configapi.ProposalID
}

func (o watchIDOption) apply(opts []_map.WatchOption) []_map.WatchOption {
	return append(opts, _map.WithFilter(_map.Filter{
		Key: string(o.id),
	}))
}

// WithProposalID returns a Watch option that watches for proposals based on a  given proposal ID
func WithProposalID(id configapi.ProposalID) WatchOption {
	return watchIDOption{id: id}
}

func (s *proposalStore) Get(ctx context.Context, id configapi.ProposalID) (*configapi.Proposal, error) {
	log.Debugf("Getting proposal %s", id)
	entry, err := s.proposals.Get(ctx, string(id))
	if err != nil {
		return nil, errors.FromAtomix(err)
	}
	return decodeProposal(*entry)
}

func (s *proposalStore) Create(ctx context.Context, proposal *configapi.Proposal) error {
	if proposal.ID == "" {
		return errors.NewInvalid("no proposal ID specified")
	}
	if proposal.TargetID == "" {
		return errors.NewInvalid("no target ID specified")
	}
	if proposal.Revision != 0 {
		return errors.NewInvalid("cannot create proposal with revision")
	}
	if proposal.Version != 0 {
		return errors.NewInvalid("cannot create proposal with version")
	}
	proposal.Revision = 1
	proposal.Created = time.Now()
	proposal.Updated = time.Now()

	log.Debugf("Creating proposal %s", proposal.ID)
	bytes, err := proto.Marshal(proposal)
	if err != nil {
		return errors.NewInvalid("proposal encoding failed: %v", err)
	}

	entry, err := s.proposals.Put(ctx, string(proposal.ID), bytes, _map.IfNotSet())
	if err != nil {
		return errors.FromAtomix(err)
	}
	proposal.Version = uint64(entry.Revision)
	return nil
}

func (s *proposalStore) Update(ctx context.Context, proposal *configapi.Proposal) error {
	if proposal.ID == "" {
		return errors.NewInvalid("no proposal ID specified")
	}
	if proposal.TransactionIndex == 0 {
		return errors.NewInvalid("no transaction index specified")
	}
	if proposal.TargetID == "" {
		return errors.NewInvalid("no target ID specified")
	}
	if proposal.Revision == 0 {
		return errors.NewInvalid("proposal must contain a revision on update")
	}
	if proposal.Version == 0 {
		return errors.NewInvalid("proposal must contain a version on update")
	}
	proposal.Revision++
	proposal.Updated = time.Now()

	log.Debugf("Updating proposal %s", proposal.ID)
	bytes, err := proto.Marshal(proposal)
	if err != nil {
		return errors.NewInvalid("proposal encoding failed: %v", err)
	}
	entry, err := s.proposals.Put(ctx, string(proposal.ID), bytes, _map.IfMatch(meta.NewRevision(meta.Revision(proposal.Version))))
	if err != nil {
		return errors.FromAtomix(err)
	}
	proposal.Version = uint64(entry.Revision)
	return nil
}

func (s *proposalStore) UpdateStatus(ctx context.Context, proposal *configapi.Proposal) error {
	if proposal.ID == "" {
		return errors.NewInvalid("no proposal ID specified")
	}
	if proposal.TransactionIndex == 0 {
		return errors.NewInvalid("no transaction index specified")
	}
	if proposal.TargetID == "" {
		return errors.NewInvalid("no target ID specified")
	}
	if proposal.Revision == 0 {
		return errors.NewInvalid("proposal must contain a revision on update")
	}
	if proposal.Version == 0 {
		return errors.NewInvalid("proposal must contain a version on update")
	}
	proposal.Updated = time.Now()

	log.Debugf("Updating proposal %s", proposal.ID)
	bytes, err := proto.Marshal(proposal)
	if err != nil {
		return errors.NewInvalid("proposal encoding failed: %v", err)
	}
	entry, err := s.proposals.Put(ctx, string(proposal.ID), bytes, _map.IfMatch(meta.NewRevision(meta.Revision(proposal.Version))))
	if err != nil {
		return errors.FromAtomix(err)
	}
	proposal.Version = uint64(entry.Revision)
	return nil
}

func (s *proposalStore) Delete(ctx context.Context, proposal *configapi.Proposal) error {
	if proposal.Version == 0 {
		return errors.NewInvalid("proposal must contain a version on delete")
	}

	if proposal.Deleted == nil {
		log.Debugf("Updating proposal %s", proposal.ID)
		t := time.Now()
		proposal.Deleted = &t
		bytes, err := proto.Marshal(proposal)
		if err != nil {
			return errors.NewInvalid("proposal encoding failed: %v", err)
		}
		entry, err := s.proposals.Put(ctx, string(proposal.ID), bytes, _map.IfMatch(meta.NewRevision(meta.Revision(proposal.Version))))
		if err != nil {
			return errors.FromAtomix(err)
		}
		proposal.Version = uint64(entry.Revision)
	} else {
		log.Debugf("Deleting proposal %s", proposal.ID)
		_, err := s.proposals.Remove(ctx, string(proposal.ID), _map.IfMatch(meta.NewRevision(meta.Revision(proposal.Version))))
		if err != nil {
			log.Warnf("Failed to delete proposal %s: %s", proposal.ID, err)
			return errors.FromAtomix(err)
		}
		proposal.Version = 0
	}
	return nil
}

func (s *proposalStore) List(ctx context.Context) ([]*configapi.Proposal, error) {
	log.Debugf("Listing proposals")
	mapCh := make(chan _map.Entry)
	if err := s.proposals.Entries(ctx, mapCh); err != nil {
		return nil, errors.FromAtomix(err)
	}

	proposals := make([]*configapi.Proposal, 0)

	for entry := range mapCh {
		if proposal, err := decodeProposal(entry); err == nil {
			proposals = append(proposals, proposal)
		}
	}
	return proposals, nil
}

func (s *proposalStore) Watch(ctx context.Context, ch chan<- configapi.ProposalEvent, opts ...WatchOption) error {
	watchOpts := make([]_map.WatchOption, 0)
	for _, opt := range opts {
		watchOpts = opt.apply(watchOpts)
	}

	mapCh := make(chan _map.Event)
	if err := s.proposals.Watch(ctx, mapCh, watchOpts...); err != nil {
		return errors.FromAtomix(err)
	}

	go func() {
		defer close(ch)
		for event := range mapCh {
			if proposal, err := decodeProposal(event.Entry); err == nil {
				var eventType configapi.ProposalEvent_EventType
				switch event.Type {
				case _map.EventReplay:
					eventType = configapi.ProposalEvent_REPLAYED
				case _map.EventInsert:
					eventType = configapi.ProposalEvent_CREATED
				case _map.EventRemove:
					eventType = configapi.ProposalEvent_DELETED
				case _map.EventUpdate:
					eventType = configapi.ProposalEvent_UPDATED
				default:
					eventType = configapi.ProposalEvent_UPDATED
				}
				ch <- configapi.ProposalEvent{
					Type:          eventType,
					Proposal: *proposal,
				}
			}
		}
	}()
	return nil
}

func (s *proposalStore) Close(ctx context.Context) error {
	err := s.proposals.Close(ctx)
	if err != nil {
		return errors.FromAtomix(err)
	}
	return nil
}

func decodeProposal(entry _map.Entry) (*configapi.Proposal, error) {
	proposal := &configapi.Proposal{}
	if err := proto.Unmarshal(entry.Value, proposal); err != nil {
		return nil, errors.NewInvalid("proposal decoding failed: %v", err)
	}
	proposal.ID = configapi.ProposalID(entry.Key)
	proposal.Key = entry.Key
	proposal.Version = uint64(entry.Revision)
	return proposal, nil
}
