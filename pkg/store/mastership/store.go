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
	"io"
	"sync"

	"github.com/atomix/go-client/pkg/client/util/net"
	"github.com/onosproject/onos-config/pkg/config"
	"github.com/onosproject/onos-lib-go/pkg/atomix"
	"github.com/onosproject/onos-lib-go/pkg/cluster"
	topodevice "github.com/onosproject/onos-topo/api/device"
)

// Term is a monotonically increasing mastership term
type Term uint64

// Store is the device mastership store
type Store interface {
	io.Closer

	// NodeID returns the local node identifier used in mastership elections
	NodeID() cluster.NodeID

	// GetMastership returns the mastership for a given device
	GetMastership(id topodevice.ID) (*Mastership, error)

	// Watch watches the store for mastership changes
	Watch(topodevice.ID, chan<- Mastership) error
}

// Mastership contains information about a device mastership term
type Mastership struct {
	// Device is the identifier of the device to which this mastership related
	Device topodevice.ID

	// Term is the mastership term
	Term Term

	// Master is the NodeID of the master for the device
	Master cluster.NodeID
}

// NewAtomixStore returns a new persistent Store
func NewAtomixStore(cluster cluster.Cluster, config config.Config) (Store, error) {
	database, err := atomix.GetDatabase(config.Atomix, config.Atomix.GetDatabase(atomix.DatabaseTypeConsensus))
	if err != nil {
		return nil, err
	}

	return &atomixStore{
		nodeID: cluster.Node().ID,
		newElection: func(id topodevice.ID) (deviceMastershipElection, error) {
			return newAtomixElection(cluster, id, database)
		},
		elections: make(map[topodevice.ID]deviceMastershipElection),
	}, nil
}

var localAddresses = make(map[string]net.Address)

// NewLocalStore returns a new local election store
func NewLocalStore(clusterID string, nodeID cluster.NodeID) (Store, error) {
	address, ok := localAddresses[clusterID]
	if !ok {
		_, address = atomix.StartLocalNode()
		localAddresses[clusterID] = address
	}
	return newLocalStore(nodeID, address)
}

// newLocalStore returns a new local device store
func newLocalStore(nodeID cluster.NodeID, address net.Address) (Store, error) {
	return &atomixStore{
		nodeID: nodeID,
		newElection: func(id topodevice.ID) (deviceMastershipElection, error) {
			return newLocalElection(id, nodeID, address)
		},
		elections: make(map[topodevice.ID]deviceMastershipElection),
	}, nil
}

// atomixStore is the default implementation of the NetworkConfig store
type atomixStore struct {
	nodeID      cluster.NodeID
	newElection func(topodevice.ID) (deviceMastershipElection, error)
	elections   map[topodevice.ID]deviceMastershipElection
	mu          sync.RWMutex
}

// getElection gets the mastership election for the given device
func (s *atomixStore) getElection(deviceID topodevice.ID) (deviceMastershipElection, error) {
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

func (s *atomixStore) GetMastership(deviceID topodevice.ID) (*Mastership, error) {
	election, err := s.getElection(deviceID)
	if err != nil {
		return nil, err
	}

	return election.getMastership(), nil
}

func (s *atomixStore) Watch(deviceID topodevice.ID, ch chan<- Mastership) error {
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
