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
	"github.com/atomix/atomix-go-client/pkg/client/util"
	"github.com/onosproject/onos-topo/pkg/service/device"
	deviceservice "github.com/onosproject/onos-topo/pkg/service/device"
	devicetype "github.com/onosproject/onos-topo/pkg/types/device"
	"google.golang.org/grpc"
	"io"
	"time"
)

const topoAddress = "onos-topo:5150"

// Store is a device store
type Store interface {
	// Get gets a device by ID
	Get(devicetype.ID) (*devicetype.Device, error)

	// Watch watches the device store for changes
	Watch(chan<- *devicetype.Device) error
}

// NewTopoStore returns a new topo-based device store
func NewTopoStore(opts ...grpc.DialOption) (Store, error) {
	opts = append(opts,
		grpc.WithUnaryInterceptor(util.RetryingUnaryClientInterceptor()),
		grpc.WithStreamInterceptor(util.RetryingStreamClientInterceptor(100*time.Millisecond)))
	conn, err := getTopoConn(opts...)
	if err != nil {
		return nil, err
	}

	client := deviceservice.NewDeviceServiceClient(conn)
	return &topoStore{
		client: client,
	}, nil
}

// A device Store that uses the topo service to propagate devices
type topoStore struct {
	client deviceservice.DeviceServiceClient
}

func (s *topoStore) Get(id devicetype.ID) (*devicetype.Device, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	response, err := s.client.Get(ctx, &deviceservice.GetRequest{
		ID: id,
	})
	if err != nil {
		return nil, err
	}
	return response.Device, nil
}

func (s *topoStore) Watch(ch chan<- *devicetype.Device) error {
	list, err := s.client.List(context.Background(), &device.ListRequest{
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
			ch <- response.Device
		}
	}()
	return nil
}

// getTopoConn gets a gRPC connection to the topology service
func getTopoConn(opts ...grpc.DialOption) (*grpc.ClientConn, error) {
	return grpc.Dial(topoAddress, opts...)
}
