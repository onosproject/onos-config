// SPDX-FileCopyrightText: 2020-present Open Networking Foundation <info@opennetworking.org>
//
// SPDX-License-Identifier: Apache-2.0

package gnmi

import (
	"context"
	gnmiutils "github.com/onosproject/onos-config/benchmark/utils/gnmi"
	gnmiapi "github.com/openconfig/gnmi/proto/gnmi"
)

// BenchmarkGet tests get of GNMI paths
func (s *BenchmarkSuite) BenchmarkGet(ctx context.Context) error {
	devicePath := gnmiutils.GetTargetPath(s.simulator.Name, "/system/config/motd-banner")
	var onosConfigGetReq = &gnmiutils.GetRequest{
		Ctx:      ctx,
		Client:   s.client,
		Paths:    devicePath,
		Encoding: gnmiapi.Encoding_PROTO,
	}
	_, err := onosConfigGetReq.Get()
	return err
}
