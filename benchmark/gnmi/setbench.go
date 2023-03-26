// SPDX-FileCopyrightText: 2020-present Open Networking Foundation <info@opennetworking.org>
//
// SPDX-License-Identifier: Apache-2.0

package gnmi

import (
	"context"
	petname "github.com/dustinkirkland/golang-petname"
	gnmiutils "github.com/onosproject/onos-config/benchmark/utils/gnmi"
	"github.com/onosproject/onos-config/benchmark/utils/proto"
	gnmiapi "github.com/openconfig/gnmi/proto/gnmi"
)

// BenchmarkSet tests set of GNMI paths
func (s *BenchmarkSuite) BenchmarkSet(ctx context.Context) error {
	devicePath := gnmiutils.GetTargetPathWithValue(s.simulator.Name, "/system/config/motd-banner", petname.Generate(2, " "), proto.StringVal)
	var setReq = &gnmiutils.SetRequest{
		Ctx:         ctx,
		Client:      s.client,
		Encoding:    gnmiapi.Encoding_PROTO,
		UpdatePaths: devicePath,
	}
	_, _, err := setReq.Set()
	return err
}
