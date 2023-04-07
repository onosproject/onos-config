// SPDX-FileCopyrightText: 2020-present Open Networking Foundation <info@opennetworking.org>
//
// SPDX-License-Identifier: Apache-2.0

package config

import (
	"github.com/onosproject/onos-config/test"
	gnmiutils "github.com/onosproject/onos-config/test/utils/gnmi"
	"github.com/onosproject/onos-config/test/utils/proto"
	"github.com/openconfig/gnmi/proto/gnmi"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// testRangeMinViolation tests that setting a range-min smaller than range-max generates an error
func (s *TestSuite) testRangeMinViolation(setReq *gnmiutils.SetRequest) error {
	const (
		testTargetName = "test1"
		rangeMaxPath   = "/cont1a/list2a[name=rangemin]/range-max"
		rangeMinPath   = "/cont1a/list2a[name=rangemin]/range-min"
	)

	// set range-max to 10
	setReq.UpdatePaths = gnmiutils.GetTargetPathWithValue(testTargetName, rangeMaxPath, "10", proto.IntVal)
	_, _, err := setReq.Set()
	s.NoError(err)

	// set range-min to 11. This should generate an error because 11 is greater than the max range
	setReq.UpdatePaths = gnmiutils.GetTargetPathWithValue(testTargetName, rangeMinPath, "11", proto.IntVal)
	_, _, err = setReq.Set()
	return err
}

// testRangeMaxViolation tests that setting a range-max value that is too big generates an error
func (s *TestSuite) testRangeMaxViolation(setReq *gnmiutils.SetRequest) error {
	const (
		testTargetName = "test1"
		rangeMaxPath   = "/cont1a/list2a[name=rangemax]/range-max"
	)

	// Set an illegal value for range max. range-max must fit in 8 bits.
	setReq.UpdatePaths = gnmiutils.GetTargetPathWithValue(testTargetName, rangeMaxPath, "10000", proto.IntVal)
	_, _, err := setReq.Set()
	return err
}

// testTXPowerViolation tests that two tx-power cannot be the same
func (s *TestSuite) testTXPowerViolation(setReq *gnmiutils.SetRequest) error {
	const (
		testTargetName = "test1"
		TXPower1Path   = "/cont1a/list2a[name=txpower1]/tx-power"
		TXPower2Path   = "/cont1a/list2a[name=txpower2]/tx-power"
	)

	// set a tx-power value of 10 in list2a[txpower1]
	setReq.UpdatePaths = gnmiutils.GetTargetPathWithValue(testTargetName, TXPower1Path, "10", proto.IntVal)
	_, _, err := setReq.Set()
	s.NoError(err)

	// set a tx-power value of 10 in list2a[txpower2]. This should be invalid.
	setReq.UpdatePaths = gnmiutils.GetTargetPathWithValue(testTargetName, TXPower2Path, "10", proto.IntVal)
	_, _, err = setReq.Set()
	return err
}

// TestGuardRails tests GNMI operations that violate guard rails
func (s *TestSuite) TestGuardRails() {
	const (
		testTargetName         = "test1"
		testDeviceModelName    = "testdevice"
		testDeviceModelVersion = "1.0.x"
	)

	testCases := map[string]struct {
		test     func(setReq *gnmiutils.SetRequest) error
		expected string
	}{
		"Range Max Violation": {
			test:     s.testRangeMaxViolation,
			expected: "Unable to unmarshal JSON: error parsing 10000 for schema range-max: value 10000 falls outside the int range [0, 255]",
		},
		"Range Min Violation": {
			test:     s.testRangeMinViolation,
			expected: "Must statement 'number(./t1:range-min) <= number(./t1:range-max)'",
		},
		"TX Power Violation": {
			test:     s.testTXPowerViolation,
			expected: "Must statement 'not(t1:list2a[set-contains(following-sibling::t1:list2a/t1:tx-power, t1:tx-power)])'",
		},
	}

	// create a test device using the 1.0.0 version of the model and register it with topo
	target, err := s.NewTargetEntity(testTargetName, testDeviceModelName, testDeviceModelVersion, "127.0.0.1")
	s.NoError(err)
	s.NotNil(target)

	topoClient, err := s.NewTopoClient()
	s.NotNil(topoClient)
	s.Nil(err)

	err = topoClient.Create(s.Context(), target)
	s.NoError(err)

	// make a set request for use in the tests
	gnmiClient := s.NewOnosConfigGNMIClientOrFail(test.NoRetry)
	setReq := &gnmiutils.SetRequest{
		Ctx:      s.Context(),
		Client:   gnmiClient,
		Encoding: gnmi.Encoding_PROTO,
	}

	// Run the test cases. Each should return an InvalidArgument error
	for name, testCase := range testCases {
		s.Run(name, func() {
			err = testCase.test(setReq)
			s.Error(err)
			if err != nil {
				s.Equal(codes.InvalidArgument, status.Code(err))
				s.Contains(err.Error(), testCase.expected)
			}
		},
		)
	}
	s.Error(err)
	s.Equal(codes.InvalidArgument, status.Code(err))
}
