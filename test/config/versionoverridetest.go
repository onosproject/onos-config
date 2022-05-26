// SPDX-FileCopyrightText: 2020-present Open Networking Foundation <info@opennetworking.org>
//
// SPDX-License-Identifier: Apache-2.0

package config

import (
	configapi "github.com/onosproject/onos-api/go/onos/config/v2"
	"github.com/onosproject/onos-config/pkg/utils"
	gnmiapi "github.com/openconfig/gnmi/proto/gnmi"
	"testing"
	"time"

	"github.com/onosproject/onos-api/go/onos/topo"

	gnmiutils "github.com/onosproject/onos-config/test/utils/gnmi"
	"github.com/stretchr/testify/assert"
)

// TestVersionOverride tests using version override extension
func (s *TestSuite) TestVersionOverride(t *testing.T) {
	const (
		value1 = "test-motd-banner"
		path1  = "/system/config/motd-banner"
		value2 = "test-login-banner"
		path2  = "/system/config/login-banner"
	)

	var (
		paths  = []string{path1, path2}
		values = []string{value1, value2}
	)

	ctx, cancel := gnmiutils.MakeContext()
	defer cancel()

	// Create the first target simulator
	target := gnmiutils.CreateSimulator(ctx, t)
	defer gnmiutils.DeleteSimulator(t, target)

	// Change topo entity for the simulator to non-existent type/version
	err := gnmiutils.UpdateTargetTypeVersion(ctx, topo.ID(target.Name()), "testdevice", "1.0.0")
	assert.NoError(t, err)

	// Wait for config to connect to the first simulator
	gnmiutils.WaitForTargetAvailable(ctx, t, topo.ID(target.Name()), time.Minute)

	// Make a GNMI client to use for requests
	gnmiClient := gnmiutils.NewOnosConfigGNMIClientOrFail(ctx, t, gnmiutils.NoRetry)

	// Set up paths for the set
	targets := []string{target.Name()}

	targetPaths := gnmiutils.GetTargetPathsWithValues(targets, paths, values)
	targetPathsForGet := gnmiutils.GetTargetPaths(targets, paths)

	// Formulate extensions with sync and with target version override
	extensions := gnmiutils.SyncExtension(t)
	ttv, err := utils.TargetVersionOverrideExtension(configapi.TargetID(target.Name()), "devicesim", "1.0.0")
	assert.NoError(t, err)
	extensions = append(extensions, ttv)

	var setReq = &gnmiutils.SetRequest{
		Ctx:         ctx,
		Client:      gnmiClient,
		Encoding:    gnmiapi.Encoding_PROTO,
		Extensions:  extensions,
		UpdatePaths: targetPaths,
	}
	setReq.SetOrFail(t)

	// Make sure the configuration has been applied to the target
	targetGnmiClient := gnmiutils.NewSimulatorGNMIClientOrFail(ctx, t, target)

	var targetGetReq = &gnmiutils.GetRequest{
		Ctx:        ctx,
		Client:     targetGnmiClient,
		Encoding:   gnmiapi.Encoding_JSON,
		Extensions: gnmiutils.SyncExtension(t),
	}
	targetGetReq.Paths = targetPathsForGet[0:1]
	targetGetReq.CheckValues(t, value1)
	targetGetReq.Paths = targetPathsForGet[1:2]
	targetGetReq.CheckValues(t, value2)
}
