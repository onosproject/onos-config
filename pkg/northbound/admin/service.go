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
	"io"
)

// Service is a Service implementation for administration
type Service struct {
	northbound.Service
}

// Register registers the Service with the grpc server
func (s Service) Register(r *grpc.Server) {
	proto.RegisterNorthboundServer(r, Server{})
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

// Shell provides CLI shell
func (s Server) Shell(stream proto.Northbound_ShellServer) error {
	for {
		in, err := stream.Recv()
		if err == io.EOF {
			return nil
		}
		if err != nil {
			return err
		}
		// Switch on the menu command
		s.shell.Exec(in.Str, func(s string) error {
			if err := stream.Send(&proto.Line{Str: s}); err != nil {
				return err
			}
			return nil
		})
	}
}

// SayHello says hello!
func (s Server) SayHello(ctx context.Context, r *proto.HelloWorldRequest) (*proto.HelloWorldResponse, error) {
	return &proto.HelloWorldResponse{}, nil
}
