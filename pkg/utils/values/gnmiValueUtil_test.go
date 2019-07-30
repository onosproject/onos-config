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
	"github.com/onosproject/onos-config/pkg/store/change"
	pb "github.com/openconfig/gnmi/proto/gnmi"
	"gotest.tools/assert"
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

	nativeString := (*change.TypedString)(nativeType)
	assert.Equal(t, nativeString.String(), testString)
}

func Test_GnmiIntToNative(t *testing.T) {
	gnmiValue := pb.TypedValue_IntVal{IntVal: testNegativeInt}
	nativeType, err := GnmiTypedValueToNativeType(&pb.TypedValue{Value: &gnmiValue})
	assert.NilError(t, err)

	nativeInt64 := (*change.TypedInt64)(nativeType)
	assert.Equal(t, nativeInt64.Int(), testNegativeInt)
}

func Test_GnmiUintToNative(t *testing.T) {
	gnmiValue := pb.TypedValue_UintVal{UintVal: uint64(testMaxUint)}
	nativeType, err := GnmiTypedValueToNativeType(&pb.TypedValue{Value: &gnmiValue})
	assert.NilError(t, err)

	nativeUint64 := (*change.TypedUint64)(nativeType)
	assert.Equal(t, nativeUint64.Uint(), testMaxUint)
}

func Test_GnmiBoolToNative(t *testing.T) {
	gnmiValue := pb.TypedValue_BoolVal{BoolVal: true}
	nativeType, err := GnmiTypedValueToNativeType(&pb.TypedValue{Value: &gnmiValue})
	assert.NilError(t, err)

	nativeBool := (*change.TypedBool)(nativeType)
	assert.Equal(t, nativeBool.Bool(), true)
}

////////////////////////////////////////////////////////////////////////////////
// Native format to gnmi
////////////////////////////////////////////////////////////////////////////////

func Test_NativeStringToGnmi(t *testing.T) {
	nativeString := change.CreateTypedValueString(testString)
	gnmiString, err := NativeTypeToGnmiTypedValue(nativeString)
	assert.NilError(t, err)
	_, ok := gnmiString.Value.(*pb.TypedValue_StringVal)
	assert.Assert(t, ok)

	assert.Equal(t, gnmiString.GetStringVal(), testString)
}

func Test_NativeIntToGnmi(t *testing.T) {
	nativeInt := change.CreateTypedValueInt64(testPositiveInt)
	gnmiInt, err := NativeTypeToGnmiTypedValue(nativeInt)
	assert.NilError(t, err)
	_, ok := gnmiInt.Value.(*pb.TypedValue_IntVal)
	assert.Assert(t, ok)

	assert.Equal(t, int(gnmiInt.GetIntVal()), testPositiveInt)
}

func Test_NativeUintToGnmi(t *testing.T) {
	nativeUint := change.CreateTypedValueUint64(testMaxUint)
	gnmiUint, err := NativeTypeToGnmiTypedValue(nativeUint)
	assert.NilError(t, err)
	_, ok := gnmiUint.Value.(*pb.TypedValue_UintVal)
	assert.Assert(t, ok)

	assert.Equal(t, uint(gnmiUint.GetUintVal()), testMaxUint)
}

func Test_NativeBoolToGnmi(t *testing.T) {
	nativeBool := change.CreateTypedValueBool(true)
	gnmiBool, err := NativeTypeToGnmiTypedValue(nativeBool)
	assert.NilError(t, err)
	_, ok := gnmiBool.Value.(*pb.TypedValue_BoolVal)
	assert.Assert(t, ok)

	assert.Equal(t, gnmiBool.GetBoolVal(), true)
}
