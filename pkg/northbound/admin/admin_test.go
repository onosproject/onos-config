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
	"github.com/onosproject/onos-config/api/admin"
	device2 "github.com/onosproject/onos-config/api/types/change/device"
	"github.com/onosproject/onos-config/api/types/device"
	devicesnapshot "github.com/onosproject/onos-config/api/types/snapshot/device"
	"github.com/onosproject/onos-config/pkg/manager"
	mockstore "github.com/onosproject/onos-config/pkg/test/mocks/store"
	"github.com/onosproject/onos-config/pkg/test/mocks/store/cache"
	"google.golang.org/grpc"
	"google.golang.org/grpc/test/bufconn"
	"gotest.tools/assert"
	"net"
	"os"
	"testing"
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

	ctrl := gomock.NewController(t)
	mgrTest, err := manager.NewManager(
		mockstore.NewMockLeadershipStore(ctrl),
		mockstore.NewMockMastershipStore(ctrl),
		mockstore.NewMockDeviceChangesStore(ctrl),
		mockstore.NewMockDeviceStateStore(ctrl),
		mockstore.NewMockDeviceStore(ctrl),
		cache.NewMockCache(ctrl),
		mockstore.NewMockNetworkChangesStore(ctrl),
		mockstore.NewMockNetworkSnapshotStore(ctrl),
		mockstore.NewMockDeviceSnapshotStore(ctrl),
		true)
	if err != nil {
		log.Error("Unable to load manager")
	}

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

func Test_GetSnapshot(t *testing.T) {
	mgrTest, conn, client, server := setUpServer(t)
	defer server.Stop()
	defer conn.Close()

	const dev1ID = "presentdevice"
	const dev1Ver = "2.0.0"
	snapshotID := devicesnapshot.ID(fmt.Sprintf("Snapshot:%s:%s", dev1ID, dev1Ver))

	mockDevSnapshotStore, ok := mgrTest.DeviceSnapshotStore.(*mockstore.MockDeviceSnapshotStore)
	assert.Assert(t, ok, "casting mock store")

	mockDevSnapshotStore.EXPECT().Load(gomock.Any()).DoAndReturn(
		func(devId device.VersionedID) (*devicesnapshot.Snapshot, error) {
			if devId.GetID() == dev1ID && devId.GetVersion() == dev1Ver {
				return &devicesnapshot.Snapshot{
					ID:            snapshotID,
					DeviceID:      dev1ID,
					DeviceVersion: dev1Ver,
					DeviceType:    "TestDevice",
					SnapshotID:    "test-snapshot",
					ChangeIndex:   0,
					Values: []*device2.PathValue{
						{Path: "/a/b/c", Value: device2.NewTypedValueInt64(12345)},
						{Path: "/a/b/d", Value: device2.NewTypedValueInt64(12346)},
					},
				}, nil
			}
			return nil, nil
		}).AnyTimes()

	snapshot, err := client.GetSnapshot(context.Background(), &admin.GetSnapshotRequest{
		DeviceID:      device.ID("testdevice"),
		DeviceVersion: device.Version(""),
	})
	assert.ErrorContains(t, err, "An ID and Version must be given")
	assert.Assert(t, snapshot == nil, "Expecting snapshot to be nil")

	// Try again with a version
	snapshot, err = client.GetSnapshot(context.Background(), &admin.GetSnapshotRequest{
		DeviceID:      device.ID("absentdevice"),
		DeviceVersion: device.Version("1.0.0"),
	})
	assert.ErrorContains(t, err, "No snapshot found")
	assert.Assert(t, snapshot == nil, "Expecting snapshot to be nil")

	// Try again with a positive response
	snapshot, err = client.GetSnapshot(context.Background(), &admin.GetSnapshotRequest{
		DeviceID:      dev1ID,
		DeviceVersion: dev1Ver,
	})
	assert.NilError(t, err, "Not expecting error on GetSnapshot")

	assert.Equal(t, string(snapshotID), string(snapshot.ID))
	assert.Equal(t, dev1ID, string(snapshot.DeviceID))
	assert.Equal(t, dev1Ver, string(snapshot.DeviceVersion))
	assert.Equal(t, "TestDevice", string(snapshot.DeviceType))
	assert.Equal(t, "test-snapshot", string(snapshot.SnapshotID))
	assert.Equal(t, 2, len(snapshot.GetValues()))
}
