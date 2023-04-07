// SPDX-FileCopyrightText: 2022-present Intel Corporation
//
// SPDX-License-Identifier: Apache-2.0

package config

import (
	topoapi "github.com/onosproject/onos-api/go/onos/topo"
	"github.com/onosproject/onos-config/test"
	gnmiutils "github.com/onosproject/onos-config/test/utils/gnmi"
	"github.com/onosproject/onos-config/test/utils/proto"
	gnmiapi "github.com/openconfig/gnmi/proto/gnmi"
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
func (s *TestSuite) TestCascadingDelete() {
	// Wait for config to connect to the target
	ready := s.WaitForTargetAvailable(topoapi.ID(s.simulator1.Name))
	s.True(ready)

	// Make a GNMI client to use for requests
	gnmiClient := s.NewOnosConfigGNMIClientOrFail(test.NoRetry)
	targetClient := s.NewSimulatorGNMIClientOrFail(s.simulator1.Name)

	// setting path-value for ../config/hostname
	// Creating the GNMI path
	targetPath1 := gnmiutils.GetTargetPathWithValue(s.simulator1.Name, cfHostnamePath, cfHostnameValue, proto.StringVal)

	// Set up requests
	var onosConfigGetReq = &gnmiutils.GetRequest{
		Ctx:      s.Context(),
		Client:   gnmiClient,
		Paths:    targetPath1,
		Encoding: gnmiapi.Encoding_PROTO,
	}
	var simulatorGetReq = &gnmiutils.GetRequest{
		Ctx:      s.Context(),
		Client:   targetClient,
		Paths:    targetPath1,
		Encoding: gnmiapi.Encoding_JSON,
	}
	var onosConfigSetReq = &gnmiutils.SetRequest{
		Ctx:         s.Context(),
		Client:      gnmiClient,
		UpdatePaths: targetPath1,
		Extensions:  s.SyncExtension(),
		Encoding:    gnmiapi.Encoding_PROTO,
	}

	// Set a new value for the time zone using onos-config
	onosConfigSetReq.SetOrFail(s.T())

	// Check that the value was set correctly, both in onos-config and on the target
	onosConfigGetReq.CheckValues(s.T(), cfHostnameValue)
	simulatorGetReq.CheckValues(s.T(), cfHostnameValue)

	// setting path-value for ../config/domain-name
	// Creating the GNMI path
	targetPath2 := gnmiutils.GetTargetPathWithValue(s.simulator1.Name, cfDomainNamePath, cfDomainNameValue, proto.StringVal)

	// Set up requests
	onosConfigGetReq = &gnmiutils.GetRequest{
		Ctx:      s.Context(),
		Client:   gnmiClient,
		Paths:    targetPath2,
		Encoding: gnmiapi.Encoding_PROTO,
	}
	simulatorGetReq = &gnmiutils.GetRequest{
		Ctx:      s.Context(),
		Client:   targetClient,
		Paths:    targetPath2,
		Encoding: gnmiapi.Encoding_JSON,
	}
	onosConfigSetReq = &gnmiutils.SetRequest{
		Ctx:         s.Context(),
		Client:      gnmiClient,
		UpdatePaths: targetPath2,
		Extensions:  s.SyncExtension(),
		Encoding:    gnmiapi.Encoding_PROTO,
	}

	// Set a new value for the time zone using onos-config
	onosConfigSetReq.SetOrFail(s.T())

	// Check that the value was set correctly, both in onos-config and on the target
	onosConfigGetReq.CheckValues(s.T(), cfDomainNameValue)
	simulatorGetReq.CheckValues(s.T(), cfDomainNameValue)

	// setting path-value for ../config/login-banner
	// Creating the GNMI path
	targetPath3 := gnmiutils.GetTargetPathWithValue(s.simulator1.Name, cfLoginBannerPath, cfLoginBannerValue, proto.StringVal)

	// Set up requests
	onosConfigGetReq = &gnmiutils.GetRequest{
		Ctx:      s.Context(),
		Client:   gnmiClient,
		Paths:    targetPath3,
		Encoding: gnmiapi.Encoding_PROTO,
	}
	simulatorGetReq = &gnmiutils.GetRequest{
		Ctx:      s.Context(),
		Client:   targetClient,
		Paths:    targetPath3,
		Encoding: gnmiapi.Encoding_JSON,
	}
	onosConfigSetReq = &gnmiutils.SetRequest{
		Ctx:         s.Context(),
		Client:      gnmiClient,
		UpdatePaths: targetPath3,
		Extensions:  s.SyncExtension(),
		Encoding:    gnmiapi.Encoding_PROTO,
	}

	// Set a new value for the time zone using onos-config
	onosConfigSetReq.SetOrFail(s.T())

	// Check that the value was set correctly, both in onos-config and on the target
	onosConfigGetReq.CheckValues(s.T(), cfLoginBannerValue)
	simulatorGetReq.CheckValues(s.T(), cfLoginBannerValue)

	// setting path-value for ../config/motd-banner
	// Creating the GNMI path
	targetPath4 := gnmiutils.GetTargetPathWithValue(s.simulator1.Name, cfMotdBannerPath, cfMotdBannerValue, proto.StringVal)

	// Set up requests
	onosConfigGetReq = &gnmiutils.GetRequest{
		Ctx:      s.Context(),
		Client:   gnmiClient,
		Paths:    targetPath4,
		Encoding: gnmiapi.Encoding_PROTO,
	}
	simulatorGetReq = &gnmiutils.GetRequest{
		Ctx:      s.Context(),
		Client:   targetClient,
		Paths:    targetPath4,
		Encoding: gnmiapi.Encoding_JSON,
	}
	onosConfigSetReq = &gnmiutils.SetRequest{
		Ctx:         s.Context(),
		Client:      gnmiClient,
		UpdatePaths: targetPath4,
		Extensions:  s.SyncExtension(),
		Encoding:    gnmiapi.Encoding_PROTO,
	}

	// Set a new value for the time zone using onos-config
	onosConfigSetReq.SetOrFail(s.T())

	// Check that the value was set correctly, both in onos-config and on the target
	onosConfigGetReq.CheckValues(s.T(), cfMotdBannerValue)
	simulatorGetReq.CheckValues(s.T(), cfMotdBannerValue)

	// setting path-value for ../agent/config/backoff-interval banner
	// Creating the GNMI path
	targetPath5 := gnmiutils.GetTargetPathWithValue(s.simulator1.Name, ofAgentConfigBOIntervalPath, ofAgentConfigBOIntervalValue, proto.IntVal)

	// Set up requests
	onosConfigGetReq = &gnmiutils.GetRequest{
		Ctx:      s.Context(),
		Client:   gnmiClient,
		Paths:    targetPath5,
		Encoding: gnmiapi.Encoding_PROTO,
	}
	simulatorGetReq = &gnmiutils.GetRequest{
		Ctx:      s.Context(),
		Client:   targetClient,
		Paths:    targetPath5,
		Encoding: gnmiapi.Encoding_JSON,
	}
	onosConfigSetReq = &gnmiutils.SetRequest{
		Ctx:         s.Context(),
		Client:      gnmiClient,
		UpdatePaths: targetPath5,
		Extensions:  s.SyncExtension(),
		Encoding:    gnmiapi.Encoding_PROTO,
	}

	// Set a new value for the time zone using onos-config
	onosConfigSetReq.SetOrFail(s.T())

	// Check that the value was set correctly, both in onos-config and on the target
	onosConfigGetReq.CheckValues(s.T(), ofAgentConfigBOIntervalValue)
	simulatorGetReq.CheckValues(s.T(), ofAgentConfigBOIntervalValue)

	// setting path-value for ../agent/config/datapath-id banner
	// Creating the GNMI path
	targetPath6 := gnmiutils.GetTargetPathWithValue(s.simulator1.Name, ofAgentConfigDpIOPath, ofAgentConfigDpIDValue, proto.StringVal)

	// Set up requests
	onosConfigGetReq = &gnmiutils.GetRequest{
		Ctx:      s.Context(),
		Client:   gnmiClient,
		Paths:    targetPath6,
		Encoding: gnmiapi.Encoding_PROTO,
	}
	simulatorGetReq = &gnmiutils.GetRequest{
		Ctx:      s.Context(),
		Client:   targetClient,
		Paths:    targetPath6,
		Encoding: gnmiapi.Encoding_JSON,
	}
	onosConfigSetReq = &gnmiutils.SetRequest{
		Ctx:         s.Context(),
		Client:      gnmiClient,
		UpdatePaths: targetPath6,
		Extensions:  s.SyncExtension(),
		Encoding:    gnmiapi.Encoding_PROTO,
	}

	// Set a new value for the time zone using onos-config
	onosConfigSetReq.SetOrFail(s.T())

	// Check that the value was set correctly, both in onos-config and on the target
	onosConfigGetReq.CheckValues(s.T(), ofAgentConfigDpIDValue)
	simulatorGetReq.CheckValues(s.T(), ofAgentConfigDpIDValue)

	// setting path-value for ../agent/config/failure-mode banner
	// Creating the GNMI path
	targetPath7 := gnmiutils.GetTargetPathWithValue(s.simulator1.Name, ofAgentConfigFailureModePath, ofAgentConfigFailureModeValue, proto.StringVal)

	// Set up requests
	onosConfigGetReq = &gnmiutils.GetRequest{
		Ctx:      s.Context(),
		Client:   gnmiClient,
		Paths:    targetPath7,
		Encoding: gnmiapi.Encoding_PROTO,
	}
	simulatorGetReq = &gnmiutils.GetRequest{
		Ctx:      s.Context(),
		Client:   targetClient,
		Paths:    targetPath7,
		Encoding: gnmiapi.Encoding_JSON,
	}
	onosConfigSetReq = &gnmiutils.SetRequest{
		Ctx:         s.Context(),
		Client:      gnmiClient,
		UpdatePaths: targetPath7,
		Extensions:  s.SyncExtension(),
		Encoding:    gnmiapi.Encoding_PROTO,
	}

	// Set a new value for the time zone using onos-config
	onosConfigSetReq.SetOrFail(s.T())

	// Check that the value was set correctly, both in onos-config and on the target
	onosConfigGetReq.CheckValues(s.T(), ofAgentConfigFailureModeValue)
	simulatorGetReq.CheckValues(s.T(), ofAgentConfigFailureModeValue)

	// setting path-value for ../agent/config/inactivity-probe banner
	// Creating the GNMI path
	targetPath8 := gnmiutils.GetTargetPathWithValue(s.simulator1.Name, ofAgentConfigInactivityProbePath, ofAgentConfigInactivityProbeValue, proto.IntVal)

	// Set up requests
	onosConfigGetReq = &gnmiutils.GetRequest{
		Ctx:      s.Context(),
		Client:   gnmiClient,
		Paths:    targetPath8,
		Encoding: gnmiapi.Encoding_PROTO,
	}
	simulatorGetReq = &gnmiutils.GetRequest{
		Ctx:      s.Context(),
		Client:   targetClient,
		Paths:    targetPath8,
		Encoding: gnmiapi.Encoding_JSON,
	}
	onosConfigSetReq = &gnmiutils.SetRequest{
		Ctx:         s.Context(),
		Client:      gnmiClient,
		UpdatePaths: targetPath8,
		Extensions:  s.SyncExtension(),
		Encoding:    gnmiapi.Encoding_PROTO,
	}

	// Set a new value for the time zone using onos-config
	onosConfigSetReq.SetOrFail(s.T())

	// Check that the value was set correctly, both in onos-config and on the target
	onosConfigGetReq.CheckValues(s.T(), ofAgentConfigInactivityProbeValue)
	simulatorGetReq.CheckValues(s.T(), ofAgentConfigInactivityProbeValue)

	// setting path-value for ../agent/config/max-backoff banner
	// Creating the GNMI path
	targetPath9 := gnmiutils.GetTargetPathWithValue(s.simulator1.Name, ofAgentConfigMaxBOPath, ofAgentConfigMaxBOValue, proto.IntVal)

	// Set up requests
	onosConfigGetReq = &gnmiutils.GetRequest{
		Ctx:      s.Context(),
		Client:   gnmiClient,
		Paths:    targetPath9,
		Encoding: gnmiapi.Encoding_PROTO,
	}
	simulatorGetReq = &gnmiutils.GetRequest{
		Ctx:      s.Context(),
		Client:   targetClient,
		Paths:    targetPath9,
		Encoding: gnmiapi.Encoding_JSON,
	}
	onosConfigSetReq = &gnmiutils.SetRequest{
		Ctx:         s.Context(),
		Client:      gnmiClient,
		UpdatePaths: targetPath9,
		Extensions:  s.SyncExtension(),
		Encoding:    gnmiapi.Encoding_PROTO,
	}

	// Set a new value for the time zone using onos-config
	onosConfigSetReq.SetOrFail(s.T())

	// Check that the value was set correctly, both in onos-config and in the target
	onosConfigGetReq.CheckValues(s.T(), ofAgentConfigMaxBOValue)
	simulatorGetReq.CheckValues(s.T(), ofAgentConfigMaxBOValue)

	// get the root path
	getPath1 := gnmiutils.GetTargetPath(s.simulator1.Name, cfPath)

	// Remove the path we added
	onosConfigSetReq.DeletePaths = getPath1
	onosConfigSetReq.UpdatePaths = nil
	onosConfigSetReq.SetOrFail(s.T())

	getPath2 := gnmiutils.GetTargetPath(s.simulator1.Name, ofAgentConfigPath)

	// Remove the path we added
	onosConfigSetReq.DeletePaths = getPath2
	onosConfigSetReq.UpdatePaths = nil
	onosConfigSetReq.SetOrFail(s.T())

	//  Make sure it got removed, both in onos-config and in the target
	onosConfigGetReq.CheckValuesDeleted(s.T())
	simulatorGetReq.CheckValuesDeleted(s.T())

	getReq := &gnmiutils.GetRequest{
		Ctx:        s.Context(),
		Client:     gnmiClient,
		Extensions: s.SyncExtension(),
		Encoding:   gnmiapi.Encoding_PROTO,
		Paths:      gnmiutils.GetTargetPath(s.simulator1.Name, "/"),
	}
	paths, err := getReq.Get()
	s.NoError(err)
	s.False(checkForPath(paths, ofAgentConfigPath))
	s.False(checkForPath(paths, cfPath))
}
