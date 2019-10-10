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
	"github.com/atomix/atomix-go-client/pkg/client/counter"
	"github.com/atomix/atomix-go-client/pkg/client/map"
	"github.com/atomix/atomix-go-client/pkg/client/primitive"
	"github.com/atomix/atomix-go-client/pkg/client/session"
	"github.com/atomix/atomix-go-local/pkg/atomix/local"
	"github.com/atomix/atomix-go-node/pkg/atomix"
	"github.com/atomix/atomix-go-node/pkg/atomix/registry"
	"github.com/gogo/protobuf/proto"
	"github.com/onosproject/onos-config/pkg/store/utils"
	networksnapshot "github.com/onosproject/onos-config/pkg/types/snapshot/network"
	"google.golang.org/grpc"
	"google.golang.org/grpc/test/bufconn"
	"io"
	"net"
	"time"
)

const counterName = "network-snapshot-indexes"
const configsName = "network-snapshots"

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

	ids, err := group.GetCounter(context.Background(), counterName, session.WithTimeout(30*time.Second))
	if err != nil {
		return nil, err
	}

	configs, err := group.GetMap(context.Background(), configsName, session.WithTimeout(30*time.Second))
	if err != nil {
		return nil, err
	}

	return &atomixStore{
		indexes: ids,
		configs: configs,
		closer:  configs,
	}, nil
}

// NewLocalStore returns a new local network snapshot store
func NewLocalStore() (Store, error) {
	_, conn := startLocalNode()
	return newLocalStore(conn)
}

// newLocalStore creates a new local network snapshot store
func newLocalStore(conn *grpc.ClientConn) (Store, error) {
	counterName := primitive.Name{
		Namespace: "local",
		Name:      counterName,
	}
	ids, err := counter.New(context.Background(), counterName, []*grpc.ClientConn{conn})
	if err != nil {
		return nil, err
	}

	configsName := primitive.Name{
		Namespace: "local",
		Name:      configsName,
	}
	configs, err := _map.New(context.Background(), configsName, []*grpc.ClientConn{conn})
	if err != nil {
		return nil, err
	}

	return &atomixStore{
		indexes: ids,
		configs: configs,
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

	conn, err := grpc.DialContext(context.Background(), configsName, grpc.WithContextDialer(dialer), grpc.WithInsecure())
	if err != nil {
		panic("Failed to dial network configurations")
	}
	return node, conn
}

// Store stores NetworkConfig changes
type Store interface {
	io.Closer

	// LastIndex gets the last index in the store
	LastIndex() (networksnapshot.Index, error)

	// Get gets a network configuration
	Get(id networksnapshot.ID) (*networksnapshot.NetworkSnapshot, error)

	// GetByIndex gets a network change by index
	GetByIndex(index networksnapshot.Index) (*networksnapshot.NetworkSnapshot, error)

	// Create creates a new network configuration
	Create(config *networksnapshot.NetworkSnapshot) error

	// Update updates an existing network configuration
	Update(config *networksnapshot.NetworkSnapshot) error

	// Delete deletes a network configuration
	Delete(config *networksnapshot.NetworkSnapshot) error

	// List lists network configurations
	List(chan<- *networksnapshot.NetworkSnapshot) error

	// Watch watches the network configuration store for changes
	Watch(chan<- *networksnapshot.NetworkSnapshot) error
}

// atomixStore is the default implementation of the NetworkConfig store
type atomixStore struct {
	indexes counter.Counter
	configs _map.Map
	closer  io.Closer
}

func (s *atomixStore) nextIndex() (networksnapshot.Index, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	index, err := s.indexes.Increment(ctx, 1)
	if err != nil {
		return 0, err
	}
	return networksnapshot.Index(index), nil
}

func (s *atomixStore) LastIndex() (networksnapshot.Index, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	index, err := s.indexes.Get(ctx)
	if err != nil {
		return 0, err
	}
	return networksnapshot.Index(index), nil
}

func (s *atomixStore) Get(id networksnapshot.ID) (*networksnapshot.NetworkSnapshot, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	entry, err := s.configs.Get(ctx, string(id))
	if err != nil {
		return nil, err
	} else if entry == nil {
		return nil, nil
	}
	return decodeConfig(entry)
}

func (s *atomixStore) GetByIndex(index networksnapshot.Index) (*networksnapshot.NetworkSnapshot, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	entry, err := s.configs.Get(ctx, string(index.GetSnapshotID()))
	if err != nil {
		return nil, err
	} else if entry == nil {
		return nil, nil
	}
	return decodeConfig(entry)
}

func (s *atomixStore) Create(config *networksnapshot.NetworkSnapshot) error {
	if config.Revision != 0 {
		return errors.New("not a new object")
	}

	index, err := s.nextIndex()
	if err != nil {
		return err
	}

	config.Index = networksnapshot.Index(index)
	config.ID = config.Index.GetSnapshotID()

	bytes, err := proto.Marshal(config)
	if err != nil {
		return err
	}

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	entry, err := s.configs.Put(ctx, string(config.ID), bytes, _map.IfNotSet())
	if err != nil {
		return err
	}

	config.Revision = networksnapshot.Revision(entry.Version)
	config.Created = entry.Created
	config.Updated = entry.Updated
	return nil
}

func (s *atomixStore) Update(config *networksnapshot.NetworkSnapshot) error {
	if config.Revision == 0 {
		return errors.New("not a stored object")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	bytes, err := proto.Marshal(config)
	if err != nil {
		return err
	}

	entry, err := s.configs.Put(ctx, string(config.ID), bytes, _map.IfVersion(int64(config.Revision)))
	if err != nil {
		return err
	}

	config.Revision = networksnapshot.Revision(entry.Version)
	config.Updated = entry.Updated
	return nil
}

func (s *atomixStore) Delete(config *networksnapshot.NetworkSnapshot) error {
	if config.Revision == 0 {
		return errors.New("not a stored object")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	entry, err := s.configs.Remove(ctx, string(config.ID), _map.IfVersion(int64(config.Revision)))
	if err != nil {
		return err
	}

	config.Revision = 0
	config.Updated = entry.Updated
	return nil
}

func (s *atomixStore) List(ch chan<- *networksnapshot.NetworkSnapshot) error {
	lastIndex, err := s.LastIndex()
	if err != nil {
		return err
	}

	go func() {
		defer close(ch)
		for i := networksnapshot.Index(1); i <= lastIndex; i++ {
			if device, err := s.GetByIndex(i); err == nil && device != nil {
				ch <- device
			}
		}
	}()
	return nil
}

func (s *atomixStore) Watch(ch chan<- *networksnapshot.NetworkSnapshot) error {
	lastIndex, err := s.LastIndex()
	if err != nil {
		return err
	}

	mapCh := make(chan *_map.Event)
	if err := s.configs.Watch(context.Background(), mapCh); err != nil {
		return err
	}

	go func() {
		defer close(ch)
		for i := networksnapshot.Index(1); i <= lastIndex; i++ {
			if device, err := s.GetByIndex(i); err == nil && device != nil {
				ch <- device
			}
		}
		for event := range mapCh {
			if config, err := decodeConfig(event.Entry); err == nil {
				ch <- config
			}
		}
	}()
	return nil
}

func (s *atomixStore) Close() error {
	return s.configs.Close()
}

func decodeConfig(entry *_map.Entry) (*networksnapshot.NetworkSnapshot, error) {
	conf := &networksnapshot.NetworkSnapshot{}
	if err := proto.Unmarshal(entry.Value, conf); err != nil {
		return nil, err
	}
	conf.ID = networksnapshot.ID(entry.Key)
	conf.Revision = networksnapshot.Revision(entry.Version)
	conf.Created = entry.Created
	conf.Updated = entry.Updated
	return conf, nil
}
