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
	"time"

	devicechange "github.com/onosproject/onos-api/go/onos/config/change/device"
	networkchange "github.com/onosproject/onos-api/go/onos/config/change/network"
	devicetype "github.com/onosproject/onos-api/go/onos/config/device"
	"github.com/onosproject/onos-config/pkg/manager"
	"github.com/onosproject/onos-config/pkg/modelregistry"
	"github.com/onosproject/onos-config/pkg/modelregistry/jsonvalues"
	"github.com/onosproject/onos-config/pkg/store/device/cache"
	"github.com/onosproject/onos-config/pkg/utils"
	"github.com/onosproject/onos-config/pkg/utils/values"
	"github.com/openconfig/gnmi/proto/gnmi"
	"github.com/openconfig/gnmi/proto/gnmi_ext"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type mapTargetUpdates map[devicetype.ID]devicechange.TypedValueMap
type mapTargetRemoves map[devicetype.ID][]string
type mapTargetModels map[devicetype.ID]modelregistry.ReadWritePathMap

// Set implements gNMI Set
func (s *Server) Set(ctx context.Context, req *gnmi.SetRequest) (*gnmi.SetResponse, error) {
	if md := metautils.ExtractIncoming(ctx); md != nil && md.Get("name") != "" {
		log.Infof("gNMI Set() called by '%s (%s)' with token %s",
			md.Get("name"), md.Get("email"), md.Get("at_hash"))
	}
	// There is only one set of extensions in Set request, regardless of number of
	// updates
	var (
		netCfgChangeName string             // May be specified as 100 in extension
		version          devicetype.Version // May be specified as 101 in extension
		deviceType       devicetype.Type    // May be specified as 102 in extension
	)

	targetUpdates := make(mapTargetUpdates)
	targetRemoves := make(mapTargetRemoves)
	targetModels := make(mapTargetModels)

	netCfgChangeName, version, deviceType, err := extractExtensions(req)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}

	log.Infof("gNMI Set Request %v", req)
	prefixTarget := devicetype.ID(req.GetPrefix().GetTarget())

	//Update - extract targets and their models
	for _, u := range req.GetUpdate() {
		target := devicetype.ID(u.Path.GetTarget())
		if target == "" { //Try the prefix
			target = prefixTarget
		}
		rwPaths, err := extractModelForTarget(target, version, deviceType, targetModels)
		if err != nil {
			return nil, err
		}
		targetUpdates[target], err = s.formatUpdateOrReplace(req.GetPrefix(), u, targetUpdates, rwPaths)
		if err != nil {
			return nil, err
		}
	}

	//Replace
	for _, u := range req.GetReplace() {
		target := devicetype.ID(u.Path.GetTarget())
		if target == "" { //Try the prefix
			target = prefixTarget
		}
		rwPaths, err := extractModelForTarget(target, version, deviceType, targetModels)
		if err != nil {
			return nil, err
		}
		targetUpdates[target], err = s.formatUpdateOrReplace(req.GetPrefix(), u, targetUpdates, rwPaths)
		if err != nil {
			log.Warn("Error in replace", err)
			return nil, status.Error(codes.InvalidArgument, err.Error())
		}
	}

	//Delete
	for _, u := range req.GetDelete() {
		target := devicetype.ID(u.GetTarget())
		if target == "" { //Try the prefix
			target = prefixTarget
		}
		rwPaths, err := extractModelForTarget(target, version, deviceType, targetModels)
		if err != nil {
			return nil, err
		}
		targetRemoves[target], err = s.doDelete(req.GetPrefix(), u, targetRemoves, rwPaths)
		if err != nil {
			return nil, fmt.Errorf("doDelete() %s", err.Error())
		}
	}

	//Temporary map in order to not to modify the original removes but optimize calculations during validation
	targetRemovesTmp := make(mapTargetRemoves)
	for k, v := range targetRemoves {
		targetRemovesTmp[k] = v
	}

	s.mu.RLock()
	lastWrite := s.lastWrite
	s.mu.RUnlock()

	mgr := manager.GetManager()
	deviceInfo := make(map[devicetype.ID]cache.Info)
	//Checking for wrong configuration against the device models for updates
	for target, updates := range targetUpdates {
		deviceType, version, err = mgr.CheckCacheForDevice(target, deviceType, version)
		if err != nil {
			return nil, status.Error(codes.InvalidArgument, err.Error())
		}
		deviceInfo[target] = cache.Info{
			DeviceID: target,
			Type:     deviceType,
			Version:  version,
		}

		// TODO: Since the change has not been stored yet, we cannot guarantee the change will be validated against
		//       the same state as will be pushed to the device. Changes must be validated after they're stored
		//       to achieve this level of consistency.
		err := validateChange(target, deviceType, version, updates, targetRemoves[target], lastWrite)
		if err != nil {
			return nil, status.Error(codes.InvalidArgument, err.Error())
		}
		delete(targetRemovesTmp, target)
	}
	//Checking for wrong configuration against the device models for deletes
	for target, removes := range targetRemovesTmp {
		deviceType, version, err = mgr.CheckCacheForDevice(target, deviceType, version)
		if err != nil {
			return nil, status.Error(codes.InvalidArgument, err.Error())
		}
		deviceInfo[target] = cache.Info{
			DeviceID: target,
			Type:     deviceType,
			Version:  version,
		}

		// TODO: Since the change has not been stored yet, we cannot guarantee the change will be validated against
		//       the same state as will be pushed to the device. Changes must be validated after they're stored
		//       to achieve this level of consistency.
		err := validateChange(target, deviceType, version, make(devicechange.TypedValueMap), removes, lastWrite)
		if err != nil {
			return nil, status.Error(codes.InvalidArgument, err.Error())
		}
	}

	// Creating and setting the config on the atomix Store
	change, errSet := mgr.SetNetworkConfig(targetUpdates, targetRemoves, deviceInfo, netCfgChangeName)
	if errSet != nil {
		log.Errorf("Error while setting config in atomix %s", errSet.Error())
		return nil, status.Error(codes.Internal, errSet.Error())
	}

	// Store the highest known change index
	s.mu.Lock()
	if change.Revision > s.lastWrite {
		s.lastWrite = change.Revision
	}
	s.mu.Unlock()

	// Build the responses
	updateResults := make([]*gnmi.UpdateResult, 0)
	for _, deviceChange := range change.Changes {
		deviceID := deviceChange.DeviceID
		for _, valueUpdate := range deviceChange.Values {
			var updateResult *gnmi.UpdateResult
			var errBuild error
			if valueUpdate.Removed {
				updateResult, errBuild = buildUpdateResult(valueUpdate.Path,
					string(deviceID), gnmi.UpdateResult_DELETE)
			} else {
				updateResult, errBuild = buildUpdateResult(valueUpdate.Path,
					string(deviceID), gnmi.UpdateResult_UPDATE)
			}
			if errBuild != nil {
				log.Error(errBuild)
				continue
			}
			updateResults = append(updateResults, updateResult)
		}
	}

	extensions := []*gnmi_ext.Extension{
		{
			Ext: &gnmi_ext.Extension_RegisteredExt{
				RegisteredExt: &gnmi_ext.RegisteredExtension{
					Id:  GnmiExtensionNetwkChangeID,
					Msg: []byte(change.ID),
				},
			},
		},
	}

	setResponse := &gnmi.SetResponse{
		Response:  updateResults,
		Timestamp: time.Now().Unix(),
		Extension: extensions,
	}

	return setResponse, nil
}

func extractExtensions(req *gnmi.SetRequest) (string, devicetype.Version, devicetype.Type, error) {
	var netcfgchangename string
	var version string
	var deviceType string
	for _, ext := range req.GetExtension() {
		if ext.GetRegisteredExt().GetId() == GnmiExtensionNetwkChangeID {
			netcfgchangename = string(ext.GetRegisteredExt().GetMsg())
		} else if ext.GetRegisteredExt().GetId() == GnmiExtensionVersion {
			version = string(ext.GetRegisteredExt().GetMsg())
		} else if ext.GetRegisteredExt().GetId() == GnmiExtensionDeviceType {
			deviceType = string(ext.GetRegisteredExt().GetMsg())
		} else {
			return "", "", "", status.Error(codes.InvalidArgument, fmt.Errorf("unexpected extension %d = '%s' in Set()",
				ext.GetRegisteredExt().GetId(), ext.GetRegisteredExt().GetMsg()).Error())
		}
	}
	log.Infof("Set called with extensions; 100: %s, 101: %s, 102: %s",
		netcfgchangename, version, deviceType)
	return netcfgchangename, devicetype.Version(version), devicetype.Type(deviceType), nil
}

// This deals with either a path and a value (simple case) or a path with
// a JSON body which implies multiple paths and values.
func (s *Server) formatUpdateOrReplace(prefix *gnmi.Path, u *gnmi.Update,
	targetUpdates mapTargetUpdates, rwPaths modelregistry.ReadWritePathMap) (devicechange.TypedValueMap, error) {
	target := devicetype.ID(u.Path.GetTarget())
	if target == "" {
		target = devicetype.ID(prefix.GetTarget())
	}
	prefixPath := utils.StrPath(prefix)
	path := utils.StrPath(u.Path)
	if prefixPath != "/" {
		path = fmt.Sprintf("%s%s", prefixPath, path)
	}

	updates, ok := targetUpdates[target]
	if !ok {
		updates = make(devicechange.TypedValueMap)
	}

	jsonVal := u.GetVal().GetJsonVal()
	if jsonVal != nil {
		log.Infof("Processing Json Value in set from base %s: %s",
			path, string(jsonVal))

		pathValues, err := jsonvalues.DecomposeJSONWithPaths(path, jsonVal, nil, rwPaths)
		if err != nil {
			log.Warnf("Json value in Set could not be parsed %v", err)
			return nil, err
		}
		if len(pathValues) == 0 {
			log.Warnf("no pathValues found for %s in %v", path, string(jsonVal))
		}
		for _, cv := range pathValues {
			updates[cv.Path] = cv.GetValue()
		}
	} else {
		rwPathElem, err := findPathFromModel(path, rwPaths)
		if err != nil {
			return nil, err
		}
		update, err := values.GnmiTypedValueToNativeType(u.Val, rwPathElem)
		if err != nil {
			return nil, err
		}
		updates[path] = update
	}

	return updates, nil

}

func (s *Server) doDelete(prefix *gnmi.Path, u *gnmi.Path,
	targetRemoves mapTargetRemoves, rwPaths modelregistry.ReadWritePathMap) ([]string, error) {

	target := devicetype.ID(u.GetTarget())
	if target == "" {
		target = devicetype.ID(prefix.GetTarget())
	}
	deletes, ok := targetRemoves[target]
	if !ok {
		deletes = make([]string, 0)
	}
	prefixPath := utils.StrPath(prefix)
	path := utils.StrPath(u)
	if prefixPath != "/" {
		path = fmt.Sprintf("%s%s", prefixPath, path)
	}
	// Checks for read only paths
	_, err := findPathFromModel(path, rwPaths)
	if err != nil {
		return nil, err
	}
	deletes = append(deletes, path)
	return deletes, nil

}

func buildUpdateResult(pathStr string, target string, op gnmi.UpdateResult_Operation) (*gnmi.UpdateResult, error) {
	path, errInPath := utils.ParseGNMIElements(utils.SplitPath(pathStr))
	if errInPath != nil {
		log.Error("ERROR: Unable to parse path ", pathStr)
		return nil, errInPath
	}
	path.Target = target
	updateResult := &gnmi.UpdateResult{
		Path: path,
		Op:   op,
	}
	return updateResult, nil

}

func validateChange(target devicetype.ID, deviceType devicetype.Type, version devicetype.Version,
	targetUpdates devicechange.TypedValueMap, targetRemoves []string, lastWrite networkchange.Revision) error {
	if len(targetUpdates) == 0 && len(targetRemoves) == 0 {
		return fmt.Errorf("no updates found in change on %s - invalid", target)
	}
	log.Infof("Validating change %s:%s:%s", target, deviceType, version)
	errValidation := manager.GetManager().ValidateNetworkConfig(target, version, deviceType,
		targetUpdates, targetRemoves, lastWrite)
	if errValidation != nil {
		log.Errorf("Error in validating config, updates %s, removes %s for target %s, err: %s", targetUpdates,
			targetRemoves, target, errValidation)
		return errValidation
	}
	log.Infof("Validating change %s:%s:%s DONE", target, deviceType, version)
	return nil
}

func extractModelForTarget(target devicetype.ID,
	ext101Version devicetype.Version, ext102Type devicetype.Type,
	targetModels mapTargetModels) (modelregistry.ReadWritePathMap, error) {

	if target == "" {
		return nil, status.Error(codes.InvalidArgument, "no target given")
	}
	if rwPaths, hasModel := targetModels[target]; hasModel {
		return rwPaths, nil
	}

	actualType, actualVersion, err := manager.GetManager().CheckCacheForDevice(target, ext102Type, ext101Version)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}
	modelName := utils.ToModelName(actualType, actualVersion)
	rwPaths, ok := manager.GetManager().ModelRegistry.ModelReadWritePaths[modelName]
	if !ok {
		return nil, status.Errorf(codes.InvalidArgument,
			"unable to find model registry for %s (for target %s)", modelName, target)
	}
	targetModels[target] = rwPaths
	return rwPaths, nil
}

func findPathFromModel(path string, rwPaths modelregistry.ReadWritePathMap) (*modelregistry.ReadWritePathElem, error) {
	searchpathNoIndices := modelregistry.RemovePathIndices(path)
	// First search through the RW paths
	var rwPathElem modelregistry.ReadWritePathElem
	var ok bool
	for modelPath, modelElem := range rwPaths {
		pathNoIndices := modelregistry.RemovePathIndices(modelPath)
		// Find a short path
		if pathNoIndices == searchpathNoIndices {
			rwPathElem = modelElem
			ok = true
			break
		}
	}
	if !ok {
		return nil, fmt.Errorf("unable to find RW model path %s", path)
	}
	return &rwPathElem, nil
}
