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
	"errors"
	"github.com/cenkalti/backoff"
	"github.com/onosproject/onos-config/pkg/events"
	"github.com/onosproject/onos-topo/pkg/northbound/device"
	"google.golang.org/grpc"
	log "k8s.io/klog"
	"time"
)

const (
	topoAddress = "onos-topo:5150"
)

// DeviceStore is the model of the Device store
type DeviceStore struct {
	client device.DeviceServiceClient
	cache  map[device.ID]*device.Device
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
		cache:  make(map[device.ID]*device.Device),
	}
	go deviceStore.start(topoChannel)
	return deviceStore, nil
}

// start starts listening for events from the DeviceService
func (s *DeviceStore) start(ch chan<- events.TopoEvent) {
	// Retry continuously to listen for devices from the device service. The root retry loop is constant, so
	// when the device listener disconnects, a new connection will be attempted a second later. Each connection
	// iteration is performed using an exponential backoff algorithm, ensuring the client doesn't attempt to connect
	// to a missing service constantly.
	_ = backoff.Retry(func() error {
		operation := func() error {
			return s.watchEvents(ch)
		}

		// Use exponential backoff until the client is able to list devices. This operation should never return
		// an error since we don't use the error type required to fail the exponential backoff operation.
		_ = backoff.Retry(operation, backoff.NewExponentialBackOff())

		// Return a placeholder error to ensure the connection is retried.
		return errors.New("retry")
	}, backoff.NewConstantBackOff(1*time.Second))
}

// watchEvents listens for events from the DeviceService
func (s *DeviceStore) watchEvents(ch chan<- events.TopoEvent) error {
	list, err := s.client.List(context.Background(), &device.ListRequest{
		Subscribe: true,
	})

	// Return an error if the client was unable to connect to the service.
	if err != nil {
		log.Error(err)
		return err
	}

	for {
		response, err := list.Recv()

		// When an error occurs, log the error and return nil to reset the exponential backoff algorithm.
		if err != nil {
			log.Error(err)
			return nil
		}

		switch response.Type {
		case device.ListResponse_NONE:
			if _, ok := s.cache[response.Device.ID]; !ok {
				s.cache[response.Device.ID] = response.Device
				ch <- events.CreateTopoEvent(response.Device.ID, true, response.Device.Address, *response.Device)
			}
		case device.ListResponse_ADDED:
			s.cache[response.Device.ID] = response.Device
			ch <- events.CreateTopoEvent(response.Device.ID, true, response.Device.Address, *response.Device)
		case device.ListResponse_UPDATED:
			s.cache[response.Device.ID] = response.Device
			ch <- events.CreateTopoEvent(response.Device.ID, true, response.Device.Address, *response.Device)
		case device.ListResponse_REMOVED:
			ch <- events.CreateTopoEvent(response.Device.ID, false, response.Device.Address, *response.Device)
		}
	}
}

// getTopoConn gets a gRPC connection to the topology service
func getTopoConn(opts ...grpc.DialOption) (*grpc.ClientConn, error) {
	return grpc.Dial(topoAddress, opts...)
}
