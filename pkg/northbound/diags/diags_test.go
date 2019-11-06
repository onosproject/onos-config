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
	"github.com/onosproject/onos-config/api/diags"
	devicechangetypes "github.com/onosproject/onos-config/api/types/change/device"
	"google.golang.org/grpc"
	"google.golang.org/grpc/test/bufconn"
	"gotest.tools/assert"
	"io"
	log "k8s.io/klog"
	"net"
	"os"
	"testing"
	"time"
)

//TestMain initializes the test suite context.
func TestMain(m *testing.M) {
	log.SetOutput(os.Stdout)
	os.Exit(m.Run())
}

func Test_GetOpState_DeviceSubscribe(t *testing.T) {
	lis := bufconn.Listen(1024 * 1024)
	s := grpc.NewServer()
	defer s.Stop()

	diags.RegisterOpStateDiagsServer(s, &Server{})

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

	client := diags.CreateOpStateDiagsClient(conn)

	opStateReq := &diags.OpStateRequest{DeviceId: "Device2", Subscribe: true}

	stream, err := client.GetOpState(context.TODO(), opStateReq)
	assert.NilError(t, err, "expected to get device-1")
	for {
		in, err := stream.Recv()
		if err == io.EOF || in == nil {
			break
		}
		assert.NilError(t, err, "unable to receive message")
		pv := in.GetPathvalue()

		switch pv.GetPath() {
		case "/cont1a/cont2a/leaf2c":
			assert.Equal(t, pv.GetValue().GetType(), devicechangetypes.ValueType_STRING)
			switch pv.GetValue().ValueToString() {
			case "test1":
				//From the initial response
				fmt.Println("Got value test1 on ", pv.GetPath())
			case "test2":
				fmt.Println("Got async follow up for ", pv.GetPath(), "closing subscription")
				//Now close the subscription
				conn.Close()
				return
			default:
				t.Fatal("Unexpected value for", pv.Path, pv.GetValue().ValueToString())
			}
		case "/cont1b-state/leaf2d":
			assert.Equal(t, pv.GetValue().GetType(), devicechangetypes.ValueType_UINT)
			assert.Equal(t, pv.GetValue().ValueToString(), "12345")
		default:
			t.Fatal("Unexpected path in opstate cache for Device2", pv.Path)
		}
	}
	time.Sleep(time.Millisecond * 10)
	err = stream.CloseSend()
	assert.NilError(t, err, "unable to close stream")
}
