// Copyright 2022-present Open Networking Foundation.
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
	"github.com/google/uuid"
	configapi "github.com/onosproject/onos-api/go/onos/config/v2"
	"github.com/onosproject/onos-config/pkg/utils"
	valueutils "github.com/onosproject/onos-config/pkg/utils/values/v2"
	"github.com/onosproject/onos-lib-go/pkg/uri"
	"github.com/openconfig/gnmi/proto/gnmi"
)

type mapTargetUpdates map[configapi.TargetID]configapi.TypedValueMap
type mapTargetRemoves map[configapi.TargetID][]string

type targetInfo struct {
	targetID      configapi.TargetID
	targetVersion configapi.TargetVersion
	targetType    configapi.TargetType
}

func newUpdateResult(pathStr string, target string, op gnmi.UpdateResult_Operation) (*gnmi.UpdateResult, error) {
	path, err := utils.ParseGNMIElements(utils.SplitPath(pathStr))
	if err != nil {
		return nil, err
	}
	path.Target = target
	updateResult := &gnmi.UpdateResult{
		Path: path,
		Op:   op,
	}
	return updateResult, nil

}

func computeChanges(targetInfo targetInfo, targetUpdates mapTargetUpdates,
	targetRemoves mapTargetRemoves) ([]configapi.Change, error) {

	targetChanges := make([]configapi.Change, 0)
	for target, updates := range targetUpdates {
		newChange, err := computeChange(targetInfo, updates, targetRemoves[target])
		if err != nil {
			return nil, err
		}
		targetChanges = append(targetChanges, newChange)
		delete(targetRemoves, target)
	}

	// Some targets might only have removes
	for _, removes := range targetRemoves {
		newChange, err := computeChange(targetInfo, make(configapi.TypedValueMap), removes)
		if err != nil {
			return nil, err
		}
		targetChanges = append(targetChanges, newChange)
	}
	return targetChanges, nil
}

// computeChange computes a given target change the given updates and deletes, according to the path
// on the configuration for the specified target
func computeChange(targetInfo targetInfo, updates configapi.TypedValueMap, deletes []string) (configapi.Change, error) {
	var newChanges = make([]configapi.ChangeValue, 0)
	//updates
	for path, value := range updates {
		updateValue, err := valueutils.NewChangeValue(path, *value, false)
		if err != nil {
			return configapi.Change{}, err
		}
		newChanges = append(newChanges, *updateValue)
	}
	//deletes
	for _, path := range deletes {
		deleteValue, _ := valueutils.NewChangeValue(path, *configapi.NewTypedValueEmpty(), true)
		newChanges = append(newChanges, *deleteValue)
	}

	changeElement := configapi.Change{
		TargetID:      targetInfo.targetID,
		TargetVersion: targetInfo.targetVersion,
		TargetType:    targetInfo.targetType,
		Values:        newChanges,
	}

	return changeElement, nil
}

func newTransaction(targetInfo targetInfo, extensions Extensions, targetUpdates mapTargetUpdates,
	targetRemoves mapTargetRemoves, username string) (*configapi.Transaction, error) {

	changes, err := computeChanges(targetInfo, targetUpdates, targetRemoves)
	if err != nil {
		return nil, err
	}
	var transactionID configapi.TransactionID
	if extensions.transactionID != "" {
		transactionID = extensions.transactionID

	} else {
		transactionID = configapi.TransactionID(uri.NewURI(
			uri.WithScheme("uuid"),
			uri.WithOpaque(uuid.New().String())).String())
	}

	transaction := &configapi.Transaction{
		ID: transactionID,
		Transaction: &configapi.Transaction_Change{
			Change: &configapi.TransactionChange{
				Changes: changes,
			},
		},
		Username: username,
		Status: configapi.TransactionStatus{
			State: configapi.TransactionState_TRANSACTION_PENDING,
		},
	}

	return transaction, nil

}
