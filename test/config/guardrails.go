// SPDX-FileCopyrightText: 2020-present Open Networking Foundation <info@opennetworking.org>
//
// SPDX-License-Identifier: Apache-2.0

package config

import (
	gnmiutils "github.com/onosproject/onos-config/test/utils/gnmi"
	"github.com/onosproject/onos-config/test/utils/proto"
	"github.com/openconfig/gnmi/proto/gnmi"
	"github.com/stretchr/testify/assert"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"testing"
)

// testRangeMinViolation tests that setting a range-min smaller than range-max generates an error
func testRangeMinViolation(t *testing.T, setReq *gnmiutils.SetRequest) error {
	const (
		testTargetName = "test1"
		rangeMaxPath   = "/cont1a/list2a[name=rangemin]/range-max"
		rangeMinPath   = "/cont1a/list2a[name=rangemin]/range-min"
	)

	// set range-max to 10
	setReq.UpdatePaths = gnmiutils.GetTargetPathWithValue(testTargetName, rangeMaxPath, "10", proto.IntVal)
	_, _, err := setReq.Set()
	assert.NoError(t, err)

	// set range-min to 11. This should generate an error because 11 is greater than the max range
	setReq.UpdatePaths = gnmiutils.GetTargetPathWithValue(testTargetName, rangeMinPath, "11", proto.IntVal)
	_, _, err = setReq.Set()
	return err
}

// testRangeMaxViolation tests that setting a range-max value that is too big generates an error
func testRangeMaxViolation(t *testing.T, setReq *gnmiutils.SetRequest) error {
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
func testTXPowerViolation(t *testing.T, setReq *gnmiutils.SetRequest) error {
	const (
		testTargetName = "test1"
		TXPower1Path   = "/cont1a/list2a[name=txpower1]/tx-power"
		TXPower2Path   = "/cont1a/list2a[name=txpower2]/tx-power"
	)

	// set a tx-power value of 10 in list2a[txpower1]
	setReq.UpdatePaths = gnmiutils.GetTargetPathWithValue(testTargetName, TXPower1Path, "10", proto.IntVal)
	_, _, err := setReq.Set()
	assert.NoError(t, err)

	// set a tx-power value of 10 in list2a[txpower2]. This should be invalid.
	setReq.UpdatePaths = gnmiutils.GetTargetPathWithValue(testTargetName, TXPower2Path, "10", proto.IntVal)
	_, _, err = setReq.Set()
	return err
}

// TestGuardRails tests GNMI operations that violate guard rails
func (s *TestSuite) TestGuardRails(t *testing.T) {
	const (
		testTargetName         = "test1"
		testDeviceModelName    = "testdevice"
		testDeviceModelVersion = "1.0.0"
	)

	testCases := map[string]struct {
		test     func(t *testing.T, setReq *gnmiutils.SetRequest) error
		expected string
	}{
		"Range Max Violation": {
			test:     testRangeMaxViolation,
			expected: "Unable to unmarshal JSON: error parsing 10000 for schema range-max: value 10000 falls outside the int range [0, 255]",
		},
		"Range Min Violation": {
			test:     testRangeMinViolation,
			expected: "Must statement 'number(./t1:range-min) <= number(./t1:range-max)'",
		},
		"TX Power Violation": {
			test:     testTXPowerViolation,
			expected: "Must statement 'not(t1:list2a[set-contains(following-sibling::t1:list2a/t1:tx-power, t1:tx-power)])'",
		},
	}

	ctx, cancel := gnmiutils.MakeContext()
	defer cancel()

	// create a test device using the 1.0.0 version of the model and register it with topo
	target, err := gnmiutils.NewTargetEntity(testTargetName, testDeviceModelName, testDeviceModelVersion, "127.0.0.1")
	assert.NoError(t, err)
	assert.NotNil(t, target)

	topoClient, err := gnmiutils.NewTopoClient()
	assert.NotNil(t, topoClient)
	assert.Nil(t, err)

	err = topoClient.Create(ctx, target)
	assert.NoError(t, err)

	// make a set request for use in the tests
	gnmiClient := gnmiutils.NewOnosConfigGNMIClientOrFail(ctx, t, gnmiutils.NoRetry)
	setReq := &gnmiutils.SetRequest{
		Ctx:      ctx,
		Client:   gnmiClient,
		Encoding: gnmi.Encoding_PROTO,
	}

	// Run the test cases. Each should return an InvalidArgument error
	for name, testCase := range testCases {
		t.Run(name,
			func(t *testing.T) {
				err = testCase.test(t, setReq)
				assert.Error(t, err)
				if err != nil {
					assert.Equal(t, codes.InvalidArgument, status.Code(err))
					assert.Contains(t, err.Error(), testCase.expected)
				}
			},
		)
	}
	assert.Error(t, err)
	assert.Equal(t, codes.InvalidArgument, status.Code(err))
}
