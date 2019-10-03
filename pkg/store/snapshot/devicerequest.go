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
	"github.com/atomix/atomix-go-client/pkg/client/map"
	"github.com/atomix/atomix-go-client/pkg/client/primitive"
	"github.com/atomix/atomix-go-client/pkg/client/session"
	"github.com/gogo/protobuf/proto"
	"github.com/onosproject/onos-config/pkg/store/utils"
	snapshottype "github.com/onosproject/onos-config/pkg/types/snapshot"
	"google.golang.org/grpc"
	"io"
	"time"
)

const deviceRequestsName = "device-requests"

// NewAtomixDeviceRequestStore returns a new persistent DeviceRequestStore
func NewAtomixDeviceRequestStore() (DeviceRequestStore, error) {
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

	return &atomixDeviceRequestStore{
		configs: configs,
		closer:  configs,
	}, nil
}

// NewLocalDeviceRequestStore returns a new local device snapshot requests store
func NewLocalDeviceRequestStore() (DeviceRequestStore, error) {
	node, conn := startLocalNode()
	configsName := primitive.Name{
		Namespace: "local",
		Name:      deviceRequestsName,
	}
	configs, err := _map.New(context.Background(), configsName, []*grpc.ClientConn{conn})
	if err != nil {
		return nil, err
	}

	return &atomixDeviceRequestStore{
		configs: configs,
		closer:  utils.NewNodeCloser(node),
	}, nil
}

// DeviceRequestStore is the interface for the device snapshot request store
type DeviceRequestStore interface {
	io.Closer

	// Get gets a device snapshot request
	Get(id snapshottype.ID) (*snapshottype.DeviceSnapshotRequest, error)

	// Create creates a new device snapshot request
	Create(config *snapshottype.DeviceSnapshotRequest) error

	// Update updates an existing device snapshot request
	Update(config *snapshottype.DeviceSnapshotRequest) error

	// Delete deletes a device snapshot request
	Delete(config *snapshottype.DeviceSnapshotRequest) error

	// List lists device snapshot request
	List(chan<- *snapshottype.DeviceSnapshotRequest) error

	// Watch watches the device snapshot request store for changes
	Watch(chan<- *snapshottype.DeviceSnapshotRequest) error
}

// atomixDeviceRequestStore is the default implementation of the DeviceSnapshotRequest store
type atomixDeviceRequestStore struct {
	configs _map.Map
	closer  io.Closer
}

func (s *atomixDeviceRequestStore) Get(id snapshottype.ID) (*snapshottype.DeviceSnapshotRequest, error) {
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

func (s *atomixDeviceRequestStore) Create(request *snapshottype.DeviceSnapshotRequest) error {
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

	request.Revision = snapshottype.Revision(entry.Version)
	request.Created = entry.Created
	request.Updated = entry.Updated
	return nil
}

func (s *atomixDeviceRequestStore) Update(request *snapshottype.DeviceSnapshotRequest) error {
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

	request.Revision = snapshottype.Revision(entry.Version)
	request.Updated = entry.Updated
	return nil
}

func (s *atomixDeviceRequestStore) Delete(request *snapshottype.DeviceSnapshotRequest) error {
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

func (s *atomixDeviceRequestStore) List(ch chan<- *snapshottype.DeviceSnapshotRequest) error {
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

func (s *atomixDeviceRequestStore) Watch(ch chan<- *snapshottype.DeviceSnapshotRequest) error {
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

func (s *atomixDeviceRequestStore) Close() error {
	_ = s.configs.Close()
	return s.closer.Close()
}

func decodeDeviceSnapshotRequest(entry *_map.Entry) (*snapshottype.DeviceSnapshotRequest, error) {
	request := &snapshottype.DeviceSnapshotRequest{}
	if err := proto.Unmarshal(entry.Value, request); err != nil {
		return nil, err
	}
	request.ID = snapshottype.ID(entry.Key)
	request.Revision = snapshottype.Revision(entry.Version)
	request.Created = entry.Created
	request.Updated = entry.Updated
	return request, nil
}
