// SPDX-FileCopyrightText: 2020-present Open Networking Foundation <info@opennetworking.org>
//
// SPDX-License-Identifier: Apache-2.0

package config

import (
	"context"
	"github.com/onosproject/helmit/pkg/helm"
	"github.com/onosproject/helmit/pkg/kubernetes"
	hautils "github.com/onosproject/onos-config/test/utils/ha"
	"github.com/onosproject/onos-config/test/utils/proto"
	gnmiapi "github.com/openconfig/gnmi/proto/gnmi"
	"github.com/stretchr/testify/assert"
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
	_ = checkTarget(ctx, t, target, targetPathsForGet, true)

	// Crash the target simulator
	client := kubernetes.NewForReleaseOrDie(target)
	pods, err := client.CoreV1().Pods().List(ctx)
	assert.NoError(t, err)
	assert.Len(t, pods, 1)
	hautils.CrashPodOrFail(t, pods[0])
	gnmiutils.WaitForTargetUnavailable(ctx, t, topo.ID(target.Name()), time.Minute)

	// Wait for it to become available
	gnmiutils.WaitForTargetAvailable(ctx, t, topo.ID(target.Name()), time.Minute)

	// Settle the race between reapplying the changes to the freshly restarted target and the subsequent checks.
	for i := 0; i < 30; i++ {
		if ok := checkTarget(ctx, t, target, targetPathsForGet, false); ok {
			break
		}
		time.Sleep(2 * time.Second)
	}

	// Make sure the configuration has been re-applied to the target
	_ = checkTarget(ctx, t, target, targetPathsForGet, true)
}

// Check that the crashedTargetValues are set on the target
func checkTarget(ctx context.Context, t *testing.T, target *helm.HelmRelease, targetPathsForGet []proto.TargetPath, enforce bool) bool {
	targetGnmiClient := gnmiutils.NewSimulatorGNMIClientOrFail(ctx, t, target)

	var targetGetReq = &gnmiutils.GetRequest{
		Ctx:        ctx,
		Client:     targetGnmiClient,
		Encoding:   gnmiapi.Encoding_JSON,
		Extensions: gnmiutils.SyncExtension(t),
	}
	targetGetReq.Paths = targetPathsForGet[0:1]

	if !enforce {
		// If we're not enforcing, simply return true if we got the expected value for the first path
		paths, err := targetGetReq.Get()
		return err == nil && len(paths) == 1 && paths[0].PathDataValue == crashedTargetValue1
	}
	targetGetReq.CheckValues(t, crashedTargetValue1)
	targetGetReq.Paths = targetPathsForGet[1:2]
	targetGetReq.CheckValues(t, crashedTargetValue2)
	return false
}
