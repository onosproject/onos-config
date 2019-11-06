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
	"github.com/atomix/atomix-go-client/pkg/client/indexedmap"
	"github.com/atomix/atomix-go-client/pkg/client/primitive"
	"github.com/atomix/atomix-go-client/pkg/client/session"
	"github.com/atomix/atomix-go-local/pkg/atomix/local"
	"github.com/atomix/atomix-go-node/pkg/atomix"
	"github.com/atomix/atomix-go-node/pkg/atomix/registry"
	"github.com/gogo/protobuf/proto"
	"github.com/google/uuid"
	networkchangetypes "github.com/onosproject/onos-config/api/types/change/network"
	"github.com/onosproject/onos-config/pkg/store/cluster"
	"github.com/onosproject/onos-config/pkg/store/stream"
	"github.com/onosproject/onos-config/pkg/store/utils"
	"google.golang.org/grpc"
	"google.golang.org/grpc/test/bufconn"
	"io"
	"net"
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

	return &atomixStore{
		changes: changes,
	}, nil
}

// NewLocalStore returns a new local network change store
func NewLocalStore() (Store, error) {
	_, conn := startLocalNode()
	return newLocalStore(conn)
}

// newLocalStore creates a new local network change store
func newLocalStore(conn *grpc.ClientConn) (Store, error) {
	configsName := primitive.Name{
		Namespace: "local",
		Name:      changesName,
	}
	changes, err := indexedmap.New(context.Background(), configsName, []*grpc.ClientConn{conn})
	if err != nil {
		return nil, err
	}

	return &atomixStore{
		changes: changes,
	}, nil
}

// startLocalNode starts a single local node
func startLocalNode() (*atomix.Node, *grpc.ClientConn) {
	lis := bufconn.Listen(1024 * 1024)
	node := local.NewNode(lis, registry.Registry)
	_ = node.Start()

	dialer := func(ctx context.Context, address string) (net.Conn, error) {
		return lis.Dial()
	}

	conn, err := grpc.DialContext(context.Background(), changesName, grpc.WithContextDialer(dialer), grpc.WithInsecure())
	if err != nil {
		panic("Failed to dial network configurations")
	}
	return node, conn
}

// Store stores NetworkConfig changes
type Store interface {
	io.Closer

	// Get gets a network configuration
	Get(id networkchangetypes.ID) (*networkchangetypes.NetworkChange, error)

	// GetByIndex gets a network change by index
	GetByIndex(index networkchangetypes.Index) (*networkchangetypes.NetworkChange, error)

	// GetPrev gets the previous network change by index
	GetPrev(index networkchangetypes.Index) (*networkchangetypes.NetworkChange, error)

	// GetNext gets the next network change by index
	GetNext(index networkchangetypes.Index) (*networkchangetypes.NetworkChange, error)

	// Create creates a new network configuration
	Create(config *networkchangetypes.NetworkChange) error

	// Update updates an existing network configuration
	Update(config *networkchangetypes.NetworkChange) error

	// Delete deletes a network configuration
	Delete(config *networkchangetypes.NetworkChange) error

	// List lists network configurations
	List(chan<- *networkchangetypes.NetworkChange) (stream.Context, error)

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
	id networkchangetypes.ID
}

func (o watchIDOption) apply(opts []indexedmap.WatchOption) []indexedmap.WatchOption {
	return append(opts, indexedmap.WithFilter(indexedmap.Filter{
		Key: string(o.id),
	}))
}

// WithChangeID returns a Watch option that watches for changes to the given change ID
func WithChangeID(id networkchangetypes.ID) WatchOption {
	return watchIDOption{id: id}
}

// newChangeID creates a new network change ID
func newChangeID() networkchangetypes.ID {
	return networkchangetypes.ID(uuid.New().String())
}

// atomixStore is the default implementation of the NetworkConfig store
type atomixStore struct {
	changes indexedmap.IndexedMap
}

func (s *atomixStore) Get(id networkchangetypes.ID) (*networkchangetypes.NetworkChange, error) {
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

func (s *atomixStore) GetByIndex(index networkchangetypes.Index) (*networkchangetypes.NetworkChange, error) {
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

func (s *atomixStore) GetPrev(index networkchangetypes.Index) (*networkchangetypes.NetworkChange, error) {
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

func (s *atomixStore) GetNext(index networkchangetypes.Index) (*networkchangetypes.NetworkChange, error) {
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

func (s *atomixStore) Create(change *networkchangetypes.NetworkChange) error {
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

	entry, err := s.changes.Put(ctx, string(change.ID), bytes, indexedmap.IfNotSet())
	if err != nil {
		return err
	}

	change.Index = networkchangetypes.Index(entry.Index)
	change.Revision = networkchangetypes.Revision(entry.Version)
	change.Created = entry.Created
	change.Updated = entry.Updated
	return nil
}

func (s *atomixStore) Update(change *networkchangetypes.NetworkChange) error {
	if change.Revision == 0 {
		return errors.New("not a stored object")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	bytes, err := proto.Marshal(change)
	if err != nil {
		return err
	}

	entry, err := s.changes.Put(ctx, string(change.ID), bytes, indexedmap.IfVersion(indexedmap.Version(change.Revision)))
	if err != nil {
		return err
	}

	change.Revision = networkchangetypes.Revision(entry.Version)
	change.Updated = entry.Updated
	return nil
}

func (s *atomixStore) Delete(change *networkchangetypes.NetworkChange) error {
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

func (s *atomixStore) List(ch chan<- *networkchangetypes.NetworkChange) (stream.Context, error) {
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
	return s.changes.Close()
}

func decodeChange(entry *indexedmap.Entry) (*networkchangetypes.NetworkChange, error) {
	change := &networkchangetypes.NetworkChange{}
	if err := proto.Unmarshal(entry.Value, change); err != nil {
		return nil, err
	}
	change.ID = networkchangetypes.ID(entry.Key)
	change.Index = networkchangetypes.Index(entry.Index)
	change.Revision = networkchangetypes.Revision(entry.Version)
	change.Created = entry.Created
	change.Updated = entry.Updated
	return change, nil
}
