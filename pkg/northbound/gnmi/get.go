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
	"fmt"
	"github.com/onosproject/onos-config/pkg/store"
	"github.com/onosproject/onos-config/pkg/utils/values"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	log "k8s.io/klog"
	"strings"
	"time"

	"github.com/onosproject/onos-config/pkg/manager"
	"github.com/onosproject/onos-config/pkg/store/change"
	"github.com/onosproject/onos-config/pkg/utils"
	"github.com/onosproject/onos-topo/pkg/northbound/device"
	"github.com/openconfig/gnmi/proto/gnmi"
	"github.com/openconfig/gnmi/proto/gnmi_ext"
)

// Get implements gNMI Get
func (s *Server) Get(ctx context.Context, req *gnmi.GetRequest) (*gnmi.GetResponse, error) {
	notifications := make([]*gnmi.Notification, 0)

	prefix := req.GetPrefix()

	disconnectedDevicesMap := make(map[device.ID]bool)

	s.mu.RLock()
	defer s.mu.RUnlock()
	for _, path := range req.GetPath() {
		update, err := getUpdate(prefix, path)
		if err != nil {
			return nil, status.Error(codes.Internal, err.Error())
		}
		//if target is already disconnected we don't do a get again.
		_, ok := disconnectedDevicesMap[device.ID(path.GetTarget())]
		if !ok {
			deviceGet, errGet := manager.GetManager().DeviceStore.Get(device.ID(prefix.GetTarget()))

			if errGet != nil && status.Convert(errGet).Code() == codes.NotFound {
				log.Infof("Device is not connected %s, %s, %s", path.GetTarget(), ok, status.Convert(errGet).Code(), errGet, deviceGet)
				disconnectedDevicesMap[device.ID(path.GetTarget())] = true
			}
		}
		notification := &gnmi.Notification{
			Timestamp: time.Now().Unix(),
			Update:    []*gnmi.Update{update},
			Prefix:    prefix,
		}

		notifications = append(notifications, notification)
	}
	// Alternatively - if there's only the prefix
	if len(req.GetPath()) == 0 {
		update, err := getUpdate(prefix, nil)
		if err != nil {
			return nil, status.Error(codes.Internal, err.Error())
		}
		//if target is already disconnected we don't do a get again.
		_, ok := disconnectedDevicesMap[device.ID(prefix.GetTarget())]
		if !ok {
			deviceGet, errGet := manager.GetManager().DeviceStore.Get(device.ID(prefix.GetTarget()))
			if errGet != nil && status.Convert(errGet).Code() == codes.NotFound {
				log.Infof("Device is not connected %s, %s, %s", prefix.GetTarget(), ok, status.Convert(errGet).Code(), errGet, deviceGet)
				disconnectedDevicesMap[device.ID(prefix.GetTarget())] = true
			}
		}
		notification := &gnmi.Notification{
			Timestamp: time.Now().Unix(),
			Update:    []*gnmi.Update{update},
			Prefix:    prefix,
		}

		notifications = append(notifications, notification)
	}

	response := gnmi.GetResponse{
		Notification: notifications,
	}
	if len(disconnectedDevicesMap) != 0 {
		disconnectedDevices := make([]string, 0)
		log.Info("Device Map {}", disconnectedDevicesMap)
		for k := range disconnectedDevicesMap {
			disconnectedDevices = append(disconnectedDevices, string(k))
		}
		disconnectedDeviceString := strings.Join(disconnectedDevices, ",")
		response.Extension = []*gnmi_ext.Extension{
			{
				Ext: &gnmi_ext.Extension_RegisteredExt{
					RegisteredExt: &gnmi_ext.RegisteredExtension{
						Id:  GnmiExtensionDevicesNotConnected,
						Msg: []byte(disconnectedDeviceString),
					},
				},
			},
		}
	}
	return &response, nil
}

// getUpdate utility method for getting an Update for a given path
func getUpdate(prefix *gnmi.Path, path *gnmi.Path) (*gnmi.Update, error) {
	if (path == nil || path.Target == "") && (prefix == nil || prefix.Target == "") {
		return nil, fmt.Errorf("Invalid request - Path %s has no target", utils.StrPath(path))
	}

	// If a target exists on the path, use it. If not use target of Prefix
	target := path.GetTarget()
	if target == "" {
		target = prefix.Target
	}
	// Special case - if target is "*" then ignore path and just return a list
	// of devices - note, this can be done in addition to other Paths in the same Get
	if target == "*" {
		log.Info("Testing subscription")
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
	if prefix != nil && prefix.Elem != nil {
		pathAsString = utils.StrPath(prefix) + pathAsString
	}
	//TODO the following can be optimized by looking if the path is in the read only
	configValues, err := manager.GetManager().GetTargetConfig(target, "",
		pathAsString, 0)
	if err != nil {
		return nil, err
	}
	stateValues := manager.GetManager().GetTargetState(target, pathAsString)
	//Merging the two results
	configValues = append(configValues, stateValues...)

	return buildUpdate(prefix, path, configValues)
}

func buildUpdate(prefix *gnmi.Path, path *gnmi.Path, configValues []*change.ConfigValue) (*gnmi.Update, error) {
	var value *gnmi.TypedValue
	var err error
	if len(configValues) == 0 {
		value = nil
	} else if len(configValues) == 1 {
		value, err = values.NativeTypeToGnmiTypedValue(&configValues[0].TypedValue)
		if err != nil {
			log.Warning("Unable to convert native value to gnmi", err)
			return nil, err
		}
		// These should match the assignments made in changevalue.go
	} else {
		json, err := store.BuildTree(configValues, false)
		if err != nil {
			return nil, err
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
	}, nil
}
