// SPDX-FileCopyrightText: 2020-present Open Networking Foundation <info@opennetworking.org>
//
// SPDX-License-Identifier: Apache-2.0

package utils

import (
	configapi "github.com/onosproject/onos-api/go/onos/config/v2"
	topoapi "github.com/onosproject/onos-api/go/onos/topo"
	"github.com/onosproject/onos-lib-go/pkg/env"
	"github.com/onosproject/onos-lib-go/pkg/logging"
	"github.com/onosproject/onos-lib-go/pkg/uri"
	"strings"
)

var log = logging.GetLogger("controller", "utils")

// GetOnosConfigID gets onos-config URI
func GetOnosConfigID() topoapi.ID {
	return topoapi.ID(uri.NewURI(
		uri.WithScheme("gnmi"),
		uri.WithOpaque(env.GetPodID())).String())
}

// AddDeleteChildren adds all children of the intermediate path which is required to be deleted
func AddDeleteChildren(changeValues map[string]*configapi.PathValue, configStore map[string]*configapi.PathValue) map[string]*configapi.PathValue {
	log.Debugf("Obtained changeValues are:\n%v", changeValues)
	// defining new changeValues map, where we will include old changeValues map and new pathValues to be cascading deleted
	var updChangeValues = make(map[string]*configapi.PathValue)
	for _, changeValue := range changeValues {
		log.Debugf("Processing value %v", changeValue)
		// if this pathValue has to be deleted, then we need to search for all children of this pathValue
		// this algorithm (lines 34-48) could be a source of potential issues..
		if changeValue.Deleted {
			log.Debugf("Value %v is required to be deleted", changeValue)
			for _, value := range configStore {
				if strings.HasPrefix(value.Path, changeValue.Path) && !strings.EqualFold(value.Path, changeValue.Path) {
					log.Debugf("Found a child - adding it\n%v", value)
					updChangeValues[value.Path] = value
					updChangeValues[value.Path].Deleted = true
				}
			}
			// overwriting itself in the store, we want the latest value (changeValue variable)
			updChangeValues[changeValue.Path] = changeValue
		} else {
			log.Debugf("Value %v is NOT required to be deleted - adding it to the map", changeValue)
			updChangeValues[changeValue.Path] = changeValue
		}
	}
	log.Debugf("Updated changeValues are:\n%v", updChangeValues)
	return updChangeValues
}
