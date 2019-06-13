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
	"github.com/onosproject/onos-config/pkg/northbound"
	"github.com/onosproject/onos-config/pkg/northbound/proto"
	"google.golang.org/grpc"
	"gotest.tools/assert"
	"io"
	"os"
	"testing"
)

// TestMain initializes the test suite context.
func TestMain(m *testing.M) {
	northbound.SetUpServer(10124, Service{})
	os.Exit(m.Run())
}

func getAdminClient() (*grpc.ClientConn, proto.AdminServiceClient) {
	conn := northbound.Connect(northbound.Address, northbound.Opts...)
	return conn, proto.NewAdminServiceClient(conn)
}

func getDeviceClient() (*grpc.ClientConn, proto.DeviceInventoryServiceClient) {
	conn := northbound.Connect(northbound.Address, northbound.Opts...)
	return conn, proto.NewDeviceInventoryServiceClient(conn)
}

func Test_GetNetworkChanges(t *testing.T) {
	conn, client := getAdminClient()
	defer conn.Close()
	stream, err := client.GetNetworkChanges(context.Background(), &proto.NetworkChangesRequest{})
	assert.Assert(t, err == nil && stream != nil, "unable to issue request")
	var name = ""
	for {
		in, err := stream.Recv()
		if err == io.EOF || in == nil {
			break
		}
		assert.Assert(t, err == nil, "unable to receive message")
		name = in.Name
	}
	err = stream.CloseSend()
	assert.Assert(t, err == nil, "unable to close stream")
	assert.Assert(t, len(name) > 0, "wrong name received")
}

func Test_RollbackNetworkChange_BadName(t *testing.T) {
	conn, client := getAdminClient()
	defer conn.Close()
	_, err := client.RollbackNetworkChange(context.Background(), &proto.RollbackRequest{Name: "BAD CHANGE"})
	assert.Assert(t, err != nil, "should not rollback")
}

func Test_RollbackNetworkChange_NoChange(t *testing.T) {
	conn, client := getAdminClient()
	defer conn.Close()
	_, err := client.RollbackNetworkChange(context.Background(), &proto.RollbackRequest{Name: ""})
	assert.Assert(t, err != nil, "should not rollback")
}

func Test_AddDevice(t *testing.T) {
	conn, client := getDeviceClient()
	defer conn.Close()
	resp, err := client.GetDeviceSummary(context.Background(), &proto.DeviceSummaryRequest{})
	oldCount := resp.Count
	_, err = client.AddOrUpdateDevice(context.Background(),
		&proto.DeviceInfo{Id: "device", Address: "address", Version: "0.9"})
	assert.Assert(t, err == nil, "should add device")
	resp, err = client.GetDeviceSummary(context.Background(), &proto.DeviceSummaryRequest{})
	assert.Assert(t, oldCount+1 == resp.Count, "should add new device")
}

func Test_GetDevices(t *testing.T) {
	conn, client := getDeviceClient()
	defer conn.Close()
	stream, err := client.GetDevices(context.Background(), &proto.GetDevicesRequest{})
	assert.Assert(t, err == nil && stream != nil, "unable to issue request")
	var id = ""
	for {
		in, err := stream.Recv()
		if err == io.EOF || in == nil {
			break
		}
		assert.Assert(t, err == nil, "unable to receive message")
		id = in.Id
	}
	err = stream.CloseSend()
	assert.Assert(t, err == nil, "unable to close stream")
	assert.Assert(t, len(id) > 0, "wrong id received")
}

func Test_RemoveDevice(t *testing.T) {
	conn, client := getDeviceClient()
	defer conn.Close()
	_, err := client.AddOrUpdateDevice(context.Background(),
		&proto.DeviceInfo{Id: "device", Address: "address", Version: "0.9"})
	assert.Assert(t, err == nil, "should add device")
	resp, err := client.GetDeviceSummary(context.Background(), &proto.DeviceSummaryRequest{})
	oldCount := resp.Count
	_, err = client.RemoveDevice(context.Background(), &proto.DeviceInfo{Id: "device"})
	assert.Assert(t, err == nil, "should remove device")
	resp, err = client.GetDeviceSummary(context.Background(), &proto.DeviceSummaryRequest{})
	assert.Assert(t, oldCount-1 == resp.Count, "should remove existing device")
}
