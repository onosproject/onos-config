// SPDX-FileCopyrightText: 2023-present Intel Corporation
//
// SPDX-License-Identifier: Apache-2.0

package scaling

import (
	"fmt"
	"github.com/divan/num2words"
	topoapi "github.com/onosproject/onos-api/go/onos/topo"
	gnmiutils "github.com/onosproject/onos-config/test/utils/gnmi"
	"github.com/onosproject/onos-config/test/utils/proto"
	gnmiapi "github.com/openconfig/gnmi/proto/gnmi"
	"github.com/stretchr/testify/assert"
	"strconv"
	"testing"
	"time"
)

const (
	configNamePath    = "/config/name"
	enabledPath       = "/config/enabled"
	descriptionPath   = "/config/description"
	mtuPath           = "/config/mtu"
	holdTimeUpPath    = "/hold-time/config/up"
	holdTimeDownPath  = "/hold-time/config/down"
	longIfDescription = "this is a long description of the interface"
	iterateCount      = 100
)

func ifPath(ifIdx int, suffix string) string {
	return fmt.Sprintf("/interfaces/interface[name=if-%d]%s", ifIdx, suffix)
}

func ifDescription(ifIdx int) string {
	return fmt.Sprintf("if-%d (%s): %s", ifIdx, num2words.Convert(ifIdx), longIfDescription)
}

func (s *TestSuite) testHugeNumberOfSets(t *testing.T, encoding gnmiapi.Encoding) {
	ctx, cancel := gnmiutils.MakeContext()
	defer cancel()

	// Make a simulated device
	simulator := gnmiutils.CreateSimulator(ctx, t)
	defer gnmiutils.DeleteSimulator(t, simulator)

	// Wait for config to connect to the target
	ready := gnmiutils.WaitForTargetAvailable(ctx, t, topoapi.ID(simulator.Name()), 1*time.Minute)
	assert.True(t, ready)

	// Make a GNMI client to use for requests
	gnmiClient := gnmiutils.NewOnosConfigGNMIClientOrFail(ctx, t, gnmiutils.NoRetry)

	var setReq = &gnmiutils.SetRequest{
		Ctx:      ctx,
		Client:   gnmiClient,
		Encoding: gnmiapi.Encoding_PROTO,
	}

	t.Logf("doing %d gNMI Sets to build up huge configuration", iterateCount)
	for i := 0; i < iterateCount; i++ {

		setReq.UpdatePaths = []proto.GNMIPath{
			{TargetName: simulator.Name(), Path: ifPath(i, configNamePath), PathDataValue: fmt.Sprintf("if-%d", i), PathDataType: proto.StringVal},
			{TargetName: simulator.Name(), Path: ifPath(i, descriptionPath), PathDataValue: ifDescription(i), PathDataType: proto.StringVal},
			{TargetName: simulator.Name(), Path: ifPath(i, enabledPath), PathDataValue: "false", PathDataType: proto.BoolVal},
			{TargetName: simulator.Name(), Path: ifPath(i, mtuPath), PathDataValue: strconv.FormatInt(int64(i), 10), PathDataType: proto.IntVal},
			{TargetName: simulator.Name(), Path: ifPath(i, holdTimeUpPath), PathDataValue: strconv.FormatInt(int64(i*1000), 10), PathDataType: proto.IntVal},
			{TargetName: simulator.Name(), Path: ifPath(i, holdTimeDownPath), PathDataValue: strconv.FormatInt(int64(i*1000+1), 10), PathDataType: proto.IntVal},
		}

		setReq.SetOrFail(t)
	}

	getConfigReq := &gnmiutils.GetRequest{
		Ctx:      ctx,
		Client:   gnmiClient,
		Encoding: encoding,
	}

	// Check that the name value was set correctly
	getConfigReq.Paths = []proto.GNMIPath{
		{TargetName: simulator.Name(), Path: "/interfaces/interface[name=*]"},
	}
	getConfigReq.CountValues(t, iterateCount*6)
}

// TestHugeNumberOfSets tests a huge number of sets to a single device, to test the limits of configuration
func (s *TestSuite) TestHugeNumberOfSets(t *testing.T) {
	t.Run("TestHugeNumberOfSets PROTO",
		func(t *testing.T) {
			s.testHugeNumberOfSets(t, gnmiapi.Encoding_PROTO)
		})
}
