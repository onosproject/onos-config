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

// Package utils holds utilities for the northbound servers
package utils

import (
	"fmt"
	devicetype "github.com/onosproject/onos-api/go/onos/config/device"
	topodevice "github.com/onosproject/onos-config/pkg/device"
	devicestore "github.com/onosproject/onos-config/pkg/store/device"
	"github.com/onosproject/onos-config/pkg/store/device/cache"
	"github.com/onosproject/onos-lib-go/pkg/logging"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

var log = logging.GetLogger("utils")

// CheckCacheForDevice checks against the device cache (of the device change store
// to see if a device of that name is already present)
func CheckCacheForDevice(target devicetype.ID, deviceType devicetype.Type,
	version devicetype.Version, deviceCache cache.Cache, deviceStore devicestore.Store) (devicetype.Type, devicetype.Version, error) {

	deviceInfos := deviceCache.GetDevicesByID(target)
	topoDevice, errTopoDevice := deviceStore.Get(topodevice.ID(target))
	if errTopoDevice != nil {
		log.Infof("Device %s not found in topo store", target)
	}

	if len(deviceInfos) == 0 {
		// New device - need type and version
		if deviceType == "" || version == "" {
			if errTopoDevice == nil && topoDevice != nil {
				return devicetype.Type(topoDevice.Type), devicetype.Version(topoDevice.Version), nil
			}
			return "", "", status.Error(codes.Internal,
				fmt.Sprintf("target %s is not known. Need to supply a type and version through Extensions 101 and 102", target))
		}
		return deviceType, version, nil
	} else if len(deviceInfos) == 1 {
		log.Infof("Handling target %s as %s:%s", target, deviceType, version)
		if deviceInfos[0].Version != version {
			log.Infof("Ignoring device type %s and version %s from extension for %s. Using %s and %s",
				deviceType, version, target, deviceInfos[0].Type, deviceInfos[0].Version)
		}
		return deviceInfos[0].Type, deviceInfos[0].Version, nil
	} else {
		// n devices of that name already exist - have to choose 1 or exit
		for _, di := range deviceInfos {
			if di.Version == version {
				log.Infof("Handling target %s as %s:%s", target, deviceType, version)
				return di.Type, di.Version, nil
			}
		}
		// Else allow it as a new version
		if deviceType == deviceInfos[0].Type && version != "" {
			log.Infof("Handling target %s as %s:%s", target, deviceType, version)
			return deviceType, version, nil
		} else if deviceType != "" && deviceType != deviceInfos[0].Type {
			return "", "", status.Error(codes.Internal,
				fmt.Sprintf("target %s type given %s does not match expected %s",
					target, deviceType, deviceInfos[0].Type))
		}

		return "", "", status.Error(codes.Internal,
			fmt.Sprintf("target %s has %d versions. Specify 1 version with extension 102",
				target, len(deviceInfos)))
	}
}
