// Copyright 2020-present Open Networking Foundation.
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
	"context"
	"github.com/onosproject/onos-config/api/admin"
	"github.com/onosproject/onos-config/api/types/device"
	testutils "github.com/onosproject/onos-config/test/utils"
	"github.com/onosproject/onos-test/pkg/onit/env"
	"github.com/stretchr/testify/assert"
	"io"
	"testing"
	"time"
)

// TestCompactChanges tests the CompactChanges and Snapshot RPCs on the Admin gRPC interface. This
// 1) sets up 2 simulators
// 2a) makes 2 gnmi change on sim 1
// 2b) makes 3 gnmi changes on sim 2
// 2c) makes another gnmi change on both sim1 and sim2
// 3) waits for the network changes to be complete (and for the simulators to start)
// 4) calls compact_changes to create a snapshot
// 5) retrieves the snapshot using both Get and List
// 6) makes another NW change
// 7) make sure the aggregate config includes snapshot + subsequent change
// See also TestSnapshotErrors
func (s *TestSuite) TestCompactChanges(t *testing.T) {
	const version = "1.0.0"
	const deviceType = "Devicesim"
	const motdPath = "/system/config/motd-banner"
	const motdValue1 = "Sim1 Motd"
	const motdValue2 = "Sim2 Motd"
	const tzParis = "Europe/Paris"
	const tzMilan = "Europe/Milan"
	const loginBnrPath = "/system/config/login-banner"
	const loginBnr1 = "Sim1 Login Banner"
	const loginBnr2 = "Sim2 Login Banner"
	const domainNamePath = "/system/config/domain-name"
	const domainNameSim1 = "sim1.domain.name"
	const domainNameSim2 = "sim2.domain.name"
	const wait = 60 * time.Second

	// Create 2 simulators
	simulator1 := env.NewSimulator().SetDeviceVersion(version).SetDeviceType(deviceType).AddOrDie()
	simulator2 := env.NewSimulator().SetDeviceVersion(version).SetDeviceType(deviceType).AddOrDie()

	// Make a GNMI client to use for requests
	gnmiClient := getGNMIClientOrFail(t)

	// Set a value using gNMI client
	sim1Path1 := getDevicePathWithValue(simulator1.Name(), tzPath, tzValue, StringVal)
	sim1nwChangeID1 := setGNMIValueOrFail(t, gnmiClient, sim1Path1, noPaths, noExtensions)
	complete := testutils.WaitForNetworkChangeComplete(t, sim1nwChangeID1, wait)
	assert.True(t, complete)

	sim1Path2 := getDevicePathWithValue(simulator1.Name(), motdPath, motdValue1, StringVal)
	sim1nwChangeID2 := setGNMIValueOrFail(t, gnmiClient, sim1Path2, noPaths, noExtensions)
	complete = testutils.WaitForNetworkChangeComplete(t, sim1nwChangeID2, wait)
	assert.True(t, complete)

	// Make a triple path change to Sim2
	sim2Path1 := getDevicePathWithValue(simulator2.Name(), tzPath, tzParis, StringVal)
	sim2Path2 := getDevicePathWithValue(simulator2.Name(), motdPath, motdValue2, StringVal)
	sim2Path3 := getDevicePathWithValue(simulator2.Name(), domainNamePath, domainNameSim2, StringVal)

	sim2nwChangeID1 := setGNMIValueOrFail(t, gnmiClient, []DevicePath{sim2Path1[0], sim2Path2[0], sim2Path3[0]}, noPaths, noExtensions)
	complete = testutils.WaitForNetworkChangeComplete(t, sim2nwChangeID1, wait)
	assert.True(t, complete)

	// Finally make a change to both devices
	sim1Path3 := getDevicePathWithValue(simulator1.Name(), loginBnrPath, loginBnr1, StringVal)
	sim2Path4 := getDevicePathWithValue(simulator2.Name(), loginBnrPath, loginBnr2, StringVal)
	bothSimNwChangeID := setGNMIValueOrFail(t, gnmiClient, []DevicePath{sim1Path3[0], sim2Path4[0]}, noPaths, noExtensions)

	// Wait for the change to transition to complete
	complete = testutils.WaitForNetworkChangeComplete(t, bothSimNwChangeID, wait)
	assert.True(t, complete)

	t.Logf("Testing CompactChanges - nw changes %s, %s on %s AND %s on %s AND %s on both",
		sim1nwChangeID1, sim1nwChangeID2, simulator1.Name(), sim2nwChangeID1, simulator2.Name(), bothSimNwChangeID)

	adminClient, err := env.Config().NewAdminServiceClient()
	assert.NoError(t, err)

	// First time around we should get an error
	snapshot, err := adminClient.GetSnapshot(context.Background(), &admin.GetSnapshotRequest{
		DeviceID:      device.ID(simulator1.Name()),
		DeviceVersion: device.Version(version),
	})
	assert.Errorf(t, err, "No simulator1 found", "expecting snapshot to not be found")
	assert.Nil(t, snapshot)

	// Now try compacting changes
	compacted, err := adminClient.CompactChanges(context.Background(), &admin.CompactChangesRequest{})
	assert.NoError(t, err)
	assert.NotNil(t, compacted)

	// Second time around we should get a response - this is using GetSnapshot
	snapshot, err = adminClient.GetSnapshot(context.Background(), &admin.GetSnapshotRequest{
		DeviceID:      device.ID(simulator1.Name()),
		DeviceVersion: device.Version(version),
	})
	assert.NoError(t, err)
	assert.NotNil(t, snapshot)
	if snapshot != nil {
		assert.Equal(t, simulator1.Name(), string(snapshot.DeviceID))
		assert.Equal(t, version, string(snapshot.DeviceVersion))
		assert.Equal(t, deviceType, string(snapshot.DeviceType))
		assert.Equal(t, 3, len(snapshot.GetValues()))
		for _, v := range snapshot.GetValues() {
			switch v.Path {
			case tzPath:
				assert.Equal(t, tzValue, v.Value.ValueToString())
			case motdPath:
				assert.Equal(t, motdValue1, v.Value.ValueToString())
			case loginBnrPath:
				assert.Equal(t, loginBnr1, v.Value.ValueToString())
			default:
				assert.Fail(t, "Unexpected path %s Value %s", v.Path, v.Value.ValueToString())
			}
		}
	}

	// Use ListSnapshots to list all snapshots in the system - there should be two
	stream, err := adminClient.ListSnapshots(context.Background(), &admin.ListSnapshotsRequest{})
	assert.NoError(t, err, "Unexpected error listing snapshots")

	for {
		breakout := false
		response, err := stream.Recv()
		if err == io.EOF {
			t.Log("Snapshot command ended cleanly")
			breakout = true
		} else if err != nil {
			assert.Fail(t, "ListSnapshots failed", err)
		} else {
			assert.NotNil(t, response, "Expecting response to be non-nil")
			t.Logf("Snapshot %s retrieved", response.ID)
			assert.Equal(t, version, string(response.DeviceVersion))
			assert.Equal(t, deviceType, string(response.DeviceType))
			switch string(response.DeviceID) {
			case simulator1.Name():
				assert.Equal(t, simulator1.Name()+":"+version, string(response.GetID()))
				assert.Equal(t, 3, len(response.GetValues()))
			case simulator2.Name():
				assert.Equal(t, simulator2.Name()+":"+version, string(response.GetID()))
				assert.Equal(t, 4, len(response.GetValues()))
			default:
				assert.Fail(t, "Unexpected Device ID in response %s", response.DeviceID)
			}
		}
		if breakout {
			break
		}
	}

	// Set a value using gNMI client
	sim1Path4 := getDevicePathWithValue(simulator1.Name(), tzPath, tzMilan, StringVal)
	sim1Path5 := getDevicePathWithValue(simulator1.Name(), domainNamePath, domainNameSim1, StringVal)
	afterSnapshotChangeID := setGNMIValueOrFail(t, gnmiClient, []DevicePath{sim1Path4[0], sim1Path5[0]}, noPaths, noExtensions)

	// Wait for the change to transition to complete
	complete = testutils.WaitForNetworkChangeComplete(t, afterSnapshotChangeID, wait)
	assert.True(t, complete)

	// Now check every value for both sim1 and sim2
	expectedValues, _, err := getGNMIValue(testutils.MakeContext(), gnmiClient,
		[]DevicePath{
			sim1Path4[0], sim1Path2[0], sim1Path3[0], sim1Path5[0],
			sim2Path1[0], sim2Path2[0], sim2Path3[0], sim2Path4[0],
		})
	assert.NoError(t, err)
	for _, expectedValue := range expectedValues {
		switch expectedValue.deviceName + "," + expectedValue.path {
		case simulator1.Name() + "," + tzPath:
			assert.Equal(t, tzMilan, expectedValue.pathDataValue, "Unexpected value for TZ on sim1")
		case simulator2.Name() + "," + tzPath:
			// Commented out because of issue https://github.com/onosproject/onos-config/issues/1031
			//assert.Equal(t, tzParis, expectedValue.pathDataValue, "Unexpected value for TZ on sim2")
		case simulator1.Name() + "," + motdPath:
			// Commented out because of issue https://github.com/onosproject/onos-config/issues/1031
			//assert.Equal(t, motdValue1, expectedValue.pathDataValue, "Unexpected value for Motd on sim1")
		case simulator2.Name() + "," + motdPath:
			// Commented out because of issue https://github.com/onosproject/onos-config/issues/1031
			//assert.Equal(t, motdValue2, expectedValue.pathDataValue, "Unexpected value for Motd on sim2")
		case simulator1.Name() + "," + loginBnrPath:
			// Commented out because of issue https://github.com/onosproject/onos-config/issues/1031
			//assert.Equal(t, loginBnr1, expectedValue.pathDataValue, "Unexpected value for LB on sim1")
		case simulator2.Name() + "," + loginBnrPath:
			// Commented out because of issue https://github.com/onosproject/onos-config/issues/1031
			//assert.Equal(t, loginBnr2, expectedValue.pathDataValue, "Unexpected value for LB on sim2")
		case simulator1.Name() + "," + domainNamePath:
			assert.Equal(t, domainNameSim1, expectedValue.pathDataValue, "Unexpected value for DN on sim1")
		case simulator2.Name() + "," + domainNamePath:
			// Commented out because of issue https://github.com/onosproject/onos-config/issues/1031
			//assert.Equal(t, domainNameSim1, expectedValue.pathDataValue, "Unexpected value for DN on sim2")
		default:
			assert.Failf(t, "Unhandled value %s,%s", expectedValue.deviceName, expectedValue.path)
		}
	}
}
