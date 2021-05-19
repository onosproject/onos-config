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
	"github.com/atomix/atomix-go-client/pkg/atomix"
	"github.com/atomix/atomix-go-client/pkg/atomix/primitive"
	"io"
	"sync"

	"github.com/onosproject/onos-config/pkg/device"
	"github.com/onosproject/onos-lib-go/pkg/cluster"
)

// Term is a monotonically increasing mastership term
type Term uint64

// Store is the device mastership store
type Store interface {
	io.Closer

	// NodeID returns the local node identifier used in mastership elections
	NodeID() cluster.NodeID

	// GetMastership returns the mastership for a given device
	GetMastership(id device.ID) (*Mastership, error)

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
func NewAtomixStore(client atomix.Client, nodeID cluster.NodeID) (Store, error) {
	return &atomixStore{
		nodeID: nodeID,
		newElection: func(id device.ID) (deviceMastershipElection, error) {
			election, err := client.GetElection(
				context.Background(),
				"onos-config-masterships",
				primitive.WithSessionID(string(nodeID)),
				primitive.WithClusterKey(string(id)))
			if err != nil {
				return nil, err
			}
			return newDeviceMastershipElection(id, election)
		},
		elections: make(map[device.ID]deviceMastershipElection),
	}, nil
}

// atomixStore is the default implementation of the NetworkConfig store
type atomixStore struct {
	nodeID      cluster.NodeID
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

func (s *atomixStore) NodeID() cluster.NodeID {
	return s.nodeID
}

func (s *atomixStore) GetMastership(deviceID device.ID) (*Mastership, error) {
	election, err := s.getElection(deviceID)
	if err != nil {
		return nil, err
	}

	return election.getMastership(), nil
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
