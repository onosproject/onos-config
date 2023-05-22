// SPDX-FileCopyrightText: 2020-present Open Networking Foundation <info@opennetworking.org>
//
// SPDX-License-Identifier: Apache-2.0

package proposal

import (
	"fmt"
	"github.com/atomix/go-sdk/pkg/primitive"
	"github.com/atomix/go-sdk/pkg/primitive/indexedmap"
	_map "github.com/atomix/go-sdk/pkg/primitive/map"
	"github.com/atomix/go-sdk/pkg/types"
	"github.com/google/uuid"
	"io"
	"sync"
	"time"

	"github.com/onosproject/onos-lib-go/pkg/errors"

	"golang.org/x/net/context"

	configapi "github.com/onosproject/onos-api/go/onos/config/v2"

	"github.com/onosproject/onos-lib-go/pkg/logging"
)

var log = logging.GetLogger()

// Store proposal store interface
type Store interface {
	// Get gets a proposal
	Get(ctx context.Context, id configapi.ProposalID) (*configapi.Proposal, error)

	// GetKey gets a proposal by its key
	GetKey(ctx context.Context, target configapi.Target, key string) (*configapi.Proposal, error)

	// Create creates a new proposal
	Create(ctx context.Context, proposal *configapi.Proposal) error

	// Update updates an existing proposal
	Update(ctx context.Context, proposal *configapi.Proposal) error

	// UpdateStatus updates the status of an existing proposal
	UpdateStatus(ctx context.Context, proposal *configapi.Proposal) error

	// List lists proposals
	List(ctx context.Context) ([]*configapi.Proposal, error)

	// Watch watches the proposal store  for changes
	Watch(ctx context.Context, ch chan<- configapi.ProposalEvent, opts ...WatchOption) error

	// Close closes the proposal store
	Close(ctx context.Context) error
}

// NewAtomixStore returns a new persistent Store
func NewAtomixStore(client primitive.Client) (Store, error) {
	targets, err := _map.NewBuilder[string, *configapi.Target](client, "proposal-targets").
		Tag("onos-config", "proposal").
		Codec(types.Proto[*configapi.Target](&configapi.Target{})).
		Get(context.Background())
	if err != nil {
		return nil, errors.FromAtomix(err)
	}
	store := &proposalStore{
		client:     client,
		targets:    targets,
		watchers:   make(map[uuid.UUID]chan<- configapi.ProposalEvent),
		idWatchers: make(map[configapi.ProposalID]map[uuid.UUID]chan<- configapi.ProposalEvent),
	}
	if err := store.open(); err != nil {
		return nil, err
	}
	return store, nil
}

type WatchOptions struct {
	ProposalID configapi.ProposalID
	Replay     bool
}

// WatchOption is a configuration option for Watch calls
type WatchOption interface {
	apply(*WatchOptions)
}

func newWatchOption(f func(*WatchOptions)) WatchOption {
	return funcWatchOption{
		f: f,
	}
}

type funcWatchOption struct {
	f func(*WatchOptions)
}

func (o funcWatchOption) apply(options *WatchOptions) {
	o.f(options)
}

// WithReplay returns a WatchOption that replays past changes
func WithReplay() WatchOption {
	return newWatchOption(func(options *WatchOptions) {
		options.Replay = true
	})
}

// WithProposalID returns a Watch option that watches for proposals based on a  given proposal ID
func WithProposalID(id configapi.ProposalID) WatchOption {
	return newWatchOption(func(options *WatchOptions) {
		options.ProposalID = id
	})
}

type proposalStore struct {
	client     primitive.Client
	targets    _map.Map[string, *configapi.Target]
	logs       sync.Map
	watchers   map[uuid.UUID]chan<- configapi.ProposalEvent
	idWatchers map[configapi.ProposalID]map[uuid.UUID]chan<- configapi.ProposalEvent
	mu         sync.RWMutex
}

func (s *proposalStore) open() error {
	events, err := s.targets.Watch(context.Background())
	if err != nil {
		return err
	}

	go func() {
		for {
			event, err := events.Next()
			if err == io.EOF {
				break
			}
			if err != nil {
				log.Error(err)
				continue
			}

			_, err = s.newProposals(context.Background(), *event.Value)
			if err != nil {
				log.Error(err)
			}
		}
	}()

	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()

	entries, err := s.targets.List(context.Background())
	if err != nil {
		return errors.FromAtomix(err)
	}

	wg := &sync.WaitGroup{}
	errCh := make(chan error)
	for {
		entry, err := entries.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}

		wg.Add(1)
		go func() {
			defer wg.Done()
			_, err := s.newProposals(ctx, *entry.Value)
			if err != nil {
				errCh <- err
				return
			}
		}()
	}

	go func() {
		wg.Wait()
		close(errCh)
	}()
	return <-errCh
}

func (s *proposalStore) getProposals(ctx context.Context, target configapi.Target) (indexedmap.IndexedMap[string, *configapi.Proposal], error) {
	log, ok := s.logs.Load(target)
	if ok {
		return log.(indexedmap.IndexedMap[string, *configapi.Proposal]), nil
	}

	key := fmt.Sprintf("%s-%s-%s", target.ID, target.Type, target.Version)
	_, err := s.targets.Put(ctx, key, &target)
	if err != nil {
		err = errors.FromAtomix(err)
		if !errors.IsAlreadyExists(err) {
			return nil, err
		}
	}
	return s.newProposals(ctx, target)
}

func (s *proposalStore) newProposals(ctx context.Context, target configapi.Target) (indexedmap.IndexedMap[string, *configapi.Proposal], error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	log, ok := s.logs.Load(target)
	if ok {
		return log.(indexedmap.IndexedMap[string, *configapi.Proposal]), nil
	}

	name := fmt.Sprintf("proposals-%s-%s-%s", target.ID, target.Type, target.Version)
	proposals, err := indexedmap.NewBuilder[string, *configapi.Proposal](s.client, name).
		Tag("onos-config", "proposal").
		Codec(types.Proto[*configapi.Proposal](&configapi.Proposal{})).
		Get(ctx)
	if err != nil {
		return nil, errors.FromAtomix(err)
	}

	if err := s.watch(target, proposals); err != nil {
		return nil, err
	}

	s.logs.Store(target, proposals)
	return proposals, nil
}

func (s *proposalStore) watch(target configapi.Target, proposals indexedmap.IndexedMap[string, *configapi.Proposal]) error {
	events, err := proposals.Events(context.Background())
	if err != nil {
		return errors.FromAtomix(err)
	}
	go func() {
		for {
			event, err := events.Next()
			if err == io.EOF {
				break
			}
			if err != nil {
				log.Error(err)
				continue
			}

			var proposalEvent configapi.ProposalEvent
			switch e := event.(type) {
			case *indexedmap.Inserted[string, *configapi.Proposal]:
				proposal := e.Entry.Value
				proposal.ID = configapi.ProposalID{
					Target: target,
					Index:  configapi.Index(e.Entry.Index),
				}
				proposal.Version = uint64(e.Entry.Version)
				proposalEvent = configapi.ProposalEvent{
					Type:     configapi.ProposalEvent_CREATED,
					Proposal: *proposal,
				}
			case *indexedmap.Updated[string, *configapi.Proposal]:
				proposal := e.Entry.Value
				proposal.ID = configapi.ProposalID{
					Target: target,
					Index:  configapi.Index(e.Entry.Index),
				}
				proposal.Version = uint64(e.Entry.Version)
				proposalEvent = configapi.ProposalEvent{
					Type:     configapi.ProposalEvent_UPDATED,
					Proposal: *proposal,
				}
			case *indexedmap.Removed[string, *configapi.Proposal]:
				proposal := e.Entry.Value
				proposal.ID = configapi.ProposalID{
					Target: target,
					Index:  configapi.Index(e.Entry.Index),
				}
				proposal.Version = uint64(e.Entry.Version)
				proposalEvent = configapi.ProposalEvent{
					Type:     configapi.ProposalEvent_DELETED,
					Proposal: *proposal,
				}
			}

			var watchers []chan<- configapi.ProposalEvent
			s.mu.RLock()
			for _, ch := range s.watchers {
				watchers = append(watchers, ch)
			}
			idWatchers, ok := s.idWatchers[proposalEvent.Proposal.ID]
			if ok {
				for _, ch := range idWatchers {
					watchers = append(watchers, ch)
				}
			}
			s.mu.RUnlock()

			for _, ch := range watchers {
				ch <- proposalEvent
			}
		}
	}()
	return nil
}

// Get gets a proposal
func (s *proposalStore) Get(ctx context.Context, proposalID configapi.ProposalID) (*configapi.Proposal, error) {
	// If the proposal is not already in the cache, get it from the underlying primitive.
	proposals, err := s.getProposals(ctx, proposalID.Target)
	if err != nil {
		return nil, err
	}
	entry, err := proposals.GetIndex(ctx, indexedmap.Index(proposalID.Index))
	if err != nil {
		return nil, errors.FromAtomix(err)
	}
	proposal := entry.Value
	proposal.ID.Index = configapi.Index(entry.Index)
	proposal.Version = uint64(entry.Version)
	return proposal, nil
}

// GetKey gets a proposal by its key
func (s *proposalStore) GetKey(ctx context.Context, target configapi.Target, key string) (*configapi.Proposal, error) {
	// If the proposal is not already in the cache, get it from the underlying primitive.
	proposals, err := s.getProposals(ctx, target)
	if err != nil {
		return nil, err
	}
	entry, err := proposals.Get(ctx, key)
	if err != nil {
		return nil, errors.FromAtomix(err)
	}
	proposal := entry.Value
	proposal.ID.Index = configapi.Index(entry.Index)
	proposal.Version = uint64(entry.Version)
	return proposal, nil
}

// Create creates a new proposal
func (s *proposalStore) Create(ctx context.Context, proposal *configapi.Proposal) error {
	if proposal.Key == "" {
		proposal.Key = configapi.NewUUID().String()
	}
	if proposal.ID.Target.ID == "" {
		return errors.NewInvalid("proposal target ID is required")
	}
	if proposal.ID.Target.Type == "" {
		return errors.NewInvalid("proposal target Type is required")
	}
	if proposal.ID.Target.Version == "" {
		return errors.NewInvalid("proposal target version is required")
	}
	if proposal.Version != 0 {
		return errors.NewInvalid("not a new object")
	}
	if proposal.Revision != 0 {
		return errors.NewInvalid("not a new object")
	}
	proposal.Revision = 1
	proposal.Created = time.Now()
	proposal.Updated = time.Now()

	// Append a new entry to the proposal log.
	proposals, err := s.getProposals(ctx, proposal.ID.Target)
	if err != nil {
		return err
	}
	entry, err := proposals.Append(ctx, proposal.Key, proposal)
	if err != nil {
		return errors.FromAtomix(err)
	}
	proposal.ID.Index = configapi.Index(entry.Index)
	proposal.Version = uint64(entry.Version)
	return nil
}

// Update updates an existing proposal
func (s *proposalStore) Update(ctx context.Context, proposal *configapi.Proposal) error {
	if proposal.Key == "" {
		return errors.NewInvalid("proposal key is required")
	}
	if proposal.ID.Target.ID == "" {
		return errors.NewInvalid("proposal target ID is required")
	}
	if proposal.ID.Target.Type == "" {
		return errors.NewInvalid("proposal target Type is required")
	}
	if proposal.ID.Target.Version == "" {
		return errors.NewInvalid("proposal target version is required")
	}
	if proposal.Revision == 0 {
		return errors.NewInvalid("proposal must contain a revision on update")
	}
	if proposal.Version == 0 {
		return errors.NewInvalid("proposal must contain a version on update")
	}
	proposal.Revision++
	proposal.Updated = time.Now()

	// Update the entry in the proposal log.
	proposals, err := s.getProposals(ctx, proposal.ID.Target)
	if err != nil {
		return err
	}
	entry, err := proposals.Update(ctx, proposal.Key, proposal, indexedmap.IfVersion(primitive.Version(proposal.Version)))
	if err != nil {
		return errors.FromAtomix(err)
	}
	proposal.ID.Index = configapi.Index(entry.Index)
	proposal.Version = uint64(entry.Version)
	return nil
}

// UpdateStatus updates an existing proposal status
func (s *proposalStore) UpdateStatus(ctx context.Context, proposal *configapi.Proposal) error {
	if proposal.Revision == 0 {
		return errors.NewInvalid("proposal must contain a revision on update")
	}
	if proposal.Version == 0 {
		return errors.NewInvalid("proposal must contain a version on update")
	}
	proposal.Updated = time.Now()

	// Update the entry in the proposal log.
	proposals, err := s.getProposals(ctx, proposal.ID.Target)
	if err != nil {
		return err
	}
	entry, err := proposals.Update(ctx, proposal.Key, proposal, indexedmap.IfVersion(primitive.Version(proposal.Version)))
	if err != nil {
		return errors.FromAtomix(err)
	}
	proposal.ID.Index = configapi.Index(entry.Index)
	proposal.Version = uint64(entry.Version)
	return nil
}

// List lists proposals
func (s *proposalStore) List(ctx context.Context) ([]*configapi.Proposal, error) {
	targets, err := s.targets.List(context.Background())
	if err != nil {
		return nil, errors.FromAtomix(err)
	}

	var proposals []*configapi.Proposal
	for {
		entry, err := targets.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, err
		}

		log, err := s.getProposals(ctx, *entry.Value)
		if err != nil {
			return nil, err
		}

		stream, err := log.List(ctx)
		if err != nil {
			return nil, errors.FromAtomix(err)
		}

		for {
			entry, err := stream.Next()
			if err == io.EOF {
				return proposals, nil
			}
			if err != nil {
				return nil, err
			}
			proposal := entry.Value
			proposal.Version = uint64(entry.Version)
			proposal.ID.Index = configapi.Index(entry.Index)
			proposals = append(proposals, proposal)
		}
	}
	return proposals, nil
}

func (s *proposalStore) Watch(ctx context.Context, ch chan<- configapi.ProposalEvent, opts ...WatchOption) error {
	var options WatchOptions
	for _, opt := range opts {
		opt.apply(&options)
	}

	id := uuid.New()
	eventCh := make(chan configapi.ProposalEvent)
	s.mu.Lock()
	if options.ProposalID.Index > 0 {
		watchers, ok := s.idWatchers[options.ProposalID]
		if !ok {
			watchers = make(map[uuid.UUID]chan<- configapi.ProposalEvent)
			s.idWatchers[options.ProposalID] = watchers
		}
		watchers[id] = eventCh
	} else {
		s.watchers[id] = eventCh
	}
	s.mu.Unlock()

	go func() {
		defer func() {
			s.mu.Lock()
			if options.ProposalID.Index > 0 {
				watchers, ok := s.idWatchers[options.ProposalID]
				if ok {
					delete(watchers, id)
					if len(watchers) == 0 {
						delete(s.idWatchers, options.ProposalID)
					}
				}
			} else {
				delete(s.watchers, id)
			}
			s.mu.Unlock()
		}()

		defer close(ch)

		if options.Replay {
			if options.ProposalID.Index > 0 {
				proposals, err := s.getProposals(ctx, options.ProposalID.Target)
				if err != nil {
					log.Error(err)
					return
				}

				entry, err := proposals.GetIndex(ctx, indexedmap.Index(options.ProposalID.Index))
				if err != nil {
					err = errors.FromAtomix(err)
					if !errors.IsNotFound(err) {
						log.Error(err)
					}
				} else {
					proposal := entry.Value
					proposal.ID.Index = configapi.Index(entry.Index)
					proposal.Version = uint64(entry.Version)
					if ctx.Err() != nil {
						return
					}
					ch <- configapi.ProposalEvent{
						Type:     configapi.ProposalEvent_REPLAYED,
						Proposal: *proposal,
					}
				}
			} else {
				targets, err := s.targets.List(context.Background())
				if err != nil {
					log.Error(err)
					return
				}

				for {
					entry, err := targets.Next()
					if err == io.EOF {
						break
					}
					if err != nil {
						log.Error(err)
						continue
					}

					proposals, err := s.getProposals(ctx, *entry.Value)
					if err != nil {
						log.Error(err)
						close(ch)
						return
					}

					stream, err := proposals.List(ctx)
					if err != nil {
						log.Error(err)
						return
					}

					for {
						entry, err := stream.Next()
						if err == io.EOF {
							break
						}
						if err != nil {
							log.Error(err)
							continue
						}
						proposal := entry.Value
						proposal.Version = uint64(entry.Version)
						proposal.ID.Index = configapi.Index(entry.Index)
						ch <- configapi.ProposalEvent{
							Type:     configapi.ProposalEvent_REPLAYED,
							Proposal: *proposal,
						}
					}
				}
			}
		}

		for {
			select {
			case event := <-eventCh:
				ch <- event
			case <-ctx.Done():
				close(ch)
				go func() {
					for range eventCh {
					}
				}()
				return
			}
		}
	}()
	return nil
}

// Close closes the store
func (s *proposalStore) Close(ctx context.Context) error {
	wg := &sync.WaitGroup{}
	s.logs.Range(func(key, value any) bool {
		wg.Add(1)
		go func() {
			_ = value.(indexedmap.IndexedMap[string, *configapi.Proposal]).Close(ctx)
		}()
		return true
	})
	wg.Wait()
	_ = s.targets.Close(ctx)
	return nil
}

var _ Store = &proposalStore{}
