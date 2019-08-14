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
	"github.com/onosproject/onos-config/pkg/northbound"
	"github.com/onosproject/onos-config/pkg/northbound/admin"
	"github.com/onosproject/onos-config/pkg/store/change"
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

func getClient() (*grpc.ClientConn, ConfigDiagsClient, OpStateDiagsClient) {
	conn := northbound.Connect(northbound.Address, northbound.Opts...)
	return conn, NewConfigDiagsClient(conn), NewOpStateDiagsClient(conn)
}

func Test_GetChanges_All(t *testing.T) {
	testGetChanges(t, "")
}

func Test_GetChanges_Change(t *testing.T) {
	testGetChanges(t, "tAk3GZSh1qbdhdm5414r46RLvqw=")
}

func testGetChanges(t *testing.T, changeID string) {
	conn, client, _ := getClient()
	defer conn.Close()

	changesReq := &ChangesRequest{ChangeIDs: make([]string, 0)}
	if changeID != "" {
		changesReq.ChangeIDs = append(changesReq.ChangeIDs, changeID)
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
	conn, client, _ := getClient()
	defer conn.Close()

	configReq := &ConfigRequest{DeviceIDs: make([]string, 0)}
	if deviceID != "" {
		configReq.DeviceIDs = append(configReq.DeviceIDs, deviceID)
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

func Test_GetOpState_DeviceSubscribe(t *testing.T) {
	conn, _, client := getClient()
	defer conn.Close()

	opStateReq := &OpStateRequest{DeviceId: "Device2", Subscribe: true}

	stream, err := client.GetOpState(context.Background(), opStateReq)
	assert.NilError(t, err, "unable to issue request")
	var pv *admin.ChangeValue
	for {
		in, err := stream.Recv()
		if err == io.EOF || in == nil {
			break
		}
		assert.NilError(t, err, "unable to receive message")
		pv = in.Pathvalue
		value, err := change.CreateTypedValue(pv.Value, change.ValueType(pv.ValueType), []int{})
		assert.NilError(t, err)

		switch pv.Path {
		case "/cont1a/cont2a/leaf2c":
			assert.Equal(t, pv.ValueType, admin.ChangeValueType_STRING)
			switch value.String() {
			case "test1":
				//From the initial response
				fmt.Println("Got value test1 on ", pv.Path)
			case "test2":
				fmt.Println("Got async follow up for ", pv.Path, "closing subscription")
				//Now close the subscription
				conn.Close()
				return
			default:
				t.Fatal("Unexpected valued for", pv.Path, value.String())
			}
		case "/cont1b-state/leaf2d":
			assert.Equal(t, pv.ValueType, admin.ChangeValueType_UINT)
			assert.Equal(t, value.String(), "12345")
		default:
			t.Fatal("Unexpected path in opstate cache for Device2", pv.Path)
		}
	}
	err = stream.CloseSend()
	assert.NilError(t, err, "unable to close stream")
}
