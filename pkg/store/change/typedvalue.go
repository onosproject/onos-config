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
	"encoding/binary"
	"fmt"
	"math"
	"strconv"
	"strings"
)

// TypedValueMap is an alias for a map of paths and values
type TypedValueMap map[string]*TypedValue

// Types given here are a rough approximation of those in the set of YANG types
// and the set of gNMI types

// ValueType is an enum for noting the type of the value
type ValueType int

const (
	// ValueTypeEMPTY for empty type
	ValueTypeEMPTY ValueType = 0
	// ValueTypeSTRING for string type
	ValueTypeSTRING ValueType = 1
	// ValueTypeINT for int type
	ValueTypeINT ValueType = 2
	// ValueTypeUINT for uint type
	ValueTypeUINT ValueType = 3
	// ValueTypeBOOL for bool type
	ValueTypeBOOL ValueType = 4
	// ValueTypeDECIMAL for decimal type
	ValueTypeDECIMAL ValueType = 5
	// ValueTypeFLOAT for float type
	ValueTypeFLOAT ValueType = 6
	// ValueTypeBYTES for bytes type
	ValueTypeBYTES ValueType = 7
	// ValueTypeLeafListSTRING for string leaf list
	ValueTypeLeafListSTRING ValueType = 8
	// ValueTypeLeafListINT for int leaf list
	ValueTypeLeafListINT ValueType = 9
	// ValueTypeLeafListUINT for uint leaf list
	ValueTypeLeafListUINT ValueType = 10
	// ValueTypeLeafListBOOL for bool leaf list
	ValueTypeLeafListBOOL ValueType = 11
	// ValueTypeLeafListDECIMAL for decimal leaf list
	ValueTypeLeafListDECIMAL ValueType = 12
	// ValueTypeLeafListFLOAT for float leaf list
	ValueTypeLeafListFLOAT ValueType = 13
	// ValueTypeLeafListBYTES for bytes leaf list
	ValueTypeLeafListBYTES ValueType = 14
)

// TypedValue is a of a value, a type and a LeafList flag
type TypedValue struct {
	Value    []byte
	Type     ValueType
	TypeOpts []int
}

func (tv *TypedValue) String() string {
	switch tv.Type {
	case ValueTypeEMPTY:
		return ""
	case ValueTypeSTRING:
		return (*TypedString)(tv).String()
	case ValueTypeINT:
		return (*TypedInt64)(tv).String()
	case ValueTypeUINT:
		return (*TypedUint64)(tv).String()
	case ValueTypeBOOL:
		return (*TypedBool)(tv).String()
	case ValueTypeDECIMAL:
		return (*TypedDecimal64)(tv).String()
	case ValueTypeFLOAT:
		return (*TypedFloat)(tv).String()
	case ValueTypeBYTES:
		return (*TypedBytes)(tv).String()
	case ValueTypeLeafListSTRING:
		return (*TypedLeafListString)(tv).String()
	case ValueTypeLeafListINT:
		return (*TypedLeafListInt64)(tv).String()
	case ValueTypeLeafListUINT:
		return (*TypedLeafListUint)(tv).String()
	case ValueTypeLeafListBOOL:
		return (*TypedLeafListBool)(tv).String()
	case ValueTypeLeafListDECIMAL:
		return (*TypedLeafListDecimal)(tv).String()
	case ValueTypeLeafListFLOAT:
		return (*TypedLeafListFloat)(tv).String()
	case ValueTypeLeafListBYTES:
		return (*TypedLeafListBytes)(tv).String()
	}

	return ""
}

// CreateTypedValue creates a TypeValue from a byte[] and type - used in changes.go
func CreateTypedValue(bytes []byte, valueType ValueType, typeOpts []int) (*TypedValue, error) {
	switch valueType {
	case ValueTypeEMPTY:
		return CreateTypedValueEmpty(), nil
	case ValueTypeSTRING:
		return CreateTypedValueString(string(bytes)), nil
	case ValueTypeINT:
		if len(bytes) != 8 {
			return nil, fmt.Errorf("Expecting 8 bytes for INT. Got %d", len(bytes))
		}
		return CreateTypedValueInt64(int(binary.LittleEndian.Uint64(bytes))), nil
	case ValueTypeUINT:
		if len(bytes) != 8 {
			return nil, fmt.Errorf("Expecting 8 bytes for UINT. Got %d", len(bytes))
		}
		return CreateTypedValueUint64(uint(binary.LittleEndian.Uint64(bytes))), nil
	case ValueTypeBOOL:
		if len(bytes) != 1 {
			return nil, fmt.Errorf("Expecting 1 byte for BOOL. Got %d", len(bytes))
		}
		value := false
		if bytes[0] == 1 {
			value = true
		}
		return CreateTypedValueBool(value), nil
	case ValueTypeDECIMAL:
		if len(bytes) != 8 {
			return nil, fmt.Errorf("Expecting 8 bytes for DECIMAL. Got %d", len(bytes))
		}
		if len(typeOpts) != 1 {
			return nil, fmt.Errorf("Expecting 1 typeopt for DECIMAL. Got %d", len(typeOpts))
		}
		precision := typeOpts[0]
		return CreateTypedValueDecimal64(int64(binary.LittleEndian.Uint64(bytes)), uint32(precision)), nil
	case ValueTypeFLOAT:
		if len(bytes) != 8 {
			return nil, fmt.Errorf("Expecting 8 bytes for FLOAT. Got %d", len(bytes))
		}
		return CreateTypedValueFloat(float32(math.Float64frombits(binary.LittleEndian.Uint64(bytes)))), nil
	case ValueTypeBYTES:
		return CreateTypedValueBytes(bytes), nil
	case ValueTypeLeafListSTRING:
		return caseValueTypeLeafListSTRING(bytes)
	case ValueTypeLeafListINT:
		return caseValueTypeLeafListINT(bytes)
	case ValueTypeLeafListUINT:
		return caseValueTypeLeafListUINT(bytes)
	case ValueTypeLeafListBOOL:
		return caseValueTypeLeafListBOOL(bytes)
	case ValueTypeLeafListDECIMAL:
		return caseValueTypeLeafListDECIMAL(bytes, typeOpts)
	case ValueTypeLeafListFLOAT:
		return caseValueTypeLeafListFLOAT(bytes)
	case ValueTypeLeafListBYTES:
		return caseValueTypeLeafListBYTES(bytes, typeOpts)
	}

	return nil, fmt.Errorf("unexpected type %d", valueType)
}

// caseValueTypeLeafListSTRING is moved out of CreateTypedValue because of gocyclo
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
	return CreateLeafListString(stringList), nil
}

// caseValueTypeLeafListINT is moved out of CreateTypedValue because of gocyclo
func caseValueTypeLeafListINT(bytes []byte) (*TypedValue, error) {
	count := len(bytes) / 8
	intList := make([]int, 0)

	for i := 0; i < count; i++ {
		v := bytes[i*8 : i*8+8]
		leafInt := binary.LittleEndian.Uint64(v)
		intList = append(intList, int(leafInt))
	}
	return CreateLeafListInt64(intList), nil
}

// caseValueTypeLeafListUINT is moved out of CreateTypedValue because of gocyclo
func caseValueTypeLeafListUINT(bytes []byte) (*TypedValue, error) {
	count := len(bytes) / 8
	uintList := make([]uint, 0)

	for i := 0; i < count; i++ {
		v := bytes[i*8 : i*8+8]
		leafInt := binary.LittleEndian.Uint64(v)
		uintList = append(uintList, uint(leafInt))
	}
	return CreateLeafListUint64(uintList), nil
}

// caseValueTypeLeafListBOOL is moved out of CreateTypedValue because of gocyclo
func caseValueTypeLeafListBOOL(bytes []byte) (*TypedValue, error) {
	count := len(bytes)
	bools := make([]bool, count)
	for i, v := range bytes {
		if v == 1 {
			bools[i] = true
		}
	}
	return CreateLeafListBool(bools), nil
}

// caseValueTypeLeafListDECIMAL is moved out of CreateTypedValue because of gocyclo
func caseValueTypeLeafListDECIMAL(bytes []byte, typeOpts []int) (*TypedValue, error) {
	count := len(bytes) / 8
	digitsList := make([]int64, 0)
	precision := 0

	if len(typeOpts) > 0 {
		precision = typeOpts[0]
	}
	for i := 0; i < count; i++ {
		v := bytes[i*8 : i*8+8]
		leafDigit := binary.LittleEndian.Uint64(v)
		digitsList = append(digitsList, int64(leafDigit))
	}
	return CreateLeafListDecimal64(digitsList, uint32(precision)), nil
}

// caseValueTypeLeafListFLOAT is moved out of CreateTypedValue because of gocyclo
func caseValueTypeLeafListFLOAT(bytes []byte) (*TypedValue, error) {
	count := len(bytes) / 8
	float32s := make([]float32, 0)

	for i := 0; i < count; i++ {
		v := bytes[i*8 : i*8+8]
		float32s = append(float32s, float32(math.Float64frombits(binary.LittleEndian.Uint64(v))))
	}
	return CreateLeafListFloat32(float32s), nil
}

// caseValueTypeLeafListBYTES is moved out of CreateTypedValue because of gocyclo
func caseValueTypeLeafListBYTES(bytes []byte, typeOpts []int) (*TypedValue, error) {
	if len(typeOpts) < 1 {
		return nil, fmt.Errorf("Expecting 1 typeopt for LeafListBytes. Got %d", len(typeOpts))
	}
	byteArrays := make([][]byte, 0)
	buf := make([]byte, 0)
	idx := 0
	startAt := 0
	for i, b := range bytes {
		valueLen := typeOpts[idx]
		if i-startAt == valueLen {
			byteArrays = append(byteArrays, buf)
			buf = make([]byte, 0)
			idx = idx + 1
			startAt = startAt + valueLen
		}
		buf = append(buf, b)
	}
	byteArrays = append(byteArrays, buf)
	return CreateLeafListBytes(byteArrays), nil
}

////////////////////////////////////////////////////////////////////////////////
// TypedEmpty
////////////////////////////////////////////////////////////////////////////////

// TypedEmpty for an empty value
type TypedEmpty TypedValue

// CreateTypedValueEmpty decodes an empty object
func CreateTypedValueEmpty() *TypedValue {
	return (*TypedValue)(NewEmpty())
}

// NewEmpty creates an instance of the Empty type
func NewEmpty() *TypedEmpty {
	typedEmpty := TypedEmpty{
		Value: make([]byte, 0),
		Type:  ValueTypeEMPTY,
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

// CreateTypedValueString decodes string value in to an object
func CreateTypedValueString(value string) *TypedValue {
	return (*TypedValue)(NewString(value))
}

// NewString decodes string value in to a String type
func NewString(value string) *TypedString {
	typedString := TypedString{
		Value: []byte(value),
		Type:  ValueTypeSTRING,
	}
	return &typedString
}

// ValueType gives the value type
func (tv *TypedString) ValueType() ValueType {
	return tv.Type
}

func (tv *TypedString) String() string {
	return string(tv.Value)
}

////////////////////////////////////////////////////////////////////////////////
// TypedInt64
////////////////////////////////////////////////////////////////////////////////

// TypedInt64 for an int value
type TypedInt64 TypedValue

// CreateTypedValueInt64 decodes an int value in to an object
func CreateTypedValueInt64(value int) *TypedValue {
	return (*TypedValue)(NewInt64(value))
}

// NewInt64 decodes an int value in to an Int type
func NewInt64(value int) *TypedInt64 {
	buf := make([]byte, 8)
	binary.LittleEndian.PutUint64(buf, uint64(value))

	typedInt64 := TypedInt64{
		Value: buf,
		Type:  ValueTypeINT,
	}
	return &typedInt64
}

// ValueType gives the value type
func (tv *TypedInt64) ValueType() ValueType {
	return tv.Type
}

func (tv *TypedInt64) String() string {
	return fmt.Sprintf("%d", int64(binary.LittleEndian.Uint64(tv.Value)))
}

// Int extracts the integer value
func (tv *TypedInt64) Int() int {
	return int(binary.LittleEndian.Uint64(tv.Value))
}

////////////////////////////////////////////////////////////////////////////////
// TypedUint64
////////////////////////////////////////////////////////////////////////////////

// TypedUint64 for a uint value
type TypedUint64 TypedValue

// CreateTypedValueUint64 decodes a uint value in to an object
func CreateTypedValueUint64(value uint) *TypedValue {
	return (*TypedValue)(NewUint64(value))
}

// NewUint64 decodes a uint value in to a Uint type
func NewUint64(value uint) *TypedUint64 {
	buf := make([]byte, 8)
	binary.LittleEndian.PutUint64(buf, uint64(value))
	typedUint64 := TypedUint64{
		Value: buf,
		Type:  ValueTypeUINT,
	}
	return &typedUint64
}

// ValueType gives the value type
func (tv *TypedUint64) ValueType() ValueType {
	return tv.Type
}

func (tv *TypedUint64) String() string {
	return fmt.Sprintf("%d", binary.LittleEndian.Uint64(tv.Value))
}

// Uint extracts the unsigned integer value
func (tv *TypedUint64) Uint() uint {
	return uint(binary.LittleEndian.Uint64(tv.Value))
}

////////////////////////////////////////////////////////////////////////////////
// TypedBool
////////////////////////////////////////////////////////////////////////////////

// TypedBool for an int value
type TypedBool TypedValue

// CreateTypedValueBool decodes a bool value in to an object
func CreateTypedValueBool(value bool) *TypedValue {
	return (*TypedValue)(NewBool(value))
}

// NewBool decodes a bool value in to an object
func NewBool(value bool) *TypedBool {
	buf := make([]byte, 1)
	if value {
		buf[0] = 1
	}
	typedBool := TypedBool{
		Value: buf,
		Type:  ValueTypeBOOL,
	}
	return &typedBool
}

// ValueType gives the value type
func (tv *TypedBool) ValueType() ValueType {
	return tv.Type
}

func (tv *TypedBool) String() string {
	if tv.Value[0] == 1 {
		return "true"
	}
	return "false"
}

// Bool extracts the unsigned bool value
func (tv *TypedBool) Bool() bool {
	return tv.Value[0] == 1
}

////////////////////////////////////////////////////////////////////////////////
// TypedDecimal64
////////////////////////////////////////////////////////////////////////////////

// TypedDecimal64 for a decimal64 value
type TypedDecimal64 TypedValue

// CreateTypedValueDecimal64 decodes a decimal value in to an object
func CreateTypedValueDecimal64(digits int64, precision uint32) *TypedValue {
	return (*TypedValue)(NewDecimal64(digits, precision))
}

// NewDecimal64 decodes a decimal value in to a Decimal type
func NewDecimal64(digits int64, precision uint32) *TypedDecimal64 {
	buf := make([]byte, 8)
	binary.LittleEndian.PutUint64(buf, uint64(digits))
	typeOpts := []int{int(precision)}
	typedDecimal64 := TypedDecimal64{
		Value:    buf,
		Type:     ValueTypeDECIMAL,
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
		return int64(binary.LittleEndian.Uint64(tv.Value)), uint32(precision)
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

// CreateTypedValueFloat decodes a decimal value in to an object
func CreateTypedValueFloat(value float32) *TypedValue {
	return (*TypedValue)(NewFloat(value))
}

// NewFloat decodes a decimal value in to a Bool type
func NewFloat(value float32) *TypedFloat {
	buf := make([]byte, 8)
	binary.LittleEndian.PutUint64(buf, math.Float64bits(float64(value)))
	typedFloat := TypedFloat{
		Value: buf,
		Type:  ValueTypeFLOAT,
	}
	return &typedFloat
}

// ValueType gives the value type
func (tv *TypedFloat) ValueType() ValueType {
	return tv.Type
}

func (tv *TypedFloat) String() string {
	return fmt.Sprintf("%f", math.Float64frombits(binary.LittleEndian.Uint64(tv.Value)))
}

// Float32 extracts the float value
func (tv *TypedFloat) Float32() float32 {
	if len(tv.Value) == 0 {
		return 0.0
	}
	return float32(math.Float64frombits(binary.LittleEndian.Uint64(tv.Value)))
}

////////////////////////////////////////////////////////////////////////////////
// TypedBytes
////////////////////////////////////////////////////////////////////////////////

// TypedBytes for a float value
type TypedBytes TypedValue

// CreateTypedValueBytes decodes an array of bytes in to an object
func CreateTypedValueBytes(value []byte) *TypedValue {
	return (*TypedValue)(NewBytes(value))
}

// NewBytes decodes an array of bytes in to a Bytes type
func NewBytes(value []byte) *TypedBytes {
	typedFloat := TypedBytes{
		Value:    value,
		Type:     ValueTypeBYTES,
		TypeOpts: []int{len(value)},
	}
	return &typedFloat
}

// ValueType gives the value type
func (tv *TypedBytes) ValueType() ValueType {
	return tv.Type
}

func (tv *TypedBytes) String() string {
	return base64.StdEncoding.EncodeToString(tv.Value)
}

// Bytes extracts the bytes value
func (tv *TypedBytes) Bytes() []byte {
	return tv.Value
}

////////////////////////////////////////////////////////////////////////////////
// TypedLeafListString
////////////////////////////////////////////////////////////////////////////////

// TypedLeafListString for a string leaf list
type TypedLeafListString TypedValue

// CreateLeafListString decodes string values in to an object
func CreateLeafListString(values []string) *TypedValue {
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
		Value: bytes,
		Type:  ValueTypeLeafListSTRING,
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
	for _, b := range tv.Value {
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

// CreateLeafListInt64 decodes int values in to an object
func CreateLeafListInt64(values []int) *TypedValue {
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
		Value: bytes,
		Type:  ValueTypeLeafListINT,
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
	count := len(tv.Value) / 8
	intList := make([]int, 0)

	for i := 0; i < count; i++ {
		v := tv.Value[i*8 : i*8+8]
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

// CreateLeafListUint64 decodes uint values in to a Leaf list
func CreateLeafListUint64(values []uint) *TypedValue {
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
		Value: bytes,
		Type:  ValueTypeLeafListUINT,
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
	count := len(tv.Value) / 8
	uintList := make([]uint, 0)

	for i := 0; i < count; i++ {
		v := tv.Value[i*8 : i*8+8]
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

// CreateLeafListBool decodes bool values in to an object
func CreateLeafListBool(values []bool) *TypedValue {
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
		Value: bytes,
		Type:  ValueTypeLeafListBOOL,
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
	count := len(tv.Value)
	bools := make([]bool, count)
	for i, v := range tv.Value {
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

// CreateLeafListDecimal64 decodes decimal values in to a Leaf list
func CreateLeafListDecimal64(digits []int64, precision uint32) *TypedValue {
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
		Value:    bytes,
		Type:     ValueTypeLeafListDECIMAL,
		TypeOpts: []int{int(precision)},
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
	count := len(tv.Value) / 8
	digitsList := make([]int64, 0)
	precision := 0

	if len(tv.TypeOpts) > 0 {
		precision = tv.TypeOpts[0]
	}

	for i := 0; i < count; i++ {
		v := tv.Value[i*8 : i*8+8]
		leafDigit := binary.LittleEndian.Uint64(v)
		digitsList = append(digitsList, int64(leafDigit))
	}

	return digitsList, uint32(precision)
}

// ListFloat extracts the leaf list values as floats
func (tv *TypedLeafListDecimal) ListFloat() []float32 {
	count := len(tv.Value) / 8
	precision := 0
	floatList := make([]float32, 0)

	if len(tv.TypeOpts) > 0 {
		precision = tv.TypeOpts[0]
	}

	for i := 0; i < count; i++ {
		v := tv.Value[i*8 : i*8+8]
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

// CreateLeafListFloat32 decodes float values in to a Leaf list
func CreateLeafListFloat32(values []float32) *TypedValue {
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
		Value: bytes,
		Type:  ValueTypeLeafListFLOAT,
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
	count := len(tv.Value) / 8
	float32s := make([]float32, 0)

	for i := 0; i < count; i++ {
		v := tv.Value[i*8 : i*8+8]
		float32s = append(float32s, float32(math.Float64frombits(binary.LittleEndian.Uint64(v))))
	}

	return float32s
}

////////////////////////////////////////////////////////////////////////////////
// TypedLeafListBytes
////////////////////////////////////////////////////////////////////////////////

// TypedLeafListBytes for an bool leaf list
type TypedLeafListBytes TypedValue

// CreateLeafListBytes decodes byte values in to a Leaf list
func CreateLeafListBytes(values [][]byte) *TypedValue {
	return (*TypedValue)(NewLeafListBytes(values))
}

// NewLeafListBytes decodes byte values in to a Leaf list type
func NewLeafListBytes(values [][]byte) *TypedLeafListBytes {
	bytes := make([]byte, 0)
	typeopts := make([]int, 0)
	for _, v := range values {
		bytes = append(bytes, []byte(v)...)
		typeopts = append(typeopts, len(v))
	}
	typedLeafListBytes := TypedLeafListBytes{
		Value:    bytes,
		Type:     ValueTypeLeafListBYTES, // Contains the lengths of each byte array in list
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
	for i, b := range tv.Value {
		valueLen := tv.TypeOpts[idx]
		if i-startAt == valueLen {
			bytes = append(bytes, buf)
			buf = make([]byte, 0)
			idx = idx + 1
			startAt = startAt + valueLen
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
