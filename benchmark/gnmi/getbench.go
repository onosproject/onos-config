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
	"time"
)

// BenchmarkGet tests get of GNMI paths
func (s *BenchmarkSuite) BenchmarkGet(b *benchmark.Benchmark) {
	b.Run(func() error {
		devicePath := getDevicePath(s.simulator.Name(), "/system/config/motd-banner")
		ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
		defer cancel()
		_, _, err := getGNMIValue(ctx, s.client, devicePath)
		return err
	})
}
