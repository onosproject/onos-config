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

package modelregistry

import (
	"fmt"
	ds1 "github.com/onosproject/onos-config/modelplugin/Devicesim-1.0.0/devicesim_1_0_0"
	"github.com/openconfig/gnmi/proto/gnmi"
	"github.com/openconfig/goyang/pkg/yang"
	"github.com/openconfig/ygot/ygot"
	"gotest.tools/assert"
	"os"
	"testing"
)

type modelPluginTest string

const modelTypeTest = "TestModel"
const modelVersionTest = "0.0.1"
const moduleNameTest = "testmodel.so.1.0.0"

var modelData = []*gnmi.ModelData{
	{Name: "testmodel", Organization: "Open Networking Lab", Version: "2019-07-10"},
}

var (
	readOnlyPaths  ReadOnlyPathMap
	readWritePaths ReadWritePathMap
)

var ds1Schema map[string]*yang.Entry

func TestMain(m *testing.M) {
	var modelPluginTest modelPluginTest

	ds1Schema, _ = modelPluginTest.Schema()
	readOnlyPaths, readWritePaths = ExtractPaths(ds1Schema["Device"], yang.TSUnset, "", "")
	os.Exit(m.Run())
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

	assert.Equal(t, len(ds1Schema), 137)

	readOnlyPathsKeys := Paths(readOnlyPaths)
	assert.Equal(t, len(readOnlyPathsKeys), 37)
	// Can be in any order
	for _, p := range readOnlyPathsKeys {
		switch p {
		case
			"/components/component[name=*]/properties/property[name=*]/state",
			"/components/component[name=*]/state",
			"/components/component[name=*]/subcomponents/subcomponent[name=*]/state",
			"/interfaces/interface[name=*]/subinterfaces/subinterface[index=*]/state",
			"/interfaces/interface[name=*]/hold-time/state",
			"/interfaces/interface[name=*]/state",
			"/system/dns/host-entries/host-entry[hostname=*]/state",
			"/system/dns/servers/server[address=*]/state",
			"/system/dns/state",
			"/system/logging/console/state",
			"/system/logging/console/selectors/selector[facility severity=*]/state",
			"/system/logging/remote-servers/remote-server[host=*]/state",
			"/system/logging/remote-servers/remote-server[host=*]/selectors/selector[facility severity=*]/state",
			"/system/state",
			"/system/telnet-server/state",
			"/system/aaa/state",
			"/system/aaa/accounting/events/event[event-type=*]/state",
			"/system/aaa/accounting/state",
			"/system/aaa/authentication/admin-user/state",
			"/system/aaa/authentication/state",
			"/system/aaa/authentication/users/user[username=*]/state",
			"/system/aaa/authorization/events/event[event-type=*]/state",
			"/system/aaa/authorization/state",
			"/system/aaa/server-groups/server-group[name=*]/servers/server[address=*]/radius/state",
			"/system/aaa/server-groups/server-group[name=*]/servers/server[address=*]/state",
			"/system/aaa/server-groups/server-group[name=*]/servers/server[address=*]/tacacs/state",
			"/system/aaa/server-groups/server-group[name=*]/state",
			"/system/clock/state",
			"/system/openflow/agent/state",
			"/system/openflow/controllers/controller[name=*]/connections/connection[aux-id=*]/state",
			"/system/openflow/controllers/controller[name=*]/state",
			"/system/processes/process[pid=*]",
			"/system/ssh-server/state",
			"/system/memory/state",
			"/system/ntp/ntp-keys/ntp-key[key-id=*]/state",
			"/system/ntp/servers/server[address=*]/state",
			"/system/ntp/state":

		default:
			t.Fatal("Unexpected readOnlyPath", p)
		}
	}
}
