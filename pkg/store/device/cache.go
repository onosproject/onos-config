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

package device

import (
	"fmt"
	networkchangestore "github.com/onosproject/onos-config/pkg/store/change/network"
	"github.com/onosproject/onos-config/pkg/store/stream"
	networkchangetypes "github.com/onosproject/onos-config/pkg/types/change/network"
	"github.com/onosproject/onos-topo/pkg/northbound/device"
	"io"
	"sync"
)

const separator = ":"

// newKey returns a new cache key
func newKey(id device.ID, version string) string {
	return fmt.Sprintf("%s%s%s", id, separator, version)
}

// Info is device type/version info
type Info struct {
	DeviceID device.ID
	Type     string
	Version  string
}

// Cache is a device type/version cache
type Cache interface {
	io.Closer

	// GetDevicesByID returns the devices that match the given device ID
	GetDevicesByID(id device.ID) []*Info

	// GetDevicesByType gets all devices of the given type
	GetDevicesByType(deviceType string) []*Info

	// GetDevicesByVersion gets all devices of the given type/version
	GetDevicesByVersion(deviceType string, deviceVersion string) []*Info

	// GetDevices returns the set of devices in the cache
	GetDevices() []*Info
}

// NewCache returns a new cache based on the NetworkChange store
func NewCache(networkChangeStore networkchangestore.Store) (Cache, error) {
	cache := &networkChangeStoreCache{
		networkChangeStore: networkChangeStore,
		devices:            make(map[string]*Info),
	}
	if err := cache.listen(); err != nil {
		return nil, err
	}
	return cache, nil
}

// networkChangeStoreCache is a device cache based on the NetworkChange store
type networkChangeStoreCache struct {
	networkChangeStore networkchangestore.Store
	devices            map[string]*Info
	mu                 sync.RWMutex
}

// listen starts listening for network changes
func (c *networkChangeStoreCache) listen() error {
	ch := make(chan stream.Event)
	ctx, err := c.networkChangeStore.Watch(ch)
	if err != nil {
		return err
	}

	go func() {
		for event := range ch {
			netChange := event.Object.(*networkchangetypes.NetworkChange)
			for _, devChange := range netChange.Changes {
				key := newKey(devChange.DeviceID, devChange.DeviceVersion)
				c.mu.Lock()
				if _, ok := c.devices[key]; !ok {
					c.devices[key] = &Info{
						DeviceID: devChange.DeviceID,
						Type:     devChange.DeviceType,
						Version:  devChange.DeviceVersion,
					}
				}
				c.mu.Unlock()
			}
		}
		ctx.Close()
	}()
	return nil
}

func (c *networkChangeStoreCache) GetDevicesByID(id device.ID) []*Info {
	c.mu.RLock()
	defer c.mu.RUnlock()
	devices := make([]*Info, 0, len(c.devices))
	for _, info := range c.devices {
		if info.DeviceID == id {
			devices = append(devices, info)
		}
	}
	return devices
}

func (c *networkChangeStoreCache) GetDevicesByType(deviceType string) []*Info {
	c.mu.RLock()
	defer c.mu.RUnlock()
	devices := make([]*Info, 0, len(c.devices))
	for _, info := range c.devices {
		if info.Type == deviceType {
			devices = append(devices, info)
		}
	}
	return devices
}

func (c *networkChangeStoreCache) GetDevicesByVersion(deviceType string, deviceVersion string) []*Info {
	c.mu.RLock()
	defer c.mu.RUnlock()
	devices := make([]*Info, 0, len(c.devices))
	for _, info := range c.devices {
		if info.Type == deviceType && info.Version == deviceVersion {
			devices = append(devices, info)
		}
	}
	return devices
}

func (c *networkChangeStoreCache) GetDevices() []*Info {
	c.mu.RLock()
	defer c.mu.RUnlock()

	devices := make([]*Info, 0, len(c.devices))
	for _, info := range c.devices {
		devices = append(devices, info)
	}
	return devices
}

func (c *networkChangeStoreCache) Close() error {
	return nil
}
