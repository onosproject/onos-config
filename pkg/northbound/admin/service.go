// Copyright 2019-present Open Networking Foundation
//
// Licensed under the Apache License, Configuration 2.0 (the "License");
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
)

type AdminService struct {
	northbound.Service
}

func (s AdminService) Register(r *grpc.Server) {
	proto.RegisterNorthboundServer(r, AdminServer{})
}

type AdminServer struct {
}

func (s AdminServer) SayHello(ctx context.Context, r *proto.HelloWorldRequest) (*proto.HelloWorldResponse, error) {
	return &proto.HelloWorldResponse{}, nil
}
