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

package devicesnapshot

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
	devicesnapshottype "github.com/onosproject/onos-config/pkg/types/devicesnapshot"
	"google.golang.org/grpc"
	"google.golang.org/grpc/test/bufconn"
	"io"
	"net"
	"time"
)

const deviceRequestsName = "device-requests"

// NewAtomixDeviceRequestStore returns a new persistent Store
func NewAtomixDeviceRequestStore() (Store, error) {
	client, err := utils.GetAtomixClient()
	if err != nil {
		return nil, err
	}

	group, err := client.GetGroup(context.Background(), utils.GetAtomixRaftGroup())
	if err != nil {
		return nil, err
	}

	configs, err := group.GetMap(context.Background(), deviceRequestsName, session.WithTimeout(30*time.Second))
	if err != nil {
		return nil, err
	}

	return &atomixStore{
		configs: configs,
		closer:  configs,
	}, nil
}

// NewLocalDeviceRequestStore returns a new local device snapshot requests store
func NewLocalDeviceRequestStore() (Store, error) {
	node, conn := startLocalNode()
	configsName := primitive.Name{
		Namespace: "local",
		Name:      deviceRequestsName,
	}
	configs, err := _map.New(context.Background(), configsName, []*grpc.ClientConn{conn})
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

	conn, err := grpc.DialContext(context.Background(), deviceRequestsName, grpc.WithContextDialer(dialer), grpc.WithInsecure())
	if err != nil {
		panic("Failed to dial network configurations")
	}
	return node, conn
}

// Store is the interface for the device snapshot request store
type Store interface {
	io.Closer

	// Get gets a device snapshot request
	Get(id devicesnapshottype.ID) (*devicesnapshottype.DeviceSnapshot, error)

	// Create creates a new device snapshot request
	Create(config *devicesnapshottype.DeviceSnapshot) error

	// Update updates an existing device snapshot request
	Update(config *devicesnapshottype.DeviceSnapshot) error

	// Delete deletes a device snapshot request
	Delete(config *devicesnapshottype.DeviceSnapshot) error

	// List lists device snapshot request
	List(chan<- *devicesnapshottype.DeviceSnapshot) error

	// Watch watches the device snapshot request store for changes
	Watch(chan<- *devicesnapshottype.DeviceSnapshot) error
}

// atomixStore is the default implementation of the DeviceSnapshotRequest store
type atomixStore struct {
	configs _map.Map
	closer  io.Closer
}

func (s *atomixStore) Get(id devicesnapshottype.ID) (*devicesnapshottype.DeviceSnapshot, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	entry, err := s.configs.Get(ctx, string(id))
	if err != nil {
		return nil, err
	} else if entry == nil {
		return nil, nil
	}
	return decodeDeviceSnapshotRequest(entry)
}

func (s *atomixStore) Create(request *devicesnapshottype.DeviceSnapshot) error {
	if request.Revision != 0 {
		return errors.New("not a new object")
	}

	bytes, err := proto.Marshal(request)
	if err != nil {
		return err
	}

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	entry, err := s.configs.Put(ctx, string(request.ID), bytes, _map.IfNotSet())
	if err != nil {
		return err
	}

	request.Revision = devicesnapshottype.Revision(entry.Version)
	request.Created = entry.Created
	request.Updated = entry.Updated
	return nil
}

func (s *atomixStore) Update(request *devicesnapshottype.DeviceSnapshot) error {
	if request.Revision == 0 {
		return errors.New("not a stored object")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	bytes, err := proto.Marshal(request)
	if err != nil {
		return err
	}

	entry, err := s.configs.Put(ctx, string(request.ID), bytes, _map.IfVersion(int64(request.Revision)))
	if err != nil {
		return err
	}

	request.Revision = devicesnapshottype.Revision(entry.Version)
	request.Updated = entry.Updated
	return nil
}

func (s *atomixStore) Delete(request *devicesnapshottype.DeviceSnapshot) error {
	if request.Revision == 0 {
		return errors.New("not a stored object")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	entry, err := s.configs.Remove(ctx, string(request.ID), _map.IfVersion(int64(request.Revision)))
	if err != nil {
		return err
	}
	request.Updated = entry.Updated
	return nil
}

func (s *atomixStore) List(ch chan<- *devicesnapshottype.DeviceSnapshot) error {
	mapCh := make(chan *_map.Entry)
	if err := s.configs.Entries(context.Background(), mapCh); err != nil {
		return err
	}

	go func() {
		defer close(ch)
		for entry := range mapCh {
			if device, err := decodeDeviceSnapshotRequest(entry); err == nil {
				ch <- device
			}
		}
	}()
	return nil
}

func (s *atomixStore) Watch(ch chan<- *devicesnapshottype.DeviceSnapshot) error {
	mapCh := make(chan *_map.Event)
	if err := s.configs.Watch(context.Background(), mapCh, _map.WithReplay()); err != nil {
		return err
	}

	go func() {
		defer close(ch)
		for event := range mapCh {
			if request, err := decodeDeviceSnapshotRequest(event.Entry); err == nil {
				ch <- request
			}
		}
	}()
	return nil
}

func (s *atomixStore) Close() error {
	_ = s.configs.Close()
	return s.closer.Close()
}

func decodeDeviceSnapshotRequest(entry *_map.Entry) (*devicesnapshottype.DeviceSnapshot, error) {
	request := &devicesnapshottype.DeviceSnapshot{}
	if err := proto.Unmarshal(entry.Value, request); err != nil {
		return nil, err
	}
	request.ID = devicesnapshottype.ID(entry.Key)
	request.Revision = devicesnapshottype.Revision(entry.Version)
	request.Created = entry.Created
	request.Updated = entry.Updated
	return request, nil
}
