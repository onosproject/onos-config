// SPDX-FileCopyrightText: 2020-present Open Networking Foundation <info@opennetworking.org>
//
// SPDX-License-Identifier: Apache-2.0

package gnmi

import (
	"context"
	"github.com/onosproject/helmit/pkg/benchmark"
	gnmiutils "github.com/onosproject/onos-config/test/utils/gnmi"
	gnmiapi "github.com/openconfig/gnmi/proto/gnmi"
	"time"
)

// BenchmarkGet tests get of GNMI paths
func (s *BenchmarkSuite) BenchmarkGet(b *benchmark.Benchmark) error {
	devicePath := gnmiutils.GetTargetPath(s.simulator.Name(), "/system/config/motd-banner")
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	var onosConfigGetReq = &gnmiutils.GetRequest{
		Ctx:      ctx,
		Client:   s.client,
		Paths:    devicePath,
		Encoding: gnmiapi.Encoding_PROTO,
	}
	_, err := onosConfigGetReq.Get()
	return err
}
