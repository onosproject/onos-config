// Copyright 2019-present Open Networking Foundation.
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

package device

import (
	"context"
	"errors"
	"fmt"
	"github.com/onosproject/onos-config/pkg/config"
	"io"
	"sync"
	"time"

	"github.com/atomix/go-client/pkg/client/indexedmap"
	"github.com/atomix/go-client/pkg/client/primitive"
	"github.com/atomix/go-client/pkg/client/util/net"
	"github.com/gogo/protobuf/proto"
	devicechange "github.com/onosproject/onos-config/api/types/change/device"
	"github.com/onosproject/onos-config/api/types/device"
	"github.com/onosproject/onos-config/pkg/store/stream"
	"github.com/onosproject/onos-lib-go/pkg/atomix"
	"github.com/onosproject/onos-lib-go/pkg/logging"
)

var log = logging.GetLogger("store", "change", "device")

// getDeviceChangesName returns the name of the changes map for the given device ID
func getDeviceChangesName(deviceID device.VersionedID) string {
	return fmt.Sprintf("device-changes-%s", deviceID)
}

// NewAtomixStore returns a new persistent Store
func NewAtomixStore(config config.Config) (Store, error) {
	database, err := atomix.GetDatabase(config.Atomix, config.Atomix.GetDatabase(atomix.DatabaseTypeConsensus))
	if err != nil {
		return nil, err
	}

	changesFactory := func(deviceID device.VersionedID) (indexedmap.IndexedMap, error) {
		return database.GetIndexedMap(context.Background(), getDeviceChangesName(deviceID))
	}

	return &atomixStore{
		changesFactory: changesFactory,
		deviceChanges:  make(map[device.VersionedID]indexedmap.IndexedMap),
	}, nil
}

// NewLocalStore returns a new local device store
func NewLocalStore() (Store, error) {
	_, address := atomix.StartLocalNode()
	return newLocalStore(address)
}

// newLocalStore creates a new local device change store
func newLocalStore(address net.Address) (Store, error) {
	changesFactory := func(deviceID device.VersionedID) (indexedmap.IndexedMap, error) {
		counterName := primitive.Name{
			Namespace: "local",
			Name:      getDeviceChangesName(deviceID),
		}
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		session, err := primitive.NewSession(ctx, primitive.Partition{ID: 1, Address: address})
		if err != nil {
			return nil, err
		}
		return indexedmap.New(context.Background(), counterName, []*primitive.Session{session})
	}

	return &atomixStore{
		changesFactory: changesFactory,
		deviceChanges:  make(map[device.VersionedID]indexedmap.IndexedMap),
	}, nil
}

// Store stores DeviceChanges
type Store interface {
	io.Closer

	// Get gets a device change
	Get(id devicechange.ID) (*devicechange.DeviceChange, error)

	// Create creates a new device change
	Create(change *devicechange.DeviceChange) error

	// Update updates an existing device change
	Update(change *devicechange.DeviceChange) error

	// Delete deletes a device change
	Delete(change *devicechange.DeviceChange) error

	// List lists device change
	List(deviceID device.VersionedID, ch chan<- *devicechange.DeviceChange) (stream.Context, error)

	// Watch watches the device change store for changes
	Watch(deviceID device.VersionedID, ch chan<- stream.Event, opts ...WatchOption) (stream.Context, error)
}

// WatchOption is a configuration option for Watch calls
type WatchOption interface {
	apply([]indexedmap.WatchOption) []indexedmap.WatchOption
}

// watchReplyOption is an option to replay events on watch
type watchReplayOption struct {
}

func (o watchReplayOption) apply(opts []indexedmap.WatchOption) []indexedmap.WatchOption {
	return append(opts, indexedmap.WithReplay())
}

// WithReplay returns a WatchOption that replays past changes
func WithReplay() WatchOption {
	return watchReplayOption{}
}

type watchIDOption struct {
	id devicechange.ID
}

func (o watchIDOption) apply(opts []indexedmap.WatchOption) []indexedmap.WatchOption {
	return append(opts, indexedmap.WithFilter(indexedmap.Filter{
		Key: string(o.id),
	}))
}

// WithChangeID returns a Watch option that watches for changes to the given change ID
func WithChangeID(id devicechange.ID) WatchOption {
	return watchIDOption{id: id}
}

// atomixStore is the default implementation of the NetworkConfig store
type atomixStore struct {
	changesFactory func(device.VersionedID) (indexedmap.IndexedMap, error)
	deviceChanges  map[device.VersionedID]indexedmap.IndexedMap
	mu             sync.RWMutex
}

func (s *atomixStore) getDeviceChanges(deviceID device.VersionedID) (indexedmap.IndexedMap, error) {
	s.mu.RLock()
	changes, ok := s.deviceChanges[deviceID]
	s.mu.RUnlock()
	if !ok {
		s.mu.Lock()
		defer s.mu.Unlock()
		changes, ok = s.deviceChanges[deviceID]
		if !ok {
			newChanges, err := s.changesFactory(deviceID)
			if err != nil {
				return nil, err
			}
			s.deviceChanges[deviceID] = newChanges
			return newChanges, nil
		}
	}
	return changes, nil
}

func (s *atomixStore) Get(id devicechange.ID) (*devicechange.DeviceChange, error) {
	changes, err := s.getDeviceChanges(id.GetDeviceVersionedID())
	if err != nil {
		return nil, err
	}

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	entry, err := changes.Get(ctx, string(id))
	if err != nil {
		return nil, err
	} else if entry == nil {
		return nil, nil
	}
	return decodeChange(entry)
}

func (s *atomixStore) Create(change *devicechange.DeviceChange) error {
	if change.Index == 0 {
		return errors.New("no change index specified")
	}
	if change.Change.DeviceID == "" {
		return errors.New("no device ID specified")
	}
	if change.NetworkChange.ID == "" {
		return errors.New("no NetworkChange ID specified")
	}
	if change.Revision != 0 {
		return errors.New("not a new object")
	}
	if change.Change.DeviceID == "" {
		return errors.New("no device ID specified")
	}
	if change.Change.DeviceVersion == "" {
		return errors.New("no device version specified")
	}
	if change.Change.DeviceType == "" {
		return errors.New("no device type specified")
	}

	change.ID = devicechange.NewID(change.NetworkChange.ID, change.Change.DeviceID, change.Change.DeviceVersion)

	changes, err := s.getDeviceChanges(change.Change.GetVersionedDeviceID())
	if err != nil {
		return err
	}

	bytes, err := proto.Marshal(change)
	if err != nil {
		return err
	}

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	entry, err := changes.Set(ctx, indexedmap.Index(change.Index), string(change.ID), bytes, indexedmap.IfNotSet())
	if err != nil {
		return err
	}

	change.Index = devicechange.Index(entry.Index)
	change.Revision = devicechange.Revision(entry.Version)
	change.Created = entry.Created
	change.Updated = entry.Updated
	log.Infof("Created new device change %s", change.ID)

	return nil
}

func (s *atomixStore) Update(change *devicechange.DeviceChange) error {
	if change.ID == "" {
		return errors.New("no change ID configured")
	}
	if change.Index == 0 {
		return errors.New("not a stored object: no storage index found")
	}
	if change.Revision == 0 {
		return errors.New("not a stored object: no storage revision found")
	}
	if change.Change.DeviceID == "" {
		return errors.New("no device ID specified")
	}
	if change.Change.DeviceVersion == "" {
		return errors.New("no device version specified")
	}
	if change.Change.DeviceType == "" {
		return errors.New("no device type specified")
	}

	changes, err := s.getDeviceChanges(change.Change.GetVersionedDeviceID())
	if err != nil {
		return err
	}

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	bytes, err := proto.Marshal(change)
	if err != nil {
		return err
	}

	entry, err := changes.Set(ctx, indexedmap.Index(change.Index), string(change.ID), bytes, indexedmap.IfVersion(indexedmap.Version(change.Revision)))
	if err != nil {
		return err
	}

	change.Revision = devicechange.Revision(entry.Version)
	if change.Created.IsZero() {
		change.Created = entry.Created
	}
	change.Updated = entry.Updated
	return nil
}

func (s *atomixStore) Delete(change *devicechange.DeviceChange) error {
	if change.ID == "" {
		return errors.New("no change ID configured")
	}
	if change.Index == 0 {
		return errors.New("not a stored object: no storage index found")
	}
	if change.Revision == 0 {
		return errors.New("not a stored object")
	}

	changes, err := s.getDeviceChanges(change.Change.GetVersionedDeviceID())
	if err != nil {
		return err
	}

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	entry, err := changes.RemoveIndex(ctx, indexedmap.Index(change.Index), indexedmap.IfVersion(indexedmap.Version(change.Revision)))
	if err != nil {
		return err
	}

	change.Revision = 0
	change.Updated = entry.Updated
	return nil
}

func (s *atomixStore) List(deviceID device.VersionedID, ch chan<- *devicechange.DeviceChange) (stream.Context, error) {
	changes, err := s.getDeviceChanges(deviceID)
	if err != nil {
		return nil, err
	}

	ctx, cancel := context.WithCancel(context.Background())

	mapCh := make(chan *indexedmap.Entry)
	if err := changes.Entries(ctx, mapCh); err != nil {
		cancel()
		return nil, err
	}

	go func() {
		defer close(ch)
		for entry := range mapCh {
			if config, err := decodeChange(entry); err == nil {
				ch <- config
			}
		}
	}()
	return stream.NewCancelContext(cancel), nil
}

func (s *atomixStore) Watch(deviceID device.VersionedID, ch chan<- stream.Event, opts ...WatchOption) (stream.Context, error) {
	changes, err := s.getDeviceChanges(deviceID)
	if err != nil {
		return nil, err
	}

	watchOpts := make([]indexedmap.WatchOption, 0)
	for _, opt := range opts {
		watchOpts = opt.apply(watchOpts)
	}

	ctx, cancel := context.WithCancel(context.Background())
	mapCh := make(chan *indexedmap.Event)
	if err := changes.Watch(ctx, mapCh, watchOpts...); err != nil {
		cancel()
		return nil, err
	}

	go func() {
		defer close(ch)
		for event := range mapCh {
			if change, err := decodeChange(event.Entry); err == nil {
				switch event.Type {
				case indexedmap.EventNone:
					ch <- stream.Event{
						Type:   stream.None,
						Object: change,
					}
				case indexedmap.EventInserted:
					ch <- stream.Event{
						Type:   stream.Created,
						Object: change,
					}
				case indexedmap.EventUpdated:
					ch <- stream.Event{
						Type:   stream.Updated,
						Object: change,
					}
				case indexedmap.EventRemoved:
					ch <- stream.Event{
						Type:   stream.Deleted,
						Object: change,
					}
				}
			}
		}
	}()
	return stream.NewCancelContext(cancel), nil
}

func (s *atomixStore) Close() error {
	var returnErr error
	for _, changes := range s.deviceChanges {
		if err := changes.Close(context.Background()); err != nil {
			returnErr = err
		}
	}
	return returnErr
}

func decodeChange(entry *indexedmap.Entry) (*devicechange.DeviceChange, error) {
	change := &devicechange.DeviceChange{}
	if err := proto.Unmarshal(entry.Value, change); err != nil {
		return nil, err
	}
	change.ID = devicechange.ID(entry.Key)
	change.Index = devicechange.Index(entry.Index)
	change.Revision = devicechange.Revision(entry.Version)
	change.Created = entry.Created
	change.Updated = entry.Updated
	return change, nil
}
