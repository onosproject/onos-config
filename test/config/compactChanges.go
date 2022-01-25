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

package config

import (
	"context"
	"github.com/onosproject/onos-api/go/onos/config/admin"
	"github.com/onosproject/onos-config/pkg/device"
	"github.com/onosproject/onos-config/test/utils/gnmi"
	"github.com/onosproject/onos-config/test/utils/proto"
	gpb "github.com/openconfig/gnmi/proto/gnmi"
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
	// Currently not working
	t.Skip()

	const version = "1.0.0"
	const deviceType = "devicesim-1.0.x"
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
	simulator1 := gnmi.CreateSimulator(t)
	simulator2 := gnmi.CreateSimulator(t)
	defer gnmi.DeleteSimulator(t, simulator1)
	defer gnmi.DeleteSimulator(t, simulator2)

	gnmi.WaitForTargetAvailable(t, device.ID(simulator1.Name()), 2*time.Minute)
	gnmi.WaitForTargetAvailable(t, device.ID(simulator2.Name()), 2*time.Minute)

	// Make a GNMI client to use for requests
	gnmiClient := gnmi.GetGNMIClientOrFail(t)

	// Set a value using gNMI client
	sim1Path1 := gnmi.GetTargetPathWithValue(simulator1.Name(), tzPath, tzValue, proto.StringVal)
	sim1nwTransactionID1, sim1nwTransactionIndex1 := gnmi.SetGNMIValueOrFail(t, gnmiClient, sim1Path1, gnmi.NoPaths, gnmi.NoExtensions)
	complete := gnmi.WaitForTransactionComplete(t, sim1nwTransactionID1, sim1nwTransactionIndex1, wait)
	assert.True(t, complete)

	sim1Path2 := gnmi.GetTargetPathWithValue(simulator1.Name(), motdPath, motdValue1, proto.StringVal)
	sim1nwTransactionID2, sim1nwTransactionIndex2 := gnmi.SetGNMIValueOrFail(t, gnmiClient, sim1Path2, gnmi.NoPaths, gnmi.NoExtensions)
	complete = gnmi.WaitForTransactionComplete(t, sim1nwTransactionID2, sim1nwTransactionIndex2, wait)
	assert.True(t, complete)

	// Make a triple path change to Sim2
	sim2Path1 := gnmi.GetTargetPathWithValue(simulator2.Name(), tzPath, tzParis, proto.StringVal)
	sim2Path2 := gnmi.GetTargetPathWithValue(simulator2.Name(), motdPath, motdValue2, proto.StringVal)
	sim2Path3 := gnmi.GetTargetPathWithValue(simulator2.Name(), domainNamePath, domainNameSim2, proto.StringVal)

	sim2nwTransactionID2, sim2nwTransactionIndex2 := gnmi.SetGNMIValueOrFail(t, gnmiClient, []proto.TargetPath{sim2Path1[0], sim2Path2[0], sim2Path3[0]}, gnmi.NoPaths, gnmi.NoExtensions)
	complete = gnmi.WaitForTransactionComplete(t, sim2nwTransactionID2, sim2nwTransactionIndex2, wait)
	assert.True(t, complete)

	// Finally make a change to both devices
	sim1Path3 := gnmi.GetTargetPathWithValue(simulator1.Name(), loginBnrPath, loginBnr1, proto.StringVal)
	sim2Path4 := gnmi.GetTargetPathWithValue(simulator2.Name(), loginBnrPath, loginBnr2, proto.StringVal)
	bothSimNwTransactionID, bothSimNwTransactionIndex := gnmi.SetGNMIValueOrFail(t, gnmiClient, []proto.TargetPath{sim1Path3[0], sim2Path4[0]}, gnmi.NoPaths, gnmi.NoExtensions)

	// Wait for the change to transition to complete
	complete = gnmi.WaitForTransactionComplete(t, bothSimNwTransactionID, bothSimNwTransactionIndex, wait)
	assert.True(t, complete)

	t.Logf("Testing CompactChanges - nw changes %s, %s on %s AND %s on %s AND %s on both",
		sim1nwTransactionID1, sim1nwTransactionID2, simulator1.Name(), sim2nwTransactionID2, simulator2.Name(), bothSimNwTransactionID)

	adminClient, err := gnmi.NewAdminServiceClient()
	assert.NoError(t, err)

	// Now try compacting changes
	compacted, err := adminClient.CompactChanges(context.Background(), &admin.CompactChangesRequest{})
	assert.NoError(t, err)
	assert.NotNil(t, compacted)

	// Use ListSnapshots to list all snapshots in the system - there should be two
	stream, err := adminClient.ListSnapshots(context.Background(), &admin.ListSnapshotsRequest{
		Subscribe: false,
	})
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
				for _, v := range response.GetValues() {
					switch v.Path {
					case tzPath:
						assert.Equal(t, tzValue, v.Value.ValueToString(), "Unexpected value for", simulator1.Name(), v.Path)
					case motdPath:
						assert.Equal(t, motdValue1, v.Value.ValueToString(), "Unexpected value for", simulator1.Name(), v.Path)
					case loginBnrPath:
						assert.Equal(t, loginBnr1, v.Value.ValueToString(), "Unexpected value for", simulator1.Name(), v.Path)
					default:
						assert.Fail(t, "Unexpected path for", simulator1.Name(), v.Path)
					}
				}
			case simulator2.Name():
				assert.Equal(t, simulator2.Name()+":"+version, string(response.GetID()))
				assert.Equal(t, 4, len(response.GetValues()))
				for _, v := range response.GetValues() {
					switch v.Path {
					case tzPath:
						assert.Equal(t, tzParis, v.Value.ValueToString(), "Unexpected value for", simulator2.Name(), v.Path)
					case motdPath:
						assert.Equal(t, motdValue2, v.Value.ValueToString(), "Unexpected value for", simulator2.Name(), v.Path)
					case loginBnrPath:
						assert.Equal(t, loginBnr2, v.Value.ValueToString(), "Unexpected value for", simulator2.Name(), v.Path)
					case domainNamePath:
						assert.Equal(t, domainNameSim2, v.Value.ValueToString(), "Unexpected value for", simulator2.Name(), v.Path)
					default:
						assert.Fail(t, "Unexpected path for", simulator2.Name(), v.Path)
					}
				}
			default:
				assert.Fail(t, "Unexpected Device ID in response %s", response.DeviceID)
			}
		}
		if breakout {
			break
		}
	}

	// Set a value using gNMI client
	sim1Path4 := gnmi.GetTargetPathWithValue(simulator1.Name(), tzPath, tzMilan, proto.StringVal)
	sim1Path5 := gnmi.GetTargetPathWithValue(simulator1.Name(), domainNamePath, domainNameSim1, proto.StringVal)
	afterSnapshotTransactionID, afterSnapshotTransactionIndex := gnmi.SetGNMIValueOrFail(t, gnmiClient, []proto.TargetPath{sim1Path4[0], sim1Path5[0]}, gnmi.NoPaths, gnmi.NoExtensions)

	// Wait for the change to transition to complete
	complete = gnmi.WaitForTransactionComplete(t, afterSnapshotTransactionID, afterSnapshotTransactionIndex, wait)
	assert.True(t, complete)

	// Now check every value for both sim1 and sim2
	expectedValues, _, err := gnmi.GetGNMIValue(gnmi.MakeContext(), gnmiClient,
		[]proto.TargetPath{
			sim1Path4[0], sim1Path2[0], sim1Path3[0], sim1Path5[0],
			sim2Path1[0], sim2Path2[0], sim2Path3[0], sim2Path4[0],
		}, gpb.Encoding_PROTO)
	assert.NoError(t, err)
	for _, expectedValue := range expectedValues {
		switch expectedValue.TargetName + "," + expectedValue.Path {
		case simulator1.Name() + "," + tzPath:
			assert.Equal(t, tzMilan, expectedValue.PathDataValue, "Unexpected value for TZ on sim1")
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
			assert.Equal(t, domainNameSim1, expectedValue.PathDataValue, "Unexpected value for DN on sim1")
		case simulator2.Name() + "," + domainNamePath:
			// Commented out because of issue https://github.com/onosproject/onos-config/issues/1031
			//assert.Equal(t, domainNameSim1, expectedValue.pathDataValue, "Unexpected value for DN on sim2")
		default:
			assert.Failf(t, "Unhandled value %s,%s", expectedValue.TargetName, expectedValue.Path)
		}
	}
}
