// SPDX-FileCopyrightText: 2020-present Open Networking Foundation <info@opennetworking.org>
//
// SPDX-License-Identifier: Apache-2.0
//

package gnmi

import (
	"context"
	"fmt"
	"os"
	"strings"
	"testing"

	"github.com/golang/protobuf/proto"
	"github.com/onosproject/onos-api/go/onos/config/v2"
	configapi "github.com/onosproject/onos-api/go/onos/config/v2"
	"github.com/onosproject/onos-config/pkg/utils"
	protoutils "github.com/onosproject/onos-config/test/utils/proto"
	"github.com/onosproject/onos-lib-go/pkg/errors"
	gnmiclient "github.com/openconfig/gnmi/client"
	gclient "github.com/openconfig/gnmi/client/gnmi"
	gnmiapi "github.com/openconfig/gnmi/proto/gnmi"
	"github.com/openconfig/gnmi/proto/gnmi_ext"
	"github.com/stretchr/testify/assert"
)

// GetRequest represents a GNMI Get request
type GetRequest struct {
	Ctx        context.Context
	Client     gnmiclient.Impl
	Paths      []protoutils.GNMIPath
	Prefix     protoutils.GNMIPath
	Extensions []*gnmi_ext.Extension
	Encoding   gnmiapi.Encoding
	DataType   gnmiapi.GetRequest_DataType
}

// convertGetResults extracts path/value pairs from a GNMI get response
func convertGetResults(response *gnmiapi.GetResponse) ([]protoutils.GNMIPath, error) {
	result := make([]protoutils.GNMIPath, 0)

	for _, notification := range response.Notification {
		for _, update := range notification.Update {
			value := update.Val

			var targetPath protoutils.GNMIPath
			targetPath.TargetName = update.Path.Target
			pathString := ""

			for _, elem := range update.Path.Elem {
				pathString = pathString + "/" + elem.Name
				if len(elem.GetKey()) != 0 {
					pathString = pathString + "["
					for key, value := range elem.GetKey() {
						pathString = pathString + key + "=" + value
					}
					pathString = pathString + "]"
				}
			}
			targetPath.Path = pathString

			targetPath.PathDataType = "string_val"
			if value != nil {
				targetPath.PathDataValue = utils.StrVal(value)
			} else {
				targetPath.PathDataValue = ""
			}
			result = append(result, targetPath)

		}
	}

	return result, nil
}

// extractSetTransactionInfo returns the transaction ID and Index from a set operation response
func extractSetTransactionInfo(response *gnmiapi.SetResponse) (configapi.TransactionID, v2.Index, error) {
	var transactionInfo *configapi.TransactionInfo
	extensionsSet := response.Extension
	for _, extension := range extensionsSet {
		if ext, ok := extension.Ext.(*gnmi_ext.Extension_RegisteredExt); ok &&
			ext.RegisteredExt.Id == configapi.TransactionInfoExtensionID {
			bytes := ext.RegisteredExt.Msg
			transactionInfo = &configapi.TransactionInfo{}
			err := proto.Unmarshal(bytes, transactionInfo)
			if err != nil {
				return "", 0, err
			}
		}
	}

	if transactionInfo == nil {
		return "", 0, errors.NewNotFound("transaction ID extension not found")
	}

	return transactionInfo.ID, transactionInfo.Index, nil
}

// Get performs a Get operation
func (req *GetRequest) Get() ([]protoutils.GNMIPath, error) {
	protoString := ""
	if req.Prefix.Path != "" || req.Prefix.TargetName != "" {
		protoString = protoString + protoutils.MakeProtoPrefix(req.Prefix.TargetName, req.Prefix.Path)
	}

	for _, targetPath := range req.Paths {
		protoString = protoString + MakeProtoPath(targetPath.TargetName, targetPath.Path)
	}
	gnmiGetRequest := &gnmiapi.GetRequest{}
	fmt.Fprintf(os.Stderr, "gnmi request is %v\n", protoString)
	if err := proto.UnmarshalText(protoString, gnmiGetRequest); err != nil {
		fmt.Printf("unable to parse gnmi.GetRequest from %q : %v\n", protoString, err)
		return nil, err
	}
	gnmiGetRequest.Encoding = req.Encoding
	gnmiGetRequest.Extension = req.Extensions
	gnmiGetRequest.Type = req.DataType
	response, err := req.Client.(*gclient.Client).Get(req.Ctx, gnmiGetRequest)
	if err != nil || response == nil {
		return nil, err
	}
	return convertGetResults(response)
}

// CheckValues checks that the correct value is read back via a gnmi get request
func (req *GetRequest) CheckValues(t *testing.T, expectedValues ...string) {
	t.Helper()
	value, err := req.Get()
	assert.NoError(t, err, "Get operation returned an unexpected error")
	for i, expectedValue := range expectedValues {
		assert.Equal(t, expectedValue, value[i].PathDataValue, "Checked value is incorrect: %s", value[i])
	}
}

// CheckValuesDeleted makes sure that the specified paths have been removed
func (req *GetRequest) CheckValuesDeleted(t *testing.T) {
	values, err := req.Get()
	if err != nil {
		if strings.Contains(err.Error(), "NotFound") {
			return
		}
		assert.NoError(t, err)
	}
	for _, value := range values {
		assert.Equal(t, "", value.PathDataValue)
	}
}

// SetRequest represents a GNMI Set request
type SetRequest struct {
	Ctx         context.Context
	Client      gnmiclient.Impl
	Prefix      protoutils.GNMIPath
	UpdatePaths []protoutils.GNMIPath
	DeletePaths []protoutils.GNMIPath
	Extensions  []*gnmi_ext.Extension
	Encoding    gnmiapi.Encoding
}

// Set performs a Set operation
func (req *SetRequest) Set() (configapi.TransactionID, v2.Index, error) {
	var protoBuilder strings.Builder
	if req.Prefix.Path != "" || req.Prefix.TargetName != "" {
		protoBuilder.WriteString(protoutils.MakeProtoPrefix(req.Prefix.TargetName, req.Prefix.Path))
	}
	for _, updatePath := range req.UpdatePaths {
		protoBuilder.WriteString(protoutils.MakeProtoUpdatePath(updatePath))
	}
	for _, deletePath := range req.DeletePaths {
		protoBuilder.WriteString(protoutils.MakeProtoDeletePath(deletePath.TargetName, deletePath.Path))
	}

	setGnmiRequest := &gnmiapi.SetRequest{}

	if err := proto.UnmarshalText(protoBuilder.String(), setGnmiRequest); err != nil {
		return "", 0, err
	}

	setGnmiRequest.Extension = req.Extensions
	setResult, err := req.Client.(*gclient.Client).Set(req.Ctx, setGnmiRequest)
	if err != nil {
		return "", 0, err
	}
	id, index, err := extractSetTransactionInfo(setResult)
	return id, index, err
}

// SetOrFail performs a Set operation and fails the test if an error occurs
func (req *SetRequest) SetOrFail(t *testing.T) (configapi.TransactionID, v2.Index) {
	transactionID, transactionIndex, err := req.Set()
	assert.NoError(t, err)
	return transactionID, transactionIndex
}
