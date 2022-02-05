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

func computeChanges(targets map[configapi.TargetID]*targetInfo) (map[configapi.TargetID]*configapi.PathValues, error) {
	allChanges := make(map[configapi.TargetID]*configapi.PathValues)
	for targetID, target := range targets {
		change, err := computeChange(target)
		if err != nil {
			return nil, err
		}
		allChanges[targetID] = change
	}
	return allChanges, nil
}

// computeChange computes a given target change the given its updates and deletes, according to the path
// on the configuration for the specified target
func computeChange(target *targetInfo) (*configapi.PathValues, error) {
	//updates
	newChanges := make(map[string]*configapi.PathValue)
	for path, value := range target.updates {
		updateValue, err := valueutils.NewChangeValue(path, *value, false)
		if err != nil {
			return &configapi.PathValues{}, err
		}
		newChanges[path] = updateValue
	}
	//deletes
	for _, path := range target.removes {
		deleteValue, _ := valueutils.NewChangeValue(path, *configapi.NewTypedValueEmpty(), true)
		newChanges[path] = deleteValue
	}

	changeElement := &configapi.PathValues{
		Values: newChanges,
	}

	return changeElement, nil
}

func newTransaction(targets map[configapi.TargetID]*targetInfo, mode configapi.TransactionMode, username string) (*configapi.Transaction, error) {
	values, err := computeChanges(targets)
	if err != nil {
		return nil, err
	}
	transactionID := configapi.TransactionID(uri.NewURI(
		uri.WithScheme("uuid"),
		uri.WithOpaque(uuid.New().String())).String())
	transaction := &configapi.Transaction{
		ID: transactionID,
		Details: &configapi.Transaction_Change{
			Change: &configapi.ChangeTransaction{
				Values: values,
			},
		},
		Username: username,
		Atomic:   mode.Atomic,
	}

	return transaction, nil

}
