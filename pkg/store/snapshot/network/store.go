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
	"github.com/onosproject/onos-config/pkg/store/cluster"
	"github.com/onosproject/onos-config/pkg/store/stream"
	"github.com/onosproject/onos-config/pkg/store/utils"
	networksnapshot "github.com/onosproject/onos-config/pkg/types/snapshot/network"
	"google.golang.org/grpc"
	"google.golang.org/grpc/test/bufconn"
	"io"
	"net"
	"time"
)

const snapshotsName = "network-snapshots"

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

	snapshots, err := group.GetIndexedMap(context.Background(), snapshotsName, session.WithTimeout(30*time.Second))
	if err != nil {
		return nil, err
	}

	return &atomixStore{
		snapshots: snapshots,
	}, nil
}

// NewLocalStore returns a new local network snapshot store
func NewLocalStore() (Store, error) {
	_, conn := startLocalNode()
	return newLocalStore(conn)
}

// newLocalStore creates a new local network snapshot store
func newLocalStore(conn *grpc.ClientConn) (Store, error) {
	snapshotsName := primitive.Name{
		Namespace: "local",
		Name:      snapshotsName,
	}
	snapshots, err := indexedmap.New(context.Background(), snapshotsName, []*grpc.ClientConn{conn})
	if err != nil {
		return nil, err
	}
	return &atomixStore{
		snapshots: snapshots,
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

	conn, err := grpc.DialContext(context.Background(), snapshotsName, grpc.WithContextDialer(dialer), grpc.WithInsecure())
	if err != nil {
		panic("Failed to dial")
	}
	return node, conn
}

// Store stores NetworkSnapshots
type Store interface {
	io.Closer

	// Get gets a network snapshot
	Get(id networksnapshot.ID) (*networksnapshot.NetworkSnapshot, error)

	// GetByIndex gets a network snapshot by index
	GetByIndex(index networksnapshot.Index) (*networksnapshot.NetworkSnapshot, error)

	// Create creates a new network snapshot
	Create(snapshot *networksnapshot.NetworkSnapshot) error

	// Update updates an existing network snapshot
	Update(snapshot *networksnapshot.NetworkSnapshot) error

	// Delete deletes a network snapshot
	Delete(snapshot *networksnapshot.NetworkSnapshot) error

	// List lists network snapshots
	List(chan<- *networksnapshot.NetworkSnapshot) (stream.Context, error)

	// Watch watches the network snapshot store for changes
	Watch(chan<- stream.Event) (stream.Context, error)
}

// newSnapshotID creates a new network snapshot ID
func newSnapshotID() networksnapshot.ID {
	return networksnapshot.ID(uuid.New().String())
}

// atomixStore is the default implementation of the NetworkSnapshot store
type atomixStore struct {
	snapshots indexedmap.IndexedMap
}

func (s *atomixStore) Get(id networksnapshot.ID) (*networksnapshot.NetworkSnapshot, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	entry, err := s.snapshots.Get(ctx, string(id))
	if err != nil {
		return nil, err
	} else if entry == nil {
		return nil, nil
	}
	return decodeSnapshot(entry)
}

func (s *atomixStore) GetByIndex(index networksnapshot.Index) (*networksnapshot.NetworkSnapshot, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	entry, err := s.snapshots.GetIndex(ctx, indexedmap.Index(index))
	if err != nil {
		return nil, err
	} else if entry == nil {
		return nil, nil
	}
	return decodeSnapshot(entry)
}

func (s *atomixStore) Create(snapshot *networksnapshot.NetworkSnapshot) error {
	if snapshot.ID == "" {
		snapshot.ID = newSnapshotID()
	}
	if snapshot.Revision != 0 {
		return errors.New("not a new object")
	}

	bytes, err := proto.Marshal(snapshot)
	if err != nil {
		return err
	}

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	entry, err := s.snapshots.Put(ctx, string(snapshot.ID), bytes, indexedmap.IfNotSet())
	if err != nil {
		return err
	}

	snapshot.Index = networksnapshot.Index(entry.Index)
	snapshot.Revision = networksnapshot.Revision(entry.Version)
	snapshot.Created = entry.Created
	snapshot.Updated = entry.Updated
	return nil
}

func (s *atomixStore) Update(snapshot *networksnapshot.NetworkSnapshot) error {
	if snapshot.Revision == 0 {
		return errors.New("not a stored object")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	bytes, err := proto.Marshal(snapshot)
	if err != nil {
		return err
	}

	entry, err := s.snapshots.Put(ctx, string(snapshot.ID), bytes, indexedmap.IfVersion(indexedmap.Version(snapshot.Revision)))
	if err != nil {
		return err
	}

	snapshot.Revision = networksnapshot.Revision(entry.Version)
	snapshot.Updated = entry.Updated
	return nil
}

func (s *atomixStore) Delete(snapshot *networksnapshot.NetworkSnapshot) error {
	if snapshot.Revision == 0 {
		return errors.New("not a stored object")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	entry, err := s.snapshots.RemoveIndex(ctx, indexedmap.Index(snapshot.Index), indexedmap.IfVersion(indexedmap.Version(snapshot.Revision)))
	if err != nil {
		return err
	}

	snapshot.Revision = 0
	snapshot.Updated = entry.Updated
	return nil
}

func (s *atomixStore) List(ch chan<- *networksnapshot.NetworkSnapshot) (stream.Context, error) {
	ctx, cancel := context.WithCancel(context.Background())

	mapCh := make(chan *indexedmap.Entry)
	if err := s.snapshots.Entries(ctx, mapCh); err != nil {
		cancel()
		return nil, err
	}

	go func() {
		defer close(ch)
		for entry := range mapCh {
			if snapshot, err := decodeSnapshot(entry); err == nil {
				ch <- snapshot
			}
		}
	}()
	return stream.NewCancelContext(cancel), nil
}

func (s *atomixStore) Watch(ch chan<- stream.Event) (stream.Context, error) {
	ctx, cancel := context.WithCancel(context.Background())

	mapCh := make(chan *indexedmap.Event)
	if err := s.snapshots.Watch(ctx, mapCh, indexedmap.WithReplay()); err != nil {
		cancel()
		return nil, err
	}

	go func() {
		defer close(ch)
		for event := range mapCh {
			if snapshot, err := decodeSnapshot(event.Entry); err == nil {
				switch event.Type {
				case indexedmap.EventNone:
					ch <- stream.Event{
						Type:   stream.None,
						Object: snapshot,
					}
				case indexedmap.EventInserted:
					ch <- stream.Event{
						Type:   stream.Created,
						Object: snapshot,
					}
				case indexedmap.EventUpdated:
					ch <- stream.Event{
						Type:   stream.Updated,
						Object: snapshot,
					}
				case indexedmap.EventRemoved:
					ch <- stream.Event{
						Type:   stream.Deleted,
						Object: snapshot,
					}
				}
			}
		}
	}()
	return stream.NewCancelContext(cancel), nil
}

func (s *atomixStore) Close() error {
	return s.snapshots.Close()
}

func decodeSnapshot(entry *indexedmap.Entry) (*networksnapshot.NetworkSnapshot, error) {
	snapshot := &networksnapshot.NetworkSnapshot{}
	if err := proto.Unmarshal(entry.Value, snapshot); err != nil {
		return nil, err
	}
	snapshot.ID = networksnapshot.ID(entry.Key)
	snapshot.Index = networksnapshot.Index(entry.Index)
	snapshot.Revision = networksnapshot.Revision(entry.Version)
	snapshot.Created = entry.Created
	snapshot.Updated = entry.Updated
	return snapshot, nil
}
