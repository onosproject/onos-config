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
	"github.com/onosproject/onos-config/pkg/northbound"
	"github.com/onosproject/onos-config/pkg/northbound/proto"
	"google.golang.org/grpc"
	"gotest.tools/assert"
	"io"
	"os"
	"sync"
	"testing"
)

// TestMain initializes the test suite context.
func TestMain(m *testing.M) {
	var waitGroup sync.WaitGroup
	waitGroup.Add(1)
	northbound.SetUpServer(10123, Service{}, &waitGroup)
	waitGroup.Wait()
	os.Exit(m.Run())
}

func getClient() (*grpc.ClientConn, proto.ConfigDiagsClient) {
	conn := northbound.Connect(northbound.Address, northbound.Opts...)
	return conn, proto.NewConfigDiagsClient(conn)
}

func Test_GetChanges_All(t *testing.T) {
	testGetChanges(t, "")
}

func Test_GetChanges_Change(t *testing.T) {
	testGetChanges(t, "1EvL+9i1ZGjKyuSjxh1hlfZ9/74=")
}

func testGetChanges(t *testing.T, changeID string) {
	conn, client := getClient()
	defer conn.Close()

	changesReq := &proto.ChangesRequest{ChangeIds: make([]string, 0)}
	if changeID != "" {
		changesReq.ChangeIds = append(changesReq.ChangeIds, changeID)
	}

	stream, err := client.GetChanges(context.Background(), changesReq)
	assert.NilError(t, err, "unable to issue request")
	var id = ""
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

func Test_GetConfigurations_All(t *testing.T) {
	testGetConfigurations(t, "")
}

func Test_GetConfigurations_Device(t *testing.T) {
	testGetConfigurations(t, "Device2")
}

func testGetConfigurations(t *testing.T, deviceID string) {
	conn, client := getClient()
	defer conn.Close()

	configReq := &proto.ConfigRequest{DeviceIds: make([]string, 0)}
	if deviceID != "" {
		configReq.DeviceIds = append(configReq.DeviceIds, deviceID)
	}
	stream, err := client.GetConfigurations(context.Background(), configReq)
	assert.NilError(t, err, "unable to issue request")
	var name = ""
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
