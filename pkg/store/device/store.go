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
	devicepb "github.com/onosproject/onos-topo/pkg/northbound/device"
	"google.golang.org/grpc"
	"io"
	log "k8s.io/klog"
	"time"
)

const topoAddress = "onos-topo:5150"

// Store is a device store
type Store interface {
	// Get gets a device by ID
	Get(devicepb.ID) (*devicepb.Device, error)

	// List lists the devices in the store
	List(chan<- *devicepb.Device) error

	// Watch watches the device store for changes
	Watch(chan<- *devicepb.Device) error
}

// NewTopoStore returns a new topo-based device store
func NewTopoStore(opts ...grpc.DialOption) (Store, error) {
	opts = append(opts, grpc.WithStreamInterceptor(util.RetryingStreamClientInterceptor(100*time.Millisecond)))
	conn, err := getTopoConn(opts...)
	if err != nil {
		return nil, err
	}

	requestConn, err := getTopoConn(opts...)
	if err != nil {
		return nil, err
	}

	client := devicepb.NewDeviceServiceClient(conn)

	requestClient := devicepb.NewDeviceServiceClient(requestConn)

	return &topoStore{
		client:        client,
		requestClient: requestClient,
	}, nil
}

// A device Store that uses the topo service to propagate devices
type topoStore struct {
	client        devicepb.DeviceServiceClient
	requestClient devicepb.DeviceServiceClient
}

func (s *topoStore) Get(id devicepb.ID) (*devicepb.Device, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	response, err := s.client.Get(ctx, &devicepb.GetRequest{
		ID: id,
	})
	if err != nil {
		return nil, err
	}
	return response.Device, nil
}

func (s *topoStore) List(ch chan<- *devicepb.Device) error {
	list, err := s.client.List(context.Background(), &devicepb.ListRequest{})
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

func (s *topoStore) Watch(ch chan<- *devicepb.Device) error {
	list, err := s.client.List(context.Background(), &devicepb.ListRequest{
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

// DeviceConnected signals the corresponding topology service that the device connected.
func (s *topoStore) DeviceConnected(id devicepb.ID) (*devicepb.Device, error) {
	log.Infof("Device %s connected", id)
	return s.updateDevice(id, devicepb.ConnectivityState_REACHABLE, devicepb.ChannelState_CONNECTED,
		devicepb.ServiceState_AVAILABLE)
}

// DeviceDisconnected signal the corresponding topology service that the device disconnected.
func (s *topoStore) DeviceDisconnected(id devicepb.ID, err error) (*devicepb.Device, error) {
	log.Infof("Device %s disconnected or had error in connection %s", id, err)
	//TODO check different possible availabilities based on error
	return s.updateDevice(id, devicepb.ConnectivityState_UNREACHABLE, devicepb.ChannelState_DISCONNECTED,
		devicepb.ServiceState_UNAVAILABLE)
}

func (s *topoStore) updateDevice(id devicepb.ID, connectivity devicepb.ConnectivityState, channel devicepb.ChannelState,
	service devicepb.ServiceState) (*devicepb.Device, error) {
	getResponse, err := s.requestClient.Get(context.Background(), &devicepb.GetRequest{
		ID: id,
	})
	if err != nil {
		return nil, err
	}
	topoDevice := getResponse.Device
	protocolState, index := containsGnmi(topoDevice.Protocols)
	if protocolState != nil {
		topoDevice.Protocols = remove(topoDevice.Protocols, index)
	} else {
		protocolState = new(devicepb.ProtocolState)
	}
	protocolState.Protocol = devicepb.Protocol_GNMI
	protocolState.ConnectivityState = connectivity
	protocolState.ChannelState = channel
	protocolState.ServiceState = service
	topoDevice.Protocols = append(topoDevice.Protocols, protocolState)
	updateReq := devicepb.UpdateRequest{
		Device: topoDevice,
	}
	updateResponse, err := s.requestClient.Update(context.Background(), &updateReq)
	if err != nil {
		log.Errorf("Device %s is not updated %s", id, err.Error())
		return nil, err
	}
	log.Infof("Device %s is updated with states %s, %s, %s", id, connectivity, channel, service)
	return updateResponse.Device, nil
}

func containsGnmi(protocols []*devicepb.ProtocolState) (*devicepb.ProtocolState, int) {
	for i, p := range protocols {
		if p.Protocol == devicepb.Protocol_GNMI {
			return p, i
		}
	}
	return nil, -1
}

func remove(s []*devicepb.ProtocolState, i int) []*devicepb.ProtocolState {
	s[i] = s[len(s)-1]
	return s[:len(s)-1]
}
