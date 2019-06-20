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
	"errors"
	"fmt"
	"github.com/golang/protobuf/proto"
	"github.com/onosproject/onos-config/pkg/utils"
	"github.com/onosproject/onos-config/test/env"
	"github.com/openconfig/gnmi/client"
	gclient "github.com/openconfig/gnmi/client/gnmi"
	gpb "github.com/openconfig/gnmi/proto/gnmi"
	"github.com/stretchr/testify/assert"
	"strings"
	"testing"
)

const (
	tzValue = "Europe/Dublin"
	tzPath  = "/system/clock/config/timezone-name"
)

func makeGNMIPath(target string, path string) string {
	var protoBuilder strings.Builder

	protoBuilder.WriteString("<target: '")
	protoBuilder.WriteString(target)
	protoBuilder.WriteString("', ")

	pathElements := utils.SplitPath(path)
	for _, pathElement := range pathElements {
		protoBuilder.WriteString("elem: <name: '")
		protoBuilder.WriteString(pathElement)
		protoBuilder.WriteString("'>")
	}
	protoBuilder.WriteString(">")
	return protoBuilder.String()
}

func makePathProto(target string, path string) string {
	var protoBuilder strings.Builder

	protoBuilder.WriteString("path: ")
	gnmiPath := makeGNMIPath(target, path)
	protoBuilder.WriteString(gnmiPath)
	return protoBuilder.String()
}

func makeSetPathProto(target string, path string, value string) string {
	var protoBuilder strings.Builder

	protoBuilder.WriteString("update: <")
	protoBuilder.WriteString(makePathProto(target, path))
	protoBuilder.WriteString(" val: <string_val: '")
	protoBuilder.WriteString(value)
	protoBuilder.WriteString("'>")
	protoBuilder.WriteString(">")
	return protoBuilder.String()
}

func makeDeletePathProto(target string, path string) string {
	var protoBuilder strings.Builder

	protoBuilder.WriteString("delete: ")
	protoBuilder.WriteString(makeGNMIPath(target, path))
	return protoBuilder.String()
}

func findPathValue(response *gpb.GetResponse, path string) (string, error) {
	if len(response.Notification) != 1 {
		return "", errors.New("response notifications must have one entry")
	}

	pathElements := utils.SplitPath(path)
	responsePathElements := response.Notification[0].Update[0].Path.Elem
	for pathIndex, pathElement := range pathElements {
		responsePathElement := responsePathElements[pathIndex]
		if pathElement != responsePathElement.Name {
			return "", fmt.Errorf("element at %d dos not match - want %s got %s", pathIndex, pathElement, responsePathElement.Name)
		}
	}
	value := response.Notification[0].Update[0].Val
	if value == nil {
		return "", fmt.Errorf("no value found for path %s", path)
	}
	return utils.StrVal(value), nil
}

func gnmiGet(ctx context.Context, c client.Impl, device string, path string) (string, error) {
	protoPath := makePathProto(device, path)

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

func gnmiSet(ctx context.Context, c client.Impl, device string, path string, value string) error {
	protoPath := makeSetPathProto(device, path, value)

	setTZRequest := &gpb.SetRequest{}
	if err := proto.UnmarshalText(protoPath, setTZRequest); err != nil {
		return err
	}

	_, err := c.(*gclient.Client).Set(ctx, setTZRequest)
	return err
}

func gnmiDelete(ctx context.Context, c client.Impl, device string, path string) error {
	protoPath := makeDeletePathProto(device, path)
	setTZRequest := &gpb.SetRequest{}
	if err := proto.UnmarshalText(protoPath, setTZRequest); err != nil {
		return err
	}

	_, err := c.(*gclient.Client).Set(ctx, setTZRequest)
	return err
}

func TestIntegration(t *testing.T) {
	// Get the first configured device from the environment.
	device := env.GetDevices()[0]

	println("1")
	c, err := env.NewGnmiClient(context.Background(), "")
	assert.NoError(t, err)
	assert.True(t, c != nil, "Fetching client returned nil")

	println("2")
	// First lookup should return an error - the path has not been given a value yet
	valueBefore, findErrorBefore := gnmiGet(context.Background(), c, device, tzPath)
	assert.Error(t, findErrorBefore, "no value found for path")
	assert.True(t, valueBefore == "", "Initial query did not return an error\n")

	println("3")
	// Set a value using gNMI client
	errorSet := gnmiSet(context.Background(), c, device, tzPath, tzValue)
	assert.NoError(t, errorSet)

	println("4")
	valueAfter, errorAfter := gnmiGet(context.Background(), c, device, tzPath)
	assert.NoError(t, errorAfter)
	assert.NotEqual(t, "", valueAfter, "Query after set returned an error: %s\n", errorAfter)
	assert.Equal(t, tzValue, valueAfter, "Query after set returned the wrong value: %s\n", valueAfter)

	println("5")
	// Remove the path
	errorDelete := gnmiDelete(context.Background(), c, device, tzPath)
	assert.NoError(t, errorDelete)
}

func init() {
	Registry.Register("test-integration-test", TestIntegration)
}
