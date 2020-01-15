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
	"github.com/onosproject/onos-test/pkg/benchmark"
	"github.com/onosproject/onos-test/pkg/benchmark/params"
	"time"
)

// BenchmarkSet tests set of GNMI paths
func (s *BenchmarkSuite) BenchmarkSet(b *benchmark.Benchmark) {
	b.Run(func(value string) error {
		devicePath := getDevicePathWithValue(s.simulator.Name(), "/system/config/motd-banner", value, StringVal)
		ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
		defer cancel()
		_, _, err := setGNMIValue(ctx, s.client, devicePath, noPaths, noExtensions)
		return err
	}, params.RandomString(8))
}
