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
	"github.com/google/uuid"
	"sync"
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

	// List lists all the proposal
	List(ctx context.Context) ([]*configapi.Proposal, error)

	// Watch watches proposal changes
	Watch(ctx context.Context, ch chan<- configapi.ProposalEvent, opts ...WatchOption) error

	// UpdateStatus updates a proposal status
	UpdateStatus(ctx context.Context, proposal *configapi.Proposal) error

	Close(ctx context.Context) error
}

type watchOptions struct {
	proposalID configapi.ProposalID
	replay     bool
}

// WatchOption is a proposal option for Watch calls
type WatchOption interface {
	apply(*watchOptions)
}

// watchReplyOption is an option to replay events on watch
type watchReplayOption struct {
}

func (o watchReplayOption) apply(options *watchOptions) {
	options.replay = true
}

// WithReplay returns a WatchOption that replays past changes
func WithReplay() WatchOption {
	return watchReplayOption{}
}

type watchIDOption struct {
	id configapi.ProposalID
}

func (o watchIDOption) apply(options *watchOptions) {
	options.proposalID = o.id
}

// WithProposalID returns a Watch option that watches for proposals based on a given proposal ID
func WithProposalID(id configapi.ProposalID) WatchOption {
	return watchIDOption{id: id}
}

// NewAtomixStore returns a new persistent Store
func NewAtomixStore(client atomix.Client) (Store, error) {
	proposals, err := client.GetMap(context.Background(), "onos-config-proposals")
	if err != nil {
		return nil, errors.FromAtomix(err)
	}
	store := &proposalStore{
		proposals: proposals,
		cache:     make(map[configapi.ProposalID]configapi.Proposal),
		watchers:  make(map[uuid.UUID]chan<- configapi.ProposalEvent),
		eventCh:   make(chan configapi.ProposalEvent, 1000),
	}
	if err := store.open(context.Background()); err != nil {
		return nil, err
	}
	return store, nil
}

type proposalStore struct {
	proposals  _map.Map
	cache      map[configapi.ProposalID]configapi.Proposal
	cacheMu    sync.RWMutex
	watchers   map[uuid.UUID]chan<- configapi.ProposalEvent
	watchersMu sync.RWMutex
	eventCh    chan configapi.ProposalEvent
}

func (s *proposalStore) open(ctx context.Context) error {
	ch := make(chan _map.Event)
	if err := s.proposals.Watch(ctx, ch, _map.WithReplay()); err != nil {
		return err
	}
	go func() {
		for event := range ch {
			var proposal configapi.Proposal
			if err := decodeProposal(&event.Entry, &proposal); err != nil {
				log.Error(err)
				continue
			}
			s.updateCache(proposal)
		}
	}()
	go s.processEvents()
	return nil
}

func (s *proposalStore) publishEvent(event configapi.ProposalEvent) {
	s.eventCh <- event
}

func (s *proposalStore) processEvents() {
	for event := range s.eventCh {
		s.watchersMu.RLock()
		for _, watcher := range s.watchers {
			watcher <- event
		}
		s.watchersMu.RUnlock()
	}
}

func (s *proposalStore) updateCache(proposal configapi.Proposal) {
	// Use a double-checked lock when updating the cache.
	// First, check for a more recent version of the proposal already in the cache.
	s.cacheMu.RLock()
	entry, ok := s.cache[proposal.ID]
	s.cacheMu.RUnlock()
	if ok && entry.Version >= proposal.Version {
		return
	}

	// The cache needs to be updated. Acquire a write lock and check once again
	// for a more recent version of the proposal.
	s.cacheMu.Lock()
	defer s.cacheMu.Unlock()
	entry, ok = s.cache[proposal.ID]
	if !ok {
		s.cache[proposal.ID] = proposal
		s.publishEvent(configapi.ProposalEvent{
			Type:     configapi.ProposalEvent_CREATED,
			Proposal: proposal,
		})
	} else if proposal.Version > entry.Version {
		// Add the proposal to the ID and index caches and publish an event.
		s.cache[proposal.ID] = proposal
		s.publishEvent(configapi.ProposalEvent{
			Type:     configapi.ProposalEvent_UPDATED,
			Proposal: proposal,
		})
	}
}

func (s *proposalStore) Get(ctx context.Context, id configapi.ProposalID) (*configapi.Proposal, error) {
	// Check the ID cache for the latest version of the proposal.
	s.cacheMu.RLock()
	cached, ok := s.cache[id]
	s.cacheMu.RUnlock()
	if ok {
		proposal := cached
		return &proposal, nil
	}

	// If the proposal is not already in the cache, get it from the underlying primitive.
	entry, err := s.proposals.Get(ctx, string(id))
	if err != nil {
		return nil, errors.FromAtomix(err)
	}

	// Decode the proposal bytes.
	proposal := &configapi.Proposal{}
	if err := decodeProposal(entry, proposal); err != nil {
		return nil, errors.NewInvalid("proposal decoding failed: %v", err)
	}

	// Update the cache before returning the proposal.
	s.updateCache(*proposal)
	return proposal, nil
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

	// Encode the proposal bytes.
	bytes, err := proto.Marshal(proposal)
	if err != nil {
		return errors.NewInvalid("proposal encoding failed: %v", err)
	}

	// Create the entry in the underlying map primitive.
	entry, err := s.proposals.Put(ctx, string(proposal.ID), bytes, _map.IfNotSet())
	if err != nil {
		return errors.FromAtomix(err)
	}

	// Decode the proposal from the returned entry bytes.
	if err := decodeProposal(entry, proposal); err != nil {
		return errors.NewInvalid("proposal decoding failed: %v", err)
	}

	// Update the cache.
	s.updateCache(*proposal)
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

	// Encode the proposal bytes.
	bytes, err := proto.Marshal(proposal)
	if err != nil {
		return errors.NewInvalid("proposal encoding failed: %v", err)
	}

	// Update the entry in the underlying map primitive using the proposal version
	// as an optimistic lock.
	entry, err := s.proposals.Put(ctx, string(proposal.ID), bytes, _map.IfMatch(meta.NewRevision(meta.Revision(proposal.Version))))
	if err != nil {
		return errors.FromAtomix(err)
	}

	// Decode the proposal from the returned entry bytes.
	if err := decodeProposal(entry, proposal); err != nil {
		return errors.NewInvalid("proposal decoding failed: %v", err)
	}

	// Update the cache.
	s.updateCache(*proposal)
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

	// Encode the proposal bytes.
	bytes, err := proto.Marshal(proposal)
	if err != nil {
		return errors.NewInvalid("proposal encoding failed: %v", err)
	}

	// Update the entry in the underlying map primitive using the proposal version
	// as an optimistic lock.
	entry, err := s.proposals.Put(ctx, string(proposal.ID), bytes, _map.IfMatch(meta.NewRevision(meta.Revision(proposal.Version))))
	if err != nil {
		return errors.FromAtomix(err)
	}

	// Decode the proposal from the returned entry bytes.
	if err := decodeProposal(entry, proposal); err != nil {
		return errors.NewInvalid("proposal decoding failed: %v", err)
	}

	// Update the cache.
	s.updateCache(*proposal)
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
		proposal := &configapi.Proposal{}
		if err := decodeProposal(&entry, proposal); err != nil {
			log.Error(err)
		} else {
			proposals = append(proposals, proposal)
		}
	}
	return proposals, nil
}

func (s *proposalStore) Watch(ctx context.Context, ch chan<- configapi.ProposalEvent, opts ...WatchOption) error {
	var options watchOptions
	for _, opt := range opts {
		opt.apply(&options)
	}

	watchCh := make(chan configapi.ProposalEvent, 10)
	id := uuid.New()
	s.watchersMu.Lock()
	s.watchers[id] = watchCh
	s.watchersMu.Unlock()

	var replay []configapi.ProposalEvent
	if options.replay {
		if options.proposalID == "" {
			s.cacheMu.RLock()
			replay = make([]configapi.ProposalEvent, 0, len(s.cache))
			for _, proposal := range s.cache {
				replay = append(replay, configapi.ProposalEvent{
					Type:     configapi.ProposalEvent_REPLAYED,
					Proposal: proposal,
				})
			}
			s.cacheMu.RUnlock()
		} else {
			s.cacheMu.RLock()
			proposal, ok := s.cache[options.proposalID]
			if ok {
				replay = []configapi.ProposalEvent{
					{
						Type:     configapi.ProposalEvent_REPLAYED,
						Proposal: proposal,
					},
				}
			}
			s.cacheMu.RUnlock()
		}
	}

	go func() {
		defer close(ch)
		for _, event := range replay {
			ch <- event
		}
		for event := range watchCh {
			if options.proposalID == "" || event.Proposal.ID == options.proposalID {
				ch <- event
			}
		}
	}()

	go func() {
		<-ctx.Done()
		s.watchersMu.Lock()
		delete(s.watchers, id)
		s.watchersMu.Unlock()
		close(watchCh)
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

func decodeProposal(entry *_map.Entry, proposal *configapi.Proposal) error {
	if err := proto.Unmarshal(entry.Value, proposal); err != nil {
		return err
	}
	proposal.ID = configapi.ProposalID(entry.Key)
	proposal.Key = entry.Key
	proposal.Version = uint64(entry.Revision)
	return nil
}
