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

package admin

import (
	"context"
	"fmt"
	"github.com/onosproject/onos-config/pkg/manager"
	"github.com/onosproject/onos-config/pkg/northbound/proto"
	"github.com/onosproject/onos-config/pkg/southbound/topocache"
	"github.com/onosproject/onos-config/pkg/store"
	"github.com/onosproject/onos-config/pkg/store/change"
	"time"
)

// GetDeviceSummary returns the summary information about the device inventory.
func (s *Server) GetDeviceSummary(c context.Context, d *proto.DeviceSummaryRequest) (*proto.DeviceSummaryResponse, error) {
	return &proto.DeviceSummaryResponse{Count: int32(len(manager.GetManager().DeviceStore.Store))}, nil
}

// AddOrUpdateDevice adds the specified device to the device inventory.
func (s *Server) AddOrUpdateDevice(c context.Context, d *proto.DeviceInfo) (*proto.DeviceResponse, error) {
	err := manager.GetManager().DeviceStore.AddOrUpdateDevice(topocache.ID(d.Id), topocache.Device{
		ID:              topocache.ID(d.Id),
		Addr:            d.Address,
		Target:          d.Target,
		SoftwareVersion: d.Version,
		Usr:             d.User,
		Pwd:             d.Password,
		CaPath:          d.CaPath,
		CertPath:        d.CertPath,
		KeyPath:         d.KeyPath,
		Insecure:        d.Insecure,
		Plain:           d.Plain,
		Timeout:         d.Timeout,
	})
	if err != nil {
		return nil, err
	}

	configStore := manager.GetManager().ConfigStore
	name := store.ConfigName(fmt.Sprintf("%s-%s", d.Id, d.Version))
	if _, ok := configStore.Store[name]; !ok {
		if d.Devicetype == "" {
			return nil, fmt.Errorf("devicetype must be specified (creating "+
				"a new config as a side effect of creating the new device %s)", name)
		}
		configStore.Store[name] = store.Configuration{
			Name:    name,
			Device:  d.Id,
			Version: d.Version,
			Type:    d.Devicetype,
			Created: time.Now(),
			Updated: time.Now(),
			Changes: []change.ID{},
		}
	}
	return &proto.DeviceResponse{}, nil
}

// RemoveDevice removes the specified device from the inventory.
func (s *Server) RemoveDevice(c context.Context, d *proto.DeviceInfo) (*proto.DeviceResponse, error) {
	manager.GetManager().DeviceStore.RemoveDevice(topocache.ID(d.Id))
	return &proto.DeviceResponse{}, nil
}

// GetDevices provides a stream of devices in the inventory.
func (s *Server) GetDevices(r *proto.GetDevicesRequest, stream proto.DeviceInventoryService_GetDevicesServer) error {
	for id, dev := range manager.GetManager().DeviceStore.Store {

		// Build the device info message
		msg := &proto.DeviceInfo{
			Id: string(id), Address: dev.Addr, Target: dev.Target, Version: dev.SoftwareVersion,
			User: dev.Usr, Password: dev.Pwd,
			CaPath: dev.CaPath, CertPath: dev.CertPath, KeyPath: dev.KeyPath,
			Plain: dev.Plain, Insecure: dev.Insecure, Timeout: dev.Timeout,
		}

		err := stream.Send(msg)
		if err != nil {
			return err
		}
	}
	return nil
}
