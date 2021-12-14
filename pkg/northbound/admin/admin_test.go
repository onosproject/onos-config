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
	"github.com/atomix/atomix-go-client/pkg/atomix/test"
	"github.com/atomix/atomix-go-client/pkg/atomix/test/rsm"
	"github.com/golang/mock/gomock"
	"github.com/onosproject/onos-api/go/onos/config/admin"
	networkchanges "github.com/onosproject/onos-config/pkg/store/change/network"
	mockstore "github.com/onosproject/onos-config/pkg/test/mocks/store"
	"github.com/stretchr/testify/assert"
	"google.golang.org/grpc"
	"google.golang.org/grpc/test/bufconn"
	"net"
	"testing"
)

func setUpServer(t *testing.T) (*grpc.ClientConn, admin.ConfigAdminServiceClient, *grpc.Server, networkchanges.Store, *mockstore.MockNetworkSnapshotStore, *mockstore.MockDeviceSnapshotStore) {
	lis := bufconn.Listen(1024 * 1024)
	s := grpc.NewServer()

	ctrl := gomock.NewController(t)

	test := test.NewTest(
		rsm.NewProtocol(),
		test.WithReplicas(1),
		test.WithPartitions(1))
	assert.NoError(t, test.Start())
	defer test.Stop()

	atomixClient, err := test.NewClient("node")
	assert.NoError(t, err)

	networkChangesStore, err := networkchanges.NewAtomixStore(atomixClient)
	assert.NoError(t, err)

	//networkChangesStore := mockstore.NewMockNetworkChangesStore(ctrl)
	networkSnapshotStore := mockstore.NewMockNetworkSnapshotStore(ctrl)
	deviceSnapshotStore := mockstore.NewMockDeviceSnapshotStore(ctrl)

	admin.RegisterConfigAdminServiceServer(s, &Server{
		networkChangesStore:  networkChangesStore,
		networkSnapshotStore: networkSnapshotStore,
		deviceSnapshotStore:  deviceSnapshotStore,
	})

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

	assert.NoError(t, err)

	return conn, client, s, networkChangesStore, networkSnapshotStore, deviceSnapshotStore
}

func Test_RollbackNetworkChange_BadName(t *testing.T) {
	conn, client, server, _, _, _ := setUpServer(t)
	defer server.Stop()
	defer conn.Close()

	//networkChangesStore.EXPECT().Get(gomock.Any()).Return(nil, errors.NewNotFound("Rollback aborted. Network change BAD CHANGE not found"))
	_, err := client.RollbackNetworkChange(context.Background(), &admin.RollbackRequest{Name: "BAD CHANGE"})
	assert.Contains(t, err.Error(), "Rollback aborted. Network change BAD CHANGE not found")
}

func Test_RollbackNetworkChange_NoChange(t *testing.T) {
	conn, client, server, _, _, _ := setUpServer(t)
	defer server.Stop()
	defer conn.Close()

	//networkChangesStore.EXPECT().Get(gomock.Any()).Return(nil, errors.NewNotFound("change is not specified")).AnyTimes()
	_, err := client.RollbackNetworkChange(context.Background(), &admin.RollbackRequest{Name: ""})
	assert.Contains(t, err.Error(), "is empty")
}

/*
func Test_ListSnapshots(t *testing.T) {
	const numSnapshots = 2
	conn, client, server, _, _, deviceSnapshotStore := setUpServer(t)
	defer server.Stop()
	defer conn.Close()

	snapshots := generateSnapshotData(numSnapshots)

	deviceSnapshotStore.EXPECT().LoadAll(gomock.Any()).DoAndReturn(
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

	listSnapshots, err := client.ListSnapshots(context.Background(), &admin.ListSnapshotsRequest{
		ID:        "device-*",
		Subscribe: false,
	})
	assert.NoError(t, err, "Not expecting error on ListSnapshots")

	go func() {
		count := 0
		for {
			snapshot, err := listSnapshots.Recv()
			if err == io.EOF || snapshot == nil {
				break
			}
			assert.NoError(t, err, "unable to receive message")

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
				assert.Failf(t, "Unhandled case %s", string(snapshot.ID))
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
			DeviceVersion: "1.0.0",
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
*/
