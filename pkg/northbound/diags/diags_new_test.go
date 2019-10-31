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

package diags

import (
	"context"
	"fmt"
	"github.com/golang/mock/gomock"
	"github.com/onosproject/onos-config/pkg/certs"
	"github.com/onosproject/onos-config/pkg/manager"
	"github.com/onosproject/onos-config/pkg/northbound"
	devicestore "github.com/onosproject/onos-config/pkg/store/device"
	"github.com/onosproject/onos-config/pkg/store/stream"
	mockstore "github.com/onosproject/onos-config/pkg/test/mocks/store"
	changetypes "github.com/onosproject/onos-config/pkg/types/change"
	devicechangetypes "github.com/onosproject/onos-config/pkg/types/change/device"
	networkchangetypes "github.com/onosproject/onos-config/pkg/types/change/network"
	"github.com/onosproject/onos-config/pkg/types/device"
	"google.golang.org/grpc"
	"gotest.tools/assert"
	"io"
	log "k8s.io/klog"
	"strings"
	"sync"
	"testing"
	"time"
)

var (
	// Address is a test server address as "127.0.0.1:port" string
	Address string

	// Opts is a set of gRPC connection options
	Opts []grpc.DialOption
)

// SetUpServer sets up a test manager and a gRPC end-point
// to which it registers the given service.
func setUpServer(port int16, service Service, waitGroup *sync.WaitGroup, ctrl *gomock.Controller) *manager.Manager {
	mgrTest, err := manager.LoadManager(
		mockstore.NewMockLeadershipStore(ctrl),
		mockstore.NewMockMastershipStore(ctrl),
		mockstore.NewMockDeviceChangesStore(ctrl),
		devicestore.NewMockCache(ctrl),
		mockstore.NewMockNetworkChangesStore(ctrl),
		mockstore.NewMockNetworkSnapshotStore(ctrl),
		mockstore.NewMockDeviceSnapshotStore(ctrl))
	if err != nil {
		log.Error("Unable to load manager")
	}

	config := northbound.NewServerConfig("", "", "")
	config.Port = port
	s := northbound.NewServer(config)
	s.AddService(service)

	empty := ""
	Address = fmt.Sprintf(":%d", port)
	Opts, err = certs.HandleCertArgs(&empty, &empty)
	if err != nil {
		log.Error("Error loading cert ", err)
	}
	go func() {
		err := s.Serve(func(started string) {
			waitGroup.Done()
			fmt.Printf("Started %v", started)
		})
		if err != nil {
			log.Error("Unable to serve ", err)
		}
	}()

	return mgrTest
}

func Test_ListNetworkChanges(t *testing.T) {
	const numevents = 40
	var waitGroup sync.WaitGroup
	waitGroup.Add(1)
	// Trying to get away from the TestMain PITA
	mgrTest := setUpServer(10124, Service{}, &waitGroup, gomock.NewController(t))
	waitGroup.Wait()

	networkChanges := generateNetworkChangeData(numevents)
	conn := northbound.Connect(Address, Opts...)
	client := NewChangeServiceClient(conn)
	defer conn.Close()

	mockNwChStore, ok := mgrTest.NetworkChangesStore.(*mockstore.MockNetworkChangesStore)
	assert.Assert(t, ok, "casting mock store")
	mockNwChStore.EXPECT().List(gomock.Any()).DoAndReturn(func(ch chan<- *networkchangetypes.NetworkChange) (stream.Context, error) {
		// Send our network changes as a streamed response to store List()
		go func() {
			for _, nwch := range networkChanges {
				ch <- nwch
			}
			close(ch)
		}()
		return stream.NewContext(func() {

		}), nil
	})
	req := ListNetworkChangeRequest{
		Subscribe: false,
		ChangeID:  "change-*",
	}
	stream, err := client.ListNetworkChanges(context.Background(), &req)
	assert.NilError(t, err)

	go func() {
		count := 0
		for {
			in, err := stream.Recv() // Should block until we receive responses
			if err == io.EOF || in == nil {
				break
			}
			assert.NilError(t, err, "unable to receive message")

			//t.Logf("Recv network change %v", netwChange.ID)
			assert.Assert(t, strings.HasPrefix(string(in.Change.ID), "change-"))
			count++
		}
		assert.Equal(t, count, numevents)
	}()

	time.Sleep(time.Millisecond * numevents * 2)
}

func Test_ListDeviceChanges(t *testing.T) {
	const numevents = 40
	var waitGroup sync.WaitGroup
	waitGroup.Add(1)
	// Trying to get away from the TestMain PITA
	mgrTest := setUpServer(10125, Service{}, &waitGroup, gomock.NewController(t))
	waitGroup.Wait()

	deviceChanges := generateDeviceChangeData(numevents)
	conn := northbound.Connect(Address, Opts...)
	client := NewChangeServiceClient(conn)
	defer conn.Close()

	mockDevChStore, ok := mgrTest.DeviceChangesStore.(*mockstore.MockDeviceChangesStore)
	assert.Assert(t, ok, "casting mock store")
	mockDevChStore.EXPECT().List(gomock.Any(), gomock.Any()).
		DoAndReturn(func(id device.VersionedID, ch chan<- *devicechangetypes.DeviceChange) (stream.Context, error) {

			// Send our network changes as a streamed response to store List()
			go func() {
				for _, devch := range deviceChanges {
					ch <- devch
				}
				close(ch)
			}()

			return stream.NewContext(func() {

			}), nil
		})
	req := ListDeviceChangeRequest{
		Subscribe:     false,
		DeviceID:      "device-1",
		DeviceVersion: "1.0.0",
	}
	stream, err := client.ListDeviceChanges(context.Background(), &req)
	assert.NilError(t, err)

	go func() {
		count := 0
		for {
			in, err := stream.Recv() // Should block until we receive responses
			if err == io.EOF || in == nil {
				break
			}
			assert.NilError(t, err, "unable to receive message")

			//t.Logf("Recv device change %v", in.Change.ID)
			assert.Assert(t, strings.HasPrefix(string(in.Change.ID), "device-"))
			count++
		}
		assert.Equal(t, count, numevents)
	}()

	time.Sleep(time.Millisecond * numevents * 2)
}

func generateNetworkChangeData(count int) []*networkchangetypes.NetworkChange {
	networkChanges := make([]*networkchangetypes.NetworkChange, count)
	now := time.Now()

	for cfgIdx := range networkChanges {
		networkID := fmt.Sprintf("change-%d", cfgIdx)

		networkChanges[cfgIdx] = &networkchangetypes.NetworkChange{
			ID:       networkchangetypes.ID(networkID),
			Index:    networkchangetypes.Index(cfgIdx),
			Revision: 0,
			Status: changetypes.Status{
				Phase:   changetypes.Phase(cfgIdx % 2),
				State:   changetypes.State(cfgIdx % 4),
				Reason:  changetypes.Reason(cfgIdx % 2),
				Message: "Test",
			},
			Created: now,
			Updated: now,
			Changes: []*devicechangetypes.Change{
				{
					DeviceID:      "device-1",
					DeviceVersion: "1.0.0",
					Values: []*devicechangetypes.ChangeValue{
						{
							Path:    "/aa/bb/cc",
							Value:   devicechangetypes.NewTypedValueString("Test1"),
							Removed: false,
						},
						{
							Path:    "/aa/bb/dd",
							Value:   devicechangetypes.NewTypedValueString("Test2"),
							Removed: false,
						},
					},
				},
				{
					DeviceID:      "device-2",
					DeviceVersion: "1.0.0",
					Values: []*devicechangetypes.ChangeValue{
						{
							Path:    "/aa/bb/cc",
							Value:   devicechangetypes.NewTypedValueString("Test3"),
							Removed: false,
						},
						{
							Path:    "/aa/bb/dd",
							Value:   devicechangetypes.NewTypedValueString("Test4"),
							Removed: false,
						},
					},
				},
			},
			Refs: []*networkchangetypes.DeviceChangeRef{
				{DeviceChangeID: "device-1:1"},
				{DeviceChangeID: "device-2:1"},
			},
			Deleted: false,
		}
	}

	return networkChanges
}

func generateDeviceChangeData(count int) []*devicechangetypes.DeviceChange {
	networkChanges := make([]*devicechangetypes.DeviceChange, count)
	now := time.Now()

	for cfgIdx := range networkChanges {
		networkID := fmt.Sprintf("device-%d", cfgIdx)

		networkChanges[cfgIdx] = &devicechangetypes.DeviceChange{
			ID:       devicechangetypes.ID(networkID),
			Index:    devicechangetypes.Index(cfgIdx),
			Revision: 0,
			NetworkChange: devicechangetypes.NetworkChangeRef{
				ID:    "network-1",
				Index: 0,
			},
			Change: &devicechangetypes.Change{
				DeviceID:      "devicesim-1",
				DeviceVersion: "1.0.0",
				Values: []*devicechangetypes.ChangeValue{
					{
						Path:    "/aa/bb/cc",
						Value:   devicechangetypes.NewTypedValueString("test1"),
						Removed: false,
					},
					{
						Path:    "/aa/bb/dd",
						Value:   devicechangetypes.NewTypedValueString("test2"),
						Removed: false,
					},
				},
			},
			Status: changetypes.Status{
				Phase:   changetypes.Phase(cfgIdx % 2),
				State:   changetypes.State(cfgIdx % 4),
				Reason:  changetypes.Reason(cfgIdx % 2),
				Message: "Test",
			},
			Created: now,
			Updated: now,
		}
	}

	return networkChanges
}
