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
//

package gnmi

import (
	"context"
	"fmt"
	"github.com/golang/protobuf/proto"
	"github.com/onosproject/onos-api/go/onos/config/v2"
	configapi "github.com/onosproject/onos-api/go/onos/config/v2"
	protoutils "github.com/onosproject/onos-config/test/utils/proto"
	gnmiclient "github.com/openconfig/gnmi/client"
	gclient "github.com/openconfig/gnmi/client/gnmi"
	protognmi "github.com/openconfig/gnmi/proto/gnmi"
	"github.com/openconfig/gnmi/proto/gnmi_ext"
	"github.com/stretchr/testify/assert"
	"strings"
	"testing"
)

// GetRequest represents a GNMI Get request
type GetRequest struct {
	Ctx        context.Context
	Client     gnmiclient.Impl
	Paths      []protoutils.TargetPath
	Extensions []*gnmi_ext.Extension
	Encoding   protognmi.Encoding
}

// Get performs a Get operation
func (req *GetRequest) Get() ([]protoutils.TargetPath, []*gnmi_ext.Extension, error) {
	protoString := ""
	for _, targetPath := range req.Paths {
		protoString = protoString + MakeProtoPath(targetPath.TargetName, targetPath.Path)
	}
	gnmiGetRequest := &protognmi.GetRequest{}
	if err := proto.UnmarshalText(protoString, gnmiGetRequest); err != nil {
		fmt.Printf("unable to parse gnmi.GetRequest from %q : %v\n", protoString, err)
		return nil, nil, err
	}
	gnmiGetRequest.Encoding = req.Encoding
	gnmiGetRequest.Extension = req.Extensions
	response, err := req.Client.(*gclient.Client).Get(req.Ctx, gnmiGetRequest)
	if err != nil || response == nil {
		return nil, nil, err
	}
	return convertGetResults(response)
}

// CheckValue checks that the correct value is read back via a gnmi get request
func (req *GetRequest) CheckValue(t *testing.T, expectedValue string, expectedExtensions int, failMessage string) {
	t.Helper()
	value, extensions, err := req.Get()
	assert.NoError(t, err, "Get operation returned an unexpected error")
	assert.Equal(t, expectedExtensions, len(extensions))
	assert.Equal(t, expectedValue, value[0].PathDataValue, "%s: %s", failMessage, value)
}

// CheckValueDeleted makes sure that the specified paths have been removed
func (req *GetRequest) CheckValueDeleted(t *testing.T) {
	_, _, err := GetGNMIValue(req.Ctx, req.Client, req.Paths, req.Extensions, protognmi.Encoding_JSON)
	if err == nil {
		assert.Fail(t, "Path not deleted", req.Paths)
	} else if !strings.Contains(err.Error(), "NotFound") {
		assert.Fail(t, "Incorrect error received", err)
	}
}

// SetRequest represents a GNMI Set request
type SetRequest struct {
	Ctx         context.Context
	Client      gnmiclient.Impl
	UpdatePaths []protoutils.TargetPath
	DeletePaths []protoutils.TargetPath
	Extensions  []*gnmi_ext.Extension
	Encoding    protognmi.Encoding
}

// Set performs a Set operation
func (req *SetRequest) Set() (configapi.TransactionID, v2.Index, error) {
	var protoBuilder strings.Builder
	for _, updatePath := range req.UpdatePaths {
		protoBuilder.WriteString(protoutils.MakeProtoUpdatePath(updatePath))
	}
	for _, deletePath := range req.DeletePaths {
		protoBuilder.WriteString(protoutils.MakeProtoDeletePath(deletePath.TargetName, deletePath.Path))
	}

	setGnmiRequest := &protognmi.SetRequest{}

	if err := proto.UnmarshalText(protoBuilder.String(), setGnmiRequest); err != nil {
		return "", 0, err
	}

	setGnmiRequest.Extension = req.Extensions
	setResult, err := req.Client.(*gclient.Client).Set(req.Ctx, setGnmiRequest)
	if err != nil {
		return "", 0, err
	}
	id, index, err := extractSetTransactionID(setResult)
	return id, index, err
}
