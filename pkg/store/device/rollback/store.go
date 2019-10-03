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

package rollback

import (
	"context"
	"errors"
	"github.com/atomix/atomix-go-client/pkg/client/map"
	"github.com/atomix/atomix-go-client/pkg/client/primitive"
	"github.com/atomix/atomix-go-client/pkg/client/session"
	"github.com/atomix/atomix-go-local/pkg/atomix/local"
	"github.com/atomix/atomix-go-node/pkg/atomix"
	"github.com/atomix/atomix-go-node/pkg/atomix/registry"
	"github.com/gogo/protobuf/proto"
	"github.com/onosproject/onos-config/pkg/store/utils"
	"github.com/onosproject/onos-config/pkg/types/device/rollback"
	"google.golang.org/grpc"
	"google.golang.org/grpc/test/bufconn"
	"io"
	"net"
	"time"
)

const primitiveName = "device-rollback"

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

	configs, err := group.GetMap(context.Background(), primitiveName, session.WithTimeout(30*time.Second))
	if err != nil {
		return nil, err
	}

	return &atomixStore{
		configs: configs,
		closer:  configs,
	}, nil
}

// NewLocalStore returns a new local device store
func NewLocalStore() (Store, error) {
	node, conn := startLocalNode()
	name := primitive.Name{
		Namespace: "local",
		Name:      primitiveName,
	}

	configs, err := _map.New(context.Background(), name, []*grpc.ClientConn{conn})
	if err != nil {
		return nil, err
	}

	return &atomixStore{
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

	conn, err := grpc.DialContext(context.Background(), primitiveName, grpc.WithContextDialer(dialer), grpc.WithInsecure())
	if err != nil {
		panic("Failed to dial network configurations")
	}
	return node, conn
}

// Store stores device Rollbacks
type Store interface {
	io.Closer

	// Get gets a device rollback
	Get(id rollback.ID) (*rollback.Rollback, error)

	// Create creates a new device rollback
	Create(config *rollback.Rollback) error

	// Update updates an existing device rollback
	Update(config *rollback.Rollback) error

	// Delete deletes a device rollback
	Delete(config *rollback.Rollback) error

	// List lists device rollback
	List(chan<- *rollback.Rollback) error

	// Watch watches the device rollback store for rollbacks
	Watch(chan<- *rollback.Rollback) error
}

// atomixStore is the default implementation of the NetworkConfig store
type atomixStore struct {
	configs _map.Map
	closer  io.Closer
}

func (s *atomixStore) Get(id rollback.ID) (*rollback.Rollback, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	entry, err := s.configs.Get(ctx, string(id))
	if err != nil {
		return nil, err
	} else if entry == nil {
		return nil, nil
	}
	return decodeRollback(entry)
}

func (s *atomixStore) Create(config *rollback.Rollback) error {
	if config.Revision != 0 {
		return errors.New("not a new object")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	config.Created = time.Now()
	config.Updated = time.Now()
	bytes, err := proto.Marshal(config)
	if err != nil {
		return err
	}

	entry, err := s.configs.Put(ctx, string(config.ID), bytes, _map.IfNotSet())
	if err != nil {
		return err
	}

	config.Revision = rollback.Revision(entry.Version)
	config.Created = entry.Created
	config.Updated = entry.Updated
	return nil
}

func (s *atomixStore) Update(config *rollback.Rollback) error {
	if config.Revision == 0 {
		return errors.New("not a stored object")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	config.Updated = time.Now()
	bytes, err := proto.Marshal(config)
	if err != nil {
		return err
	}

	entry, err := s.configs.Put(ctx, string(config.ID), bytes, _map.IfVersion(int64(config.Revision)))
	if err != nil {
		return err
	}

	config.Revision = rollback.Revision(entry.Version)
	config.Updated = entry.Updated
	return nil
}

func (s *atomixStore) Delete(config *rollback.Rollback) error {
	if config.Revision == 0 {
		return errors.New("not a stored object")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	entry, err := s.configs.Remove(ctx, string(config.ID), _map.IfVersion(int64(config.Revision)))
	if err != nil {
		return err
	}
	config.Updated = entry.Updated
	return nil
}

func (s *atomixStore) List(ch chan<- *rollback.Rollback) error {
	mapCh := make(chan *_map.Entry)
	if err := s.configs.Entries(context.Background(), mapCh); err != nil {
		return err
	}

	go func() {
		defer close(ch)
		for entry := range mapCh {
			if device, err := decodeRollback(entry); err == nil {
				ch <- device
			}
		}
	}()
	return nil
}

func (s *atomixStore) Watch(ch chan<- *rollback.Rollback) error {
	mapCh := make(chan *_map.Event)
	if err := s.configs.Watch(context.Background(), mapCh, _map.WithReplay()); err != nil {
		return err
	}

	go func() {
		defer close(ch)
		for event := range mapCh {
			if config, err := decodeRollback(event.Entry); err == nil {
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

func decodeRollback(entry *_map.Entry) (*rollback.Rollback, error) {
	config := &rollback.Rollback{}
	if err := proto.Unmarshal(entry.Value, config); err != nil {
		return nil, err
	}
	config.ID = rollback.ID(entry.Key)
	config.Revision = rollback.Revision(entry.Version)
	config.Created = entry.Created
	config.Updated = entry.Updated
	return config, nil
}
