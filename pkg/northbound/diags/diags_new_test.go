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
	"github.com/onosproject/onos-config/api/diags"
	changetypes "github.com/onosproject/onos-config/api/types/change"
	devicechange "github.com/onosproject/onos-config/api/types/change/device"
	networkchange "github.com/onosproject/onos-config/api/types/change/network"
	devicetypes "github.com/onosproject/onos-config/api/types/device"
	"github.com/onosproject/onos-config/pkg/manager"
	"github.com/onosproject/onos-config/pkg/store/device/cache"
	"github.com/onosproject/onos-config/pkg/store/stream"
	mockstore "github.com/onosproject/onos-config/pkg/test/mocks/store"
	mockcache "github.com/onosproject/onos-config/pkg/test/mocks/store/cache"
	topodevice "github.com/onosproject/onos-topo/api/device"
	"google.golang.org/grpc"
	"google.golang.org/grpc/test/bufconn"
	"gotest.tools/assert"
	"io"
	"net"
	"strings"
	"testing"
	"time"
)

// SetUpServer sets up a test manager and a gRPC end-point
// to which it registers the given service.
func setUpServer(t *testing.T) (*manager.Manager, *grpc.ClientConn, diags.ChangeServiceClient, *grpc.Server) {
	lis := bufconn.Listen(1024 * 1024)
	s := grpc.NewServer()

	diags.RegisterChangeServiceServer(s, &Server{})

	go func() {
		if err := s.Serve(lis); err != nil {
			t.Error("Server exited with error")
		}
	}()

	dialer := func(ctx context.Context, address string) (net.Conn, error) {
		return lis.Dial()
	}

	conn, err := grpc.DialContext(context.Background(), "bufnet", grpc.WithContextDialer(dialer), grpc.WithInsecure())
	if err != nil {
		t.Error("Failed to dial bufnet")
	}

	client := diags.CreateChangeServiceClient(conn)

	ctrl := gomock.NewController(t)
	mgrTest, err := manager.NewManager(
		mockstore.NewMockLeadershipStore(ctrl),
		mockstore.NewMockMastershipStore(ctrl),
		mockstore.NewMockDeviceChangesStore(ctrl),
		mockstore.NewMockDeviceStateStore(ctrl),
		mockstore.NewMockDeviceStore(ctrl),
		mockcache.NewMockCache(ctrl),
		mockstore.NewMockNetworkChangesStore(ctrl),
		mockstore.NewMockNetworkSnapshotStore(ctrl),
		mockstore.NewMockDeviceSnapshotStore(ctrl),
		true)
	if err != nil {
		log.Error("Unable to load manager")
	}

	mgrTest.DeviceStore = mockstore.NewMockDeviceStore(ctrl)

	return mgrTest, conn, client, s
}

func Test_ListNetworkChanges(t *testing.T) {
	const numevents = 40
	mgrTest, conn, client, server := setUpServer(t)
	defer server.Stop()
	defer conn.Close()

	networkChanges := generateNetworkChangeData(numevents)

	mockNwChStore, ok := mgrTest.NetworkChangesStore.(*mockstore.MockNetworkChangesStore)
	assert.Assert(t, ok, "casting mock store")
	mockNwChStore.EXPECT().List(gomock.Any()).DoAndReturn(func(ch chan<- *networkchange.NetworkChange) (stream.Context, error) {
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
	req := diags.ListNetworkChangeRequest{
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
	mgrTest, conn, client, server := setUpServer(t)
	defer server.Stop()
	defer conn.Close()

	deviceChanges := generateDeviceChangeData(numevents)

	mockDevChStore, ok := mgrTest.DeviceChangesStore.(*mockstore.MockDeviceChangesStore)
	assert.Assert(t, ok, "casting mock device changes store")
	mockDevChStore.EXPECT().List(gomock.Any(), gomock.Any()).
		DoAndReturn(func(id devicetypes.VersionedID, ch chan<- *devicechange.DeviceChange) (stream.Context, error) {

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

	mockDeviceStore, ok := mgrTest.DeviceStore.(*mockstore.MockDeviceStore)
	assert.Assert(t, ok, "casting mock device store")
	mockDeviceStore.EXPECT().Get(topodevice.ID("device-1")).Return(
		&topodevice.Device{
			ID:      "device-1",
			Address: "localhost:10126",
			Type:    "Devicesim",
			Role:    "leaf",
		}, nil,
	)

	mockDeviceCache, ok := mgrTest.DeviceCache.(*mockcache.MockCache)
	assert.Assert(t, ok, "casting mock cache")
	mockDeviceCache.EXPECT().GetDevicesByID(devicetypes.ID("device-1")).Return([]*cache.Info{
		{
			DeviceID: "device-1",
			Type:     "Devicesim",
			Version:  "1.0.0",
		},
	})

	req := diags.ListDeviceChangeRequest{
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

func Test_ListDeviceChangesNoVersionManyPresent(t *testing.T) {
	mgrTest, conn, client, server := setUpServer(t)
	defer server.Stop()
	defer conn.Close()

	mockDeviceStore, ok := mgrTest.DeviceStore.(*mockstore.MockDeviceStore)
	assert.Assert(t, ok, "casting mock device store")
	mockDeviceStore.EXPECT().Get(topodevice.ID("device-1")).Return(
		nil, nil,
	)

	mockDeviceCache, ok := mgrTest.DeviceCache.(*mockcache.MockCache)
	assert.Assert(t, ok, "casting mock cache")
	mockDeviceCache.EXPECT().GetDevicesByID(devicetypes.ID("device-1")).Return([]*cache.Info{
		{
			DeviceID: "device-1",
			Type:     "Devicesim",
			Version:  "1.0.0",
		},
		{
			DeviceID: "device-1",
			Type:     "Devicesim",
			Version:  "2.0.0",
		},
	})
	time.Sleep(time.Millisecond * 10)

	req := diags.ListDeviceChangeRequest{
		Subscribe: false,
		DeviceID:  "device-1",
	}
	stream, err := client.ListDeviceChanges(context.TODO(), &req)
	assert.NilError(t, err) // Doesn't get error here - this is just the client - error is in stream
	assert.Assert(t, stream != nil)

	go func() {
		for {
			_, err := stream.Recv()                                                                                                        // Should block until we receive responses
			assert.Error(t, err, "rpc error: code = Internal desc = target device-1 has 2 versions. Specify 1 version with extension 102") //Expecting an error here
		}
	}()
	time.Sleep(time.Millisecond * 10)

}

func generateNetworkChangeData(count int) []*networkchange.NetworkChange {
	networkChanges := make([]*networkchange.NetworkChange, count)
	now := time.Now()

	for cfgIdx := range networkChanges {
		networkID := fmt.Sprintf("change-%d", cfgIdx)

		networkChanges[cfgIdx] = &networkchange.NetworkChange{
			ID:       networkchange.ID(networkID),
			Index:    networkchange.Index(cfgIdx),
			Revision: 0,
			Status: changetypes.Status{
				Phase:   changetypes.Phase(cfgIdx % 2),
				State:   changetypes.State(cfgIdx % 4),
				Reason:  changetypes.Reason(cfgIdx % 2),
				Message: "Test",
			},
			Created: now,
			Updated: now,
			Changes: []*devicechange.Change{
				{
					DeviceID:      "device-1",
					DeviceVersion: "1.0.0",
					Values: []*devicechange.ChangeValue{
						{
							Path:    "/aa/bb/cc",
							Value:   devicechange.NewTypedValueString("Test1"),
							Removed: false,
						},
						{
							Path:    "/aa/bb/dd",
							Value:   devicechange.NewTypedValueString("Test2"),
							Removed: false,
						},
					},
				},
				{
					DeviceID:      "device-2",
					DeviceVersion: "1.0.0",
					Values: []*devicechange.ChangeValue{
						{
							Path:    "/aa/bb/cc",
							Value:   devicechange.NewTypedValueString("Test3"),
							Removed: false,
						},
						{
							Path:    "/aa/bb/dd",
							Value:   devicechange.NewTypedValueString("Test4"),
							Removed: false,
						},
					},
				},
			},
			Refs: []*networkchange.DeviceChangeRef{
				{DeviceChangeID: "device-1:1"},
				{DeviceChangeID: "device-2:1"},
			},
			Deleted: false,
		}
	}

	return networkChanges
}

func generateDeviceChangeData(count int) []*devicechange.DeviceChange {
	networkChanges := make([]*devicechange.DeviceChange, count)
	now := time.Now()

	for cfgIdx := range networkChanges {
		networkID := fmt.Sprintf("device-%d", cfgIdx)

		networkChanges[cfgIdx] = &devicechange.DeviceChange{
			ID:       devicechange.ID(networkID),
			Index:    devicechange.Index(cfgIdx),
			Revision: 0,
			NetworkChange: devicechange.NetworkChangeRef{
				ID:    "network-1",
				Index: 0,
			},
			Change: &devicechange.Change{
				DeviceID:      "devicesim-1",
				DeviceVersion: "1.0.0",
				Values: []*devicechange.ChangeValue{
					{
						Path:    "/aa/bb/cc",
						Value:   devicechange.NewTypedValueString("test1"),
						Removed: false,
					},
					{
						Path:    "/aa/bb/dd",
						Value:   devicechange.NewTypedValueString("test2"),
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
