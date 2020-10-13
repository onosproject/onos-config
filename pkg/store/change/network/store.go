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

package network

import (
	"context"
	"errors"
	"io"
	"time"

	"github.com/onosproject/onos-config/pkg/utils"

	"github.com/atomix/go-client/pkg/client/indexedmap"
	"github.com/atomix/go-client/pkg/client/primitive"
	"github.com/atomix/go-client/pkg/client/util/net"
	"github.com/gogo/protobuf/proto"
	"github.com/google/uuid"
	networkchange "github.com/onosproject/onos-config/api/types/change/network"
	"github.com/onosproject/onos-config/pkg/config"
	"github.com/onosproject/onos-config/pkg/store/stream"
	"github.com/onosproject/onos-lib-go/pkg/atomix"
	"github.com/onosproject/onos-lib-go/pkg/cluster"
)

const changesName = "network-changes"

// NewAtomixStore returns a new persistent Store
func NewAtomixStore(cluster cluster.Cluster, config config.Config) (Store, error) {
	uuid.SetNodeID([]byte(cluster.Node().ID))
	database, err := atomix.GetDatabase(config.Atomix, config.Atomix.GetDatabase(atomix.DatabaseTypeConsensus))
	if err != nil {
		return nil, err
	}

	changes, err := database.GetIndexedMap(context.Background(), changesName)
	if err != nil {
		return nil, err
	}

	return &atomixStore{
		changes: changes,
	}, nil
}

// NewLocalStore returns a new local network change store
func NewLocalStore() (Store, error) {
	_, address := atomix.StartLocalNode()
	return newLocalStore(address)
}

// newLocalStore creates a new local network change store
func newLocalStore(address net.Address) (Store, error) {
	configsName := primitive.Name{
		Namespace: "local",
		Name:      changesName,
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	session, err := primitive.NewSession(ctx, primitive.Partition{ID: 1, Address: address})
	if err != nil {
		return nil, err
	}
	changes, err := indexedmap.New(context.Background(), configsName, []*primitive.Session{session})
	if err != nil {
		return nil, err
	}

	return &atomixStore{
		changes: changes,
	}, nil
}

// Store stores NetworkConfig changes
type Store interface {
	io.Closer

	// Get gets a network configuration
	Get(id networkchange.ID) (*networkchange.NetworkChange, error)

	// GetByIndex gets a network change by index
	GetByIndex(index networkchange.Index) (*networkchange.NetworkChange, error)

	// GetPrev gets the previous network change by index
	GetPrev(index networkchange.Index) (*networkchange.NetworkChange, error)

	// GetNext gets the next network change by index
	GetNext(index networkchange.Index) (*networkchange.NetworkChange, error)

	// Create creates a new network configuration
	Create(config *networkchange.NetworkChange) error

	// Update updates an existing network configuration
	Update(config *networkchange.NetworkChange) error

	// Delete deletes a network configuration
	Delete(config *networkchange.NetworkChange) error

	// List lists network configurations
	List(chan<- *networkchange.NetworkChange) (stream.Context, error)

	// Watch watches the network configuration store for changes
	Watch(chan<- stream.Event, ...WatchOption) (stream.Context, error)
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
	id networkchange.ID
}

func (o watchIDOption) apply(opts []indexedmap.WatchOption) []indexedmap.WatchOption {
	return append(opts, indexedmap.WithFilter(indexedmap.Filter{
		Key: string(o.id),
	}))
}

// WithChangeID returns a Watch option that watches for changes to the given change ID
func WithChangeID(id networkchange.ID) WatchOption {
	return watchIDOption{id: id}
}

// newChangeID creates a new network change ID
func newChangeID() networkchange.ID {
	newUUID := utils.NewUUID()
	return networkchange.ID(newUUID.String())
}

// atomixStore is the default implementation of the NetworkConfig store
type atomixStore struct {
	changes indexedmap.IndexedMap
}

func (s *atomixStore) Get(id networkchange.ID) (*networkchange.NetworkChange, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	entry, err := s.changes.Get(ctx, string(id))
	if err != nil {
		return nil, err
	} else if entry == nil {
		return nil, nil
	}
	return decodeChange(entry)
}

func (s *atomixStore) GetByIndex(index networkchange.Index) (*networkchange.NetworkChange, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	entry, err := s.changes.GetIndex(ctx, indexedmap.Index(index))
	if err != nil {
		return nil, err
	} else if entry == nil {
		return nil, nil
	}
	return decodeChange(entry)
}

func (s *atomixStore) GetPrev(index networkchange.Index) (*networkchange.NetworkChange, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	entry, err := s.changes.PrevEntry(ctx, indexedmap.Index(index))
	if err != nil {
		return nil, err
	} else if entry == nil {
		return nil, nil
	}
	return decodeChange(entry)
}

func (s *atomixStore) GetNext(index networkchange.Index) (*networkchange.NetworkChange, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	entry, err := s.changes.NextEntry(ctx, indexedmap.Index(index))
	if err != nil {
		return nil, err
	} else if entry == nil {
		return nil, nil
	}
	return decodeChange(entry)
}

func (s *atomixStore) Create(change *networkchange.NetworkChange) error {
	if change.ID == "" {
		change.ID = newChangeID()
	}
	if change.Revision != 0 {
		return errors.New("not a new object")
	}

	bytes, err := proto.Marshal(change)
	if err != nil {
		return err
	}

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	entry, err := s.changes.Append(ctx, string(change.ID), bytes)
	if err != nil {
		return err
	}

	change.Index = networkchange.Index(entry.Index)
	change.Revision = networkchange.Revision(entry.Version)
	change.Created = entry.Created
	change.Updated = entry.Updated
	return nil
}

func (s *atomixStore) Update(change *networkchange.NetworkChange) error {
	if change.Revision == 0 {
		return errors.New("not a stored object")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	bytes, err := proto.Marshal(change)
	if err != nil {
		return err
	}

	entry, err := s.changes.Set(ctx, indexedmap.Index(change.Index), string(change.ID), bytes, indexedmap.IfVersion(indexedmap.Version(change.Revision)))
	if err != nil {
		return err
	}

	change.Revision = networkchange.Revision(entry.Version)
	change.Updated = entry.Updated
	return nil
}

func (s *atomixStore) Delete(change *networkchange.NetworkChange) error {
	if change.Revision == 0 {
		return errors.New("not a stored object")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	entry, err := s.changes.RemoveIndex(ctx, indexedmap.Index(change.Index), indexedmap.IfVersion(indexedmap.Version(change.Revision)))
	if err != nil {
		return err
	}

	change.Revision = 0
	change.Updated = entry.Updated
	return nil
}

func (s *atomixStore) List(ch chan<- *networkchange.NetworkChange) (stream.Context, error) {
	ctx, cancel := context.WithCancel(context.Background())

	mapCh := make(chan *indexedmap.Entry)
	if err := s.changes.Entries(ctx, mapCh); err != nil {
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

func (s *atomixStore) Watch(ch chan<- stream.Event, opts ...WatchOption) (stream.Context, error) {
	watchOpts := make([]indexedmap.WatchOption, 0)
	for _, opt := range opts {
		watchOpts = opt.apply(watchOpts)
	}

	ctx, cancel := context.WithCancel(context.Background())

	mapCh := make(chan *indexedmap.Event)
	if err := s.changes.Watch(ctx, mapCh, watchOpts...); err != nil {
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
	return stream.NewContext(func() {
		cancel()
	}), nil
}

func (s *atomixStore) Close() error {
	return s.changes.Close(context.Background())
}

func decodeChange(entry *indexedmap.Entry) (*networkchange.NetworkChange, error) {
	change := &networkchange.NetworkChange{}
	if err := proto.Unmarshal(entry.Value, change); err != nil {
		return nil, err
	}
	change.ID = networkchange.ID(entry.Key)
	change.Index = networkchange.Index(entry.Index)
	change.Revision = networkchange.Revision(entry.Version)
	change.Created = entry.Created
	change.Updated = entry.Updated
	return change, nil
}
