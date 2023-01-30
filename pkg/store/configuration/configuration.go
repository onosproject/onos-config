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
	"io"
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
	return &configurationStore{
		configurations: configurations,
	}, nil
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

	configuration.Key = string(configuration.ID)
	configuration.Revision = 1
	configuration.Created = time.Now()
	configuration.Updated = time.Now()

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
	configuration.Revision++
	configuration.Updated = time.Now()

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
	configuration.Updated = time.Now()

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
		configurations = append(configurations, configuration)
	}
}

func (s *configurationStore) Watch(ctx context.Context, ch chan<- configapi.ConfigurationEvent, opts ...WatchOption) error {
	var options watchOptions
	for _, opt := range opts {
		opt.apply(&options)
	}

	var eventsOpts []_map.EventsOption
	if options.configurationID != "" {
		eventsOpts = append(eventsOpts, _map.WithKey[configapi.ConfigurationID](options.configurationID))
	}
	events, err := s.configurations.Events(ctx, eventsOpts...)
	if err != nil {
		return errors.FromAtomix(err)
	}

	if options.replay {
		if options.configurationID != "" {
			entry, err := s.configurations.Get(ctx, options.configurationID)
			if err != nil {
				err = errors.FromAtomix(err)
				if !errors.IsNotFound(err) {
					return err
				}
				go propagateEvents(events, ch)
			} else {
				go func() {
					configuration := entry.Value
					configuration.Version = uint64(entry.Version)
					ch <- configapi.ConfigurationEvent{
						Type:          configapi.ConfigurationEvent_REPLAYED,
						Configuration: *configuration,
					}
					propagateEvents(events, ch)
				}()
			}
		} else {
			entries, err := s.configurations.List(ctx)
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
					configuration := entry.Value
					configuration.Version = uint64(entry.Version)
					ch <- configapi.ConfigurationEvent{
						Type:          configapi.ConfigurationEvent_REPLAYED,
						Configuration: *configuration,
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

func propagateEvents(events _map.EventStream[configapi.ConfigurationID, *configapi.Configuration], ch chan<- configapi.ConfigurationEvent) {
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
		case *_map.Inserted[configapi.ConfigurationID, *configapi.Configuration]:
			configuration := e.Entry.Value
			configuration.Version = uint64(e.Entry.Version)
			ch <- configapi.ConfigurationEvent{
				Type:          configapi.ConfigurationEvent_CREATED,
				Configuration: *configuration,
			}
		case *_map.Updated[configapi.ConfigurationID, *configapi.Configuration]:
			configuration := e.NewEntry.Value
			configuration.Version = uint64(e.NewEntry.Version)
			ch <- configapi.ConfigurationEvent{
				Type:          configapi.ConfigurationEvent_UPDATED,
				Configuration: *configuration,
			}
		case *_map.Removed[configapi.ConfigurationID, *configapi.Configuration]:
			configuration := e.Entry.Value
			configuration.Version = uint64(e.Entry.Version)
			ch <- configapi.ConfigurationEvent{
				Type:          configapi.ConfigurationEvent_DELETED,
				Configuration: *configuration,
			}
		}
	}
}

func (s *configurationStore) Close(ctx context.Context) error {
	err := s.configurations.Close(ctx)
	if err != nil {
		return errors.FromAtomix(err)
	}
	return nil
}
