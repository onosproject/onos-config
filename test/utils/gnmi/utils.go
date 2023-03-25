// SPDX-FileCopyrightText: 2020-present Open Networking Foundation <info@opennetworking.org>
//
// SPDX-License-Identifier: Apache-2.0
//

package gnmi

import (
	protoutils "github.com/onosproject/onos-config/test/utils/proto"
	"strings"
)

// GetTargetPath creates a target path
func GetTargetPath(target string, path string) []protoutils.GNMIPath {
	return GetTargetPathWithValue(target, path, "", "")
}

// GetTargetPathWithValue creates a target path with a value to set
func GetTargetPathWithValue(target string, path string, value string, valueType string) []protoutils.GNMIPath {
	targetPath := make([]protoutils.GNMIPath, 1)
	targetPath[0].TargetName = target
	targetPath[0].Path = path
	targetPath[0].PathDataValue = value
	targetPath[0].PathDataType = valueType
	return targetPath
}

// GetTargetPaths creates multiple target paths
func GetTargetPaths(targets []string, paths []string) []protoutils.GNMIPath {
	var targetPaths = make([]protoutils.GNMIPath, len(paths)*len(targets))
	pathIndex := 0
	for _, dev := range targets {
		for _, path := range paths {
			targetPaths[pathIndex].TargetName = dev
			targetPaths[pathIndex].Path = path
			pathIndex++
		}
	}
	return targetPaths
}

// GetTargetPathsWithValues creates multiple target paths with values to set
func GetTargetPathsWithValues(targets []string, paths []string, values []string) []protoutils.GNMIPath {
	var targetPaths = GetTargetPaths(targets, paths)
	valueIndex := 0
	for range targets {
		for _, value := range values {
			targetPaths[valueIndex].PathDataValue = value
			targetPaths[valueIndex].PathDataType = protoutils.StringVal
			valueIndex++
		}
	}
	return targetPaths
}

// MakeProtoPath returns a Path: element for a given target and Path
func MakeProtoPath(target string, path string) string {
	var protoBuilder strings.Builder

	if target != "" || path != "" {
		protoBuilder.WriteString("path: ")
		gnmiPath := protoutils.MakeProtoTarget(target, path)
		protoBuilder.WriteString(gnmiPath)
	}
	return protoBuilder.String()
}
