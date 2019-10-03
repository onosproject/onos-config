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
	"github.com/gogo/protobuf/proto"
	"github.com/onosproject/onos-config/pkg/store/utils"
	snapshottype "github.com/onosproject/onos-config/pkg/types/snapshot"
	"google.golang.org/grpc"
	"io"
	"time"
)

const networkRequestsIDsName = "network-request-ids"
const networkRequestsName = "network-requests"

// NewAtomixNetworkRequestStore returns a new persistent NetworkRequestStore
func NewAtomixNetworkRequestStore() (NetworkRequestStore, error) {
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

	return &atomixNetworkRequestStore{
		ids:     ids,
		configs: configs,
		closer:  configs,
	}, nil
}

// NewLocalNetworkRequestStore returns a new local network snapshot requests store
func NewLocalNetworkRequestStore() (NetworkRequestStore, error) {
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

	return &atomixNetworkRequestStore{
		ids:     ids,
		configs: configs,
		closer:  utils.NewNodeCloser(node),
	}, nil
}

// NetworkRequestStore is the interface for the network snapshot request store
type NetworkRequestStore interface {
	io.Closer

	// Get gets a network snapshot request
	Get(id snapshottype.ID) (*snapshottype.NetworkSnapshotRequest, error)

	// Create creates a new network snapshot request
	Create(config *snapshottype.NetworkSnapshotRequest) error

	// Update updates an existing network snapshot request
	Update(config *snapshottype.NetworkSnapshotRequest) error

	// Delete deletes a network snapshot request
	Delete(config *snapshottype.NetworkSnapshotRequest) error

	// List lists network snapshot request
	List(chan<- *snapshottype.NetworkSnapshotRequest) error

	// Watch watches the network snapshot request store for changes
	Watch(chan<- *snapshottype.NetworkSnapshotRequest) error
}

// atomixNetworkRequestStore is the default implementation of the NetworkSnapshotRequest store
type atomixNetworkRequestStore struct {
	ids     counter.Counter
	configs _map.Map
	closer  io.Closer
}

func (s *atomixNetworkRequestStore) Get(id snapshottype.ID) (*snapshottype.NetworkSnapshotRequest, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	entry, err := s.configs.Get(ctx, string(id))
	if err != nil {
		return nil, err
	} else if entry == nil {
		return nil, nil
	}
	return decodeNetworkSnapshotRequest(entry)
}

func (s *atomixNetworkRequestStore) Create(request *snapshottype.NetworkSnapshotRequest) error {
	if request.Revision != 0 {
		return errors.New("not a new object")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	id, err := s.ids.Increment(ctx, 1)
	cancel()
	if err != nil {
		return err
	}

	request.Index = snapshottype.Index(id)

	bytes, err := proto.Marshal(request)
	if err != nil {
		return err
	}

	ctx, cancel = context.WithTimeout(context.Background(), 15*time.Second)
	entry, err := s.configs.Put(ctx, string(request.ID), bytes, _map.IfNotSet())
	cancel()
	if err != nil {
		return err
	}

	request.Revision = snapshottype.Revision(entry.Version)
	request.Created = entry.Created
	request.Updated = entry.Updated
	return nil
}

func (s *atomixNetworkRequestStore) Update(request *snapshottype.NetworkSnapshotRequest) error {
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

func (s *atomixNetworkRequestStore) Delete(request *snapshottype.NetworkSnapshotRequest) error {
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

func (s *atomixNetworkRequestStore) List(ch chan<- *snapshottype.NetworkSnapshotRequest) error {
	mapCh := make(chan *_map.Entry)
	if err := s.configs.Entries(context.Background(), mapCh); err != nil {
		return err
	}

	go func() {
		defer close(ch)
		for entry := range mapCh {
			if device, err := decodeNetworkSnapshotRequest(entry); err == nil {
				ch <- device
			}
		}
	}()
	return nil
}

func (s *atomixNetworkRequestStore) Watch(ch chan<- *snapshottype.NetworkSnapshotRequest) error {
	mapCh := make(chan *_map.Event)
	if err := s.configs.Watch(context.Background(), mapCh, _map.WithReplay()); err != nil {
		return err
	}

	go func() {
		defer close(ch)
		for event := range mapCh {
			if request, err := decodeNetworkSnapshotRequest(event.Entry); err == nil {
				ch <- request
			}
		}
	}()
	return nil
}

func (s *atomixNetworkRequestStore) Close() error {
	_ = s.configs.Close()
	return s.closer.Close()
}

func decodeNetworkSnapshotRequest(entry *_map.Entry) (*snapshottype.NetworkSnapshotRequest, error) {
	request := &snapshottype.NetworkSnapshotRequest{}
	if err := proto.Unmarshal(entry.Value, request); err != nil {
		return nil, err
	}
	request.ID = snapshottype.ID(entry.Key)
	request.Revision = snapshottype.Revision(entry.Version)
	request.Created = entry.Created
	request.Updated = entry.Updated
	return request, nil
}
