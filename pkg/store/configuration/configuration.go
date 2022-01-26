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
func NewID(targetID configapi.TargetID, targetType configapi.TargetType, targetVersion configapi.TargetVersion) configapi.ConfigurationID {
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

	Close(ctx context.Context) error
}

// NewAtomixStore returns a new persistent Store
func NewAtomixStore(client atomix.Client) (Store, error) {
	configurations, err := client.GetMap(context.Background(), "onos-config-configurations")
	if err != nil {
		return nil, errors.FromAtomix(err)
	}
	return &configurationStore{
		configurations: configurations,
	}, nil
}

type configurationStore struct {
	configurations _map.Map
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
	if configuration.ID != "" {
		return errors.NewInvalid("configuration ID is assigned internally")
	}

	if configuration.TargetID == "" {
		return errors.NewInvalid("no target ID specified")
	}

	configuration.ID = configapi.ConfigurationID(configuration.TargetID)
	if configuration.TargetVersion == "" {
		return errors.NewInvalid("no target version specified")
	}

	log.Debugf("Creating configuration %s", configuration.ID)
	bytes, err := proto.Marshal(configuration)
	if err != nil {
		return errors.NewInvalid("configuration encoding failed: %v", err)
	}

	entry, err := s.configurations.Put(ctx, string(configuration.ID), bytes, _map.IfNotSet())
	if err != nil {
		return errors.FromAtomix(err)
	}
	configuration.Revision = configapi.Revision(entry.Revision)
	return nil
}

func (s *configurationStore) Update(ctx context.Context, configuration *configapi.Configuration) error {
	if configuration.ID == "" {
		return errors.NewInvalid("no configuration ID specified")
	}

	if configuration.TargetID == "" {
		return errors.NewInvalid("no target ID specified")
	}

	if configuration.TargetVersion == "" {
		return errors.NewInvalid("no target version specified")
	}

	if configuration.Revision == 0 {
		return errors.NewInvalid("configuration must contain a revision on update")
	}

	log.Debugf("Updating configuration %s", configuration.ID)
	bytes, err := proto.Marshal(configuration)
	if err != nil {
		return errors.NewInvalid("configuration encoding failed: %v", err)
	}
	entry, err := s.configurations.Put(ctx, string(configuration.ID), bytes, _map.IfMatch(meta.NewRevision(meta.Revision(configuration.Revision))))
	if err != nil {
		return errors.FromAtomix(err)
	}
	configuration.Revision = configapi.Revision(entry.Revision)
	return nil
}

func (s *configurationStore) Delete(ctx context.Context, configuration *configapi.Configuration) error {
	if configuration.Revision == 0 {
		return errors.NewInvalid("configuration must contain a revision on delete")
	}

	log.Debugf("Deleting configuration %s", configuration.ID)
	_, err := s.configurations.Remove(ctx, string(configuration.ID), _map.IfMatch(meta.NewRevision(meta.Revision(configuration.Revision))))
	if err != nil {
		log.Warnf("Failed to delete configuration %s: %s", configuration.ID, err)
		return errors.FromAtomix(err)
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
	watchOpts := make([]_map.WatchOption, 0)
	for _, opt := range opts {
		watchOpts = opt.apply(watchOpts)
	}

	mapCh := make(chan _map.Event)
	if err := s.configurations.Watch(ctx, mapCh, watchOpts...); err != nil {
		return errors.FromAtomix(err)
	}

	go func() {
		defer close(ch)
		for event := range mapCh {
			if configuration, err := decodeConfiguration(event.Entry); err == nil {
				var eventType configapi.ConfigurationEventType
				switch event.Type {
				case _map.EventReplay:
					eventType = configapi.ConfigurationEventType_CONFIGURATION_EVENT_UNKNOWN
				case _map.EventInsert:
					eventType = configapi.ConfigurationEventType_CONFIGURATION_CREATED
				case _map.EventRemove:
					eventType = configapi.ConfigurationEventType_CONFIGURATION_DELETED
				case _map.EventUpdate:
					eventType = configapi.ConfigurationEventType_CONFIGURATION_UPDATED
				default:
					eventType = configapi.ConfigurationEventType_CONFIGURATION_UPDATED
				}
				ch <- configapi.ConfigurationEvent{
					Type:          eventType,
					Configuration: *configuration,
				}
			}
		}
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

func decodeConfiguration(entry _map.Entry) (*configapi.Configuration, error) {
	configuration := &configapi.Configuration{}
	if err := proto.Unmarshal(entry.Value, configuration); err != nil {
		return nil, errors.NewInvalid("configuration decoding failed: %v", err)
	}
	configuration.ID = configapi.ConfigurationID(entry.Key)
	configuration.Revision = configapi.Revision(entry.Revision)
	return configuration, nil
}
