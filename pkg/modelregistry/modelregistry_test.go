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

/**
 * Please refer to the tree view of DeviceSim at
 * ../../modelplugin/Devicesim-1.0.0/devicesim-1.0.0.tree
 * to see how the YANG files are rendered together in the model plugin
 */
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

func Test_State(t *testing.T) {

	k := "/components/component[name=*]/properties/property[name=*]/state"
	for p := range readOnlyPaths[k] {
		switch p {
		case "/configurable", "/name", "/value":
		default:
			t.Fatalf("Unexpected readOnlyPath sub path %s for %s", p, k)
		}
	}
}

func Test_Component(t *testing.T) {

	k := "/components/component[name=*]/state"
	for p := range readOnlyPaths[k] {
		switch p {
		case "/description", "/id", "/mfg-name", "/name", "/part-no", "/serial-no", "/type", "/version",
			"/temperature/instant", "/temperature/avg", "/temperature/min", "/temperature/max":
		default:
			t.Fatalf("Unexpected readOnlyPath sub path %s for %s", p, k)
		}
	}
}

func Test_SubComponent(t *testing.T) {

	k := "/components/component[name=*]/subcomponents/subcomponent[name=*]/state"
	for p := range readOnlyPaths[k] {
		switch p {
		case "/name":
		default:
			t.Fatalf("Unexpected readOnlyPath sub path %s for %s", p, k)
		}
	}
}

func Test_hold_time(t *testing.T) {

	k := "/interfaces/interface[name=*]/hold-time/state"
	for p := range readOnlyPaths[k] {
		switch p {
		case "/down", "/up":
		default:
			t.Fatalf("Unexpected readOnlyPath sub path %s for %s", p, k)
		}
	}
}

func Test_SubInterfaces(t *testing.T) {

	k := "/interfaces/interface[name=*]/subinterfaces/subinterface[index=*]/state"
	for p := range readOnlyPaths[k] {
		switch p {
		case "/admin-status", "/description", "/enabled", "/ifindex", "/index",
			"/last-change", "/name", "/oper-status", "/counters/in-octets",
			"/counters/in-unicast-pkts", "/counters/in-broadcast-pkts",
			"/counters/in-multicast-pkts", "/counters/in-discards", "/counters/in-errors",
			"/counters/in-unknown-protos", "/counters/in-fcs-errors",
			"/counters/out-octets", "/counters/out-unicast-pkts", "/counters/out-broadcast-pkts",
			"/counters/out-multicast-pkts", "/counters/out-discards", "/counters/out-errors",
			"/counters/carrier-transitions", "/counters/last-clear":
		default:
			t.Fatalf("Unexpected readOnlyPath sub path %s for %s", p, k)
		}
	}
}

func Test_InterfaceName(t *testing.T) {

	k := "/interfaces/interface[name=*]/state"
	for p := range readOnlyPaths[k] {
		switch p {
		case "/admin-status", "/description", "/enabled", "/hardware-port", "/ifindex",
			"/last-change", "/mtu", "/name", "/oper-status", "/type", "/counters/in-octets",
			"/counters/in-unicast-pkts", "/counters/in-broadcast-pkts",
			"/counters/in-multicast-pkts", "/counters/in-discards", "/counters/in-errors",
			"/counters/in-unknown-protos", "/counters/in-fcs-errors",
			"/counters/out-octets", "/counters/out-unicast-pkts", "/counters/out-broadcast-pkts",
			"/counters/out-multicast-pkts", "/counters/out-discards", "/counters/out-errors",
			"/counters/carrier-transitions", "/counters/last-clear":
		default:
			t.Fatalf("Unexpected readOnlyPath sub path %s for %s", p, k)
		}
	}
}

func Test_DnsHost(t *testing.T) {

	k := "/system/dns/host-entries/host-entry[hostname=*]/state"
	for p := range readOnlyPaths[k] {
		switch p {
		case "/hostname", "/alias", "/ipv4-address", "/ipv6-address":
		default:
			t.Fatalf("Unexpected readOnlyPath sub path %s for %s", p, k)
		}
	}
}

func Test_DnsServer(t *testing.T) {

	k := "/system/dns/servers/server[address=*]/state"
	for p := range readOnlyPaths[k] {
		switch p {
		case "/address", "/port":
		default:
			t.Fatalf("Unexpected readOnlyPath sub path %s for %s", p, k)
		}
	}
}

func Test_Dns(t *testing.T) {

	k := "/system/dns/state"
	for p := range readOnlyPaths[k] {
		switch p {
		case "/search":
		default:
			t.Fatalf("Unexpected readOnlyPath sub path %s for %s", p, k)
		}
	}
}

func Test_LoggingConsole(t *testing.T) {

	k := "/system/logging/console/state"
	if len(readOnlyPaths[k]) != 0 {
		t.Fatalf("Unexpected readOnlyPath sub path %v for %s", readOnlyPaths, k)
	}
}

func Test_LoggingSelector(t *testing.T) {

	k := "/system/logging/console/selectors/selector[facility severity=*]/state"
	for p := range readOnlyPaths[k] {
		switch p {
		case "/facility", "/severity":
		default:
			t.Fatalf("Unexpected readOnlyPath sub path %s for %s", p, k)
		}
	}
}

func Test_LoggingServer(t *testing.T) {

	k := "/system/logging/remote-servers/remote-server[host=*]/state"
	for p := range readOnlyPaths[k] {
		switch p {
		case "/host", "/remote-port", "/source-address":
		default:
			t.Fatalf("Unexpected readOnlyPath sub path %s for %s", p, k)
		}
	}
}

func Test_Logging(t *testing.T) {

	k := "/system/logging/remote-servers/remote-server[host=*]/selectors/selector[facility severity=*]/state"
	for p := range readOnlyPaths[k] {
		switch p {
		case "/facility", "/severity":
		default:
			t.Fatalf("Unexpected readOnlyPath sub path %s for %s", p, k)
		}
	}
}

func Test_SysState(t *testing.T) {

	k := "/system/state"
	for p := range readOnlyPaths[k] {
		switch p {
		case "/boot-time", "/current-datetime", "/domain-name", "/hostname", "/login-banner", "/motd-banner":
		default:
			t.Fatalf("Unexpected readOnlyPath sub path %s for %s", p, k)
		}
	}
}

func Test_Telnet(t *testing.T) {

	k := "/system/telnet-server/state"
	for p := range readOnlyPaths[k] {
		switch p {
		case "/enable", "/rate-limit", "/session-limit", "/timeout":
		default:
			t.Fatalf("Unexpected readOnlyPath sub path %s for %s", p, k)
		}
	}
}

func Test_AAA(t *testing.T) {

	k := "/system/aaa/state"
	if len(readOnlyPaths[k]) != 0 {
		t.Fatalf("Unexpected readOnlyPath sub path %v for %s", readOnlyPaths, k)
	}
}

func Test_AccountingEvents(t *testing.T) {

	k := "/system/aaa/accounting/events/event[event-type=*]/state"
	for p := range readOnlyPaths[k] {
		switch p {
		case "/event-type", "/record":
		default:
			t.Fatalf("Unexpected readOnlyPath sub path %s for %s", p, k)
		}
	}
}

func Test_Accounting(t *testing.T) {

	k := "/system/aaa/accounting/state"
	for p := range readOnlyPaths[k] {
		switch p {
		case "/accounting-method":
		default:
			t.Fatalf("Unexpected readOnlyPath sub path %s for %s", p, k)
		}
	}
}

func Test_AuthAdmin(t *testing.T) {

	k := "/system/aaa/authentication/admin-user/state"
	for p := range readOnlyPaths[k] {
		switch p {
		case "/admin-password", "/admin-password-hashed", "/admin-username":
		default:
			t.Fatalf("Unexpected readOnlyPath sub path %s for %s", p, k)
		}
	}
}

func Test_AuthState(t *testing.T) {

	k := "/system/aaa/authentication/state"
	for p := range readOnlyPaths[k] {
		switch p {
		case "/authentication-method":
		default:
			t.Fatalf("Unexpected readOnlyPath sub path %s for %s", p, k)
		}
	}
}

func Test_AuthUser(t *testing.T) {

	k := "/system/aaa/authentication/users/user[username=*]/state"
	for p := range readOnlyPaths[k] {
		switch p {
		case "/password", "/password-hashed", "/role", "/ssh-key", "/username":
		default:
			t.Fatalf("Unexpected readOnlyPath sub path %s for %s", p, k)
		}
	}
}

func Test_AuthEvent(t *testing.T) {

	k := "/system/aaa/authorization/events/event[event-type=*]/state"
	for p := range readOnlyPaths[k] {
		switch p {
		case "/event-type":
		default:
			t.Fatalf("Unexpected readOnlyPath sub path %s for %s", p, k)
		}
	}
}

func Test_Auth(t *testing.T) {

	k := "/system/aaa/authorization/state"
	for p := range readOnlyPaths[k] {
		switch p {
		case "/authorization-method":
		default:
			t.Fatalf("Unexpected readOnlyPath sub path %s for %s", p, k)
		}
	}
}

func Test_Radius(t *testing.T) {

	k := "/system/aaa/server-groups/server-group[name=*]/servers/server[address=*]/radius/state"
	for p := range readOnlyPaths[k] {
		switch p {
		case "/acct-port", "/auth-port", "/retransmit-attempts", "/secret-key", "/source-address",
			"/counters/retried-access-requests", "/counters/access-accepts",
			"/counters/access-rejects", "/counters/timeout-access-requests":
		default:
			t.Fatalf("Unexpected readOnlyPath sub path %s for %s", p, k)
		}
	}
}

func Test_AAAServerGroup(t *testing.T) {

	k := "/system/aaa/server-groups/server-group[name=*]/servers/server[address=*]/state"
	for p := range readOnlyPaths[k] {
		switch p {
		case "/address", "/connection-aborts", "/connection-closes", "/connection-failures", "/connection-opens",
			"/connection-timeouts", "/errors-received", "/messages-received", "/messages-sent", "/name", "/timeout":
		default:
			t.Fatalf("Unexpected readOnlyPath sub path %s for %s", p, k)
		}
	}
}

func Test_AAAtacacs(t *testing.T) {

	k := "/system/aaa/server-groups/server-group[name=*]/servers/server[address=*]/tacacs/state"
	for p := range readOnlyPaths[k] {
		switch p {
		case "/port", "/secret-key", "/source-address":
		default:
			t.Fatalf("Unexpected readOnlyPath sub path %s for %s", p, k)
		}
	}
}

func Test_AAAServer(t *testing.T) {

	k := "/system/aaa/server-groups/server-group[name=*]/state"
	for p := range readOnlyPaths[k] {
		switch p {
		case "/name", "/type":
		default:
			t.Fatalf("Unexpected readOnlyPath sub path %s for %s", p, k)
		}
	}
}

func Test_Clock(t *testing.T) {

	k := "/system/clock/state"
	for p := range readOnlyPaths[k] {
		switch p {
		case "/timezone-name":
		default:
			t.Fatalf("Unexpected readOnlyPath sub path %s for %s", p, k)
		}
	}
}

func Test_OfAgent(t *testing.T) {

	k := "/system/openflow/agent/state"
	for p := range readOnlyPaths[k] {
		switch p {
		case "/backoff-interval", "/datapath-id", "/failure-mode", "/inactivity-probe", "/max-backoff":
		default:
			t.Fatalf("Unexpected readOnlyPath sub path %s for %s", p, k)
		}
	}
}

func Test_OFControllerConnection(t *testing.T) {

	k := "/system/openflow/controllers/controller[name=*]/connections/connection[aux-id=*]/state"
	for p := range readOnlyPaths[k] {
		switch p {
		case "/address", "/aux-id", "/connected", "/port", "/priority", "/source-interface", "/transport":
		default:
			t.Fatalf("Unexpected readOnlyPath sub path %s for %s", p, k)
		}
	}
}

func Test_OfControllers(t *testing.T) {

	k := "/system/openflow/controllers/controller[name=*]/state"
	for p := range readOnlyPaths[k] {
		switch p {
		case "/name":
		default:
			t.Fatalf("Unexpected readOnlyPath sub path %s for %s", p, k)
		}
	}
}

func Test_ProcMonProcess(t *testing.T) {

	k := "/system/processes/process[pid=*]/state"
	for p := range readOnlyPaths[k] {
		switch p {
		case "/cpu-usage-system", "/cpu-usage-user", "/cpu-utilization", "/memory-usage", "/memory-utilization",
			"/name", "/pid", "/start-time", "/uptime":
		default:
			t.Fatalf("Unexpected readOnlyPath sub path %s for %s", p, k)
		}
	}
}

func Test_SshServer(t *testing.T) {

	k := "/system/ssh-server/state"
	for p := range readOnlyPaths[k] {
		switch p {
		case "/enable", "/protocol-version", "/rate-limit", "/session-limit", "/timeout":
		default:
			t.Fatalf("Unexpected readOnlyPath sub path %s for %s", p, k)
		}
	}
}

func Test_SystemMemory(t *testing.T) {

	k := "/system/memory/state"
	for p := range readOnlyPaths[k] {
		switch p {
		case "/physical", "/reserved":
		default:
			t.Fatalf("Unexpected readOnlyPath sub path %s for %s", p, k)
		}
	}
}

func Test_NtpKeys(t *testing.T) {

	k := "/system/ntp/ntp-keys/ntp-key[key-id=*]/state"
	for p := range readOnlyPaths[k] {
		switch p {
		case "/key-id", "/key-type", "/key-value":
		default:
			t.Fatalf("Unexpected readOnlyPath sub path %s for %s", p, k)
		}
	}
}

func Test_NtpServer(t *testing.T) {

	k := "/system/ntp/servers/server[address=*]/state"
	for p, v := range readOnlyPaths[k] {
		switch p {
		case "/address", "/association-type", "/port":
			assert.Equal(t, int(v), 1, "Unexpected type %i for %s", v, p)
		case "/iburst", "/prefer":
			assert.Equal(t, int(v), 4, "Unexpected type %i for %s", v, p)
		case "/offset", "/poll-interval", "/root-delay", "/root-dispersion", "/stratum", "/version":
			assert.Equal(t, int(v), 3, "Unexpected type %i for %s", v, p)
		default:
			t.Fatalf("Unexpected readOnlyPath sub path %s for %s", p, k)
		}
	}
}

func Test_Ntp(t *testing.T) {

	k := "/system/ntp/state"
	for p := range readOnlyPaths[k] {
		switch p {
		case "/auth-mismatch", "/enable-ntp-auth", "/enabled", "/ntp-source-address":
		default:
			t.Fatalf("Unexpected readOnlyPath sub path %s for %s", p, k)
		}
	}
}

func Test_RemovePathIndices(t *testing.T) {
	assert.Equal(t,
		RemovePathIndices("/cont1b-state"),
		"/cont1b-state")

	assert.Equal(t,
		RemovePathIndices("/cont1b-state/list2b[index=test1]/index"),
		"/cont1b-state/list2b[index=*]/index")

	assert.Equal(t,
		RemovePathIndices("/cont1b-state/list2b[index=test1,test2]/index"),
		"/cont1b-state/list2b[index=*]/index")

	assert.Equal(t,
		RemovePathIndices("/cont1b-state/list2b[index=test1]/index/t3[name=5]"),
		"/cont1b-state/list2b[index=*]/index/t3[name=*]")

}

func Test_formatName1(t *testing.T) {
	dirEntry1 := yang.Entry{
		Name:   "testname",
		Prefix: &yang.Value{Name: "pfx1"},
	}
	assert.Equal(t, formatName(&dirEntry1, false, "/testpath/testpath2", ""), "/testpath/testpath2/testname")
}

func Test_formatName2(t *testing.T) {
	dirEntry1 := yang.Entry{
		Name:   "testname",
		Key:    "name",
		Prefix: &yang.Value{Name: "pfx1"},
	}
	assert.Equal(t, formatName(&dirEntry1, true, "/testpath/testpath2", ""), "/testpath/testpath2/testname[name=*]")
}
