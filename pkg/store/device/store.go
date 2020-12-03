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

package device

import (
	"context"
	"fmt"
	"io"
	"time"

	"github.com/onosproject/onos-api/go/onos/topo"
	"github.com/onosproject/onos-config/pkg/device"
	"github.com/onosproject/onos-lib-go/pkg/southbound"

	"google.golang.org/grpc"
)

// Store is a device store
type Store interface {
	// Get gets a device by ID
	Get(device.ID) (*device.Device, error)

	// Update updates a given device
	Update(*device.Device) (*device.Device, error)

	// List lists the devices in the store
	List(chan<- *device.Device) error

	// Watch watches the device store for changes
	Watch(chan<- *device.ListResponse) error
}

// NewTopoStore returns a new topo-based device store
func NewTopoStore(topoEndpoint string, opts ...grpc.DialOption) (Store, error) {
	if len(opts) == 0 {
		return nil, fmt.Errorf("no opts given when creating topo store")
	}
	opts = append(opts, grpc.WithStreamInterceptor(southbound.RetryingStreamClientInterceptor(100*time.Millisecond)))
	conn, err := getTopoConn(topoEndpoint, opts...)
	if err != nil {
		return nil, err
	}
	client := topo.NewTopoClient(conn)

	return &topoStore{
		client: client,
	}, nil
}

// NewStore returns a new device store for the given client
func NewStore(client topo.TopoClient) (Store, error) {
	return &topoStore{client: client}, nil
}

// A device Store that uses the topo service to propagate devices
type topoStore struct {
	client topo.TopoClient
}

func (s *topoStore) Get(id device.ID) (*device.Device, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	response, err := s.client.Get(ctx, &topo.GetRequest{
		ID: topo.ID(id),
	})
	if err != nil {
		return nil, err
	}
	return device.ToDevice(response.Object), nil
}

func (s *topoStore) Update(updatedDevice *device.Device) (*device.Device, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	updateReq := &topo.UpdateRequest{
		Object: device.ToObject(updatedDevice),
	}
	response, err := s.client.Update(ctx, updateReq)
	if err != nil {
		return nil, err
	}
	return device.ToDevice(response.Object), nil
}

func (s *topoStore) List(ch chan<- *device.Device) error {
	resp, err := s.client.List(context.Background(), &topo.ListRequest{})
	if err != nil {
		return err
	}

	go func() {
		for _, object := range resp.Objects {
			ch <- device.ToDevice(&object)
		}
	}()
	return nil
}

func (s *topoStore) Watch(ch chan<- *device.ListResponse) error {
	stream, err := s.client.Watch(context.Background(), &topo.WatchRequest{
		Noreplay: true,
	})
	if err != nil {
		return err
	}
	go func() {
		for {
			resp, err := stream.Recv()
			if err == io.EOF {
				break
			}
			if err != nil {
				break
			}
			if resp.Event.Object.Type == topo.Object_ENTITY {
				ch <- &device.ListResponse{
					Device: device.ToDevice(&resp.Event.Object),
					Type:   toEventType(resp.Event.Type),
				}
			}
		}
	}()
	return nil
}

func toEventType(et topo.EventType) device.ListResponseType {
	if et == topo.EventType_ADDED {
		return device.ListResponseADDED
	} else if et == topo.EventType_UPDATED {
		return device.ListResponseUPDATED
	} else if et == topo.EventType_REMOVED {
		return device.ListResponseREMOVED
	}
	return device.ListResponseNONE
}

// getTopoConn gets a gRPC connection to the topology service
func getTopoConn(topoEndpoint string, opts ...grpc.DialOption) (*grpc.ClientConn, error) {
	return grpc.Dial(topoEndpoint, opts...)
}
