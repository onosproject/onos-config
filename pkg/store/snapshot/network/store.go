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

package network

import (
	"context"
	"fmt"
	"github.com/atomix/atomix-go-framework/pkg/atomix/meta"
	"github.com/onosproject/onos-lib-go/pkg/errors"
	"io"
	"os"
	"time"

	"github.com/atomix/atomix-go-client/pkg/atomix"
	"github.com/atomix/atomix-go-client/pkg/atomix/indexedmap"
	"github.com/gogo/protobuf/proto"
	"github.com/google/uuid"
	networksnapshot "github.com/onosproject/onos-api/go/onos/config/snapshot/network"
	"github.com/onosproject/onos-config/pkg/store/stream"
)

// NewAtomixStore returns a new persistent Store
func NewAtomixStore() (Store, error) {
	client := atomix.NewClient(atomix.WithClientID(os.Getenv("POD_NAME")))
	snapshots, err := client.GetIndexedMap(context.Background(), fmt.Sprintf("%s-network-snapshots", os.Getenv("SERVICE_NAME")))
	if err != nil {
		return nil, errors.FromAtomix(err)
	}
	return &atomixStore{
		snapshots: snapshots,
	}, nil
}

// Store stores NetworkSnapshots
type Store interface {
	io.Closer

	// Get gets a network snapshot
	Get(id networksnapshot.ID) (*networksnapshot.NetworkSnapshot, error)

	// GetByIndex gets a network snapshot by index
	GetByIndex(index networksnapshot.Index) (*networksnapshot.NetworkSnapshot, error)

	// Create creates a new network snapshot
	Create(snapshot *networksnapshot.NetworkSnapshot) error

	// Update updates an existing network snapshot
	Update(snapshot *networksnapshot.NetworkSnapshot) error

	// Delete deletes a network snapshot
	Delete(snapshot *networksnapshot.NetworkSnapshot) error

	// List lists network snapshots
	List(chan<- *networksnapshot.NetworkSnapshot) (stream.Context, error)

	// Watch watches the network snapshot store for changes
	Watch(chan<- stream.Event) (stream.Context, error)
}

// newSnapshotID creates a new network snapshot ID
func newSnapshotID() networksnapshot.ID {
	return networksnapshot.ID(uuid.New().String())
}

// atomixStore is the default implementation of the NetworkSnapshot store
type atomixStore struct {
	snapshots indexedmap.IndexedMap
}

func (s *atomixStore) Get(id networksnapshot.ID) (*networksnapshot.NetworkSnapshot, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	entry, err := s.snapshots.Get(ctx, string(id))
	if err != nil {
		return nil, errors.FromAtomix(err)
	} else if entry == nil {
		return nil, nil
	}
	return decodeSnapshot(entry)
}

func (s *atomixStore) GetByIndex(index networksnapshot.Index) (*networksnapshot.NetworkSnapshot, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	entry, err := s.snapshots.GetIndex(ctx, indexedmap.Index(index))
	if err != nil {
		return nil, errors.FromAtomix(err)
	} else if entry == nil {
		return nil, nil
	}
	return decodeSnapshot(entry)
}

func (s *atomixStore) Create(snapshot *networksnapshot.NetworkSnapshot) error {
	if snapshot.ID == "" {
		snapshot.ID = newSnapshotID()
	}
	if snapshot.Revision != 0 {
		return errors.NewInvalid("not a new object")
	}

	bytes, err := proto.Marshal(snapshot)
	if err != nil {
		return errors.NewInvalid("snapshot encoding failed: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	entry, err := s.snapshots.Append(ctx, string(snapshot.ID), bytes)
	if err != nil {
		return errors.FromAtomix(err)
	}

	snapshot.Index = networksnapshot.Index(entry.Index)
	snapshot.Revision = networksnapshot.Revision(entry.Version)
	return nil
}

func (s *atomixStore) Update(snapshot *networksnapshot.NetworkSnapshot) error {
	if snapshot.Revision == 0 {
		return errors.NewInvalid("not a stored object")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	bytes, err := proto.Marshal(snapshot)
	if err != nil {
		return errors.NewInvalid("snapshot encoding failed: %v", err)
	}

	entry, err := s.snapshots.Set(ctx, indexedmap.Index(snapshot.Index), string(snapshot.ID), bytes, indexedmap.IfMatch(meta.NewRevision(meta.Revision(snapshot.Revision))))
	if err != nil {
		return errors.FromAtomix(err)
	}

	snapshot.Revision = networksnapshot.Revision(entry.Version)
	return nil
}

func (s *atomixStore) Delete(snapshot *networksnapshot.NetworkSnapshot) error {
	if snapshot.Revision == 0 {
		return errors.NewInvalid("not a stored object")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	_, err := s.snapshots.RemoveIndex(ctx, indexedmap.Index(snapshot.Index), indexedmap.IfMatch(meta.NewRevision(meta.Revision(snapshot.Revision))))
	if err != nil {
		return errors.FromAtomix(err)
	}

	snapshot.Revision = 0
	return nil
}

func (s *atomixStore) List(ch chan<- *networksnapshot.NetworkSnapshot) (stream.Context, error) {
	ctx, cancel := context.WithCancel(context.Background())

	mapCh := make(chan indexedmap.Entry)
	if err := s.snapshots.Entries(ctx, mapCh); err != nil {
		cancel()
		return nil, errors.FromAtomix(err)
	}

	go func() {
		defer close(ch)
		for entry := range mapCh {
			if snapshot, err := decodeSnapshot(&entry); err == nil {
				ch <- snapshot
			}
		}
	}()
	return stream.NewCancelContext(cancel), nil
}

func (s *atomixStore) Watch(ch chan<- stream.Event) (stream.Context, error) {
	ctx, cancel := context.WithCancel(context.Background())

	mapCh := make(chan indexedmap.Event)
	if err := s.snapshots.Watch(ctx, mapCh, indexedmap.WithReplay()); err != nil {
		cancel()
		return nil, errors.FromAtomix(err)
	}

	go func() {
		defer close(ch)
		for event := range mapCh {
			if snapshot, err := decodeSnapshot(&event.Entry); err == nil {
				switch event.Type {
				case indexedmap.EventReplay:
					ch <- stream.Event{
						Type:   stream.None,
						Object: snapshot,
					}
				case indexedmap.EventInsert:
					ch <- stream.Event{
						Type:   stream.Created,
						Object: snapshot,
					}
				case indexedmap.EventUpdate:
					ch <- stream.Event{
						Type:   stream.Updated,
						Object: snapshot,
					}
				case indexedmap.EventRemove:
					ch <- stream.Event{
						Type:   stream.Deleted,
						Object: snapshot,
					}
				}
			}
		}
	}()
	return stream.NewCancelContext(cancel), nil
}

func (s *atomixStore) Close() error {
	err := s.snapshots.Close(context.Background())
	if err != nil {
		return errors.FromAtomix(err)
	}
	return nil
}

func decodeSnapshot(entry *indexedmap.Entry) (*networksnapshot.NetworkSnapshot, error) {
	snapshot := &networksnapshot.NetworkSnapshot{}
	if err := proto.Unmarshal(entry.Value, snapshot); err != nil {
		return nil, errors.NewInvalid("snapshot decoding failed: %v", err)
	}
	snapshot.ID = networksnapshot.ID(entry.Key)
	snapshot.Index = networksnapshot.Index(entry.Index)
	snapshot.Revision = networksnapshot.Revision(entry.Version)
	return snapshot, nil
}
