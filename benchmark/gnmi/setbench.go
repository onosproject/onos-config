// SPDX-FileCopyrightText: 2020-present Open Networking Foundation <info@opennetworking.org>
//
// SPDX-License-Identifier: Apache-2.0

package gnmi

import (
	"context"
	"github.com/onosproject/helmit/pkg/benchmark"
	gnmiutils "github.com/onosproject/onos-config/test/utils/gnmi"
	"github.com/onosproject/onos-config/test/utils/proto"
	gnmiapi "github.com/openconfig/gnmi/proto/gnmi"
	"time"
)

// BenchmarkSet tests set of GNMI paths
func (s *BenchmarkSuite) BenchmarkSet(b *benchmark.Benchmark) error {
	devicePath := gnmiutils.GetTargetPathWithValue(s.simulator.Name(), "/system/config/motd-banner", s.value.Next().String(), proto.StringVal)
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	var setReq = &gnmiutils.SetRequest{
		Ctx:         ctx,
		Client:      s.client,
		Encoding:    gnmiapi.Encoding_PROTO,
		UpdatePaths: devicePath,
	}
	_, _, err := setReq.Set()
	return err
}
