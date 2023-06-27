// SPDX-FileCopyrightText: 2020-present Open Networking Foundation <info@opennetworking.org>
//
// SPDX-License-Identifier: Apache-2.0

// Package utils test various gNMI value manipulation facilities.

package values

import (
	"fmt"
	configapi "github.com/onosproject/onos-api/go/onos/config/v3"
	"reflect"
	"testing"

	"github.com/openconfig/gnmi/proto/gnmi"
	"github.com/stretchr/testify/assert"
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
	nativeType, err := GnmiTypedValueToNativeType(&gnmi.TypedValue{Value: &gnmiValue}, nil)
	assert.NoError(t, err)

	nativeString := (*configapi.TypedString)(nativeType)
	assert.Equal(t, nativeString.String(), testString)
}

func Test_GnmiIntToNative(t *testing.T) {
	pathElem := configapi.ReadWritePath{
		ValueType: configapi.ValueType_INT,
		TypeOpts:  []uint64{uint64(configapi.WidthThirtyTwo)},
	}
	gnmiValue := gnmi.TypedValue_IntVal{IntVal: testNegativeInt}
	nativeType, err := GnmiTypedValueToNativeType(&gnmi.TypedValue{Value: &gnmiValue}, &pathElem)
	assert.NoError(t, err)

	nativeInt64 := (*configapi.TypedInt)(nativeType)
	assert.Equal(t, nativeInt64.Int(), testNegativeInt)
}

func Test_GnmiUintToNative(t *testing.T) {
	pathElem := configapi.ReadWritePath{
		ValueType: configapi.ValueType_UINT,
		TypeOpts:  []uint64{uint64(configapi.WidthSixtyFour)},
	}
	gnmiValue := gnmi.TypedValue_UintVal{UintVal: uint64(testMaxUint)}
	nativeType, err := GnmiTypedValueToNativeType(&gnmi.TypedValue{Value: &gnmiValue}, &pathElem)
	assert.NoError(t, err)

	nativeUint64 := (*configapi.TypedUint)(nativeType)
	assert.Equal(t, nativeUint64.Uint(), testMaxUint)
}

func Test_GnmiBoolToNative(t *testing.T) {
	gnmiValue := gnmi.TypedValue_BoolVal{BoolVal: true}
	nativeType, err := GnmiTypedValueToNativeType(&gnmi.TypedValue{Value: &gnmiValue}, nil)
	assert.NoError(t, err)

	nativeBool := (*configapi.TypedBool)(nativeType)
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
		expectedType configapi.ValueType
		testValue    *gnmi.TypedValue
	}{
		{description: "Int List", expectedType: configapi.ValueType_LEAFLIST_INT, testValue: intListTestValue},
		{description: "Uint List", expectedType: configapi.ValueType_LEAFLIST_UINT, testValue: uintListTestValue},
		{description: "Decimal List", expectedType: configapi.ValueType_LEAFLIST_DECIMAL, testValue: decimalListTestValue},
		{description: "Boolean List", expectedType: configapi.ValueType_LEAFLIST_BOOL, testValue: booleanListTestValue},
		{description: "Float List", expectedType: configapi.ValueType_LEAFLIST_FLOAT, testValue: floatListTestValue},
		{description: "Bytes List", expectedType: configapi.ValueType_LEAFLIST_BYTES, testValue: bytesListTestValue},
		{description: "Strings List", expectedType: configapi.ValueType_LEAFLIST_STRING, testValue: stringListTestValue},
		{description: "Bytes Leaf", expectedType: configapi.ValueType_BYTES, testValue: bytesLeafTestValue},
		{description: "Float Leaf", expectedType: configapi.ValueType_FLOAT, testValue: floatLeafTestValue},
		{description: "Decimal Leaf", expectedType: configapi.ValueType_DECIMAL, testValue: decimalLeafTestValue},
	}

	for _, testCase := range testCases {
		nativeType, err := GnmiTypedValueToNativeType(testCase.testValue, nil)
		assert.NoError(t, err)
		assert.NotNil(t, nativeType)
		assert.Equal(t, nativeType.Type, testCase.expectedType)

		convertedValue, convertedErr := NativeTypeToGnmiTypedValue(nativeType)
		assert.NoError(t, convertedErr)
		assert.True(t, reflect.DeepEqual(convertedValue, testCase.testValue), "%s", testCase.description)
	}
}

func Test_ascii(t *testing.T) {
	nativeType, err := GnmiTypedValueToNativeType(asciiLeafTestValue, nil)
	assert.NoError(t, err)
	assert.NotNil(t, nativeType)
	assert.Equal(t, nativeType.Type, configapi.ValueType_STRING)

	convertedValue, convertedErr := NativeTypeToGnmiTypedValue(nativeType)
	assert.NoError(t, convertedErr)
	assert.Contains(t, convertedValue.String(), "ascii", "%s", "Ascii")
}

func Test_asciiList(t *testing.T) {
	nativeType, err := GnmiTypedValueToNativeType(asciiListTestValue, nil)
	assert.NoError(t, err)
	assert.NotNil(t, nativeType)
	assert.Equal(t, nativeType.Type, configapi.ValueType_LEAFLIST_STRING)

	convertedValue, convertedErr := NativeTypeToGnmiTypedValue(nativeType)
	assert.NoError(t, convertedErr)
	s := convertedValue.String()
	assert.Contains(t, s, `element:{string_val:"abc"}`, "%s", "Ascii")
	assert.Contains(t, s, `element:{string_val:"jkl"}`, "%s", "Ascii")
}

func Test_empty(t *testing.T) {
	convertedValue, convertedErr := NativeTypeToGnmiTypedValue(configapi.NewTypedValueEmpty())
	assert.NoError(t, convertedErr)
	s := convertedValue.String()
	fmt.Println(s)
	assert.Contains(t, s, "{}", "%s", "Ascii")
}

func Test_errors(t *testing.T) {
	//  Bad length on typed value
	badTypedValue := configapi.NewTypedValueEmpty()

	//  Bad type
	badTypedValue.Type = 99
	badTypedValue.Bytes = make([]byte, 4)
	badType, badTypeErr := NativeTypeToGnmiTypedValue(badTypedValue)
	assert.Error(t, badTypeErr)
	assert.Contains(t, badTypeErr.Error(), "Unsupported type 99")
	assert.Nil(t, badType)
}

////////////////////////////////////////////////////////////////////////////////
// Native format to gnmi
////////////////////////////////////////////////////////////////////////////////

func Test_NativeStringToGnmi(t *testing.T) {
	nativeString := configapi.NewTypedValueString(testString)
	gnmiString, err := NativeTypeToGnmiTypedValue(nativeString)
	assert.NoError(t, err)
	_, ok := gnmiString.Value.(*gnmi.TypedValue_StringVal)
	assert.True(t, ok)

	assert.Equal(t, gnmiString.GetStringVal(), testString)
}

func Test_NativeIntToGnmi(t *testing.T) {
	nativeInt := configapi.NewTypedValueInt(testPositiveInt, 64)
	gnmiInt, err := NativeTypeToGnmiTypedValue(nativeInt)
	assert.NoError(t, err)
	_, ok := gnmiInt.Value.(*gnmi.TypedValue_IntVal)
	assert.True(t, ok)

	assert.Equal(t, int(gnmiInt.GetIntVal()), testPositiveInt)
}

func Test_NativeUintToGnmi(t *testing.T) {
	nativeUint := configapi.NewTypedValueUint(testMaxUint, 64)
	gnmiUint, err := NativeTypeToGnmiTypedValue(nativeUint)
	assert.NoError(t, err)
	_, ok := gnmiUint.Value.(*gnmi.TypedValue_UintVal)
	assert.True(t, ok)

	assert.Equal(t, uint(gnmiUint.GetUintVal()), testMaxUint)
}

func Test_NativeBoolToGnmi(t *testing.T) {
	nativeBool := configapi.NewTypedValueBool(true)
	gnmiBool, err := NativeTypeToGnmiTypedValue(nativeBool)
	assert.NoError(t, err)
	_, ok := gnmiBool.Value.(*gnmi.TypedValue_BoolVal)
	assert.True(t, ok)

	assert.Equal(t, gnmiBool.GetBoolVal(), true)
}
