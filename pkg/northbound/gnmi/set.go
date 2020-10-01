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
	"strings"
	"time"

	devicechange "github.com/onosproject/onos-config/api/types/change/device"
	networkchange "github.com/onosproject/onos-config/api/types/change/network"
	devicetype "github.com/onosproject/onos-config/api/types/device"
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

type mapTargetUpdates map[string]devicechange.TypedValueMap
type mapTargetRemoves map[string][]string

// Set implements gNMI Set
func (s *Server) Set(ctx context.Context, req *gnmi.SetRequest) (*gnmi.SetResponse, error) {
	// There is only one set of extensions in Set request, regardless of number of
	// updates
	var (
		netCfgChangeName string             // May be specified as 100 in extension
		version          devicetype.Version // May be specified as 101 in extension
		deviceType       devicetype.Type    // May be specified as 102 in extension
	)

	targetUpdates := make(mapTargetUpdates)
	targetRemoves := make(mapTargetRemoves)

	log.Infof("gNMI Set Request %v", req)
	//Update
	for _, u := range req.GetUpdate() {
		target := u.Path.GetTarget()
		if target == "" { //Try the prefix
			target = req.GetPrefix().GetTarget()
		}
		if target == "" {
			return nil, status.Error(codes.InvalidArgument, "no target given in update")
		}
		var err error
		targetUpdates[target], err = s.formatUpdateOrReplace(req.GetPrefix(), u, targetUpdates)
		if err != nil {
			log.Warn("Error in update ", err)
			return nil, status.Error(codes.InvalidArgument, err.Error())
		}
	}

	//Replace
	for _, u := range req.GetReplace() {
		target := u.Path.GetTarget()
		if target == "" { //Try the prefix
			target = req.GetPrefix().GetTarget()
		}
		if target == "" {
			return nil, status.Errorf(codes.InvalidArgument, "No target given in update %v", u)
		}
		var err error
		targetUpdates[target], err = s.formatUpdateOrReplace(req.GetPrefix(), u, targetUpdates)
		if err != nil {
			log.Warn("Error in replace", err)
			return nil, status.Error(codes.InvalidArgument, err.Error())
		}
	}

	//Delete
	for _, u := range req.GetDelete() {
		target := u.GetTarget()
		if target == "" { //Try the prefix
			target = req.GetPrefix().GetTarget()
		}
		if target == "" {
			return nil, status.Errorf(codes.InvalidArgument, "No target given in delete %v", u)
		}
		targetRemoves[target] = s.doDelete(req.GetPrefix(), u, targetRemoves)
	}

	netCfgChangeName, version, deviceType, err := extractExtensions(req)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
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
		deviceType, version, err = mgr.CheckCacheForDevice(devicetype.ID(target), deviceType, version)
		if err != nil {
			return nil, status.Error(codes.InvalidArgument, err.Error())
		}
		deviceInfo[devicetype.ID(target)] = cache.Info{
			DeviceID: devicetype.ID(target),
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

		err = s.checkForReadOnly(target, deviceType, version, updates, targetRemoves[target])
		if err != nil {
			return nil, status.Error(codes.InvalidArgument, err.Error())
		}
		delete(targetRemovesTmp, target)
	}
	//Checking for wrong configuration against the device models for deletes
	for target, removes := range targetRemovesTmp {
		deviceType, version, err = mgr.CheckCacheForDevice(devicetype.ID(target), deviceType, version)
		if err != nil {
			return nil, status.Error(codes.InvalidArgument, err.Error())
		}
		deviceInfo[devicetype.ID(target)] = cache.Info{
			DeviceID: devicetype.ID(target),
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

		err = s.checkForReadOnly(target, deviceType, version, make(devicechange.TypedValueMap), removes)
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
	targetUpdates mapTargetUpdates) (devicechange.TypedValueMap, error) {
	target := u.Path.GetTarget()
	if target == "" {
		target = prefix.GetTarget()
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

		var rwPaths modelregistry.ReadWritePathMap
		infos := manager.GetManager().DeviceCache.GetDevicesByID(devicetype.ID(target))
		if len(infos) == 0 {
			return nil, fmt.Errorf("cannot process JSON payload because "+
				"device %s is not in DeviceCache", target)
		}
		// Iterate through configs to find match for target
		for _, info := range infos {
			rwPaths, ok = manager.GetManager().ModelRegistry.
				ModelReadWritePaths[utils.ToModelName(info.Type, info.Version)]
			if ok {
				break
			}
		}
		if rwPaths == nil {
			return nil, fmt.Errorf("cannot process JSON payload because "+
				"Model Plugin not available for target %s", target)
		}

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

		update, err := values.GnmiTypedValueToNativeType(u.Val)
		if err != nil {
			return nil, err
		}
		updates[path] = update
	}

	return updates, nil

}

func (s *Server) doDelete(prefix *gnmi.Path, u *gnmi.Path, targetRemoves mapTargetRemoves) []string {
	target := u.GetTarget()
	if target == "" {
		target = prefix.GetTarget()
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
	deletes = append(deletes, path)
	return deletes

}

// iterate through the updates and check that none of them include a `set` of a
// readonly attribute - this is done by checking with the relevant model
func (s *Server) checkForReadOnly(target string, deviceType devicetype.Type, version devicetype.Version,
	targetUpdates devicechange.TypedValueMap, targetRemoves []string) error {

	modelreg := manager.GetManager().ModelRegistry

	// This ignores versions - if it's RO in one version will be regarded
	// as RO in all versions - very unlikely that modelRoPaths would change
	// YANG items from `config false` to `config true` across versions
	modelRoPaths, ok := modelreg.
		ModelReadOnlyPaths[utils.ToModelName(deviceType, version)]
	if !ok {
		log.Warnf("Cannot check for Read Only paths for %s %s because "+
			"Model Plugin not available - continuing", deviceType, version)
		return nil
	}

	modelRwPaths, ok := modelreg.
		ModelReadWritePaths[utils.ToModelName(deviceType, version)]
	if !ok {
		log.Warnf("Cannot check for Read Only paths for %s %s because "+
			"Model Plugin not available - continuing", deviceType, version)
		return nil
	}

	// Now iterate through the consolidated set of targets and see if any are read-only paths
	for path := range targetUpdates { // map - just need the key
		if err := compareRoPaths(path, modelRoPaths, modelRwPaths); err != nil {
			return fmt.Errorf("update %s", err)
		}
	}

	// Now iterate through the consolidated set of targets and see if any are read-only paths
	for _, path := range targetRemoves { // map - just need the key
		if err := compareRoPaths(path, modelRoPaths, modelRwPaths); err != nil {
			return fmt.Errorf("remove %s", err)
		}
	}

	return nil
}

func compareRoPaths(path string, modelRoPaths modelregistry.ReadOnlyPathMap, modelRwPaths modelregistry.ReadWritePathMap) error {
	log.Infof("Testing %s for read only", path)
	for ropath, subpaths := range modelRoPaths {
		// Search through for list indices and replace with generic
		modelPathNiIdx := modelregistry.RemovePathIndices(path)
		ropathNoIdx := modelregistry.RemovePathIndices(ropath)
		if strings.HasPrefix(modelPathNiIdx, ropathNoIdx) {
			for s := range subpaths {
				fullpath := ropathNoIdx
				if s != "/" {
					fullpath = fmt.Sprintf("%s%s", ropathNoIdx, s)
				}
				if fullpath == modelPathNiIdx {
					// Check that this is not one of those in both config and state (e.g. index of a list)
					for rwpath := range modelRwPaths {
						rwpathNoIdx := modelregistry.RemovePathIndices(rwpath)
						if rwpathNoIdx == modelPathNiIdx {
							return nil
						}
					}
					return fmt.Errorf("contains a change to a "+
						"read only path %s. Rejected. %s, %s, %s, %s, %s",
						path, modelPathNiIdx, ropath, ropathNoIdx, s, fullpath)
				}
			}
		}
	}
	return nil
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

func validateChange(target string, deviceType devicetype.Type, version devicetype.Version,
	targetUpdates devicechange.TypedValueMap, targetRemoves []string, lastWrite networkchange.Revision) error {
	if len(targetUpdates) == 0 && len(targetRemoves) == 0 {
		return fmt.Errorf("no updates found in change on %s - invalid", target)
	}
	log.Infof("Validating change %s:%s:%s", target, deviceType, version)
	errValidation := manager.GetManager().ValidateNetworkConfig(devicetype.ID(target), version, deviceType,
		targetUpdates, targetRemoves, lastWrite)
	if errValidation != nil {
		log.Errorf("Error in validating config, updates %s, removes %s for target %s, err: %s", targetUpdates,
			targetRemoves, target, errValidation)
		return errValidation
	}
	log.Infof("Validating change %s:%s:%s DONE", target, deviceType, version)
	return nil
}
