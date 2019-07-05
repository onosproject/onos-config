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

// Types given here are a rough approximation of those in the set of YANG types
// and the set of gNMI types
const (
	TypeBool      = "bool"
	TypeBytes     = "bytes"
	TypeDecimal64 = "decimal64"
	TypeEmpty     = "empty" // TODO check if this is really necessary
	TypeFloat     = "float"
	TypeInt       = "int"
	TypeString    = "string"
	TypeUint      = "uint"
)

// TypedValue is a of a value, a type and a LeafList flag
type TypedValue struct {
	Value    []byte
	Type     string
	LeafList bool
}

////////////////////////////////////////////////////////////////////////////////
// TypedEmpty
////////////////////////////////////////////////////////////////////////////////

// TypedEmpty for an empty value
type TypedEmpty TypedValue

// CreateTypedValueEmpty decodes an empty object
func CreateTypedValueEmpty() *TypedEmpty {
	typedEmpty := TypedEmpty{
		Value:    make([]byte, 0),
		Type:     TypeEmpty,
		LeafList: false,
	}
	return &typedEmpty
}

////////////////////////////////////////////////////////////////////////////////
// TypedString
////////////////////////////////////////////////////////////////////////////////

// TypedString for a string value
type TypedString TypedValue

// CreateTypedValueString decodes string value in to an object
func CreateTypedValueString(value string) *TypedString {
	typedString := TypedString{
		Value:    []byte(value),
		Type:     TypeString,
		LeafList: false,
	}
	return &typedString
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
func CreateTypedValueInt64(value int) *TypedInt64 {
	buf := make([]byte, 8)
	binary.LittleEndian.PutUint64(buf, uint64(value))

	typedInt64 := TypedInt64{
		Value:    buf,
		Type:     TypeInt,
		LeafList: false,
	}
	return &typedInt64
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
func CreateTypedValueUint64(value uint) *TypedUint64 {
	buf := make([]byte, 8)
	binary.LittleEndian.PutUint64(buf, uint64(value))
	typedUint64 := TypedUint64{
		Value:    buf,
		Type:     TypeUint,
		LeafList: false,
	}
	return &typedUint64
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
func CreateTypedValueBool(value bool) *TypedBool {
	buf := make([]byte, 1)
	if value {
		buf[0] = 1
	}
	typedBool := TypedBool{
		Value:    buf,
		Type:     TypeBool,
		LeafList: false,
	}
	return &typedBool
}

func (tv *TypedBool) String() string {
	if tv.Value[0] == 1 {
		return "true"
	}
	return "false"
}

// Bool extracts the unsigned bool value
func (tv *TypedBool) Bool() bool {
	if tv.Value[0] == 1 {
		return true
	}
	return false
}

////////////////////////////////////////////////////////////////////////////////
// TypedDecimal64
////////////////////////////////////////////////////////////////////////////////

// TypedDecimal64 for a decimal64 value
type TypedDecimal64 TypedValue

// CreateTypedValueDecimal64 decodes a decimal value in to an object
func CreateTypedValueDecimal64(digits int64, precision uint32) *TypedDecimal64 {
	buf := make([]byte, 8)
	binary.LittleEndian.PutUint64(buf, uint64(digits))
	valueType := fmt.Sprintf("%s-%d", TypeDecimal64, precision)
	typedDecimal64 := TypedDecimal64{
		Value:    buf,
		Type:     valueType,
		LeafList: false,
	}
	return &typedDecimal64
}

func (tv *TypedDecimal64) String() string {
	params := strings.Split(tv.Type, "-")
	if len(params) > 1 {
		precision, _ := strconv.Atoi(params[1])
		return strDecimal64(int64(binary.LittleEndian.Uint64(tv.Value)), uint32(precision))
	}
	return ""
}

// Decimal64 extracts the unsigned decimal value
func (tv *TypedDecimal64) Decimal64() (int64, uint32) {
	params := strings.Split(tv.Type, "-")
	if len(params) > 1 {
		precision, _ := strconv.Atoi(params[1])
		return int64(binary.LittleEndian.Uint64(tv.Value)), uint32(precision)
	}
	return 0, 0
}

////////////////////////////////////////////////////////////////////////////////
// TypedFloat
////////////////////////////////////////////////////////////////////////////////

// TypedFloat for a float value
type TypedFloat TypedValue

// CreateTypedValueFloat decodes a decimal value in to an object
func CreateTypedValueFloat(value float32) *TypedFloat {
	buf := make([]byte, 8)
	binary.LittleEndian.PutUint64(buf, math.Float64bits(float64(value)))
	typedFloat := TypedFloat{
		Value:    buf,
		Type:     TypeFloat,
		LeafList: false,
	}
	return &typedFloat
}

func (tv *TypedFloat) String() string {
	return fmt.Sprintf("%f", math.Float64frombits(binary.LittleEndian.Uint64(tv.Value)))
}

// Float32 extracts the float value
func (tv *TypedFloat) Float32() float32 {
	return float32(math.Float64frombits(binary.LittleEndian.Uint64(tv.Value)))
}

////////////////////////////////////////////////////////////////////////////////
// TypedBytes
////////////////////////////////////////////////////////////////////////////////

// TypedBytes for a float value
type TypedBytes TypedValue

// CreateTypedValueBytes decodes an array of bytes in to an object
func CreateTypedValueBytes(value []byte) *TypedBytes {
	typedFloat := TypedBytes{
		Value:    value,
		Type:     TypeBytes,
		LeafList: false,
	}
	return &typedFloat
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

// CreateLeafListString decodes string values in to an Leaf list
func CreateLeafListString(values []string) *TypedLeafListString {
	first := true
	bytes := make([]byte, 0)
	for _, v := range values {
		if first {
			first = false
		} else {
			bytes = append(bytes, 0x1D) // Group separator
		}
		for _, s := range []byte(v) {
			bytes = append(bytes, s)
		}
	}
	typedLeafListString := TypedLeafListString{
		Value:    bytes,
		Type:     TypeString,
		LeafList: true,
	}
	return &typedLeafListString
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

// CreateLeafListInt64 decodes int values in to a Leaf list
func CreateLeafListInt64(values []int) *TypedLeafListInt64 {
	bytes := make([]byte, 0)
	for _, v := range values {
		buf := make([]byte, 8)
		binary.LittleEndian.PutUint64(buf, uint64(v))
		for _, b := range buf {
			bytes = append(bytes, b)
		}
	}
	typedLeafListInt64 := TypedLeafListInt64{
		Value:    bytes,
		Type:     TypeInt,
		LeafList: true,
	}
	return &typedLeafListInt64
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
func CreateLeafListUint64(values []uint) *TypedLeafListUint {
	bytes := make([]byte, 0)
	for _, v := range values {
		buf := make([]byte, 8)
		binary.LittleEndian.PutUint64(buf, uint64(v))
		for _, b := range buf {
			bytes = append(bytes, b)
		}
	}
	typedLeafListUint := TypedLeafListUint{
		Value:    bytes,
		Type:     TypeUint,
		LeafList: true,
	}
	return &typedLeafListUint
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

// CreateLeafListBool decodes bool values in to a Leaf list
func CreateLeafListBool(values []bool) *TypedLeafListBool {
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
		Value:    bytes,
		Type:     TypeBool,
		LeafList: true,
	}
	return &typedLeafListBool
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
func CreateLeafListDecimal64(digits []int64, precision uint32) *TypedLeafListDecimal {
	bytes := make([]byte, 0)
	for _, d := range digits {
		buf := make([]byte, 8)
		binary.LittleEndian.PutUint64(buf, uint64(d))
		for _, b := range buf {
			bytes = append(bytes, b)
		}
	}
	valueType := fmt.Sprintf("%s-%d", TypeDecimal64, precision)
	typedLeafListDecimal := TypedLeafListDecimal{
		Value:    bytes,
		Type:     valueType,
		LeafList: true,
	}
	return &typedLeafListDecimal
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

	params := strings.Split(tv.Type, "-")
	if len(params) > 1 {
		precision, _ = strconv.Atoi(params[1])
	}

	for i := 0; i < count; i++ {
		v := tv.Value[i*8 : i*8+8]
		leafDigit := binary.LittleEndian.Uint64(v)
		digitsList = append(digitsList, int64(leafDigit))
	}

	return digitsList, uint32(precision)
}

////////////////////////////////////////////////////////////////////////////////
// TypedLeafListFloat
////////////////////////////////////////////////////////////////////////////////

// TypedLeafListFloat for a decimal leaf list
type TypedLeafListFloat TypedValue

// CreateLeafListFloat32 decodes float values in to a Leaf list
func CreateLeafListFloat32(values []float32) *TypedLeafListFloat {
	bytes := make([]byte, 0)
	for _, f := range values {
		buf := make([]byte, 8)
		binary.LittleEndian.PutUint64(buf, math.Float64bits(float64(f)))
		for _, b := range buf {
			bytes = append(bytes, b)
		}
	}
	typedLeafListFloat := TypedLeafListFloat{
		Value:    bytes,
		Type:     TypeFloat,
		LeafList: true,
	}
	return &typedLeafListFloat
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
func CreateLeafListBytes(values [][]byte) *TypedLeafListBytes {
	bytes := make([]byte, 0)
	typestr := TypeBytes
	for _, v := range values {
		for _, b := range []byte(v) {
			bytes = append(bytes, b)
		}
		typestr = fmt.Sprintf("%s_%d", typestr, len(v))
	}
	typedLeafListBytes := TypedLeafListBytes{
		Value:    bytes,
		Type:     typestr, // Contains the lengths of each byte array in list
		LeafList: true,
	}
	return &typedLeafListBytes
}

func (tv *TypedLeafListBytes) String() string {
	return fmt.Sprintf("%v", tv.List())
}

// List extracts the leaf list values
func (tv *TypedLeafListBytes) List() [][]byte {
	bytes := make([][]byte, 0)
	buf := make([]byte, 0)
	lengths := strings.Split(tv.Type, "_")
	idx := 1
	startAt := 0
	for i, b := range tv.Value {
		valueLen, _ := strconv.Atoi(lengths[idx])
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
	return fmt.Sprintf("%d.%d", i, frac)
}
