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
	"github.com/onosproject/onos-lib-go/pkg/errors"
	"io"
	"sync"
	"time"

	"github.com/atomix/atomix-go-client/pkg/atomix/election"
	topodevice "github.com/onosproject/onos-config/pkg/device"
	"github.com/onosproject/onos-lib-go/pkg/cluster"
)

// newDeviceMastershipElection creates and enters a new device mastership election
func newDeviceMastershipElection(deviceID topodevice.ID, election election.Election) (deviceMastershipElection, error) {
	deviceElection := &atomixDeviceMastershipElection{
		deviceID: deviceID,
		election: election,
		watchers: make([]chan<- Mastership, 0, 1),
	}
	if err := deviceElection.enter(); err != nil {
		return nil, err
	}
	return deviceElection, nil
}

// deviceMastershipElection is an election for a single device mastership
type deviceMastershipElection interface {
	io.Closer

	// NodeID returns the local node identifier used in the election
	NodeID() cluster.NodeID

	// DeviceID returns the device for which this election provides mastership
	DeviceID() topodevice.ID

	// getMastership returns the mastership info
	getMastership() *Mastership

	// watch watches the election for changes
	watch(ch chan<- Mastership) error
}

// atomixDeviceMastershipElection is a persistent device mastership election
type atomixDeviceMastershipElection struct {
	deviceID   topodevice.ID
	election   election.Election
	mastership *Mastership
	watchers   []chan<- Mastership
	mu         sync.RWMutex
}

func (e *atomixDeviceMastershipElection) NodeID() cluster.NodeID {
	return cluster.NodeID(e.election.ID())
}

func (e *atomixDeviceMastershipElection) DeviceID() topodevice.ID {
	return e.deviceID
}

// enter enters the election
func (e *atomixDeviceMastershipElection) enter() error {
	ch := make(chan election.Event)
	if err := e.election.Watch(context.Background(), ch); err != nil {
		return errors.FromAtomix(err)
	}

	// Enter the election to get the current leadership term
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	term, err := e.election.Enter(ctx)
	cancel()
	if err != nil {
		_ = e.election.Close(context.Background())
		return errors.FromAtomix(err)
	}

	// Set the mastership term
	e.mu.Lock()
	e.mastership = &Mastership{
		Device: e.deviceID,
		Master: cluster.NodeID(term.Leader),
		Term:   Term(term.Revision),
	}
	e.mu.Unlock()
	go e.watchElection(*term, ch)
	return nil
}

// watchElection watches the election events and updates mastership info
func (e *atomixDeviceMastershipElection) watchElection(term election.Term, ch <-chan election.Event) {
	for event := range ch {
		// Ignore events that occurred prior to entering the election
		if event.Term.Revision < term.Revision {
			continue
		}

		var mastership *Mastership
		e.mu.Lock()
		if e.mastership.Term != Term(event.Term.Revision) {
			mastership = &Mastership{
				Device: e.deviceID,
				Term:   Term(event.Term.Revision),
				Master: cluster.NodeID(event.Term.Leader),
			}
			e.mastership = mastership
		}
		e.mu.Unlock()

		if mastership != nil {
			e.mu.RLock()
			for _, watcher := range e.watchers {
				watcher <- *mastership
			}
			e.mu.RUnlock()
		}
	}
}

func (e *atomixDeviceMastershipElection) getMastership() *Mastership {
	e.mu.RLock()
	defer e.mu.RUnlock()
	return e.mastership
}

func (e *atomixDeviceMastershipElection) watch(ch chan<- Mastership) error {
	e.mu.Lock()
	e.watchers = append(e.watchers, ch)
	e.mu.Unlock()
	return nil
}

func (e *atomixDeviceMastershipElection) Close() error {
	err := e.election.Close(context.Background())
	if err != nil {
		return errors.FromAtomix(err)
	}
	return nil
}

var _ deviceMastershipElection = &atomixDeviceMastershipElection{}
