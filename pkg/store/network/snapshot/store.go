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

package snapshot

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
	"github.com/onosproject/onos-config/pkg/types/network/snapshot"
	"google.golang.org/grpc"
	"google.golang.org/grpc/test/bufconn"
	"io"
	"net"
	"time"
)

const networkRequestsIDsName = "network-request-ids"
const networkRequestsName = "network-requests"

// NewAtomixNetworkRequestStore returns a new persistent Store
func NewAtomixNetworkRequestStore() (Store, error) {
	client, err := utils.GetAtomixClient()
	if err != nil {
		return nil, err
	}

	group, err := client.GetGroup(context.Background(), utils.GetAtomixRaftGroup())
	if err != nil {
		return nil, err
	}

	ids, err := group.GetCounter(context.Background(), networkRequestsIDsName, session.WithTimeout(30*time.Second))
	if err != nil {
		return nil, err
	}

	configs, err := group.GetMap(context.Background(), networkRequestsName, session.WithTimeout(30*time.Second))
	if err != nil {
		return nil, err
	}

	return &atomixStore{
		ids:     ids,
		configs: configs,
		closer:  configs,
	}, nil
}

// NewLocalNetworkRequestStore returns a new local network snapshot requests store
func NewLocalNetworkRequestStore() (Store, error) {
	node, conn := startLocalNode()

	counterName := primitive.Name{
		Namespace: "local",
		Name:      networkRequestsIDsName,
	}
	ids, err := counter.New(context.Background(), counterName, []*grpc.ClientConn{conn})
	if err != nil {
		return nil, err
	}

	configsName := primitive.Name{
		Namespace: "local",
		Name:      networkRequestsName,
	}
	configs, err := _map.New(context.Background(), configsName, []*grpc.ClientConn{conn})
	if err != nil {
		return nil, err
	}

	return &atomixStore{
		ids:     ids,
		configs: configs,
		closer:  utils.NewNodeCloser(node),
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

	conn, err := grpc.DialContext(context.Background(), networkRequestsName, grpc.WithContextDialer(dialer), grpc.WithInsecure())
	if err != nil {
		panic("Failed to dial network configurations")
	}
	return node, conn
}

// Store is the interface for the network snapshot request store
type Store interface {
	io.Closer

	// Get gets a network snapshot request
	Get(id snapshot.ID) (*snapshot.NetworkSnapshot, error)

	// Create creates a new network snapshot request
	Create(config *snapshot.NetworkSnapshot) error

	// Update updates an existing network snapshot request
	Update(config *snapshot.NetworkSnapshot) error

	// Delete deletes a network snapshot request
	Delete(config *snapshot.NetworkSnapshot) error

	// List lists network snapshot request
	List(chan<- *snapshot.NetworkSnapshot) error

	// Watch watches the network snapshot request store for changes
	Watch(chan<- *snapshot.NetworkSnapshot) error
}

// atomixStore is the default implementation of the NetworkSnapshot store
type atomixStore struct {
	ids     counter.Counter
	configs _map.Map
	closer  io.Closer
}

func (s *atomixStore) Get(id snapshot.ID) (*snapshot.NetworkSnapshot, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	entry, err := s.configs.Get(ctx, string(id))
	if err != nil {
		return nil, err
	} else if entry == nil {
		return nil, nil
	}
	return decodeNetworkSnapshot(entry)
}

func (s *atomixStore) Create(config *snapshot.NetworkSnapshot) error {
	if config.Revision != 0 {
		return errors.New("not a new object")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	id, err := s.ids.Increment(ctx, 1)
	cancel()
	if err != nil {
		return err
	}

	config.Index = snapshot.Index(id)

	bytes, err := proto.Marshal(config)
	if err != nil {
		return err
	}

	ctx, cancel = context.WithTimeout(context.Background(), 15*time.Second)
	entry, err := s.configs.Put(ctx, string(config.ID), bytes, _map.IfNotSet())
	cancel()
	if err != nil {
		return err
	}

	config.Revision = snapshot.Revision(entry.Version)
	config.Created = entry.Created
	config.Updated = entry.Updated
	return nil
}

func (s *atomixStore) Update(config *snapshot.NetworkSnapshot) error {
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

	config.Revision = snapshot.Revision(entry.Version)
	config.Updated = entry.Updated
	return nil
}

func (s *atomixStore) Delete(config *snapshot.NetworkSnapshot) error {
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

func (s *atomixStore) List(ch chan<- *snapshot.NetworkSnapshot) error {
	mapCh := make(chan *_map.Entry)
	if err := s.configs.Entries(context.Background(), mapCh); err != nil {
		return err
	}

	go func() {
		defer close(ch)
		for entry := range mapCh {
			if device, err := decodeNetworkSnapshot(entry); err == nil {
				ch <- device
			}
		}
	}()
	return nil
}

func (s *atomixStore) Watch(ch chan<- *snapshot.NetworkSnapshot) error {
	mapCh := make(chan *_map.Event)
	if err := s.configs.Watch(context.Background(), mapCh, _map.WithReplay()); err != nil {
		return err
	}

	go func() {
		defer close(ch)
		for event := range mapCh {
			if config, err := decodeNetworkSnapshot(event.Entry); err == nil {
				ch <- config
			}
		}
	}()
	return nil
}

func (s *atomixStore) Close() error {
	_ = s.configs.Close()
	return s.closer.Close()
}

func decodeNetworkSnapshot(entry *_map.Entry) (*snapshot.NetworkSnapshot, error) {
	config := &snapshot.NetworkSnapshot{}
	if err := proto.Unmarshal(entry.Value, config); err != nil {
		return nil, err
	}
	config.ID = snapshot.ID(entry.Key)
	config.Revision = snapshot.Revision(entry.Version)
	config.Created = entry.Created
	config.Updated = entry.Updated
	return config, nil
}
