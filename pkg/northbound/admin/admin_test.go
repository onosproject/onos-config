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
	"io"
	"os"
	"sync"
	"testing"

	"github.com/onosproject/onos-config/pkg/northbound"
	"github.com/onosproject/onos-config/pkg/northbound/proto"
	"google.golang.org/grpc"
	"gotest.tools/assert"
)

// TestMain initializes the test suite context.
func TestMain(m *testing.M) {
	var waitGroup sync.WaitGroup
	waitGroup.Add(1)
	northbound.SetUpServer(10124, Service{}, &waitGroup)
	waitGroup.Wait()
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
	assert.NilError(t, err, "unable to issue request")
	var name string
	for {
		in, err := stream.Recv()
		if err == io.EOF || in == nil {
			break
		}
		assert.NilError(t, err, "unable to receive message")
		name = in.Name
	}
	err = stream.CloseSend()
	assert.NilError(t, err, "unable to close stream")
	assert.Assert(t, len(name) > 0, "no name received")
}

func Test_RollbackNetworkChange_BadName(t *testing.T) {
	conn, client := getAdminClient()
	defer conn.Close()
	_, err := client.RollbackNetworkChange(context.Background(), &proto.RollbackRequest{Name: "BAD CHANGE"})
	assert.ErrorContains(t, err, "Rollback aborted. Network change BAD CHANGE not found")
}

func Test_RollbackNetworkChange_NoChange(t *testing.T) {
	conn, client := getAdminClient()
	defer conn.Close()
	_, err := client.RollbackNetworkChange(context.Background(), &proto.RollbackRequest{Name: ""})
	assert.ErrorContains(t, err, "is not")
}
func Test_AddDevice(t *testing.T) {
	conn, client := getDeviceClient()
	defer conn.Close()
	resp, _ := client.GetDeviceSummary(context.Background(), &proto.DeviceSummaryRequest{})
	oldCount := resp.Count
	_, err := client.AddOrUpdateDevice(context.Background(),
		&proto.DeviceInfo{Id: "device", Address: "address", Version: "0.9"})
	assert.NilError(t, err, "should add device")
	resp, _ = client.GetDeviceSummary(context.Background(), &proto.DeviceSummaryRequest{})
	assert.Equal(t, oldCount+1, resp.Count, "should add new device")
}

func Test_GetDevices(t *testing.T) {
	conn, client := getDeviceClient()
	defer conn.Close()
	stream, err := client.GetDevices(context.Background(), &proto.GetDevicesRequest{})
	assert.NilError(t, err, "unable to issue request")
	var id string
	for {
		in, err := stream.Recv()
		if err == io.EOF || in == nil {
			break
		}
		assert.NilError(t, err, "unable to receive message")
		id = in.Id
	}
	err = stream.CloseSend()
	assert.NilError(t, err, "unable to close stream")
	assert.Assert(t, len(id) > 0, "no id received")
}

func Test_RemoveDevice(t *testing.T) {
	conn, client := getDeviceClient()
	defer conn.Close()
	_, err := client.AddOrUpdateDevice(context.Background(),
		&proto.DeviceInfo{Id: "device", Address: "address", Version: "0.9"})
	assert.NilError(t, err, "should add device")
	resp, _ := client.GetDeviceSummary(context.Background(), &proto.DeviceSummaryRequest{})
	oldCount := resp.Count
	_, err = client.RemoveDevice(context.Background(), &proto.DeviceInfo{Id: "device"})
	assert.NilError(t, err, "should remove device")
	resp, _ = client.GetDeviceSummary(context.Background(), &proto.DeviceSummaryRequest{})
	assert.Equal(t, oldCount-1, resp.Count, "should remove existing device")
}
