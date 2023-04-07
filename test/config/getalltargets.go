// SPDX-FileCopyrightText: 2020-present Open Networking Foundation <info@opennetworking.org>
//
// SPDX-License-Identifier: Apache-2.0

package config

import (
	"github.com/onosproject/onos-api/go/onos/topo"
	"github.com/onosproject/onos-config/test"
	gnmiapi "github.com/openconfig/gnmi/proto/gnmi"
	"strings"

	gnmiutils "github.com/onosproject/onos-config/test/utils/gnmi"
)

// TestGetAllTargets tests retrieval of all target IDs via path.Target="*"
func (s *TestSuite) TestGetAllTargets() {
	// Wait for config to connect to both simulators
	s.WaitForTargetAvailable(topo.ID(s.simulator1.Name))
	s.WaitForTargetAvailable(topo.ID(s.simulator2.Name))

	// Make a GNMI client to use for requests
	gnmiClient := s.NewOnosConfigGNMIClientOrFail(test.NoRetry)

	// Get the list of all targets via get query on target "*"
	var getReq = &gnmiutils.GetRequest{
		Ctx:        s.Context(),
		Client:     gnmiClient,
		Encoding:   gnmiapi.Encoding_PROTO,
		Extensions: s.SyncExtension(),
		Paths:      gnmiutils.GetTargetPath("*", ""),
	}
	getValue, err := getReq.Get()
	s.NoError(err)
	s.Len(getValue, 1)
	s.Equal("/all-targets", getValue[0].Path)
	s.True(strings.Contains(getValue[0].PathDataValue, s.simulator1.Name))
	s.True(strings.Contains(getValue[0].PathDataValue, s.simulator2.Name))
}
