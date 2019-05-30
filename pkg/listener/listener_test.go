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

package listener

import (
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/onosproject/onos-config/pkg/events"
	"github.com/onosproject/onos-config/pkg/southbound/topocache"
	"github.com/onosproject/onos-config/pkg/store"
	"gotest.tools/assert"
	is "gotest.tools/assert/cmp"
)

var (
	device1Channel, device2Channel, device3Channel chan events.ConfigEvent
	device1, device2, device3                      topocache.Device
	err                                            error
)

const (
	configStoreDefaultFileName = "../store/testout/configStore-sample.json"
	changeStoreDefaultFileName = "../store/testout/changeStore-sample.json"
)

var (
	configStoreTest store.ConfigurationStore
	changeStoreTest store.ChangeStore
)

func setUp() *Dispatcher {
	d := NewDispatcher()
	device1Channel, err = d.Register(device1.Addr, true)
	device2Channel, err = d.Register(device2.Addr, true)
	device3Channel, err = d.Register(device3.Addr, true)
	return &d
}

func tearDown(d *Dispatcher) {
	d.Unregister(device1.Addr, true)
	d.Unregister(device2.Addr, true)
	d.Unregister(device3.Addr, true)

}

func TestMain(m *testing.M) {
	device1 = topocache.Device{Addr: "localhost:10161"}
	device2 = topocache.Device{Addr: "localhost:10162"}
	device3 = topocache.Device{Addr: "localhost:10163"}

	configStoreTest, err = store.LoadConfigStore(configStoreDefaultFileName)
	if err != nil {
		wd, _ := os.Getwd()
		fmt.Println("Cannot load config store ", err, wd)
		return
	}
	fmt.Println("Configuration store loaded from", configStoreDefaultFileName)

	changeStoreTest, err = store.LoadChangeStore(changeStoreDefaultFileName)
	if err != nil {
		fmt.Println("Cannot load change store ", err)
		return
	}

	os.Exit(m.Run())
}

func Test_getListeners(t *testing.T) {
	d := setUp()
	listeners := d.GetListeners()

	assert.Assert(t, len(listeners) == 3, "Expected to find 3 devices in list. Got %d", len(listeners))

	listenerStr := strings.Join(listeners, ",")
	// Could be in any order
	assert.Assert(t, is.Contains(listenerStr, "localhost:10161"), "Expected to find device1 in list. Got %s", listeners)
	assert.Assert(t, is.Contains(listenerStr, "localhost:10162"), "Expected to find device2 in list. Got %s", listeners)
	assert.Assert(t, is.Contains(listenerStr, "localhost:10163"), "Expected to find device3 in list. Got %s", listeners)
	tearDown(d)
}

func Test_register(t *testing.T) {
	d := setUp()
	device4Channel, err := d.Register("device4", true)
	assert.NilError(t, err, "Unexpected error when registering device %s", err)

	var deviceChannelIf interface{} = device4Channel
	chanType, ok := deviceChannelIf.(chan events.ConfigEvent)
	assert.Assert(t, ok, "Unexpected channel type when registering device %v", chanType)
	d.Unregister("device4", true)
	tearDown(d)
}

func Test_unregister(t *testing.T) {
	d := setUp()
	err1 := d.Unregister("device5", true)

	assert.Assert(t, err1 != nil, "Unexpected lack of error when unregistering non existent device")
	assert.Assert(t, is.Contains(err1.Error(), "had not been registered"), "Unexpected error text when unregistering non existent device %s", err1)

	_, err1 = d.Register("device6", true)
	_, err1 = d.Register("device7", true)

	err2 := d.Unregister("device6", true)
	assert.NilError(t, err2, "Unexpected error when unregistering device6 %s", err)
	d.Unregister("device7", true)
	tearDown(d)
}

func Test_listen(t *testing.T) {
	d := setUp()

	// Start a test listener
	testChan := make(chan events.Event, 10)
	go testSync(testChan)
	// Start the main listener system
	changesChannel := make(chan events.ConfigEvent, 10)
	go d.Listen(changesChannel)
	changeID := []byte("test")
	// Send down some changes
	for i := 1; i < 13; i++ {
		event := events.CreateConfigEvent("device"+strconv.Itoa(i), changeID, true)

		changesChannel <- event
	}

	// Wait for the changes to get distributed
	time.Sleep(time.Second)
	tearDown(d)
	close(testChan)
	close(changesChannel)
}

func Test_listen_nbi(t *testing.T) {
	d := NewDispatcher()
	d.Register("nbi", false)
	assert.Equal(t, 1, len(d.GetListeners()), "One NBI listener expected")

	// Start the main listener system
	changesChannel := make(chan events.ConfigEvent, 10)
	go d.Listen(changesChannel)
	changeID := []byte("test")
	// Send down some changes
	for i := 1; i < 3; i++ {
		event := events.CreateConfigEvent("foobar", changeID, true)
		changesChannel <- event
	}

	d.Unregister("nbi", false)
	close(changesChannel)
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
	d.Register("nbi", false)
	d.Register("dev1", true)
	i := len(d.GetListeners())
	d.Register("nbi", false)
	assert.Equal(t, i, len(d.GetListeners()), "Duplicate NBI listener added")
	d.Register("dev1", true)
	assert.Equal(t, i, len(d.GetListeners()), "Duplicate device listener added")
}

func testSync(testChan <-chan events.Event) {
	log.Println("Listen for config changes for Test")

	for nbiChange := range testChan {
		fmt.Println("Change for Test", nbiChange)
	}
}
