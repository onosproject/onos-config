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
	deviceproto "github.com/onosproject/onos-topo/pkg/northbound/proto"
	"google.golang.org/grpc"
	"io"
	log "k8s.io/klog"
	"time"
)

const (
	topoAddress = "onos-topo:5150"
)

// DeviceStore is the model of the Device store
type DeviceStore struct {
	client deviceproto.DeviceServiceClient
}

// LoadDeviceStore loads a device store
func LoadDeviceStore(topoChannel chan<- events.TopoEvent, opts ...grpc.DialOption) (*DeviceStore, error) {
	conn, err := getTopoConn(opts...)
	if err != nil {
		return nil, err
	}

	client := deviceproto.NewDeviceServiceClient(conn)
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
	list, err := s.client.List(context.Background(), &deviceproto.ListRequest{
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
			ch <- events.CreateTopoEvent(response.Device.Id, true, response.Device.Address, response.Device)
		}
	}()
	return nil
}

// AddOrUpdateDevice adds or updates the specified device in the device inventory.
func (s *DeviceStore) AddOrUpdateDevice(device *deviceproto.Device) error {
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	if device.Metadata == nil {
		response, err := s.client.Add(ctx, &deviceproto.AddDeviceRequest{
			Device: device,
		})
		if err == nil {
			device.Metadata = response.Metadata
		}
		return err
	} else {
		_, err := s.client.Update(ctx, &deviceproto.UpdateDeviceRequest{
			Device: device,
		})
		return err
	}
}

// RemoveDevice removes the device with the specified address from the device inventory.
func (s *DeviceStore) RemoveDevice(device *deviceproto.Device) error {
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	_, err := s.client.Remove(ctx, &deviceproto.RemoveDeviceRequest{
		Device: device,
	})
	return err
}

// getTopoConn gets a gRPC connection to the topology service
func getTopoConn(opts ...grpc.DialOption) (*grpc.ClientConn, error) {
	return grpc.Dial(topoAddress, opts...)
}
