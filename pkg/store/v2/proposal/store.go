// SPDX-FileCopyrightText: 2020-present Open Networking Foundation <info@opennetworking.org>
//
// SPDX-License-Identifier: Apache-2.0

package proposal

import (
	"context"
	"fmt"
	"github.com/atomix/go-sdk/pkg/primitive"
	"github.com/atomix/go-sdk/pkg/types"
	"io"
	"time"

	configapi "github.com/onosproject/onos-api/go/onos/config/v2"

	"github.com/onosproject/onos-lib-go/pkg/logging"

	_map "github.com/atomix/go-sdk/pkg/primitive/map"
	"github.com/onosproject/onos-lib-go/pkg/errors"
)

var log = logging.GetLogger()

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
func NewAtomixStore(client primitive.Client) (Store, error) {
	proposals, err := _map.NewBuilder[configapi.ProposalID, *configapi.Proposal](client, "proposals").
		Tag("onos-config", "proposal").
		Codec(types.Proto[*configapi.Proposal](&configapi.Proposal{})).
		Get(context.Background())
	if err != nil {
		return nil, errors.FromAtomix(err)
	}
	return &proposalStore{
		proposals: proposals,
	}, nil
}

type proposalStore struct {
	proposals _map.Map[configapi.ProposalID, *configapi.Proposal]
}

func (s *proposalStore) Get(ctx context.Context, id configapi.ProposalID) (*configapi.Proposal, error) {
	// If the proposal is not already in the cache, get it from the underlying primitive.
	entry, err := s.proposals.Get(ctx, id)
	if err != nil {
		return nil, errors.FromAtomix(err)
	}
	proposal := entry.Value
	proposal.Version = uint64(entry.Version)
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

	// Create the entry in the underlying map primitive.
	entry, err := s.proposals.Insert(ctx, proposal.ID, proposal)
	if err != nil {
		return errors.FromAtomix(err)
	}
	proposal.Version = uint64(entry.Version)
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

	// Update the entry in the underlying map primitive using the proposal version
	// as an optimistic lock.
	entry, err := s.proposals.Update(ctx, proposal.ID, proposal, _map.IfVersion(primitive.Version(proposal.Version)))
	if err != nil {
		return errors.FromAtomix(err)
	}
	proposal.Version = uint64(entry.Version)
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

	// Update the entry in the underlying map primitive using the proposal version
	// as an optimistic lock.
	entry, err := s.proposals.Update(ctx, proposal.ID, proposal, _map.IfVersion(primitive.Version(proposal.Version)))
	if err != nil {
		return errors.FromAtomix(err)
	}
	proposal.Version = uint64(entry.Version)
	return nil
}

func (s *proposalStore) List(ctx context.Context) ([]*configapi.Proposal, error) {
	log.Debugf("Listing proposals")
	stream, err := s.proposals.List(ctx)
	if err != nil {
		return nil, errors.FromAtomix(err)
	}

	var proposals []*configapi.Proposal
	for {
		entry, err := stream.Next()
		if err == io.EOF {
			return proposals, nil
		}
		if err != nil {
			log.Error(err)
			return nil, err
		}
		proposal := entry.Value
		proposal.Version = uint64(entry.Version)
		proposals = append(proposals, proposal)
	}
}

func (s *proposalStore) Watch(ctx context.Context, ch chan<- configapi.ProposalEvent, opts ...WatchOption) error {
	var options watchOptions
	for _, opt := range opts {
		opt.apply(&options)
	}

	var eventsOpts []_map.EventsOption
	if options.proposalID != "" {
		eventsOpts = append(eventsOpts, _map.WithKey[configapi.ProposalID](options.proposalID))
	}
	events, err := s.proposals.Events(ctx, eventsOpts...)
	if err != nil {
		return errors.FromAtomix(err)
	}

	if options.replay {
		if options.proposalID != "" {
			entry, err := s.proposals.Get(ctx, options.proposalID)
			if err != nil {
				err = errors.FromAtomix(err)
				if !errors.IsNotFound(err) {
					return err
				}
				go propagateEvents(events, ch)
			} else {
				go func() {
					proposal := entry.Value
					proposal.Version = uint64(entry.Version)
					ch <- configapi.ProposalEvent{
						Type:     configapi.ProposalEvent_REPLAYED,
						Proposal: *proposal,
					}
					propagateEvents(events, ch)
				}()
			}
		} else {
			entries, err := s.proposals.List(ctx)
			if err != nil {
				return errors.FromAtomix(err)
			}
			go func() {
				for {
					entry, err := entries.Next()
					if err == io.EOF {
						break
					}
					if err != nil {
						log.Error(err)
						continue
					}
					proposal := entry.Value
					proposal.Version = uint64(entry.Version)
					ch <- configapi.ProposalEvent{
						Type:     configapi.ProposalEvent_REPLAYED,
						Proposal: *proposal,
					}
				}
				propagateEvents(events, ch)
			}()
		}
	} else {
		go propagateEvents(events, ch)
	}
	return nil
}

func propagateEvents(events _map.EventStream[configapi.ProposalID, *configapi.Proposal], ch chan<- configapi.ProposalEvent) {
	for {
		event, err := events.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			log.Error(err)
			continue
		}
		switch e := event.(type) {
		case *_map.Inserted[configapi.ProposalID, *configapi.Proposal]:
			proposal := e.Entry.Value
			proposal.Version = uint64(e.Entry.Version)
			ch <- configapi.ProposalEvent{
				Type:     configapi.ProposalEvent_CREATED,
				Proposal: *proposal,
			}
		case *_map.Updated[configapi.ProposalID, *configapi.Proposal]:
			proposal := e.Entry.Value
			proposal.Version = uint64(e.Entry.Version)
			ch <- configapi.ProposalEvent{
				Type:     configapi.ProposalEvent_UPDATED,
				Proposal: *proposal,
			}
		case *_map.Removed[configapi.ProposalID, *configapi.Proposal]:
			proposal := e.Entry.Value
			proposal.Version = uint64(e.Entry.Version)
			ch <- configapi.ProposalEvent{
				Type:     configapi.ProposalEvent_DELETED,
				Proposal: *proposal,
			}
		}
	}
}

func (s *proposalStore) Close(ctx context.Context) error {
	err := s.proposals.Close(ctx)
	if err != nil {
		return errors.FromAtomix(err)
	}
	return nil
}
