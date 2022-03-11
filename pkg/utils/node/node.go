// SPDX-FileCopyrightText: 2020-present Open Networking Foundation <info@opennetworking.org>
//
// SPDX-License-Identifier: Apache-2.0

package node

import (
	topoapi "github.com/onosproject/onos-api/go/onos/topo"
	"github.com/onosproject/onos-lib-go/pkg/env"
	"github.com/onosproject/onos-lib-go/pkg/uri"
)

// GetOnosConfigID gets onos-config URI
func GetOnosConfigID() topoapi.ID {
	return topoapi.ID(uri.NewURI(
		uri.WithScheme("gnmi"),
		uri.WithOpaque(env.GetPodID())).String())
}
