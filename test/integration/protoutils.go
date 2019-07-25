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

package integration

import (
	"github.com/onosproject/onos-config/pkg/utils"
	"strings"
)

const (
	// StripNamespaces will remove namespaces from path elements
	StripNamespaces = true

	// LeaveNamespaces doesn't change namespaces
	LeaveNamespaces = false

	// StringVal :
	StringVal = "string_val"

	// IntVal :
	IntVal = "int_val"

	// BoolVal :
	BoolVal = "bool_val"
)

// MakeProtoTarget returns a GNMI proto path for a given target
func MakeProtoTarget(target string, path string, stripNamespaces bool) string {
	var protoBuilder strings.Builder
	var pathElements []string

	protoBuilder.WriteString("<target: '")
	protoBuilder.WriteString(target)
	protoBuilder.WriteString("', ")

	//  This is a temporary hack. Some of the tests want namespaces stripped, so allowing this
	//  for now and will remove this when the stripping is removed everywhere.
	if !stripNamespaces {
		pathElements = strings.Split(path[1:], "/")
	} else {
		pathElements = utils.SplitPath(path)
	}

	for _, pathElement := range pathElements {
		protoBuilder.WriteString("elem: <name: '")
		protoBuilder.WriteString(pathElement)
		protoBuilder.WriteString("'>")
	}
	protoBuilder.WriteString(">")
	return protoBuilder.String()
}

// MakeProtoPath returns a path: element for a given target and path
func MakeProtoPath(target string, path string, stripNamespaces bool) string {
	var protoBuilder strings.Builder

	protoBuilder.WriteString("path: ")
	gnmiPath := MakeProtoTarget(target, path, stripNamespaces)
	protoBuilder.WriteString(gnmiPath)
	return protoBuilder.String()
}

func makeProtoValue(value string, valueType string) string {
	var protoBuilder strings.Builder

	var valueString string

	if valueType == StringVal {
		valueString = "'" + value + "'"
	} else {
		valueString = value
	}
	protoBuilder.WriteString(" val: <")
	protoBuilder.WriteString(valueType)
	protoBuilder.WriteString(":")
	protoBuilder.WriteString(valueString)
	protoBuilder.WriteString(">")
	return protoBuilder.String()
}

// MakeProtoUpdatePath returns an update: element for a target, path, and new value
func MakeProtoUpdatePath(devicePath DevicePath, stripNamespaces bool) string {
	var protoBuilder strings.Builder

	protoBuilder.WriteString("update: <")
	protoBuilder.WriteString(MakeProtoPath(devicePath.deviceName, devicePath.path, stripNamespaces))
	protoBuilder.WriteString(makeProtoValue(devicePath.pathDataValue, devicePath.pathDataType))
	protoBuilder.WriteString(">")
	return protoBuilder.String()
}

// MakeProtoDeletePath returns a delete: element for a given target and path
func MakeProtoDeletePath(target string, path string) string {
	var protoBuilder strings.Builder

	protoBuilder.WriteString("delete: ")
	protoBuilder.WriteString(MakeProtoTarget(target, path, true))
	return protoBuilder.String()
}
