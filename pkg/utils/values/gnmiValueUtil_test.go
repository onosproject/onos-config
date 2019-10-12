// Copyright 2019-present Open Networking Foundation.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

// Package utils test various gNMI value manipulation facilities.

package values

import (
	types "github.com/onosproject/onos-config/pkg/types/change/device"
	pb "github.com/openconfig/gnmi/proto/gnmi"
	"gotest.tools/assert"
	"reflect"
	"testing"
)

const testString = "This is a test"
const (
	testNegativeInt = -9223372036854775808
	testPositiveInt = 9223372036854775807
	testMaxUint     = uint(18446744073709551615)
)

////////////////////////////////////////////////////////////////////////////////
// gNMI format to Native
////////////////////////////////////////////////////////////////////////////////

func Test_GnmiStringToNative(t *testing.T) {
	gnmiValue := pb.TypedValue_StringVal{StringVal: testString}
	nativeType, err := GnmiTypedValueToNativeType(&pb.TypedValue{Value: &gnmiValue})
	assert.NilError(t, err)

	nativeString := (*types.TypedString)(nativeType)
	assert.Equal(t, nativeString.String(), testString)
}

func Test_GnmiIntToNative(t *testing.T) {
	gnmiValue := pb.TypedValue_IntVal{IntVal: testNegativeInt}
	nativeType, err := GnmiTypedValueToNativeType(&pb.TypedValue{Value: &gnmiValue})
	assert.NilError(t, err)

	nativeInt64 := (*types.TypedInt64)(nativeType)
	assert.Equal(t, nativeInt64.Int(), testNegativeInt)
}

func Test_GnmiUintToNative(t *testing.T) {
	gnmiValue := pb.TypedValue_UintVal{UintVal: uint64(testMaxUint)}
	nativeType, err := GnmiTypedValueToNativeType(&pb.TypedValue{Value: &gnmiValue})
	assert.NilError(t, err)

	nativeUint64 := (*types.TypedUint64)(nativeType)
	assert.Equal(t, nativeUint64.Uint(), testMaxUint)
}

func Test_GnmiBoolToNative(t *testing.T) {
	gnmiValue := pb.TypedValue_BoolVal{BoolVal: true}
	nativeType, err := GnmiTypedValueToNativeType(&pb.TypedValue{Value: &gnmiValue})
	assert.NilError(t, err)

	nativeBool := (*types.TypedBool)(nativeType)
	assert.Equal(t, nativeBool.Bool(), true)
}

var intTestValue = &pb.TypedValue{
	Value: &pb.TypedValue_LeaflistVal{
		LeaflistVal: &pb.ScalarArray{
			Element: []*pb.TypedValue{
				{Value: &pb.TypedValue_IntVal{IntVal: 100}},
				{Value: &pb.TypedValue_IntVal{IntVal: 101}},
				{Value: &pb.TypedValue_IntVal{IntVal: 102}},
				{Value: &pb.TypedValue_IntVal{IntVal: 103}},
			},
		},
	},
}

var uintTestValue = &pb.TypedValue{
	Value: &pb.TypedValue_LeaflistVal{
		LeaflistVal: &pb.ScalarArray{
			Element: []*pb.TypedValue{
				{Value: &pb.TypedValue_UintVal{UintVal: 100}},
				{Value: &pb.TypedValue_UintVal{UintVal: 101}},
				{Value: &pb.TypedValue_UintVal{UintVal: 102}},
				{Value: &pb.TypedValue_UintVal{UintVal: 103}},
			},
		},
	},
}

var decimalTestValue = &pb.TypedValue{
	Value: &pb.TypedValue_LeaflistVal{
		LeaflistVal: &pb.ScalarArray{
			Element: []*pb.TypedValue{
				{
					Value: &pb.TypedValue_DecimalVal{
						DecimalVal: &pb.Decimal64{
							Digits:    6,
							Precision: 0,
						},
					},
				},
			},
		},
	},
}

var booleanTestValue = &pb.TypedValue{
	Value: &pb.TypedValue_LeaflistVal{
		LeaflistVal: &pb.ScalarArray{
			Element: []*pb.TypedValue{
				{Value: &pb.TypedValue_BoolVal{BoolVal: true}},
				{Value: &pb.TypedValue_BoolVal{BoolVal: false}},
				{Value: &pb.TypedValue_BoolVal{BoolVal: true}},
				{Value: &pb.TypedValue_BoolVal{BoolVal: false}},
			},
		},
	},
}

var floatTestValue = &pb.TypedValue{
	Value: &pb.TypedValue_LeaflistVal{
		LeaflistVal: &pb.ScalarArray{
			Element: []*pb.TypedValue{
				{Value: &pb.TypedValue_FloatVal{FloatVal: 1.0}},
				{Value: &pb.TypedValue_FloatVal{FloatVal: 2.0}},
				{Value: &pb.TypedValue_FloatVal{FloatVal: 3.0}},
				{Value: &pb.TypedValue_FloatVal{FloatVal: 4.0}},
			},
		},
	},
}

var bytesTestValue = &pb.TypedValue{
	Value: &pb.TypedValue_LeaflistVal{
		LeaflistVal: &pb.ScalarArray{
			Element: []*pb.TypedValue{
				{Value: &pb.TypedValue_BytesVal{BytesVal: []byte("abc")}},
				{Value: &pb.TypedValue_BytesVal{BytesVal: []byte("def")}},
				{Value: &pb.TypedValue_BytesVal{BytesVal: []byte("ghi")}},
				{Value: &pb.TypedValue_BytesVal{BytesVal: []byte("jkl")}},
			},
		},
	},
}

var stringTestValue = &pb.TypedValue{
	Value: &pb.TypedValue_LeaflistVal{
		LeaflistVal: &pb.ScalarArray{
			Element: []*pb.TypedValue{
				{Value: &pb.TypedValue_StringVal{StringVal: "abc"}},
				{Value: &pb.TypedValue_StringVal{StringVal: "def"}},
				{Value: &pb.TypedValue_StringVal{StringVal: "ghi"}},
				{Value: &pb.TypedValue_StringVal{StringVal: "jkl"}},
			},
		},
	},
}

func Test_Leaflists(t *testing.T) {
	testCases := []struct {
		description  string
		expectedType types.ValueType
		testValue    *pb.TypedValue
	}{
		{description: "Int", expectedType: types.ValueType_LEAFLIST_INT, testValue: intTestValue},
		{description: "Uint", expectedType: types.ValueType_LEAFLIST_UINT, testValue: uintTestValue},
		{description: "Decimal", expectedType: types.ValueType_LEAFLIST_DECIMAL, testValue: decimalTestValue},
		{description: "Boolean", expectedType: types.ValueType_LEAFLIST_BOOL, testValue: booleanTestValue},
		{description: "Float", expectedType: types.ValueType_LEAFLIST_FLOAT, testValue: floatTestValue},
		{description: "Bytes", expectedType: types.ValueType_LEAFLIST_BYTES, testValue: bytesTestValue},
		{description: "Strings", expectedType: types.ValueType_LEAFLIST_STRING, testValue: stringTestValue},
	}

	for _, testCase := range testCases {
		nativeType, err := GnmiTypedValueToNativeType(testCase.testValue)
		assert.NilError(t, err)
		assert.Assert(t, nativeType != nil)
		assert.Equal(t, nativeType.Type, testCase.expectedType)

		convertedValue, convertedErr := NativeTypeToGnmiTypedValue(nativeType)
		assert.NilError(t, convertedErr)
		assert.Assert(t, reflect.DeepEqual(*convertedValue, *testCase.testValue), "%s", testCase.description)
	}
}

////////////////////////////////////////////////////////////////////////////////
// Native format to gnmi
////////////////////////////////////////////////////////////////////////////////

func Test_NativeStringToGnmi(t *testing.T) {
	nativeString := types.NewTypedValueString(testString)
	gnmiString, err := NativeTypeToGnmiTypedValue(nativeString)
	assert.NilError(t, err)
	_, ok := gnmiString.Value.(*pb.TypedValue_StringVal)
	assert.Assert(t, ok)

	assert.Equal(t, gnmiString.GetStringVal(), testString)
}

func Test_NativeIntToGnmi(t *testing.T) {
	nativeInt := types.NewTypedValueInt64(testPositiveInt)
	gnmiInt, err := NativeTypeToGnmiTypedValue(nativeInt)
	assert.NilError(t, err)
	_, ok := gnmiInt.Value.(*pb.TypedValue_IntVal)
	assert.Assert(t, ok)

	assert.Equal(t, int(gnmiInt.GetIntVal()), testPositiveInt)
}

func Test_NativeUintToGnmi(t *testing.T) {
	nativeUint := types.NewTypedValueUint64(testMaxUint)
	gnmiUint, err := NativeTypeToGnmiTypedValue(nativeUint)
	assert.NilError(t, err)
	_, ok := gnmiUint.Value.(*pb.TypedValue_UintVal)
	assert.Assert(t, ok)

	assert.Equal(t, uint(gnmiUint.GetUintVal()), testMaxUint)
}

func Test_NativeBoolToGnmi(t *testing.T) {
	nativeBool := types.NewTypedValueBool(true)
	gnmiBool, err := NativeTypeToGnmiTypedValue(nativeBool)
	assert.NilError(t, err)
	_, ok := gnmiBool.Value.(*pb.TypedValue_BoolVal)
	assert.Assert(t, ok)

	assert.Equal(t, gnmiBool.GetBoolVal(), true)
}
