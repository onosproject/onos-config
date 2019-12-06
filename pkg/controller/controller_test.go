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
	"github.com/onosproject/onos-config/api/types"
	"github.com/stretchr/testify/assert"
	"strconv"
	"sync"
	"sync/atomic"
	"testing"
)

func TestController(t *testing.T) {
	ctrl := gomock.NewController(t)

	activatorValue := &atomic.Value{}
	activator := NewMockActivator(ctrl)
	activator.EXPECT().
		Start(gomock.Any()).
		DoAndReturn(func(ch chan<- bool) error {
			activatorValue.Store(ch)
			return nil
		})
	activator.EXPECT().Stop()

	filter := NewMockFilter(ctrl)
	filter.EXPECT().
		Accept(gomock.Any()).
		DoAndReturn(func(id types.ID) bool {
			i, _ := strconv.Atoi(string(id))
			return i%2 == 0
		}).
		AnyTimes()

	wg := &sync.WaitGroup{}
	wg.Add(1)

	watcherValue := &atomic.Value{}
	watcher := NewMockWatcher(ctrl)
	watcher.EXPECT().
		Start(gomock.Any()).
		DoAndReturn(func(ch chan<- types.ID) error {
			watcherValue.Store(ch)
			wg.Done()
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

	controller := NewController("Test").
		Activate(activator).
		Filter(filter).
		Watch(watcher).
		Partition(partitioner).
		Reconcile(reconciler)
	defer controller.Stop()

	err := controller.Start()
	assert.NoError(t, err)

	activatorCh := activatorValue.Load().(chan<- bool)
	activatorCh <- true

	reconciler.EXPECT().
		Reconcile(gomock.Eq(types.ID("2"))).
		Return(Result{}, nil)
	reconciler.EXPECT().
		Reconcile(gomock.Eq(types.ID("2"))).
		Return(Result{}, errors.New("some error"))
	reconciler.EXPECT().
		Reconcile(gomock.Eq(types.ID("2"))).
		Return(Result{}, errors.New("some error"))
	reconciler.EXPECT().
		Reconcile(gomock.Eq(types.ID("2"))).
		Return(Result{}, errors.New("some error"))
	reconciler.EXPECT().
		Reconcile(gomock.Eq(types.ID("2"))).
		Return(Result{}, nil)

	reconciler.EXPECT().
		Reconcile(gomock.Eq(types.ID("4"))).
		Return(Result{}, errors.New("some error"))
	reconciler.EXPECT().
		Reconcile(gomock.Eq(types.ID("4"))).
		Return(Result{}, nil)

	wg.Wait()
	watcherCh := watcherValue.Load().(chan<- types.ID)
	watcherCh <- types.ID("1")
	watcherCh <- types.ID("2")
	watcherCh <- types.ID("3")
	watcherCh <- types.ID("4")
}
