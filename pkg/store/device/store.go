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
	topodevice "github.com/onosproject/onos-topo/api/topo"
	"google.golang.org/grpc"
)

// Store is a device store
type Store interface {
	// Get gets a device by ID
	Get(topodevice.ID) (*topodevice.Object, error)

	// Update updates a given device
	Update(*topodevice.Object) (*topodevice.Object, error)

	// List lists the devices in the store
	List(chan<- *topodevice.Object) error

	// Watch watches the device store for changes
	Watch(chan<- *topodevice.SubscribeResponse) error
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
	client := topodevice.NewTopoClient(conn)

	return &topoStore{
		client: client,
	}, nil
}

// NewStore returns a new device store for the given client
func NewStore(client topodevice.TopoClient) (Store, error) {
	return &topoStore{client: client}, nil
}

// A device Store that uses the topo service to propagate devices
type topoStore struct {
	client topodevice.TopoClient
}

func (s *topoStore) Get(id topodevice.ID) (*topodevice.Object, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	response, err := s.client.Get(ctx, &topodevice.GetRequest{
		ID: id,
	})
	if err != nil {
		return nil, err
	}
	return response.Object, nil
}

func (s *topoStore) Update(updatedDevice *topodevice.Object) (*topodevice.Object, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	updateReq := &topodevice.SetRequest{
		Objects: []*topodevice.Object{updatedDevice},
	}
	response, err := s.client.Set(ctx, updateReq)
	if response == nil || err != nil {
		return nil, err
	}
	return updatedDevice, nil
}

func (s *topoStore) List(ch chan<- *topodevice.Object) error {
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
			if isDevice(response.Object) {
				ch <- response.Object
			}
		}
	}()
	return nil
}

func (s *topoStore) Watch(ch chan<- *topodevice.SubscribeResponse) error {
	list, err := s.client.Subscribe(context.Background(), &topodevice.SubscribeRequest{})
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
			if response.Update.Object.Type != topodevice.Object_ENTITY {
				continue
			}
			if len(response.Update.Object.GetEntity().Protocols) == 0 {
				continue
			}
			if isDevice(response.Update.Object) {
				ch <- response
			}
		}
	}()
	return nil
}

// getTopoConn gets a gRPC connection to the topology service
func getTopoConn(topoEndpoint string, opts ...grpc.DialOption) (*grpc.ClientConn, error) {
	return grpc.Dial(topoEndpoint, opts...)
}

func isDevice(device *topodevice.Object) bool {
	return device.Type == topodevice.Object_ENTITY
}
