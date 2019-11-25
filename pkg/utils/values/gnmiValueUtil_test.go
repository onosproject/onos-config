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
	devicechange "github.com/onosproject/onos-config/api/types/change/device"
	"github.com/openconfig/gnmi/proto/gnmi"
	"gotest.tools/assert"
	"reflect"
	"strings"
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
	gnmiValue := gnmi.TypedValue_StringVal{StringVal: testString}
	nativeType, err := GnmiTypedValueToNativeType(&gnmi.TypedValue{Value: &gnmiValue})
	assert.NilError(t, err)

	nativeString := (*devicechange.TypedString)(nativeType)
	assert.Equal(t, nativeString.String(), testString)
}

func Test_GnmiIntToNative(t *testing.T) {
	gnmiValue := gnmi.TypedValue_IntVal{IntVal: testNegativeInt}
	nativeType, err := GnmiTypedValueToNativeType(&gnmi.TypedValue{Value: &gnmiValue})
	assert.NilError(t, err)

	nativeInt64 := (*devicechange.TypedInt64)(nativeType)
	assert.Equal(t, nativeInt64.Int(), testNegativeInt)
}

func Test_GnmiUintToNative(t *testing.T) {
	gnmiValue := gnmi.TypedValue_UintVal{UintVal: uint64(testMaxUint)}
	nativeType, err := GnmiTypedValueToNativeType(&gnmi.TypedValue{Value: &gnmiValue})
	assert.NilError(t, err)

	nativeUint64 := (*devicechange.TypedUint64)(nativeType)
	assert.Equal(t, nativeUint64.Uint(), testMaxUint)
}

func Test_GnmiBoolToNative(t *testing.T) {
	gnmiValue := gnmi.TypedValue_BoolVal{BoolVal: true}
	nativeType, err := GnmiTypedValueToNativeType(&gnmi.TypedValue{Value: &gnmiValue})
	assert.NilError(t, err)

	nativeBool := (*devicechange.TypedBool)(nativeType)
	assert.Equal(t, nativeBool.Bool(), true)
}

var intListTestValue = &gnmi.TypedValue{
	Value: &gnmi.TypedValue_LeaflistVal{
		LeaflistVal: &gnmi.ScalarArray{
			Element: []*gnmi.TypedValue{
				{Value: &gnmi.TypedValue_IntVal{IntVal: 100}},
				{Value: &gnmi.TypedValue_IntVal{IntVal: 101}},
				{Value: &gnmi.TypedValue_IntVal{IntVal: 102}},
				{Value: &gnmi.TypedValue_IntVal{IntVal: 103}},
			},
		},
	},
}

var uintListTestValue = &gnmi.TypedValue{
	Value: &gnmi.TypedValue_LeaflistVal{
		LeaflistVal: &gnmi.ScalarArray{
			Element: []*gnmi.TypedValue{
				{Value: &gnmi.TypedValue_UintVal{UintVal: 100}},
				{Value: &gnmi.TypedValue_UintVal{UintVal: 101}},
				{Value: &gnmi.TypedValue_UintVal{UintVal: 102}},
				{Value: &gnmi.TypedValue_UintVal{UintVal: 103}},
			},
		},
	},
}

var decimalListTestValue = &gnmi.TypedValue{
	Value: &gnmi.TypedValue_LeaflistVal{
		LeaflistVal: &gnmi.ScalarArray{
			Element: []*gnmi.TypedValue{
				{
					Value: &gnmi.TypedValue_DecimalVal{
						DecimalVal: &gnmi.Decimal64{
							Digits:    6,
							Precision: 0,
						},
					},
				},
			},
		},
	},
}

var booleanListTestValue = &gnmi.TypedValue{
	Value: &gnmi.TypedValue_LeaflistVal{
		LeaflistVal: &gnmi.ScalarArray{
			Element: []*gnmi.TypedValue{
				{Value: &gnmi.TypedValue_BoolVal{BoolVal: true}},
				{Value: &gnmi.TypedValue_BoolVal{BoolVal: false}},
				{Value: &gnmi.TypedValue_BoolVal{BoolVal: true}},
				{Value: &gnmi.TypedValue_BoolVal{BoolVal: false}},
			},
		},
	},
}

var floatListTestValue = &gnmi.TypedValue{
	Value: &gnmi.TypedValue_LeaflistVal{
		LeaflistVal: &gnmi.ScalarArray{
			Element: []*gnmi.TypedValue{
				{Value: &gnmi.TypedValue_FloatVal{FloatVal: 1.0}},
				{Value: &gnmi.TypedValue_FloatVal{FloatVal: 2.0}},
				{Value: &gnmi.TypedValue_FloatVal{FloatVal: 3.0}},
				{Value: &gnmi.TypedValue_FloatVal{FloatVal: 4.0}},
			},
		},
	},
}

var bytesListTestValue = &gnmi.TypedValue{
	Value: &gnmi.TypedValue_LeaflistVal{
		LeaflistVal: &gnmi.ScalarArray{
			Element: []*gnmi.TypedValue{
				{Value: &gnmi.TypedValue_BytesVal{BytesVal: []byte("abc")}},
				{Value: &gnmi.TypedValue_BytesVal{BytesVal: []byte("def")}},
				{Value: &gnmi.TypedValue_BytesVal{BytesVal: []byte("ghi")}},
				{Value: &gnmi.TypedValue_BytesVal{BytesVal: []byte("jkl")}},
			},
		},
	},
}

var stringListTestValue = &gnmi.TypedValue{
	Value: &gnmi.TypedValue_LeaflistVal{
		LeaflistVal: &gnmi.ScalarArray{
			Element: []*gnmi.TypedValue{
				{Value: &gnmi.TypedValue_StringVal{StringVal: "abc"}},
				{Value: &gnmi.TypedValue_StringVal{StringVal: "def"}},
				{Value: &gnmi.TypedValue_StringVal{StringVal: "ghi"}},
				{Value: &gnmi.TypedValue_StringVal{StringVal: "jkl"}},
			},
		},
	},
}

var asciiListTestValue = &gnmi.TypedValue{
	Value: &gnmi.TypedValue_LeaflistVal{
		LeaflistVal: &gnmi.ScalarArray{
			Element: []*gnmi.TypedValue{
				{Value: &gnmi.TypedValue_AsciiVal{AsciiVal: "abc"}},
				{Value: &gnmi.TypedValue_AsciiVal{AsciiVal: "def"}},
				{Value: &gnmi.TypedValue_AsciiVal{AsciiVal: "ghi"}},
				{Value: &gnmi.TypedValue_AsciiVal{AsciiVal: "jkl"}},
			},
		},
	},
}

var bytesLeafTestValue = &gnmi.TypedValue{
	Value: &gnmi.TypedValue_BytesVal{
		BytesVal: []byte("abc")},
}

var floatLeafTestValue = &gnmi.TypedValue{
	Value: &gnmi.TypedValue_FloatVal{
		FloatVal: 1.234,
	},
}

var decimalLeafTestValue = &gnmi.TypedValue{
	Value: &gnmi.TypedValue_DecimalVal{
		DecimalVal: &gnmi.Decimal64{
			Digits:    1234,
			Precision: 2,
		},
	},
}

var asciiLeafTestValue = &gnmi.TypedValue{
	Value: &gnmi.TypedValue_AsciiVal{AsciiVal: "ascii"},
}

func Test_comparables(t *testing.T) {
	testCases := []struct {
		description  string
		expectedType devicechange.ValueType
		testValue    *gnmi.TypedValue
	}{
		{description: "Int", expectedType: devicechange.ValueType_LEAFLIST_INT, testValue: intListTestValue},
		{description: "Uint", expectedType: devicechange.ValueType_LEAFLIST_UINT, testValue: uintListTestValue},
		{description: "Decimal", expectedType: devicechange.ValueType_LEAFLIST_DECIMAL, testValue: decimalListTestValue},
		{description: "Boolean", expectedType: devicechange.ValueType_LEAFLIST_BOOL, testValue: booleanListTestValue},
		{description: "Float", expectedType: devicechange.ValueType_LEAFLIST_FLOAT, testValue: floatListTestValue},
		{description: "Bytes", expectedType: devicechange.ValueType_LEAFLIST_BYTES, testValue: bytesListTestValue},
		{description: "Strings", expectedType: devicechange.ValueType_LEAFLIST_STRING, testValue: stringListTestValue},
		{description: "Bytes", expectedType: devicechange.ValueType_BYTES, testValue: bytesLeafTestValue},
		{description: "Float", expectedType: devicechange.ValueType_FLOAT, testValue: floatLeafTestValue},
		{description: "Decimal", expectedType: devicechange.ValueType_DECIMAL, testValue: decimalLeafTestValue},
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

func Test_ascii(t *testing.T) {
	nativeType, err := GnmiTypedValueToNativeType(asciiLeafTestValue)
	assert.NilError(t, err)
	assert.Assert(t, nativeType != nil)
	assert.Equal(t, nativeType.Type, devicechange.ValueType_STRING)

	convertedValue, convertedErr := NativeTypeToGnmiTypedValue(nativeType)
	assert.NilError(t, convertedErr)
	assert.Assert(t, strings.Contains(convertedValue.String(), "ascii"), "%s", "Ascii")
}

func Test_asciiList(t *testing.T) {
	nativeType, err := GnmiTypedValueToNativeType(asciiListTestValue)
	assert.NilError(t, err)
	assert.Assert(t, nativeType != nil)
	assert.Equal(t, nativeType.Type, devicechange.ValueType_LEAFLIST_STRING)

	convertedValue, convertedErr := NativeTypeToGnmiTypedValue(nativeType)
	assert.NilError(t, convertedErr)
	s := convertedValue.String()
	assert.Assert(t, strings.Contains(s, `element:<string_val:"abc"`), "%s", "Ascii")
	assert.Assert(t, strings.Contains(s, `element:<string_val:"jkl"`), "%s", "Ascii")
}

func Test_empty(t *testing.T) {
	convertedValue, convertedErr := NativeTypeToGnmiTypedValue(devicechange.NewTypedValueEmpty())
	assert.NilError(t, convertedErr)
	s := convertedValue.String()
	assert.Assert(t, strings.Contains(s, "nil"), "%s", "Ascii")
}

func Test_errors(t *testing.T) {
	//  Bad length on typed value
	badTypedValue := devicechange.NewTypedValueEmpty()
	badTypedValue.Type = devicechange.ValueType_BYTES
	badTypedValue.Bytes = make([]byte, 0)
	invalidTypedLength, invalidTypedLengthErr := NativeTypeToGnmiTypedValue(badTypedValue)
	assert.ErrorContains(t, invalidTypedLengthErr, "invalid TypedValue Length 0")
	assert.Assert(t, invalidTypedLength == nil)

	//  Bad type
	badTypedValue.Type = 99
	badTypedValue.Bytes = make([]byte, 4)
	badType, badTypeErr := NativeTypeToGnmiTypedValue(badTypedValue)
	assert.ErrorContains(t, badTypeErr, "Unsupported type 99")
	assert.Assert(t, badType == nil)
}

////////////////////////////////////////////////////////////////////////////////
// Native format to gnmi
////////////////////////////////////////////////////////////////////////////////

func Test_NativeStringToGnmi(t *testing.T) {
	nativeString := devicechange.NewTypedValueString(testString)
	gnmiString, err := NativeTypeToGnmiTypedValue(nativeString)
	assert.NilError(t, err)
	_, ok := gnmiString.Value.(*gnmi.TypedValue_StringVal)
	assert.Assert(t, ok)

	assert.Equal(t, gnmiString.GetStringVal(), testString)
}

func Test_NativeIntToGnmi(t *testing.T) {
	nativeInt := devicechange.NewTypedValueInt64(testPositiveInt)
	gnmiInt, err := NativeTypeToGnmiTypedValue(nativeInt)
	assert.NilError(t, err)
	_, ok := gnmiInt.Value.(*gnmi.TypedValue_IntVal)
	assert.Assert(t, ok)

	assert.Equal(t, int(gnmiInt.GetIntVal()), testPositiveInt)
}

func Test_NativeUintToGnmi(t *testing.T) {
	nativeUint := devicechange.NewTypedValueUint64(testMaxUint)
	gnmiUint, err := NativeTypeToGnmiTypedValue(nativeUint)
	assert.NilError(t, err)
	_, ok := gnmiUint.Value.(*gnmi.TypedValue_UintVal)
	assert.Assert(t, ok)

	assert.Equal(t, uint(gnmiUint.GetUintVal()), testMaxUint)
}

func Test_NativeBoolToGnmi(t *testing.T) {
	nativeBool := devicechange.NewTypedValueBool(true)
	gnmiBool, err := NativeTypeToGnmiTypedValue(nativeBool)
	assert.NilError(t, err)
	_, ok := gnmiBool.Value.(*gnmi.TypedValue_BoolVal)
	assert.Assert(t, ok)

	assert.Equal(t, gnmiBool.GetBoolVal(), true)
}
