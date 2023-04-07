// SPDX-FileCopyrightText: 2023-present Intel Corporation
//
// SPDX-License-Identifier: Apache-2.0

package scaling

import (
	"fmt"
	"github.com/divan/num2words"
	topoapi "github.com/onosproject/onos-api/go/onos/topo"
	"github.com/onosproject/onos-config/test"
	gnmiutils "github.com/onosproject/onos-config/test/utils/gnmi"
	"github.com/onosproject/onos-config/test/utils/proto"
	gnmiapi "github.com/openconfig/gnmi/proto/gnmi"
	"strconv"
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

func (s *TestSuite) testHugeNumberOfSets(encoding gnmiapi.Encoding) {
	// Wait for config to connect to the target
	ready := s.WaitForTargetAvailable(topoapi.ID(s.simulator.Name))
	s.True(ready)

	// Make a GNMI client to use for requests
	gnmiClient := s.NewOnosConfigGNMIClientOrFail(test.NoRetry)

	var setReq = &gnmiutils.SetRequest{
		Ctx:      s.Context(),
		Client:   gnmiClient,
		Encoding: gnmiapi.Encoding_PROTO,
	}

	s.T().Logf("doing %d gNMI Sets to build up huge configuration", iterateCount)
	for i := 0; i < iterateCount; i++ {

		setReq.UpdatePaths = []proto.GNMIPath{
			{TargetName: s.simulator.Name, Path: ifPath(i, configNamePath), PathDataValue: fmt.Sprintf("if-%d", i), PathDataType: proto.StringVal},
			{TargetName: s.simulator.Name, Path: ifPath(i, descriptionPath), PathDataValue: ifDescription(i), PathDataType: proto.StringVal},
			{TargetName: s.simulator.Name, Path: ifPath(i, enabledPath), PathDataValue: "false", PathDataType: proto.BoolVal},
			{TargetName: s.simulator.Name, Path: ifPath(i, mtuPath), PathDataValue: strconv.FormatInt(int64(i), 10), PathDataType: proto.IntVal},
			{TargetName: s.simulator.Name, Path: ifPath(i, holdTimeUpPath), PathDataValue: strconv.FormatInt(int64(i*1000), 10), PathDataType: proto.IntVal},
			{TargetName: s.simulator.Name, Path: ifPath(i, holdTimeDownPath), PathDataValue: strconv.FormatInt(int64(i*1000+1), 10), PathDataType: proto.IntVal},
		}

		setReq.SetOrFail(s.T())
	}

	getConfigReq := &gnmiutils.GetRequest{
		Ctx:      s.Context(),
		Client:   gnmiClient,
		Encoding: encoding,
	}

	// Check that the name value was set correctly
	getConfigReq.Paths = []proto.GNMIPath{
		{TargetName: s.simulator.Name, Path: "/interfaces/interface[name=*]"},
	}
	getConfigReq.CountValues(s.T(), iterateCount*6)
}

// TestHugeNumberOfSets tests a huge number of sets to a single device, to test the limits of configuration
func (s *TestSuite) TestHugeNumberOfSets() {
	s.Run("TestHugeNumberOfSets PROTO", func() {
		s.testHugeNumberOfSets(gnmiapi.Encoding_PROTO)
	})
}
