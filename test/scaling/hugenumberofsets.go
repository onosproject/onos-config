// SPDX-FileCopyrightText: 2023-present Intel Corporation
//
// SPDX-License-Identifier: Apache-2.0

package scaling

import (
	"context"
	"fmt"
	"github.com/divan/num2words"
	topoapi "github.com/onosproject/onos-api/go/onos/topo"
	"github.com/onosproject/onos-config/test"
	gnmiutils "github.com/onosproject/onos-config/test/utils/gnmi"
	"github.com/onosproject/onos-config/test/utils/proto"
	gnmiapi "github.com/openconfig/gnmi/proto/gnmi"
	"strconv"
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

func (s *TestSuite) testHugeNumberOfSets(ctx context.Context, encoding gnmiapi.Encoding) {
	// Wait for config to connect to the target
	ready := s.WaitForTargetAvailable(ctx, topoapi.ID(s.simulator), 1*time.Minute)
	s.True(ready)

	// Make a GNMI client to use for requests
	gnmiClient := s.NewOnosConfigGNMIClientOrFail(ctx, test.NoRetry)

	var setReq = &gnmiutils.SetRequest{
		Ctx:      ctx,
		Client:   gnmiClient,
		Encoding: gnmiapi.Encoding_PROTO,
	}

	s.T().Logf("doing %d gNMI Sets to build up huge configuration", iterateCount)
	for i := 0; i < iterateCount; i++ {

		setReq.UpdatePaths = []proto.GNMIPath{
			{TargetName: s.simulator, Path: ifPath(i, configNamePath), PathDataValue: fmt.Sprintf("if-%d", i), PathDataType: proto.StringVal},
			{TargetName: s.simulator, Path: ifPath(i, descriptionPath), PathDataValue: ifDescription(i), PathDataType: proto.StringVal},
			{TargetName: s.simulator, Path: ifPath(i, enabledPath), PathDataValue: "false", PathDataType: proto.BoolVal},
			{TargetName: s.simulator, Path: ifPath(i, mtuPath), PathDataValue: strconv.FormatInt(int64(i), 10), PathDataType: proto.IntVal},
			{TargetName: s.simulator, Path: ifPath(i, holdTimeUpPath), PathDataValue: strconv.FormatInt(int64(i*1000), 10), PathDataType: proto.IntVal},
			{TargetName: s.simulator, Path: ifPath(i, holdTimeDownPath), PathDataValue: strconv.FormatInt(int64(i*1000+1), 10), PathDataType: proto.IntVal},
		}

		setReq.SetOrFail(s.T())
	}

	getConfigReq := &gnmiutils.GetRequest{
		Ctx:      ctx,
		Client:   gnmiClient,
		Encoding: encoding,
	}

	// Check that the name value was set correctly
	getConfigReq.Paths = []proto.GNMIPath{
		{TargetName: s.simulator, Path: "/interfaces/interface[name=*]"},
	}
	getConfigReq.CountValues(s.T(), iterateCount*6)
}

// TestHugeNumberOfSets tests a huge number of sets to a single device, to test the limits of configuration
func (s *TestSuite) TestHugeNumberOfSets(ctx context.Context) {
	s.Run("TestHugeNumberOfSets PROTO", func() {
		s.testHugeNumberOfSets(ctx, gnmiapi.Encoding_PROTO)
	})
}
