// Copyright 2021-present Open Networking Foundation.
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

package configuration

import (
	"context"
	"sync"
	"time"

	"github.com/google/uuid"

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

	// Delete deletes a configuration
	Delete(ctx context.Context, configuration *configapi.Configuration) error

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
	configStore := &configurationStore{
		configurations: configurations,
		watchers:       newWatchers(),
		cache:          make(map[configapi.ConfigurationID]configapi.Configuration),
	}

	eventCh := make(chan _map.Event)
	if err := configurations.Watch(context.Background(), eventCh, _map.WithReplay()); err != nil {
		return nil, errors.FromAtomix(err)
	}

	go configStore.watchConfigurationEvent(eventCh)
	return configStore, nil
}

type configurationStore struct {
	configurations _map.Map
	watchers       *watchers
	cache          map[configapi.ConfigurationID]configapi.Configuration
	cacheMu        sync.RWMutex
}

// WatchOption is a configuration option for Watch calls
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
	id configapi.ConfigurationID
}

func (o watchIDOption) apply(opts []_map.WatchOption) []_map.WatchOption {
	return append(opts, _map.WithFilter(_map.Filter{
		Key: string(o.id),
	}))
}

// WithConfigurationID returns a Watch option that watches for configurations based on a  given configuration ID
func WithConfigurationID(id configapi.ConfigurationID) WatchOption {
	return watchIDOption{id: id}
}

func (s *configurationStore) Get(ctx context.Context, id configapi.ConfigurationID) (*configapi.Configuration, error) {
	log.Debugf("Getting configuration %s", id)
	entry, err := s.configurations.Get(ctx, string(id))
	if err != nil {
		return nil, errors.FromAtomix(err)
	}
	return decodeConfiguration(*entry)
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

	log.Debugf("Creating configuration %s", configuration.ID)
	bytes, err := proto.Marshal(configuration)
	if err != nil {
		return errors.NewInvalid("configuration encoding failed: %v", err)
	}

	entry, err := s.configurations.Put(ctx, string(configuration.ID), bytes, _map.IfNotSet())
	if err != nil {
		return errors.FromAtomix(err)
	}
	configuration.Version = uint64(entry.Revision)
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

	log.Debugf("Updating configuration %s", configuration.ID)
	bytes, err := proto.Marshal(configuration)
	if err != nil {
		return errors.NewInvalid("configuration encoding failed: %v", err)
	}
	entry, err := s.configurations.Put(ctx, string(configuration.ID), bytes, _map.IfMatch(meta.NewRevision(meta.Revision(configuration.Version))))
	if err != nil {
		return errors.FromAtomix(err)
	}
	configuration.Version = uint64(entry.Revision)
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

	log.Debugf("Updating configuration %s", configuration.ID)
	bytes, err := proto.Marshal(configuration)
	if err != nil {
		return errors.NewInvalid("configuration encoding failed: %v", err)
	}
	entry, err := s.configurations.Put(ctx, string(configuration.ID), bytes, _map.IfMatch(meta.NewRevision(meta.Revision(configuration.Version))))
	if err != nil {
		return errors.FromAtomix(err)
	}
	configuration.Version = uint64(entry.Revision)
	return nil
}

func (s *configurationStore) Delete(ctx context.Context, configuration *configapi.Configuration) error {
	if configuration.Version == 0 {
		return errors.NewInvalid("configuration must contain a version on delete")
	}

	if configuration.Deleted == nil {
		log.Debugf("Updating configuration %s", configuration.ID)
		t := time.Now()
		configuration.Deleted = &t
		bytes, err := proto.Marshal(configuration)
		if err != nil {
			return errors.NewInvalid("configuration encoding failed: %v", err)
		}
		entry, err := s.configurations.Put(ctx, string(configuration.ID), bytes, _map.IfMatch(meta.NewRevision(meta.Revision(configuration.Version))))
		if err != nil {
			return errors.FromAtomix(err)
		}
		configuration.Version = uint64(entry.Revision)
	} else {
		log.Debugf("Deleting configuration %s", configuration.ID)
		_, err := s.configurations.Remove(ctx, string(configuration.ID), _map.IfMatch(meta.NewRevision(meta.Revision(configuration.Version))))
		if err != nil {
			log.Warnf("Failed to delete configuration %s: %s", configuration.ID, err)
			return errors.FromAtomix(err)
		}
		configuration.Version = 0
	}
	return nil
}

func (s *configurationStore) List(ctx context.Context) ([]*configapi.Configuration, error) {
	log.Debugf("Listing configurations")
	mapCh := make(chan _map.Entry)
	if err := s.configurations.Entries(ctx, mapCh); err != nil {
		return nil, errors.FromAtomix(err)
	}

	configurations := make([]*configapi.Configuration, 0)

	for entry := range mapCh {
		if configuration, err := decodeConfiguration(entry); err == nil {
			configurations = append(configurations, configuration)
		}
	}
	return configurations, nil
}

func (s *configurationStore) Watch(ctx context.Context, ch chan<- configapi.ConfigurationEvent, opts ...WatchOption) error {
	watcher := watcher{}
	for _, opt := range opts {
		switch e := opt.(type) {
		case watchReplayOption:
			watcher.replay = true
		case watchIDOption:
			watcher.watchID = e.id
		}
	}

	watcher.eventCh = ch
	watcherID := uuid.New()
	err := s.watchers.add(watcherID, watcher)
	if err != nil {
		return err
	}

	go func() {
		<-ctx.Done()
		s.watchers.remove(watcherID)
		close(ch)
	}()

	return nil
}

func (s *configurationStore) watchConfigurationEvent(eventCh chan _map.Event) {
	log.Debugf("Starting watching configuration changes")
	for event := range eventCh {
		configuration, err := decodeConfiguration(event.Entry)
		if err != nil {
			continue
		}
		var eventType configapi.ConfigurationEvent_EventType
		switch event.Type {
		case _map.EventReplay:
			eventType = configapi.ConfigurationEvent_REPLAYED
			s.cacheMu.Lock()
			s.cache[configuration.ID] = *configuration
			s.cacheMu.Unlock()
		case _map.EventInsert:
			eventType = configapi.ConfigurationEvent_CREATED
			s.cacheMu.Lock()
			s.cache[configuration.ID] = *configuration
			s.cacheMu.Unlock()
		case _map.EventUpdate:
			eventType = configapi.ConfigurationEvent_UPDATED
			s.cacheMu.Lock()
			s.cache[configuration.ID] = *configuration
			s.cacheMu.Unlock()
		case _map.EventRemove:
			eventType = configapi.ConfigurationEvent_DELETED
			s.cacheMu.Lock()
			delete(s.cache, configapi.ConfigurationID(event.Entry.Key))
			s.cacheMu.Unlock()
		}
		configEvent := configapi.ConfigurationEvent{
			Type:          eventType,
			Configuration: *configuration,
		}
		s.watchers.sendAll(configEvent)
	}
}

func (s *configurationStore) Close(ctx context.Context) error {
	err := s.configurations.Close(ctx)
	if err != nil {
		return errors.FromAtomix(err)
	}
	return nil
}

func decodeConfiguration(entry _map.Entry) (*configapi.Configuration, error) {
	configuration := &configapi.Configuration{}
	if err := proto.Unmarshal(entry.Value, configuration); err != nil {
		return nil, errors.NewInvalid("configuration decoding failed: %v", err)
	}
	configuration.ID = configapi.ConfigurationID(entry.Key)
	configuration.Key = entry.Key
	configuration.Version = uint64(entry.Revision)
	return configuration, nil
}
