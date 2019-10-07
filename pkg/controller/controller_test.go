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

package controller

import (
	"errors"
	"github.com/golang/mock/gomock"
	"github.com/onosproject/onos-config/pkg/types"
	"github.com/stretchr/testify/assert"
	"strconv"
	"testing"
)

func TestController(t *testing.T) {
	ctrl := gomock.NewController(t)

	var activatorCh chan<- bool
	activator := NewMockActivator(ctrl)
	activator.EXPECT().
		Start(gomock.Any()).
		DoAndReturn(func(ch chan<- bool) error {
			activatorCh = ch
			return nil
		})

	filter := NewMockFilter(ctrl)
	filter.EXPECT().
		Accept(gomock.Any()).
		DoAndReturn(func(id types.ID) bool {
			i, _ := strconv.Atoi(string(id))
			return i%2 == 0
		}).
		AnyTimes()

	var watcherCh chan<- types.ID
	watcher := NewMockWatcher(ctrl)
	watcher.EXPECT().
		Start(gomock.Any()).
		DoAndReturn(func(ch chan<- types.ID) error {
			watcherCh = ch
			return nil
		})

	partitions := 3
	partitioner := NewMockWorkPartitioner(ctrl)
	partitioner.EXPECT().
		Partition(gomock.Any()).
		DoAndReturn(func(id types.ID) (PartitionKey, error) {
			i, _ := strconv.Atoi(string(id))
			partition := i % partitions
			return PartitionKey(strconv.Itoa(partition)), nil
		}).
		AnyTimes()

	reconciler := NewMockReconciler(ctrl)

	controller := NewController().
		Activate(activator).
		Filter(filter).
		Watch(watcher).
		Partition(partitioner).
		Reconcile(reconciler)

	err := controller.Start()
	assert.NoError(t, err)

	activatorCh <- true

	reconciler.EXPECT().
		Reconcile(gomock.Eq(types.ID("2"))).
		Return(false, nil)
	reconciler.EXPECT().
		Reconcile(gomock.Eq(types.ID("2"))).
		Return(false, errors.New("some error"))
	reconciler.EXPECT().
		Reconcile(gomock.Eq(types.ID("2"))).
		Return(false, errors.New("some error"))
	reconciler.EXPECT().
		Reconcile(gomock.Eq(types.ID("2"))).
		Return(false, errors.New("some error"))
	reconciler.EXPECT().
		Reconcile(gomock.Eq(types.ID("2"))).
		Return(true, nil)

	reconciler.EXPECT().
		Reconcile(gomock.Eq(types.ID("4"))).
		Return(false, errors.New("some error"))
	reconciler.EXPECT().
		Reconcile(gomock.Eq(types.ID("4"))).
		Return(true, nil)

	watcherCh <- types.ID("1")
	watcherCh <- types.ID("2")
	watcherCh <- types.ID("3")
	watcherCh <- types.ID("4")
}
