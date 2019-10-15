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

package jsonvalues

import (
	"fmt"
	ds1 "github.com/onosproject/onos-config/modelplugin/Devicesim-1.0.0/devicesim_1_0_0"
	"github.com/onosproject/onos-config/pkg/modelregistry"
	"github.com/onosproject/onos-config/pkg/store"
	types "github.com/onosproject/onos-config/pkg/types/change/device"
	"github.com/openconfig/gnmi/proto/gnmi"
	"github.com/openconfig/goyang/pkg/yang"
	"github.com/openconfig/ygot/ygot"
	"gotest.tools/assert"
	"io/ioutil"
	"testing"
)

type modelPluginTest string

const modelTypeTest = "TestModel"
const modelVersionTest = "0.0.1"
const moduleNameTest = "testmodel.so.1.0.0"

var modelData = []*gnmi.ModelData{
	{Name: "testmodel", Organization: "Open Networking Lab", Version: "2019-07-10"},
}

func (m modelPluginTest) ModelData() (string, string, []*gnmi.ModelData, string) {
	return modelTypeTest, modelVersionTest, modelData, moduleNameTest
}

// UnmarshalConfigValues uses the `generated.go` of the DeviceSim-1.0.0 plugin module
func (m modelPluginTest) UnmarshalConfigValues(jsonTree []byte) (*ygot.ValidatedGoStruct, error) {
	device := &ds1.Device{}
	vgs := ygot.ValidatedGoStruct(device)

	if err := ds1.Unmarshal([]byte(jsonTree), device); err != nil {
		return nil, err
	}

	return &vgs, nil
}

// Validate uses the `generated.go` of the DeviceSim-1.0.0 plugin module
func (m modelPluginTest) Validate(ygotModel *ygot.ValidatedGoStruct, opts ...ygot.ValidationOption) error {
	deviceDeref := *ygotModel
	device, ok := deviceDeref.(*ds1.Device)
	if !ok {
		return fmt.Errorf("unable to convert model in to testdevice_1_0_0")
	}
	return device.Validate()
}

// Schema uses the `generated.go` of the DeviceSim-1.0.0 plugin module
func (m modelPluginTest) Schema() (map[string]*yang.Entry, error) {
	return ds1.UnzipSchema()
}

func Test_correctJsonPathValues(t *testing.T) {

	var modelPluginTest modelPluginTest

	ds1Schema, err := modelPluginTest.Schema()
	assert.NilError(t, err)
	assert.Equal(t, len(ds1Schema), 137)

	readOnlyPaths, _ := modelregistry.ExtractPaths(ds1Schema["Device"], yang.TSUnset, "", "")
	assert.Equal(t, len(readOnlyPaths), 37)

	// All values are taken from testdata/sample-openconfig.json and defined
	// here in the intermediate jsonToValues format
	const systemNtpAuthMismatchNoNs = "/system/ntp/state/auth-mismatch"
	const systemNtpAuthMismatchValue = 123456.00000
	val01 := types.PathValue{Path: systemNtpAuthMismatchNoNs,
		Value: types.NewTypedValueFloat(systemNtpAuthMismatchValue)}

	const systemNtpEnableAuth = "/system/ntp/state/enable-ntp-auth"
	const systemNtpEnableAuthValue = true
	val02 := types.PathValue{Path: systemNtpEnableAuth,
		Value: types.NewTypedValueBool(systemNtpEnableAuthValue)}

	const systemNtpSourceAddr = "/system/ntp/state/ntp-source-address"
	const systemNtpSourceAddrValue = "192.168.0.17"
	val03 := types.PathValue{Path: systemNtpSourceAddr,
		Value: types.NewTypedValueString(systemNtpSourceAddrValue)}

	const iface1Mtu = "/interfaces/interface[1]/state/mtu"
	const iface1MtuValue = 960.0
	val04 := types.PathValue{Path: iface1Mtu,
		Value: types.NewTypedValueFloat(iface1MtuValue)}
	const iface1Name = "/interfaces/interface[1]/name"
	const iface1NameValue = "eth1"
	val05 := types.PathValue{Path: iface1Name,
		Value: types.NewTypedValueString(iface1NameValue)}
	const iface1Desc = "/interfaces/interface[1]/state/description"
	const iface1DescValue = "new if desc"
	val06 := types.PathValue{Path: iface1Desc,
		Value: types.NewTypedValueString(iface1DescValue)}

	const iface1Sub15IfaceIdx = "/interfaces/interface[1]/subinterfaces/subinterface[15]/state/ifindex"
	const iface1Sub15IfaceIdxValue = 10.0
	val07 := types.PathValue{Path: iface1Sub15IfaceIdx,
		Value: types.NewTypedValueFloat(iface1Sub15IfaceIdxValue)}
	const iface1Sub15Idx = "/interfaces/interface[1]/subinterfaces/subinterface[15]/index"
	const iface1Sub15IdxValue = "120"
	val08 := types.PathValue{Path: iface1Sub15Idx,
		Value: types.NewTypedValueString(iface1Sub15IdxValue)}
	const iface1Sub15AdSt = "/interfaces/interface[1]/subinterfaces/subinterface[15]/state/admin-status"
	const iface1Sub15AdStValue = "UP"
	val09 := types.PathValue{Path: iface1Sub15AdSt,
		Value: types.NewTypedValueString(iface1Sub15AdStValue)}

	const sysAaaGr10Svr199Ca = "/system/aaa/server-groups/server-group[10]/servers/server[199]/state/connection-aborts"
	const sysAaaGr10Svr199CaValue = 12.00
	val10 := types.PathValue{Path: sysAaaGr10Svr199Ca,
		Value: types.NewTypedValueFloat(sysAaaGr10Svr199CaValue)}
	const sysAaaGr10Name = "/system/aaa/server-groups/server-group[10]/name"
	const sysAaaGr10NameValue = "g1"
	val11 := types.PathValue{Path: sysAaaGr10Name,
		Value: types.NewTypedValueString(sysAaaGr10NameValue)}
	const sysAaaGr10Svr199Addr = "/system/aaa/server-groups/server-group[10]/servers/server[199]/address"
	const sysAaaGr10Svr199AddrValue = "192.168.0.7"
	val12 := types.PathValue{Path: sysAaaGr10Svr199Addr,
		Value: types.NewTypedValueString(sysAaaGr10Svr199AddrValue)}

	const sysAaaGr12Svr199Ca = "/system/aaa/server-groups/server-group[12]/servers/server[199]/state/connection-aborts"
	const sysAaaGr12Svr199CaValue = 4.0
	val13 := types.PathValue{Path: sysAaaGr12Svr199Ca,
		Value: types.NewTypedValueFloat(sysAaaGr12Svr199CaValue)}
	const sysAaaGr12Name = "/system/aaa/server-groups/server-group[12]/name"
	const sysAaaGr12NameValue = "g2"
	val14 := types.PathValue{Path: sysAaaGr12Name,
		Value: types.NewTypedValueString(sysAaaGr12NameValue)}
	const sysAaaGr12Svr199Addr = "/system/aaa/server-groups/server-group[12]/servers/server[199]/address"
	const sysAaaGr12Svr199AddrValue = "192.168.0.4"
	val15 := types.PathValue{Path: sysAaaGr12Svr199Addr,
		Value: types.NewTypedValueString(sysAaaGr12Svr199AddrValue)}

	const sysAaaGr12Svr200Ca = "/system/aaa/server-groups/server-group[12]/servers/server[200]/state/connection-aborts"
	const sysAaaGr12Svr200CaValue = 5.0
	val16 := types.PathValue{Path: sysAaaGr12Svr200Ca,
		Value: types.NewTypedValueFloat(sysAaaGr12Svr200CaValue)}
	const sysAaaGr12Svr200Addr = "/system/aaa/server-groups/server-group[12]/servers/server[200]/address"
	const sysAaaGr12Svr200AddrValue = "192.168.0.5"
	val17 := types.PathValue{Path: sysAaaGr12Svr200Addr,
		Value: types.NewTypedValueString(sysAaaGr12Svr200AddrValue)}
	jsonPathValues := []*types.PathValue{&val01, &val02, &val03, &val04,
		&val05, &val06, &val07, &val08, &val09, &val10, &val11, &val12, &val13,
		&val14, &val15, &val16, &val17}

	correctedPathValues, err := CorrectJSONPaths("", jsonPathValues, readOnlyPaths, true)
	assert.NilError(t, err)

	for _, v := range correctedPathValues {
		fmt.Printf("%s %v\n", (*v).Path, v.String())
	}
	assert.Equal(t, len(correctedPathValues), 10)

	for _, correctedPathValue := range correctedPathValues {
		const ifEth1Mtu = "/interfaces/interface[name=eth1]/state/mtu"
		const ifEth1Desc = "/interfaces/interface[name=eth1]/state/description"
		const ifEth1Sub120IfIdx = "/interfaces/interface[name=eth1]/subinterfaces/subinterface[index=120]/state/ifindex"
		const ifEth1Sub120AdSt = "/interfaces/interface[name=eth1]/subinterfaces/subinterface[index=120]/state/admin-status"
		const sysAaaGrG1Srv7Ca = "/system/aaa/server-groups/server-group[name=g1]/servers/server[address=192.168.0.7]/state/connection-aborts"
		const sysAaaGrG2Srv4Ca = "/system/aaa/server-groups/server-group[name=g2]/servers/server[address=192.168.0.4]/state/connection-aborts"
		const sysAaaGrG2Srv5Ca = "/system/aaa/server-groups/server-group[name=g2]/servers/server[address=192.168.0.5]/state/connection-aborts"
		switch correctedPathValue.Path {
		case systemNtpAuthMismatchNoNs:
			assert.Equal(t, correctedPathValue.GetValue().GetType(), types.ValueType_UINT)
			assert.Equal(t, len(correctedPathValue.GetValue().GetTypeOpts()), 0)
			assert.Equal(t, (*types.TypedUint64)(correctedPathValue.GetValue()).Uint(), uint(systemNtpAuthMismatchValue))
		case systemNtpEnableAuth:
			assert.Equal(t, correctedPathValue.GetValue().GetType(), types.ValueType_BOOL)
			assert.Equal(t, len(correctedPathValue.GetValue().GetTypeOpts()), 0)
			assert.Equal(t, (*types.TypedBool)(correctedPathValue.GetValue()).Bool(), true)
		case systemNtpSourceAddr:
			assert.Equal(t, correctedPathValue.GetValue().GetType(), types.ValueType_STRING)
			assert.Equal(t, len(correctedPathValue.GetValue().GetTypeOpts()), 0)
			assert.Equal(t, correctedPathValue.GetValue().ValueToString(), systemNtpSourceAddrValue)
		case ifEth1Mtu:
			assert.Equal(t, correctedPathValue.GetValue().GetType(), types.ValueType_UINT)
			assert.Equal(t, len(correctedPathValue.GetValue().GetTypeOpts()), 0)
			assert.Equal(t, (*types.TypedUint64)(correctedPathValue.GetValue()).Uint(), uint(iface1MtuValue))
		case ifEth1Desc:
			assert.Equal(t, correctedPathValue.GetValue().GetType(), types.ValueType_STRING)
			assert.Equal(t, len(correctedPathValue.GetValue().GetTypeOpts()), 0)
			assert.Equal(t, correctedPathValue.GetValue().ValueToString(), iface1DescValue)
		case ifEth1Sub120IfIdx:
			assert.Equal(t, correctedPathValue.GetValue().GetType(), types.ValueType_UINT)
			assert.Equal(t, len(correctedPathValue.GetValue().GetTypeOpts()), 0)
			assert.Equal(t, (*types.TypedUint64)(correctedPathValue.GetValue()).Uint(), uint(iface1Sub15IfaceIdxValue))
		case ifEth1Sub120AdSt:
			assert.Equal(t, correctedPathValue.GetValue().GetType(), types.ValueType_STRING)
			assert.Equal(t, len(correctedPathValue.GetValue().GetTypeOpts()), 0)
			assert.Equal(t, correctedPathValue.GetValue().ValueToString(), iface1Sub15AdStValue)
		case sysAaaGrG1Srv7Ca:
			assert.Equal(t, correctedPathValue.GetValue().GetType(), types.ValueType_UINT)
			assert.Equal(t, len(correctedPathValue.GetValue().GetTypeOpts()), 0)
			assert.Equal(t, (*types.TypedUint64)(correctedPathValue.GetValue()).Uint(), uint(sysAaaGr10Svr199CaValue))
		case sysAaaGrG2Srv4Ca:
			assert.Equal(t, correctedPathValue.GetValue().GetType(), types.ValueType_UINT)
			assert.Equal(t, len(correctedPathValue.GetValue().GetTypeOpts()), 0)
			assert.Equal(t, (*types.TypedUint64)(correctedPathValue.GetValue()).Uint(), uint(sysAaaGr12Svr199CaValue))
		case sysAaaGrG2Srv5Ca:
			assert.Equal(t, correctedPathValue.GetValue().GetType(), types.ValueType_UINT)
			assert.Equal(t, len(correctedPathValue.GetValue().GetTypeOpts()), 0)
			assert.Equal(t, (*types.TypedUint64)(correctedPathValue.GetValue()).Uint(), uint(sysAaaGr12Svr200CaValue))
		default:
			t.Fatal("Unexpected path", correctedPathValue.Path)
		}
	}
}

func Test_correctJsonPathValues2(t *testing.T) {

	var modelPluginTest modelPluginTest

	ds1Schema, err := modelPluginTest.Schema()
	assert.NilError(t, err)
	assert.Equal(t, len(ds1Schema), 137)

	readOnlyPaths, _ := modelregistry.ExtractPaths(ds1Schema["Device"], yang.TSUnset, "", "")
	assert.Equal(t, len(readOnlyPaths), 37)

	// All values are taken from testdata/sample-openconfig.json and defined
	// here in the intermediate jsonToValues format
	sampleTree, err := ioutil.ReadFile("./testdata/sample-openconfig2.json")
	assert.NilError(t, err)

	values, err := store.DecomposeTree(sampleTree)
	assert.NilError(t, err)
	assert.Equal(t, len(values), 31)

	correctedPathValues, err := CorrectJSONPaths("", values, readOnlyPaths, true)
	assert.NilError(t, err)

	for _, v := range correctedPathValues {
		fmt.Printf("%s %v\n", (*v).Path, v.String())
	}
	assert.Equal(t, len(correctedPathValues), 24)

	for _, correctedPathValue := range correctedPathValues {
		switch correctedPathValue.Path {
		case
			"/system/openflow/controllers/controller[name=main]/connections/connection[aux-id=0]/state/source-interface",
			"/system/openflow/controllers/controller[name=main]/connections/connection[aux-id=0]/state/transport",
			"/system/openflow/controllers/controller[name=main]/connections/connection[aux-id=0]/state/address",
			"/system/openflow/controllers/controller[name=main]/connections/connection[aux-id=0]/state/aux-id",
			"/system/openflow/controllers/controller[name=main]/connections/connection[aux-id=0]/state/port",
			"/system/openflow/controllers/controller[name=main]/connections/connection[aux-id=1]/state/source-interface",
			"/system/openflow/controllers/controller[name=main]/connections/connection[aux-id=1]/state/transport",
			"/system/openflow/controllers/controller[name=main]/connections/connection[aux-id=1]/state/address",
			"/system/openflow/controllers/controller[name=main]/connections/connection[aux-id=1]/state/aux-id",
			"/system/openflow/controllers/controller[name=main]/connections/connection[aux-id=1]/state/port",
			"/system/openflow/controllers/controller[name=second]/connections/connection[aux-id=0]/state/source-interface",
			"/system/openflow/controllers/controller[name=second]/connections/connection[aux-id=0]/state/transport",
			"/system/openflow/controllers/controller[name=second]/connections/connection[aux-id=0]/state/address",
			"/system/openflow/controllers/controller[name=second]/connections/connection[aux-id=0]/state/aux-id",
			"/system/openflow/controllers/controller[name=second]/connections/connection[aux-id=0]/state/port",
			"/system/openflow/controllers/controller[name=second]/connections/connection[aux-id=1]/state/source-interface",
			"/system/openflow/controllers/controller[name=second]/connections/connection[aux-id=1]/state/transport",
			"/system/openflow/controllers/controller[name=second]/connections/connection[aux-id=1]/state/address",
			"/system/openflow/controllers/controller[name=second]/connections/connection[aux-id=1]/state/aux-id",
			"/system/openflow/controllers/controller[name=second]/connections/connection[aux-id=1]/state/port":
			assert.Equal(t, correctedPathValue.GetValue().GetType(), types.ValueType_STRING, correctedPathValue.Path)
			assert.Equal(t, len(correctedPathValue.GetValue().GetTypeOpts()), 0)
		case
			"/system/openflow/controllers/controller[name=main]/connections/connection[aux-id=0]/state/priority",
			"/system/openflow/controllers/controller[name=main]/connections/connection[aux-id=1]/state/priority",
			"/system/openflow/controllers/controller[name=second]/connections/connection[aux-id=0]/state/priority",
			"/system/openflow/controllers/controller[name=second]/connections/connection[aux-id=1]/state/priority":
			assert.Equal(t, correctedPathValue.GetValue().GetType(), types.ValueType_UINT, correctedPathValue.Path)
			assert.Equal(t, len(correctedPathValue.GetValue().GetTypeOpts()), 0)
		default:
			t.Fatal("Unexpected path", correctedPathValue.Path)
		}
	}
}

func Test_hasPrefixMultipleIdx(t *testing.T) {

	const a = "/interfaces/interface[*]/subinterfaces/subinterface[*]/state/mtu"
	const b = "/interfaces/interface[*]/subinterfaces/subinterface[*]/state"

	assert.Assert(t, hasPrefixMultipleIdx(a, b), "a should match b", a, b)

	const c = "/interfaces/interface[*]/state"
	assert.Assert(t, !hasPrefixMultipleIdx(a, c), "a should NOT match b", a, c)
}
