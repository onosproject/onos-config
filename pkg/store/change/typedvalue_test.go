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

package change

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"gotest.tools/assert"
	"testing"
)

const testString = "This is a test"
const (
	testNegativeInt = -9223372036854775808
	testZeroInt     = 0
	testPositiveInt = 9223372036854775807
)
const (
	testZeroUint   = uint(0)
	testElevenUint = uint(11)
	testMaxUint    = uint(18446744073709551615)
)
const (
	testPrecision3 = 3
	testPrecision6 = 6
	testDigitsZero = 0
	testPrecision0 = 0
)
const (
	testFloatNeg = float32(-3.4E+38)
	testFloatPos = float32(3.4E+38)
)
const (
	testStringBytes    = "onos rocks!"
	testStringBytesB64 = "b25vcyByb2NrcyE="
)

var testLeafListString = []string{"abc", "def", "ghi", "with,comma"}

var testLeafListInt = []int{testNegativeInt, testZeroInt, testPositiveInt}

var testLeafListUint = []uint{testZeroUint, testElevenUint, testMaxUint}

var testLeafListBool = []bool{true, false, false, true}

var testLeafListDecimal = []int64{testNegativeInt, testZeroInt, testPositiveInt}

var testLeafListFloat = []float32{testFloatNeg, testZeroInt, testFloatPos}

var testByteArr0 = []byte("abc")
var testByteArr1 = []byte("defg")
var testByteArr2 = []byte("ghijk")
var testLeafListBytes = [][]byte{testByteArr0, testByteArr1, testByteArr2}

func Test_TypedValueEmpty(t *testing.T) {
	tv := (CreateTypedValueEmpty())

	assert.Equal(t, len(tv.Value), 0)
	testConversion(t, (*TypedValue)(tv))
}

func Test_TypedValueString(t *testing.T) {
	tv := (*TypedValue)(CreateTypedValueString(testString))

	assert.Equal(t, len(tv.Value), 14)
	testConversion(t, (*TypedValue)(tv))
}

func Test_TypedValueInt(t *testing.T) {
	tv := CreateTypedValueInt64(testNegativeInt)
	assert.Equal(t, tv.String(), fmt.Sprintf("%d", testNegativeInt))
	assert.Equal(t, len(tv.Value), 8)

	testConversion(t, (*TypedValue)(tv))

	tv = CreateTypedValueInt64(testPositiveInt)
	assert.Equal(t, tv.String(), fmt.Sprintf("%d", testPositiveInt))

	tv = CreateTypedValueInt64(testZeroInt)
	assert.Equal(t, tv.String(), fmt.Sprintf("%d", testZeroInt))
}

func Test_TypedValueUint(t *testing.T) {
	tv := CreateTypedValueUint64(testZeroUint)
	assert.Equal(t, tv.String(), fmt.Sprintf("%d", testZeroUint))
	assert.Equal(t, len(tv.Value), 8)

	tv = CreateTypedValueUint64(testMaxUint)
	assert.Equal(t, tv.String(), fmt.Sprintf("%d", testMaxUint))

	testConversion(t, (*TypedValue)(tv))
}

func Test_TypedValueBool(t *testing.T) {
	tv := CreateTypedValueBool(true)

	assert.Equal(t, len(tv.Value), 1)
	assert.Equal(t, tv.String(), "true")

	testConversion(t, (*TypedValue)(tv))

	tv = CreateTypedValueBool(false)
	assert.Equal(t, tv.String(), "false")
}

func Test_TypedValueDecimal64(t *testing.T) {
	tv := CreateTypedValueDecimal64(testNegativeInt, testPrecision3)
	assert.Equal(t, len(tv.Value), 8)
	assert.Equal(t, tv.String(), "-9223372036854775.808")
	assert.Equal(t, tv.TypeOpts[0], testPrecision3)

	testConversion(t, (*TypedValue)(tv))

	tv = CreateTypedValueDecimal64(testPositiveInt, testPrecision6)
	assert.Equal(t, tv.String(), "9223372036854.775807")
	assert.Equal(t, tv.TypeOpts[0], testPrecision6)

	tv = CreateTypedValueDecimal64(testDigitsZero, testPrecision0)
	assert.Equal(t, tv.String(), "0.0")
	assert.Equal(t, tv.TypeOpts[0], testPrecision0)
}

func Test_TypedValueFloat(t *testing.T) {
	tv := CreateTypedValueFloat(testFloatNeg)
	assert.Equal(t, len(tv.Value), 8)
	assert.Equal(t, tv.String(), "-339999995214436424907732413799364296704.000000")

	testConversion(t, (*TypedValue)(tv))

	tv = CreateTypedValueFloat(testFloatPos)
	assert.Equal(t, tv.String(), "339999995214436424907732413799364296704.000000")
}

func Test_TypedValueBytes(t *testing.T) {
	tv := CreateTypedValueBytes([]byte(testStringBytes))
	assert.Equal(t, len(tv.Value), 11)
	assert.Equal(t, tv.String(), testStringBytesB64)

	testConversion(t, (*TypedValue)(tv))
}

func Test_LeafListString(t *testing.T) {
	tv := CreateLeafListString(testLeafListString)
	assert.Equal(t, len(tv.Value), 22)
	assert.Equal(t, tv.String(), "abc,def,ghi,with,comma")
	testConversion(t, (*TypedValue)(tv))

	testArray := []string{"one"}
	tv = CreateLeafListString(testArray)
	assert.Equal(t, len(tv.Value), 3)
	assert.Equal(t, tv.String(), "one")
}

func Test_LeafListInt64(t *testing.T) {
	tv := CreateLeafListInt64(testLeafListInt)
	assert.Equal(t, len(tv.Value), 24)
	assert.Equal(t, tv.String(), "[-9223372036854775808 0 9223372036854775807]")

	testConversion(t, (*TypedValue)(tv))
}

func Test_LeafListUint64(t *testing.T) {
	tv := CreateLeafListUint64(testLeafListUint)

	assert.Equal(t, len(tv.Value), 24)
	assert.Equal(t, tv.String(), "[0 11 18446744073709551615]")

	testConversion(t, (*TypedValue)(tv))
}

func Test_LeafListBool(t *testing.T) {
	tv := CreateLeafListBool(testLeafListBool)

	assert.Equal(t, len(tv.Value), 4)
	assert.Equal(t, tv.String(), "[true false false true]")

	testConversion(t, (*TypedValue)(tv))
}

func Test_LeafListDecimal64(t *testing.T) {
	tv := CreateLeafListDecimal64(testLeafListDecimal, testPrecision6)

	assert.Equal(t, len(tv.Value), 24)
	assert.Equal(t, tv.String(), "[-9223372036854775808 0 9223372036854775807] 6")

	testConversion(t, (*TypedValue)(tv))
}

func Test_LeafListFloat32(t *testing.T) {
	tv := CreateLeafListFloat32(testLeafListFloat)

	assert.Equal(t, len(tv.Value), 24)
	assert.Equal(t, tv.String(), "-339999995214436424907732413799364296704.000000,0.000000,339999995214436424907732413799364296704.000000")

	testConversion(t, (*TypedValue)(tv))
}

func Test_LeafListBytes(t *testing.T) {
	tv := CreateLeafListBytes(testLeafListBytes)

	testConversion(t, (*TypedValue)(tv))

	assert.Equal(t, len(tv.Value), 12)
	assert.Equal(t, tv.String(), "[[97 98 99] [100 101 102 103] [103 104 105 106 107]]")
	assert.Equal(t, tv.TypeOpts[0], 3)
	assert.Equal(t, tv.TypeOpts[1], 4)
	assert.Equal(t, tv.TypeOpts[2], 5)
}

func Test_JsonSerializationDecimal(t *testing.T) {
	tv := (*TypedValue)(CreateTypedValueDecimal64(testPositiveInt, 6))

	jsonStr, err := json.Marshal(tv)
	assert.NilError(t, err)

	assert.Equal(t, string(jsonStr), `{"Value":"/////////38=","Type":5,"TypeOpts":[6]}`)

	unmarshalledTv := TypedValue{}
	err = json.Unmarshal(jsonStr, &unmarshalledTv)
	assert.NilError(t, err)

	assert.Equal(t, unmarshalledTv.Type, ValueTypeDECIMAL)
	assert.Equal(t, len(unmarshalledTv.TypeOpts), 1)
	assert.Equal(t, unmarshalledTv.TypeOpts[0], 6)
	assert.DeepEqual(t, unmarshalledTv.Value, []byte{0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0x7f})

	decFloat, _ := (*TypedDecimal64)(&unmarshalledTv).Float()
	assert.Equal(t, decFloat, 9.223372036854775e+12)
}

func Test_JsonSerializationLeafListBytes(t *testing.T) {
	tv := (*TypedValue)(CreateLeafListBytes(testLeafListBytes))

	jsonStr, err := json.Marshal(tv)
	assert.NilError(t, err)

	assert.Equal(t, string(jsonStr), `{"Value":"YWJjZGVmZ2doaWpr","Type":14,"TypeOpts":[3,4,5]}`)

	unmarshalledTv := TypedValue{}
	err = json.Unmarshal(jsonStr, &unmarshalledTv)
	assert.NilError(t, err)

	assert.Equal(t, unmarshalledTv.Type, ValueTypeLeafListBYTES)
	assert.Equal(t, len(unmarshalledTv.TypeOpts), 3)
	assert.Equal(t, unmarshalledTv.TypeOpts[0], 3)
	assert.Equal(t, unmarshalledTv.TypeOpts[1], 4)
	assert.Equal(t, unmarshalledTv.TypeOpts[2], 5)
	assert.DeepEqual(t, unmarshalledTv.Value, []byte{0x61, 0x62, 0x63, 0x64, 0x65, 0x66, 0x67, 0x67, 0x68, 0x69, 0x6a, 0x6b})

	assert.Equal(t, (*TypedLeafListBytes)(&unmarshalledTv).String(), "[[97 98 99] [100 101 102 103] [103 104 105 106 107]]")
}

func testConversion(t *testing.T, tv *TypedValue) {

	switch tv.Type {
	case ValueTypeEMPTY:
		assert.Equal(t, (*TypedEmpty)(tv).String(), "")
	case ValueTypeSTRING:
		assert.Equal(t, (*TypedString)(tv).String(), testString)
	case ValueTypeINT:
		assert.Equal(t, (*TypedInt64)(tv).Int(), testNegativeInt)
	case ValueTypeUINT:
		assert.Equal(t, (*TypedUint64)(tv).Uint(), testMaxUint)
	case ValueTypeBOOL:
		assert.Equal(t, (*TypedBool)(tv).Bool(), true)
	case ValueTypeDECIMAL:
		digits, precision := (*TypedDecimal64)(tv).Decimal64()
		assert.Equal(t, digits, int64(testNegativeInt))
		assert.Equal(t, precision, uint32(testPrecision3))
	case ValueTypeFLOAT:
		assert.Equal(t, (*TypedFloat)(tv).Float32(), testFloatNeg)
	case ValueTypeBYTES:
		assert.Equal(t, base64.StdEncoding.EncodeToString((*TypedBytes)(tv).Bytes()), testStringBytesB64)
		assert.Equal(t, len(tv.TypeOpts), 1)
		assert.Equal(t, tv.TypeOpts[0], 11)
	case ValueTypeLeafListSTRING:
		assert.DeepEqual(t, (*TypedLeafListString)(tv).List(), testLeafListString)
	case ValueTypeLeafListINT:
		assert.DeepEqual(t, (*TypedLeafListInt64)(tv).List(), testLeafListInt)
	case ValueTypeLeafListUINT:
		assert.DeepEqual(t, (*TypedLeafListUint)(tv).List(), testLeafListUint)
	case ValueTypeLeafListBOOL:
		assert.DeepEqual(t, (*TypedLeafListBool)(tv).List(), testLeafListBool)
	case ValueTypeLeafListDECIMAL:
		digits, precision := (*TypedLeafListDecimal)(tv).List()
		assert.DeepEqual(t, digits, testLeafListDecimal)
		assert.Equal(t, precision, uint32(testPrecision6))
	case ValueTypeLeafListFLOAT:
		assert.DeepEqual(t, (*TypedLeafListFloat)(tv).List(), testLeafListFloat)
	case ValueTypeLeafListBYTES:
		assert.DeepEqual(t, (*TypedLeafListBytes)(tv).List(), testLeafListBytes)
		assert.Equal(t, len(tv.TypeOpts), 3)
		assert.Equal(t, tv.TypeOpts[0], 3)
		assert.Equal(t, tv.TypeOpts[1], 4)
		assert.Equal(t, tv.TypeOpts[2], 5)

	default:
		t.Log("Unexpected type", tv.Type)
		t.Fail()
	}

}
