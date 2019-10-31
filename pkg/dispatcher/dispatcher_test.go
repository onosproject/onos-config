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

package dispatcher

import (
	"github.com/onosproject/onos-config/pkg/events"
	devicechangetypes "github.com/onosproject/onos-config/pkg/types/change/device"
	devicetopo "github.com/onosproject/onos-topo/pkg/northbound/device"
	"gotest.tools/assert"
	is "gotest.tools/assert/cmp"
	log "k8s.io/klog"
	"os"
	"strings"
	"testing"
	"time"
)

var (
	optStateChannel           chan events.OperationalStateEvent
	device1, device2, device3 devicetopo.Device
	err                       error
)

const (
	opStateTest = "opStateListener"
)

func setUp() *Dispatcher {
	d := NewDispatcher()
	optStateChannel, err = d.RegisterOpState(opStateTest)
	return d
}

func tearDown(t *testing.T, d *Dispatcher) {
	d.UnregisterOperationalState(opStateTest)
}

func TestMain(m *testing.M) {
	log.SetOutput(os.Stdout)

	device1 = devicetopo.Device{ID: "localhost-1", Address: "localhost:10161"}
	device2 = devicetopo.Device{ID: "localhost-2", Address: "localhost:10162"}
	device3 = devicetopo.Device{ID: "localhost-3", Address: "localhost:10163"}

	os.Exit(m.Run())
}

func Test_getListeners(t *testing.T) {
	d := setUp()
	listeners := d.GetListeners()

	listenerStr := strings.Join(listeners, ",")
	assert.Assert(t, is.Contains(listenerStr, opStateTest), "Expected to find %s in list. Got %s", opStateTest, listeners)
	tearDown(t, d)
}

func Test_register(t *testing.T) {
	d := setUp()

	opStateChannel2, err := d.RegisterOpState("opStateTest2")
	assert.NilError(t, err, "Unexpected error when registering opStatetest 2 %s", err)

	var opChannelIf interface{} = opStateChannel2
	chanTypeOp, ok := opChannelIf.(chan events.OperationalStateEvent)
	assert.Assert(t, ok, "Unexpected channel type when registering device %v", chanTypeOp)

	tearDown(t, d)
}

func Test_listen_operational(t *testing.T) {
	d := NewDispatcher()
	ch, err := d.RegisterOpState("nbiOpState")
	assert.NilError(t, err, "Unexpected error when registering nbi %s", err)
	assert.Equal(t, 1, len(d.GetListeners()), "One OpState listener expected")
	// Start the main listener system
	go testSyncOpState(ch, func(path string, action events.EventAction) {
		assert.Equal(t, path, "testpath")
		assert.Equal(t, action, events.EventItemUpdated)
	})
	// Create a channel on which to send these
	opStateCh := make(chan events.OperationalStateEvent, 10)
	go d.ListenOperationalState(opStateCh)
	// Send down some changes
	event := events.NewOperationalStateEvent("foobar", "testpath",
		devicechangetypes.NewTypedValueString("testValue"), events.EventItemUpdated)
	opStateCh <- event

	// Wait for the changes to get distributed
	time.Sleep(time.Second)

	close(opStateCh)

	d.UnregisterOperationalState("nbiOpState")
}

func testSyncOpState(testChan <-chan events.OperationalStateEvent, callback func(string, events.EventAction)) {
	log.Info("Listen for config changes for Test")

	for opStateChange := range testChan {
		callback(opStateChange.Path(), opStateChange.ItemAction())
		log.Info("OperationalState change for Test ", opStateChange)
	}
}
