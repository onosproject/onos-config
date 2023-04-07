// SPDX-FileCopyrightText: 2020-present Open Networking Foundation <info@opennetworking.org>
//
// SPDX-License-Identifier: Apache-2.0

package config

import (
	topoapi "github.com/onosproject/onos-api/go/onos/topo"
	"github.com/onosproject/onos-config/test"
	gnmiutils "github.com/onosproject/onos-config/test/utils/gnmi"
	"github.com/onosproject/onos-config/test/utils/proto"
	gnmiapi "github.com/openconfig/gnmi/proto/gnmi"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
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
func (s *TestSuite) TestGetOperationAfterNodeRestart() {
	// Wait for config to connect to the target
	ready := s.WaitForTargetAvailable(topoapi.ID(s.simulator1.Name))
	s.True(ready)

	// Make a GNMI client to use for onos-config requests
	gnmiClient := s.NewOnosConfigGNMIClientOrFail(test.WithRetry)

	targetPath := gnmiutils.GetTargetPathWithValue(s.simulator1.Name, restartTzPath, restartTzValue, proto.StringVal)

	// Set a value using onos-config

	var setReq = &gnmiutils.SetRequest{
		Ctx:         s.Context(),
		Client:      gnmiClient,
		UpdatePaths: targetPath,
		Extensions:  s.SyncExtension(),
		Encoding:    gnmiapi.Encoding_PROTO,
	}
	setReq.SetOrFail(s.T())

	// Check that the value was set correctly
	var getReq = &gnmiutils.GetRequest{
		Ctx:        s.Context(),
		Client:     gnmiClient,
		Paths:      targetPath,
		Extensions: s.SyncExtension(),
		Encoding:   gnmiapi.Encoding_PROTO,
	}
	getReq.CheckValues(s.T(), restartTzValue)

	// Get the onos-config pod
	pods, err := s.CoreV1().Pods(s.Namespace()).List(s.Context(), metav1.ListOptions{
		LabelSelector: "app=onos,type=config",
	})
	s.NoError(err)
	s.Len(pods.Items, 1)

	// Delete the onos-config pod to restart it
	err = s.CoreV1().Pods(s.Namespace()).Delete(s.Context(), pods.Items[0].Name, metav1.DeleteOptions{})
	s.NoError(err)

	// Check that the value was set correctly in the new onos-config instance
	getReq.CheckValues(s.T(), restartTzValue)

	// Check that the value is set on the target
	targetGnmiClient := s.NewSimulatorGNMIClientOrFail(s.simulator1.Name)
	var getTargetReq = &gnmiutils.GetRequest{
		Ctx:      s.Context(),
		Client:   targetGnmiClient,
		Encoding: gnmiapi.Encoding_JSON,
		Paths:    targetPath,
	}
	getTargetReq.CheckValues(s.T(), restartTzValue)
}

// TestSetOperationAfterNodeRestart tests a Set operation after restarting the onos-config node
func (s *TestSuite) TestSetOperationAfterNodeRestart() {
	// Make a GNMI client to use for onos-config requests
	gnmiClient := s.NewOnosConfigGNMIClientOrFail(test.WithRetry)

	tzPath := gnmiutils.GetTargetPathWithValue(s.simulator1.Name, restartTzPath, restartTzValue, proto.StringVal)
	loginBannerPath := gnmiutils.GetTargetPathWithValue(s.simulator1.Name, restartLoginBannerPath, restartLoginBannerValue, proto.StringVal)
	motdBannerPath := gnmiutils.GetTargetPathWithValue(s.simulator1.Name, restartMotdBannerPath, restartMotdBannerValue, proto.StringVal)

	targets := []string{s.simulator1.Name, s.simulator1.Name}
	paths := []string{restartLoginBannerPath, restartMotdBannerPath}
	values := []string{restartLoginBannerValue, restartMotdBannerValue}

	bannerPaths := gnmiutils.GetTargetPathsWithValues(targets, paths, values)

	// Get the onos-config pod
	pods, err := s.CoreV1().Pods(s.Namespace()).List(s.Context(), metav1.ListOptions{
		LabelSelector: "app=onos,type=config",
	})
	s.NoError(err)
	s.Len(pods.Items, 1)

	// Delete the onos-config pod to restart it
	err = s.CoreV1().Pods(s.Namespace()).Delete(s.Context(), pods.Items[0].Name, metav1.DeleteOptions{})
	s.NoError(err)

	// Set values using onos-config
	var setReq = &gnmiutils.SetRequest{
		Ctx:        s.Context(),
		Client:     gnmiClient,
		Extensions: s.SyncExtension(),
		Encoding:   gnmiapi.Encoding_PROTO,
	}
	setReq.UpdatePaths = tzPath
	setReq.SetOrFail(s.T())
	setReq.UpdatePaths = bannerPaths
	setReq.SetOrFail(s.T())

	// Check that the values were set correctly
	var getConfigReq = &gnmiutils.GetRequest{
		Ctx:        s.Context(),
		Client:     gnmiClient,
		Extensions: s.SyncExtension(),
		Encoding:   gnmiapi.Encoding_PROTO,
	}
	getConfigReq.Paths = tzPath
	getConfigReq.CheckValues(s.T(), restartTzValue)
	getConfigReq.Paths = loginBannerPath
	getConfigReq.CheckValues(s.T(), restartLoginBannerValue)
	getConfigReq.Paths = motdBannerPath
	getConfigReq.CheckValues(s.T(), restartMotdBannerValue)

	// Check that the values are set on the target
	targetGnmiClient := s.NewSimulatorGNMIClientOrFail(s.simulator1.Name)
	var getTargetReq = &gnmiutils.GetRequest{
		Ctx:      s.Context(),
		Client:   targetGnmiClient,
		Encoding: gnmiapi.Encoding_JSON,
	}
	getTargetReq.Paths = tzPath
	getTargetReq.CheckValues(s.T(), restartTzValue)
	getTargetReq.Paths = loginBannerPath
	getTargetReq.CheckValues(s.T(), restartLoginBannerValue)
	getTargetReq.Paths = motdBannerPath
	getTargetReq.CheckValues(s.T(), restartMotdBannerValue)

}
