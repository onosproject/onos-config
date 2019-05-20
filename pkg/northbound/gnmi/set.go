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
	"log"
	"strings"
	"time"

	"github.com/onosproject/onos-config/pkg/manager"
	"github.com/onosproject/onos-config/pkg/store"
	"github.com/onosproject/onos-config/pkg/store/change"
	"github.com/onosproject/onos-config/pkg/utils"
	"github.com/openconfig/gnmi/proto/gnmi"
)

// ConfigNameSuffix is appended to the Configuration name when it is created
const ConfigNameSuffix = "Running"

type mapTargetUpdates map[string]map[string]string
type mapTargetRemoves map[string][]string
type mapNetworkChanges map[store.ConfigName]change.ID

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
	targetRemoves mapTargetRemoves) ([]*gnmi.UpdateResult, mapNetworkChanges) {

	networkChanges := make(map[store.ConfigName]change.ID)
	updateResults := make([]*gnmi.UpdateResult, 0)
	for target, updates := range targetUpdates {
		changeID, err := manager.GetManager().SetNetworkConfig(
			target, store.ConfigName(target+ConfigNameSuffix), updates, targetRemoves[target])
		var op = gnmi.UpdateResult_UPDATE

		if err != nil {
			if strings.Contains(err.Error(), manager.SetConfigAlreadyApplied) {
				log.Println(manager.SetConfigAlreadyApplied, "Change", store.B64(changeID), "to", target)
				continue
			}

			//FIXME this at the moment fails at a device level. we can specify a per path failure
			// if the store could return us that info
			log.Println("Error in setting config:", changeID, "for target", target)
			op = gnmi.UpdateResult_INVALID
			//TODO initiate rollback
		}

		for k := range updates {
			path, errInPath := utils.ParseGNMIElements(strings.Split(k, "/")[1:])
			if errInPath != nil {
				log.Println("ERROR: Unable to parse path", k)
				continue
			}
			path.Target = target

			updateResult := &gnmi.UpdateResult{
				Path: path,
				Op:   op,
			}
			updateResults = append(updateResults, updateResult)
		}

		if op == gnmi.UpdateResult_UPDATE {
			op = gnmi.UpdateResult_DELETE
		}
		for _, r := range targetRemoves[target] {
			path, errInPath := utils.ParseGNMIElements(strings.Split(r, "/")[1:])
			if errInPath != nil {
				log.Println("ERROR: Unable to parse path", r)
				continue
			}
			path.Target = target

			updateResult := &gnmi.UpdateResult{
				Path: path,
				Op:   op,
			}
			updateResults = append(updateResults, updateResult)
		}

		networkChanges[store.ConfigName(target+ConfigNameSuffix)] = changeID
	}

	return updateResults, networkChanges
}

// Set implements gNMI Set
func (s *Server) Set(ctx context.Context, req *gnmi.SetRequest) (*gnmi.SetResponse, error) {

	targetUpdates := make(mapTargetUpdates)
	targetRemoves := make(mapTargetRemoves)

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

	updateResults, networkChanges := s.buildUpdateResults(targetUpdates, targetRemoves)

	if len(updateResults) == 0 {
		log.Println("All target changes were duplicated - Set rejected")
		return nil, fmt.Errorf("set change rejected as it is a " +
			"duplicate of the last change for all targets")
	}

	networkConfig, err := store.CreateNetworkStore("User1", networkChanges)
	if err != nil {
		return nil, err
	}

	manager.GetManager().NetworkStore.Store =
		append(manager.GetManager().NetworkStore.Store, *networkConfig)

	setResponse := &gnmi.SetResponse{
		Response:  updateResults,
		Timestamp: time.Now().Unix(),
	}
	return setResponse, nil
}
