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
	"github.com/grpc-ecosystem/go-grpc-middleware/util/metautils"
	devicechange "github.com/onosproject/onos-api/go/onos/config/change/device"
	devicetype "github.com/onosproject/onos-api/go/onos/config/device"
	"github.com/onosproject/onos-config/pkg/manager"
	"github.com/onosproject/onos-config/pkg/store"
	"github.com/onosproject/onos-config/pkg/utils"
	"github.com/onosproject/onos-config/pkg/utils/values"
	"github.com/openconfig/gnmi/proto/gnmi"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"strings"
	"time"
)

// Get implements gNMI Get
func (s *Server) Get(ctx context.Context, req *gnmi.GetRequest) (*gnmi.GetResponse, error) {
	notifications := make([]*gnmi.Notification, 0)
	groups := make([]string, 0)
	if md := metautils.ExtractIncoming(ctx); md != nil && md.Get("name") != "" {
		groups = append(groups, strings.Split(md.Get("groups"), ";")...)
		log.Infof("gNMI Get() called by '%s (%s)'. Groups %v. Token %s",
			md.Get("name"), md.Get("email"), groups, md.Get("at_hash"))
	}
	if req == nil || (req.GetEncoding() != gnmi.Encoding_PROTO && req.GetEncoding() != gnmi.Encoding_JSON_IETF && req.GetEncoding() != gnmi.Encoding_JSON) {
		return nil, fmt.Errorf("invalid encoding format in Get request. Only JSON_IETF and PROTO accepted. %v", req.Encoding)
	}
	prefix := req.GetPrefix()

	version, err := extractGetVersion(req)
	if err != nil {
		return nil, err
	}

	for _, path := range req.GetPath() {
		updates, err := s.getUpdate(version, prefix, path, req.GetEncoding(), groups)
		if err != nil {
			return nil, status.Error(codes.Internal, err.Error())
		}
		notification := &gnmi.Notification{
			Timestamp: time.Now().Unix(),
			Update:    updates,
			Prefix:    prefix,
		}

		notifications = append(notifications, notification)
	}
	// Alternatively - if there's only the prefix
	if len(req.GetPath()) == 0 {
		updates, err := s.getUpdate(version, prefix, nil, req.GetEncoding(), groups)
		if err != nil {
			return nil, status.Error(codes.Internal, err.Error())
		}
		notification := &gnmi.Notification{
			Timestamp: time.Now().Unix(),
			Update:    updates,
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
func (s *Server) getUpdate(version devicetype.Version, prefix *gnmi.Path, path *gnmi.Path,
	encoding gnmi.Encoding, userGroups []string) ([]*gnmi.Update, error) {
	if (path == nil || path.Target == "") && (prefix == nil || prefix.Target == "") {
		return nil, fmt.Errorf("invalid request - Path %s has no target", utils.StrPath(path))
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
		deviceIDs := *manager.GetManager().GetAllDeviceIds()

		var allDevicesPathElem = make([]*gnmi.PathElem, 0)
		allDevicesPathElem = append(allDevicesPathElem, &gnmi.PathElem{Name: "all-devices"})
		allDevicesPath := gnmi.Path{Elem: allDevicesPathElem, Target: "*"}
		var typedVal gnmi.TypedValue
		switch encoding {
		case gnmi.Encoding_JSON, gnmi.Encoding_JSON_IETF:
			typedVal = gnmi.TypedValue{
				Value: &gnmi.TypedValue_JsonVal{
					JsonVal: []byte(fmt.Sprintf("{\"targets\": [\"%s\"]}", strings.Join(deviceIDs, "\",\""))),
				},
			}
		case gnmi.Encoding_PROTO:
			deviceIDStrs := make([]*gnmi.TypedValue, 0)
			for _, deviceID := range deviceIDs {
				typedVal := gnmi.TypedValue_StringVal{StringVal: deviceID}
				deviceIDStrs = append(deviceIDStrs, &gnmi.TypedValue{Value: &typedVal})
			}
			typedVal = gnmi.TypedValue{
				Value: &gnmi.TypedValue_LeaflistVal{LeaflistVal: &gnmi.ScalarArray{Element: deviceIDStrs}}}
		default:
			return nil, fmt.Errorf("get targets - unhandled encoding format %v", encoding)
		}
		update := gnmi.Update{
			Path: &allDevicesPath,
			Val:  &typedVal,
		}
		updates := []*gnmi.Update{
			&update,
		}
		return updates, nil
	}

	deviceType, version, errTypeVersion := manager.GetManager().CheckCacheForDevice(devicetype.ID(target), "", version)
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
		devicetype.ID(target), version, deviceType, pathAsString, revision, userGroups)
	if errGetTargetCfg != nil {
		log.Error("Error while extracting config", errGetTargetCfg)
		return nil, errGetTargetCfg
	}

	stateValues := manager.GetManager().GetTargetState(target, pathAsString)
	//Merging the two results
	configValues = append(configValues, stateValues...)

	return buildUpdate(prefix, path, configValues, encoding)
}

func buildUpdate(prefix *gnmi.Path, path *gnmi.Path, configValues []*devicechange.PathValue, encoding gnmi.Encoding) ([]*gnmi.Update, error) {
	if len(configValues) == 0 {
		emptyUpdate := gnmi.Update{
			Path: path,
			Val:  nil,
		}
		return []*gnmi.Update{
			&emptyUpdate,
		}, nil
	}

	switch encoding {
	case gnmi.Encoding_JSON, gnmi.Encoding_JSON_IETF:
		json, err := store.BuildTree(configValues, true)
		if err != nil {
			return nil, err
		}
		update := &gnmi.Update{
			Val: &gnmi.TypedValue{
				Value: &gnmi.TypedValue_JsonVal{
					JsonVal: json,
				},
			},
			Path: path,
		}
		return []*gnmi.Update{
			update,
		}, nil
	case gnmi.Encoding_PROTO:
		updates := make([]*gnmi.Update, 0, len(configValues))
		for _, cv := range configValues {
			gnmiVal, err := values.NativeTypeToGnmiTypedValue(cv.Value)
			if err != nil {
				return nil, err
			}
			prefixPath := ""
			if prefix != nil {
				prefixPath = utils.StrPathElem(prefix.Elem)
			}
			pathCv, err := utils.ParseGNMIElements(strings.Split(cv.Path[len(prefixPath)+1:], "/"))
			if err != nil {
				return nil, err
			}
			if path != nil {
				pathCv.Target = path.Target
				pathCv.Origin = path.Origin
			}
			update := &gnmi.Update{
				Path: pathCv,
				Val:  gnmiVal,
			}
			updates = append(updates, update)
		}
		return updates, nil
	default:
		return nil, fmt.Errorf("unsupported encoding %v", encoding)
	}

}

func extractGetVersion(req *gnmi.GetRequest) (devicetype.Version, error) {
	var version devicetype.Version
	for _, ext := range req.GetExtension() {
		if ext.GetRegisteredExt().GetId() == GnmiExtensionVersion {
			version = devicetype.Version(ext.GetRegisteredExt().GetMsg())
		} else {
			return "", status.Error(codes.InvalidArgument, fmt.Errorf("unexpected extension %d = '%s' in Get()",
				ext.GetRegisteredExt().GetId(), ext.GetRegisteredExt().GetMsg()).Error())
		}
	}
	return version, nil
}
