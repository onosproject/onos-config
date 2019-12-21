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

package manager

import (
	"fmt"
	changetypes "github.com/onosproject/onos-config/api/types/change"
	networkchange "github.com/onosproject/onos-config/api/types/change/network"
	networkchangestore "github.com/onosproject/onos-config/pkg/store/change/network"
	"github.com/onosproject/onos-config/pkg/store/stream"
)

// RollbackTargetConfig rollbacks the last change for a given configuration on the target, by setting phase to
// rollback and state to pending.
func (m *Manager) RollbackTargetConfig(networkChangeID networkchange.ID) error {

	changeRollback, errGet := m.NetworkChangesStore.Get(networkChangeID)
	if errGet != nil {
		log.Errorf("Error on get change %s for rollback: %s", networkChangeID, errGet)
		return errGet
	}

	//Making sure that the change is the last one
	next, errGetNext := m.NetworkChangesStore.GetNext(changeRollback.Index)
	if errGetNext != nil {
		log.Errorf("Error on get next change during rollback: %s", errGetNext)
		return errGet
	}
	// if the error is nil and the change is nil the requested one is the last one thus we proceed.
	// if there is a next change but the phase is different from ROLLBACK and the status is different from COMPLETE we
	// fail the operation because there is a need to rollback the previous one.
	if next != nil && (next.Status.Phase != changetypes.Phase_ROLLBACK ||
		(next.Status.Phase == changetypes.Phase_ROLLBACK && next.Status.State != changetypes.State_COMPLETE)) {
		errLast := fmt.Errorf("change %s is not the last active on the stack of changes", networkChangeID)
		return errLast
	}

	changeRollback.Status.Incarnation++
	changeRollback.Status.Phase = changetypes.Phase_ROLLBACK
	changeRollback.Status.State = changetypes.State_PENDING
	changeRollback.Status.Reason = changetypes.Reason_NONE
	changeRollback.Status.Message = "Administratively requested rollback"

	errUpdate := m.NetworkChangesStore.Update(changeRollback)
	if errUpdate != nil {
		log.Errorf("Error on setting change %s rollback: %s", networkChangeID, errUpdate)
		return errUpdate
	}
	return listenForChangeNotification(m, networkChangeID)
}

func listenForChangeNotification(mgr *Manager, changeID networkchange.ID) error {
	networkChan := make(chan stream.Event)
	ctx, errWatch := mgr.NetworkChangesStore.Watch(networkChan, networkchangestore.WithChangeID(changeID))
	if errWatch != nil {
		return fmt.Errorf("can't complete rollback operation on target due to %s", errWatch)
	}
	defer ctx.Close()
	for changeEvent := range networkChan {
		change := changeEvent.Object.(*networkchange.NetworkChange)
		log.Infof("Received notification for change ID %s, phase %s, state %s", change.ID,
			change.Status.Phase, change.Status.State)
		if change.Status.Phase == changetypes.Phase_ROLLBACK {
			switch changeStatus := change.Status.State; changeStatus {
			case changetypes.State_COMPLETE:
				log.Infof("Rollback succeeded for change %s ", changeID)
				return nil
			case changetypes.State_FAILED:
				log.Infof("Received Change Status %s", changeStatus)
				return fmt.Errorf("issue in setting config reson %s, error %s, rolling back change %s",
					change.Status.Reason, change.Status.Message, changeID)
			default:
				continue
			}
		}
	}
	return nil
}
