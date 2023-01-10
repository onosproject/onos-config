// SPDX-FileCopyrightText: 2022-present Intel Corporation
//
// SPDX-License-Identifier: Apache-2.0

package config

import (
	topoapi "github.com/onosproject/onos-api/go/onos/topo"
	gnmiutils "github.com/onosproject/onos-config/test/utils/gnmi"
	"github.com/onosproject/onos-config/test/utils/proto"
	gnmiapi "github.com/openconfig/gnmi/proto/gnmi"
	"github.com/stretchr/testify/assert"
	"testing"
	"time"
)

const (
	cfPath                            = "/system/config"
	cfHostnameValue                   = "switch1"
	cfHostnamePath                    = cfPath + "/hostname"
	cfDomainNameValue                 = "opennetworking.org"
	cfDomainNamePath                  = cfPath + "/domain-name"
	cfLoginBannerValue                = "Some banner"
	cfLoginBannerPath                 = cfPath + "/login-banner"
	cfMotdBannerValue                 = "Some motd banner"
	cfMotdBannerPath                  = cfPath + "/motd-banner"
	ofAgentConfigPath                 = "/system/openflow/agent/config"
	ofAgentConfigBOIntervalValue      = "5"
	ofAgentConfigBOIntervalPath       = ofAgentConfigPath + "/backoff-interval"
	ofAgentConfigDpIDValue            = "00:16:3e:00:00:00:00:00"
	ofAgentConfigDpIOPath             = ofAgentConfigPath + "/datapath-id"
	ofAgentConfigFailureModeValue     = "SECURE"
	ofAgentConfigFailureModePath      = ofAgentConfigPath + "/failure-mode"
	ofAgentConfigInactivityProbeValue = "10"
	ofAgentConfigInactivityProbePath  = ofAgentConfigPath + "/inactivity-probe"
	ofAgentConfigMaxBOValue           = "10"
	ofAgentConfigMaxBOPath            = ofAgentConfigPath + "/max-backoff"
)

// TestCascadingDelete checks that when a root leaf or an intermediate leaf is removed, all its children are removed too.
// This test tests verifies only single scenario - tree-path deletion (i.e., /system/config and /system/openflow/agent)
// Container-path deletion is verified in treepathtest.go.
func (s *TestSuite) TestCascadingDelete(t *testing.T) {
	ctx, cancel := gnmiutils.MakeContext()
	defer cancel()

	// Create a simulated target
	simulator := gnmiutils.CreateSimulator(ctx, t)
	defer gnmiutils.DeleteSimulator(t, simulator)

	// Wait for config to connect to the target
	ready := gnmiutils.WaitForTargetAvailable(ctx, t, topoapi.ID(simulator.Name()), 1*time.Minute)
	assert.True(t, ready)

	// Make a GNMI client to use for requests
	gnmiClient := gnmiutils.NewOnosConfigGNMIClientOrFail(ctx, t, gnmiutils.NoRetry)
	targetClient := gnmiutils.NewSimulatorGNMIClientOrFail(ctx, t, simulator)

	// setting path-value for ../config/hostname
	// Creating the GNMI path
	targetPath1 := gnmiutils.GetTargetPathWithValue(simulator.Name(), cfHostnamePath, cfHostnameValue, proto.StringVal)

	// Set up requests
	var onosConfigGetReq = &gnmiutils.GetRequest{
		Ctx:      ctx,
		Client:   gnmiClient,
		Paths:    targetPath1,
		Encoding: gnmiapi.Encoding_PROTO,
	}
	var simulatorGetReq = &gnmiutils.GetRequest{
		Ctx:      ctx,
		Client:   targetClient,
		Paths:    targetPath1,
		Encoding: gnmiapi.Encoding_JSON,
	}
	var onosConfigSetReq = &gnmiutils.SetRequest{
		Ctx:         ctx,
		Client:      gnmiClient,
		UpdatePaths: targetPath1,
		Extensions:  gnmiutils.SyncExtension(t),
		Encoding:    gnmiapi.Encoding_PROTO,
	}

	// Set a new value for the time zone using onos-config
	onosConfigSetReq.SetOrFail(t)

	// Check that the value was set correctly, both in onos-config and on the target
	onosConfigGetReq.CheckValues(t, cfHostnameValue)
	simulatorGetReq.CheckValues(t, cfHostnameValue)

	// setting path-value for ../config/domain-name
	// Creating the GNMI path
	targetPath2 := gnmiutils.GetTargetPathWithValue(simulator.Name(), cfDomainNamePath, cfDomainNameValue, proto.StringVal)

	// Set up requests
	onosConfigGetReq = &gnmiutils.GetRequest{
		Ctx:      ctx,
		Client:   gnmiClient,
		Paths:    targetPath2,
		Encoding: gnmiapi.Encoding_PROTO,
	}
	simulatorGetReq = &gnmiutils.GetRequest{
		Ctx:      ctx,
		Client:   targetClient,
		Paths:    targetPath2,
		Encoding: gnmiapi.Encoding_JSON,
	}
	onosConfigSetReq = &gnmiutils.SetRequest{
		Ctx:         ctx,
		Client:      gnmiClient,
		UpdatePaths: targetPath2,
		Extensions:  gnmiutils.SyncExtension(t),
		Encoding:    gnmiapi.Encoding_PROTO,
	}

	// Set a new value for the time zone using onos-config
	onosConfigSetReq.SetOrFail(t)

	// Check that the value was set correctly, both in onos-config and on the target
	onosConfigGetReq.CheckValues(t, cfDomainNameValue)
	simulatorGetReq.CheckValues(t, cfDomainNameValue)

	// setting path-value for ../config/login-banner
	// Creating the GNMI path
	targetPath3 := gnmiutils.GetTargetPathWithValue(simulator.Name(), cfLoginBannerPath, cfLoginBannerValue, proto.StringVal)

	// Set up requests
	onosConfigGetReq = &gnmiutils.GetRequest{
		Ctx:      ctx,
		Client:   gnmiClient,
		Paths:    targetPath3,
		Encoding: gnmiapi.Encoding_PROTO,
	}
	simulatorGetReq = &gnmiutils.GetRequest{
		Ctx:      ctx,
		Client:   targetClient,
		Paths:    targetPath3,
		Encoding: gnmiapi.Encoding_JSON,
	}
	onosConfigSetReq = &gnmiutils.SetRequest{
		Ctx:         ctx,
		Client:      gnmiClient,
		UpdatePaths: targetPath3,
		Extensions:  gnmiutils.SyncExtension(t),
		Encoding:    gnmiapi.Encoding_PROTO,
	}

	// Set a new value for the time zone using onos-config
	onosConfigSetReq.SetOrFail(t)

	// Check that the value was set correctly, both in onos-config and on the target
	onosConfigGetReq.CheckValues(t, cfLoginBannerValue)
	simulatorGetReq.CheckValues(t, cfLoginBannerValue)

	// setting path-value for ../config/motd-banner
	// Creating the GNMI path
	targetPath4 := gnmiutils.GetTargetPathWithValue(simulator.Name(), cfMotdBannerPath, cfMotdBannerValue, proto.StringVal)

	// Set up requests
	onosConfigGetReq = &gnmiutils.GetRequest{
		Ctx:      ctx,
		Client:   gnmiClient,
		Paths:    targetPath4,
		Encoding: gnmiapi.Encoding_PROTO,
	}
	simulatorGetReq = &gnmiutils.GetRequest{
		Ctx:      ctx,
		Client:   targetClient,
		Paths:    targetPath4,
		Encoding: gnmiapi.Encoding_JSON,
	}
	onosConfigSetReq = &gnmiutils.SetRequest{
		Ctx:         ctx,
		Client:      gnmiClient,
		UpdatePaths: targetPath4,
		Extensions:  gnmiutils.SyncExtension(t),
		Encoding:    gnmiapi.Encoding_PROTO,
	}

	// Set a new value for the time zone using onos-config
	onosConfigSetReq.SetOrFail(t)

	// Check that the value was set correctly, both in onos-config and on the target
	onosConfigGetReq.CheckValues(t, cfMotdBannerValue)
	simulatorGetReq.CheckValues(t, cfMotdBannerValue)

	// setting path-value for ../agent/config/backoff-interval banner
	// Creating the GNMI path
	targetPath5 := gnmiutils.GetTargetPathWithValue(simulator.Name(), ofAgentConfigBOIntervalPath, ofAgentConfigBOIntervalValue, proto.IntVal)

	// Set up requests
	onosConfigGetReq = &gnmiutils.GetRequest{
		Ctx:      ctx,
		Client:   gnmiClient,
		Paths:    targetPath5,
		Encoding: gnmiapi.Encoding_PROTO,
	}
	simulatorGetReq = &gnmiutils.GetRequest{
		Ctx:      ctx,
		Client:   targetClient,
		Paths:    targetPath5,
		Encoding: gnmiapi.Encoding_JSON,
	}
	onosConfigSetReq = &gnmiutils.SetRequest{
		Ctx:         ctx,
		Client:      gnmiClient,
		UpdatePaths: targetPath5,
		Extensions:  gnmiutils.SyncExtension(t),
		Encoding:    gnmiapi.Encoding_PROTO,
	}

	// Set a new value for the time zone using onos-config
	onosConfigSetReq.SetOrFail(t)

	// Check that the value was set correctly, both in onos-config and on the target
	onosConfigGetReq.CheckValues(t, ofAgentConfigBOIntervalValue)
	simulatorGetReq.CheckValues(t, ofAgentConfigBOIntervalValue)

	// setting path-value for ../agent/config/datapath-id banner
	// Creating the GNMI path
	targetPath6 := gnmiutils.GetTargetPathWithValue(simulator.Name(), ofAgentConfigDpIOPath, ofAgentConfigDpIDValue, proto.StringVal)

	// Set up requests
	onosConfigGetReq = &gnmiutils.GetRequest{
		Ctx:      ctx,
		Client:   gnmiClient,
		Paths:    targetPath6,
		Encoding: gnmiapi.Encoding_PROTO,
	}
	simulatorGetReq = &gnmiutils.GetRequest{
		Ctx:      ctx,
		Client:   targetClient,
		Paths:    targetPath6,
		Encoding: gnmiapi.Encoding_JSON,
	}
	onosConfigSetReq = &gnmiutils.SetRequest{
		Ctx:         ctx,
		Client:      gnmiClient,
		UpdatePaths: targetPath6,
		Extensions:  gnmiutils.SyncExtension(t),
		Encoding:    gnmiapi.Encoding_PROTO,
	}

	// Set a new value for the time zone using onos-config
	onosConfigSetReq.SetOrFail(t)

	// Check that the value was set correctly, both in onos-config and on the target
	onosConfigGetReq.CheckValues(t, ofAgentConfigDpIDValue)
	simulatorGetReq.CheckValues(t, ofAgentConfigDpIDValue)

	// setting path-value for ../agent/config/failure-mode banner
	// Creating the GNMI path
	targetPath7 := gnmiutils.GetTargetPathWithValue(simulator.Name(), ofAgentConfigFailureModePath, ofAgentConfigFailureModeValue, proto.StringVal)

	// Set up requests
	onosConfigGetReq = &gnmiutils.GetRequest{
		Ctx:      ctx,
		Client:   gnmiClient,
		Paths:    targetPath7,
		Encoding: gnmiapi.Encoding_PROTO,
	}
	simulatorGetReq = &gnmiutils.GetRequest{
		Ctx:      ctx,
		Client:   targetClient,
		Paths:    targetPath7,
		Encoding: gnmiapi.Encoding_JSON,
	}
	onosConfigSetReq = &gnmiutils.SetRequest{
		Ctx:         ctx,
		Client:      gnmiClient,
		UpdatePaths: targetPath7,
		Extensions:  gnmiutils.SyncExtension(t),
		Encoding:    gnmiapi.Encoding_PROTO,
	}

	// Set a new value for the time zone using onos-config
	onosConfigSetReq.SetOrFail(t)

	// Check that the value was set correctly, both in onos-config and on the target
	onosConfigGetReq.CheckValues(t, ofAgentConfigFailureModeValue)
	simulatorGetReq.CheckValues(t, ofAgentConfigFailureModeValue)

	// setting path-value for ../agent/config/inactivity-probe banner
	// Creating the GNMI path
	targetPath8 := gnmiutils.GetTargetPathWithValue(simulator.Name(), ofAgentConfigInactivityProbePath, ofAgentConfigInactivityProbeValue, proto.IntVal)

	// Set up requests
	onosConfigGetReq = &gnmiutils.GetRequest{
		Ctx:      ctx,
		Client:   gnmiClient,
		Paths:    targetPath8,
		Encoding: gnmiapi.Encoding_PROTO,
	}
	simulatorGetReq = &gnmiutils.GetRequest{
		Ctx:      ctx,
		Client:   targetClient,
		Paths:    targetPath8,
		Encoding: gnmiapi.Encoding_JSON,
	}
	onosConfigSetReq = &gnmiutils.SetRequest{
		Ctx:         ctx,
		Client:      gnmiClient,
		UpdatePaths: targetPath8,
		Extensions:  gnmiutils.SyncExtension(t),
		Encoding:    gnmiapi.Encoding_PROTO,
	}

	// Set a new value for the time zone using onos-config
	onosConfigSetReq.SetOrFail(t)

	// Check that the value was set correctly, both in onos-config and on the target
	onosConfigGetReq.CheckValues(t, ofAgentConfigInactivityProbeValue)
	simulatorGetReq.CheckValues(t, ofAgentConfigInactivityProbeValue)

	// setting path-value for ../agent/config/max-backoff banner
	// Creating the GNMI path
	targetPath9 := gnmiutils.GetTargetPathWithValue(simulator.Name(), ofAgentConfigMaxBOPath, ofAgentConfigMaxBOValue, proto.IntVal)

	// Set up requests
	onosConfigGetReq = &gnmiutils.GetRequest{
		Ctx:      ctx,
		Client:   gnmiClient,
		Paths:    targetPath9,
		Encoding: gnmiapi.Encoding_PROTO,
	}
	simulatorGetReq = &gnmiutils.GetRequest{
		Ctx:      ctx,
		Client:   targetClient,
		Paths:    targetPath9,
		Encoding: gnmiapi.Encoding_JSON,
	}
	onosConfigSetReq = &gnmiutils.SetRequest{
		Ctx:         ctx,
		Client:      gnmiClient,
		UpdatePaths: targetPath9,
		Extensions:  gnmiutils.SyncExtension(t),
		Encoding:    gnmiapi.Encoding_PROTO,
	}

	// Set a new value for the time zone using onos-config
	onosConfigSetReq.SetOrFail(t)

	// Check that the value was set correctly, both in onos-config and in the target
	onosConfigGetReq.CheckValues(t, ofAgentConfigMaxBOValue)
	simulatorGetReq.CheckValues(t, ofAgentConfigMaxBOValue)

	// get the root path
	getPath1 := gnmiutils.GetTargetPath(simulator.Name(), cfPath)

	// Remove the path we added
	onosConfigSetReq.DeletePaths = getPath1
	onosConfigSetReq.UpdatePaths = nil
	onosConfigSetReq.SetOrFail(t)

	getPath2 := gnmiutils.GetTargetPath(simulator.Name(), ofAgentConfigPath)

	// Remove the path we added
	onosConfigSetReq.DeletePaths = getPath2
	onosConfigSetReq.UpdatePaths = nil
	onosConfigSetReq.SetOrFail(t)

	//  Make sure it got removed, both in onos-config and in the target
	onosConfigGetReq.CheckValuesDeleted(t)
	simulatorGetReq.CheckValuesDeleted(t)
}
