// Copyright 2020-present Open Networking Foundation.
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

package gnmi

import (
	"context"
	topoapi "github.com/onosproject/onos-api/go/onos/topo"
	"github.com/onosproject/onos-lib-go/pkg/errors"
	baseClient "github.com/openconfig/gnmi/client"
	gclient "github.com/openconfig/gnmi/client/gnmi"
	"google.golang.org/grpc"
	"google.golang.org/grpc/connectivity"
	"google.golang.org/grpc/credentials"
	"math"
	"time"
)

const defaultKeepAliveInterval = 30 * time.Second

type Target interface {
	Connect(ctx context.Context) error
	Close(ctx context.Context) error
}

func newTarget(manager *connManager, target *topoapi.Object) Target {
	return &connTarget{
		connManager: manager,
		target:      target,
	}
}

type connTarget struct {
	*connManager
	target     *topoapi.Object
	sbConn     Conn
	clientConn *grpc.ClientConn
}

func (t *connTarget) Connect(ctx context.Context) error {
	if t.target.Type != topoapi.Object_ENTITY {
		return errors.NewInvalid("object is not a topo entity %v+", t.target)
	}

	typeKindID := string(t.target.GetEntity().KindID)
	if len(typeKindID) == 0 {
		return errors.NewInvalid("target entity %s must have a 'kindID' to work with onos-config", t.target.ID)
	}

	destination, err := newDestination(t.target)
	if err != nil {
		log.Warnf("Failed to create a new target %s", err)
		return err
	}

	log.Infof("Connecting to gNMI target: %+v", destination)
	opts := []grpc.DialOption{
		grpc.WithDefaultCallOptions(grpc.MaxCallRecvMsgSize(math.MaxInt32)),
	}
	gnmiClient, clientConn, err := connect(ctx, *destination, opts...)
	if err != nil {
		log.Warnf("Failed to connect to the gNMI target %s: %s", destination.Target, err)
		return err
	}
	t.clientConn = clientConn

	keepAliveCtx, keepAliveCancel := context.WithCancel(context.Background())
	go func() {
		ticker := time.NewTicker(defaultKeepAliveInterval)
		for {
			select {
			case _, ok := <-ticker.C:
				if !ok {
					return
				}
				clientConn.Connect()
			case <-keepAliveCtx.Done():
				ticker.Stop()
			}
		}
	}()

	go func() {
		state := clientConn.GetState()
		switch state {
		case connectivity.Connecting, connectivity.Ready, connectivity.Idle:
			t.sbConn = newConn(t.target.ID, gnmiClient)
			t.addConn(t.sbConn)
		}
		for clientConn.WaitForStateChange(context.Background(), state) {
			state = clientConn.GetState()
			switch state {
			case connectivity.Connecting, connectivity.Ready, connectivity.Idle:
				if t.sbConn == nil {
					t.sbConn = newConn(t.target.ID, gnmiClient)
					t.addConn(t.sbConn)
				}
			case connectivity.TransientFailure:
				if t.sbConn != nil {
					t.removeConn(t.sbConn.ID())
					t.sbConn = nil
				}
			case connectivity.Shutdown:
				if t.sbConn != nil {
					t.removeConn(t.sbConn.ID())
					t.sbConn = nil
				}
				keepAliveCancel()
				return
			}
		}
	}()
	return nil
}

func (t *connTarget) Close(ctx context.Context) error {
	return t.clientConn.Close()
}

func connect(ctx context.Context, d baseClient.Destination, opts ...grpc.DialOption) (*client, *grpc.ClientConn, error) {
	switch d.TLS {
	case nil:
		opts = append(opts, grpc.WithInsecure())
	default:
		opts = append(opts, grpc.WithTransportCredentials(credentials.NewTLS(d.TLS)))
	}

	if d.Credentials != nil {
		secure := true
		if d.TLS == nil {
			secure = false
		}
		pc := newPassCred(d.Credentials.Username, d.Credentials.Password, secure)
		opts = append(opts, grpc.WithPerRPCCredentials(pc))
	}

	gCtx, cancel := context.WithTimeout(ctx, d.Timeout)
	defer cancel()

	addr := ""
	if len(d.Addrs) != 0 {
		addr = d.Addrs[0]
	}
	conn, err := grpc.DialContext(gCtx, addr, opts...)
	if err != nil {
		return nil, nil, errors.NewInternal("Dialer(%s, %v): %v", addr, d.Timeout, err)
	}

	cl, err := gclient.NewFromConn(gCtx, conn, d)
	if err != nil {
		return nil, nil, err
	}

	gnmiClient := &client{
		client: cl,
	}

	return gnmiClient, conn, nil
}
