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
	protognmi "github.com/openconfig/gnmi/proto/gnmi"
	"testing"

	gnmiutils "github.com/onosproject/onos-config/test/utils/gnmi"
	hautils "github.com/onosproject/onos-config/test/utils/ha"
	"github.com/onosproject/onos-config/test/utils/proto"
)

const (
	restartTzValue          = "Europe/Milan"
	restartTzPath           = "/system/clock/config/timezone-name"
	restartLoginBannerPath  = "/system/config/login-banner"
	restartMotdBannerPath   = "/system/config/motd-banner"
	restartLoginBannerValue = "LOGIN BANNER"
	restartMotdBannerValue  = "MOTD BANNER"
)

// TestGetOperationAfterNodeRestart tests a Get operation after restarting the onos-config node
func (s *TestSuite) TestGetOperationAfterNodeRestart(t *testing.T) {
	ctx, cancel := gnmiutils.MakeContext()
	defer cancel()

	// Create a simulated target
	simulator := gnmiutils.CreateSimulator(ctx, t)
	defer gnmiutils.DeleteSimulator(t, simulator)

	// Make a GNMI client to use for onos-config requests
	gnmiClient := gnmiutils.NewOnosConfigGNMIClientOrFail(ctx, t, gnmiutils.WithRetry)

	targetPath := gnmiutils.GetTargetPathWithValue(simulator.Name(), restartTzPath, restartTzValue, proto.StringVal)

	// Set a value using onos-config

	var setReq = &gnmiutils.SetRequest{
		Ctx:         ctx,
		Client:      gnmiClient,
		UpdatePaths: targetPath,
		Extensions:  gnmiutils.SyncExtension(t),
		Encoding:    protognmi.Encoding_PROTO,
	}
	setReq.SetOrFail(t)

	// Check that the value was set correctly
	gnmiutils.CheckGNMIValue(ctx, t, gnmiClient, targetPath, gnmiutils.NoExtensions, restartTzValue, 0, "Query after set returned the wrong value")

	// Restart onos-config
	configPod := hautils.FindPodWithPrefix(t, "onos-config")
	hautils.CrashPodOrFail(t, configPod)

	// Check that the value was set correctly in the new onos-config instance
	gnmiutils.CheckGNMIValue(ctx, t, gnmiClient, targetPath, gnmiutils.NoExtensions, restartTzValue, 0, "Query after restart returned the wrong value")

	// Check that the value is set on the target
	targetGnmiClient := gnmiutils.NewSimulatorGNMIClientOrFail(ctx, t, simulator)
	gnmiutils.CheckTargetValue(ctx, t, targetGnmiClient, targetPath, gnmiutils.NoExtensions, restartTzValue)
}

// TestSetOperationAfterNodeRestart tests a Set operation after restarting the onos-config node
func (s *TestSuite) TestSetOperationAfterNodeRestart(t *testing.T) {
	ctx, cancel := gnmiutils.MakeContext()
	defer cancel()

	// Create a simulated target
	simulator := gnmiutils.CreateSimulator(ctx, t)
	defer gnmiutils.DeleteSimulator(t, simulator)

	// Make a GNMI client to use for onos-config requests
	gnmiClient := gnmiutils.NewOnosConfigGNMIClientOrFail(ctx, t, gnmiutils.WithRetry)

	tzPath := gnmiutils.GetTargetPathWithValue(simulator.Name(), restartTzPath, restartTzValue, proto.StringVal)
	loginBannerPath := gnmiutils.GetTargetPathWithValue(simulator.Name(), restartLoginBannerPath, restartLoginBannerValue, proto.StringVal)
	motdBannerPath := gnmiutils.GetTargetPathWithValue(simulator.Name(), restartMotdBannerPath, restartMotdBannerValue, proto.StringVal)

	targets := []string{simulator.Name(), simulator.Name()}
	paths := []string{restartLoginBannerPath, restartMotdBannerPath}
	values := []string{restartLoginBannerValue, restartMotdBannerValue}

	bannerPaths := gnmiutils.GetTargetPathsWithValues(targets, paths, values)

	// Restart onos-config
	configPod := hautils.FindPodWithPrefix(t, "onos-config")
	hautils.CrashPodOrFail(t, configPod)

	// Set values using onos-config
	var setReq = &gnmiutils.SetRequest{
		Ctx:        ctx,
		Client:     gnmiClient,
		Extensions: gnmiutils.SyncExtension(t),
		Encoding:   protognmi.Encoding_PROTO,
	}
	setReq.UpdatePaths = tzPath
	setReq.SetOrFail(t)
	setReq.UpdatePaths = bannerPaths
	setReq.SetOrFail(t)

	// Check that the values were set correctly
	gnmiutils.CheckGNMIValue(ctx, t, gnmiClient, tzPath, gnmiutils.SyncExtension(t), restartTzValue, 0, "Query TZ after set returned the wrong value")
	gnmiutils.CheckGNMIValue(ctx, t, gnmiClient, loginBannerPath, gnmiutils.SyncExtension(t), restartLoginBannerValue, 0, "Query login banner after set returned the wrong value")
	gnmiutils.CheckGNMIValue(ctx, t, gnmiClient, motdBannerPath, gnmiutils.SyncExtension(t), restartMotdBannerValue, 0, "Query MOTD banner after set returned the wrong value")

	// Check that the values are set on the target
	targetGnmiClient := gnmiutils.NewSimulatorGNMIClientOrFail(ctx, t, simulator)
	gnmiutils.CheckTargetValue(ctx, t, targetGnmiClient, tzPath, gnmiutils.NoExtensions, restartTzValue)
	gnmiutils.CheckTargetValue(ctx, t, targetGnmiClient, loginBannerPath, gnmiutils.NoExtensions, restartLoginBannerValue)
	gnmiutils.CheckTargetValue(ctx, t, targetGnmiClient, motdBannerPath, gnmiutils.NoExtensions, restartMotdBannerValue)

}
