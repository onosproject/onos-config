// SPDX-FileCopyrightText: 2020-present Open Networking Foundation <info@opennetworking.org>
//
// SPDX-License-Identifier: Apache-2.0

package gnmi

import (
	"github.com/gogo/protobuf/proto"
	configapi "github.com/onosproject/onos-api/go/onos/config/v2"
	"github.com/onosproject/onos-config/pkg/utils"
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

func createOverrideExt(t *testing.T, id configapi.TargetID, targetType configapi.TargetType, targetVersion configapi.TargetVersion) *gnmi_ext.Extension {
	ext, err := utils.TargetVersionOverrideExtension(id, targetType, targetVersion)
	assert.NilError(t, err)
	return ext
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
		{"SetReqSingleExtensionLiteral", &gnmi.GetRequest{Extension: []*gnmi_ext.Extension{{Ext: &gnmi_ext.Extension_RegisteredExt{
			RegisteredExt: &gnmi_ext.RegisteredExtension{
				Id:  111,
				Msg: []byte{0x8, 0x1},
				// for gnmi_cli - use msg:"\x08\x01"
			},
		},
		}}}, configapi.TransactionStrategy_SYNCHRONOUS},
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

func TestGetTargetVersionOverrides(t *testing.T) {
	const testTargetLen = uint8(len("test-target"))
	const testDeviceLen = uint8(len("testdevice"))
	const Ver100Len = uint8(len("1.0.0"))

	tests := []struct {
		name string
		args interface{} // a gNMI Get or Set
		want configapi.TargetVersionOverrides
	}{
		{"SetReqEmptyExtension", &gnmi.SetRequest{Extension: []*gnmi_ext.Extension{}}, configapi.TargetVersionOverrides{Overrides: nil}},
		{"SetReqExtensionTestDevice1", &gnmi.SetRequest{Extension: []*gnmi_ext.Extension{createOverrideExt(t, "test-target", "testdevice", "1.0.x")}},
			configapi.TargetVersionOverrides{Overrides: map[string]*configapi.TargetTypeVersion{"test-target": {TargetType: "testdevice", TargetVersion: "1.0.x"}}}},
		{"SetReqExtensionTestDevice1Literal", &gnmi.SetRequest{Extension: []*gnmi_ext.Extension{
			{
				Ext: &gnmi_ext.Extension_RegisteredExt{
					RegisteredExt: &gnmi_ext.RegisteredExtension{
						Id:  configapi.TargetVersionOverridesID,
						Msg: []byte{0x0a, 0x22, 0x0a, testTargetLen, 't', 'e', 's', 't', '-', 't', 'a', 'r', 'g', 'e', 't', 0x12, 0x13, 0x0a, testDeviceLen, 't', 'e', 's', 't', 'd', 'e', 'v', 'i', 'c', 'e', 0x12, Ver100Len, '1', '.', '0', '.', 'x'},
						// Got by printing out the value of 'b' in TargetVersionOverrideExtension()
						// for gnmi_cli - use msg:"\x0a\x22\x0a\x0btest-target\x12\x13\x0a\x0atestdevice\x12\x051.0.x"
					},
				},
			},
		}},
			configapi.TargetVersionOverrides{Overrides: map[string]*configapi.TargetTypeVersion{"test-target": {TargetType: "testdevice", TargetVersion: "1.0.x"}}}},
		{"SetReqExtensionAether21", &gnmi.SetRequest{Extension: []*gnmi_ext.Extension{createOverrideExt(t, "starbucks", "aether", "2.1.x")}},
			configapi.TargetVersionOverrides{Overrides: map[string]*configapi.TargetTypeVersion{"starbucks": {TargetType: "aether", TargetVersion: "2.1.x"}}}},
		{"SetReqExtensionAether21Literal", &gnmi.SetRequest{Extension: []*gnmi_ext.Extension{
			{
				Ext: &gnmi_ext.Extension_RegisteredExt{
					RegisteredExt: &gnmi_ext.RegisteredExtension{
						Id:  configapi.TargetVersionOverridesID,
						Msg: []byte{0x0a, 0x1c, 0x0a, 9, 's', 't', 'a', 'r', 'b', 'u', 'c', 'k', 's', 0x12, 0x0f, 0x0a, 6, 'a', 'e', 't', 'h', 'e', 'r', 0x12, 5, '2', '.', '1', '.', 'x'},
						// Got by printing out the value of 'b' in TargetVersionOverrideExtension()
						// for gnmi_cli - use msg:"\x0a\x1c\x0a\x09starbucks\x12\x0f\x0a\x06aether\x12\x052.1.x"
					},
				},
			},
		}},
			configapi.TargetVersionOverrides{Overrides: map[string]*configapi.TargetTypeVersion{"starbucks": {TargetType: "aether", TargetVersion: "2.1.x"}}}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			res, err := getTargetVersionOverrides(tt.args)
			assert.NilError(t, err, "Error while getting target version override")
			assert.Equal(t, res.String(), tt.want.String())
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
