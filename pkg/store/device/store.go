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

	"github.com/onosproject/onos-lib-go/pkg/southbound"
	topodevice "github.com/onosproject/onos-topo/api/device"
	"google.golang.org/grpc"
)

// Store is a device store
type Store interface {
	// Get gets a device by ID
	Get(topodevice.ID) (*topodevice.Device, error)

	// Update updates a given device
	Update(*topodevice.Device) (*topodevice.Device, error)

	// List lists the devices in the store
	List(chan<- *topodevice.Device) error

	// Watch watches the device store for changes
	Watch(chan<- *topodevice.ListResponse) error
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
	client := topodevice.NewDeviceServiceClient(conn)

	return &topoStore{
		client: client,
	}, nil
}

// NewStore returns a new device store for the given client
func NewStore(client topodevice.DeviceServiceClient) (Store, error) {
	return &topoStore{client: client}, nil
}

// A device Store that uses the topo service to propagate devices
type topoStore struct {
	client topodevice.DeviceServiceClient
}

func (s *topoStore) Get(id topodevice.ID) (*topodevice.Device, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	response, err := s.client.Get(ctx, &topodevice.GetRequest{
		ID: id,
	})
	if err != nil {
		return nil, err
	}
	return response.Device, nil
}

func (s *topoStore) Update(updatedDevice *topodevice.Device) (*topodevice.Device, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	updateReq := &topodevice.UpdateRequest{
		Device: updatedDevice,
	}
	response, err := s.client.Update(ctx, updateReq)
	if err != nil {
		return nil, err
	}
	return response.Device, nil
}

func (s *topoStore) List(ch chan<- *topodevice.Device) error {
	list, err := s.client.List(context.Background(), &topodevice.ListRequest{})
	if err != nil {
		return err
	}

	go func() {
		for {
			response, err := list.Recv()
			if err == io.EOF {
				break
			}
			if err != nil {
				break
			}
			ch <- response.Device
		}
	}()
	return nil
}

func (s *topoStore) Watch(ch chan<- *topodevice.ListResponse) error {
	list, err := s.client.List(context.Background(), &topodevice.ListRequest{
		Subscribe: true,
	})
	if err != nil {
		return err
	}
	go func() {
		for {
			response, err := list.Recv()
			if err == io.EOF {
				break
			}
			if err != nil {
				break
			}
			ch <- response
		}
	}()
	return nil
}

// getTopoConn gets a gRPC connection to the topology service
func getTopoConn(topoEndpoint string, opts ...grpc.DialOption) (*grpc.ClientConn, error) {
	return grpc.Dial(topoEndpoint, opts...)
}
