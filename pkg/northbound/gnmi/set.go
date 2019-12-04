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
	"github.com/docker/docker/pkg/namesgenerator"
	changetypes "github.com/onosproject/onos-config/api/types/change"
	devicechange "github.com/onosproject/onos-config/api/types/change/device"
	networkchange "github.com/onosproject/onos-config/api/types/change/network"
	devicetype "github.com/onosproject/onos-config/api/types/device"
	"github.com/onosproject/onos-config/pkg/manager"
	"github.com/onosproject/onos-config/pkg/modelregistry"
	"github.com/onosproject/onos-config/pkg/modelregistry/jsonvalues"
	"github.com/onosproject/onos-config/pkg/store"
	networkchangestore "github.com/onosproject/onos-config/pkg/store/change/network"
	"github.com/onosproject/onos-config/pkg/store/device/cache"
	"github.com/onosproject/onos-config/pkg/store/stream"
	"github.com/onosproject/onos-config/pkg/utils"
	"github.com/onosproject/onos-config/pkg/utils/values"
	topodevice "github.com/onosproject/onos-topo/api/device"
	"github.com/openconfig/gnmi/proto/gnmi"
	"github.com/openconfig/gnmi/proto/gnmi_ext"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"strings"
	"time"
)

type mapTargetUpdates map[string]devicechange.TypedValueMap
type mapTargetRemoves map[string][]string

// Set implements gNMI Set
func (s *Server) Set(ctx context.Context, req *gnmi.SetRequest) (*gnmi.SetResponse, error) {
	// There is only one set of extensions in Set request, regardless of number of
	// updates
	var (
		netCfgChangeName    string             // May be specified as 100 in extension
		version             devicetype.Version // May be specified as 101 in extension
		deviceType          devicetype.Type    // May be specified as 102 in extension
		disconnectedDevices []string
	)

	disconnectedDevices = make([]string, 0)
	targetUpdates := make(mapTargetUpdates)
	targetRemoves := make(mapTargetRemoves)

	log.Info("gNMI Set Request", req)
	s.mu.Lock()
	defer s.mu.Unlock()
	//Update
	for _, u := range req.GetUpdate() {
		target := u.Path.GetTarget()
		var err error
		targetUpdates[target], err = s.formatUpdateOrReplace(u, targetUpdates)
		if err != nil {
			log.Warn("Error in update ", err)
			return nil, status.Error(codes.InvalidArgument, err.Error())
		}
	}

	//Replace
	for _, u := range req.GetReplace() {
		target := u.Path.GetTarget()
		var err error
		targetUpdates[target], err = s.formatUpdateOrReplace(u, targetUpdates)
		if err != nil {
			log.Warn("Error in replace", err)
			return nil, status.Error(codes.InvalidArgument, err.Error())
		}
	}

	//Delete
	for _, u := range req.GetDelete() {
		target := u.GetTarget()
		targetRemoves[target] = s.doDelete(u, targetRemoves)
	}

	netCfgChangeName, version, deviceType, err := extractExtensions(req)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}

	if netCfgChangeName == "" {
		netCfgChangeName = namesgenerator.GetRandomName(0)
	}

	//Temporary map in order to not to modify the original removes but optimize calculations during validation
	targetRemovesTmp := make(mapTargetRemoves)
	for k, v := range targetRemoves {
		targetRemovesTmp[k] = v
	}

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

		_, errDevice := mgr.DeviceStore.Get(topodevice.ID(target))
		if errDevice != nil && status.Convert(errDevice).Code() == codes.NotFound {
			disconnectedDevices = append(disconnectedDevices, target)
		} else if errDevice != nil {
			//handling gRPC errors
			return nil, status.Error(codes.Internal, errDevice.Error())
		}

		err := validateChange(target, deviceType, version, updates, targetRemoves[target])
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

		_, errDevice := mgr.DeviceStore.Get(topodevice.ID(target))
		if errDevice != nil && status.Convert(errDevice).Code() == codes.NotFound {
			disconnectedDevices = append(disconnectedDevices, target)
		} else if errDevice != nil {
			//handling gRPC errors
			return nil, status.Error(codes.Internal, errDevice.Error())
		}

		err := validateChange(target, deviceType, version, make(devicechange.TypedValueMap), removes)
		if err != nil {
			return nil, status.Error(codes.InvalidArgument, err.Error())
		}

		err = s.checkForReadOnly(target, deviceType, version, make(devicechange.TypedValueMap), removes)
		if err != nil {
			return nil, status.Error(codes.InvalidArgument, err.Error())
		}
	}

	//Creating and setting the config on the atomix Store
	errSet := mgr.SetNetworkConfig(targetUpdates, targetRemoves, deviceInfo, netCfgChangeName, false)

	if errSet != nil {
		log.Errorf("Error while setting config in atomix %s", errSet.Error())
		return nil, status.Error(codes.Internal, errSet.Error())
	}

	// TODO: Not clear if there is a period of time where this misses out on events
	//Obtaining response based on distributed store generated events
	updateResultsAtomix, errListen := listenAndBuildResponse(mgr, networkchange.ID(netCfgChangeName))
	if errListen != nil {
		log.Errorf("Error while building atomix based response %s", errListen.Error())
		return nil, status.Error(codes.Internal, errListen.Error())
	}

	extensions := []*gnmi_ext.Extension{
		{
			Ext: &gnmi_ext.Extension_RegisteredExt{
				RegisteredExt: &gnmi_ext.RegisteredExtension{
					Id:  GnmiExtensionNetwkChangeID,
					Msg: []byte(netCfgChangeName),
				},
			},
		},
	}

	if len(disconnectedDevices) != 0 {
		disconnectedDeviceString := strings.Join(disconnectedDevices, ",")
		disconnectedExt := &gnmi_ext.Extension{
			Ext: &gnmi_ext.Extension_RegisteredExt{
				RegisteredExt: &gnmi_ext.RegisteredExtension{
					Id:  GnmiExtensionDevicesNotConnected,
					Msg: []byte(disconnectedDeviceString),
				},
			},
		}
		extensions = append(extensions, disconnectedExt)
	}

	setResponse := &gnmi.SetResponse{
		Response:  updateResultsAtomix,
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
		log.Infof("Set called with extensions; 100: %s, 101: %s, 102: %s",
			netcfgchangename, version, deviceType)
	}
	return netcfgchangename, devicetype.Version(version), devicetype.Type(deviceType), nil
}

// This deals with either a path and a value (simple case) or a path with
// a JSON body which implies multiple paths and values.
func (s *Server) formatUpdateOrReplace(u *gnmi.Update,
	targetUpdates mapTargetUpdates) (devicechange.TypedValueMap, error) {
	target := u.Path.GetTarget()
	updates, ok := targetUpdates[target]
	if !ok {
		updates = make(devicechange.TypedValueMap)
	}

	jsonVal := u.GetVal().GetJsonVal()
	if jsonVal != nil {
		log.Info("Processing Json Value in set", string(jsonVal))

		intermediateConfigValues, err := store.DecomposeTree(jsonVal)
		if err != nil {
			return nil, fmt.Errorf("invalid JSON payload %s", string(jsonVal))
		}

		var rwPaths modelregistry.ReadWritePathMap
		infos := manager.GetManager().DeviceCache.GetDevicesByID(devicetype.ID(target))
		// Iterate through configs to find match for target
		for _, info := range infos {
			rwPaths, ok = manager.GetManager().ModelRegistry.
				ModelReadWritePaths[utils.ToModelName(info.Type, info.Version)]
			if ok {
				break
			}
		}
		if rwPaths == nil {
			return nil, fmt.Errorf("Cannot process JSON payload because "+
				"Model Plugin not available for target %s", target)
		}
		correctedValues, err := jsonvalues.CorrectJSONPaths(
			utils.StrPath(u.Path), intermediateConfigValues, rwPaths, true)
		if err != nil {
			log.Warn("Json value in Set could not be parsed", err)
			return nil, err
		}

		for _, cv := range correctedValues {
			updates[cv.Path] = cv.GetValue()
		}
	} else {
		path := utils.StrPath(u.Path)
		update, err := values.GnmiTypedValueToNativeType(u.Val)
		if err != nil {
			return nil, err
		}
		updates[path] = update
	}

	return updates, nil

}

func (s *Server) doDelete(u *gnmi.Path, targetRemoves mapTargetRemoves) []string {
	target := u.GetTarget()
	deletes, ok := targetRemoves[target]
	if !ok {
		deletes = make([]string, 0)
	}
	path := utils.StrPath(u)
	deletes = append(deletes, path)
	return deletes

}

// iterate through the updates and check that none of them include a `set` of a
// readonly attribute - this is done by checking with the relevant model
func (s *Server) checkForReadOnly(target string, deviceType devicetype.Type, version devicetype.Version,
	targetUpdates devicechange.TypedValueMap, targetRemoves []string) error {

	modelreg := manager.GetManager().ModelRegistry

	// This ignores versions - if it's RO in one version will be regarded
	// as RO in all versions - very unlikely that model would change
	// YANG items from `config false` to `config true` across versions
	model, ok := modelreg.
		ModelReadOnlyPaths[utils.ToModelName(deviceType, version)]
	if !ok {
		log.Warnf("Cannot check for Read Only paths for %s %s because "+
			"Model Plugin not available - continuing", deviceType, version)
		return nil
	}

	// Now iterate through the consolidated set of targets and see if any are read-only paths
	for path := range targetUpdates { // map - just need the key
		log.Infof("Testing %s for read only", path)
		for ropath := range model {
			// Search through for list indices and replace with generic
			modelPath := modelregistry.RemovePathIndices(path)
			if strings.HasPrefix(modelPath, ropath) {
				return fmt.Errorf("update contains a change to a read only path %s. Rejected", path)
			}
		}
	}

	// Now iterate through the consolidated set of targets and see if any are read-only paths
	for _, path := range targetRemoves { // map - just need the key
		log.Infof("Testing %s for read only", path)
		for ropath := range model {
			// Search through for list indices and replace with generic
			modelPath := modelregistry.RemovePathIndices(path)
			if strings.HasPrefix(modelPath, ropath) {
				return fmt.Errorf("remove contains a change to a read only path %s. Rejected", path)
			}
		}
	}

	return nil
}

func listenAndBuildResponse(mgr *manager.Manager, changeID networkchange.ID) ([]*gnmi.UpdateResult, error) {
	networkChan := make(chan stream.Event)
	ctx, errWatch := mgr.NetworkChangesStore.Watch(networkChan, networkchangestore.WithChangeID(changeID))
	if errWatch != nil {
		return nil, fmt.Errorf("can't complete set operation on target due to %s", errWatch)
	}
	defer ctx.Close()
	updateResults := make([]*gnmi.UpdateResult, 0)
	for changeEvent := range networkChan {
		change := changeEvent.Object.(*networkchange.NetworkChange)
		log.Infof("Received notification for change ID %s, phase %s, state %s", change.ID,
			change.Status.Phase, change.Status.State)
		if change.Status.Phase == changetypes.Phase_CHANGE {
			switch changeStatus := change.Status.State; changeStatus {
			case changetypes.State_COMPLETE:
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
			case changetypes.State_FAILED:
				return nil, fmt.Errorf("issue in setting config reson %s, error %s, rolling back change %s",
					change.Status.Reason, change.Status.Message, changeID)
			default:
				continue
			}
			break
		}
	}
	return updateResults, nil
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
	targetUpdates devicechange.TypedValueMap, targetRemoves []string) error {
	if len(targetUpdates) == 0 && len(targetRemoves) == 0 {
		return fmt.Errorf("no updates found in change on %s - invalid", target)
	}
	log.Infof("Validating change %s:%s:%s", target, deviceType, version)
	errValidation := manager.GetManager().ValidateNetworkConfig(devicetype.ID(target), version, deviceType,
		targetUpdates, targetRemoves)
	if errValidation != nil {
		log.Errorf("Error in validating config, updates %s, removes %s for target %s, err: %s", targetUpdates,
			targetRemoves, target, errValidation)
		return errValidation
	}
	log.Infof("Validating change %s:%s:%s DONE", target, deviceType, version)
	return nil
}
