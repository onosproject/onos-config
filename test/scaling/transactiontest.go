// SPDX-FileCopyrightText: 2023-present Intel Corporation
//
// SPDX-License-Identifier: Apache-2.0

package scaling

import (
	"fmt"
	gnmiapi "github.com/openconfig/gnmi/proto/gnmi"
	"testing"
	"time"

	"github.com/onosproject/onos-api/go/onos/topo"

	gnmiutils "github.com/onosproject/onos-config/test/utils/gnmi"
	"github.com/stretchr/testify/assert"
)

// TestTransaction tests setting multiple paths in a single request and rolling it back
func (s *TestSuite) testTransaction(t *testing.T, encoding gnmiapi.Encoding) {
	const (
		value1     = "test-motd-banner"
		path1      = "/system/config/motd-banner"
		value2     = "test-login-banner"
		path2      = "/system/config/login-banner"
		initValue1 = "1"
		initValue2 = "2"
	)

	var (
		paths         = []string{path1, path2}
		initialValues = []string{initValue1, initValue2}
	)

	ctx, cancel := gnmiutils.MakeContext()
	defer cancel()

	// Create the first target simulator
	target1 := gnmiutils.CreateSimulator(ctx, t)
	defer gnmiutils.DeleteSimulator(t, target1)

	// Wait for config to connect to the first simulator
	gnmiutils.WaitForTargetAvailable(ctx, t, topo.ID(target1.Name()), time.Minute)

	// Create the second target simulator
	target2 := gnmiutils.CreateSimulator(ctx, t)
	defer gnmiutils.DeleteSimulator(t, target2)

	// Wait for config to connect to the second simulator
	gnmiutils.WaitForTargetAvailable(ctx, t, topo.ID(target2.Name()), time.Minute)

	// Set up paths for the two targets
	targets := []string{target1.Name(), target2.Name()}

	// Make a GNMI client to use for requests
	gnmiClient := gnmiutils.NewOnosConfigGNMIClientOrFail(ctx, t, gnmiutils.NoRetry)

	// Set initial values
	targetPathsForInit := gnmiutils.GetTargetPathsWithValues(targets, paths, initialValues)

	var setReq = &gnmiutils.SetRequest{
		Ctx:         ctx,
		Client:      gnmiClient,
		Encoding:    gnmiapi.Encoding_PROTO,
		Extensions:  gnmiutils.SyncExtension(t),
		UpdatePaths: targetPathsForInit,
	}
	setError := setReq.SetExpectFail(t)

	assert.Equal(t, fmt.Sprintf("rpc error: code = InvalidArgument desc = "+
		"gNMI Set must contain only 1 target. Found: 2. GNMI_SET_SIZE_LIMIT=%d", gnmiSetLimitForTest), setError.Error())
	t.Logf("Successfully prevented gNMI Set with more than 1 target, when GNMI_SET_SIZE_LIMIT = %d", gnmiSetLimitForTest)
}

// TestTransaction tests setting multiple paths in a single request and rolling it back
func (s *TestSuite) TestTransaction(t *testing.T) {
	t.Run("TestTransaction limited",
		func(t *testing.T) {
			s.testTransaction(t, gnmiapi.Encoding_PROTO)
		})
}
