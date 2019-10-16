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
	types "github.com/onosproject/onos-config/pkg/types/change/device"
	pb "github.com/openconfig/gnmi/proto/gnmi"
)

// GnmiTypedValueToNativeType converts gnmi type based values in to native byte array types
func GnmiTypedValueToNativeType(gnmiTv *pb.TypedValue) (*types.TypedValue, error) {

	switch v := gnmiTv.GetValue().(type) {
	case *pb.TypedValue_StringVal:
		return types.NewTypedValueString(v.StringVal), nil
	case *pb.TypedValue_AsciiVal:
		return types.NewTypedValueString(v.AsciiVal), nil
	case *pb.TypedValue_IntVal:
		return types.NewTypedValueInt64(int(v.IntVal)), nil
	case *pb.TypedValue_UintVal:
		return types.NewTypedValueUint64(uint(v.UintVal)), nil
	case *pb.TypedValue_BoolVal:
		return types.NewTypedValueBool(v.BoolVal), nil
	case *pb.TypedValue_BytesVal:
		return types.NewTypedValueBytes(v.BytesVal), nil
	case *pb.TypedValue_DecimalVal:
		return types.NewTypedValueDecimal64(v.DecimalVal.Digits, v.DecimalVal.Precision), nil
	case *pb.TypedValue_FloatVal:
		return types.NewTypedValueFloat(v.FloatVal), nil
	case *pb.TypedValue_LeaflistVal:
		return handleLeafList(v)
	default:
		return nil, fmt.Errorf("Not yet supported %v", v)
	}
}

func handleLeafList(gnmiLl *pb.TypedValue_LeaflistVal) (*types.TypedValue, error) {
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
		case *pb.TypedValue_StringVal:
			stringList = append(stringList, u.StringVal)
		case *pb.TypedValue_AsciiVal:
			stringList = append(stringList, u.AsciiVal)
		case *pb.TypedValue_IntVal:
			intList = append(intList, int(u.IntVal))
		case *pb.TypedValue_UintVal:
			uintList = append(uintList, uint(u.UintVal))
		case *pb.TypedValue_BoolVal:
			boolList = append(boolList, u.BoolVal)
		case *pb.TypedValue_BytesVal:
			bytesList = append(bytesList, u.BytesVal)
		case *pb.TypedValue_DecimalVal:
			digitsList = append(digitsList, u.DecimalVal.Digits)
			precision = u.DecimalVal.Precision
		case *pb.TypedValue_FloatVal:
			floatList = append(floatList, u.FloatVal)
		default:
			return nil, fmt.Errorf("Leaf list type Not yet supported %v", u)
		}
	}

	if len(stringList) > 0 {
		return types.NewLeafListStringTv(stringList), nil
	} else if len(intList) > 0 {
		return types.NewLeafListInt64Tv(intList), nil
	} else if len(uintList) > 0 {
		return types.NewLeafListUint64Tv(uintList), nil
	} else if len(boolList) > 0 {
		return types.NewLeafListBoolTv(boolList), nil
	} else if len(bytesList) > 0 {
		return types.NewLeafListBytesTv(bytesList), nil
	} else if len(digitsList) > 0 {
		return types.NewLeafListDecimal64Tv(digitsList, precision), nil
	} else if len(floatList) > 0 {
		return types.NewLeafListFloat32Tv(floatList), nil
	}
	return nil, fmt.Errorf("Empty leaf list given")
}

// NativeTypeToGnmiTypedValue converts native byte array based values in to gnmi types
func NativeTypeToGnmiTypedValue(typedValue *types.TypedValue) (*pb.TypedValue, error) {
	if len(typedValue.Bytes) == 0 && typedValue.Type != types.ValueType_EMPTY {
		return nil, fmt.Errorf("invalid TypedValue Length 0")
	}
	switch typedValue.Type {
	case types.ValueType_EMPTY:
		gnmiValue := &pb.TypedValue_AnyVal{AnyVal: nil}
		return &pb.TypedValue{Value: gnmiValue}, nil

	case types.ValueType_STRING:
		gnmiValue := pb.TypedValue_StringVal{StringVal: (*types.TypedString)(typedValue).String()}
		return &pb.TypedValue{Value: &gnmiValue}, nil

	case types.ValueType_INT:
		gnmiValue := pb.TypedValue_IntVal{IntVal: int64((*types.TypedInt64)(typedValue).Int())}
		return &pb.TypedValue{Value: &gnmiValue}, nil

	case types.ValueType_UINT:
		gnmiValue := pb.TypedValue_UintVal{UintVal: uint64((*types.TypedUint64)(typedValue).Uint())}
		return &pb.TypedValue{Value: &gnmiValue}, nil

	case types.ValueType_BOOL:
		gnmiValue := pb.TypedValue_BoolVal{BoolVal: (*types.TypedBool)(typedValue).Bool()}
		return &pb.TypedValue{Value: &gnmiValue}, nil

	case types.ValueType_DECIMAL:
		digits, precision := (*types.TypedDecimal64)(typedValue).Decimal64()
		gnmiValue := pb.TypedValue_DecimalVal{DecimalVal: &pb.Decimal64{Digits: digits, Precision: precision}}
		return &pb.TypedValue{Value: &gnmiValue}, nil

	case types.ValueType_FLOAT:
		gnmiValue := pb.TypedValue_FloatVal{FloatVal: (*types.TypedFloat)(typedValue).Float32()}
		return &pb.TypedValue{Value: &gnmiValue}, nil

	case types.ValueType_BYTES:
		gnmiValue := pb.TypedValue_BytesVal{BytesVal: (*types.TypedBytes)(typedValue).ByteArray()}
		return &pb.TypedValue{Value: &gnmiValue}, nil

	case types.ValueType_LEAFLIST_STRING:
		strings := (*types.TypedLeafListString)(typedValue).List()
		gnmiTypedValues := make([]*pb.TypedValue, 0)
		for _, s := range strings {
			gnmiValue := pb.TypedValue_StringVal{StringVal: s}
			gnmiTypedValues = append(gnmiTypedValues, &pb.TypedValue{Value: &gnmiValue})
		}
		gnmiLeafList := pb.TypedValue_LeaflistVal{LeaflistVal: &pb.ScalarArray{Element: gnmiTypedValues}}
		return &pb.TypedValue{Value: &gnmiLeafList}, nil

	case types.ValueType_LEAFLIST_INT:
		ints := (*types.TypedLeafListInt64)(typedValue).List()
		gnmiTypedValues := make([]*pb.TypedValue, 0)
		for _, i := range ints {
			gnmiValue := pb.TypedValue_IntVal{IntVal: int64(i)}
			gnmiTypedValues = append(gnmiTypedValues, &pb.TypedValue{Value: &gnmiValue})
		}
		gnmiLeafList := pb.TypedValue_LeaflistVal{LeaflistVal: &pb.ScalarArray{Element: gnmiTypedValues}}
		return &pb.TypedValue{Value: &gnmiLeafList}, nil

	case types.ValueType_LEAFLIST_UINT:
		uints := (*types.TypedLeafListUint)(typedValue).List()
		gnmiTypedValues := make([]*pb.TypedValue, 0)
		for _, u := range uints {
			gnmiValue := pb.TypedValue_UintVal{UintVal: uint64(u)}
			gnmiTypedValues = append(gnmiTypedValues, &pb.TypedValue{Value: &gnmiValue})
		}
		gnmiLeafList := pb.TypedValue_LeaflistVal{LeaflistVal: &pb.ScalarArray{Element: gnmiTypedValues}}
		return &pb.TypedValue{Value: &gnmiLeafList}, nil

	case types.ValueType_LEAFLIST_BOOL:
		bools := (*types.TypedLeafListBool)(typedValue).List()
		gnmiTypedValues := make([]*pb.TypedValue, 0)
		for _, b := range bools {
			gnmiValue := pb.TypedValue_BoolVal{BoolVal: b}
			gnmiTypedValues = append(gnmiTypedValues, &pb.TypedValue{Value: &gnmiValue})
		}
		gnmiLeafList := pb.TypedValue_LeaflistVal{LeaflistVal: &pb.ScalarArray{Element: gnmiTypedValues}}
		return &pb.TypedValue{Value: &gnmiLeafList}, nil

	case types.ValueType_LEAFLIST_DECIMAL:
		digits, precision := (*types.TypedLeafListDecimal)(typedValue).List()
		gnmiTypedValues := make([]*pb.TypedValue, 0)
		for _, d := range digits {
			gnmiValue := pb.TypedValue_DecimalVal{DecimalVal: &pb.Decimal64{Digits: d, Precision: precision}}
			gnmiTypedValues = append(gnmiTypedValues, &pb.TypedValue{Value: &gnmiValue})
		}
		gnmiLeafList := pb.TypedValue_LeaflistVal{LeaflistVal: &pb.ScalarArray{Element: gnmiTypedValues}}
		return &pb.TypedValue{Value: &gnmiLeafList}, nil

	case types.ValueType_LEAFLIST_FLOAT:
		floats := (*types.TypedLeafListFloat)(typedValue).List()
		gnmiTypedValues := make([]*pb.TypedValue, 0)
		for _, f := range floats {
			gnmiValue := pb.TypedValue_FloatVal{FloatVal: f}
			gnmiTypedValues = append(gnmiTypedValues, &pb.TypedValue{Value: &gnmiValue})
		}
		gnmiLeafList := pb.TypedValue_LeaflistVal{LeaflistVal: &pb.ScalarArray{Element: gnmiTypedValues}}
		return &pb.TypedValue{Value: &gnmiLeafList}, nil

	case types.ValueType_LEAFLIST_BYTES:
		bytes := (*types.TypedLeafListBytes)(typedValue).List()
		gnmiTypedValues := make([]*pb.TypedValue, 0)
		for _, b := range bytes {
			gnmiValue := pb.TypedValue_BytesVal{BytesVal: b}
			gnmiTypedValues = append(gnmiTypedValues, &pb.TypedValue{Value: &gnmiValue})
		}
		gnmiLeafList := pb.TypedValue_LeaflistVal{LeaflistVal: &pb.ScalarArray{Element: gnmiTypedValues}}
		return &pb.TypedValue{Value: &gnmiLeafList}, nil

	default:
		return nil, fmt.Errorf("Unsupported type %d", typedValue.Type)
	}
}
