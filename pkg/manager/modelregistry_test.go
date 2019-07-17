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

package manager

import (
	"fmt"
	ds1 "github.com/onosproject/onos-config/modelplugin/Devicesim-1.0.0/devicesim_1_0_0"
	"github.com/openconfig/gnmi/proto/gnmi"
	"github.com/openconfig/goyang/pkg/yang"
	"github.com/openconfig/ygot/ygot"
	"gotest.tools/assert"
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

// UnmarshalConfigValues uses the `generated.go` of the TestDevice1 plugin module
func (m modelPluginTest) UnmarshalConfigValues(jsonTree []byte) (*ygot.ValidatedGoStruct, error) {
	device := &ds1.Device{}
	vgs := ygot.ValidatedGoStruct(device)

	if err := ds1.Unmarshal([]byte(jsonTree), device); err != nil {
		return nil, err
	}

	return &vgs, nil
}

// Validate uses the `generated.go` of the TestDevice1 plugin module
func (m modelPluginTest) Validate(ygotModel *ygot.ValidatedGoStruct, opts ...ygot.ValidationOption) error {
	deviceDeref := *ygotModel
	device, ok := deviceDeref.(*ds1.Device)
	if !ok {
		return fmt.Errorf("unable to convert model in to testdevice_1_0_0")
	}
	return device.Validate()
}

// Schema uses the `generated.go` of the TestDevice1 plugin module
func (m modelPluginTest) Schema() (map[string]*yang.Entry, error) {
	return ds1.UnzipSchema()
}

func Test_CastModelPlugin(t *testing.T) {
	var modelPluginTest modelPluginTest
	mpt := interface{}(modelPluginTest)

	modelPlugin, ok := mpt.(ModelPlugin)
	assert.Assert(t, ok, "Testing cast of model plugin")
	name, version, _, _ := modelPlugin.ModelData()
	assert.Equal(t, name, modelTypeTest)
	assert.Equal(t, version, modelVersionTest)

}

func Test_Schema(t *testing.T) {
	var modelPluginTest modelPluginTest

	ds1Schema, err := modelPluginTest.Schema()
	assert.NilError(t, err)
	assert.Equal(t, len(ds1Schema), 137)

	readOnlyPaths := extractReadOnlyPaths(ds1Schema["Device"], yang.TSUnset, "", "")
	assert.Equal(t, len(readOnlyPaths), 37)
	// Can be in any order
	for _, p := range readOnlyPaths {
		switch p {
		case
			"/openconfig-platform:components/component[name=*]/properties/property[name=*]/state",
			"/openconfig-platform:components/component[name=*]/state",
			"/openconfig-platform:components/component[name=*]/subcomponents/subcomponent[name=*]/state",
			"/openconfig-interfaces:interfaces/interface[name=*]/subinterfaces/subinterface[index=*]/state",
			"/openconfig-interfaces:interfaces/interface[name=*]/hold-time/state",
			"/openconfig-interfaces:interfaces/interface[name=*]/state",
			"/openconfig-system:system/dns/host-entries/host-entry[hostname=*]/state",
			"/openconfig-system:system/dns/servers/server[address=*]/state",
			"/openconfig-system:system/dns/state",
			"/openconfig-system:system/openconfig-system-logging:logging/console/state",
			"/openconfig-system:system/openconfig-system-logging:logging/console/selectors/selector[facility severity=*]/state",
			"/openconfig-system:system/openconfig-system-logging:logging/remote-servers/remote-server[host=*]/state",
			"/openconfig-system:system/openconfig-system-logging:logging/remote-servers/remote-server[host=*]/selectors/selector[facility severity=*]/state",
			"/openconfig-system:system/state",
			"/openconfig-system:system/openconfig-system-terminal:telnet-server/state",
			"/openconfig-system:system/openconfig-aaa:aaa/state",
			"/openconfig-system:system/openconfig-aaa:aaa/accounting/events/event[event-type=*]/state",
			"/openconfig-system:system/openconfig-aaa:aaa/accounting/state",
			"/openconfig-system:system/openconfig-aaa:aaa/authentication/admin-user/state",
			"/openconfig-system:system/openconfig-aaa:aaa/authentication/state",
			"/openconfig-system:system/openconfig-aaa:aaa/authentication/users/user[username=*]/state",
			"/openconfig-system:system/openconfig-aaa:aaa/authorization/events/event[event-type=*]/state",
			"/openconfig-system:system/openconfig-aaa:aaa/authorization/state",
			"/openconfig-system:system/openconfig-aaa:aaa/server-groups/server-group[name=*]/servers/server[address=*]/radius/state",
			"/openconfig-system:system/openconfig-aaa:aaa/server-groups/server-group[name=*]/servers/server[address=*]/state",
			"/openconfig-system:system/openconfig-aaa:aaa/server-groups/server-group[name=*]/servers/server[address=*]/tacacs/state",
			"/openconfig-system:system/openconfig-aaa:aaa/server-groups/server-group[name=*]/state",
			"/openconfig-system:system/clock/state",
			"/openconfig-system:system/openconfig-openflow:openflow/agent/state",
			"/openconfig-system:system/openconfig-openflow:openflow/controllers/controller[name=*]/connections/connection[aux-id=*]/state",
			"/openconfig-system:system/openconfig-openflow:openflow/controllers/controller[name=*]/state",
			"/openconfig-system:system/openconfig-procmon:processes/process[pid=*]",
			"/openconfig-system:system/openconfig-system-terminal:ssh-server/state",
			"/openconfig-system:system/memory/state",
			"/openconfig-system:system/ntp/ntp-keys/ntp-key[key-id=*]/state",
			"/openconfig-system:system/ntp/servers/server[address=*]/state",
			"/openconfig-system:system/ntp/state":

		default:
			t.Fatal("Unexpected readOnlyPath", p)
		}
	}
}

func Test_RemovePathIndices(t *testing.T) {
	assert.Equal(t,
		RemovePathIndices("/test1:cont1b-state"),
		"/test1:cont1b-state")

	assert.Equal(t,
		RemovePathIndices("/test1:cont1b-state/list2b[index=test1]/index"),
		"/test1:cont1b-state/list2b[index=*]/index")

	assert.Equal(t,
		RemovePathIndices("/test1:cont1b-state/list2b[index=test1,test2]/index"),
		"/test1:cont1b-state/list2b[index=*]/index")

	assert.Equal(t,
		RemovePathIndices("/test1:cont1b-state/list2b[index=test1]/index/t3[name=5]"),
		"/test1:cont1b-state/list2b[index=*]/index/t3[name=*]")

}

func Test_formatName1(t *testing.T) {
	dirEntry1 := yang.Entry{
		Name:   "testname",
		Prefix: &yang.Value{Name: "pfx1"},
	}
	assert.Equal(t, formatName(&dirEntry1, false, "pfx1", "/pfx1:testpath/testpath2"), "/pfx1:testpath/testpath2/testname")
}

func Test_formatName2(t *testing.T) {
	dirEntry1 := yang.Entry{
		Name:   "testname",
		Key:    "name",
		Prefix: &yang.Value{Name: "pfx1"},
	}
	assert.Equal(t, formatName(&dirEntry1, true, "pfx1", "/pfx1:testpath/testpath2"), "/pfx1:testpath/testpath2/testname[name=*]")
}
