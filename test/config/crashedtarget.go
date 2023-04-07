// SPDX-FileCopyrightText: 2020-present Open Networking Foundation <info@opennetworking.org>
//
// SPDX-License-Identifier: Apache-2.0

package config

import (
	"fmt"
	"github.com/onosproject/onos-config/test"
	"github.com/onosproject/onos-config/test/utils/proto"
	gnmiapi "github.com/openconfig/gnmi/proto/gnmi"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
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
func (s *TestSuite) TestCrashedTarget() {
	// Wait for the simulator to become available
	s.WaitForTargetAvailable(topo.ID(s.simulator1.Name))

	// Set up crashedTargetPaths to configure
	targets := []string{s.simulator1.Name}
	targetPathsForGet := gnmiutils.GetTargetPaths(targets, crashedTargetPaths)

	// Make a GNMI client to use for requests
	gnmiClient := s.NewOnosConfigGNMIClientOrFail(test.NoRetry)

	// Set initial crashedTargetValues
	targetPathsForInit := gnmiutils.GetTargetPathsWithValues(targets, crashedTargetPaths, crashedTargetValues)

	var setReq = &gnmiutils.SetRequest{
		Ctx:         s.Context(),
		Client:      gnmiClient,
		Encoding:    gnmiapi.Encoding_PROTO,
		Extensions:  s.SyncExtension(),
		UpdatePaths: targetPathsForInit,
	}
	setReq.SetOrFail(s.T())

	// Make sure the configuration has been applied to both onos-config
	var getReq = &gnmiutils.GetRequest{
		Ctx:        s.Context(),
		Client:     gnmiClient,
		Encoding:   gnmiapi.Encoding_PROTO,
		Extensions: s.SyncExtension(),
	}
	targetPath1 := gnmiutils.GetTargetPath(s.simulator1.Name, crashedTargetPath1)
	targetPath2 := gnmiutils.GetTargetPath(s.simulator1.Name, crashedTargetPath2)

	// Check that the crashedTargetValues were set correctly
	getReq.Paths = targetPath1
	getReq.CheckValues(s.T(), crashedTargetValue1)
	getReq.Paths = targetPath2
	getReq.CheckValues(s.T(), crashedTargetValue2)

	// ... and the target
	_ = s.checkTarget(s.simulator1.Name, targetPathsForGet, true)

	// Crash the target simulator
	pods, err := s.CoreV1().Pods(s.Namespace()).List(s.Context(), metav1.ListOptions{
		LabelSelector: fmt.Sprintf("name=%s-device-simulator", s.simulator1.Name),
	})
	s.NoError(err)
	s.Len(pods.Items, 1)

	// Delete the simulator pod
	err = s.CoreV1().Pods(s.Namespace()).Delete(s.Context(), pods.Items[0].Name, metav1.DeleteOptions{})
	s.NoError(err)

	s.WaitForTargetUnavailable(topo.ID(s.simulator1.Name))

	// Wait for it to become available
	s.WaitForTargetAvailable(topo.ID(s.simulator1.Name))

	// Settle the race between reapplying the changes to the freshly restarted target and the subsequent checks.
	for i := 0; i < 30; i++ {
		if ok := s.checkTarget(s.simulator1.Name, targetPathsForGet, false); ok {
			break
		}
		time.Sleep(2 * time.Second)
	}

	// Make sure the configuration has been re-applied to the target
	_ = s.checkTarget(s.simulator1.Name, targetPathsForGet, true)
}

// Check that the crashedTargetValues are set on the target
func (s *TestSuite) checkTarget(target string, targetPathsForGet []proto.GNMIPath, enforce bool) bool {
	targetGnmiClient := s.NewSimulatorGNMIClientOrFail(target)

	var targetGetReq = &gnmiutils.GetRequest{
		Ctx:        s.Context(),
		Client:     targetGnmiClient,
		Encoding:   gnmiapi.Encoding_JSON,
		Extensions: s.SyncExtension(),
	}
	targetGetReq.Paths = targetPathsForGet[0:1]

	if !enforce {
		// If we're not enforcing, simply return true if we got the expected value for the first path
		paths, err := targetGetReq.Get()
		return err == nil && len(paths) == 1 && paths[0].PathDataValue == crashedTargetValue1
	}
	targetGetReq.CheckValues(s.T(), crashedTargetValue1)
	targetGetReq.Paths = targetPathsForGet[1:2]
	targetGetReq.CheckValues(s.T(), crashedTargetValue2)
	return false
}
