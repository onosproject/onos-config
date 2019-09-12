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
	"github.com/onosproject/onos-config/pkg/events"
	"github.com/onosproject/onos-config/pkg/manager"
	"github.com/onosproject/onos-config/pkg/modelregistry"
	"github.com/onosproject/onos-config/pkg/modelregistry/jsonvalues"
	"github.com/onosproject/onos-config/pkg/store"
	"github.com/onosproject/onos-config/pkg/store/change"
	"github.com/onosproject/onos-config/pkg/utils"
	"github.com/onosproject/onos-config/pkg/utils/values"
	"github.com/onosproject/onos-topo/pkg/northbound/device"
	"github.com/openconfig/gnmi/proto/gnmi"
	"github.com/openconfig/gnmi/proto/gnmi_ext"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	log "k8s.io/klog"
	"strings"
	"time"
)

type mapTargetUpdates map[string]change.TypedValueMap
type mapTargetRemoves map[string][]string
type mapNetworkChanges map[store.ConfigName]change.ID

// Set implements gNMI Set
func (s *Server) Set(ctx context.Context, req *gnmi.SetRequest) (*gnmi.SetResponse, error) {
	// There is only one set of extensions in Set request, regardless of number of
	// updates
	var (
		netcfgchangename string // May be specified as 100 in extension
		version          string // May be specified as 101 in extension
		deviceType       string // May be specified as 102 in extension
	)
	targetUpdates := make(mapTargetUpdates)
	targetRemoves := make(mapTargetRemoves)

	s.mu.Lock()
	defer s.mu.Unlock()
	//Update
	for _, u := range req.GetUpdate() {
		target := u.Path.GetTarget()
		var err error
		targetUpdates[target], err = s.formatUpdateOrReplace(u, targetUpdates)
		if err != nil {
			log.Warning("Error in update ", err)
			return nil, err
		}
	}

	//Replace
	for _, u := range req.GetReplace() {
		target := u.Path.GetTarget()
		var err error
		targetUpdates[target], err = s.formatUpdateOrReplace(u, targetUpdates)
		if err != nil {
			log.Warning("Error in replace", err)
			return nil, err
		}
	}

	//Delete
	for _, u := range req.GetDelete() {
		target := u.GetTarget()
		targetRemoves[target] = s.doDelete(u, targetRemoves)
	}

	for _, ext := range req.GetExtension() {
		if ext.GetRegisteredExt().GetId() == GnmiExtensionNetwkChangeID {
			netcfgchangename = string(ext.GetRegisteredExt().GetMsg())
		} else if ext.GetRegisteredExt().GetId() == GnmiExtensionVersion {
			version = string(ext.GetRegisteredExt().GetMsg())
		} else if ext.GetRegisteredExt().GetId() == GnmiExtensionDeviceType {
			deviceType = string(ext.GetRegisteredExt().GetMsg())
		} else {
			return nil, status.Error(codes.InvalidArgument, fmt.Errorf("unexpected extension %d = '%s' in Set()",
				ext.GetRegisteredExt().GetId(), ext.GetRegisteredExt().GetMsg()).Error())
		}
	}
	log.Infof("Set called with extensions; 100: %s, 101: %s, 102: %s",
		netcfgchangename, version, deviceType)
	errRo := s.checkForReadOnly(deviceType, version, targetUpdates, targetRemoves)
	if errRo != nil {
		return nil, status.Error(codes.InvalidArgument, errRo.Error())
	}

	if netcfgchangename == "" {
		netcfgchangename = namesgenerator.GetRandomName(0)
	}

	//Temporary map in order to not to modify the original removes but optimize calculations during validation
	targetRemovesTmp := make(mapTargetRemoves)
	for k, v := range targetRemoves {
		targetRemovesTmp[k] = v
	}

	//TODO this can be parallelized with a pattern manager.go ValidateStores()
	//Checking for wrong configuration against the device models for updates
	for target, updates := range targetUpdates {
		err := validateChange(target, version, deviceType, updates, targetRemoves[target])
		if err != nil {
			return nil, err
		}
		delete(targetRemovesTmp, target)
	}
	//Checking for wrong configuration against the device models for deletes
	for target, removes := range targetRemovesTmp {
		err := validateChange(target, version, deviceType, make(change.TypedValueMap), removes)
		if err != nil {
			return nil, err
		}
	}

	updateResults, networkChanges, err :=
		s.executeSetConfig(targetUpdates, targetRemoves, version, deviceType, netcfgchangename)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	if len(updateResults) == 0 {
		log.Warning("All target changes were duplicated - Set rejected")
		return nil, status.Error(codes.AlreadyExists, fmt.Errorf("set change rejected as it is a duplicate of the last change for all targets").Error())
	}

	// Look for use of this name already
	for _, nwCfg := range manager.GetManager().NetworkStore.Store {
		if nwCfg.Name == netcfgchangename {
			return nil, status.Error(codes.InvalidArgument, fmt.Errorf(
				"name %s is already used for a Network Configuration", netcfgchangename).Error())
		}
	}

	networkConfig, err := store.NewNetworkConfiguration(netcfgchangename, "User1", networkChanges)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	manager.GetManager().NetworkStore.Store =
		append(manager.GetManager().NetworkStore.Store, *networkConfig)

	setResponse := &gnmi.SetResponse{
		Response:  updateResults,
		Timestamp: time.Now().Unix(),
		Extension: []*gnmi_ext.Extension{
			{
				Ext: &gnmi_ext.Extension_RegisteredExt{
					RegisteredExt: &gnmi_ext.RegisteredExtension{
						Id:  100,
						Msg: []byte(networkConfig.Name),
					},
				},
			},
		},
	}
	return setResponse, nil
}

// This deals with either a path and a value (simple case) or a path with
// a JSON body which implies multiple paths and values.
func (s *Server) formatUpdateOrReplace(u *gnmi.Update, targetUpdates mapTargetUpdates) (change.TypedValueMap, error) {
	target := u.Path.GetTarget()
	updates, ok := targetUpdates[target]
	if !ok {
		updates = make(change.TypedValueMap)
	}

	jsonVal := u.GetVal().GetJsonVal()
	if jsonVal != nil {
		log.Info("Processing Json Value in set", string(jsonVal))

		intermediateConfigValues, err := store.DecomposeTree(jsonVal)
		if err != nil {
			return nil, fmt.Errorf("invalid JSON payload %s", string(jsonVal))
		}

		configs := manager.GetManager().ConfigStore.Store

		var rwPaths modelregistry.ReadWritePathMap
		// Iterate through configs to find match for target
		for _, config := range configs {
			if config.Device == target {
				rwPaths, ok = manager.GetManager().ModelRegistry.
					ModelReadWritePaths[utils.ToModelName(config.Type, config.Version)]
				if !ok {
					return nil, fmt.Errorf("Cannot process JSON payload  on %s %s because "+
						"Model Plugin not available", config.Type, config.Version)
				}
				break
			}
		}

		correctedValues, err := jsonvalues.CorrectJSONPaths(
			utils.StrPath(u.Path), intermediateConfigValues, rwPaths, true)
		if err != nil {
			log.Warning("Json value in Set could not be parsed", err)
			return nil, err
		}

		for _, cv := range correctedValues {
			updates[cv.Path] = &cv.TypedValue
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

func (s *Server) checkForReadOnly(deviceType string, version string, targetUpdates mapTargetUpdates,
	targetRemoves mapTargetRemoves) error {
	configs := manager.GetManager().ConfigStore.Store

	// Iterate through all the updates - many may use the same target - here we
	// create a map of the models for all of the targets
	targetModelTypes := make(map[string][]string)
	for t := range targetUpdates { // map - just need the key
		if _, ok := targetModelTypes[t]; ok {
			continue
		}
		for _, config := range configs {
			if config.Device == t {
				m, ok := manager.GetManager().ModelRegistry.
					ModelReadOnlyPaths[utils.ToModelName(config.Type, config.Version)]
				if !ok {
					log.Warningf("Cannot check for Read Only paths for %s %s because "+
						"Model Plugin not available - continuing", config.Type, config.Version)
					return nil
				}
				targetModelTypes[t] = modelregistry.Paths(m)
			}
		}
	}
	for t := range targetRemoves { // map - just need the key
		if _, ok := targetModelTypes[t]; ok {
			continue
		}
		for _, config := range configs {
			if config.Device == t {
				m, ok := manager.GetManager().ModelRegistry.
					ModelReadOnlyPaths[utils.ToModelName(config.Type, config.Version)]
				if !ok {
					log.Warningf("Cannot check for Read Only paths for %s %s because "+
						"Model Plugin not available - continuing", config.Type, config.Version)
					return nil
				}
				targetModelTypes[t] = modelregistry.Paths(m)
			}
		}
	}

	// Now iterate through the consolidated set of targets and see if any are read-only paths
	for t, update := range targetUpdates {
		model := targetModelTypes[t]
		for path := range update { // map - just need the key
			for _, ropath := range model {
				// Search through for list indices and replace with generic

				modelPath := modelregistry.RemovePathIndices(path)
				if strings.HasPrefix(modelPath, ropath) {
					return fmt.Errorf("update contains a change to a read only path %s. Rejected", path)
				}
			}
		}
	}

	return nil
}

func (s *Server) executeSetConfig(targetUpdates mapTargetUpdates,
	targetRemoves mapTargetRemoves, version string, deviceType string,
	description string) ([]*gnmi.UpdateResult, mapNetworkChanges, error) {

	networkChanges := make(mapNetworkChanges)
	updateResults := make([]*gnmi.UpdateResult, 0)
	for target, updates := range targetUpdates {
		//FIXME this is a sequential job, not parallelized

		// target is a device name with no version
		changeID, configName, cont, err := setChange(target, version, deviceType,
			updates, targetRemoves[target], description)
		//if the error is not nil and we need to continue do so
		if err != nil && !cont {
			//Rolling back in case of setChange error.
			return nil, nil, doRollback(networkChanges, manager.GetManager(), target, *configName, err)
		} else if err == nil && cont {
			continue
		}
		responseErr := listenForDeviceResponse(networkChanges, target, *configName)
		//TODO Maybe do this with a callback mechanism --> when resposne on channel is done trigger callback
		// that sends reply
		if responseErr != nil {
			return nil, nil, fmt.Errorf("can't complete set operation on target %s due to %s", target, responseErr)
		}
		for k := range updates {
			updateResult, err := buildUpdateResult(k, target, gnmi.UpdateResult_UPDATE)
			if err != nil {
				continue
			}
			updateResults = append(updateResults, updateResult)
		}

		for _, r := range targetRemoves[target] {
			updateResult, err := buildUpdateResult(r, target, gnmi.UpdateResult_DELETE)
			if err != nil {
				continue
			}
			updateResults = append(updateResults, updateResult)
			//Removing from targetRemoves since a pass was already done for this target
			delete(targetRemoves, target)
		}

		networkChanges[*configName] = changeID
	}

	for target, removes := range targetRemoves {
		changeID, configName, cont, err := setChange(target, version, deviceType,
			make(change.TypedValueMap), removes, description)
		//if the error is not nil and we need to continue do so
		if err != nil && !cont {
			return nil, nil, err
		} else if err == nil && cont {
			continue

		}
		responseErr := listenForDeviceResponse(networkChanges, target, *configName)
		if responseErr != nil {
			return nil, nil, responseErr
		}
		for _, r := range removes {
			updateResult, err := buildUpdateResult(r, target, gnmi.UpdateResult_DELETE)
			if err != nil {
				continue
			}
			updateResults = append(updateResults, updateResult)
		}
		networkChanges[*configName] = changeID
	}
	return updateResults, networkChanges, nil
}

func listenForDeviceResponse(changes mapNetworkChanges, target string, name store.ConfigName) error {
	mgr := manager.GetManager()
	respChan, ok := mgr.Dispatcher.GetResponseListener(device.ID(target))
	if !ok {
		log.Infof("Device %s not properly registered, not waiting for southbound confirmation ", target)
		return nil
	}
	//blocking until we receive something from the channel or for 5 seconds, whatever comes first.
	select {
	case response := <-respChan:
		switch eventType := response.EventType(); eventType {
		case events.EventTypeAchievedSetConfig:
			log.Info("Set is properly configured ", response.ChangeID())
			return nil
		case events.EventTypeErrorSetConfig:
			//Removing previously applied config
			return doRollback(changes, mgr, target, name, response.Error())

		default:
			return fmt.Errorf("undhandled Error Type")

		}
	case <-time.After(5 * time.Second):
		return fmt.Errorf("Timeout on waiting for device reply %s", target)
	}
}

func doRollback(changes mapNetworkChanges, mgr *manager.Manager, target string,
	name store.ConfigName, errResp error) error {
	rolledbackIDs := make([]string, 0)
	for configName := range changes {
		changeID, err := mgr.RollbackTargetConfig(configName)
		if err != nil {
			log.Errorf("Can't remove last entry for %s, on config %s, err, %v", target, name, err)
		}
		log.Infof("Rolled back %s on %s", store.B64(changeID), configName)
		rolledbackIDs = append(rolledbackIDs, store.B64(changeID))
	}
	//Removing the failed target but not computing the delta singe gNMI ensures device transactionality
	id, err := mgr.ConfigStore.RemoveLastChangeEntry(store.ConfigName(name))
	if err != nil {
		return fmt.Errorf(fmt.Sprintf("Can't remove last entry ( %s ) for %s", store.B64(id), name), err)
	}
	return fmt.Errorf("Issue in setting config %s, rolling back changes %s",
		errResp.Error(), rolledbackIDs)
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

func setChange(target string, version string, devicetype string, targetUpdates change.TypedValueMap,
	targetRemoves []string, description string) (change.ID, *store.ConfigName, bool, error) {
	changeID, configName, err := manager.GetManager().SetNetworkConfig(
		target, version, devicetype, targetUpdates, targetRemoves, description)
	if err != nil {
		if strings.Contains(err.Error(), manager.SetConfigAlreadyApplied) {
			log.Warning(manager.SetConfigAlreadyApplied, "Change ", store.B64(changeID), " to ", configName)
			return nil, nil, true, nil
		}
		log.Error("Error in setting config: ", changeID, " for target ", configName, err)
		return nil, nil, false, err
	}
	return changeID, configName, false, nil
}

func validateChange(target string, version string, deviceType string, targetUpdates change.TypedValueMap, targetRemoves []string) error {
	if len(targetUpdates) == 0 && len(targetRemoves) == 0 {
		return fmt.Errorf("no updates found in change on %s - invalid", target)
	}

	err := manager.GetManager().ValidateNetworkConfig(target, version, deviceType, targetUpdates, targetRemoves)
	if err != nil {
		log.Errorf("Error in validating config, updates %s, removes %s for target %s, err: %s", targetUpdates,
			targetRemoves, target, err)
		return err
	}
	return nil
}
