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
	gnmiapi "github.com/openconfig/gnmi/proto/gnmi"
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
		Encoding:    gnmiapi.Encoding_PROTO,
	}
	setReq.SetOrFail(t)

	// Check that the value was set correctly
	var getReq = &gnmiutils.GetRequest{
		Ctx:        ctx,
		Client:     gnmiClient,
		Paths:      targetPath,
		Extensions: gnmiutils.SyncExtension(t),
		Encoding:   gnmiapi.Encoding_PROTO,
	}
	getReq.CheckValue(t, restartTzValue)

	// Restart onos-config
	configPod := hautils.FindPodWithPrefix(t, "onos-config")
	hautils.CrashPodOrFail(t, configPod)

	// Check that the value was set correctly in the new onos-config instance
	getReq.CheckValue(t, restartTzValue)

	// Check that the value is set on the target
	targetGnmiClient := gnmiutils.NewSimulatorGNMIClientOrFail(ctx, t, simulator)
	var getTargetReq = &gnmiutils.GetRequest{
		Ctx:      ctx,
		Client:   targetGnmiClient,
		Encoding: gnmiapi.Encoding_JSON,
		Paths:    targetPath,
	}
	getTargetReq.CheckValue(t, restartTzValue)
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
		Encoding:   gnmiapi.Encoding_PROTO,
	}
	setReq.UpdatePaths = tzPath
	setReq.SetOrFail(t)
	setReq.UpdatePaths = bannerPaths
	setReq.SetOrFail(t)

	// Check that the values were set correctly
	var getConfigReq = &gnmiutils.GetRequest{
		Ctx:        ctx,
		Client:     gnmiClient,
		Extensions: gnmiutils.SyncExtension(t),
		Encoding:   gnmiapi.Encoding_PROTO,
	}
	getConfigReq.Paths = tzPath
	getConfigReq.CheckValue(t, restartTzValue)
	getConfigReq.Paths = loginBannerPath
	getConfigReq.CheckValue(t, restartLoginBannerValue)
	getConfigReq.Paths = motdBannerPath
	getConfigReq.CheckValue(t, restartMotdBannerValue)

	// Check that the values are set on the target
	targetGnmiClient := gnmiutils.NewSimulatorGNMIClientOrFail(ctx, t, simulator)
	var getTargetReq = &gnmiutils.GetRequest{
		Ctx:      ctx,
		Client:   targetGnmiClient,
		Encoding: gnmiapi.Encoding_JSON,
	}
	getTargetReq.Paths = tzPath
	getTargetReq.CheckValue(t, restartTzValue)
	getTargetReq.Paths = loginBannerPath
	getTargetReq.CheckValue(t, restartLoginBannerValue)
	getTargetReq.Paths = motdBannerPath
	getTargetReq.CheckValue(t, restartMotdBannerValue)

}
