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

package leadership

import (
	"context"
	"errors"
	"io"
	"sync"
	"time"

	"github.com/atomix/go-client/pkg/client/election"
	"github.com/atomix/go-client/pkg/client/primitive"
	"github.com/atomix/go-client/pkg/client/util/net"
	"github.com/onosproject/onos-config/pkg/config"
	"github.com/onosproject/onos-lib-go/pkg/atomix"
	"github.com/onosproject/onos-lib-go/pkg/cluster"
)

const primitiveName = "leaderships"

// Term is a monotonically increasing leadership term
type Term uint64

// Store is the cluster wide leadership store
type Store interface {
	io.Closer

	// NodeID returns the local node identifier used in the election
	NodeID() cluster.NodeID

	// IsLeader returns a boolean indicating whether the local node is the leader
	IsLeader() (bool, error)

	// Watch watches the store for changes
	Watch(chan<- Leadership) error
}

// Leadership contains information about a leadership term
type Leadership struct {
	// Term is the leadership term
	Term Term

	// Leader is the NodeID of the leader
	Leader cluster.NodeID
}

// NewAtomixStore returns a new persistent Store
func NewAtomixStore(cluster cluster.Cluster, config config.Config) (Store, error) {
	database, err := atomix.GetDatabase(config.Atomix, config.Atomix.GetDatabase(atomix.DatabaseTypeConsensus))
	if err != nil {
		return nil, err
	}

	election, err := database.GetElection(context.Background(), primitiveName, election.WithID(string(cluster.Node().ID)))
	if err != nil {
		return nil, err
	}

	store := &atomixStore{
		election: election,
		watchers: make([]chan<- Leadership, 0, 1),
	}
	if err := store.enter(); err != nil {
		return nil, err
	}
	return store, nil
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

// newLocalStore returns a new local election store
func newLocalStore(nodeID cluster.NodeID, address net.Address) (Store, error) {
	name := primitive.Name{
		Namespace: "local",
		Name:      primitiveName,
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	session, err := primitive.NewSession(ctx, primitive.Partition{ID: 1, Address: address})
	if err != nil {
		return nil, err
	}

	election, err := election.New(context.Background(), name, []*primitive.Session{session}, election.WithID(string(nodeID)))
	if err != nil {
		return nil, err
	}

	store := &atomixStore{
		election: election,
		watchers: make([]chan<- Leadership, 0, 1),
	}
	if err := store.enter(); err != nil {
		return nil, err
	}
	return store, nil
}

// atomixStore is the default implementation of the NetworkConfig store
type atomixStore struct {
	election   election.Election
	leadership *Leadership
	watchers   []chan<- Leadership
	mu         sync.RWMutex
}

// enter enters the election
func (s *atomixStore) enter() error {
	ch := make(chan *election.Event)
	if err := s.election.Watch(context.Background(), ch); err != nil {
		return err
	}

	// Enter the election to get the current leadership term
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	term, err := s.election.Enter(ctx)
	cancel()
	if err != nil {
		_ = s.election.Close(context.Background())
		return err
	}

	// Set the leadership term
	s.mu.Lock()
	s.leadership = &Leadership{
		Term:   Term(term.ID),
		Leader: cluster.NodeID(term.Leader),
	}
	s.mu.Unlock()

	// Wait for the election event to be received before returning
	for event := range ch {
		if event.Term.ID == term.ID {
			go s.watchElection(ch)
			return nil
		}
	}

	_ = s.election.Close(context.Background())
	return errors.New("failed to enter election")
}

// watchElection watches the election events and updates leadership info
func (s *atomixStore) watchElection(ch <-chan *election.Event) {
	for event := range ch {
		var leadership *Leadership
		s.mu.Lock()
		if uint64(s.leadership.Term) != event.Term.ID {
			leadership = &Leadership{
				Term:   Term(event.Term.ID),
				Leader: cluster.NodeID(event.Term.Leader),
			}
			s.leadership = leadership
		}
		s.mu.Unlock()

		if leadership != nil {
			s.mu.RLock()
			for _, watcher := range s.watchers {
				watcher <- *leadership
			}
			s.mu.RUnlock()
		}
	}
}

func (s *atomixStore) NodeID() cluster.NodeID {
	return cluster.NodeID(s.election.ID())
}

func (s *atomixStore) IsLeader() (bool, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if s.leadership == nil || string(s.leadership.Leader) != s.election.ID() {
		return false, nil
	}
	return true, nil
}

func (s *atomixStore) Watch(ch chan<- Leadership) error {
	s.mu.Lock()
	s.watchers = append(s.watchers, ch)
	s.mu.Unlock()
	return nil
}

func (s *atomixStore) Close() error {
	return s.election.Close(context.Background())
}
