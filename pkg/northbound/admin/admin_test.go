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
	"fmt"
	"github.com/atomix/atomix-go-client/pkg/atomix/test"
	"github.com/atomix/atomix-go-client/pkg/atomix/test/rsm"
	"github.com/onosproject/onos-api/go/onos/config/admin"
	devicechange "github.com/onosproject/onos-api/go/onos/config/change/device"
	"github.com/onosproject/onos-api/go/onos/config/device"
	devicesnapshot "github.com/onosproject/onos-api/go/onos/config/snapshot/device"
	networkchanges "github.com/onosproject/onos-config/pkg/store/change/network"
	devicesnapshotstore "github.com/onosproject/onos-config/pkg/store/snapshot/device"
	networksnapshotstore "github.com/onosproject/onos-config/pkg/store/snapshot/network"
	"github.com/stretchr/testify/assert"
	"google.golang.org/grpc"
	"google.golang.org/grpc/test/bufconn"
	"io"
	"net"
	"testing"
)

func setUpServer(ctx context.Context, t *testing.T) (admin.ConfigAdminServiceClient, devicesnapshotstore.Store) {
	lis := bufconn.Listen(1024 * 1024)
	s := grpc.NewServer()

	atomixTest := test.NewTest(
		rsm.NewProtocol(),
		test.WithReplicas(1),
		test.WithPartitions(1))
	assert.NoError(t, atomixTest.Start())

	atomixClient, err := atomixTest.NewClient("node")
	assert.NoError(t, err)

	networkChangesStore, err := networkchanges.NewAtomixStore(atomixClient)
	assert.NoError(t, err)

	networkSnapshotStore, err := networksnapshotstore.NewAtomixStore(atomixClient)
	assert.NoError(t, err)

	deviceSnapshotStore, err := devicesnapshotstore.NewAtomixStore(atomixClient)
	assert.NoError(t, err)

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

	go func() {
		<-ctx.Done()
		_ = atomixTest.Stop()
		s.Stop()
		_ = conn.Close()
	}()

	return client, deviceSnapshotStore
}

func Test_RollbackNetworkChange_BadName(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	client, _ := setUpServer(ctx, t)

	_, err := client.RollbackNetworkChange(context.Background(), &admin.RollbackRequest{Name: "BAD CHANGE"})
	assert.Contains(t, err.Error(), "no entry found at key BAD CHANGE")
}

func Test_RollbackNetworkChange_NoChange(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	client, _ := setUpServer(ctx, t)

	_, err := client.RollbackNetworkChange(context.Background(), &admin.RollbackRequest{Name: ""})
	assert.Contains(t, err.Error(), "is empty")
}

func Test_ListSnapshots(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	const numSnapshots = 2
	client, deviceSnapshotStore := setUpServer(ctx, t)

	snapshots := generateSnapshotData(numSnapshots)

	for _, snapshot := range snapshots {
		assert.NoError(t, deviceSnapshotStore.Store(snapshot))
	}

	listSnapshots, err := client.ListSnapshots(context.Background(), &admin.ListSnapshotsRequest{
		ID:        "device-*",
		Subscribe: false,
	})
	assert.NoError(t, err, "Not expecting error on ListSnapshots")

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
			ChangeIndex:   devicechange.Index(shIdx),
			Values: []*devicechange.PathValue{
				{Path: "/a/b/c", Value: devicechange.NewTypedValueInt(shIdx, 32)},
				{Path: "/a/b/d", Value: devicechange.NewTypedValueInt(10*shIdx, 32)},
			},
		}
	}
	return snapshots
}
