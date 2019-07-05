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
	"fmt"
	"gotest.tools/assert"
	"testing"
)

func Test_TypedValueEmpty(t *testing.T) {
	tv := CreateTypedValueEmpty()

	assert.Equal(t, len(tv.Value), 0)
	assert.Equal(t, tv.Type, TypeEmpty)
	assert.Equal(t, tv.LeafList, false)
}

func Test_TypedValueString(t *testing.T) {
	const testString = "This is a test"
	tv := CreateTypedValueString(testString)

	assert.Equal(t, len(tv.Value), 14)
	assert.Equal(t, tv.String(), testString)
	assert.Equal(t, tv.Type, TypeString)
	assert.Equal(t, tv.LeafList, false)
}

func Test_TypedValueInt(t *testing.T) {
	const testNegative = -9223372036854775808
	tv := CreateTypedValueInt64(testNegative)

	assert.Equal(t, len(tv.Value), 8)
	assert.Equal(t, tv.String(), fmt.Sprintf("%d", testNegative))
	assert.Equal(t, tv.Type, TypeInt)
	assert.Equal(t, tv.LeafList, false)

	const testPositive = 9223372036854775807
	tv = CreateTypedValueInt64(testPositive)
	assert.Equal(t, tv.String(), fmt.Sprintf("%d", testPositive))

	const testZero = 0
	tv = CreateTypedValueInt64(testZero)
	assert.Equal(t, tv.String(), fmt.Sprintf("%d", testZero))
}

func Test_TypedValueUint(t *testing.T) {
	const testZero = uint(0)
	tv := CreateTypedValueUint64(testZero)

	assert.Equal(t, len(tv.Value), 8)
	assert.Equal(t, tv.String(), fmt.Sprintf("%d", testZero))
	assert.Equal(t, tv.Type, TypeUint)
	assert.Equal(t, tv.LeafList, false)

	const testMax = uint(18446744073709551615)
	tv = CreateTypedValueUint64(testMax)
	assert.Equal(t, tv.String(), fmt.Sprintf("%d", testMax))
}

func Test_TypedValueBool(t *testing.T) {
	tv := CreateTypedValueBool(true)

	assert.Equal(t, len(tv.Value), 1)
	assert.Equal(t, tv.String(), "true")
	assert.Equal(t, tv.Type, TypeBool)
	assert.Equal(t, tv.LeafList, false)

	tv = CreateTypedValueBool(false)
	assert.Equal(t, tv.String(), "false")
}

func Test_TypedValueDecimal64(t *testing.T) {
	const testDigitsNeg = -9223372036854775808
	const testPrecision3 = 3
	tv := CreateTypedValueDecimal64(testDigitsNeg, testPrecision3)

	assert.Equal(t, len(tv.Value), 8)
	assert.Equal(t, tv.String(), "-9223372036854775.808")
	assert.Equal(t, tv.Type, fmt.Sprintf("%s-%d", TypeDecimal64, testPrecision3))
	assert.Equal(t, tv.LeafList, false)

	const testDigitsPos = 9223372036854775807
	const testPrecision6 = 6
	tv = CreateTypedValueDecimal64(testDigitsPos, testPrecision6)

	assert.Equal(t, tv.String(), "9223372036854.775807")
	assert.Equal(t, tv.Type, fmt.Sprintf("%s-%d", TypeDecimal64, testPrecision6))

	const testDigitsZero = 0
	const testPrecision0 = 0
	tv = CreateTypedValueDecimal64(testDigitsZero, testPrecision0)

	assert.Equal(t, tv.String(), "0.0")
	assert.Equal(t, tv.Type, fmt.Sprintf("%s-%d", TypeDecimal64, testPrecision0))
}

func Test_TypedValueFloat(t *testing.T) {
	const testFloatNeg = float32(-3.4E+38)
	tv := CreateTypedValueFloat(testFloatNeg)

	assert.Equal(t, len(tv.Value), 8)
	assert.Equal(t, tv.String(), "-339999995214436424907732413799364296704.000000")
	assert.Equal(t, tv.Type, TypeFloat)
	assert.Equal(t, tv.LeafList, false)

	const testFloatPos = float32(3.4E+38)
	tv = CreateTypedValueFloat(testFloatPos)
	assert.Equal(t, tv.String(), "339999995214436424907732413799364296704.000000")
}

func Test_TypedValueBytes(t *testing.T) {
	const testString = "onos rocks!"
	tv := CreateTypedValueBytes([]byte(testString))

	assert.Equal(t, len(tv.Value), 11)
	assert.Equal(t, tv.String(), "b25vcyByb2NrcyE=")
	assert.Equal(t, tv.Type, TypeBytes)
	assert.Equal(t, tv.LeafList, false)
}

func Test_LeafListString(t *testing.T) {
	testArray := []string{"abc", "def", "ghi", "with,comma"}
	tv := CreateLeafListString(testArray)

	assert.Equal(t, len(tv.Value), 22)
	assert.Equal(t, tv.String(), "abc,def,ghi,with,comma")
	assert.Equal(t, tv.Type, TypeString)
	assert.Equal(t, tv.LeafList, true)

	testArray = []string{"one"}
	tv = CreateLeafListString(testArray)
	assert.Equal(t, len(tv.Value), 3)
	assert.Equal(t, tv.String(), "one")
}

func Test_LeafListInt64(t *testing.T) {
	testArray := []int{-9223372036854775808, 11, 9223372036854775807}
	tv := CreateLeafListInt64(testArray)

	assert.Equal(t, len(tv.Value), 24)
	assert.Equal(t, tv.String(), "[-9223372036854775808 11 9223372036854775807]")
	assert.Equal(t, tv.Type, TypeInt)
	assert.Equal(t, tv.LeafList, true)
}

func Test_LeafListUint64(t *testing.T) {
	testArray := []uint{0, 11, 18446744073709551615}
	tv := CreateLeafListUint64(testArray)

	assert.Equal(t, len(tv.Value), 24)
	assert.Equal(t, tv.String(), "[0 11 18446744073709551615]")
	assert.Equal(t, tv.Type, TypeUint)
	assert.Equal(t, tv.LeafList, true)
}

func Test_LeafListBool(t *testing.T) {
	testArray := []bool{true, false, false, true}
	tv := CreateLeafListBool(testArray)

	assert.Equal(t, len(tv.Value), 4)
	assert.Equal(t, tv.String(), "[true false false true]")
	assert.Equal(t, tv.Type, TypeBool)
	assert.Equal(t, tv.LeafList, true)
}

func Test_LeafListDecimal64(t *testing.T) {
	const precision = 6
	testArray := []int64{-9223372036854775808, 11, 9223372036854775807}
	tv := CreateLeafListDecimal64(testArray, precision)

	assert.Equal(t, len(tv.Value), 24)
	assert.Equal(t, tv.String(), "[-9223372036854775808 11 9223372036854775807] 6")
	assert.Equal(t, tv.Type, TypeDecimal64+"-6")
	assert.Equal(t, tv.LeafList, true)
}

func Test_LeafListFloat32(t *testing.T) {
	testArray := []float32{-3.4E+30, 0, 3.4E+30}
	tv := CreateLeafListFloat32(testArray)

	assert.Equal(t, len(tv.Value), 24)
	assert.Equal(t, tv.String(), "-3399999900045657695752095268864.000000,0.000000,3399999900045657695752095268864.000000")
	assert.Equal(t, tv.Type, TypeFloat)
	assert.Equal(t, tv.LeafList, true)
}

func Test_LeafListBytes(t *testing.T) {
	testArray := [][]byte{[]byte("abc"), []byte("defg"), []byte("ghijk")}
	tv := CreateLeafListBytes(testArray)

	assert.Equal(t, len(tv.Value), 12)
	assert.Equal(t, tv.String(), "[[97 98 99] [100 101 102 103] [103 104 105 106 107]]")
	assert.Equal(t, tv.Type, TypeBytes+"_3_4_5")
	assert.Equal(t, tv.LeafList, true)
}
