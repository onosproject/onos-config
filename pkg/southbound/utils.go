// Copyright 2019-present Open Networking Foundation
//
// Licensed under the Apache License, Configuration 2.0 (the "License");
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

package southbound

import (
	"fmt"
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
		result = append(result, path[:i])
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
