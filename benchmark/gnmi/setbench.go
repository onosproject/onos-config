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
	"github.com/onosproject/onos-config/test/utils/gnmi"
	"github.com/onosproject/onos-config/test/utils/proto"
	"time"
)

// BenchmarkSet tests set of GNMI paths
func (s *BenchmarkSuite) BenchmarkSet(b *benchmark.Benchmark) error {
	devicePath := gnmi.GetTargetPathWithValue(s.simulator.Name(), "/system/config/motd-banner", s.value.Next().String(), proto.StringVal)
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	_, _, err := gnmi.SetGNMIValue(ctx, s.client, devicePath, gnmi.NoPaths, gnmi.NoExtensions)
	return err
}
