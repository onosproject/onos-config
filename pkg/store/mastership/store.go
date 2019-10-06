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

package mastership

import (
	"context"
	"github.com/atomix/atomix-go-local/pkg/atomix/local"
	"github.com/atomix/atomix-go-node/pkg/atomix"
	"github.com/atomix/atomix-go-node/pkg/atomix/registry"
	"github.com/onosproject/onos-config/pkg/store/cluster"
	"github.com/onosproject/onos-config/pkg/store/utils"
	"github.com/onosproject/onos-topo/pkg/northbound/device"
	"google.golang.org/grpc"
	"google.golang.org/grpc/test/bufconn"
	"io"
	"net"
	"sync"
)

// Term is a monotonically increasing mastership term
type Term uint64

// Store is the device mastership store
type Store interface {
	io.Closer

	// IsMaster returns a boolean indicating whether the local node is the master for the given device
	IsMaster(id device.ID) (bool, error)

	// Watch watches the store for mastership changes
	Watch(device.ID, chan<- Mastership) error
}

// Mastership contains information about a device mastership term
type Mastership struct {
	// Device is the identifier of the device to which this mastership related
	Device device.ID

	// Term is the mastership term
	Term Term

	// Master is the NodeID of the master for the device
	Master cluster.NodeID
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

	return &atomixStore{
		newElection: func(id device.ID) (deviceMastershipElection, error) {
			return newAtomixElection(id, group)
		},
		elections: make(map[device.ID]deviceMastershipElection),
	}, nil
}

// NewLocalStore returns a new local election store
func NewLocalStore(nodeID cluster.NodeID) (Store, error) {
	_, conn := startLocalNode()
	return newLocalStore(nodeID, conn)
}

// newLocalStore returns a new local device store
func newLocalStore(nodeID cluster.NodeID, conn *grpc.ClientConn) (Store, error) {
	return &atomixStore{
		newElection: func(id device.ID) (deviceMastershipElection, error) {
			return newLocalElection(id, nodeID, conn)
		},
		elections: make(map[device.ID]deviceMastershipElection),
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

	conn, err := grpc.DialContext(context.Background(), "mastership", grpc.WithContextDialer(dialer), grpc.WithInsecure())
	if err != nil {
		panic("Failed to dial network configurations")
	}
	return node, conn
}

// atomixStore is the default implementation of the NetworkConfig store
type atomixStore struct {
	newElection func(device.ID) (deviceMastershipElection, error)
	elections   map[device.ID]deviceMastershipElection
	mu          sync.RWMutex
}

// getElection gets the mastership election for the given device
func (s *atomixStore) getElection(deviceID device.ID) (deviceMastershipElection, error) {
	s.mu.RLock()
	election, ok := s.elections[deviceID]
	s.mu.RUnlock()
	if !ok {
		s.mu.Lock()
		election, ok = s.elections[deviceID]
		if !ok {
			e, err := s.newElection(deviceID)
			if err != nil {
				s.mu.Unlock()
				return nil, err
			}
			election = e
			s.elections[deviceID] = election
		}
		s.mu.Unlock()
	}
	return election, nil
}

func (s *atomixStore) IsMaster(deviceID device.ID) (bool, error) {
	election, err := s.getElection(deviceID)
	if err != nil {
		return false, err
	}
	return election.isMaster()
}

func (s *atomixStore) Watch(deviceID device.ID, ch chan<- Mastership) error {
	election, err := s.getElection(deviceID)
	if err != nil {
		return err
	}
	return election.watch(ch)
}

func (s *atomixStore) Close() error {
	var returnErr error
	for _, election := range s.elections {
		if err := election.Close(); err != nil && returnErr == nil {
			returnErr = err
		}
	}
	return returnErr
}

var _ Store = &atomixStore{}
