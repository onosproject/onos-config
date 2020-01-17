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
	devicechange "github.com/onosproject/onos-config/api/types/change/device"
	devicetype "github.com/onosproject/onos-config/api/types/device"
	"github.com/onosproject/onos-config/pkg/manager"
	"github.com/onosproject/onos-config/pkg/store"
	"github.com/onosproject/onos-config/pkg/utils"
	"github.com/onosproject/onos-config/pkg/utils/values"
	"github.com/openconfig/gnmi/proto/gnmi"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"time"
)

// Get implements gNMI Get
func (s *Server) Get(ctx context.Context, req *gnmi.GetRequest) (*gnmi.GetResponse, error) {
	notifications := make([]*gnmi.Notification, 0)

	prefix := req.GetPrefix()

	version, err := extractGetVersion(req)
	if err != nil {
		return nil, err
	}

	for _, path := range req.GetPath() {
		update, err := s.getUpdate(version, prefix, path)
		if err != nil {
			return nil, status.Error(codes.Internal, err.Error())
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
		update, err := s.getUpdate(version, prefix, nil)
		if err != nil {
			return nil, status.Error(codes.Internal, err.Error())
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
	return &response, nil
}

// getUpdate utility method for getting an Update for a given path
func (s *Server) getUpdate(version devicetype.Version, prefix *gnmi.Path, path *gnmi.Path) (*gnmi.Update, error) {
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

	_, version, errTypeVersion := manager.GetManager().CheckCacheForDevice(devicetype.ID(target), "", version)
	if errTypeVersion != nil {
		log.Errorf("Error while extracting type and version for target %s with err %v", target, errTypeVersion)
		return nil, status.Error(codes.InvalidArgument, errTypeVersion.Error())
	}

	pathAsString := utils.StrPath(path)
	if prefix != nil && prefix.Elem != nil {
		pathAsString = utils.StrPath(prefix) + pathAsString
	}

	s.mu.RLock()
	revision := s.lastWrite
	s.mu.RUnlock()

	configValues, errGetTargetCfg := manager.GetManager().GetTargetConfig(
		devicetype.ID(target), version, pathAsString, revision)
	if errGetTargetCfg != nil {
		log.Error("Error while extracting config", errGetTargetCfg)
		return nil, errGetTargetCfg
	}

	stateValues := manager.GetManager().GetTargetState(target, pathAsString)
	//Merging the two results
	configValues = append(configValues, stateValues...)

	return buildUpdate(prefix, path, configValues)
}

func buildUpdate(prefix *gnmi.Path, path *gnmi.Path, configValues []*devicechange.PathValue) (*gnmi.Update, error) {
	var value *gnmi.TypedValue
	var err error
	if len(configValues) == 0 {
		value = nil
	} else if len(configValues) == 1 {
		value, err = values.NativeTypeToGnmiTypedValue(configValues[0].GetValue())
		if err != nil {
			log.Warn("Unable to convert native value to gnmi", err)
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

func extractGetVersion(req *gnmi.GetRequest) (devicetype.Version, error) {
	var version devicetype.Version
	for _, ext := range req.GetExtension() {
		if ext.GetRegisteredExt().GetId() == GnmiExtensionVersion {
			version = devicetype.Version(ext.GetRegisteredExt().GetMsg())
		} else {
			return "", status.Error(codes.InvalidArgument, fmt.Errorf("unexpected extension %d = '%s' in Set()",
				ext.GetRegisteredExt().GetId(), ext.GetRegisteredExt().GetMsg()).Error())
		}
	}
	return version, nil
}
