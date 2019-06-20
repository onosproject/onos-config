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
	"github.com/onosproject/onos-config/pkg/manager"
	"github.com/onosproject/onos-config/pkg/store"
	"github.com/onosproject/onos-config/pkg/store/change"
	"github.com/onosproject/onos-config/pkg/utils"
	"github.com/openconfig/gnmi/proto/gnmi"
	"github.com/openconfig/gnmi/proto/gnmi_ext"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	log "k8s.io/klog"
	"strings"
	"time"
)

type mapTargetUpdates map[string]map[string]string
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
		targetUpdates[target] = s.doUpdateOrReplace(u, targetUpdates)
	}

	//Replace
	for _, u := range req.GetReplace() {
		target := u.Path.GetTarget()
		targetUpdates[target] = s.doUpdateOrReplace(u, targetUpdates)
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
			return nil, status.Error(codes.InvalidArgument, fmt.Errorf("Unexpected extension %d = '%s' in Set()",
				ext.GetRegisteredExt().GetId(), ext.GetRegisteredExt().GetMsg()).Error())
		}
	}

	updateResults, networkChanges, err :=
		s.buildUpdateResults(targetUpdates, targetRemoves, version, deviceType)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	if len(updateResults) == 0 {
		log.Warning("All target changes were duplicated - Set rejected")
		return nil, status.Error(codes.AlreadyExists, fmt.Errorf("set change rejected as it is a "+
			"duplicate of the last change for all targets").Error())
	}

	if netcfgchangename == "" {
		netcfgchangename = namesgenerator.GetRandomName(0)
	}

	// Look for use of this name already
	for _, nwCfg := range manager.GetManager().NetworkStore.Store {
		if nwCfg.Name == netcfgchangename {
			return nil, status.Error(codes.InvalidArgument, fmt.Errorf(
				"Name %s is already used for a Network Configuration", netcfgchangename).Error())
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

func (s *Server) doUpdateOrReplace(u *gnmi.Update, targetUpdates mapTargetUpdates) map[string]string {
	target := u.Path.GetTarget()
	updates, ok := targetUpdates[target]
	if !ok {
		updates = make(map[string]string)
	}
	path := utils.StrPath(u.Path)
	updates[path] = utils.StrVal(u.Val)
	return updates

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

func (s *Server) buildUpdateResults(targetUpdates mapTargetUpdates,
	targetRemoves mapTargetRemoves, version string, deviceType string) ([]*gnmi.UpdateResult, mapNetworkChanges, error) {

	networkChanges := make(mapNetworkChanges)
	updateResults := make([]*gnmi.UpdateResult, 0)

	for target, updates := range targetUpdates {
		// target is a device name with no version
		changeID, configName, cont, err := setChange(target, version, updates, targetRemoves[target])
		//if the error is not nil and we need to continue do so
		if err != nil && !cont {
			return nil, nil, err
		} else if err == nil && cont {
			continue
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

		networkChanges[store.ConfigName(configName)] = changeID
	}

	for target, removes := range targetRemoves {
		changeID, configName, cont, err := setChange(target, version, make(map[string]string), targetRemoves[target])
		//if the error is not nil and we need to continue do so
		if err != nil && !cont {
			return nil, nil, err
		} else if err == nil && cont {
			continue
		}
		for _, r := range removes {
			updateResult, err := buildUpdateResult(r, target, gnmi.UpdateResult_DELETE)
			if err != nil {
				continue
			}
			updateResults = append(updateResults, updateResult)
		}
		networkChanges[store.ConfigName(configName)] = changeID
	}

	return updateResults, networkChanges, nil
}

func buildUpdateResult(pathStr string, target string, op gnmi.UpdateResult_Operation) (*gnmi.UpdateResult, error) {
	path, errInPath := utils.ParseGNMIElements(strings.Split(pathStr, "/")[1:])
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

func setChange(target string, version string, targetUpdates map[string]string, targetRemoves []string) (change.ID, store.ConfigName, bool, error) {
	configName := store.ConfigName(target)
	// target is a device name with no version
	if version != "" {
		configName = store.ConfigName(strings.Join([]string{target, version}, "-"))
	}
	changeID, configName, err := manager.GetManager().SetNetworkConfig(
		store.ConfigName(configName), targetUpdates, targetRemoves)
	if err != nil {
		if strings.Contains(err.Error(), manager.SetConfigAlreadyApplied) {
			log.Warning(manager.SetConfigAlreadyApplied, "Change ", store.B64(changeID), " to ", configName)
			return nil, "", true, nil
		}

		//FIXME this at the moment fails at a device level. we can specify a per path failure
		// if the store could return us that info
		log.Error("Error in setting config: ", changeID, " for target ", configName, err)
		return nil, "", false, err
	}
	return changeID, configName, false, nil
}
