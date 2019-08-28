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
	"encoding/base64"
	"github.com/onosproject/onos-config/pkg/events"
	"github.com/onosproject/onos-config/pkg/store"
	"github.com/onosproject/onos-topo/pkg/northbound/device"
	"gotest.tools/assert"
	is "gotest.tools/assert/cmp"
	log "k8s.io/klog"
	"os"
	"strings"
	"testing"
	"time"
)

var (
	device1Channel, device2Channel, device3Channel chan events.ConfigEvent
	respChannel1, respChannel2, respChannel3       chan events.DeviceResponse
	optStateChannel                                chan events.OperationalStateEvent
	device1, device2, device3                      device.Device
	err                                            error
)

const (
	configStoreDefaultFileName = "testdata/configStore-sample.json"
	changeStoreDefaultFileName = "testdata/changeStore-sample.json"
	opStateTest                = "opStateListener"
)

var (
	configStoreTest store.ConfigurationStore
	changeStoreTest store.ChangeStore
)

func setUp() *Dispatcher {
	d := NewDispatcher()
	device1Channel, respChannel1, err = d.RegisterDevice(device1.ID)
	device2Channel, respChannel2, err = d.RegisterDevice(device2.ID)
	device3Channel, respChannel3, err = d.RegisterDevice(device3.ID)
	optStateChannel, err = d.RegisterOpState(opStateTest)
	return d
}

func tearDown(t *testing.T, d *Dispatcher) {
	assert.NilError(t, d.UnregisterDevice(device1.ID))
	assert.NilError(t, d.UnregisterDevice(device2.ID))
	assert.NilError(t, d.UnregisterDevice(device3.ID))
	d.UnregisterOperationalState(opStateTest)
}

func TestMain(m *testing.M) {
	log.SetOutput(os.Stdout)

	device1 = device.Device{ID: "localhost-1", Address: "localhost:10161"}
	device2 = device.Device{ID: "localhost-2", Address: "localhost:10162"}
	device3 = device.Device{ID: "localhost-3", Address: "localhost:10163"}

	configStoreTest, err = store.LoadConfigStore(configStoreDefaultFileName)
	if err != nil {
		wd, _ := os.Getwd()
		log.Warning("Cannot load config store ", err, wd)
		os.Exit(1)
	}
	log.Info("Configuration store loaded from ", configStoreDefaultFileName)

	changeStoreTest, err = store.LoadChangeStore(changeStoreDefaultFileName)
	if err != nil {
		log.Error("Cannot load change store ", err)
		os.Exit(1)
	}

	os.Exit(m.Run())
}

func Test_getListeners(t *testing.T) {
	d := setUp()
	listeners := d.GetListeners()
	assert.Equal(t, len(listeners), 7, "Expected to find 7 listeners in list. Got %d", len(listeners))
	assert.Assert(t, d.HasListener(device1.ID), "Device 1 not registered")

	listenerStr := strings.Join(listeners, ",")
	// Could be in any order
	assert.Assert(t, is.Contains(listenerStr, "localhost-1"), "Expected to find device1 in list. Got %s", listeners)
	assert.Assert(t, is.Contains(listenerStr, "localhost-2"), "Expected to find device2 in list. Got %s", listeners)
	assert.Assert(t, is.Contains(listenerStr, "localhost-3"), "Expected to find device3 in list. Got %s", listeners)
	assert.Assert(t, is.Contains(listenerStr, opStateTest), "Expected to find %s in list. Got %s", opStateTest, listeners)
	tearDown(t, d)
}

func Test_register(t *testing.T) {
	d := setUp()
	device4Channel, respChannel4, err := d.RegisterDevice(device.ID("device4"))
	assert.NilError(t, err, "Unexpected error when registering device %s", err)

	opStateChannel2, err := d.RegisterOpState("opStateTest2")
	assert.NilError(t, err, "Unexpected error when registering opStatetest 2 %s", err)

	var deviceChannelIf interface{} = device4Channel
	chanType, ok := deviceChannelIf.(chan events.ConfigEvent)
	assert.Assert(t, ok, "Unexpected channel type when registering device %v", chanType)
	var respChannelIf interface{} = respChannel4
	chanTypeResp, ok := respChannelIf.(chan events.DeviceResponse)
	assert.Assert(t, ok, "Unexpected channel type when registering device %v", chanTypeResp)
	_ = d.UnregisterDevice(device.ID("device4"))

	var opChannelIf interface{} = opStateChannel2
	chanTypeOp, ok := opChannelIf.(chan events.OperationalStateEvent)
	assert.Assert(t, ok, "Unexpected channel type when registering device %v", chanTypeOp)
	_ = d.UnregisterDevice(device.ID("opStateTest2"))

	tearDown(t, d)
}

func Test_unregister(t *testing.T) {
	d := setUp()
	err1 := d.UnregisterDevice(device.ID("device5"))

	assert.Assert(t, err1 != nil, "Unexpected lack of error when unregistering non existent device")
	assert.Assert(t, is.Contains(err1.Error(), "had not been registered"),
		"Unexpected error text when unregistering non existent device %s", err1)

	_, _, _ = d.RegisterDevice(device.ID("device6"))
	_, _, _ = d.RegisterDevice(device.ID("device7"))

	err2 := d.UnregisterDevice(device.ID("device6"))
	assert.NilError(t, err2, "Unexpected error when unregistering device6 %s", err2)
	_ = d.UnregisterDevice(device.ID("device7"))

	tearDown(t, d)
}

func Test_listen_device(t *testing.T) {
	d := setUp()
	defer tearDown(t, d)
	changes := make(map[string]bool)
	// Start the main listener system
	go testSync(device1Channel, changes)
	changesChannel := make(chan events.ConfigEvent, 10)
	go d.Listen(changesChannel)
	changeID := []byte("test")
	// Send down some changes
	for i := 1; i < 3; i++ {
		event := events.CreateConfigEvent(device1.Address, changeID, true)
		changesChannel <- event
	}
	close(changesChannel)

	// Wait for the changes to get distributed
	time.Sleep(time.Second)
}

func Test_listen_nbi(t *testing.T) {
	d := NewDispatcher()
	ch, err := d.RegisterNbi("nbi")
	assert.NilError(t, err, "Unexpected error when registering nbi %s", err)
	assert.Equal(t, 1, len(d.GetListeners()), "One NBI listener expected")
	changes := make(map[string]bool)
	// Start the main listener system
	go testSync(ch, changes)
	changesChannel := make(chan events.ConfigEvent, 10)
	go d.Listen(changesChannel)
	changeID := []byte("test")
	changeIDStr := base64.StdEncoding.EncodeToString(changeID)
	// Send down some changes
	for i := 1; i < 3; i++ {
		event := events.CreateConfigEvent("foobar", changeID, true)
		changesChannel <- event
	}

	// Wait for the changes to get distributed
	time.Sleep(time.Second)
	assert.Equal(t, changes[changeIDStr], true, "Wrong key/value pair")

	close(changesChannel)

	d.UnregisterNbi("nbi")
}

func Test_listen_operational(t *testing.T) {
	d := NewDispatcher()
	ch, err := d.RegisterOpState("nbiOpState")
	assert.NilError(t, err, "Unexpected error when registering nbi %s", err)
	assert.Equal(t, 1, len(d.GetListeners()), "One OpState listener expected")
	changes := make(map[string]string)
	// Start the main listener system
	go testSyncOpState(ch, changes)
	opStateCh := make(chan events.OperationalStateEvent, 10)
	go d.ListenOperationalState(opStateCh)
	changesEvent := make(map[string]string)
	// Send down some changes
	changesEvent["test"] = "testValue"
	event := events.CreateOperationalStateEvent("foobar", changesEvent)
	opStateCh <- event

	// Wait for the changes to get distributed
	time.Sleep(time.Second)
	assert.Equal(t, changes["test"], "testValue", "Wrong key/value pair")

	close(opStateCh)

	d.UnregisterOperationalState("nbiOpState")
}

func Test_listen_none(t *testing.T) {
	d := NewDispatcher()
	assert.Equal(t, 0, len(d.GetListeners()), "No listeners expected")

	// Start the main listener system
	changesChannel := make(chan events.ConfigEvent, 10)
	go d.Listen(changesChannel)
	changeID := []byte("test")
	// Send down some changes
	for i := 1; i < 3; i++ {
		event := events.CreateConfigEvent("foobar", changeID, true)
		changesChannel <- event
	}
	close(changesChannel)
}

func Test_register_dup(t *testing.T) {
	d := NewDispatcher()
	_, _ = d.RegisterNbi("nbi")
	_, _, _ = d.RegisterDevice(device.ID("dev1"))
	i := len(d.GetListeners())
	_, _ = d.RegisterNbi("nbi")
	assert.Equal(t, i, len(d.GetListeners()), "Duplicate NBI listener added")
	_, _, _ = d.RegisterDevice(device.ID("dev1"))
	assert.Equal(t, i, len(d.GetListeners()), "Duplicate device listener added")
}

func testSync(testChan <-chan events.ConfigEvent, changes map[string]bool) {
	log.Info("Listen for config changes for Test")

	for nbiChange := range testChan {
		changeID := nbiChange.ChangeID()
		changes[changeID] = nbiChange.Applied()
		log.Info("Change for Test ", nbiChange)
	}
}

func testSyncOpState(testChan <-chan events.OperationalStateEvent, changes map[string]string) {
	log.Info("Listen for config changes for Test")

	for opStateChange := range testChan {
		changesEvent := events.Event(opStateChange).Values()
		for k, v := range *changesEvent {
			changes[k] = v
		}
		log.Info("OperationalState change for Test ", opStateChange)
	}
}
