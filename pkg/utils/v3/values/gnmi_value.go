// SPDX-FileCopyrightText: 2020-present Open Networking Foundation <info@opennetworking.org>
//
// SPDX-License-Identifier: Apache-2.0

// Package values implements various gNMI value manipulation facilities.
package values

import (
	"fmt"

	configapi "github.com/onosproject/onos-api/go/onos/config/v3"

	gnmi "github.com/openconfig/gnmi/proto/gnmi"
)

// GnmiTypedValueToNativeType converts gnmi type based values in to native byte array changes
func GnmiTypedValueToNativeType(gnmiTv *gnmi.TypedValue, modelPath *configapi.ReadWritePath) (*configapi.TypedValue, error) {

	switch v := gnmiTv.GetValue().(type) {
	case *gnmi.TypedValue_StringVal:
		return configapi.NewTypedValueString(v.StringVal), nil
	case *gnmi.TypedValue_AsciiVal:
		return configapi.NewTypedValueString(v.AsciiVal), nil
	case *gnmi.TypedValue_IntVal:
		var intWidth = configapi.WidthThirtyTwo
		if modelPath != nil && len(modelPath.TypeOpts) > 0 {
			intWidth = configapi.Width(modelPath.TypeOpts[0])
		}
		return configapi.NewTypedValueInt(int(v.IntVal), intWidth), nil
	case *gnmi.TypedValue_UintVal:
		var uintWidth = configapi.WidthThirtyTwo
		if modelPath != nil && len(modelPath.TypeOpts) > 0 {
			uintWidth = configapi.Width(modelPath.TypeOpts[0])
		}
		return configapi.NewTypedValueUint(uint(v.UintVal), uintWidth), nil
	case *gnmi.TypedValue_BoolVal:
		return configapi.NewTypedValueBool(v.BoolVal), nil
	case *gnmi.TypedValue_BytesVal:
		return configapi.NewTypedValueBytes(v.BytesVal), nil
	case *gnmi.TypedValue_DecimalVal:
		return configapi.NewTypedValueDecimal(v.DecimalVal.Digits, uint8(v.DecimalVal.Precision)), nil
	case *gnmi.TypedValue_FloatVal:
		return configapi.NewTypedValueFloat(float64(v.FloatVal)), nil
	case *gnmi.TypedValue_LeaflistVal:
		var typeOpt0 uint64
		if modelPath != nil && len(modelPath.TypeOpts) > 0 {
			typeOpt0 = modelPath.TypeOpts[0]
		}
		return handleLeafList(v, uint8(typeOpt0))
	default:
		return nil, fmt.Errorf("not yet supported %v", v)
	}
}

// typeOpt0 could be a width in case of int or uint OR a precision in case of Decimal
func handleLeafList(gnmiLl *gnmi.TypedValue_LeaflistVal, typeOpt0 uint8) (*configapi.TypedValue, error) {
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

	var width = configapi.WidthThirtyTwo
	if typeOpt0 > 0 {
		width = configapi.Width(typeOpt0)
	}

	if len(stringList) > 0 {
		return configapi.NewLeafListStringTv(stringList), nil
	} else if len(intList) > 0 {
		return configapi.NewLeafListIntTv(intList, width), nil // TODO Width should be driven from model
	} else if len(uintList) > 0 {
		return configapi.NewLeafListUintTv(uintList, width), nil // TODO Width should be driven from model
	} else if len(boolList) > 0 {
		return configapi.NewLeafListBoolTv(boolList), nil
	} else if len(bytesList) > 0 {
		return configapi.NewLeafListBytesTv(bytesList), nil
	} else if len(digitsList) > 0 {
		return configapi.NewLeafListDecimalTv(digitsList, precision), nil
	} else if len(floatList) > 0 {
		return configapi.NewLeafListFloatTv(floatList), nil
	}
	return nil, fmt.Errorf("empty leaf list given")
}

// NativeTypeToGnmiTypedValue converts native byte array based values in to gnmi types
func NativeTypeToGnmiTypedValue(typedValue *configapi.TypedValue) (*gnmi.TypedValue, error) {
	switch typedValue.Type {
	case configapi.ValueType_EMPTY:
		gnmiValue := &gnmi.TypedValue_AnyVal{AnyVal: nil}
		return &gnmi.TypedValue{Value: gnmiValue}, nil

	case configapi.ValueType_STRING:
		gnmiValue := gnmi.TypedValue_StringVal{StringVal: (*configapi.TypedString)(typedValue).String()}
		return &gnmi.TypedValue{Value: &gnmiValue}, nil

	case configapi.ValueType_INT:
		gnmiValue := gnmi.TypedValue_IntVal{IntVal: int64((*configapi.TypedInt)(typedValue).Int())}
		return &gnmi.TypedValue{Value: &gnmiValue}, nil

	case configapi.ValueType_UINT:
		gnmiValue := gnmi.TypedValue_UintVal{UintVal: uint64((*configapi.TypedUint)(typedValue).Uint())}
		return &gnmi.TypedValue{Value: &gnmiValue}, nil

	case configapi.ValueType_BOOL:
		gnmiValue := gnmi.TypedValue_BoolVal{BoolVal: (*configapi.TypedBool)(typedValue).Bool()}
		return &gnmi.TypedValue{Value: &gnmiValue}, nil

	case configapi.ValueType_DECIMAL:
		digits, precision := (*configapi.TypedDecimal)(typedValue).Decimal64()
		gnmiValue := gnmi.TypedValue_DecimalVal{DecimalVal: &gnmi.Decimal64{Digits: digits, Precision: uint32(precision)}}
		return &gnmi.TypedValue{Value: &gnmiValue}, nil

	case configapi.ValueType_FLOAT:
		gnmiValue := gnmi.TypedValue_FloatVal{FloatVal: (*configapi.TypedFloat)(typedValue).Float32()}
		return &gnmi.TypedValue{Value: &gnmiValue}, nil

	case configapi.ValueType_BYTES:
		gnmiValue := gnmi.TypedValue_BytesVal{BytesVal: (*configapi.TypedBytes)(typedValue).ByteArray()}
		return &gnmi.TypedValue{Value: &gnmiValue}, nil

	case configapi.ValueType_LEAFLIST_STRING:
		strings := (*configapi.TypedLeafListString)(typedValue).List()
		gnmiTypedValues := make([]*gnmi.TypedValue, 0)
		for _, s := range strings {
			gnmiValue := gnmi.TypedValue_StringVal{StringVal: s}
			gnmiTypedValues = append(gnmiTypedValues, &gnmi.TypedValue{Value: &gnmiValue})
		}
		gnmiLeafList := gnmi.TypedValue_LeaflistVal{LeaflistVal: &gnmi.ScalarArray{Element: gnmiTypedValues}}
		return &gnmi.TypedValue{Value: &gnmiLeafList}, nil

	case configapi.ValueType_LEAFLIST_INT:
		ints, _ := (*configapi.TypedLeafListInt)(typedValue).List()
		gnmiTypedValues := make([]*gnmi.TypedValue, 0)
		for _, i := range ints {
			gnmiValue := gnmi.TypedValue_IntVal{IntVal: i}
			gnmiTypedValues = append(gnmiTypedValues, &gnmi.TypedValue{Value: &gnmiValue})
		}
		gnmiLeafList := gnmi.TypedValue_LeaflistVal{LeaflistVal: &gnmi.ScalarArray{Element: gnmiTypedValues}}
		return &gnmi.TypedValue{Value: &gnmiLeafList}, nil

	case configapi.ValueType_LEAFLIST_UINT:
		uints, _ := (*configapi.TypedLeafListUint)(typedValue).List()
		gnmiTypedValues := make([]*gnmi.TypedValue, 0)
		for _, u := range uints {
			gnmiValue := gnmi.TypedValue_UintVal{UintVal: u}
			gnmiTypedValues = append(gnmiTypedValues, &gnmi.TypedValue{Value: &gnmiValue})
		}
		gnmiLeafList := gnmi.TypedValue_LeaflistVal{LeaflistVal: &gnmi.ScalarArray{Element: gnmiTypedValues}}
		return &gnmi.TypedValue{Value: &gnmiLeafList}, nil

	case configapi.ValueType_LEAFLIST_BOOL:
		bools := (*configapi.TypedLeafListBool)(typedValue).List()
		gnmiTypedValues := make([]*gnmi.TypedValue, 0)
		for _, b := range bools {
			gnmiValue := gnmi.TypedValue_BoolVal{BoolVal: b}
			gnmiTypedValues = append(gnmiTypedValues, &gnmi.TypedValue{Value: &gnmiValue})
		}
		gnmiLeafList := gnmi.TypedValue_LeaflistVal{LeaflistVal: &gnmi.ScalarArray{Element: gnmiTypedValues}}
		return &gnmi.TypedValue{Value: &gnmiLeafList}, nil

	case configapi.ValueType_LEAFLIST_DECIMAL:
		digits, precision := (*configapi.TypedLeafListDecimal)(typedValue).List()
		gnmiTypedValues := make([]*gnmi.TypedValue, 0)
		for _, d := range digits {
			gnmiValue := gnmi.TypedValue_DecimalVal{DecimalVal: &gnmi.Decimal64{Digits: d, Precision: uint32(precision)}}
			gnmiTypedValues = append(gnmiTypedValues, &gnmi.TypedValue{Value: &gnmiValue})
		}
		gnmiLeafList := gnmi.TypedValue_LeaflistVal{LeaflistVal: &gnmi.ScalarArray{Element: gnmiTypedValues}}
		return &gnmi.TypedValue{Value: &gnmiLeafList}, nil

	case configapi.ValueType_LEAFLIST_FLOAT:
		floats := (*configapi.TypedLeafListFloat)(typedValue).List()
		gnmiTypedValues := make([]*gnmi.TypedValue, 0)
		for _, f := range floats {
			gnmiValue := gnmi.TypedValue_FloatVal{FloatVal: f}
			gnmiTypedValues = append(gnmiTypedValues, &gnmi.TypedValue{Value: &gnmiValue})
		}
		gnmiLeafList := gnmi.TypedValue_LeaflistVal{LeaflistVal: &gnmi.ScalarArray{Element: gnmiTypedValues}}
		return &gnmi.TypedValue{Value: &gnmiLeafList}, nil

	case configapi.ValueType_LEAFLIST_BYTES:
		bytes := (*configapi.TypedLeafListBytes)(typedValue).List()
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
