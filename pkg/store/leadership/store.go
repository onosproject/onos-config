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
	"github.com/onosproject/onos-lib-go/pkg/errors"
	"io"
	"sync"
	"time"

	"github.com/atomix/atomix-go-client/pkg/atomix"
	"github.com/atomix/atomix-go-client/pkg/atomix/election"
)

// Term is a monotonically increasing leadership term
type Term uint64

// Store is the cluster wide leadership store
type Store interface {
	io.Closer

	// NodeID returns the local node identifier used in the election
	NodeID() string

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
	Leader string
}

// NewAtomixStore returns a new persistent Store
func NewAtomixStore(client atomix.Client) (Store, error) {
	election, err := client.GetElection(context.Background(), "onos-config-leaderships")
	if err != nil {
		return nil, errors.FromAtomix(err)
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
	ch := make(chan election.Event)
	if err := s.election.Watch(context.Background(), ch); err != nil {
		return errors.FromAtomix(err)
	}

	// Enter the election to get the current leadership term
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	term, err := s.election.Enter(ctx)
	cancel()
	if err != nil {
		_ = s.election.Close(context.Background())
		return errors.FromAtomix(err)
	}

	// Set the leadership term
	s.mu.Lock()
	s.leadership = &Leadership{
		Term:   Term(term.Revision),
		Leader: term.Leader,
	}
	s.mu.Unlock()

	// Wait for the election event to be received before returning
	for event := range ch {
		if event.Term.Revision == term.Revision {
			go s.watchElection(ch)
			return nil
		}
	}

	_ = s.election.Close(context.Background())
	return errors.NewUnavailable("failed to enter election")
}

// watchElection watches the election events and updates leadership info
func (s *atomixStore) watchElection(ch <-chan election.Event) {
	for event := range ch {
		var leadership *Leadership
		s.mu.Lock()
		if s.leadership.Term != Term(event.Term.Revision) {
			leadership = &Leadership{
				Term:   Term(event.Term.Revision),
				Leader: event.Term.Leader,
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

func (s *atomixStore) NodeID() string {
	return s.election.ID()
}

func (s *atomixStore) IsLeader() (bool, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if s.leadership == nil || s.leadership.Leader != s.election.ID() {
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
	err := s.election.Close(context.Background())
	if err != nil {
		return errors.FromAtomix(err)
	}
	return nil
}
