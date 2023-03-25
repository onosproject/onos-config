// SPDX-FileCopyrightText: 2020-present Open Networking Foundation <info@opennetworking.org>
//
// SPDX-License-Identifier: Apache-2.0

package rbac

import (
	"context"
	"github.com/onosproject/onos-config/test"
	gnmiutils "github.com/onosproject/onos-config/test/utils/gnmi"
	"github.com/onosproject/onos-config/test/utils/proto"
	gnmiapi "github.com/openconfig/gnmi/proto/gnmi"
)

// TestNoToken tests access to a protected API with no access token supplied
func (s *TestSuite) TestNoToken(ctx context.Context) {
	const (
		tzValue = "Europe/Dublin"
		tzPath  = "/system/clock/config/timezone-name"
	)

	// Make a GNMI client to use for requests
	gnmiClient := s.NewOnosConfigGNMIClientOrFail(ctx, test.NoRetry)

	// Try to fetch a value from the GNMI client
	devicePath := gnmiutils.GetTargetPathWithValue(s.simulator, tzPath, tzValue, proto.StringVal)
	var onosConfigGetReq = &gnmiutils.GetRequest{
		Ctx:      ctx,
		Client:   gnmiClient,
		Paths:    devicePath,
		Encoding: gnmiapi.Encoding_PROTO,
		DataType: gnmiapi.GetRequest_CONFIG,
	}
	_, err := onosConfigGetReq.Get()

	// An error indicating an unauthenticated request is expected
	s.Error(err)
	if err != nil {
		s.Contains(err.Error(), "Request unauthenticated with bearer")
	}
}
