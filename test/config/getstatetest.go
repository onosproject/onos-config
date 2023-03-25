// SPDX-FileCopyrightText: 2020-present Open Networking Foundation <info@opennetworking.org>
//
// SPDX-License-Identifier: Apache-2.0

package config

import (
	"context"
	"github.com/onosproject/onos-config/test"
	gnmiutils "github.com/onosproject/onos-config/test/utils/gnmi"
	gnmiapi "github.com/openconfig/gnmi/proto/gnmi"
)

const (
	stateValue1 = "opennetworking.org"
	statePath1  = "/system/state/domain-name"
	statePath2  = "/system/state/login-banner"
	stateValue2 = "This device is for authorized use only"
)

// TestGetState tests query/set/delete of a single GNMI path to a single device
func (s *TestSuite) TestGetState(ctx context.Context) {
	// Make a GNMI client to use for requests
	gnmiClient := s.NewOnosConfigGNMIClientOrFail(ctx, test.WithRetry)

	// Get the GNMI path
	targetPaths := gnmiutils.GetTargetPathsWithValues([]string{s.simulator1, s.simulator1}, []string{statePath1, statePath2}, []string{stateValue1, stateValue2})

	// Set up requests
	var onosConfigGetReq = &gnmiutils.GetRequest{
		Ctx:      ctx,
		Client:   gnmiClient,
		Paths:    targetPaths,
		Encoding: gnmiapi.Encoding_JSON,
		DataType: gnmiapi.GetRequest_STATE,
	}

	// Check that the value was set correctly, both in onos-config and on the target
	onosConfigGetReq.CheckValues(s.T(), stateValue1, stateValue2)

}
