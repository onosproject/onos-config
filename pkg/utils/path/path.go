// SPDX-FileCopyrightText: 2020-present Open Networking Foundation <info@opennetworking.org>
//
// SPDX-License-Identifier: Apache-2.0

package path

import (
	"fmt"
	"regexp"
	"strings"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/onosproject/onos-lib-go/pkg/errors"

	"github.com/onosproject/onos-api/go/onos/config/admin"
	configapi "github.com/onosproject/onos-api/go/onos/config/v2"
	"github.com/onosproject/onos-lib-go/pkg/logging"
)

var log = logging.GetLogger("utils", "path")

// MatchOnIndex - regexp to find indices in paths names
const MatchOnIndex = `(\[.*?]).*?`

// validPathRegexp - permissible values in paths
const validPathRegexp = `(/[a-zA-Z0-9:=\-\._[\]]+)+`

// IndexAllowedChars - regexp to restrict characters in index names
const IndexAllowedChars = `^([a-zA-Z0-9\*\-\._])+$`

// ReadOnlySubPathMap abstracts the read only subpath
type ReadOnlySubPathMap map[string]admin.ReadOnlySubPath

// ReadOnlyPathMap abstracts the read only path
type ReadOnlyPathMap map[string]ReadOnlySubPathMap

var rOnIndex = regexp.MustCompile(MatchOnIndex)
var rIndexAllowedChars = regexp.MustCompile(IndexAllowedChars)

// JustPaths extracts keys from a read only path map
func (ro ReadOnlyPathMap) JustPaths() []string {
	keys := make([]string, 0)
	for k, subPaths := range ro {
		for k1 := range subPaths {
			if k1 == "/" {
				keys = append(keys, k)
			} else {
				keys = append(keys, k+k1)
			}
		}
	}
	return keys
}

// TypeForPath finds the type from the model for a particular path
func (ro ReadOnlyPathMap) TypeForPath(path string) (configapi.ValueType, error) {
	for k, subPaths := range ro {
		for k1, sp := range subPaths {
			if k1 == "/" {
				if k == path {
					return sp.ValueType, nil
				}
			} else {
				if k+k1 == path {
					return sp.ValueType, nil
				}
			}
		}
	}
	return configapi.ValueType_EMPTY, fmt.Errorf("path %s not found in RO paths of model", path)
}

// ReadWritePathMap is a map of ReadWrite paths their metadata
type ReadWritePathMap map[string]admin.ReadWritePath

// NamespaceMap is a map of namespace prefixes to full names
type NamespaceMap map[string]string

// RemovePathIndices removes the index value from a path to allow it to be compared to a model path
func RemovePathIndices(path string) string {
	indices := rOnIndex.FindAllStringSubmatch(path, -1)
	for _, i := range indices {
		path = strings.Replace(path, i[0], "", 1)
	}
	return path
}

// AnonymizePathIndices anonymizes index value in a path (replaces it with *)
func AnonymizePathIndices(path string) string {
	indices := rOnIndex.FindAllStringSubmatch(path, -1)
	for _, i := range indices {
		idxParts := strings.Split(i[0], "=")
		idxParts[len(idxParts)-1] = "*]"
		path = strings.Replace(path, i[0], strings.Join(idxParts, "="), 1)
	}
	return path
}

// CheckPathIndexIsValid - check that index values have only the specified chars
func CheckPathIndexIsValid(index string) error {
	if !rIndexAllowedChars.MatchString(index) {
		return errors.NewInvalid("index value '%s' does not match pattern '%s'", index, IndexAllowedChars)
	}
	return nil
}

// ExtractIndexNames - get an ordered array of index names and index values
func ExtractIndexNames(path string) ([]string, []string) {
	indexNames := make([]string, 0)
	indexValues := make([]string, 0)
	jsonMatches := rOnIndex.FindAllStringSubmatch(path, -1)
	for _, m := range jsonMatches {
		idxName := m[1][1:strings.LastIndex(m[1], "=")]
		indexNames = append(indexNames, idxName)
		idxValue := m[1][strings.LastIndex(m[1], "=")+1 : len(m[1])-1]
		indexValues = append(indexValues, idxValue)
	}
	return indexNames, indexValues
}

// FindPathFromModel ...
func FindPathFromModel(path string, rwPaths ReadWritePathMap, exact bool) (bool, *admin.ReadWritePath, error) {
	searchPathNoIndices := RemovePathIndices(path)

	// try exact match first
	if rwPath, isExactMatch := rwPaths[AnonymizePathIndices(path)]; isExactMatch {
		return true, &rwPath, nil
	} else if exact {
		return false, nil,
			status.Errorf(codes.InvalidArgument, "unable to find exact match for RW model path %s. %d paths inspected",
				path, len(rwPaths))
	}

	if strings.HasSuffix(path, "]") { //Ends with index
		indices, _ := ExtractIndexNames(path)
		// Add on the last index
		searchPathNoIndices = fmt.Sprintf("%s/%s", searchPathNoIndices, indices[len(indices)-1])
	}

	// First search through the RW paths
	for modelPath, modelElem := range rwPaths {
		pathNoIndices := RemovePathIndices(modelPath)
		// Find a short path
		if exact && pathNoIndices == searchPathNoIndices {
			return false, &modelElem, nil
		} else if !exact && strings.HasPrefix(pathNoIndices, searchPathNoIndices) {
			return false, &modelElem, nil // returns the first thing it finds that matches the prefix
		}
	}

	return false, nil,
		errors.NewInvalid("unable to find RW model path %s ( without index %s). %d paths inspected", path, searchPathNoIndices, len(rwPaths))
}

// CheckKeyValue checks that if this is a Key attribute, that the value is the same as its parent's key
func CheckKeyValue(path string, rwPath *admin.ReadWritePath, val *configapi.TypedValue) error {
	indexNames, indexValues := ExtractIndexNames(path)
	if len(indexNames) == 0 {
		return nil
	}
	for i, idxName := range indexNames {
		if err := CheckPathIndexIsValid(indexValues[i]); err != nil {
			return err
		}
		if !rwPath.IsAKey || rwPath.AttrName == idxName && indexValues[i] == val.ValueToString() {
			return nil
		}
	}
	return errors.NewInvalid("index attribute %s=%s does not match %s", rwPath.AttrName, val.ValueToString(), path)
}

// IsPathValid tests for valid paths. Path is valid if it
// 1) starts with a slash
// 2) is followed by at least one of alphanumeric or any of : = - _ [ ]
// 3) and any further combinations of 1+2
// Two contiguous slashes are not allowed
// Paths not starting with slash are not allowed
func IsPathValid(path string) error {
	r1 := regexp.MustCompile(validPathRegexp)

	match := r1.FindString(path)
	if path != match {
		return errors.NewInvalid("invalid path %s. Must match %s", path, validPathRegexp)
	}
	return nil
}

// GetParentPath returns the immediate parent path of the specified path; empty string if "/" is given
func GetParentPath(path string) string {
	i := strings.LastIndex(path, "/")
	if i <= 0 {
		return ""
	}
	return path[0:i]
}
