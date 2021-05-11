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
	devicechange "github.com/onosproject/onos-api/go/onos/config/change/device"
	"github.com/onosproject/onos-config/pkg/modelregistry"
	gnmi "github.com/openconfig/gnmi/proto/gnmi"
)

// GnmiTypedValueToNativeType converts gnmi type based values in to native byte array devicechange
func GnmiTypedValueToNativeType(gnmiTv *gnmi.TypedValue, modelPath *modelregistry.ReadWritePathElem) (*devicechange.TypedValue, error) {

	switch v := gnmiTv.GetValue().(type) {
	case *gnmi.TypedValue_StringVal:
		return devicechange.NewTypedValueString(v.StringVal), nil
	case *gnmi.TypedValue_AsciiVal:
		return devicechange.NewTypedValueString(v.AsciiVal), nil
	case *gnmi.TypedValue_IntVal:
		var intWidth = devicechange.WidthThirtyTwo
		if modelPath != nil && len(modelPath.TypeOpts) > 0 {
			intWidth = devicechange.Width(modelPath.TypeOpts[0])
		}
		return devicechange.NewTypedValueInt(int(v.IntVal), intWidth), nil
	case *gnmi.TypedValue_UintVal:
		var uintWidth = devicechange.WidthThirtyTwo
		if modelPath != nil && len(modelPath.TypeOpts) > 0 {
			uintWidth = devicechange.Width(modelPath.TypeOpts[0])
		}
		return devicechange.NewTypedValueUint(uint(v.UintVal), uintWidth), nil
	case *gnmi.TypedValue_BoolVal:
		return devicechange.NewTypedValueBool(v.BoolVal), nil
	case *gnmi.TypedValue_BytesVal:
		return devicechange.NewTypedValueBytes(v.BytesVal), nil
	case *gnmi.TypedValue_DecimalVal:
		return devicechange.NewTypedValueDecimal(v.DecimalVal.Digits, uint8(v.DecimalVal.Precision)), nil
	case *gnmi.TypedValue_FloatVal:
		return devicechange.NewTypedValueFloat(float64(v.FloatVal)), nil
	case *gnmi.TypedValue_LeaflistVal:
		var typeOpt0 uint8 = 0
		if modelPath != nil && len(modelPath.TypeOpts) > 0 {
			typeOpt0 = modelPath.TypeOpts[0]
		}
		return handleLeafList(v, typeOpt0)
	default:
		return nil, fmt.Errorf("not yet supported %v", v)
	}
}

// typeOpt0 could be a width in case of int or uint OR a precision in case of Decimal
func handleLeafList(gnmiLl *gnmi.TypedValue_LeaflistVal, typeOpt0 uint8) (*devicechange.TypedValue, error) {
	stringList := make([]string, 0)
	intList := make([]int64, 0)
	uintList := make([]uint64, 0)
	boolList := make([]bool, 0)
	bytesList := make([][]byte, 0)
	digitsList := make([]int64, 0)
	precision := typeOpt0
	floatList := make([]float32, 0)

	// All values will be of the same type, but we don't know what that is yet
	for _, leaf := range gnmiLl.LeaflistVal.GetElement() {
		switch u := leaf.GetValue().(type) {
		case *gnmi.TypedValue_StringVal:
			stringList = append(stringList, u.StringVal)
		case *gnmi.TypedValue_AsciiVal:
			stringList = append(stringList, u.AsciiVal)
		case *gnmi.TypedValue_IntVal:
			intList = append(intList, u.IntVal)
		case *gnmi.TypedValue_UintVal:
			uintList = append(uintList, u.UintVal)
		case *gnmi.TypedValue_BoolVal:
			boolList = append(boolList, u.BoolVal)
		case *gnmi.TypedValue_BytesVal:
			bytesList = append(bytesList, u.BytesVal)
		case *gnmi.TypedValue_DecimalVal:
			digitsList = append(digitsList, u.DecimalVal.Digits)
			precision = uint8(u.DecimalVal.Precision)
		case *gnmi.TypedValue_FloatVal:
			floatList = append(floatList, u.FloatVal)
		default:
			return nil, fmt.Errorf("leaf list type Not yet supported %v", u)
		}
	}

	var width = devicechange.WidthThirtyTwo
	if typeOpt0 > 0 {
		width = devicechange.Width(typeOpt0)
	}

	if len(stringList) > 0 {
		return devicechange.NewLeafListStringTv(stringList), nil
	} else if len(intList) > 0 {
		return devicechange.NewLeafListIntTv(intList, width), nil // TODO Width should be driven from model
	} else if len(uintList) > 0 {
		return devicechange.NewLeafListUintTv(uintList, width), nil // TODO Width should be driven from model
	} else if len(boolList) > 0 {
		return devicechange.NewLeafListBoolTv(boolList), nil
	} else if len(bytesList) > 0 {
		return devicechange.NewLeafListBytesTv(bytesList), nil
	} else if len(digitsList) > 0 {
		return devicechange.NewLeafListDecimalTv(digitsList, precision), nil
	} else if len(floatList) > 0 {
		return devicechange.NewLeafListFloatTv(floatList), nil
	}
	return nil, fmt.Errorf("empty leaf list given")
}

// NativeTypeToGnmiTypedValue converts native byte array based values in to gnmi types
func NativeTypeToGnmiTypedValue(typedValue *devicechange.TypedValue) (*gnmi.TypedValue, error) {
	switch typedValue.Type {
	case devicechange.ValueType_EMPTY:
		gnmiValue := &gnmi.TypedValue_AnyVal{AnyVal: nil}
		return &gnmi.TypedValue{Value: gnmiValue}, nil

	case devicechange.ValueType_STRING:
		gnmiValue := gnmi.TypedValue_StringVal{StringVal: (*devicechange.TypedString)(typedValue).String()}
		return &gnmi.TypedValue{Value: &gnmiValue}, nil

	case devicechange.ValueType_INT:
		gnmiValue := gnmi.TypedValue_IntVal{IntVal: int64((*devicechange.TypedInt)(typedValue).Int())}
		return &gnmi.TypedValue{Value: &gnmiValue}, nil

	case devicechange.ValueType_UINT:
		gnmiValue := gnmi.TypedValue_UintVal{UintVal: uint64((*devicechange.TypedUint)(typedValue).Uint())}
		return &gnmi.TypedValue{Value: &gnmiValue}, nil

	case devicechange.ValueType_BOOL:
		gnmiValue := gnmi.TypedValue_BoolVal{BoolVal: (*devicechange.TypedBool)(typedValue).Bool()}
		return &gnmi.TypedValue{Value: &gnmiValue}, nil

	case devicechange.ValueType_DECIMAL:
		digits, precision := (*devicechange.TypedDecimal)(typedValue).Decimal64()
		gnmiValue := gnmi.TypedValue_DecimalVal{DecimalVal: &gnmi.Decimal64{Digits: digits, Precision: uint32(precision)}}
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
		ints, _ := (*devicechange.TypedLeafListInt)(typedValue).List()
		gnmiTypedValues := make([]*gnmi.TypedValue, 0)
		for _, i := range ints {
			gnmiValue := gnmi.TypedValue_IntVal{IntVal: i}
			gnmiTypedValues = append(gnmiTypedValues, &gnmi.TypedValue{Value: &gnmiValue})
		}
		gnmiLeafList := gnmi.TypedValue_LeaflistVal{LeaflistVal: &gnmi.ScalarArray{Element: gnmiTypedValues}}
		return &gnmi.TypedValue{Value: &gnmiLeafList}, nil

	case devicechange.ValueType_LEAFLIST_UINT:
		uints, _ := (*devicechange.TypedLeafListUint)(typedValue).List()
		gnmiTypedValues := make([]*gnmi.TypedValue, 0)
		for _, u := range uints {
			gnmiValue := gnmi.TypedValue_UintVal{UintVal: u}
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
			gnmiValue := gnmi.TypedValue_DecimalVal{DecimalVal: &gnmi.Decimal64{Digits: d, Precision: uint32(precision)}}
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
