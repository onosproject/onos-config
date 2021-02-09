// Copyright 2021-present Open Networking Foundation.
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

package rbac

import (
	"fmt"
	devicechange "github.com/onosproject/onos-api/go/onos/config/change/device"
	"github.com/onosproject/onos-api/go/onos/config/device"
	devicechangestore "github.com/onosproject/onos-config/pkg/store/change/device"
	devicesnapshotstore "github.com/onosproject/onos-config/pkg/store/snapshot/device"
	"github.com/onosproject/onos-config/pkg/store/stream"
	"regexp"
	"strings"

	"github.com/onosproject/onos-lib-go/pkg/logging"
	"io"
)

const (
	roleQueryExact      = `^/rbac/role\[roleid=.*](/roleid)?$`
	groupQueryExact     = `^/rbac/group\[groupid=.*](/groupid)?$`
	groupRoleQueryExact = `^/rbac/group\[groupid=.*]/role\[roleid=.*](roleid)?$`
)

var log = logging.GetLogger("store", "change", "device", "rbac")
var (
	roleSearchExact      = regexp.MustCompile(roleQueryExact)
	groupSearchExact     = regexp.MustCompile(groupQueryExact)
	groupRoleSearchExact = regexp.MustCompile(groupRoleQueryExact)
)

// Cache is the Role Based Access Control (RBAC) cache
// We can deal only with Paths and Values, and stay out of the realm of
// the rbac_1_0_0 object model. Validation will ensure we only get clean
// data in to this map
type Cache interface {
	io.Closer

	FilterGetByRbacRoles(getResults []*devicechange.PathValue,
		groupIds []string) ([]*devicechange.PathValue, error)
}

// NewRbacCache - create a new Rbac Cache
func NewRbacCache(devicesChangeStore devicechangestore.Store,
	deviceSnapshotStore devicesnapshotstore.Store,
	rbacInternalDevice device.VersionedID) (Cache, error) {
	cache := &rbacCache{
		rbacMap:             make(devicechange.TypedValueMap),
		devicesChangeStore:  devicesChangeStore,
		deviceSnapshotStore: deviceSnapshotStore,
		id:                  rbacInternalDevice,
	}
	if err := cache.listen(); err != nil {
		return nil, err
	}
	return cache, nil
}

type rbacCache struct {
	rbacMap             devicechange.TypedValueMap
	devicesChangeStore  devicechangestore.Store
	deviceSnapshotStore devicesnapshotstore.Store
	id                  device.VersionedID
}

func (r *rbacCache) listen() error {
	log.Infof("Starting RBAC Cache listener on %s", r.id)
	ch := make(chan stream.Event)
	ctx, err := r.devicesChangeStore.Watch(r.id, ch, devicechangestore.WithReplay())
	if err != nil {
		return err
	}

	// Also check the snapshots
	ssCtx, err := r.deviceSnapshotStore.Watch(ch)
	if err != nil {
		return err
	}

	go func() {
		for event := range ch {
			switch chType := event.Object.(type) {
			case *devicechange.DeviceChange:
				r.handleChanges(chType.Change.Values)

			default:
				log.Infof("did not expect type %v as device change event", chType)
			}
		}
		ctx.Close()
		ssCtx.Close()
	}()
	return nil
}

func (r *rbacCache) handleChanges(change []*devicechange.ChangeValue) {
	var added, updated, removed int
	for _, c := range change {
		if c.Removed {
			delete(r.rbacMap, c.Path)
			searchPath := fmt.Sprintf("^%s", strings.ReplaceAll(c.Path, `-`, `\-`))
			searchPath = strings.ReplaceAll(searchPath, `[`, `\[`)
			wildSearch := regexp.MustCompile(searchPath)
			if roleSearchExact.MatchString(c.Path) {
				// Have to remove children too
				for rKey := range r.rbacMap {
					if wildSearch.MatchString(rKey) {
						delete(r.rbacMap, rKey)
						removed++
					}
				}
			} else if groupSearchExact.MatchString(c.Path) {
				// Have to remove children too
				for rKey := range r.rbacMap {
					if wildSearch.MatchString(rKey) {
						delete(r.rbacMap, rKey)
						removed++
					}
				}
			} else if groupRoleSearchExact.MatchString(c.Path) {
				// Have to remove children too
				for rKey := range r.rbacMap {
					if wildSearch.MatchString(rKey) {
						delete(r.rbacMap, rKey)
						removed++
					}
				}
			}
			removed++
		} else {
			if _, ok := r.rbacMap[c.Path]; !ok {
				added++
			} else {
				updated++
			}
			r.rbacMap[c.Path] = c.Value
		}
	}
	log.Infof("RBAC cache new size %d. Event %d, added %d, updated %d, removed %d",
		len(r.rbacMap), len(change), added, updated, removed)

}

func (r *rbacCache) Close() error {
	return nil
}
