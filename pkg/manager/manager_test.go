package manager

import (
	"fmt"
	"github.com/onosproject/onos-config/pkg/southbound/topocache"
	"github.com/onosproject/onos-config/pkg/store"
	"github.com/onosproject/onos-config/pkg/store/change"
	"os"
	"testing"
	"time"
)

var (
	change1 *change.Change
)

var (
	device1config store.Configuration
)

var (
	changeStoreTest        map[string]*change.Change
	configurationStoreTest map[string]store.Configuration
	networkStoreTest       []store.NetworkConfiguration
	deviceStoreTest        map[string]topocache.Device
)

const (
	Test1Cont1A                  = "/test1:cont1a"
	Test1Cont1ACont2A            = "/test1:cont1a/cont2a"
	Test1Cont1ACont2ALeaf2A      = "/test1:cont1a/cont2a/leaf2a"
)

const (
	ValueEmpty          = ""
	ValueLeaf2A13       = "13"
)

func TestMain(m *testing.M) {
	var err error
	config1Value01, _ := change.CreateChangeValue(Test1Cont1A, ValueEmpty, false)
	config1Value02, _ := change.CreateChangeValue(Test1Cont1ACont2A, ValueEmpty, false)
	config1Value03, _ := change.CreateChangeValue(Test1Cont1ACont2ALeaf2A, ValueLeaf2A13, false)
	change1, err = change.CreateChange(change.ValueCollections{
		config1Value01, config1Value02, config1Value03,}, "Original Config for test switch")
	if err != nil {
		fmt.Println(err)
		os.Exit(-1)
	}
	changeStoreTest = make(map[string]*change.Change)
	changeStoreTest[store.B64(change1.ID)] = change1

	device1config = store.Configuration{
		Name:        "Device1Version",
		Device:      "Device1",
		Created:     time.Now(),
		Updated:     time.Now(),
		User:        "onos",
		Description: "Configuration for Device 1",
		Changes:     []change.ID{change1.ID},
	}
	configurationStoreTest = make(map[string]store.Configuration)
	configurationStoreTest["Device1Version"] = device1config

	deviceStoreTest = make(map[string]topocache.Device)
	deviceStoreTest["Device1"] = topocache.Device{
		Addr: "127.0.0.1:10161",
		Timeout: 10,
	}

	go Manager(&store.ConfigurationStore{"1.0", "config", configurationStoreTest},
	&store.ChangeStore{"1.0", "change", changeStoreTest},
	&topocache.DeviceStore{"1.0", "change", deviceStoreTest},
	&store.NetworkStore{"1.0", "network", networkStoreTest})

	os.Exit(m.Run())

}

func Test_GetNetworkConfig(t *testing.T) {

	result, err := GetNetworkConfig("Device1", "running", "*", 0)
	if err != nil {
		t.Errorf("%s", err)
	}
	if len(result) != 3 {
		t.Errorf("result is empty %s", result)
	}

	if result[0].Path != Test1Cont1A {
		t.Errorf("result %s is different from %s", result[0].Path, Test1Cont1A)
	}
}

