// SPDX-FileCopyrightText: 2020-present Open Networking Foundation <info@opennetworking.org>
//
// SPDX-License-Identifier: Apache-2.0

package configuration

import (
	"context"
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

var log = logging.GetLogger("store", "configuration")

// NewID returns a new Configuration ID for the given target/type/version
func NewID(targetID configapi.TargetID) configapi.ConfigurationID {
	return configapi.ConfigurationID(targetID)
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
func NewAtomixStore(client atomix.Client) (Store, error) {
	configurations, err := client.GetMap(context.Background(), "onos-config-configurations")
	if err != nil {
		return nil, errors.FromAtomix(err)
	}
	store := &configurationStore{
		configurations: configurations,
		cache:          make(map[configapi.ConfigurationID]*_map.Entry),
		watchers:       make(map[uuid.UUID]chan<- configapi.ConfigurationEvent),
		eventCh:        make(chan configapi.ConfigurationEvent, 1000),
	}
	if err := store.open(context.Background()); err != nil {
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
	configurations _map.Map
	cache          map[configapi.ConfigurationID]*_map.Entry
	cacheMu        sync.RWMutex
	watchers       map[uuid.UUID]chan<- configapi.ConfigurationEvent
	watchersMu     sync.RWMutex
	eventCh        chan configapi.ConfigurationEvent
}

func (s *configurationStore) open(ctx context.Context) error {
	ch := make(chan _map.Event)
	if err := s.configurations.Watch(ctx, ch, _map.WithReplay()); err != nil {
		return err
	}
	go func() {
		for event := range ch {
			entry := event.Entry
			s.updateCache(&entry)
		}
	}()
	go s.processEvents()
	return nil
}

func (s *configurationStore) processEvents() {
	for event := range s.eventCh {
		s.watchersMu.RLock()
		for _, watcher := range s.watchers {
			watcher <- event
		}
		s.watchersMu.RUnlock()
	}
}

func (s *configurationStore) updateCache(newEntry *_map.Entry) {
	configurationID := configapi.ConfigurationID(newEntry.Key)

	// Use a double-checked lock when updating the cache.
	// First, check for a more recent version of the configuration already in the cache.
	s.cacheMu.RLock()
	entry, ok := s.cache[configurationID]
	s.cacheMu.RUnlock()
	if ok && entry.Revision >= newEntry.Revision {
		return
	}

	// The cache needs to be updated. Acquire a write lock and check once again
	// for a more recent version of the configuration.
	s.cacheMu.Lock()
	defer s.cacheMu.Unlock()
	entry, ok = s.cache[configurationID]
	if !ok {
		s.cache[configurationID] = newEntry
		var configuration configapi.Configuration
		if err := decodeConfiguration(newEntry, &configuration); err != nil {
			log.Error(err)
		} else {
			s.eventCh <- configapi.ConfigurationEvent{
				Type:          configapi.ConfigurationEvent_CREATED,
				Configuration: configuration,
			}
		}
	} else if newEntry.Revision > entry.Revision {
		// Add the configuration to the ID and index caches and publish an event.
		s.cache[configurationID] = newEntry
		var configuration configapi.Configuration
		if err := decodeConfiguration(newEntry, &configuration); err != nil {
			log.Error(err)
		} else {
			s.eventCh <- configapi.ConfigurationEvent{
				Type:          configapi.ConfigurationEvent_UPDATED,
				Configuration: configuration,
			}
		}
	}
}

func (s *configurationStore) Get(ctx context.Context, id configapi.ConfigurationID) (*configapi.Configuration, error) {
	// Check the ID cache for the latest version of the configuration.
	s.cacheMu.RLock()
	cachedEntry, ok := s.cache[id]
	s.cacheMu.RUnlock()
	if ok {
		configuration := &configapi.Configuration{}
		if err := decodeConfiguration(cachedEntry, configuration); err != nil {
			return nil, errors.NewInvalid("configuration decoding failed: %v", err)
		}
		return configuration, nil
	}

	// If the configuration is not already in the cache, get it from the underlying primitive.
	entry, err := s.configurations.Get(ctx, string(id))
	if err != nil {
		return nil, errors.FromAtomix(err)
	}

	// Decode and return the Configuration.
	configuration := &configapi.Configuration{}
	if err := decodeConfiguration(entry, configuration); err != nil {
		return nil, errors.NewInvalid("configuration decoding failed: %v", err)
	}

	// Update the cache.
	s.updateCache(entry)
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
	configuration.Revision = 1
	configuration.Created = time.Now()
	configuration.Updated = time.Now()

	// Encode the configuration bytes.
	bytes, err := proto.Marshal(configuration)
	if err != nil {
		return errors.NewInvalid("configuration encoding failed: %v", err)
	}

	// Create the entry in the underlying map primitive.
	entry, err := s.configurations.Put(ctx, string(configuration.ID), bytes, _map.IfNotSet())
	if err != nil {
		return errors.FromAtomix(err)
	}

	// Decode the configuration from the returned entry bytes.
	if err := decodeConfiguration(entry, configuration); err != nil {
		return errors.NewInvalid("configuration decoding failed: %v", err)
	}

	// Update the cache.
	s.updateCache(entry)
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
	configuration.Revision++
	configuration.Updated = time.Now()

	// Encode the configuration bytes.
	bytes, err := proto.Marshal(configuration)
	if err != nil {
		return errors.NewInvalid("configuration encoding failed: %v", err)
	}

	// Update the entry in the underlying map primitive using the configuration version
	// as an optimistic lock.
	entry, err := s.configurations.Put(ctx, string(configuration.ID), bytes, _map.IfMatch(meta.NewRevision(meta.Revision(configuration.Version))))
	if err != nil {
		return errors.FromAtomix(err)
	}

	// Decode the configuration from the returned entry bytes.
	if err := decodeConfiguration(entry, configuration); err != nil {
		return errors.NewInvalid("configuration decoding failed: %v", err)
	}

	// Update the cache.
	s.updateCache(entry)
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
	configuration.Updated = time.Now()

	// Encode the configuration bytes.
	bytes, err := proto.Marshal(configuration)
	if err != nil {
		return errors.NewInvalid("configuration encoding failed: %v", err)
	}

	// Update the entry in the underlying map primitive using the configuration version
	// as an optimistic lock.
	entry, err := s.configurations.Put(ctx, string(configuration.ID), bytes, _map.IfMatch(meta.NewRevision(meta.Revision(configuration.Version))))
	if err != nil {
		return errors.FromAtomix(err)
	}

	// Decode the configuration from the returned entry bytes.
	if err := decodeConfiguration(entry, configuration); err != nil {
		return errors.NewInvalid("configuration decoding failed: %v", err)
	}

	// Update the cache.
	s.updateCache(entry)
	return nil
}

func (s *configurationStore) List(ctx context.Context) ([]*configapi.Configuration, error) {
	mapCh := make(chan _map.Entry)
	if err := s.configurations.Entries(ctx, mapCh); err != nil {
		return nil, errors.FromAtomix(err)
	}

	configurations := make([]*configapi.Configuration, 0)

	for entry := range mapCh {
		configuration := &configapi.Configuration{}
		if err := decodeConfiguration(&entry, configuration); err != nil {
			log.Error(err)
		} else {
			configurations = append(configurations, configuration)
		}
	}
	return configurations, nil
}

func (s *configurationStore) Watch(ctx context.Context, ch chan<- configapi.ConfigurationEvent, opts ...WatchOption) error {
	var options watchOptions
	for _, opt := range opts {
		opt.apply(&options)
	}

	watchCh := make(chan configapi.ConfigurationEvent, 10)
	id := uuid.New()
	s.watchersMu.Lock()
	s.watchers[id] = watchCh
	s.watchersMu.Unlock()

	var replay []configapi.ConfigurationEvent
	if options.replay {
		if options.configurationID == "" {
			s.cacheMu.RLock()
			replay = make([]configapi.ConfigurationEvent, 0, len(s.cache))
			for _, entry := range s.cache {
				var configuration configapi.Configuration
				if err := decodeConfiguration(entry, &configuration); err != nil {
					log.Error(err)
				} else {
					replay = append(replay, configapi.ConfigurationEvent{
						Type:          configapi.ConfigurationEvent_REPLAYED,
						Configuration: configuration,
					})
				}
			}
			s.cacheMu.RUnlock()
		} else {
			s.cacheMu.RLock()
			entry, ok := s.cache[options.configurationID]
			if ok {
				var configuration configapi.Configuration
				if err := decodeConfiguration(entry, &configuration); err != nil {
					log.Error(err)
				} else {
					replay = []configapi.ConfigurationEvent{
						{
							Type:          configapi.ConfigurationEvent_REPLAYED,
							Configuration: configuration,
						},
					}
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
			if options.configurationID == "" || event.Configuration.ID == options.configurationID {
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

func (s *configurationStore) Close(ctx context.Context) error {
	err := s.configurations.Close(ctx)
	if err != nil {
		return errors.FromAtomix(err)
	}
	return nil
}

func decodeConfiguration(entry *_map.Entry, configuration *configapi.Configuration) error {
	if err := proto.Unmarshal(entry.Value, configuration); err != nil {
		return err
	}
	configuration.ID = configapi.ConfigurationID(entry.Key)
	configuration.Key = entry.Key
	configuration.Version = uint64(entry.Revision)
	return nil
}
