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

package integration

import (
	"context"
	"fmt"
	"github.com/golang/protobuf/proto"
	"github.com/openconfig/gnmi/client"
	gclient "github.com/openconfig/gnmi/client/gnmi"
	gpb "github.com/openconfig/gnmi/proto/gnmi"
)

// GNMIGet generates a GET request on the given client for a path on a device
func GNMIGet(ctx context.Context, c client.Impl, device string, path string) (string, error) {
	protoPath := MakeProtoPath(device, path)

	getTZRequest := &gpb.GetRequest{}
	if err := proto.UnmarshalText(protoPath, getTZRequest); err != nil {
		fmt.Printf("unable to parse gnmi.GetRequest from %q : %v", protoPath, err)
		return "", err
	}

	response, err := c.(*gclient.Client).Get(ctx, getTZRequest)
	if err != nil || response == nil {
		return "", err
	}

	return findPathValue(response, path)
}

// GNMISet generates a SET request on the given client for a path on a device
func GNMISet(ctx context.Context, c client.Impl, device string, path string, value string) error {
	protoPath := MakeProtoUpdatePath(device, path, value)

	setTZRequest := &gpb.SetRequest{}
	if err := proto.UnmarshalText(protoPath, setTZRequest); err != nil {
		return err
	}

	_, err := c.(*gclient.Client).Set(ctx, setTZRequest)
	return err
}

// GNMIDelete generates a SET request on the given client to delete a path for a device
func GNMIDelete(ctx context.Context, c client.Impl, device string, path string) error {
	protoPath := MakeProtoDeletePath(device, path)
	setTZRequest := &gpb.SetRequest{}
	if err := proto.UnmarshalText(protoPath, setTZRequest); err != nil {
		return err
	}

	_, err := c.(*gclient.Client).Set(ctx, setTZRequest)
	return err
}

// MakeContext returns a new context for use in GNMI requests
func MakeContext() context.Context {
	// TODO: Investigate using context.WithCancel() here
	ctx := context.Background()
	return ctx
}
