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
//

package device

import (
	"encoding/binary"
	"fmt"
	"gotest.tools/assert"
	"math"
	"strings"
	"testing"
)

func Test_NewChangedValue(t *testing.T) {
	path := "/a/b/c"
	badPath := "a@b@c"
	value := NewTypedValueUint64(64)
	const isRemove = false
	changeValueBad, errorBad := NewChangeValue(badPath, value, isRemove)
	assert.Assert(t, errorBad != nil)
	assert.Assert(t, strings.Contains(errorBad.Error(), badPath))
	assert.Assert(t, changeValueBad == nil)
	changeValue, err := NewChangeValue(path, value, isRemove)
	assert.Assert(t, err == nil)
	assert.Assert(t, changeValue != nil)
	assert.Assert(t, changeValue.Path == path)
	assert.Assert(t, changeValue.Value.ValueToString() == "64")
}

func Test_NewValueTypeString(t *testing.T) {
	const value = "xyzzy"
	typedValue, err := NewTypedValue([]byte(value), ValueType_STRING, make([]int32, 0))
	assert.NilError(t, err)
	assert.Equal(t, typedValue.ValueToString(), value)
}

func floatToBinary(value float64, size int) []byte {
	bs := make([]byte, size)
	binary.LittleEndian.PutUint64(bs, math.Float64bits(3.0))
	return bs
}

func uintToBinary(value uint64, size int) []byte {
	bs := make([]byte, size)
	binary.LittleEndian.PutUint64(bs, value)
	return bs
}

func boolToBinary(value bool, size int) []byte {
	bs := make([]byte, size)
	intValue := 0
	if value {
		intValue = 1
	} else {
		intValue = 0
	}
	bs[0] = byte(intValue)
	return bs
}

func TestValueTypes(t *testing.T) {
	testCases := []struct {
		description   string
		value         []byte
		valueType     ValueType
		expectedValue string
		expectedError string
		typeOpts      []int32
	}{
		{
			description:   "NewValueEmpty",
			value:         nil,
			valueType:     ValueType_EMPTY,
			expectedValue: "",
			expectedError: "",
		},
		{
			description:   "NewValueString",
			value:         []byte("abacab"),
			valueType:     ValueType_STRING,
			expectedValue: "abacab",
			expectedError: "",
		},
		{
			description:   "NewValueBytes",
			value:         []byte("abacab"),
			valueType:     ValueType_BYTES,
			expectedValue: "YWJhY2Fi",
			expectedError: "",
		},
		{
			description:   "NewValueTypeFloatSuccess",
			value:         floatToBinary(3.0, 8),
			valueType:     ValueType_FLOAT,
			expectedValue: fmt.Sprintf("%f", 3.0),
			expectedError: "",
		},
		{
			description:   "NewValueTypeFloatFailure",
			value:         floatToBinary(3.0, 12),
			valueType:     ValueType_FLOAT,
			expectedValue: fmt.Sprintf("%f", 3.0),
			expectedError: "Expecting 8 bytes for FLOAT. Got 12",
		},
		{
			description:   "NewValueTypeUintSuccess",
			value:         uintToBinary(345678, 8),
			valueType:     ValueType_UINT,
			expectedValue: "345678",
			expectedError: "",
		},
		{
			description:   "NewValueTypeUintFailure",
			value:         uintToBinary(345678, 12),
			valueType:     ValueType_UINT,
			expectedValue: "345678",
			expectedError: "Expecting 8 bytes for UINT. Got 12",
		},
		{
			description:   "NewValueTypeIntSuccess",
			value:         uintToBinary(345678, 8),
			valueType:     ValueType_INT,
			expectedValue: "345678",
			expectedError: "",
		},
		{
			description:   "NewValueTypeIntFailure",
			value:         uintToBinary(345678, 12),
			valueType:     ValueType_INT,
			expectedValue: "345678",
			expectedError: "Expecting 8 bytes for INT. Got 12",
		},
		{
			description:   "NewValueTypeIntSuccess",
			value:         boolToBinary(true, 1),
			valueType:     ValueType_BOOL,
			expectedValue: "true",
			expectedError: "",
		},
		{
			description:   "NewValueTypeIntFailure",
			value:         boolToBinary(true, 2),
			valueType:     ValueType_BOOL,
			expectedValue: "",
			expectedError: "Expecting 1 byte for BOOL. Got 2",
		},
		{
			description:   "NewValueTypeDecimalSuccess",
			value:         uintToBinary(1234, 8),
			valueType:     ValueType_DECIMAL,
			typeOpts:      []int32{0},
			expectedValue: "1234",
			expectedError: "",
		},
		{
			description:   "NewValueTypeDecimalFailure",
			value:         uintToBinary(1234, 12),
			valueType:     ValueType_DECIMAL,
			typeOpts:      []int32{3},
			expectedValue: "",
			expectedError: "Expecting 8 bytes for DECIMAL. Got 12",
		},
		{
			description:   "NewValueTypeLeafListString",
			value:         []byte{'a', 0x1D, 'b', 0x1D, 'c'},
			valueType:     ValueType_LEAFLIST_STRING,
			expectedValue: "a,b,c",
			expectedError: "",
		},
		{
			description: "NewValueTypeLeafListInt",
			value: []byte{
				11, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0,
				22, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0,
				33, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0,
			},
			valueType:     ValueType_LEAFLIST_INT,
			expectedValue: "[11 22 33]",
			expectedError: "",
		},
		{
			description: "NewValueTypeLeafListUint",
			value: []byte{
				3, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0,
				5, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0,
				7, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0,
			},
			valueType:     ValueType_LEAFLIST_UINT,
			expectedValue: "[3 5 7]",
			expectedError: "",
		},
		{
			description: "NewValueTypeLeafListBool",
			value: []byte{
				1, 0, 0, 1,
			},
			valueType:     ValueType_LEAFLIST_BOOL,
			expectedValue: "[true false false true]",
			expectedError: "",
		},
		{
			description:   "NewValueTypeLeafListDecimal",
			value:         append(uintToBinary(1234, 8), uintToBinary(1234, 8)...),
			valueType:     ValueType_LEAFLIST_DECIMAL,
			typeOpts:      []int32{0},
			expectedValue: "[1234 1234] 0",
			expectedError: "",
		},
		{
			description:   "NewValueTypeLeafListBytes",
			value:         []byte("12345678"),
			valueType:     ValueType_LEAFLIST_BYTES,
			typeOpts:      []int32{4, 4},
			expectedValue: "[[49 50 51 52] [53 54 55 56]]",
			expectedError: "",
		},
		{
			description:   "NewValueTypeLeafListFloat",
			value:         append(floatToBinary(1234, 8), floatToBinary(1234, 8)...),
			valueType:     ValueType_LEAFLIST_FLOAT,
			typeOpts:      []int32{0},
			expectedValue: "3.000000,3.000000",
			expectedError: "",
		},
	}

	for _, testCase := range testCases {
		typedValue, err := NewTypedValue(testCase.value, testCase.valueType, testCase.typeOpts)
		if testCase.expectedError != "" {
			assert.Error(t, err, testCase.expectedError, testCase.description)
			assert.Assert(t, typedValue == nil, testCase.description)
		} else {
			assert.NilError(t, err, testCase.description)
			s := typedValue.ValueToString()
			assert.Equal(t, s, testCase.expectedValue, testCase.description)
		}
	}
}

func TestTypedEmpty(t *testing.T) {
	empty := NewEmpty()
	assert.Equal(t, empty.ValueType(), ValueType_EMPTY)
	assert.Equal(t, empty.String(), "")
}

func TestTypedInt(t *testing.T) {
	intValue := NewInt64(112233)
	assert.Equal(t, intValue.ValueType(), ValueType_INT)
	assert.Equal(t, intValue.String(), "112233")
	assert.Equal(t, intValue.Int(), 112233)
}

func TestTypedUint(t *testing.T) {
	intValue := NewUint64(112233)
	assert.Equal(t, intValue.ValueType(), ValueType_UINT)
	assert.Equal(t, intValue.String(), "112233")
	assert.Equal(t, uint64(intValue.Uint()), uint64(112233))
}

func TestTypedString(t *testing.T) {
	intValue := NewString("xyzzy")
	assert.Equal(t, intValue.ValueType(), ValueType_STRING)
	assert.Equal(t, intValue.String(), "xyzzy")
}

func TestTypedBool(t *testing.T) {
	boolValue := NewBool(false)
	assert.Equal(t, boolValue.ValueType(), ValueType_BOOL)
	assert.Equal(t, boolValue.String(), "false")
	assert.Equal(t, boolValue.Bool(), false)
}

func TestTypedDecimal(t *testing.T) {
	decimal64Value := NewDecimal64(1234, 2)
	assert.Equal(t, decimal64Value.ValueType(), ValueType_DECIMAL)
	assert.Equal(t, decimal64Value.String(), "12.34")
	digits, precision := decimal64Value.Decimal64()
	assert.Equal(t, int(digits), 1234)
	assert.Equal(t, precision, uint32(2))
}

func TestTypedFloat(t *testing.T) {
	float64Value := NewFloat(2222.0)
	assert.Equal(t, float64Value.ValueType(), ValueType_FLOAT)
	assert.Equal(t, float64Value.String(), "2222.000000")
	assert.Equal(t, float64Value.Float32(), float32(2222.0))
}

func TestTypedByte(t *testing.T) {
	bytes := []byte("bytes")
	bytesValue := NewBytes(bytes)
	assert.Equal(t, bytesValue.ValueType(), ValueType_BYTES)
	assert.Equal(t, bytesValue.String(), "Ynl0ZXM=")
	assert.Equal(t, len(bytesValue.ByteArray()), len(bytes))
}

func TestTypedLeafListString(t *testing.T) {
	values := []string{"a", "b"}
	listValue := NewLeafListString(values)
	assert.Equal(t, listValue.ValueType(), ValueType_LEAFLIST_STRING)
}

func TestTypedLeafListUint(t *testing.T) {
	values := []uint{1, 2}
	listValue := NewLeafListUint64(values)
	assert.Equal(t, listValue.ValueType(), ValueType_LEAFLIST_UINT)
}

func TestTypedLeafListInt(t *testing.T) {
	values := []int{1, 2}
	listValue := NewLeafListInt64(values)
	assert.Equal(t, listValue.ValueType(), ValueType_LEAFLIST_INT)
}

func TestTypedLeafListBool(t *testing.T) {
	values := []bool{true, false}
	listValue := NewLeafListBool(values)
	assert.Equal(t, listValue.ValueType(), ValueType_LEAFLIST_BOOL)
}

func TestTypedLeafListFloat(t *testing.T) {
	values := []float32{111.0, 112.0}
	listValue := NewLeafListFloat32(values)
	assert.Equal(t, listValue.ValueType(), ValueType_LEAFLIST_FLOAT)
}

func TestTypedLeafListDecimal(t *testing.T) {
	digits := []int64{22, 33}
	listValue := NewLeafListDecimal64(digits, 2)
	assert.Equal(t, listValue.ValueType(), ValueType_LEAFLIST_DECIMAL)
	floats := listValue.ListFloat()
	assert.Equal(t, len(floats), len(digits))
	assert.Assert(t, floats[0] == 0.22)
	assert.Assert(t, floats[1] == 0.33)
}

func TestTypedLeafListBytes(t *testing.T) {
	values := make([][]byte, 2)
	values[0] = []byte("abc")
	values[1] = []byte("xyz")

	listValue := NewLeafListBytes(values)
	assert.Equal(t, listValue.ValueType(), ValueType_LEAFLIST_BYTES)
}

func TestLeafListBytesCrash(t *testing.T) {
	bytes := []byte("12345678")
	typeOpts := []int32{4}
	value, err := NewTypedValue(bytes, ValueType_LEAFLIST_BYTES, typeOpts)
	assert.Assert(t, value == nil)
	assert.Assert(t, err != nil)
}
