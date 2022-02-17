// Copyright 2019-present Open Networking Foundation.
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

package config

import (
	"context"
	"github.com/onosproject/helmit/pkg/helm"
	"github.com/onosproject/onos-config/test/utils/proto"
	gnmiapi "github.com/openconfig/gnmi/proto/gnmi"
	"testing"
	"time"

	"github.com/onosproject/onos-api/go/onos/topo"

	gnmiutils "github.com/onosproject/onos-config/test/utils/gnmi"
)

const (
	crashedTargetValue1 = "test-motd-banner"
	crashedTargetPath1  = "/system/config/motd-banner"
	crashedTargetValue2 = "test-login-banner"
	crashedTargetPath2  = "/system/config/login-banner"
)

var (
	crashedTargetPaths  = []string{crashedTargetPath1, crashedTargetPath2}
	crashedTargetValues = []string{crashedTargetValue1, crashedTargetValue2}
)

// TestCrashedTarget tests that a crashed target receives proper configuration restoration
func (s *TestSuite) TestCrashedTarget(t *testing.T) {
	ctx, cancel := gnmiutils.MakeContext()
	defer cancel()

	// Create a simulator and wait for it to become available
	target := gnmiutils.CreateSimulator(ctx, t)
	gnmiutils.WaitForTargetAvailable(ctx, t, topo.ID(target.Name()), time.Minute)

	// Set up crashedTargetPaths to configure
	targets := []string{target.Name()}
	targetPathsForGet := gnmiutils.GetTargetPaths(targets, crashedTargetPaths)

	// Make a GNMI client to use for requests
	gnmiClient := gnmiutils.NewOnosConfigGNMIClientOrFail(ctx, t, gnmiutils.NoRetry)

	// Set initial crashedTargetValues
	targetPathsForInit := gnmiutils.GetTargetPathsWithValues(targets, crashedTargetPaths, crashedTargetValues)

	var setReq = &gnmiutils.SetRequest{
		Ctx:         ctx,
		Client:      gnmiClient,
		Encoding:    gnmiapi.Encoding_PROTO,
		Extensions:  gnmiutils.SyncExtension(t),
		UpdatePaths: targetPathsForInit,
	}
	setReq.SetOrFail(t)

	// Make sure the configuration has been applied to both onos-config
	var getReq = &gnmiutils.GetRequest{
		Ctx:        ctx,
		Client:     gnmiClient,
		Encoding:   gnmiapi.Encoding_PROTO,
		Extensions: gnmiutils.SyncExtension(t),
	}
	targetPath1 := gnmiutils.GetTargetPath(target.Name(), crashedTargetPath1)
	targetPath2 := gnmiutils.GetTargetPath(target.Name(), crashedTargetPath2)

	// Check that the crashedTargetValues were set correctly
	getReq.Paths = targetPath1
	getReq.CheckValues(t, crashedTargetValue1)
	getReq.Paths = targetPath2
	getReq.CheckValues(t, crashedTargetValue2)

	// ... and the target
	checkTarget(ctx, t, target, targetPathsForGet)

	// Kill the target simulator
	gnmiutils.DeleteSimulator(t, target)

	// Re-create the target simulator with the same name and wait for it to become available
	target = gnmiutils.CreateSimulatorWithName(ctx, t, target.Name(), false)
	defer gnmiutils.DeleteSimulator(t, target)
	gnmiutils.WaitForTargetAvailable(ctx, t, topo.ID(target.Name()), time.Minute)

	// FIXME: Here is a potential for a race between onos-config reapplying the changes to the freshly restarted
	// target and the following checks. The race is won favorably most of the time, but possibility exists of the
	// race being lost. We need to come up with a more robust guard, but this solves the problem in the meantime.
	time.Sleep(5 * time.Second)

	// Make sure the configuration has been re-applied to the target
	checkTarget(ctx, t, target, targetPathsForGet)
}

// Check that the crashedTargetValues are set on the target
func checkTarget(ctx context.Context, t *testing.T, target *helm.HelmRelease, targetPathsForGet []proto.TargetPath) {
	targetGnmiClient := gnmiutils.NewSimulatorGNMIClientOrFail(ctx, t, target)

	var targetGetReq = &gnmiutils.GetRequest{
		Ctx:        ctx,
		Client:     targetGnmiClient,
		Encoding:   gnmiapi.Encoding_JSON,
		Extensions: gnmiutils.SyncExtension(t),
	}
	targetGetReq.Paths = targetPathsForGet[0:1]
	targetGetReq.CheckValues(t, crashedTargetValue1)
	targetGetReq.Paths = targetPathsForGet[1:2]
	targetGetReq.CheckValues(t, crashedTargetValue2)
}
