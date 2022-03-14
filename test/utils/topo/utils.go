// SPDX-FileCopyrightText: 2020-present Open Networking Foundation <info@opennetworking.org>
//
// SPDX-License-Identifier: Apache-2.0
//

package topo

import (
	topoapi "github.com/onosproject/onos-api/go/onos/topo"
)

func getFilter(kind string) *topoapi.Filters {
	controlRelationFilter := &topoapi.Filters{
		KindFilter: &topoapi.Filter{
			Filter: &topoapi.Filter_Equal_{
				Equal_: &topoapi.EqualFilter{
					Value: kind,
				},
			},
		},
	}
	return controlRelationFilter

}

// GetControlRelationFilter gets control relation filter
func GetControlRelationFilter() *topoapi.Filters {
	return getFilter(topoapi.CONTROLS)
}
