// SPDX-FileCopyrightText: 2020-present Open Networking Foundation <info@opennetworking.org>
//
// SPDX-License-Identifier: Apache-2.0

package configuration

import (
	"context"
	"fmt"
	"github.com/atomix/go-sdk/pkg/primitive"
	_map "github.com/atomix/go-sdk/pkg/primitive/map"
	"github.com/atomix/go-sdk/pkg/types"
	"github.com/google/uuid"
	"github.com/onosproject/onos-config/pkg/utils/v2/tree"
	"io"
	"sync"
	"time"

	configapi "github.com/onosproject/onos-api/go/onos/config/v2"

	"github.com/onosproject/onos-lib-go/pkg/logging"

	"github.com/onosproject/onos-lib-go/pkg/errors"
)

var log = logging.GetLogger()

// NewID returns a new Configuration ID for the given target/type/version
func NewID(targetID configapi.TargetID, targetType configapi.TargetType, targetVersion configapi.TargetVersion) configapi.ConfigurationID {
	return configapi.ConfigurationID(fmt.Sprintf("%s-%s-%s", targetID, targetType, targetVersion))
}

// Store configuration store interface
type Store interface {
	// Get gets the configuration intended for a given target ID
	Get(ctx context.Context, id configapi.ConfigurationID) (*configapi.Configuration, error)

	// Create creates a configuration
	Create(ctx context.Context, configuration *configapi.Configuration) error

	// Update updates a configuration
	Update(ctx context.Context, configuration *configapi.Configuration) error

	// List lists all the configuration
	List(ctx context.Context) ([]*configapi.Configuration, error)

	// Watch watches configuration changes
	Watch(ctx context.Context, ch chan<- configapi.ConfigurationEvent, opts ...WatchOption) error

	// UpdateStatus updates a configuration status
	UpdateStatus(ctx context.Context, configuration *configapi.Configuration) error

	Close(ctx context.Context) error
}

// NewAtomixStore returns a new persistent Store
func NewAtomixStore(client primitive.Client) (Store, error) {
	configurations, err := _map.NewBuilder[configapi.ConfigurationID, *configapi.Configuration](client, "configurations").
		Tag("onos-config", "configuration").
		Codec(types.Proto[*configapi.Configuration](&configapi.Configuration{})).
		Get(context.Background())
	if err != nil {
		return nil, errors.FromAtomix(err)
	}
	store := &configurationStore{
		client:         client,
		configurations: configurations,
		committed:      make(map[configapi.ConfigurationID]_map.Map[string, *configapi.PathValue]),
		applied:        make(map[configapi.ConfigurationID]_map.Map[string, *configapi.PathValue]),
		watchers:       make(map[uuid.UUID]chan<- configapi.ConfigurationEvent),
		idWatchers:     make(map[configapi.ConfigurationID]map[uuid.UUID]chan<- configapi.ConfigurationEvent),
	}
	if err := store.open(); err != nil {
		return nil, err
	}
	return store, nil
}

type watchOptions struct {
	configurationID configapi.ConfigurationID
	replay          bool
}

// WatchOption is a configuration option for Watch calls
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
	id configapi.ConfigurationID
}

func (o watchIDOption) apply(options *watchOptions) {
	options.configurationID = o.id
}

// WithConfigurationID returns a Watch option that watches for configurations based on a given configuration ID
func WithConfigurationID(id configapi.ConfigurationID) WatchOption {
	return watchIDOption{id: id}
}

type configurationStore struct {
	configurations _map.Map[configapi.ConfigurationID, *configapi.Configuration]
	client         primitive.Client
	committed      map[configapi.ConfigurationID]_map.Map[string, *configapi.PathValue]
	applied        map[configapi.ConfigurationID]_map.Map[string, *configapi.PathValue]
	watchers       map[uuid.UUID]chan<- configapi.ConfigurationEvent
	idWatchers     map[configapi.ConfigurationID]map[uuid.UUID]chan<- configapi.ConfigurationEvent
	mu             sync.RWMutex
}

func (s *configurationStore) open() error {
	events, err := s.configurations.Events(context.Background())
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

			var configurationEvent configapi.ConfigurationEvent
			switch e := event.(type) {
			case *_map.Inserted[configapi.ConfigurationID, *configapi.Configuration]:
				configuration := e.Entry.Value
				configuration.Version = uint64(e.Entry.Version)
				if err := s.populate(context.Background(), configuration); err != nil {
					log.Error(err)
					continue
				}
				configurationEvent = configapi.ConfigurationEvent{
					Type:          configapi.ConfigurationEvent_CREATED,
					Configuration: *configuration,
				}
			case *_map.Updated[configapi.ConfigurationID, *configapi.Configuration]:
				configuration := e.Entry.Value
				configuration.Version = uint64(e.Entry.Version)
				if err := s.populate(context.Background(), configuration); err != nil {
					log.Error(err)
					continue
				}
				configurationEvent = configapi.ConfigurationEvent{
					Type:          configapi.ConfigurationEvent_UPDATED,
					Configuration: *configuration,
				}
			case *_map.Removed[configapi.ConfigurationID, *configapi.Configuration]:
				configuration := e.Entry.Value
				configuration.Version = uint64(e.Entry.Version)
				if err := s.populate(context.Background(), configuration); err != nil {
					log.Error(err)
					continue
				}
				configurationEvent = configapi.ConfigurationEvent{
					Type:          configapi.ConfigurationEvent_DELETED,
					Configuration: *configuration,
				}
			}

			var watchers []chan<- configapi.ConfigurationEvent
			s.mu.RLock()
			for _, ch := range s.watchers {
				watchers = append(watchers, ch)
			}
			idWatchers, ok := s.idWatchers[configurationEvent.Configuration.ID]
			if ok {
				for _, ch := range idWatchers {
					watchers = append(watchers, ch)
				}
			}
			s.mu.RUnlock()

			for _, ch := range watchers {
				ch <- configurationEvent
			}
		}
	}()
	return nil
}

func (s *configurationStore) Get(ctx context.Context, id configapi.ConfigurationID) (*configapi.Configuration, error) {
	// If the configuration is not already in the cache, get it from the underlying primitive.
	entry, err := s.configurations.Get(ctx, id)
	if err != nil {
		return nil, errors.FromAtomix(err)
	}

	configuration := entry.Value
	configuration.Key = string(entry.Key)
	configuration.Version = uint64(entry.Version)
	if err := s.populate(ctx, configuration); err != nil {
		log.Error(err)
		return nil, err
	}
	return configuration, nil
}

func (s *configurationStore) Create(ctx context.Context, configuration *configapi.Configuration) error {
	if configuration.ID == "" {
		return errors.NewInvalid("no configuration ID specified")
	}
	if configuration.TargetID == "" {
		return errors.NewInvalid("no target ID specified")
	}
	if configuration.Revision != 0 {
		return errors.NewInvalid("cannot create configuration with revision")
	}
	if configuration.Version != 0 {
		return errors.NewInvalid("cannot create configuration with version")
	}

	if configuration.Values != nil {
		committed, err := s.getCommitted(ctx, configuration.ID)
		if err != nil {
			return err
		}
		if err := s.store(ctx, committed, configuration.Values); err != nil {
			return err
		}
	}

	configuration.Key = string(configuration.ID)
	configuration.Revision = 1
	configuration.Created = time.Now()
	configuration.Updated = time.Now()
	configuration.Values = nil

	// Create the entry in the underlying map primitive.
	entry, err := s.configurations.Insert(ctx, configuration.ID, configuration)
	if err != nil {
		return errors.FromAtomix(err)
	}
	configuration.Version = uint64(entry.Version)
	return nil
}

func (s *configurationStore) Update(ctx context.Context, configuration *configapi.Configuration) error {
	if configuration.ID == "" {
		return errors.NewInvalid("no configuration ID specified")
	}
	if configuration.TargetID == "" {
		return errors.NewInvalid("no target ID specified")
	}
	if configuration.Revision == 0 {
		return errors.NewInvalid("configuration must contain a revision on update")
	}
	if configuration.Version == 0 {
		return errors.NewInvalid("configuration must contain a version on update")
	}

	if configuration.Values != nil {
		committed, err := s.getCommitted(ctx, configuration.ID)
		if err != nil {
			return err
		}
		if err := s.store(ctx, committed, configuration.Values); err != nil {
			return err
		}
	}

	configuration.Revision++
	configuration.Updated = time.Now()
	configuration.Values = nil

	// Update the entry in the underlying map primitive using the configuration version
	// as an optimistic lock.
	entry, err := s.configurations.Update(ctx, configuration.ID, configuration, _map.IfVersion(primitive.Version(configuration.Version)))
	if err != nil {
		return errors.FromAtomix(err)
	}
	configuration.Version = uint64(entry.Version)
	return nil
}

func (s *configurationStore) UpdateStatus(ctx context.Context, configuration *configapi.Configuration) error {
	if configuration.ID == "" {
		return errors.NewInvalid("no configuration ID specified")
	}
	if configuration.TargetID == "" {
		return errors.NewInvalid("no target ID specified")
	}
	if configuration.Revision == 0 {
		return errors.NewInvalid("configuration must contain a revision on update")
	}
	if configuration.Version == 0 {
		return errors.NewInvalid("configuration must contain a version on update")
	}

	if configuration.Status.Applied.Values != nil {
		applied, err := s.getApplied(ctx, configuration.ID)
		if err != nil {
			return err
		}
		if err := s.store(ctx, applied, configuration.Status.Applied.Values); err != nil {
			return err
		}
	}

	configuration.Updated = time.Now()
	configuration.Status.Applied.Values = nil

	// Update the entry in the underlying map primitive using the configuration version
	// as an optimistic lock.
	entry, err := s.configurations.Update(ctx, configuration.ID, configuration, _map.IfVersion(primitive.Version(configuration.Version)))
	if err != nil {
		return errors.FromAtomix(err)
	}
	configuration.Version = uint64(entry.Version)
	return nil
}

func (s *configurationStore) List(ctx context.Context) ([]*configapi.Configuration, error) {
	stream, err := s.configurations.List(ctx)
	if err != nil {
		return nil, errors.FromAtomix(err)
	}

	var configurations []*configapi.Configuration
	for {
		entry, err := stream.Next()
		if err == io.EOF {
			return configurations, nil
		}
		if err != nil {
			log.Error(err)
			return nil, err
		}
		configuration := entry.Value
		configuration.Version = uint64(entry.Version)
		if err := s.populate(ctx, configuration); err != nil {
			log.Error(err)
			return nil, err
		}
		configurations = append(configurations, configuration)
	}
}

func (s *configurationStore) Watch(ctx context.Context, ch chan<- configapi.ConfigurationEvent, opts ...WatchOption) error {
	var options watchOptions
	for _, opt := range opts {
		opt.apply(&options)
	}

	id := uuid.New()
	eventCh := make(chan configapi.ConfigurationEvent)
	s.mu.Lock()
	if options.configurationID != "" {
		watchers, ok := s.idWatchers[options.configurationID]
		if !ok {
			watchers = make(map[uuid.UUID]chan<- configapi.ConfigurationEvent)
			s.idWatchers[options.configurationID] = watchers
		}
		watchers[id] = eventCh
	} else {
		s.watchers[id] = eventCh
	}
	s.mu.Unlock()

	go func() {
		defer func() {
			s.mu.Lock()
			if options.configurationID != "" {
				watchers, ok := s.idWatchers[options.configurationID]
				if ok {
					delete(watchers, id)
					if len(watchers) == 0 {
						delete(s.idWatchers, options.configurationID)
					}
				}
			} else {
				delete(s.watchers, id)
			}
			s.mu.Unlock()
		}()

		if options.replay {
			if options.configurationID != "" {
				entry, err := s.configurations.Get(ctx, options.configurationID)
				if err != nil {
					err = errors.FromAtomix(err)
					if !errors.IsNotFound(err) {
						log.Error(err)
					}
				} else {
					configuration := entry.Value
					configuration.Version = uint64(entry.Version)
					if ctx.Err() != nil {
						close(ch)
						return
					}
					if err := s.populate(ctx, configuration); err != nil {
						log.Error(err)
						return
					}
					ch <- configapi.ConfigurationEvent{
						Type:          configapi.ConfigurationEvent_REPLAYED,
						Configuration: *configuration,
					}
				}
			} else {
				entries, err := s.configurations.List(ctx)
				if err != nil {
					log.Error(err)
					close(ch)
					return
				}
				for {
					entry, err := entries.Next()
					if err == io.EOF {
						break
					}
					if err != nil {
						log.Error(err)
						continue
					}
					if ctx.Err() != nil {
						close(ch)
						return
					}
					configuration := entry.Value
					configuration.Version = uint64(entry.Version)
					if err := s.populate(ctx, configuration); err != nil {
						log.Error(err)
						return
					}
					ch <- configapi.ConfigurationEvent{
						Type:          configapi.ConfigurationEvent_REPLAYED,
						Configuration: *configuration,
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

func (s *configurationStore) populate(ctx context.Context, configuration *configapi.Configuration) error {
	committed, err := s.getCommitted(ctx, configuration.ID)
	if err != nil {
		return err
	}
	stream, err := committed.List(ctx)
	if err != nil {
		return err
	}
	for {
		entry, err := stream.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}
		if configuration.Values == nil {
			configuration.Values = make(map[string]*configapi.PathValue)
		}
		configuration.Values[entry.Key] = entry.Value
	}

	applied, err := s.getApplied(ctx, configuration.ID)
	if err != nil {
		return err
	}
	stream, err = applied.List(ctx)
	if err != nil {
		return err
	}
	for {
		entry, err := stream.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}
		if configuration.Status.Applied.Values == nil {
			configuration.Status.Applied.Values = make(map[string]*configapi.PathValue)
		}
		configuration.Status.Applied.Values[entry.Key] = entry.Value
	}
	return nil
}

func (s *configurationStore) getCommitted(ctx context.Context, id configapi.ConfigurationID) (_map.Map[string, *configapi.PathValue], error) {
	return s.getTarget(ctx, s.committed, id)
}

func (s *configurationStore) getApplied(ctx context.Context, id configapi.ConfigurationID) (_map.Map[string, *configapi.PathValue], error) {
	return s.getTarget(ctx, s.applied, id)
}

func (s *configurationStore) store(ctx context.Context, store _map.Map[string, *configapi.PathValue], values map[string]*configapi.PathValue) error {
	prunedValues := tree.PrunePathMap(values, true)
	transaction := store.Transaction(ctx)
	for _, pv := range values {
		entry, err := store.Get(ctx, pv.Path)
		if err != nil {
			err = errors.FromAtomix(err)
			if !errors.IsNotFound(err) {
				return err
			}
			if _, ok := prunedValues[pv.Path]; ok {
				transaction.Insert(pv.Path, pv)
			}
		} else if _, ok := prunedValues[pv.Path]; !ok {
			transaction.Remove(pv.Path, _map.IfVersion(entry.Version))
		} else if pv.Index != entry.Value.Index {
			transaction.Update(pv.Path, pv, _map.IfVersion(entry.Version))
		}
	}
	if _, err := transaction.Commit(); err != nil {
		err = errors.FromAtomix(err)
		if errors.IsNotFound(err) || errors.IsAlreadyExists(err) || errors.IsConflict(err) {
			return errors.NewConflict(err.Error())
		}
		return err
	}
	return nil
}

func (s *configurationStore) getTarget(
	ctx context.Context,
	targets map[configapi.ConfigurationID]_map.Map[string, *configapi.PathValue],
	id configapi.ConfigurationID) (_map.Map[string, *configapi.PathValue], error) {
	s.mu.RLock()
	target, ok := targets[id]
	s.mu.RUnlock()
	if ok {
		return target, nil
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	target, ok = targets[id]
	if ok {
		return target, nil
	}

	var err error
	target, err = _map.NewBuilder[string, *configapi.PathValue](s.client, fmt.Sprintf("configurations-%s", id)).
		Tag("onos-config", "path-value").
		Codec(types.Proto[*configapi.PathValue](&configapi.PathValue{})).
		Get(ctx)
	if err != nil {
		return nil, errors.FromAtomix(err)
	}
	targets[id] = target
	return target, nil
}

func (s *configurationStore) Close(ctx context.Context) error {
	err := s.configurations.Close(ctx)
	if err != nil {
		return errors.FromAtomix(err)
	}
	return nil
}
