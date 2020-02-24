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

package cache

import (
	"fmt"
	"io"
	"sync"

	networkchange "github.com/onosproject/onos-config/api/types/change/network"
	"github.com/onosproject/onos-config/api/types/device"
	devicesnapshot "github.com/onosproject/onos-config/api/types/snapshot/device"
	networkchangestore "github.com/onosproject/onos-config/pkg/store/change/network"
	devicesnapshotstore "github.com/onosproject/onos-config/pkg/store/snapshot/device"
	"github.com/onosproject/onos-config/pkg/store/stream"
	"github.com/onosproject/onos-lib-go/pkg/logging"
)

var log = logging.GetLogger("store", "device", "cache")

// Info is device type/version info
type Info struct {
	DeviceID device.ID
	Type     device.Type
	Version  device.Version
}

// Cache is a device type/version cache
type Cache interface {
	io.Closer

	// GetDevicesByID returns the devices that match the given device ID
	GetDevicesByID(id device.ID) []*Info

	// GetDevicesByType gets all devices of the given type
	GetDevicesByType(deviceType device.Type) []*Info

	// GetDevicesByVersion gets all devices of the given type/version
	GetDevicesByVersion(deviceType device.Type, deviceVersion device.Version) []*Info

	// GetDevices returns the set of devices in the cache
	GetDevices() []*Info

	// Watch allows tracking updates of the cache
	Watch(ch chan<- stream.Event, replay bool) (stream.Context, error)
}

// NewCache returns a new cache based on the NetworkChange store
func NewCache(networkChangeStore networkchangestore.Store,
	deviceSnapshotStore devicesnapshotstore.Store) (Cache, error) {
	cache := &networkChangeStoreCache{
		networkChangeStore:  networkChangeStore,
		deviceSnapshotStore: deviceSnapshotStore,
		devices:             make(map[device.VersionedID]*Info),
		listeners:           make(map[chan<- stream.Event]struct{}),
	}

	if err := cache.listen(); err != nil {
		return nil, err
	}
	return cache, nil
}

// networkChangeStoreCache is a device cache based on the NetworkChange store
type networkChangeStoreCache struct {
	networkChangeStore  networkchangestore.Store
	deviceSnapshotStore devicesnapshotstore.Store
	devices             map[device.VersionedID]*Info
	mu                  sync.RWMutex
	listeners           map[chan<- stream.Event]struct{}
}

func (c *networkChangeStoreCache) getListeners() []chan<- stream.Event {
	listeners := make([]chan<- stream.Event, 0, len(c.listeners))
	for listener := range c.listeners {
		listeners = append(listeners, listener)
	}
	return listeners
}

// listen starts listening for network changes
func (c *networkChangeStoreCache) listen() error {
	ch := make(chan stream.Event)
	ctx, err := c.networkChangeStore.Watch(ch, networkchangestore.WithReplay())
	if err != nil {
		return err
	}

	// Also check the snapshots
	ssCtx, err := c.deviceSnapshotStore.Watch(ch)
	if err != nil {
		return err
	}

	go func() {
		for event := range ch {
			if event.Type == stream.Deleted {
				// TODO handle delete properly - it should check to see if any
				//  of the devices in the removed/deleted NW change exist in any
				//  of the remaining NW changes. If they don't then remove them
				//  from the cache and send an event to listeners to that effect
				//  This will be needed in the case where a network change is made
				//  but something in it is invalid and it needs to be removed.
				//  The related devices should not be present in the cache after
				//  this check
				continue
			}
			netChange, ok := event.Object.(*networkchange.NetworkChange)
			if ok {
				for _, devChange := range netChange.Changes {
					key := device.NewVersionedID(devChange.DeviceID, devChange.DeviceVersion)
					c.mu.Lock()
					if _, ok := c.devices[key]; !ok {
						info := Info{
							DeviceID: devChange.DeviceID,
							Type:     devChange.DeviceType,
							Version:  devChange.DeviceVersion,
						}
						c.devices[key] = &info
						log.Infof("Updating cache with %v. Size %d Listeners %d", info, len(c.devices), len(c.listeners))
						listeners := c.getListeners()
						c.mu.Unlock()
						for _, l := range listeners {
							if l != nil {
								l <- stream.Event{
									Type:   stream.Created,
									Object: &info,
								}
							}
						}
					} else {
						c.mu.Unlock()
					}
				}
			}
			ssChange, ok := event.Object.(*devicesnapshot.DeviceSnapshot)
			if ok {
				key := device.NewVersionedID(ssChange.DeviceID, ssChange.DeviceVersion)
				info := Info{
					DeviceID: ssChange.DeviceID,
					Type:     ssChange.DeviceType,
					Version:  ssChange.DeviceVersion,
				}
				c.mu.Lock()
				if _, ok := c.devices[key]; !ok {
					c.devices[key] = &info
					log.Infof("Updating cache with %v. Size %d Listeners %d", info, len(c.devices), len(c.listeners))
					listeners := c.getListeners()
					c.mu.Unlock()
					for _, l := range listeners {
						if l != nil {
							l <- stream.Event{
								Type:   stream.Created,
								Object: &info,
							}
						}
					}
				} else {
					c.mu.Unlock()
				}
			}
		}
		ctx.Close()
		ssCtx.Close()
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

func (c *networkChangeStoreCache) GetDevicesByType(deviceType device.Type) []*Info {
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

func (c *networkChangeStoreCache) GetDevicesByVersion(deviceType device.Type, deviceVersion device.Version) []*Info {
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

// Watch streams device cache updates to the caller
// Unlike Watch on an Atomix store this Watch has to take care that an event is
// sent to each watch caller - hence the listener array
// A replay option allows former entries to be replayed to the caller
// The stream.Context should be closed when the caller is finished, otherwise
// a deadlock or panic will occur
// Also **before** calling this Watch() please ensure that the channel `ch` is active
// and listening on a thread - otherwise deadlock will occur
func (c *networkChangeStoreCache) Watch(ch chan<- stream.Event, replay bool) (stream.Context, error) {
	c.mu.RLock()
	_, ok := c.listeners[ch]
	c.mu.RUnlock()
	if ok {
		return nil, fmt.Errorf("already listening to channel %v", ch)
	}
	c.mu.Lock()
	c.listeners[ch] = struct{}{}
	if replay {
		devices := make(map[device.VersionedID]*Info)
		for device, info := range c.devices {
			devices[device] = info
		}
		c.mu.Unlock()
		for _, info := range devices {
			event := stream.Event{
				Type:   stream.None,
				Object: info,
			}
			ch <- event
		}
	} else {
		c.mu.Unlock()
	}

	return stream.NewContext(func() {
		c.mu.Lock()
		delete(c.listeners, ch)
		c.mu.Unlock()
	}), nil
}

func (c *networkChangeStoreCache) Close() error {
	return nil
}
