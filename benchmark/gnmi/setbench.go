// Copyright 2020-present Open Networking Foundation.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

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
