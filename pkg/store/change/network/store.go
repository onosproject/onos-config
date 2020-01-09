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
	"fmt"
	"github.com/atomix/atomix-go-client/pkg/client/indexedmap"
	"github.com/atomix/atomix-go-client/pkg/client/primitive"
	"github.com/atomix/atomix-go-client/pkg/client/session"
	"github.com/atomix/atomix-go-client/pkg/client/util/net"
	"github.com/gogo/protobuf/proto"
	"github.com/google/uuid"
	networkchange "github.com/onosproject/onos-config/api/types/change/network"
	"github.com/onosproject/onos-config/pkg/store/cluster"
	"github.com/onosproject/onos-config/pkg/store/stream"
	"github.com/onosproject/onos-config/pkg/store/utils"
	"io"
	"sync"
	"time"
)

const changesName = "network-changes"

func init() {
	uuid.SetNodeID([]byte(cluster.GetNodeID()))
}

// NewAtomixStore returns a new persistent Store
func NewAtomixStore() (Store, error) {
	client, err := utils.GetAtomixClient()
	if err != nil {
		return nil, err
	}

	group, err := client.GetGroup(context.Background(), utils.GetAtomixRaftGroup())
	if err != nil {
		return nil, err
	}

	changes, err := group.GetIndexedMap(context.Background(), changesName, session.WithTimeout(30*time.Second))
	if err != nil {
		return nil, err
	}

	store := &atomixStore{
		changes:      changes,
		idWatches:    make(map[networkchange.ID]chan<- stream.Event),
		indexWatches: make(map[networkchange.Index]chan<- stream.Event),
	}
	if err := store.watch(); err != nil {
		return nil, err
	}
	return store, nil
}

// NewLocalStore returns a new local network change store
func NewLocalStore() (Store, error) {
	_, address := utils.StartLocalNode()
	return newLocalStore(address)
}

// newLocalStore creates a new local network change store
func newLocalStore(address net.Address) (Store, error) {
	configsName := primitive.Name{
		Namespace: "local",
		Name:      changesName,
	}
	changes, err := indexedmap.New(context.Background(), configsName, []net.Address{address})
	if err != nil {
		return nil, err
	}

	store := &atomixStore{
		changes:      changes,
		idWatches:    make(map[networkchange.ID]chan<- stream.Event),
		indexWatches: make(map[networkchange.Index]chan<- stream.Event),
	}
	if err := store.watch(); err != nil {
		return nil, err
	}
	return store, nil
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

// watchOptions is a set of options for the store's Watch call
type watchOptions struct {
	replay      bool
	changeID    networkchange.ID
	changeIndex networkchange.Index
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
	id networkchange.ID
}

func (o watchIDOption) apply(options *watchOptions) {
	options.changeID = o.id
}

// WithChangeID returns a Watch option that watches for changes to the given change ID
func WithChangeID(id networkchange.ID) WatchOption {
	return watchIDOption{id: id}
}

type watchIndexOption struct {
	index networkchange.Index
}

func (o watchIndexOption) apply(options *watchOptions) {
	options.changeIndex = o.index
}

// WithChangeIndex returns a Watch option that watches for changes to the given index
func WithChangeIndex(index networkchange.Index) WatchOption {
	return watchIndexOption{index: index}
}

// newChangeID creates a new network change ID
func newChangeID() networkchange.ID {
	return networkchange.ID(uuid.New().String())
}

// atomixStore is the default implementation of the NetworkConfig store
type atomixStore struct {
	changes      indexedmap.IndexedMap
	idWatches    map[networkchange.ID]chan<- stream.Event
	indexWatches map[networkchange.Index]chan<- stream.Event
	mu           sync.RWMutex
}

func (s *atomixStore) watch() error {
	ch := make(chan *indexedmap.Event)
	if err := s.changes.Watch(context.Background(), ch); err != nil {
		return err
	}

	go func() {
		for event := range ch {
			id := networkchange.ID(event.Entry.Key)
			s.mu.RLock()
			watch, ok := s.idWatches[id]
			s.mu.RUnlock()
			if !ok {
				index := networkchange.Index(event.Entry.Index)
				s.mu.RLock()
				watch, ok = s.indexWatches[index]
				s.mu.RUnlock()
				if !ok {
					continue
				}
			}
			if change, err := decodeChange(event.Entry); err == nil {
				switch event.Type {
				case indexedmap.EventNone:
					watch <- stream.Event{
						Type:   stream.None,
						Object: change,
					}
				case indexedmap.EventInserted:
					watch <- stream.Event{
						Type:   stream.Created,
						Object: change,
					}
				case indexedmap.EventUpdated:
					watch <- stream.Event{
						Type:   stream.Updated,
						Object: change,
					}
				case indexedmap.EventRemoved:
					watch <- stream.Event{
						Type:   stream.Deleted,
						Object: change,
					}
				}
			}
		}
	}()
	return nil
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
	watchOpts := &watchOptions{}
	for _, opt := range opts {
		opt.apply(watchOpts)
	}

	// If a change ID was specified and replay is disabled, watch the change on a single stream
	if (watchOpts.changeID != "" || watchOpts.changeIndex > 0) && !watchOpts.replay {
		if watchOpts.changeID != "" {
			return s.watchChangeID(watchOpts.changeID, ch)
		}
		return s.watchChangeIndex(watchOpts.changeIndex, ch)
	}

	mapOpts := []indexedmap.WatchOption{}
	if watchOpts.changeID != "" {
		mapOpts = append(mapOpts, indexedmap.WithFilter(indexedmap.Filter{
			Key: string(watchOpts.changeID),
		}))
	}
	if watchOpts.replay {
		mapOpts = append(mapOpts, indexedmap.WithReplay())
	}

	ctx, cancel := context.WithCancel(context.Background())

	mapCh := make(chan *indexedmap.Event)
	if err := s.changes.Watch(ctx, mapCh, mapOpts...); err != nil {
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

func (s *atomixStore) watchChangeID(id networkchange.ID, ch chan<- stream.Event) (stream.Context, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.idWatches[id]; ok {
		return nil, fmt.Errorf("watch already exists for %v", id)
	}
	s.idWatches[id] = ch
	return stream.NewContext(func() {
		s.mu.Lock()
		defer s.mu.Unlock()
		watch, ok := s.idWatches[id]
		if ok {
			delete(s.idWatches, id)
			close(watch)
		}
	}), nil
}

func (s *atomixStore) watchChangeIndex(index networkchange.Index, ch chan<- stream.Event) (stream.Context, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.indexWatches[index]; ok {
		return nil, fmt.Errorf("watch already exists for %d", index)
	}
	s.indexWatches[index] = ch
	return stream.NewContext(func() {
		s.mu.Lock()
		defer s.mu.Unlock()
		watch, ok := s.indexWatches[index]
		if ok {
			delete(s.indexWatches, index)
			close(watch)
		}
	}), nil
}

func (s *atomixStore) Close() error {
	return s.changes.Close()
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
