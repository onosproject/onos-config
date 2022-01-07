// Copyright 2021-present Open Networking Foundation.
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

package path

import (
	"fmt"
	"regexp"
	"sort"
	"strings"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/onosproject/onos-lib-go/pkg/errors"

	configapi "github.com/onosproject/onos-api/go/onos/config/v2"
	"github.com/onosproject/onos-lib-go/pkg/logging"
	"github.com/openconfig/goyang/pkg/yang"
)

var log = logging.GetLogger("utils", "path")

// MatchOnIndex - regexp to find indices in paths names
const MatchOnIndex = `(\[.*?]).*?`

// validPathRegexp - permissible values in paths
const validPathRegexp = `(/[a-zA-Z0-9:=\-\._[\]]+)+`

// IndexAllowedChars - regexp to restrict characters in index names
const IndexAllowedChars = `^([a-zA-Z0-9\*\-\._])+$`

// ReadOnlyAttrib is the known metadata about a Read Only leaf
type ReadOnlyAttrib struct {
	ValueType   configapi.ValueType
	TypeOpts    []uint8
	Description string
	Units       string
	Enum        map[int]string
	IsAKey      bool
	AttrName    string
}

// ReadOnlySubPathMap abstracts the read only subpath
type ReadOnlySubPathMap map[string]ReadOnlyAttrib

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

// ReadWritePathElem holds data about a leaf or container
type ReadWritePathElem struct {
	ReadOnlyAttrib
	Mandatory bool
	Default   string
	Range     []string
	Length    []string
}

// ReadWritePathMap is a map of ReadWrite paths their metadata
type ReadWritePathMap map[string]ReadWritePathElem

// JustPaths extracts keys from a read write path map
// expandSubPaths is not relevant for RW paths
func (rw ReadWritePathMap) JustPaths() []string {
	keys := make([]string, len(rw))
	i := 0
	for k := range rw {
		keys[i] = k
		i++
	}
	return keys
}

// TypeForPath finds the type from the model for a particular path
func (rw ReadWritePathMap) TypeForPath(path string) (configapi.ValueType, error) {
	for k, elem := range rw {
		if k == path {
			return elem.ValueType, nil
		}
	}
	return configapi.ValueType_EMPTY, fmt.Errorf("path %s not found in RW paths of model", path)
}

// ExtractPaths is a recursive function to extract a list of read only paths from a YGOT schema
func ExtractPaths(deviceEntry *yang.Entry, parentState yang.TriState, parentPath string,
	subpathPrefix string) (ReadOnlyPathMap, ReadWritePathMap) {
	readOnlyPaths := make(ReadOnlyPathMap)
	readWritePaths := make(ReadWritePathMap)
	for _, dirEntry := range deviceEntry.Dir {
		itemPath := formatName(dirEntry, false, parentPath, subpathPrefix)
		if dirEntry.IsLeaf() || dirEntry.IsLeafList() {
			// No need to recurse
			t, typeOpts, err := toValueType(dirEntry.Type, dirEntry.IsLeafList())
			tObj := ReadOnlyAttrib{
				ValueType:   t,
				TypeOpts:    typeOpts,
				Description: dirEntry.Description,
				Units:       dirEntry.Units,
				AttrName:    dirEntry.Name,
			}
			if err != nil {
				log.Errorf(err.Error())
			}
			var enum map[int]string
			if dirEntry.Type.Kind == yang.Yidentityref {
				enum = handleIdentity(dirEntry.Type)
			}
			tObj.Enum = enum
			// Check to see if this attribute is a key in a list
			if dirEntry.Parent.IsList() {
				keyNames := strings.Split(dirEntry.Parent.Key, " ")
				itemPathParts := strings.Split(itemPath, "/")
				attrName := itemPathParts[len(itemPathParts)-1]
				for _, k := range keyNames {
					if strings.EqualFold(attrName, k) {
						tObj.IsAKey = true
						break
					}
				}
			}
			if parentState == yang.TSFalse {
				leafMap, ok := readOnlyPaths[parentPath]
				if !ok {
					leafMap = make(ReadOnlySubPathMap)
					readOnlyPaths[parentPath] = leafMap
				}
				leafMap[strings.Replace(itemPath, parentPath, "", 1)] = tObj
			} else if dirEntry.Config == yang.TSFalse {
				leafMap := make(ReadOnlySubPathMap)
				leafMap["/"] = tObj
				readOnlyPaths[itemPath] = leafMap
			} else {
				ranges := make([]string, 0)
				for _, r := range dirEntry.Type.Range {
					ranges = append(ranges, fmt.Sprintf("%v", r))
				}
				lengths := make([]string, 0)
				for _, l := range dirEntry.Type.Length {
					lengths = append(lengths, fmt.Sprintf("%v", l))
				}
				rwElem := ReadWritePathElem{
					ReadOnlyAttrib: tObj,
					Mandatory:      dirEntry.Mandatory == yang.TSTrue,
					Default:        dirEntry.Default,
					Range:          ranges,
					Length:         lengths,
				}
				readWritePaths[itemPath] = rwElem
			}
		} else if dirEntry.IsContainer() {
			if dirEntry.Config == yang.TSFalse || parentState == yang.TSFalse {
				subpathPfx := subpathPrefix
				if parentState == yang.TSFalse {
					subpathPfx = itemPath[len(parentPath):]
				}
				subPaths, _ := ExtractPaths(dirEntry, yang.TSFalse, itemPath, subpathPfx)
				subPathsMap := make(ReadOnlySubPathMap)
				for _, v := range subPaths {
					for k, u := range v {
						subPathsMap[k] = u
					}
				}
				readOnlyPaths[itemPath] = subPathsMap
				continue
			}
			readOnlyPathsTemp, readWritePathTemp := ExtractPaths(dirEntry, dirEntry.Config, itemPath, "")
			for k, v := range readOnlyPathsTemp {
				readOnlyPaths[k] = v
			}
			for k, v := range readWritePathTemp {
				readWritePaths[k] = v
			}
		} else if dirEntry.IsList() {
			itemPath = formatName(dirEntry, true, parentPath, subpathPrefix)
			if dirEntry.Config == yang.TSFalse || parentState == yang.TSFalse {
				subpathPfx := subpathPrefix
				if parentState == yang.TSFalse {
					subpathPfx = itemPath[len(parentPath):]
				}
				subPaths, _ := ExtractPaths(dirEntry, yang.TSFalse, parentPath, subpathPfx)
				subPathsMap := make(ReadOnlySubPathMap)
				for _, v := range subPaths {
					for k, u := range v {
						subPathsMap[k] = u
					}
				}
				readOnlyPaths[itemPath] = subPathsMap
				continue
			}
			readOnlyPathsTemp, readWritePathsTemp := ExtractPaths(dirEntry, dirEntry.Config, itemPath, "")
			for k, v := range readOnlyPathsTemp {
				readOnlyPaths[k] = v
			}
			for k, v := range readWritePathsTemp {
				readWritePaths[k] = v
				// Need to copy the index of the list across to the RO list too
				roIdxName := k[:strings.LastIndex(k, "/")]
				roIdxSubPath := k[strings.LastIndex(k, "/"):]
				indices, _ := ExtractIndexNames(itemPath[strings.LastIndex(itemPath, "/"):])
				isIdxAttr := false
				for _, idx := range indices {
					if roIdxSubPath == fmt.Sprintf("/%s", idx) {
						isIdxAttr = true
					}
				}
				if roIdxName == itemPath && isIdxAttr {
					roIdx := ReadOnlyAttrib{
						ValueType:   v.ValueType,
						Description: v.Description,
						Units:       v.Units,
					}
					readOnlyPaths[roIdxName] = make(map[string]ReadOnlyAttrib)
					readOnlyPaths[roIdxName][roIdxSubPath] = roIdx
				}
			}

		} else if dirEntry.IsChoice() || dirEntry.IsCase() {
			// Recurse down through Choice and Case
			readOnlyPathsTemp, readWritePathsTemp := ExtractPaths(dirEntry, dirEntry.Config, parentPath, "")
			for k, v := range readOnlyPathsTemp {
				readOnlyPaths[k] = v
			}
			for k, v := range readWritePathsTemp {
				readWritePaths[k] = v
			}
		} else {
			log.Warnf("Unexpected type of leaf for %s %v", itemPath, dirEntry)
		}
	}
	return readOnlyPaths, readWritePaths
}

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

// AddMissingIndexName - a delete might just include the index at the end and not the index attribute
// e.g. /cont1a/list5[key1=abc][key2=123] - this means that we want to delete it and all of its children
// To match the model path though this has to include the key index name e.g.
// /cont1a/list5[key1=abc][key2=123]/key1 OR /cont1a/list5[key1=abc][key2=123]/key2
func AddMissingIndexName(path string) []string {
	extendedPaths := make([]string, 0)
	if strings.HasSuffix(path, "]") {
		lastElemIdx := strings.LastIndex(path, "/")
		indexNames, _ := ExtractIndexNames(path[lastElemIdx:])
		for _, idxName := range indexNames {
			extendedPaths = append(extendedPaths, fmt.Sprintf("%s/%s", path, idxName))
		}
	}
	return extendedPaths
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

func formatName(dirEntry *yang.Entry, isList bool, parentPath string, subpathPrefix string) string {
	parentAndSubPath := parentPath
	if subpathPrefix != "/" {
		parentAndSubPath = fmt.Sprintf("%s%s", parentPath, subpathPrefix)
	}

	var name string
	if isList {
		//have to ensure index order is consistent where there's more than one
		keyParts := strings.Split(dirEntry.Key, " ")
		sort.Slice(keyParts, func(i, j int) bool {
			return keyParts[i] < keyParts[j]
		})
		name = fmt.Sprintf("%s/%s", parentAndSubPath, dirEntry.Name)
		for _, k := range keyParts {
			name += fmt.Sprintf("[%s=*]", k)
		}
	} else {
		name = fmt.Sprintf("%s/%s", parentAndSubPath, dirEntry.Name)
	}

	return name
}

//Paths extract the read only path up to the first read only container
func Paths(readOnly ReadOnlyPathMap) []string {
	keys := make([]string, 0, len(readOnly))
	for k := range readOnly {
		keys = append(keys, k)
	}
	return keys
}

//PathsRW extract the read write path
func PathsRW(rwPathMap ReadWritePathMap) []string {
	keys := make([]string, 0, len(rwPathMap))
	for k := range rwPathMap {
		keys = append(keys, k)
	}
	return keys
}

func toValueType(entry *yang.YangType, isLeafList bool) (configapi.ValueType, []uint8, error) {
	switch entry.Kind.String() {
	case "int8", "int16", "int32", "int64":
		width := extractIntegerWidth(entry.Kind.String())
		if isLeafList {
			return configapi.ValueType_LEAFLIST_INT, []uint8{uint8(width)}, nil
		}
		return configapi.ValueType_INT, []uint8{uint8(width)}, nil
	case "uint8", "uint16", "uint32", "uint64":
		width := extractIntegerWidth(entry.Kind.String())
		if isLeafList {
			return configapi.ValueType_LEAFLIST_UINT, []uint8{uint8(width)}, nil
		}
		return configapi.ValueType_UINT, []uint8{uint8(width)}, nil
	case "decimal64":
		if isLeafList {
			return configapi.ValueType_LEAFLIST_DECIMAL, []uint8{uint8(entry.FractionDigits)}, nil
		}
		return configapi.ValueType_DECIMAL, []uint8{uint8(entry.FractionDigits)}, nil
	case "string", "enumeration", "leafref", "identityref", "union", "instance-identifier":
		if isLeafList {
			return configapi.ValueType_LEAFLIST_STRING, nil, nil
		}
		return configapi.ValueType_STRING, nil, nil
	case "boolean":
		if isLeafList {
			return configapi.ValueType_LEAFLIST_BOOL, nil, nil
		}
		return configapi.ValueType_BOOL, nil, nil
	case "bits", "binary":
		if isLeafList {
			return configapi.ValueType_LEAFLIST_BYTES, nil, nil
		}
		return configapi.ValueType_BYTES, nil, nil
	case "empty":
		return configapi.ValueType_EMPTY, nil, nil
	default:
		return configapi.ValueType_EMPTY, nil,
			errors.NewInvalid("unhandled type in ModelPlugin %s %s %s",
				entry.Name, entry.Kind.String(), entry.Type)
	}
}

func extractIntegerWidth(typeName string) configapi.Width {
	switch typeName {
	case "int8", "uint8":
		return configapi.WidthEight
	case "int16", "uint16":
		return configapi.WidthSixteen
	case "int32", "uint32":
		return configapi.WidthThirtyTwo
	case "int64", "uint64", "counter64":
		return configapi.WidthSixtyFour
	default:
		return configapi.WidthThirtyTwo
	}
}

func handleIdentity(yangType *yang.YangType) map[int]string {
	identityMap := make(map[int]string)
	identityMap[0] = "UNSET"
	for i, val := range yangType.IdentityBase.Values {
		identityMap[i+1] = val.Name
	}
	return identityMap
}

func FindPathFromModel(path string, rwPaths ReadWritePathMap, exact bool) (bool, *ReadWritePathElem, error) {
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
func CheckKeyValue(path string, rwPath *ReadWritePathElem, val *configapi.TypedValue) error {
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
