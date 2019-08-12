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

/*
Package topocache is a mechanism for holding a cache of Devices.

When onos-topology is in place it will be the ultimate reference of device
availability and accessibility
Until then this simple cache will load a set of Device definitions from file
*/
package topocache

import (
	"context"
	"github.com/onosproject/onos-config/pkg/events"
	"github.com/onosproject/onos-topo/pkg/northbound/device"
	"github.com/prometheus/common/log"
	"google.golang.org/grpc"
	"io"
)

const (
	topoAddress = "onos-topo:5150"
)

// DeviceStore is the model of the Device store
type DeviceStore struct {
	client device.DeviceServiceClient
}

// LoadDeviceStore loads a device store
func LoadDeviceStore(topoChannel chan<- events.TopoEvent, opts ...grpc.DialOption) (*DeviceStore, error) {
	if len(opts) == 0 {
		return nil, nil
	}

	conn, err := getTopoConn(opts...)
	if err != nil {
		return nil, err
	}

	client := device.NewDeviceServiceClient(conn)
	deviceStore := &DeviceStore{
		client: client,
	}

	err = deviceStore.start(topoChannel)
	if err != nil {
		return nil, err
	}
	return deviceStore, nil
}

// start subscribes the device store to device events via the DeviceService
func (s *DeviceStore) start(ch chan<- events.TopoEvent) error {
	list, err := s.client.List(context.Background(), &device.ListRequest{
		Subscribe: true,
	})
	if err != nil {
		return err
	}

	go func() {
		defer close(ch)
		for {
			response, err := list.Recv()
			if err == io.EOF {
				break
			}
			if err != nil {
				log.Error(err)
				break
			}
			ch <- events.CreateTopoEvent(response.Device.ID, true, response.Device.Address, *response.Device)
		}
	}()
	return nil
}

// getTopoConn gets a gRPC connection to the topology service
func getTopoConn(opts ...grpc.DialOption) (*grpc.ClientConn, error) {
	return grpc.Dial(topoAddress, opts...)
}
