// SPDX-FileCopyrightText: 2020-present Open Networking Foundation <info@opennetworking.org>
//
// SPDX-License-Identifier: Apache-2.0

package utils

import (
	configapi "github.com/onosproject/onos-api/go/onos/config/v2"
	topoapi "github.com/onosproject/onos-api/go/onos/topo"
	"github.com/onosproject/onos-lib-go/pkg/env"
	"github.com/onosproject/onos-lib-go/pkg/uri"
	"strings"
)

// GetOnosConfigID gets onos-config URI
func GetOnosConfigID() topoapi.ID {
	return topoapi.ID(uri.NewURI(
		uri.WithScheme("gnmi"),
		uri.WithOpaque(env.GetPodID())).String())
}

// AddDeleteChildren adds all children of the intermediate path which is required to be deleted
func AddDeleteChildren(changeValues map[string]*configapi.PathValue, configStore map[string]*configapi.PathValue) map[string]*configapi.PathValue {
	// defining new changeValues map, where we will include old changeValues map and new pathValues to be cascading deleted
	var updChangeValues = make(map[string]*configapi.PathValue)
	for prefix, changeValue := range changeValues {
		// if this pathValue has to be deleted, then we need to search for all children of this pathValue
		if changeValue.Deleted {
			for path, value := range configStore {
				if strings.HasPrefix(path, prefix) {
					updChangeValues[path] = value
				}
			}
		} else {
			updChangeValues[prefix] = changeValue
		}
	}
	return updChangeValues
}
