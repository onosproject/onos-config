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

// Client Mocks
package cli

import (
	"context"
	"fmt"
	"github.com/onosproject/onos-config/pkg/northbound/admin"
	"github.com/openconfig/gnmi/proto/gnmi"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
	"io"
)

type mockConfigAdminServiceClient struct {
	rollBackID string
}

type mockConfigAdminService_ListRegisteredModelsClient struct{}

var called = 1 //  hack
func (c mockConfigAdminService_ListRegisteredModelsClient) Recv() (*admin.ModelInfo, error) {
	if called <= 2 {
		index := called
		called++
		roPaths := make ([]*admin.ReadOnlyPath, 1)
		roPaths[0] = &admin.ReadOnlyPath{
			Path:                 fmt.Sprintf("/root/ropath/path%d", index),
			SubPath:              nil,
		}
		modelData := make ([]*gnmi.ModelData, 1)
		modelData[0] = &gnmi.ModelData{
			Name:                 "UT NAME",
			Organization:         "UT ORG",
			Version:              "3.3.3",
		}
		return &admin.ModelInfo{
			Name: fmt.Sprintf("Model-%d", index),
			Version: "1.0",
			Module: fmt.Sprintf("Module-%d", index),
			ReadOnlyPath: roPaths,
			ModelData: modelData,
		}, nil
	}
	return nil, io.EOF
}

func (c mockConfigAdminService_ListRegisteredModelsClient) Header() (metadata.MD, error) {
	panic("implement me")
}

func (c mockConfigAdminService_ListRegisteredModelsClient) Trailer() metadata.MD {
	panic("implement me")
}

func (c mockConfigAdminService_ListRegisteredModelsClient) CloseSend() error {
	panic("implement me")
}

func (c mockConfigAdminService_ListRegisteredModelsClient) Context() context.Context {
	panic("implement me")
}

func (c mockConfigAdminService_ListRegisteredModelsClient) SendMsg(m interface{}) error {
	panic("implement me")
}

func (c mockConfigAdminService_ListRegisteredModelsClient) RecvMsg(m interface{}) error {
	panic("implement me")
}

var LastCreatedClient *mockConfigAdminServiceClient

func (c mockConfigAdminServiceClient) RegisterModel(ctx context.Context, in *admin.RegisterRequest, opts ...grpc.CallOption) (*admin.RegisterResponse, error) {
	return nil, nil
}

func (c mockConfigAdminServiceClient) UploadRegisterModel(ctx context.Context, opts ...grpc.CallOption) (admin.ConfigAdminService_UploadRegisterModelClient, error) {
	return nil, nil
}

func (c mockConfigAdminServiceClient) ListRegisteredModels(ctx context.Context, in *admin.ListModelsRequest, opts ...grpc.CallOption) (admin.ConfigAdminService_ListRegisteredModelsClient, error) {
	client := mockConfigAdminService_ListRegisteredModelsClient{}
	return client, nil
}

func (c mockConfigAdminServiceClient) GetNetworkChanges(ctx context.Context, in *admin.NetworkChangesRequest, opts ...grpc.CallOption) (admin.ConfigAdminService_GetNetworkChangesClient, error) {
	return nil, nil
}

func (c mockConfigAdminServiceClient) RollbackNetworkChange(ctx context.Context, in *admin.RollbackRequest, opts ...grpc.CallOption) (*admin.RollbackResponse, error) {
	response := &admin.RollbackResponse{
		Message:              "Rollback was successful",
		XXX_NoUnkeyedLiteral: struct{}{},
		XXX_unrecognized:     nil,
		XXX_sizecache:        0,
	}
	LastCreatedClient.rollBackID = in.Name
	return response, nil
}

func setUpMockClients() {
	admin.ConfigAdminClientFactory = func(cc *grpc.ClientConn) admin.ConfigAdminServiceClient {
		LastCreatedClient = &mockConfigAdminServiceClient{
			rollBackID: "",
		}
		return LastCreatedClient
	}
}
