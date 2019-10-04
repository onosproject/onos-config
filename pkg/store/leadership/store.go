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
	"github.com/atomix/atomix-go-client/pkg/client/election"
	"github.com/atomix/atomix-go-client/pkg/client/primitive"
	"github.com/atomix/atomix-go-client/pkg/client/session"
	"github.com/atomix/atomix-go-local/pkg/atomix/local"
	"github.com/atomix/atomix-go-node/pkg/atomix"
	"github.com/atomix/atomix-go-node/pkg/atomix/registry"
	"github.com/onosproject/onos-config/pkg/store/cluster"
	"github.com/onosproject/onos-config/pkg/store/utils"
	"google.golang.org/grpc"
	"google.golang.org/grpc/test/bufconn"
	"io"
	"net"
	"sync"
	"time"
)

const primitiveName = "leaderships"

// Term is a monotonically increasing leadership term
type Term uint64

// Store is the cluster wide leadership store
type Store interface {
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
func NewAtomixStore() (Store, error) {
	client, err := utils.GetAtomixClient()
	if err != nil {
		return nil, err
	}

	group, err := client.GetGroup(context.Background(), utils.GetAtomixRaftGroup())
	if err != nil {
		return nil, err
	}

	election, err := group.GetElection(context.Background(), primitiveName, session.WithTimeout(30*time.Second))
	if err != nil {
		return nil, err
	}

	return &atomixStore{
		election: election,
		closer:   election,
	}, nil
}

// NewLocalStore returns a new local device store
func NewLocalStore() (Store, error) {
	node, conn := startLocalNode()
	name := primitive.Name{
		Namespace: "local",
		Name:      primitiveName,
	}

	election, err := election.New(context.Background(), name, []*grpc.ClientConn{conn})
	if err != nil {
		return nil, err
	}

	return &atomixStore{
		election: election,
		closer:   utils.NewNodeCloser(node),
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

// atomixStore is the default implementation of the NetworkConfig store
type atomixStore struct {
	election   election.Election
	closer     io.Closer
	leadership *Leadership
	mu         sync.RWMutex
}

func (s *atomixStore) IsLeader() (bool, error) {
	if s.leadership == nil {
		ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
		defer cancel()

		term, err := s.election.GetTerm(ctx)
		if err != nil {
			return false, err
		} else if term != nil {
			s.mu.Lock()
			s.leadership = &Leadership{
				Term:   Term(term.ID),
				Leader: cluster.NodeID(term.Leader),
			}
			s.mu.Unlock()
		}
	}

	s.mu.RLock()
	defer s.mu.RUnlock()
	if s.leadership == nil || s.leadership.Leader != cluster.GetNodeID() {
		return false, nil
	}
	return true, nil
}

func (s *atomixStore) Watch(ch chan<- Leadership) error {
	electionCh := make(chan *election.Event)
	go func() {
		for event := range electionCh {
			leadership := Leadership{
				Term:   Term(event.Term.ID),
				Leader: cluster.NodeID(event.Term.Leader),
			}
			s.mu.Lock()
			s.leadership = &leadership
			s.mu.Unlock()
			ch <- leadership
		}
	}()
	return s.election.Watch(context.Background(), electionCh)
}

func (s *atomixStore) Close() error {
	_ = s.election.Close()
	return s.closer.Close()
}
