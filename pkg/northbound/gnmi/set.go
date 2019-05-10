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
	"github.com/onosproject/onos-config/pkg/manager"
	"github.com/onosproject/onos-config/pkg/store"
	"github.com/onosproject/onos-config/pkg/store/change"
	"github.com/onosproject/onos-config/pkg/utils"
	"github.com/openconfig/gnmi/proto/gnmi"
	"log"
	"time"
)

// Set implements gNMI Set
func (s *Server) Set(ctx context.Context, req *gnmi.SetRequest) (*gnmi.SetResponse, error) {
	//updates := make(map[string]string)
	targetUpdates := make(map[string]map[string]string)
	targetRemoves := make(map[string][]string)

	//TODO consolidate these lines into methods
	//Update
	for _, u := range req.Update {
		target := u.Path.Target
		updates, ok := targetUpdates[target]
		if !ok {
			updates = make(map[string]string)
		}
		path := utils.StrPath(u.Path)
		updates[path] = utils.StrVal(u.Val)
		targetUpdates[target] = updates
	}

	//Replace
	for _, u := range req.Replace {
		target := u.Path.Target
		updates, ok := targetUpdates[target]
		if !ok {
			updates = make(map[string]string)
		}
		path := utils.StrPath(u.Path)
		updates[path] = utils.StrVal(u.Val)
		targetUpdates[target] = updates
	}

	//Delete
	for _, u := range req.Delete {
		target := u.Target
		deletes, ok := targetRemoves[target]
		if !ok {
			deletes = make([]string, 0)
		}
		path := utils.StrPath(u)
		deletes = append(deletes, path)
		targetRemoves[target] = deletes
	}

	networkChangeIds := make(map[string]change.ID)
	configName := "Running"
	for target, updates := range targetUpdates {
		changeID, err := manager.GetManager().SetNetworkConfig(target, configName, updates, targetRemoves[target])
		if err != nil {
			log.Println("Error in setting config:", changeID, "for target", target)
			//TODO save error and stop process and initiate rollback
		}
		networkChangeIds[target] = changeID
	}

	//TODO move to manager.CreateNewNetworkConfig
	networkConfig := store.NetworkConfiguration{
		Name:                 "Current",
		Created:              time.Now(),
		User:                 "User1",
		ConfigurationChanges: networkChangeIds,
	}
	manager.GetManager().NetworkStore.Store = append(manager.GetManager().NetworkStore.Store, networkConfig)

	//TODO Create the SetResponse
	return &gnmi.SetResponse{}, nil
}
