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

package admin

import (
	"context"
	"errors"
	"fmt"
	"github.com/golang/mock/gomock"
	"github.com/onosproject/onos-api/go/onos/config/admin"
	device2 "github.com/onosproject/onos-api/go/onos/config/change/device"
	"github.com/onosproject/onos-api/go/onos/config/device"
	devicesnapshot "github.com/onosproject/onos-api/go/onos/config/snapshot/device"
	"github.com/onosproject/onos-config/pkg/manager"
	"github.com/onosproject/onos-config/pkg/modelregistry"
	"github.com/onosproject/onos-config/pkg/store/stream"
	mockstore "github.com/onosproject/onos-config/pkg/test/mocks/store"
	"github.com/onosproject/onos-config/pkg/test/mocks/store/cache"
	"google.golang.org/grpc"
	"google.golang.org/grpc/test/bufconn"
	"gotest.tools/assert"
	"io"
	"net"
	"os"
	"testing"
	"time"
)

// TestMain initializes the test suite context.
func TestMain(m *testing.M) {
	os.Exit(m.Run())
}

func setUpServer(t *testing.T) (*manager.Manager, *grpc.ClientConn, admin.ConfigAdminServiceClient, *grpc.Server) {
	lis := bufconn.Listen(1024 * 1024)
	s := grpc.NewServer()

	admin.RegisterConfigAdminServiceServer(s, &Server{})

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

	client := admin.CreateConfigAdminServiceClient(conn)

	registry, err := modelregistry.NewModelRegistry(modelregistry.Config{
		ModPath:      "test/data/" + t.Name() + "/mod",
		RegistryPath: "test/data/" + t.Name() + "/registry",
		PluginPath:   "test/data/" + t.Name() + "/plugins",
		ModTarget:    "github.com/onosproject/onos-config",
	})
	assert.NilError(t, err)

	ctrl := gomock.NewController(t)
	mgrTest := manager.NewManager(
		mockstore.NewMockLeadershipStore(ctrl),
		mockstore.NewMockMastershipStore(ctrl),
		mockstore.NewMockDeviceChangesStore(ctrl),
		mockstore.NewMockDeviceStateStore(ctrl),
		mockstore.NewMockDeviceStore(ctrl),
		cache.NewMockCache(ctrl),
		mockstore.NewMockNetworkChangesStore(ctrl),
		mockstore.NewMockNetworkSnapshotStore(ctrl),
		mockstore.NewMockDeviceSnapshotStore(ctrl),
		true,
		nil,
		registry)

	return mgrTest, conn, client, s
}

func Test_RollbackNetworkChange_BadName(t *testing.T) {
	mgrTest, conn, client, server := setUpServer(t)
	defer server.Stop()
	defer conn.Close()

	mockNwChStore, ok := mgrTest.NetworkChangesStore.(*mockstore.MockNetworkChangesStore)
	assert.Assert(t, ok, "casting mock store")

	mockNwChStore.EXPECT().Get(gomock.Any()).Return(nil, errors.New("Rollback aborted. Network change BAD CHANGE not found"))
	_, err := client.RollbackNetworkChange(context.Background(), &admin.RollbackRequest{Name: "BAD CHANGE"})
	assert.ErrorContains(t, err, "Rollback aborted. Network change BAD CHANGE not found")
}

func Test_RollbackNetworkChange_NoChange(t *testing.T) {
	mgrTest, conn, client, server := setUpServer(t)
	defer server.Stop()
	defer conn.Close()

	mockNwChStore, ok := mgrTest.NetworkChangesStore.(*mockstore.MockNetworkChangesStore)
	assert.Assert(t, ok, "casting mock store")

	mockNwChStore.EXPECT().Get(gomock.Any()).Return(nil, errors.New("change is not specified"))
	_, err := client.RollbackNetworkChange(context.Background(), &admin.RollbackRequest{Name: ""})
	assert.ErrorContains(t, err, "is not")
}

func Test_ListSnapshots(t *testing.T) {
	const numSnapshots = 2
	mgrTest, conn, client, server := setUpServer(t)
	defer server.Stop()
	defer conn.Close()

	snapshots := generateSnapshotData(numSnapshots)

	mockDevSnapshotStore, ok := mgrTest.DeviceSnapshotStore.(*mockstore.MockDeviceSnapshotStore)
	assert.Assert(t, ok, "casting mock store")

	mockDevSnapshotStore.EXPECT().LoadAll(gomock.Any()).DoAndReturn(
		func(ch chan<- *devicesnapshot.Snapshot) (stream.Context, error) {
			go func() {
				for _, snapshot := range snapshots {
					ch <- snapshot
				}
				close(ch)
			}()
			return stream.NewContext(func() {
			}), nil
		})

	stream, err := client.ListSnapshots(context.Background(), &admin.ListSnapshotsRequest{
		ID:        "device-*",
		Subscribe: false,
	})
	assert.NilError(t, err, "Not expecting error on ListSnapshots")

	go func() {
		count := 0
		for {
			snapshot, err := stream.Recv()
			if err == io.EOF || snapshot == nil {
				break
			}
			assert.NilError(t, err, "unable to receive message")

			assert.Equal(t, "test-snapshot", string(snapshot.SnapshotID))
			assert.Equal(t, "1.0.0", string(snapshot.DeviceVersion))
			assert.Equal(t, "TestDevice", string(snapshot.DeviceType))
			assert.Equal(t, 2, len(snapshot.GetValues()))
			switch string(snapshot.ID) {
			case "device-0:1.0.0":
				assert.Equal(t, "device-0", string(snapshot.DeviceID))
			case "device-1:1.0.0":
				assert.Equal(t, "device-1", string(snapshot.DeviceID))
			default:
				assert.Assert(t, false, "Unhandled case %s", string(snapshot.ID))
			}
			count++
		}
		assert.Equal(t, numSnapshots, count)
	}()

	time.Sleep(time.Millisecond * numSnapshots * 2)
}

func generateSnapshotData(count int) []*devicesnapshot.Snapshot {
	snapshots := make([]*devicesnapshot.Snapshot, count)

	for shIdx := range snapshots {
		deviceID := fmt.Sprintf("device-%d", shIdx)
		snapshots[shIdx] = &devicesnapshot.Snapshot{
			ID:            devicesnapshot.ID(deviceID + ":1.0.0"),
			DeviceID:      device.ID(deviceID),
			DeviceVersion: device.Version("1.0.0"),
			DeviceType:    "TestDevice",
			SnapshotID:    "test-snapshot",
			ChangeIndex:   device2.Index(shIdx),
			Values: []*device2.PathValue{
				{Path: "/a/b/c", Value: device2.NewTypedValueInt(shIdx, 32)},
				{Path: "/a/b/d", Value: device2.NewTypedValueInt(10*shIdx, 32)},
			},
		}
	}
	return snapshots
}
