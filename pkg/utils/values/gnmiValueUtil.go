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

// Package values implements various gNMI value manipulation facilities.
package values

import (
	"fmt"
	devicechange "github.com/onosproject/onos-config/api/types/change/device"
	gnmi "github.com/openconfig/gnmi/proto/gnmi"
)

// GnmiTypedValueToNativeType converts gnmi type based values in to native byte array devicechange
func GnmiTypedValueToNativeType(gnmiTv *gnmi.TypedValue) (*devicechange.TypedValue, error) {

	switch v := gnmiTv.GetValue().(type) {
	case *gnmi.TypedValue_StringVal:
		return devicechange.NewTypedValueString(v.StringVal), nil
	case *gnmi.TypedValue_AsciiVal:
		return devicechange.NewTypedValueString(v.AsciiVal), nil
	case *gnmi.TypedValue_IntVal:
		return devicechange.NewTypedValueInt64(int(v.IntVal)), nil
	case *gnmi.TypedValue_UintVal:
		return devicechange.NewTypedValueUint64(uint(v.UintVal)), nil
	case *gnmi.TypedValue_BoolVal:
		return devicechange.NewTypedValueBool(v.BoolVal), nil
	case *gnmi.TypedValue_BytesVal:
		return devicechange.NewTypedValueBytes(v.BytesVal), nil
	case *gnmi.TypedValue_DecimalVal:
		return devicechange.NewTypedValueDecimal64(v.DecimalVal.Digits, v.DecimalVal.Precision), nil
	case *gnmi.TypedValue_FloatVal:
		return devicechange.NewTypedValueFloat(v.FloatVal), nil
	case *gnmi.TypedValue_LeaflistVal:
		return handleLeafList(v)
	default:
		return nil, fmt.Errorf("Not yet supported %v", v)
	}
}

func handleLeafList(gnmiLl *gnmi.TypedValue_LeaflistVal) (*devicechange.TypedValue, error) {
	stringList := make([]string, 0)
	intList := make([]int, 0)
	uintList := make([]uint, 0)
	boolList := make([]bool, 0)
	bytesList := make([][]byte, 0)
	digitsList := make([]int64, 0)
	precision := uint32(0)
	floatList := make([]float32, 0)

	// All values will be of the same type, but we don't know what that is yet
	for _, leaf := range gnmiLl.LeaflistVal.GetElement() {
		switch u := leaf.GetValue().(type) {
		case *gnmi.TypedValue_StringVal:
			stringList = append(stringList, u.StringVal)
		case *gnmi.TypedValue_AsciiVal:
			stringList = append(stringList, u.AsciiVal)
		case *gnmi.TypedValue_IntVal:
			intList = append(intList, int(u.IntVal))
		case *gnmi.TypedValue_UintVal:
			uintList = append(uintList, uint(u.UintVal))
		case *gnmi.TypedValue_BoolVal:
			boolList = append(boolList, u.BoolVal)
		case *gnmi.TypedValue_BytesVal:
			bytesList = append(bytesList, u.BytesVal)
		case *gnmi.TypedValue_DecimalVal:
			digitsList = append(digitsList, u.DecimalVal.Digits)
			precision = u.DecimalVal.Precision
		case *gnmi.TypedValue_FloatVal:
			floatList = append(floatList, u.FloatVal)
		default:
			return nil, fmt.Errorf("Leaf list type Not yet supported %v", u)
		}
	}

	if len(stringList) > 0 {
		return devicechange.NewLeafListStringTv(stringList), nil
	} else if len(intList) > 0 {
		return devicechange.NewLeafListInt64Tv(intList), nil
	} else if len(uintList) > 0 {
		return devicechange.NewLeafListUint64Tv(uintList), nil
	} else if len(boolList) > 0 {
		return devicechange.NewLeafListBoolTv(boolList), nil
	} else if len(bytesList) > 0 {
		return devicechange.NewLeafListBytesTv(bytesList), nil
	} else if len(digitsList) > 0 {
		return devicechange.NewLeafListDecimal64Tv(digitsList, precision), nil
	} else if len(floatList) > 0 {
		return devicechange.NewLeafListFloat32Tv(floatList), nil
	}
	return nil, fmt.Errorf("Empty leaf list given")
}

// NativeTypeToGnmiTypedValue converts native byte array based values in to gnmi types
func NativeTypeToGnmiTypedValue(typedValue *devicechange.TypedValue) (*gnmi.TypedValue, error) {
	if len(typedValue.Bytes) == 0 && typedValue.Type != devicechange.ValueType_EMPTY {
		return nil, fmt.Errorf("invalid TypedValue Length 0")
	}
	switch typedValue.Type {
	case devicechange.ValueType_EMPTY:
		gnmiValue := &gnmi.TypedValue_AnyVal{AnyVal: nil}
		return &gnmi.TypedValue{Value: gnmiValue}, nil

	case devicechange.ValueType_STRING:
		gnmiValue := gnmi.TypedValue_StringVal{StringVal: (*devicechange.TypedString)(typedValue).String()}
		return &gnmi.TypedValue{Value: &gnmiValue}, nil

	case devicechange.ValueType_INT:
		gnmiValue := gnmi.TypedValue_IntVal{IntVal: int64((*devicechange.TypedInt64)(typedValue).Int())}
		return &gnmi.TypedValue{Value: &gnmiValue}, nil

	case devicechange.ValueType_UINT:
		gnmiValue := gnmi.TypedValue_UintVal{UintVal: uint64((*devicechange.TypedUint64)(typedValue).Uint())}
		return &gnmi.TypedValue{Value: &gnmiValue}, nil

	case devicechange.ValueType_BOOL:
		gnmiValue := gnmi.TypedValue_BoolVal{BoolVal: (*devicechange.TypedBool)(typedValue).Bool()}
		return &gnmi.TypedValue{Value: &gnmiValue}, nil

	case devicechange.ValueType_DECIMAL:
		digits, precision := (*devicechange.TypedDecimal64)(typedValue).Decimal64()
		gnmiValue := gnmi.TypedValue_DecimalVal{DecimalVal: &gnmi.Decimal64{Digits: digits, Precision: precision}}
		return &gnmi.TypedValue{Value: &gnmiValue}, nil

	case devicechange.ValueType_FLOAT:
		gnmiValue := gnmi.TypedValue_FloatVal{FloatVal: (*devicechange.TypedFloat)(typedValue).Float32()}
		return &gnmi.TypedValue{Value: &gnmiValue}, nil

	case devicechange.ValueType_BYTES:
		gnmiValue := gnmi.TypedValue_BytesVal{BytesVal: (*devicechange.TypedBytes)(typedValue).ByteArray()}
		return &gnmi.TypedValue{Value: &gnmiValue}, nil

	case devicechange.ValueType_LEAFLIST_STRING:
		strings := (*devicechange.TypedLeafListString)(typedValue).List()
		gnmiTypedValues := make([]*gnmi.TypedValue, 0)
		for _, s := range strings {
			gnmiValue := gnmi.TypedValue_StringVal{StringVal: s}
			gnmiTypedValues = append(gnmiTypedValues, &gnmi.TypedValue{Value: &gnmiValue})
		}
		gnmiLeafList := gnmi.TypedValue_LeaflistVal{LeaflistVal: &gnmi.ScalarArray{Element: gnmiTypedValues}}
		return &gnmi.TypedValue{Value: &gnmiLeafList}, nil

	case devicechange.ValueType_LEAFLIST_INT:
		ints := (*devicechange.TypedLeafListInt64)(typedValue).List()
		gnmiTypedValues := make([]*gnmi.TypedValue, 0)
		for _, i := range ints {
			gnmiValue := gnmi.TypedValue_IntVal{IntVal: int64(i)}
			gnmiTypedValues = append(gnmiTypedValues, &gnmi.TypedValue{Value: &gnmiValue})
		}
		gnmiLeafList := gnmi.TypedValue_LeaflistVal{LeaflistVal: &gnmi.ScalarArray{Element: gnmiTypedValues}}
		return &gnmi.TypedValue{Value: &gnmiLeafList}, nil

	case devicechange.ValueType_LEAFLIST_UINT:
		uints := (*devicechange.TypedLeafListUint)(typedValue).List()
		gnmiTypedValues := make([]*gnmi.TypedValue, 0)
		for _, u := range uints {
			gnmiValue := gnmi.TypedValue_UintVal{UintVal: uint64(u)}
			gnmiTypedValues = append(gnmiTypedValues, &gnmi.TypedValue{Value: &gnmiValue})
		}
		gnmiLeafList := gnmi.TypedValue_LeaflistVal{LeaflistVal: &gnmi.ScalarArray{Element: gnmiTypedValues}}
		return &gnmi.TypedValue{Value: &gnmiLeafList}, nil

	case devicechange.ValueType_LEAFLIST_BOOL:
		bools := (*devicechange.TypedLeafListBool)(typedValue).List()
		gnmiTypedValues := make([]*gnmi.TypedValue, 0)
		for _, b := range bools {
			gnmiValue := gnmi.TypedValue_BoolVal{BoolVal: b}
			gnmiTypedValues = append(gnmiTypedValues, &gnmi.TypedValue{Value: &gnmiValue})
		}
		gnmiLeafList := gnmi.TypedValue_LeaflistVal{LeaflistVal: &gnmi.ScalarArray{Element: gnmiTypedValues}}
		return &gnmi.TypedValue{Value: &gnmiLeafList}, nil

	case devicechange.ValueType_LEAFLIST_DECIMAL:
		digits, precision := (*devicechange.TypedLeafListDecimal)(typedValue).List()
		gnmiTypedValues := make([]*gnmi.TypedValue, 0)
		for _, d := range digits {
			gnmiValue := gnmi.TypedValue_DecimalVal{DecimalVal: &gnmi.Decimal64{Digits: d, Precision: precision}}
			gnmiTypedValues = append(gnmiTypedValues, &gnmi.TypedValue{Value: &gnmiValue})
		}
		gnmiLeafList := gnmi.TypedValue_LeaflistVal{LeaflistVal: &gnmi.ScalarArray{Element: gnmiTypedValues}}
		return &gnmi.TypedValue{Value: &gnmiLeafList}, nil

	case devicechange.ValueType_LEAFLIST_FLOAT:
		floats := (*devicechange.TypedLeafListFloat)(typedValue).List()
		gnmiTypedValues := make([]*gnmi.TypedValue, 0)
		for _, f := range floats {
			gnmiValue := gnmi.TypedValue_FloatVal{FloatVal: f}
			gnmiTypedValues = append(gnmiTypedValues, &gnmi.TypedValue{Value: &gnmiValue})
		}
		gnmiLeafList := gnmi.TypedValue_LeaflistVal{LeaflistVal: &gnmi.ScalarArray{Element: gnmiTypedValues}}
		return &gnmi.TypedValue{Value: &gnmiLeafList}, nil

	case devicechange.ValueType_LEAFLIST_BYTES:
		bytes := (*devicechange.TypedLeafListBytes)(typedValue).List()
		gnmiTypedValues := make([]*gnmi.TypedValue, 0)
		for _, b := range bytes {
			gnmiValue := gnmi.TypedValue_BytesVal{BytesVal: b}
			gnmiTypedValues = append(gnmiTypedValues, &gnmi.TypedValue{Value: &gnmiValue})
		}
		gnmiLeafList := gnmi.TypedValue_LeaflistVal{LeaflistVal: &gnmi.ScalarArray{Element: gnmiTypedValues}}
		return &gnmi.TypedValue{Value: &gnmiLeafList}, nil

	default:
		return nil, fmt.Errorf("Unsupported type %d", typedValue.Type)
	}
}
