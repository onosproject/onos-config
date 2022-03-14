// SPDX-FileCopyrightText: 2020-present Open Networking Foundation <info@opennetworking.org>
//
// SPDX-License-Identifier: Apache-2.0

package gnmi

import (
	"github.com/gogo/protobuf/proto"
	configapi "github.com/onosproject/onos-api/go/onos/config/v2"
	"github.com/openconfig/gnmi/proto/gnmi"
	"github.com/openconfig/gnmi/proto/gnmi_ext"
	"gotest.tools/assert"
	"testing"
)

func createExtension(t *testing.T, strategy configapi.TransactionStrategy_Synchronicity) *gnmi_ext.Extension {
	ext := configapi.TransactionStrategy{
		Synchronicity: strategy,
	}
	b, err := ext.Marshal()
	if err != nil {
		t.Fatalf("cannot marshal transaction strategy: %s", err)
	}
	return &gnmi_ext.Extension{
		Ext: &gnmi_ext.Extension_RegisteredExt{
			RegisteredExt: &gnmi_ext.RegisteredExtension{
				Id:  111,
				Msg: b,
			},
		},
	}
}

func TestGetTransactionStategy(t *testing.T) {
	tests := []struct {
		name string
		args interface{} // a gNMI Get or Set
		want configapi.TransactionStrategy_Synchronicity
	}{
		{"SetReqSingleExtension", &gnmi.SetRequest{Extension: []*gnmi_ext.Extension{}}, configapi.TransactionStrategy_ASYNCHRONOUS},
		{"SetReqSingleExtension", &gnmi.SetRequest{Extension: []*gnmi_ext.Extension{createExtension(t, configapi.TransactionStrategy_SYNCHRONOUS)}}, configapi.TransactionStrategy_SYNCHRONOUS},
		{"GetReqSingleExtension", &gnmi.GetRequest{Extension: []*gnmi_ext.Extension{createExtension(t, configapi.TransactionStrategy_SYNCHRONOUS)}}, configapi.TransactionStrategy_SYNCHRONOUS},
		{"SetReqMultipleExtensions", &gnmi.SetRequest{Extension: []*gnmi_ext.Extension{
			createExtension(t, configapi.TransactionStrategy_SYNCHRONOUS),
			{Ext: &gnmi_ext.Extension_RegisteredExt{
				RegisteredExt: &gnmi_ext.RegisteredExtension{
					Id:  100,
					Msg: []byte("request-name"),
				},
			}},
		}},
			configapi.TransactionStrategy_SYNCHRONOUS},
		{"SetReqMultipleExtensions-2", &gnmi.SetRequest{Extension: []*gnmi_ext.Extension{
			{Ext: &gnmi_ext.Extension_RegisteredExt{
				RegisteredExt: &gnmi_ext.RegisteredExtension{
					Id:  100,
					Msg: []byte("request-name"),
				},
			}},
			createExtension(t, configapi.TransactionStrategy_SYNCHRONOUS),
		}},
			configapi.TransactionStrategy_SYNCHRONOUS},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			res, err := getTransactionStrategy(tt.args)
			assert.NilError(t, err, "Error while getting transaction strategy")
			assert.Equal(t, res.Synchronicity, tt.want)
		})
	}
}

func TestExtractExtension(t *testing.T) {

	type testArgs struct {
		extensions []*gnmi_ext.Extension
		extID      gnmi_ext.ExtensionID
		extType    proto.Message
	}

	tests := []struct {
		name      string
		args      testArgs
		validator func(t2 *testing.T, i interface{})
	}{
		{"requestName", testArgs{
			[]*gnmi_ext.Extension{
				{Ext: &gnmi_ext.Extension_RegisteredExt{
					RegisteredExt: &gnmi_ext.RegisteredExtension{
						Id:  100,
						Msg: []byte("request-name"),
					},
				}},
			},
			100,
			nil,
		},
			func(t2 *testing.T, i interface{}) {
				res, ok := i.([]byte)
				if !ok {
					t2.Fail()
				}
				assert.Equal(t2, "request-name", string(res))
			},
		},
		{"transactionStrategy", testArgs{
			[]*gnmi_ext.Extension{
				createExtension(t, configapi.TransactionStrategy_SYNCHRONOUS),
			},
			111,
			&configapi.TransactionStrategy{},
		},
			func(t2 *testing.T, i interface{}) {
				strategy, ok := i.(*configapi.TransactionStrategy)
				if !ok {
					t2.Fail()
				}
				assert.Equal(t2, strategy.Synchronicity, configapi.TransactionStrategy_SYNCHRONOUS)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			res, err := extractExtension(tt.args.extensions, tt.args.extID, tt.args.extType)
			assert.NilError(t, err, "Error while getting extension")
			tt.validator(t, res)
		})
	}
}
