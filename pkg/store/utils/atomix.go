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

package utils

import (
	"context"
	"github.com/atomix/atomix-go-client/pkg/client"
	"github.com/atomix/atomix-go-local/pkg/atomix/local"
	"github.com/atomix/atomix-go-node/pkg/atomix"
	"github.com/atomix/atomix-go-node/pkg/atomix/registry"
	"google.golang.org/grpc"
	"google.golang.org/grpc/test/bufconn"
	"io"
	"net"
	"os"
)

const (
	atomixControllerEnv = "ATOMIX_CONTROLLER"
	atomixNamespaceEnv  = "ATOMIX_NAMESPACE"
	atomixAppEnv        = "ATOMIX_APP"
	atomixRaftGroup     = "ATOMIX_RAFT"
)

// StartLocalNode starts a single local Atomix node
func StartLocalNode() (*atomix.Node, *grpc.ClientConn) {
	lis := bufconn.Listen(1024 * 1024)
	node := local.NewNode(lis, registry.Registry)
	_ = node.Start()

	dialer := func(ctx context.Context, address string) (net.Conn, error) {
		return lis.Dial()
	}

	conn, err := grpc.DialContext(context.Background(), "local", grpc.WithContextDialer(dialer), grpc.WithInsecure())
	if err != nil {
		panic("Failed to dial leadership store")
	}
	return node, conn
}

// NewNodeCloser returns a new closer for an Atomix node
func NewNodeCloser(node *atomix.Node) io.Closer {
	return &nodeCloser{
		node: node,
	}
}

// nodeCloser is a Closer for Atomix nodes
type nodeCloser struct {
	node *atomix.Node
}

func (c *nodeCloser) Close() error {
	return c.node.Stop()
}

func getAtomixController() string {
	return os.Getenv(atomixControllerEnv)
}

func getAtomixNamespace() string {
	return os.Getenv(atomixNamespaceEnv)
}

func getAtomixApp() string {
	return os.Getenv(atomixAppEnv)
}

// GetAtomixRaftGroup get the Atomix Raft group
func GetAtomixRaftGroup() string {
	return os.Getenv(atomixRaftGroup)
}

// GetAtomixClient returns the Atomix client
func GetAtomixClient() (*client.Client, error) {
	opts := []client.Option{
		client.WithNamespace(getAtomixNamespace()),
		client.WithApplication(getAtomixApp()),
	}
	return client.NewClient(getAtomixController(), opts...)
}
