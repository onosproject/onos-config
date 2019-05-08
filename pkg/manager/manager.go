package manager

import (
	"fmt"
	"github.com/onosproject/onos-config/pkg/southbound/topocache"
	"github.com/onosproject/onos-config/pkg/store"
	"github.com/onosproject/onos-config/pkg/store/change"
	"log"
)

var (
	configStore *store.ConfigurationStore
	changeStore *store.ChangeStore
	deviceStore *topocache.DeviceStore
	networkStore *store.NetworkStore
)

// Manager is a thread and the entry point for getting, setting and manipulating changes to and from the stores.
func Manager(configs *store.ConfigurationStore, changes *store.ChangeStore, device *topocache.DeviceStore,
	network *store.NetworkStore) {
	configStore = configs
	changeStore = changes
	deviceStore = device
	networkStore = network
	log.Println("Starting Manager")
}

// GetNetworkConfig returns a set of change values given a target, a configuration name, a path and a layer.
// The layer is the numbers of configu changes we want to go back in time for.
func GetNetworkConfig(target string, configname string, path string, layer int) ([]change.ConfigValue, error){
	if _, ok := deviceStore.Store[target]; !ok {
		return nil, fmt.Errorf("Device not present %s", target)
	}
	//TODO the key of the config store shoudl be a tuple of (devicename, configname) use the param
	var config store.Configuration
	for configID, cfg := range configStore.Store{
		if cfg.Device == target {
			configname = configID
			config = cfg
			break
		}
	}
	configValues := config.ExtractFullConfig(changeStore.Store, layer)
	return configValues, nil
}
