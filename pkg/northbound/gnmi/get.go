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
	"log"
	"time"

	"github.com/onosproject/onos-config/pkg/manager"
	"github.com/onosproject/onos-config/pkg/store/change"
	"github.com/onosproject/onos-config/pkg/utils"
	"github.com/openconfig/gnmi/proto/gnmi"
)

// Get implements gNMI Get
func (s *Server) Get(ctx context.Context, req *gnmi.GetRequest) (*gnmi.GetResponse, error) {

	updates := make([]*gnmi.Update, 0)

	for _, path := range req.GetPath() {
		update, err := GetUpdate(path)
		if err != nil {
			return nil, err
		}
		updates = append(updates, update)
	}

	notification := &gnmi.Notification{
		Timestamp: time.Now().Unix(),
		Update:    updates,
	}
	notifications := make([]*gnmi.Notification, 0)
	notifications = append(notifications, notification)
	response := gnmi.GetResponse{
		Notification: notifications,
	}
	return &response, nil
}

// GetUpdate utility method for getting an Update for a given path
func GetUpdate(path *gnmi.Path) (*gnmi.Update, error) {
	target := path.GetTarget()
	// Special case - if target is "*" then ignore path and just return a list
	// of devices
	if target == "*" {
		log.Println("Testing subscription")
		deviceIds := make([]*gnmi.TypedValue, 0)

		for _, deviceID := range *manager.GetManager().GetAllDeviceIds() {
			typedVal := gnmi.TypedValue_StringVal{StringVal: deviceID}
			deviceIds = append(deviceIds, &gnmi.TypedValue{Value: &typedVal})
		}
		var allDevicesPathElem = make([]*gnmi.PathElem, 0)
		allDevicesPathElem = append(allDevicesPathElem, &gnmi.PathElem{Name: "all-devices"})
		allDevicesPath := gnmi.Path{Elem: allDevicesPathElem, Target: "*"}
		typedVal := gnmi.TypedValue_LeaflistVal{LeaflistVal: &gnmi.ScalarArray{Element: deviceIds}}

		update := &gnmi.Update{
			Path: &allDevicesPath,
			Val:  &gnmi.TypedValue{Value: &typedVal},
		}
		return update, nil
	}

	pathAsString := utils.StrPath(path)
	configValues, err := manager.GetManager().GetNetworkConfig(target, "",
		pathAsString, 0)
	if err != nil {
		return nil, err
	}

	return buildUpdate(path, configValues), nil
}

func buildUpdate(path *gnmi.Path, configValues []change.ConfigValue) *gnmi.Update {
	var value *gnmi.TypedValue
	if len(configValues) == 0 {
		value = nil
	} else if len(configValues) == 1 {
		typedValue := &gnmi.TypedValue_StringVal{StringVal: configValues[0].Value}
		value = &gnmi.TypedValue{
			Value: typedValue,
		}
	} else {
		json, err := manager.BuildTree(configValues, false)
		if err != nil {

		}
		typedValue := &gnmi.TypedValue_JsonVal{
			JsonVal: json,
		}
		value = &gnmi.TypedValue{
			Value: typedValue,
		}
	}
	return &gnmi.Update{
		Path: path,
		Val:  value,
	}
}
