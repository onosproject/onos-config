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

package gnmi

import (
	"context"
	"github.com/onosproject/onos-config/pkg/manager"
	"github.com/onosproject/onos-config/pkg/utils"
	"github.com/openconfig/gnmi/proto/gnmi"
	"time"
)

// Get implements gNMI Get
func (s *Server) Get(ctx context.Context, req *gnmi.GetRequest) (*gnmi.GetResponse, error) {

	updates := make([]*gnmi.Update,0)

	for _, path := range req.Path {
		target := path.Target
		// Special case - if target is "*" then ignore path and just return a list
		// of devices
		if target == "*" {
			deviceIds := make([]*gnmi.TypedValue, 0)

			for _, deviceID := range *manager.GetManager().GetAllDeviceIds() {
				typedVal := gnmi.TypedValue_StringVal{StringVal: deviceID}
				deviceIds = append(deviceIds, &gnmi.TypedValue{Value: &typedVal})
			}
			var allDevicesPathElem = make([]*gnmi.PathElem, 0)
			allDevicesPathElem = append(allDevicesPathElem, &gnmi.PathElem{Name: "all-devices"})
			allDevicesPath := gnmi.Path{Elem:allDevicesPathElem, Target: "*"}
			typedVal := gnmi.TypedValue_LeaflistVal{LeaflistVal: &gnmi.ScalarArray{Element: deviceIds}}

			update := &gnmi.Update{
				Path: &allDevicesPath,
				Val:  &gnmi.TypedValue{Value: &typedVal},
			}
			updates = append(updates, update)
			break
		}

		pathAsString := utils.StrPath(path)
		configValues, err := manager.GetManager().GetNetworkConfig(target, "",
			pathAsString, 0)
		if err != nil {
			return nil, err
		}

		var value *gnmi.TypedValue
		if len(configValues) == 0 {
			value = nil
		} else if len(configValues) == 1 {
			typedValue := &gnmi.TypedValue_AsciiVal{AsciiVal:configValues[0].Value}
			value = &gnmi.TypedValue{
				Value: typedValue,
			}
		} else {
			json, err := manager.BuildTree(configValues)
			if err != nil {

			}
			typedValue := &gnmi.TypedValue_JsonVal{
				JsonVal: json,
			}
			value = &gnmi.TypedValue{
				Value: typedValue,
			}
		}

		update := &gnmi.Update{
			Path: path,
			Val:  value,
		}
		updates = append(updates, update)
	}

	notification := &gnmi.Notification{
		Timestamp: time.Now().Unix(),
		Update: updates,

	}
	notifications := make([]*gnmi.Notification,0)
	notifications = append(notifications, notification)
	response := gnmi.GetResponse{
		Notification: notifications,
	}
	return &response, nil
}