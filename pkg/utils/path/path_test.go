/*
 * SPDX-FileCopyrightText: 2022-present Intel Corporation
 *
 * SPDX-License-Identifier: Apache-2.0
 */

package path

import (
	"fmt"
	td1 "github.com/onosproject/config-models/models/testdevice-1.0.x/api"
	"github.com/onosproject/onos-api/go/onos/config/admin"
	configapi "github.com/onosproject/onos-api/go/onos/config/v2"
	"github.com/onosproject/onos-lib-go/pkg/errors"
	"github.com/openconfig/goyang/pkg/yang"
	"github.com/stretchr/testify/assert"
	"os"
	"sort"
	"strings"
	"testing"
)

const Prefixed = "PREFIXED"

func Test_FindPathFromModel(t *testing.T) {

	modelSchema, err := td1.UnzipSchema()
	assert.NoError(t, err)

	roPaths, rwPaths, _, err := extractPaths(modelSchema["Device"], yang.TSUnset, "", "")
	assert.NoError(t, err)
	assert.NotNil(t, roPaths)
	assert.NotNil(t, rwPaths)

	rwPathMap := make(map[string]admin.ReadWritePath)
	for _, pe := range rwPaths {
		rwPathMap[pe.Path] = *pe
	}

	// Regular leaf
	isExactMatch, rwPath, err := FindPathFromModel("/cont1a/leaf1a", rwPathMap, true)
	assert.NoError(t, err)
	assert.True(t, isExactMatch)
	assert.Equal(t, "leaf1a", rwPath.AttrName)

	//Container
	isExactMatch, rwPath, err = FindPathFromModel("/cont1a", rwPathMap, false)
	assert.NoError(t, err)
	assert.False(t, isExactMatch)
	assert.True(t, len(rwPath.AttrName) > 0)

	_, _, err = FindPathFromModel("/cont1a", rwPathMap, true)
	assert.EqualError(t, err, "rpc error: code = InvalidArgument desc = unable to find exact match for RW model path /cont1a. 21 paths inspected")

	// Another leaf
	isExactMatch, rwPath, err = FindPathFromModel("/cont1a/cont2a/leaf2a", rwPathMap, false)
	assert.NoError(t, err)
	assert.True(t, isExactMatch)
	assert.Equal(t, "leaf2a", rwPath.AttrName)

	// List Anon Index entry
	isExactMatch, rwPath, err = FindPathFromModel("/cont1a/list2a[name=test]", rwPathMap, false)
	assert.NoError(t, err)
	assert.False(t, isExactMatch)
	assert.Equal(t, "name", rwPath.AttrName)

	// List key exact
	isExactMatch, rwPath, err = FindPathFromModel("/cont1a/list2a[name=test]/name", rwPathMap, false)
	assert.NoError(t, err)
	assert.True(t, isExactMatch)
	assert.Equal(t, "name", rwPath.AttrName)

	// List non key
	isExactMatch, rwPath, err = FindPathFromModel("/cont1a/list2a[name=test]/tx-power", rwPathMap, false)
	assert.NoError(t, err)
	assert.True(t, isExactMatch)
	assert.Equal(t, "tx-power", rwPath.AttrName)

	// Double keyed List key attr
	isExactMatch, rwPath, err = FindPathFromModel("/cont1a/list5[key1=test][key2=10]/key1", rwPathMap, false)
	assert.NoError(t, err)
	assert.True(t, isExactMatch)
	assert.Equal(t, "key1", rwPath.AttrName)

	// Double keyed List - just list
	isExactMatch, rwPath, err = FindPathFromModel("/cont1a/list5[key1=test][key2=10]", rwPathMap, false)
	assert.NoError(t, err)
	assert.False(t, isExactMatch)
	switch rwPath.AttrName {
	case "key1", "key2":
	default:
		t.Errorf("unexpected attr name %s", rwPath.AttrName)
	}

	// Double keyed List non-key attr
	isExactMatch, rwPath, err = FindPathFromModel("/cont1a/list5[key1=test][key2=10]/leaf5a", rwPathMap, false)
	assert.NoError(t, err)
	assert.True(t, isExactMatch)
	assert.Equal(t, "leaf5a", rwPath.AttrName)

	// Double keyed List invalid attr
	_, _, err = FindPathFromModel("/cont1a/list5[key1=test][key2=10]/invalid", rwPathMap, false)
	assert.EqualError(t, err, "unable to find RW model path /cont1a/list5[key1=test][key2=10]/invalid ( without index /cont1a/list5/invalid). 21 paths inspected")
}

// extractPaths - recursive function that walks the YGOT tree to extract paths
// COPIED FROM config-models/pkg/path/extract.go
// TODO move it out to onos-lib-go or similar
func extractPaths(deviceEntry *yang.Entry, parentState yang.TriState, parentPath string,
	subpathPrefix string) ([]*admin.ReadOnlyPath, []*admin.ReadWritePath, map[string]string, error) {

	readOnlyPaths := make([]*admin.ReadOnlyPath, 0)
	readWritePaths := make([]*admin.ReadWritePath, 0)
	namespaceMappings := make(map[string]string, 0)

	for _, dirEntry := range deviceEntry.Dir {
		itemPath := formatNameAsPath(dirEntry, parentPath, subpathPrefix)
		modname, pfx := extractNamespace(dirEntry)
		if modname != "" && pfx != "" {
			namespaceMappings[modname] = pfx
		}
		if dirEntry.IsLeaf() || dirEntry.IsLeafList() {
			roBase, roSubPath, isReadOnly := earliestRoAncestor(dirEntry)
			// No need to recurse
			t, typeOpts, err := toValueType(dirEntry.Type, dirEntry.IsLeafList())
			if err != nil {
				return nil, nil, nil, err
			}
			// Check to see if this attribute is a key in a list
			tObj := admin.ReadOnlySubPath{
				SubPath:     itemPath,
				ValueType:   t,
				TypeOpts:    typeOpts,
				Description: dirEntry.Description,
				Units:       dirEntry.Units,
				IsAKey:      false,
				AttrName:    dirEntry.Name,
			}
			var enum map[int]string
			if dirEntry.Type.Kind == yang.Yidentityref {
				enum = handleIdentity(dirEntry.Type)
				fmt.Println(enum)
			}
			// Check to see if this attribute is a key in a list
			if dirEntry.Parent.IsList() {
				keyNames := strings.Split(dirEntry.Parent.Key, " ")
				itemPathParts := strings.Split(itemPath, "/")
				attrName := itemPathParts[len(itemPathParts)-1]
				for _, k := range keyNames {
					if strings.EqualFold(stripNamespace(attrName), k) {
						tObj.IsAKey = true
						break
					}
				}
			}
			if isReadOnly {
				roBasePath := fmt.Sprintf("/%s", strings.Join(roBase[1:], "/"))
				var parentPathObj *admin.ReadOnlyPath
				var existing bool
				for _, roPath := range readOnlyPaths {
					if roPath.Path == roBasePath {
						parentPathObj = roPath
						existing = true
					}
				}
				tObj.SubPath = fmt.Sprintf("/%s", strings.Join(roSubPath, "/"))
				if !existing {
					parentPathObj = new(admin.ReadOnlyPath)
					parentPathObj.Path = roBasePath
					readOnlyPaths = append(readOnlyPaths, parentPathObj)
				}
				parentPathObj.SubPath = append(parentPathObj.SubPath, &tObj)
			} else {
				ranges := make([]string, 0)
				for _, r := range dirEntry.Type.Range {
					ranges = append(ranges, fmt.Sprintf("%v", r))
				}
				lengths := make([]string, 0)
				for _, l := range dirEntry.Type.Length {
					lengths = append(lengths, fmt.Sprintf("%v", l))
				}
				firstDefault := ""
				if len(dirEntry.Default) > 0 {
					firstDefault = dirEntry.Default[0]
				}
				rwElem := admin.ReadWritePath{
					Path:        itemPath,
					ValueType:   tObj.ValueType,
					TypeOpts:    tObj.TypeOpts,
					Description: tObj.Description,
					Units:       tObj.Units,
					IsAKey:      tObj.IsAKey,
					AttrName:    tObj.AttrName,
					Mandatory:   dirEntry.Mandatory == yang.TSTrue,
					Default:     firstDefault,
					Range:       ranges,
					Length:      lengths,
				}
				readWritePaths = append(readWritePaths, &rwElem)
			}
		} else if dirEntry.IsContainer() {
			if dirEntry.Config == yang.TSFalse || parentState == yang.TSFalse {
				subpathPfx := subpathPrefix
				if parentState == yang.TSFalse {
					subpathPfx = itemPath[len(parentPath):]
				}
				roChildrenOfRoContainer, _, _, err := extractPaths(dirEntry, yang.TSFalse, itemPath, subpathPfx)
				if err != nil {
					return nil, nil, nil, err
				}
				var currentContainer *admin.ReadOnlyPath
				for _, ro := range readOnlyPaths {
					if ro.Path == itemPath {
						currentContainer = ro
					}
				}
				// Children of a RO container can only ever be RO
				for _, roChildOfRoContainer := range roChildrenOfRoContainer {
					if currentContainer == nil {
						currentContainer = roChildOfRoContainer
						readOnlyPaths = append(readOnlyPaths, currentContainer)
					} else {
						currentContainer.SubPath = append(currentContainer.SubPath, roChildOfRoContainer.SubPath...)
					}
				}
				continue
			}
			readOnlyPathsChildren, readWritePathChildren, namespaceMappingsChildren, err := extractPaths(dirEntry, dirEntry.Config, itemPath, "")
			if err != nil {
				return nil, nil, nil, err
			}
			readOnlyPaths = append(readOnlyPaths, readOnlyPathsChildren...)
			readWritePaths = append(readWritePaths, readWritePathChildren...)
			for k, v := range namespaceMappingsChildren {
				namespaceMappings[k] = v
			}
		} else if dirEntry.IsList() {
			itemPath = formatNameAsPath(dirEntry, parentPath, subpathPrefix)
			if dirEntry.Config == yang.TSFalse || parentState == yang.TSFalse {
				subpathPfx := subpathPrefix
				if parentState == yang.TSFalse {
					subpathPfx = itemPath[len(parentPath):]
				}
				readOnlyPathsChildren, _, _, err := extractPaths(dirEntry, yang.TSFalse, parentPath, subpathPfx)
				if err != nil {
					return nil, nil, nil, err
				}
			next_child:
				for _, readOnlyPathsChild := range readOnlyPathsChildren {
					sameParent := false
					for _, readOnlyPath := range readOnlyPaths {
						if readOnlyPath.Path == readOnlyPathsChild.Path {
							readOnlyPath.SubPath = append(readOnlyPath.SubPath, readOnlyPathsChild.SubPath...)
							sameParent = true
							continue next_child
						}
					}
					if !sameParent {
						readOnlyPaths = append(readOnlyPaths, readOnlyPathsChild)
					}
				}
				continue
			}
			readOnlyPathsChildren, readWritePathsChildren, namespaceMappingsChildren, err := extractPaths(dirEntry, dirEntry.Config, itemPath, "")
			if err != nil {
				return nil, nil, nil, err
			}

			readOnlyPaths = append(readOnlyPaths, readOnlyPathsChildren...)
			readWritePaths = append(readWritePaths, readWritePathsChildren...)
			for k, v := range namespaceMappingsChildren {
				namespaceMappings[k] = v
			}

		} else if dirEntry.IsChoice() || dirEntry.IsCase() {
			// Recurse down through Choice and Case
			readOnlyPathsTemp, readWritePathsTemp, namespaceMappingsTemp, err := extractPaths(dirEntry, dirEntry.Config, parentPath, "")
			if err != nil {
				return nil, nil, nil, err
			}
			readOnlyPaths = append(readOnlyPaths, readOnlyPathsTemp...)
			readWritePaths = append(readWritePaths, readWritePathsTemp...)
			for k, v := range namespaceMappingsTemp {
				namespaceMappings[k] = v
			}
		} else {
			log.Warnf("Unexpected type of leaf for %s %v", itemPath, dirEntry)
		}
	}

	return readOnlyPaths, readWritePaths, namespaceMappings, nil
}

func formatNameAsPath(dirEntry *yang.Entry, parentPath string, subpathPrefix string) string {
	parentAndSubPath := parentPath
	if subpathPrefix != "/" {
		parentAndSubPath = fmt.Sprintf("%s%s", parentPath, subpathPrefix)
	}

	name := formatNameOfChildEntry(dirEntry)

	return fmt.Sprintf("%s/%s", parentAndSubPath, name)
}

func formatNameOfChildEntry(dirEntry *yang.Entry) string {
	name := dirEntry.Name
	_, addPrefixes := os.LookupEnv(Prefixed)
	if addPrefixes && dirEntry.Prefix != nil {
		prefix := dirEntry.Prefix.Name
		if dirEntry.Parent == nil || dirEntry.Parent.Prefix == nil || dirEntry.Parent.Prefix.Name != prefix {
			name = fmt.Sprintf("%s:%s", prefix, name)
		}
	}
	if dirEntry.IsList() {
		//have to ensure index order is consistent where there's more than one
		keyParts := strings.Split(dirEntry.Key, " ")
		sort.Slice(keyParts, func(i, j int) bool {
			return keyParts[i] < keyParts[j]
		})
		for _, k := range keyParts {
			name += fmt.Sprintf("[%s=*]", k)
		}
	}

	return name
}

func toValueType(entry *yang.YangType, isLeafList bool) (configapi.ValueType, []uint64, error) {
	switch entry.Kind.String() {
	case "int8", "int16", "int32", "int64":
		width := extractIntegerWidth(entry.Kind.String())
		if isLeafList {
			return configapi.ValueType_LEAFLIST_INT, []uint64{uint64(width)}, nil
		}
		return configapi.ValueType_INT, []uint64{uint64(width)}, nil
	case "uint8", "uint16", "uint32", "uint64":
		width := extractIntegerWidth(entry.Kind.String())
		if isLeafList {
			return configapi.ValueType_LEAFLIST_UINT, []uint64{uint64(width)}, nil
		}
		return configapi.ValueType_UINT, []uint64{uint64(width)}, nil
	case "decimal64":
		if isLeafList {
			return configapi.ValueType_LEAFLIST_DECIMAL, []uint64{uint64(entry.FractionDigits)}, nil
		}
		return configapi.ValueType_DECIMAL, []uint64{uint64(entry.FractionDigits)}, nil
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

func handleIdentity(yangType *yang.YangType) map[int]string {
	identityMap := make(map[int]string)
	identityMap[0] = "UNSET"
	for i, val := range yangType.IdentityBase.Values {
		identityMap[i+1] = val.Name
	}
	return identityMap
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

// earliestRoAncestor - recursive function to get to the base of the config only ancestor
func earliestRoAncestor(dirEntry *yang.Entry) ([]string, []string, bool) {
	var configFalse bool
	if dirEntry.Parent == nil {
		if dirEntry.Config == yang.TSFalse {
			configFalse = true
		}
		return []string{dirEntry.Name}, nil, configFalse
	}
	itemName := formatNameOfChildEntry(dirEntry)
	base, subPath, parentFalse := earliestRoAncestor(dirEntry.Parent)
	if parentFalse {
		subPath = append(subPath, itemName)
		return base, subPath, parentFalse
	} else if dirEntry.Config == yang.TSFalse {
		base = append(base, itemName)
		return base, nil, true
	}
	base = append(base, itemName)
	return base, nil, configFalse
}

func extractNamespace(e *yang.Entry) (string, string) {
	return e.Namespace().Name, e.Prefix.Name
}

// YGOT does not handle namespaces, so there is no point in us maintaining them
// They may come from the southbound or northbound in a JSON payload though, so
// we have to be able to deal with them
func stripNamespace(path string) string {
	pathParts := strings.Split(path, "/")
	for idx, pathPart := range pathParts {
		colonPos := strings.Index(pathPart, ":")
		if colonPos > 0 {
			pathParts[idx] = pathPart[colonPos+1:]
		}
	}
	return strings.Join(pathParts, "/")
}
