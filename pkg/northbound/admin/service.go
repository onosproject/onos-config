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
	"github.com/golang/protobuf/ptypes/timestamp"
	"github.com/onosproject/onos-config/pkg/manager"
	"github.com/onosproject/onos-config/pkg/northbound"
	"github.com/onosproject/onos-config/pkg/northbound/proto"
	"github.com/onosproject/onos-config/pkg/store"
	"google.golang.org/grpc"
)

// Service is a Service implementation for administration
type Service struct {
	northbound.Service
}

// Register registers the Service with the grpc server
func (s Service) Register(r *grpc.Server) {
	proto.RegisterAdminServer(r, Server{})
}

// Interpreter provides means to process remote input and write to remote output
type Interpreter interface {
	// Exec executes a
	Exec(cmd string, o func(s string) error)
}

// Server implements the grpc service for admin
type Server struct {
	shell Interpreter
}

// SetShellInterpreter sets the shell interpreter
func (s Server) SetShellInterpreter(shell Interpreter) {
	s.shell = shell
}

// NetworkChanges provides a stream of submitted network changes
func (s Server) NetworkChanges(r *proto.VoidRequest, stream proto.Admin_NetworkChangesServer) error {
	for _, nc := range manager.GetManager().NetworkStore.Store {

		// Build net change message
		msg := &proto.NetChange{
			Time: &timestamp.Timestamp{Seconds: nc.Created.Unix(), Nanos: int32(nc.Created.Nanosecond())},
			Name: nc.Name,
			User: nc.User,
			Changes: make([]*proto.ConfigChange, 0),
		}

		// Build list of config change messages
		for k, v := range nc.ConfigurationChanges {
			msg.Changes = append(msg.Changes, &proto.ConfigChange{Id: k, Hash: store.B64(v)})
		}

		err := stream.Send(msg)
		if err != nil {
			return err
		}
	}
	return nil
}
