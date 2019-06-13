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
	"github.com/onosproject/onos-config/pkg/certs"
	"github.com/onosproject/onos-config/pkg/manager"
	"github.com/onosproject/onos-config/pkg/northbound"
	"github.com/onosproject/onos-config/pkg/northbound/proto"
	"google.golang.org/grpc"
	"gotest.tools/assert"
	"io"
	"log"
	"os"
	"testing"
)

var (
	mgr     *manager.Manager
	opts    []grpc.DialOption
	address string
)

// TestMain initializes the test suite context.
func TestMain(m *testing.M) {
	var err error
	mgr, err = manager.LoadManager(
		"../../../configs/configStore-sample.json",
		"../../../configs/changeStore-sample.json",
		"../../../configs/deviceStore-sample.json",
		"../../../configs/networkStore-sample.json",
	)
	if err != nil {
		log.Fatal("Unable to load manager")
	}

	s := northbound.NewServer(northbound.NewServerConfig("", "", ""))
	s.AddService(Service{})
	go func() {
		err := s.Serve()
		if err != nil {
			log.Fatal("Unable to serve", err)
		}
	}()

	empty := ""
	address = "localhost:5150"
	opts, err = certs.HandleCertArgs(&empty, &empty)
	if err != nil {
		log.Fatal("Error loading cert", err)
	}

	os.Exit(m.Run())
}

func getClient() (*grpc.ClientConn, proto.ConfigDiagsClient) {
	conn := northbound.Connect(address, opts...)
	return conn, proto.NewConfigDiagsClient(conn)
}

func Test_GetChanges_All(t *testing.T) {
	testGetChanges(t, "")
}

func Test_GetChanges_Change(t *testing.T) {
	testGetChanges(t, "5FTJ5QDC0y/RoFnJPfdh8NsB+4Y=")
}

func testGetChanges(t *testing.T, changeID string) {
	conn, client := getClient()
	defer conn.Close()

	changesReq := &proto.ChangesRequest{ChangeIds: make([]string, 0)}
	if changeID != "" {
		changesReq.ChangeIds = append(changesReq.ChangeIds, changeID)
	}

	stream, err := client.GetChanges(context.Background(), changesReq)
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
