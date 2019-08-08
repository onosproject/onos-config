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
	ds1 "github.com/onosproject/onos-config/modelplugin/Devicesim-1.0.0/devicesim_1_0_0"
	"github.com/onosproject/onos-config/pkg/store/change"
	"github.com/openconfig/goyang/pkg/yang"
	"gotest.tools/assert"
	"testing"
)

func Test_SchemaDeviceSim(t *testing.T) {
	td2Schema, _ := ds1.UnzipSchema()
	readOnlyPathsDeviceSim1, readWritePathsDeviceSim1 :=
		ExtractPaths(td2Schema["Device"], yang.TSUnset, "", "")

	readOnlyPathsKeys := Paths(readOnlyPathsDeviceSim1)
	assert.Equal(t, len(readOnlyPathsKeys), 37)
	// Can be in any order
	for _, p := range readOnlyPathsKeys {
		switch p {
		case
			"/system/clock/state",
			"/components/component[name=*]/state",
			"/system/ntp/servers/server[address=*]/state",
			"/system/aaa/accounting/events/event[event-type=*]/state",
			"/system/ssh-server/state",
			"/system/aaa/server-groups/server-group[name=*]/servers/server[address=*]/state",
			"/system/aaa/authorization/state",
			"/components/component[name=*]/subcomponents/subcomponent[name=*]/state",
			"/components/component[name=*]/properties/property[name=*]/state",
			"/system/logging/console/selectors/selector[facility severity=*]/state",
			"/system/aaa/authentication/state",
			"/system/ntp/state",
			"/interfaces/interface[name=*]/hold-time/state",
			"/system/logging/remote-servers/remote-server[host=*]/selectors/selector[facility severity=*]/state",
			"/system/openflow/agent/state",
			"/system/aaa/server-groups/server-group[name=*]/servers/server[address=*]/tacacs/state",
			"/system/aaa/server-groups/server-group[name=*]/state",
			"/system/dns/state",
			"/system/aaa/authorization/events/event[event-type=*]/state",
			"/system/aaa/state",
			"/system/processes/process[pid=*]",
			"/system/aaa/authentication/admin-user/state",
			"/system/aaa/accounting/state",
			"/interfaces/interface[name=*]/subinterfaces/subinterface[index=*]/state",
			"/system/dns/servers/server[address=*]/state",
			"/system/logging/console/state",
			"/system/memory/state",
			"/system/aaa/server-groups/server-group[name=*]/servers/server[address=*]/radius/state",
			"/interfaces/interface[name=*]/state",
			"/system/state",
			"/system/telnet-server/state",
			"/system/openflow/controllers/controller[name=*]/connections/connection[aux-id=*]/state",
			"/system/dns/host-entries/host-entry[hostname=*]/state",
			"/system/openflow/controllers/controller[name=*]/state",
			"/system/aaa/authentication/users/user[username=*]/state",
			"/system/logging/remote-servers/remote-server[host=*]/state",
			"/system/ntp/ntp-keys/ntp-key[key-id=*]/state":

		default:
			t.Fatal("Unexpected readOnlyPath", p)
		}
	}

	////////////////////////////////////////////////////
	/// Read write paths
	////////////////////////////////////////////////////
	readWritePathsKeys := PathsRW(readWritePathsDeviceSim1)
	assert.Equal(t, len(readWritePathsKeys), 113)

	// Can be in any order
	for _, p := range readWritePathsKeys {
		switch p {
		case
			"/components/component[name=*]/config/name",
			"/system/logging/remote-servers/remote-server[host=*]/config/source-address",
			"/system/openflow/agent/config/backoff-interval",
			"/system/aaa/server-groups/server-group[name=*]/name",
			"/system/aaa/authentication/users/user[username=*]/username",
			"/system/logging/remote-servers/remote-server[host=*]/selectors/selector[facility severity=*]/config/facility",
			"/system/aaa/server-groups/server-group[name=*]/servers/server[address=*]/address",
			"/system/ntp/servers/server[address=*]/config/prefer",
			"/system/logging/console/selectors/selector[facility severity=*]/config/severity",
			"/system/openflow/controllers/controller[name=*]/connections/connection[aux-id=*]/config/address",
			"/system/aaa/authentication/users/user[username=*]/config/password",
			"/system/ntp/ntp-keys/ntp-key[key-id=*]/config/key-value",
			"/system/aaa/authentication/config/authentication-method",
			"/system/aaa/authorization/events/event[event-type=*]/event-type",
			"/system/config/domain-name",
			"/system/config/motd-banner",
			"/interfaces/interface[name=*]/name",
			"/system/ntp/servers/server[address=*]/config/address",
			"/system/openflow/controllers/controller[name=*]/connections/connection[aux-id=*]/config/priority",
			"/system/openflow/agent/config/failure-mode",
			"/system/openflow/controllers/controller[name=*]/connections/connection[aux-id=*]/config/port",
			"/components/component[name=*]/name",
			"/interfaces/interface[name=*]/hold-time/config/down",
			"/system/openflow/controllers/controller[name=*]/connections/connection[aux-id=*]/config/source-interface",
			"/system/aaa/authentication/users/user[username=*]/config/username",
			"/system/dns/host-entries/host-entry[hostname=*]/hostname",
			"/system/aaa/server-groups/server-group[name=*]/servers/server[address=*]/tacacs/config/secret-key",
			"/components/component[name=*]/properties/property[name=*]/config/value",
			"/interfaces/interface[name=*]/subinterfaces/subinterface[index=*]/config/enabled",
			"/system/aaa/accounting/events/event[event-type=*]/event-type",
			"/system/aaa/authentication/users/user[username=*]/config/password-hashed",
			"/interfaces/interface[name=*]/config/description",
			"/system/aaa/authentication/admin-user/config/admin-password-hashed",
			"/system/openflow/agent/config/inactivity-probe",
			"/system/aaa/accounting/events/event[event-type=*]/config/record",
			"/system/ntp/ntp-keys/ntp-key[key-id=*]/config/key-type",
			"/system/logging/remote-servers/remote-server[host=*]/config/remote-port",
			"/system/dns/host-entries/host-entry[hostname=*]/config/alias",
			"/system/ntp/config/enabled",
			"/system/ntp/servers/server[address=*]/address",
			"/system/aaa/authentication/admin-user/config/admin-password",
			"/system/ntp/servers/server[address=*]/config/association-type",
			"/system/openflow/controllers/controller[name=*]/connections/connection[aux-id=*]/config/transport",
			"/components/component[name=*]/properties/property[name=*]/config/name",
			"/interfaces/interface[name=*]/config/type",
			"/system/aaa/accounting/events/event[event-type=*]/config/event-type",
			"/system/ssh-server/config/enable",
			"/system/ssh-server/config/timeout",
			"/system/logging/console/selectors/selector[facility severity=*]/config/facility",
			"/interfaces/interface[name=*]/subinterfaces/subinterface[index=*]/config/description",
			"/interfaces/interface[name=*]/hold-time/config/up",
			"/system/dns/host-entries/host-entry[hostname=*]/config/hostname",
			"/system/aaa/authorization/config/authorization-method",
			"/system/openflow/agent/config/datapath-id",
			"/system/aaa/authorization/events/event[event-type=*]/config/event-type",
			"/system/aaa/server-groups/server-group[name=*]/servers/server[address=*]/tacacs/config/source-address",
			"/system/config/hostname",
			"/system/aaa/accounting/config/accounting-method",
			"/system/dns/host-entries/host-entry[hostname=*]/config/ipv4-address",
			"/system/logging/remote-servers/remote-server[host=*]/selectors/selector[facility severity=*]/severity",
			"/components/component[name=*]/subcomponents/subcomponent[name=*]/config/name",
			"/system/ntp/config/enable-ntp-auth",
			"/system/ntp/servers/server[address=*]/config/iburst",
			"/system/telnet-server/config/enable",
			"/components/component[name=*]/properties/property[name=*]/name",
			"/interfaces/interface[name=*]/config/name",
			"/system/ntp/ntp-keys/ntp-key[key-id=*]/config/key-id",
			"/system/logging/remote-servers/remote-server[host=*]/host",
			"/system/ssh-server/config/protocol-version",
			"/components/component[name=*]/subcomponents/subcomponent[name=*]/name",
			"/system/aaa/server-groups/server-group[name=*]/servers/server[address=*]/radius/config/auth-port",
			"/system/openflow/agent/config/max-backoff",
			"/system/ssh-server/config/rate-limit",
			"/system/ntp/ntp-keys/ntp-key[key-id=*]/key-id",
			"/system/clock/config/timezone-name",
			"/system/aaa/server-groups/server-group[name=*]/servers/server[address=*]/config/address",
			"/system/aaa/server-groups/server-group[name=*]/servers/server[address=*]/radius/config/source-address",
			"/system/aaa/server-groups/server-group[name=*]/servers/server[address=*]/radius/config/acct-port",
			"/system/dns/servers/server[address=*]/config/address",
			"/system/openflow/controllers/controller[name=*]/name",
			"/system/openflow/controllers/controller[name=*]/connections/connection[aux-id=*]/aux-id",
			"/interfaces/interface[name=*]/subinterfaces/subinterface[index=*]/index",
			"/system/aaa/server-groups/server-group[name=*]/servers/server[address=*]/config/name",
			"/system/telnet-server/config/rate-limit",
			"/system/aaa/server-groups/server-group[name=*]/servers/server[address=*]/tacacs/config/port",
			"/interfaces/interface[name=*]/config/enabled",
			"/system/telnet-server/config/timeout",
			"/system/aaa/server-groups/server-group[name=*]/config/name",
			"/system/aaa/authentication/users/user[username=*]/config/ssh-key",
			"/system/logging/remote-servers/remote-server[host=*]/config/host",
			"/system/aaa/server-groups/server-group[name=*]/servers/server[address=*]/radius/config/secret-key",
			"/system/logging/console/selectors/selector[facility severity=*]/severity",
			"/system/ntp/servers/server[address=*]/config/port",
			"/system/ntp/config/ntp-source-address",
			"/system/aaa/server-groups/server-group[name=*]/servers/server[address=*]/radius/config/retransmit-attempts",
			"/system/openflow/controllers/controller[name=*]/connections/connection[aux-id=*]/config/aux-id",
			"/system/aaa/server-groups/server-group[name=*]/config/type",
			"/system/dns/config/search",
			"/system/ntp/servers/server[address=*]/config/version",
			"/system/logging/remote-servers/remote-server[host=*]/selectors/selector[facility severity=*]/facility",
			"/system/ssh-server/config/session-limit",
			"/system/logging/console/selectors/selector[facility severity=*]/facility",
			"/system/dns/host-entries/host-entry[hostname=*]/config/ipv6-address",
			"/system/openflow/controllers/controller[name=*]/config/name",
			"/system/dns/servers/server[address=*]/address",
			"/system/dns/servers/server[address=*]/config/port",
			"/interfaces/interface[name=*]/config/mtu",
			"/interfaces/interface[name=*]/subinterfaces/subinterface[index=*]/config/index",
			"/system/config/login-banner",
			"/system/aaa/authentication/users/user[username=*]/config/role",
			"/system/telnet-server/config/session-limit",
			"/system/aaa/server-groups/server-group[name=*]/servers/server[address=*]/config/timeout",
			"/system/logging/remote-servers/remote-server[host=*]/selectors/selector[facility severity=*]/config/severity":
		default:
			t.Fatal("Unexpected readWritePath", p)
		}
	}

	logSvrSelFacName, logSvrSelFacNameOk := readWritePathsDeviceSim1["/system/logging/remote-servers/remote-server[host=*]/selectors/selector[facility severity=*]/config/facility"]
	assert.Assert(t, logSvrSelFacNameOk, "expected to get /system/logging/remote-servers/remote-server[host=*]/selectors/selector[facility severity=*]/config/facility")
	assert.Equal(t, logSvrSelFacName.ValueType, change.ValueTypeSTRING, "expected /system/logging/remote-servers/remote-server[host=*]/selectors/selector[facility severity=*]/config/facility to be STRING")

}
