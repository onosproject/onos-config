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

package device

import (
	"encoding/base64"
	"encoding/binary"
	"fmt"
	"math"
	"strconv"
	"strings"
)

// TypedValueMap is an alias for a map of paths and values
type TypedValueMap map[string]*TypedValue

// ValueToString is the String calculated as a Native type
// There is already a String() in the protobuf generated code that can't be
// overridden
func (tv *TypedValue) ValueToString() string {
	switch tv.Type {
	case ValueType_EMPTY:
		return ""
	case ValueType_STRING:
		return (*TypedString)(tv).String()
	case ValueType_INT:
		return (*TypedInt64)(tv).String()
	case ValueType_UINT:
		return (*TypedUint64)(tv).String()
	case ValueType_BOOL:
		return (*TypedBool)(tv).String()
	case ValueType_DECIMAL:
		return (*TypedDecimal64)(tv).String()
	case ValueType_FLOAT:
		return (*TypedFloat)(tv).String()
	case ValueType_BYTES:
		return (*TypedBytes)(tv).String()
	case ValueType_LEAFLIST_STRING:
		return (*TypedLeafListString)(tv).String()
	case ValueType_LEAFLIST_INT:
		return (*TypedLeafListInt64)(tv).String()
	case ValueType_LEAFLIST_UINT:
		return (*TypedLeafListUint)(tv).String()
	case ValueType_LEAFLIST_BOOL:
		return (*TypedLeafListBool)(tv).String()
	case ValueType_LEAFLIST_DECIMAL:
		return (*TypedLeafListDecimal)(tv).String()
	case ValueType_LEAFLIST_FLOAT:
		return (*TypedLeafListFloat)(tv).String()
	case ValueType_LEAFLIST_BYTES:
		return (*TypedLeafListBytes)(tv).String()
	}

	return ""
}

// NewTypedValue creates a TypeValue from a byte[] and type - used in changes.go
func NewTypedValue(bytes []byte, valueType ValueType, typeOpts []int32) (*TypedValue, error) {
	switch valueType {
	case ValueType_EMPTY:
		return NewTypedValueEmpty(), nil
	case ValueType_STRING:
		return NewTypedValueString(string(bytes)), nil
	case ValueType_INT:
		if len(bytes) != 8 {
			return nil, fmt.Errorf("Expecting 8 bytes for INT. Got %d", len(bytes))
		}
		return NewTypedValueInt64(int(binary.LittleEndian.Uint64(bytes))), nil
	case ValueType_UINT:
		if len(bytes) != 8 {
			return nil, fmt.Errorf("Expecting 8 bytes for UINT. Got %d", len(bytes))
		}
		return NewTypedValueUint64(uint(binary.LittleEndian.Uint64(bytes))), nil
	case ValueType_BOOL:
		if len(bytes) != 1 {
			return nil, fmt.Errorf("Expecting 1 byte for BOOL. Got %d", len(bytes))
		}
		value := false
		if bytes[0] == 1 {
			value = true
		}
		return NewTypedValueBool(value), nil
	case ValueType_DECIMAL:
		if len(bytes) != 8 {
			return nil, fmt.Errorf("Expecting 8 bytes for DECIMAL. Got %d", len(bytes))
		}
		if len(typeOpts) != 1 {
			return nil, fmt.Errorf("Expecting 1 typeopt for DECIMAL. Got %d", len(typeOpts))
		}
		precision := typeOpts[0]
		return NewTypedValueDecimal64(int64(binary.LittleEndian.Uint64(bytes)), uint32(precision)), nil
	case ValueType_FLOAT:
		if len(bytes) != 8 {
			return nil, fmt.Errorf("Expecting 8 bytes for FLOAT. Got %d", len(bytes))
		}
		return NewTypedValueFloat(float32(math.Float64frombits(binary.LittleEndian.Uint64(bytes)))), nil
	case ValueType_BYTES:
		return NewTypedValueBytes(bytes), nil
	case ValueType_LEAFLIST_STRING:
		return caseValueTypeLeafListSTRING(bytes)
	case ValueType_LEAFLIST_INT:
		return caseValueTypeLeafListINT(bytes)
	case ValueType_LEAFLIST_UINT:
		return caseValueTypeLeafListUINT(bytes)
	case ValueType_LEAFLIST_BOOL:
		return caseValueTypeLeafListBOOL(bytes)
	case ValueType_LEAFLIST_DECIMAL:
		return caseValueTypeLeafListDECIMAL(bytes, typeOpts)
	case ValueType_LEAFLIST_FLOAT:
		return caseValueTypeLeafListFLOAT(bytes)
	case ValueType_LEAFLIST_BYTES:
		return caseValueTypeLeafListBYTES(bytes, typeOpts)
	}

	return nil, fmt.Errorf("unexpected type %d", valueType)
}

// caseValueTypeLeafListSTRING is moved out of NewTypedValue because of gocyclo
func caseValueTypeLeafListSTRING(bytes []byte) (*TypedValue, error) {
	stringList := make([]string, 0)
	buf := make([]byte, 0)
	for _, b := range bytes {
		if b != 0x1D {
			buf = append(buf, b)
		} else {
			stringList = append(stringList, string(buf))
			buf = make([]byte, 0)
		}
	}
	stringList = append(stringList, string(buf))
	return NewLeafListStringTv(stringList), nil
}

// caseValueTypeLeafListINT is moved out of NewTypedValue because of gocyclo
func caseValueTypeLeafListINT(bytes []byte) (*TypedValue, error) {
	count := len(bytes) / 8
	intList := make([]int, 0)

	for i := 0; i < count; i++ {
		v := bytes[i*8 : i*8+8]
		leafInt := binary.LittleEndian.Uint64(v)
		intList = append(intList, int(leafInt))
	}
	return NewLeafListInt64Tv(intList), nil
}

// caseValueTypeLeafListUINT is moved out of NewTypedValue because of gocyclo
func caseValueTypeLeafListUINT(bytes []byte) (*TypedValue, error) {
	count := len(bytes) / 8
	uintList := make([]uint, 0)

	for i := 0; i < count; i++ {
		v := bytes[i*8 : i*8+8]
		leafInt := binary.LittleEndian.Uint64(v)
		uintList = append(uintList, uint(leafInt))
	}
	return NewLeafListUint64Tv(uintList), nil
}

// caseValueTypeLeafListBOOL is moved out of NewTypedValue because of gocyclo
func caseValueTypeLeafListBOOL(bytes []byte) (*TypedValue, error) {
	count := len(bytes)
	bools := make([]bool, count)
	for i, v := range bytes {
		if v == 1 {
			bools[i] = true
		}
	}
	return NewLeafListBoolTv(bools), nil
}

// caseValueTypeLeafListDECIMAL is moved out of NewTypedValue because of gocyclo
func caseValueTypeLeafListDECIMAL(bytes []byte, typeOpts []int32) (*TypedValue, error) {
	count := len(bytes) / 8
	digitsList := make([]int64, 0)
	var precision int32

	if len(typeOpts) > 0 {
		precision = typeOpts[0]
	}
	for i := 0; i < count; i++ {
		v := bytes[i*8 : i*8+8]
		leafDigit := binary.LittleEndian.Uint64(v)
		digitsList = append(digitsList, int64(leafDigit))
	}
	return NewLeafListDecimal64Tv(digitsList, uint32(precision)), nil
}

// caseValueTypeLeafListFLOAT is moved out of NewTypedValue because of gocyclo
func caseValueTypeLeafListFLOAT(bytes []byte) (*TypedValue, error) {
	count := len(bytes) / 8
	float32s := make([]float32, 0)

	for i := 0; i < count; i++ {
		v := bytes[i*8 : i*8+8]
		float32s = append(float32s, float32(math.Float64frombits(binary.LittleEndian.Uint64(v))))
	}
	return NewLeafListFloat32Tv(float32s), nil
}

// caseValueTypeLeafListBYTES is moved out of NewTypedValue because of gocyclo
func caseValueTypeLeafListBYTES(bytes []byte, typeOpts []int32) (*TypedValue, error) {
	if len(typeOpts) < 1 {
		return nil, fmt.Errorf("Expecting 1 typeopt for LeafListBytes. Got %d", len(typeOpts))
	}
	byteArrays := make([][]byte, 0)
	buf := make([]byte, 0)
	idx := 0
	startAt := 0
	for i, b := range bytes {
		valueLen := typeOpts[idx]
		if i-startAt == int(valueLen) {
			byteArrays = append(byteArrays, buf)
			buf = make([]byte, 0)
			idx = idx + 1
			if idx >= len(typeOpts) {
				return nil, fmt.Errorf("not enough typeOpts provided - found %d need at least %d", len(typeOpts), idx+1)
			}
			startAt = startAt + int(valueLen)
		}
		buf = append(buf, b)
	}
	byteArrays = append(byteArrays, buf)
	return NewLeafListBytesTv(byteArrays), nil
}

////////////////////////////////////////////////////////////////////////////////
// TypedEmpty
////////////////////////////////////////////////////////////////////////////////

// TypedEmpty for an empty value
type TypedEmpty TypedValue

// NewTypedValueEmpty decodes an empty object
func NewTypedValueEmpty() *TypedValue {
	return (*TypedValue)(NewEmpty())
}

// NewEmpty creates an instance of the Empty type
func NewEmpty() *TypedEmpty {
	typedEmpty := TypedEmpty{
		Bytes: make([]byte, 0),
		Type:  ValueType_EMPTY,
	}
	return &typedEmpty
}

// ValueType gives the value type
func (tv *TypedEmpty) ValueType() ValueType {
	return tv.Type
}

func (tv *TypedEmpty) String() string {
	return ""
}

////////////////////////////////////////////////////////////////////////////////
// TypedString
////////////////////////////////////////////////////////////////////////////////

// TypedString for a string value
type TypedString TypedValue

// NewTypedValueString decodes string value in to an object
func NewTypedValueString(value string) *TypedValue {
	return (*TypedValue)(NewString(value))
}

// NewString decodes string value in to a String type
func NewString(value string) *TypedString {
	typedString := TypedString{
		Bytes: []byte(value),
		Type:  ValueType_STRING,
	}
	return &typedString
}

// ValueType gives the value type
func (tv *TypedString) ValueType() ValueType {
	return tv.Type
}

func (tv *TypedString) String() string {
	return string(tv.Bytes)
}

////////////////////////////////////////////////////////////////////////////////
// TypedInt64
////////////////////////////////////////////////////////////////////////////////

// TypedInt64 for an int value
type TypedInt64 TypedValue

// NewTypedValueInt64 decodes an int value in to an object
func NewTypedValueInt64(value int) *TypedValue {
	return (*TypedValue)(NewInt64(value))
}

// NewInt64 decodes an int value in to an Int type
func NewInt64(value int) *TypedInt64 {
	buf := make([]byte, 8)
	binary.LittleEndian.PutUint64(buf, uint64(value))

	typedInt64 := TypedInt64{
		Bytes: buf,
		Type:  ValueType_INT,
	}
	return &typedInt64
}

// ValueType gives the value type
func (tv *TypedInt64) ValueType() ValueType {
	return tv.Type
}

func (tv *TypedInt64) String() string {
	return fmt.Sprintf("%d", int64(binary.LittleEndian.Uint64(tv.Bytes)))
}

// Int extracts the integer value
func (tv *TypedInt64) Int() int {
	return int(binary.LittleEndian.Uint64(tv.Bytes))
}

////////////////////////////////////////////////////////////////////////////////
// TypedUint64
////////////////////////////////////////////////////////////////////////////////

// TypedUint64 for a uint value
type TypedUint64 TypedValue

// NewTypedValueUint64 decodes a uint value in to an object
func NewTypedValueUint64(value uint) *TypedValue {
	return (*TypedValue)(NewUint64(value))
}

// NewUint64 decodes a uint value in to a Uint type
func NewUint64(value uint) *TypedUint64 {
	buf := make([]byte, 8)
	binary.LittleEndian.PutUint64(buf, uint64(value))
	typedUint64 := TypedUint64{
		Bytes: buf,
		Type:  ValueType_UINT,
	}
	return &typedUint64
}

// ValueType gives the value type
func (tv *TypedUint64) ValueType() ValueType {
	return tv.Type
}

func (tv *TypedUint64) String() string {
	return fmt.Sprintf("%d", binary.LittleEndian.Uint64(tv.Bytes))
}

// Uint extracts the unsigned integer value
func (tv *TypedUint64) Uint() uint {
	return uint(binary.LittleEndian.Uint64(tv.Bytes))
}

////////////////////////////////////////////////////////////////////////////////
// TypedBool
////////////////////////////////////////////////////////////////////////////////

// TypedBool for an int value
type TypedBool TypedValue

// NewTypedValueBool decodes a bool value in to an object
func NewTypedValueBool(value bool) *TypedValue {
	return (*TypedValue)(NewBool(value))
}

// NewBool decodes a bool value in to an object
func NewBool(value bool) *TypedBool {
	buf := make([]byte, 1)
	if value {
		buf[0] = 1
	}
	typedBool := TypedBool{
		Bytes: buf,
		Type:  ValueType_BOOL,
	}
	return &typedBool
}

// ValueType gives the value type
func (tv *TypedBool) ValueType() ValueType {
	return tv.Type
}

func (tv *TypedBool) String() string {
	if tv.Bytes[0] == 1 {
		return "true"
	}
	return "false"
}

// Bool extracts the unsigned bool value
func (tv *TypedBool) Bool() bool {
	return tv.Bytes[0] == 1
}

////////////////////////////////////////////////////////////////////////////////
// TypedDecimal64
////////////////////////////////////////////////////////////////////////////////

// TypedDecimal64 for a decimal64 value
type TypedDecimal64 TypedValue

// NewTypedValueDecimal64 decodes a decimal value in to an object
func NewTypedValueDecimal64(digits int64, precision uint32) *TypedValue {
	return (*TypedValue)(NewDecimal64(digits, precision))
}

// NewDecimal64 decodes a decimal value in to a Decimal type
func NewDecimal64(digits int64, precision uint32) *TypedDecimal64 {
	buf := make([]byte, 8)
	binary.LittleEndian.PutUint64(buf, uint64(digits))
	typeOpts := []int32{int32(precision)}
	typedDecimal64 := TypedDecimal64{
		Bytes:    buf,
		Type:     ValueType_DECIMAL,
		TypeOpts: typeOpts,
	}
	return &typedDecimal64
}

// ValueType gives the value type
func (tv *TypedDecimal64) ValueType() ValueType {
	return tv.Type
}

func (tv *TypedDecimal64) String() string {
	return strDecimal64(tv.Decimal64())
}

// Decimal64 extracts the unsigned decimal value
func (tv *TypedDecimal64) Decimal64() (int64, uint32) {
	if len(tv.TypeOpts) > 0 {
		precision := tv.TypeOpts[0]
		return int64(binary.LittleEndian.Uint64(tv.Bytes)), uint32(precision)
	}
	return 0, 0
}

// Float extracts the unsigned decimal value as a float
func (tv *TypedDecimal64) Float() float64 {
	floatVal, _ := strconv.ParseFloat(strDecimal64(tv.Decimal64()), 64)
	return floatVal
}

////////////////////////////////////////////////////////////////////////////////
// TypedFloat
////////////////////////////////////////////////////////////////////////////////

// TypedFloat for a float value
type TypedFloat TypedValue

// NewTypedValueFloat decodes a decimal value in to an object
func NewTypedValueFloat(value float32) *TypedValue {
	return (*TypedValue)(NewFloat(value))
}

// NewFloat decodes a decimal value in to a Bool type
func NewFloat(value float32) *TypedFloat {
	buf := make([]byte, 8)
	binary.LittleEndian.PutUint64(buf, math.Float64bits(float64(value)))
	typedFloat := TypedFloat{
		Bytes: buf,
		Type:  ValueType_FLOAT,
	}
	return &typedFloat
}

// ValueType gives the value type
func (tv *TypedFloat) ValueType() ValueType {
	return tv.Type
}

func (tv *TypedFloat) String() string {
	return fmt.Sprintf("%f", math.Float64frombits(binary.LittleEndian.Uint64(tv.Bytes)))
}

// Float32 extracts the float value
func (tv *TypedFloat) Float32() float32 {
	if len(tv.Bytes) == 0 {
		return 0.0
	}
	return float32(math.Float64frombits(binary.LittleEndian.Uint64(tv.Bytes)))
}

////////////////////////////////////////////////////////////////////////////////
// TypedBytes
////////////////////////////////////////////////////////////////////////////////

// TypedBytes for a float value
type TypedBytes TypedValue

// NewTypedValueBytes decodes an array of bytes in to an object
func NewTypedValueBytes(value []byte) *TypedValue {
	return (*TypedValue)(NewBytes(value))
}

// NewBytes decodes an array of bytes in to a Bytes type
func NewBytes(value []byte) *TypedBytes {
	typedFloat := TypedBytes{
		Bytes:    value,
		Type:     ValueType_BYTES,
		TypeOpts: []int32{int32(len(value))},
	}
	return &typedFloat
}

// ValueType gives the value type
func (tv *TypedBytes) ValueType() ValueType {
	return tv.Type
}

func (tv *TypedBytes) String() string {
	return base64.StdEncoding.EncodeToString(tv.Bytes)
}

// ByteArray extracts the bytes value
func (tv *TypedBytes) ByteArray() []byte {
	return tv.Bytes
}

////////////////////////////////////////////////////////////////////////////////
// TypedLeafListString
////////////////////////////////////////////////////////////////////////////////

// TypedLeafListString for a string leaf list
type TypedLeafListString TypedValue

// NewLeafListStringTv decodes string values in to an object
func NewLeafListStringTv(values []string) *TypedValue {
	return (*TypedValue)(NewLeafListString(values))
}

// NewLeafListString decodes string values in to an Leaf list type
func NewLeafListString(values []string) *TypedLeafListString {
	first := true
	bytes := make([]byte, 0)
	for _, v := range values {
		if first {
			first = false
		} else {
			bytes = append(bytes, 0x1D) // Group separator
		}
		bytes = append(bytes, []byte(v)...)
	}
	typedLeafListString := TypedLeafListString{
		Bytes: bytes,
		Type:  ValueType_LEAFLIST_STRING,
	}
	return &typedLeafListString
}

// ValueType gives the value type
func (tv *TypedLeafListString) ValueType() ValueType {
	return tv.Type
}

func (tv *TypedLeafListString) String() string {
	return strings.Join(tv.List(), ",")
}

// List extracts the leaf list values
func (tv *TypedLeafListString) List() []string {
	stringList := make([]string, 0)
	buf := make([]byte, 0)
	for _, b := range tv.Bytes {
		if b != 0x1D {
			buf = append(buf, b)
		} else {
			stringList = append(stringList, string(buf))
			buf = make([]byte, 0)
		}
	}
	stringList = append(stringList, string(buf))
	return stringList
}

////////////////////////////////////////////////////////////////////////////////
// TypedLeafListInt64
////////////////////////////////////////////////////////////////////////////////

// TypedLeafListInt64 for an int leaf list
type TypedLeafListInt64 TypedValue

// NewLeafListInt64Tv decodes int values in to an object
func NewLeafListInt64Tv(values []int) *TypedValue {
	return (*TypedValue)(NewLeafListInt64(values))
}

// NewLeafListInt64 decodes int values in to a Leaf list type
func NewLeafListInt64(values []int) *TypedLeafListInt64 {
	bytes := make([]byte, 0)
	for _, v := range values {
		buf := make([]byte, 8)
		binary.LittleEndian.PutUint64(buf, uint64(v))
		bytes = append(bytes, buf...)
	}
	typedLeafListInt64 := TypedLeafListInt64{
		Bytes: bytes,
		Type:  ValueType_LEAFLIST_INT,
	}
	return &typedLeafListInt64
}

// ValueType gives the value type
func (tv *TypedLeafListInt64) ValueType() ValueType {
	return tv.Type
}

func (tv *TypedLeafListInt64) String() string {
	return fmt.Sprintf("%v", tv.List())
}

// List extracts the leaf list values
func (tv *TypedLeafListInt64) List() []int {
	count := len(tv.Bytes) / 8
	intList := make([]int, 0)

	for i := 0; i < count; i++ {
		v := tv.Bytes[i*8 : i*8+8]
		leafInt := binary.LittleEndian.Uint64(v)
		intList = append(intList, int(leafInt))
	}

	return intList
}

////////////////////////////////////////////////////////////////////////////////
// TypedLeafListUint
////////////////////////////////////////////////////////////////////////////////

// TypedLeafListUint for an uint leaf list
type TypedLeafListUint TypedValue

// NewLeafListUint64Tv decodes uint values in to a Leaf list
func NewLeafListUint64Tv(values []uint) *TypedValue {
	return (*TypedValue)(NewLeafListUint64(values))
}

// NewLeafListUint64 decodes uint values in to a Leaf list type
func NewLeafListUint64(values []uint) *TypedLeafListUint {
	bytes := make([]byte, 0)
	for _, v := range values {
		buf := make([]byte, 8)
		binary.LittleEndian.PutUint64(buf, uint64(v))
		bytes = append(bytes, buf...)
	}
	typedLeafListUint := TypedLeafListUint{
		Bytes: bytes,
		Type:  ValueType_LEAFLIST_UINT,
	}
	return &typedLeafListUint
}

// ValueType gives the value type
func (tv *TypedLeafListUint) ValueType() ValueType {
	return tv.Type
}

func (tv *TypedLeafListUint) String() string {
	return fmt.Sprintf("%v", tv.List())
}

// List extracts the leaf list values
func (tv *TypedLeafListUint) List() []uint {
	count := len(tv.Bytes) / 8
	uintList := make([]uint, 0)

	for i := 0; i < count; i++ {
		v := tv.Bytes[i*8 : i*8+8]
		leafInt := binary.LittleEndian.Uint64(v)
		uintList = append(uintList, uint(leafInt))
	}

	return uintList
}

////////////////////////////////////////////////////////////////////////////////
// TypedLeafListBool
////////////////////////////////////////////////////////////////////////////////

// TypedLeafListBool for an bool leaf list
type TypedLeafListBool TypedValue

// NewLeafListBoolTv decodes bool values in to an object
func NewLeafListBoolTv(values []bool) *TypedValue {
	return (*TypedValue)(NewLeafListBool(values))
}

// NewLeafListBool decodes bool values in to a Leaf list type
func NewLeafListBool(values []bool) *TypedLeafListBool {
	count := len(values)
	bytes := make([]byte, count)
	for i, b := range values {
		// just use one byte per bool - inefficient but not worth the hassle
		var intval uint8
		if b {
			intval = 1
		}
		bytes[i] = intval
	}
	typedLeafListBool := TypedLeafListBool{
		Bytes: bytes,
		Type:  ValueType_LEAFLIST_BOOL,
	}
	return &typedLeafListBool
}

// ValueType gives the value type
func (tv *TypedLeafListBool) ValueType() ValueType {
	return tv.Type
}

func (tv *TypedLeafListBool) String() string {
	return fmt.Sprintf("%v", tv.List())
}

// List extracts the leaf list values
func (tv *TypedLeafListBool) List() []bool {
	count := len(tv.Bytes)
	bools := make([]bool, count)
	for i, v := range tv.Bytes {
		if v == 1 {
			bools[i] = true
		}
	}

	return bools
}

////////////////////////////////////////////////////////////////////////////////
// TypedLeafListDecimal
////////////////////////////////////////////////////////////////////////////////

// TypedLeafListDecimal for a decimal leaf list
type TypedLeafListDecimal TypedValue

// NewLeafListDecimal64Tv decodes decimal values in to a Leaf list
func NewLeafListDecimal64Tv(digits []int64, precision uint32) *TypedValue {
	return (*TypedValue)(NewLeafListDecimal64(digits, precision))
}

// NewLeafListDecimal64 decodes decimal values in to a Leaf list type
func NewLeafListDecimal64(digits []int64, precision uint32) *TypedLeafListDecimal {
	bytes := make([]byte, 0)
	for _, d := range digits {
		buf := make([]byte, 8)
		binary.LittleEndian.PutUint64(buf, uint64(d))
		bytes = append(bytes, buf...)
	}
	typedLeafListDecimal := TypedLeafListDecimal{
		Bytes:    bytes,
		Type:     ValueType_LEAFLIST_DECIMAL,
		TypeOpts: []int32{int32(precision)},
	}
	return &typedLeafListDecimal
}

// ValueType gives the value type
func (tv *TypedLeafListDecimal) ValueType() ValueType {
	return tv.Type
}

func (tv *TypedLeafListDecimal) String() string {
	digits, precision := tv.List()
	return fmt.Sprintf("%v %d", digits, precision)
}

// List extracts the leaf list values
func (tv *TypedLeafListDecimal) List() ([]int64, uint32) {
	count := len(tv.Bytes) / 8
	digitsList := make([]int64, 0)
	var precision int32

	if len(tv.TypeOpts) > 0 {
		precision = tv.TypeOpts[0]
	}

	for i := 0; i < count; i++ {
		v := tv.Bytes[i*8 : i*8+8]
		leafDigit := binary.LittleEndian.Uint64(v)
		digitsList = append(digitsList, int64(leafDigit))
	}

	return digitsList, uint32(precision)
}

// ListFloat extracts the leaf list values as floats
func (tv *TypedLeafListDecimal) ListFloat() []float32 {
	count := len(tv.Bytes) / 8
	var precision int32
	floatList := make([]float32, 0)

	if len(tv.TypeOpts) > 0 {
		precision = tv.TypeOpts[0]
	}

	for i := 0; i < count; i++ {
		v := tv.Bytes[i*8 : i*8+8]
		leafDigit := binary.LittleEndian.Uint64(v)
		floatVal, _ := strconv.ParseFloat(strDecimal64(int64(leafDigit), uint32(precision)), 64)
		floatList = append(floatList, float32(floatVal))
	}

	return floatList
}

////////////////////////////////////////////////////////////////////////////////
// TypedLeafListFloat
////////////////////////////////////////////////////////////////////////////////

// TypedLeafListFloat for a decimal leaf list
type TypedLeafListFloat TypedValue

// NewLeafListFloat32Tv decodes float values in to a Leaf list
func NewLeafListFloat32Tv(values []float32) *TypedValue {
	return (*TypedValue)(NewLeafListFloat32(values))
}

// NewLeafListFloat32 decodes float values in to a Leaf list type
func NewLeafListFloat32(values []float32) *TypedLeafListFloat {
	bytes := make([]byte, 0)
	for _, f := range values {
		buf := make([]byte, 8)
		binary.LittleEndian.PutUint64(buf, math.Float64bits(float64(f)))
		bytes = append(bytes, buf...)
	}
	typedLeafListFloat := TypedLeafListFloat{
		Bytes: bytes,
		Type:  ValueType_LEAFLIST_FLOAT,
	}
	return &typedLeafListFloat
}

// ValueType gives the value type
func (tv *TypedLeafListFloat) ValueType() ValueType {
	return tv.Type
}

func (tv *TypedLeafListFloat) String() string {
	listStr := make([]string, 0)
	for _, f := range tv.List() {
		listStr = append(listStr, fmt.Sprintf("%f", f))
	}

	return strings.Join(listStr, ",")
}

// List extracts the leaf list values
func (tv *TypedLeafListFloat) List() []float32 {
	count := len(tv.Bytes) / 8
	float32s := make([]float32, 0)

	for i := 0; i < count; i++ {
		v := tv.Bytes[i*8 : i*8+8]
		float32s = append(float32s, float32(math.Float64frombits(binary.LittleEndian.Uint64(v))))
	}

	return float32s
}

////////////////////////////////////////////////////////////////////////////////
// TypedLeafListBytes
////////////////////////////////////////////////////////////////////////////////

// TypedLeafListBytes for an bool leaf list
type TypedLeafListBytes TypedValue

// NewLeafListBytesTv decodes byte values in to a Leaf list
func NewLeafListBytesTv(values [][]byte) *TypedValue {
	return (*TypedValue)(NewLeafListBytes(values))
}

// NewLeafListBytes decodes byte values in to a Leaf list type
func NewLeafListBytes(values [][]byte) *TypedLeafListBytes {
	bytes := make([]byte, 0)
	typeopts := make([]int32, 0)
	for _, v := range values {
		bytes = append(bytes, v...)
		typeopts = append(typeopts, int32(len(v)))
	}
	typedLeafListBytes := TypedLeafListBytes{
		Bytes:    bytes,
		Type:     ValueType_LEAFLIST_BYTES, // Contains the lengths of each byte array in list
		TypeOpts: typeopts,
	}
	return &typedLeafListBytes
}

// ValueType gives the value type
func (tv *TypedLeafListBytes) ValueType() ValueType {
	return tv.Type
}

func (tv *TypedLeafListBytes) String() string {
	return fmt.Sprintf("%v", tv.List())
}

// List extracts the leaf list values
func (tv *TypedLeafListBytes) List() [][]byte {
	bytes := make([][]byte, 0)
	buf := make([]byte, 0)
	idx := 0
	startAt := 0
	for i, b := range tv.Bytes {
		valueLen := tv.TypeOpts[idx]
		if i-startAt == int(valueLen) {
			bytes = append(bytes, buf)
			buf = make([]byte, 0)
			idx = idx + 1
			startAt = startAt + int(valueLen)
		}
		buf = append(buf, b)
	}
	bytes = append(bytes, buf)
	return bytes
}

func strDecimal64(digits int64, precision uint32) string {
	var i, frac int64
	if precision > 0 {
		div := int64(10)
		it := precision - 1
		for it > 0 {
			div *= 10
			it--
		}
		i = digits / div
		frac = digits % div
	} else {
		i = digits
	}
	if frac < 0 {
		frac = -frac
	}
	if precision == 0 {
		return fmt.Sprintf("%d", i)
	}
	fmtString := fmt.Sprintf("%s0.%d%s", "%d.%", precision, "d")
	return fmt.Sprintf(fmtString, i, frac)
}
