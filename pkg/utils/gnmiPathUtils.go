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

// Package utils implements various gNMI path manipulation facilities.
package utils

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"sort"
	"strconv"
	"strings"

	pb "github.com/openconfig/gnmi/proto/gnmi"
)

// ParseGNMIElements builds up a gnmi path, from user-supplied text
func ParseGNMIElements(elms []string) (*pb.Path, error) {
	var parsed []*pb.PathElem
	for _, e := range elms {
		n, keys, err := parseElement(e)
		if err != nil {
			return nil, err
		}
		parsed = append(parsed, &pb.PathElem{Name: n, Key: keys})
	}
	return &pb.Path{
		Elem: parsed,
	}, nil
}

// parseElement parses a path element, according to the gNMI specification. See
// https://github.com/openconfig/reference/blame/master/rpc/gnmi/gnmi-path-conventions.md
//
// It returns the first string (the current element name), and an optional map of key name
// value pairs.
func parseElement(pathElement string) (string, map[string]string, error) {

	// First check if there are any keys, i.e. do we have at least one '[' in the element
	name, keyStart := findUnescaped(pathElement, '[')
	if keyStart < 0 {
		return name, nil, nil
	}

	// Error if there is no element name or if the "[" is at the beginning of the path element
	if len(name) == 0 {
		return "", nil, fmt.Errorf("failed to find element name in %q", pathElement)
	}

	// Look at the keys now.
	keys := make(map[string]string)
	keyPart := pathElement[keyStart:]
	for keyPart != "" {
		k, v, nextKey, err := parseKey(keyPart)
		if err != nil {
			return "", nil, err
		}
		keys[k] = v
		keyPart = nextKey
	}
	return name, keys, nil
}

// parseKey returns the key name, key value and the remaining string to be parsed,
func parseKey(s string) (string, string, string, error) {
	if s[0] != '[' {
		return "", "", "", fmt.Errorf("failed to find opening '[' in %q", s)
	}
	k, iEq := findUnescaped(s[1:], '=')
	if iEq < 0 {
		return "", "", "", fmt.Errorf("failed to find '=' in %q", s)
	}
	if k == "" {
		return "", "", "", fmt.Errorf("failed to find key name in %q", s)
	}

	rhs := s[1+iEq+1:]
	v, iClosBr := findUnescaped(rhs, ']')
	if iClosBr < 0 {
		return "", "", "", fmt.Errorf("failed to find ']' in %q", s)
	}
	if v == "" {
		return "", "", "", fmt.Errorf("failed to find key value in %q", s)
	}

	next := rhs[iClosBr+1:]
	return k, v, next, nil
}

// findUnescaped will return the index of the first unescaped match of 'find', and the unescaped
// string leading up to it.
func findUnescaped(s string, find byte) (string, int) {
	// Take a fast track if there are no escape sequences
	if strings.IndexByte(s, '\\') == -1 {
		i := strings.IndexByte(s, find)
		if i < 0 {
			return s, -1
		}
		return s[:i], i
	}

	// Find the first match, taking care of escaped chars.
	var b strings.Builder
	var i int
	len := len(s)
	for i = 0; i < len; {
		ch := s[i]
		if ch == find {
			return b.String(), i
		} else if ch == '\\' && i < len-1 {
			i++
			ch = s[i]
		}
		b.WriteByte(ch)
		i++
	}
	return b.String(), -1
}

// SplitPaths splits multiple gnmi paths
func SplitPaths(paths []string) [][]string {
	out := make([][]string, len(paths))
	for i, path := range paths {
		out[i] = SplitPath(path)
	}
	return out
}

// SplitPath splits a gnmi path according to the spec. See
// https://github.com/openconfig/reference/blob/master/rpc/gnmi/gnmi-path-conventions.md
// No validation is done. Behavior is undefined if path is an invalid
// gnmi path. TODO: Do validation?
func SplitPath(path string) []string {
	var result []string
	if len(path) > 0 && path[0] == '/' {
		path = path[1:]
	}
	for len(path) > 0 {
		i := nextTokenIndex(path)
		part := path[:i]
		partsNs := strings.Split(part, ":")
		if len(partsNs) == 2 {
			// We have to discard the namespace as gNMI doesn't handle it
			part = partsNs[1]
		}
		result = append(result, part)
		path = path[i:]
		if len(path) > 0 && path[0] == '/' {
			path = path[1:]
		}
	}
	return result
}

// nextTokenIndex returns the end index of the first token.
func nextTokenIndex(path string) int {
	var inBrackets bool
	var escape bool
	for i, c := range path {
		switch c {
		case '[':
			inBrackets = true
			escape = false
		case ']':
			if !escape {
				inBrackets = false
			}
			escape = false
		case '\\':
			escape = !escape
		case '/':
			if !inBrackets && !escape {
				return i
			}
			escape = false
		default:
			escape = false
		}
	}
	return len(path)
}

func writeSafeString(b *strings.Builder, s string, esc rune) {
	for _, c := range s {
		if c == esc || c == '\\' {
			b.WriteRune('\\')
		}
		b.WriteRune(c)
	}
}

// StrPath builds a human-readable form of a gnmi path.
// e.g. /a/b/c[e=f]
func StrPath(path *pb.Path) string {
	if path == nil {
		return "/"
	} else if len(path.Elem) != 0 {
		return strPathV04(path)
	} else if len(path.Element) != 0 {
		return strPathV03(path)
	}
	return "/"
}

// StrPathElem builds a human-readable form of a list of path elements.
// e.g. /a/b/c[e=f]
func StrPathElem(pathElem []*pb.PathElem) string {
	b := &strings.Builder{}
	for _, elm := range pathElem {
		b.WriteRune('/')
		writeSafeString(b, elm.Name, '/')
		if len(elm.Key) > 0 {
			// Sort the keys so that they print in a conistent
			// order. We don't have the YANG AST information, so the
			// best we can do is sort them alphabetically.
			keys := make([]string, 0, len(elm.Key))
			for k := range elm.Key {
				keys = append(keys, k)
			}
			sort.Strings(keys)
			for _, k := range keys {
				b.WriteRune('[')
				b.WriteString(k)
				b.WriteRune('=')
				writeSafeString(b, elm.Key[k], ']')
				b.WriteRune(']')
			}
		}
	}
	return b.String()
}

// strPathV04 handles the v0.4 gnmi and later path.Elem member.
func strPathV04(path *pb.Path) string {
	pathElem := path.Elem
	return StrPathElem(pathElem)
}

// strPathV03 handles the v0.3 gnmi and earlier path.Element member.
func strPathV03(path *pb.Path) string {
	return "/" + strings.Join(path.Element, "/")
}

// StrVal will return a string representing the supplied value
func StrVal(val *pb.TypedValue) string {
	switch v := val.GetValue().(type) {
	case *pb.TypedValue_StringVal:
		return v.StringVal
	case *pb.TypedValue_JsonIetfVal:
		return strJSON(v.JsonIetfVal)
	case *pb.TypedValue_JsonVal:
		return strJSON(v.JsonVal)
	case *pb.TypedValue_IntVal:
		return strconv.FormatInt(v.IntVal, 10)
	case *pb.TypedValue_UintVal:
		return strconv.FormatUint(v.UintVal, 10)
	case *pb.TypedValue_BoolVal:
		return strconv.FormatBool(v.BoolVal)
	case *pb.TypedValue_BytesVal:
		return base64.StdEncoding.EncodeToString(v.BytesVal)
	case *pb.TypedValue_DecimalVal:
		return strDecimal64(v.DecimalVal)
	case *pb.TypedValue_FloatVal:
		return strconv.FormatFloat(float64(v.FloatVal), 'g', -1, 32)
	case *pb.TypedValue_LeaflistVal:
		return strLeaflist(v.LeaflistVal)
	case *pb.TypedValue_AsciiVal:
		return v.AsciiVal
	case *pb.TypedValue_AnyVal:
		return v.AnyVal.String()
	case *pb.TypedValue_ProtoBytes:
		return base64.StdEncoding.EncodeToString(v.ProtoBytes)
	default:
		panic(v)
	}
}

func strJSON(inJSON []byte) string {
	var out bytes.Buffer
	err := json.Indent(&out, inJSON, "", "  ")
	if err != nil {
		return fmt.Sprintf("(error unmarshalling json: %s)\n", err) + string(inJSON)
	}
	return out.String()
}

func strDecimal64(d *pb.Decimal64) string {
	var i, frac int64
	if d.Precision > 0 {
		div := int64(10)
		it := d.Precision - 1
		for it > 0 {
			div *= 10
			it--
		}
		i = d.Digits / div
		frac = d.Digits % div
	} else {
		i = d.Digits
	}
	if frac < 0 {
		frac = -frac
	}
	return fmt.Sprintf("%d.%d", i, frac)
}

// strLeafList builds a human-readable form of a leaf-list. e.g. [1, 2, 3] or [a, b, c]
func strLeaflist(v *pb.ScalarArray) string {
	var b strings.Builder
	b.WriteByte('[')

	for i, elm := range v.Element {
		b.WriteString(StrVal(elm))
		if i < len(v.Element)-1 {
			b.WriteString(", ")
		}
	}

	b.WriteByte(']')
	return b.String()
}
