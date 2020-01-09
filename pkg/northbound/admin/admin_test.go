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
	"errors"
	"github.com/golang/mock/gomock"
	"github.com/onosproject/onos-config/api/admin"
	"github.com/onosproject/onos-config/pkg/manager"
	mockstore "github.com/onosproject/onos-config/pkg/test/mocks/store"
	"github.com/onosproject/onos-config/pkg/test/mocks/store/cache"
	"google.golang.org/grpc"
	"google.golang.org/grpc/test/bufconn"
	"gotest.tools/assert"
	"net"
	"os"
	"testing"
)

// TestMain initializes the test suite context.
func TestMain(m *testing.M) {
	os.Exit(m.Run())
}

func setUpServer(t *testing.T) (*manager.Manager, *grpc.ClientConn, admin.ConfigAdminServiceClient, *grpc.Server) {
	lis := bufconn.Listen(1024 * 1024)
	s := grpc.NewServer()

	admin.RegisterConfigAdminServiceServer(s, &Server{})

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

	ctrl := gomock.NewController(t)
	mgrTest, err := manager.NewManager(
		mockstore.NewMockLeadershipStore(ctrl),
		mockstore.NewMockMastershipStore(ctrl),
		mockstore.NewMockDeviceChangesStore(ctrl),
		mockstore.NewMockDeviceStateStore(ctrl),
		mockstore.NewMockDeviceStore(ctrl),
		cache.NewMockCache(ctrl),
		mockstore.NewMockNetworkChangesStore(ctrl),
		mockstore.NewMockNetworkSnapshotStore(ctrl),
		mockstore.NewMockDeviceSnapshotStore(ctrl),
		true)
	if err != nil {
		log.Error("Unable to load manager")
	}

	return mgrTest, conn, client, s
}

func Test_RollbackNetworkChange_BadName(t *testing.T) {
	mgrTest, conn, client, server := setUpServer(t)
	defer server.Stop()
	defer conn.Close()

	mockNwChStore, ok := mgrTest.NetworkChangesStore.(*mockstore.MockNetworkChangesStore)
	assert.Assert(t, ok, "casting mock store")

	mockNwChStore.EXPECT().Get(gomock.Any()).Return(nil, errors.New("Rollback aborted. Network change BAD CHANGE not found"))
	_, err := client.RollbackNetworkChange(context.Background(), &admin.RollbackRequest{Name: "BAD CHANGE"})
	assert.ErrorContains(t, err, "Rollback aborted. Network change BAD CHANGE not found")
}

func Test_RollbackNetworkChange_NoChange(t *testing.T) {
	mgrTest, conn, client, server := setUpServer(t)
	defer server.Stop()
	defer conn.Close()

	mockNwChStore, ok := mgrTest.NetworkChangesStore.(*mockstore.MockNetworkChangesStore)
	assert.Assert(t, ok, "casting mock store")

	mockNwChStore.EXPECT().Get(gomock.Any()).Return(nil, errors.New("change is not specified"))
	_, err := client.RollbackNetworkChange(context.Background(), &admin.RollbackRequest{Name: ""})
	assert.ErrorContains(t, err, "is not")
}
