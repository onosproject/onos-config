/*
 * Copyright 2019-present Open Networking Foundation
 *
 * Licensed under the Apache License, Configuration 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */
package store

import (
	"crypto/sha1"
	"encoding/hex"
	"encoding/json"
	"github.com/openconfig/gnmi/proto/gnmi"
	"io"
	"sort"
	"strings"
	"time"
)

type ErrInvalidChange string

func (e ErrInvalidChange) Error() string {
	return hex.EncodeToString([]byte(e))
}

type ChangeId []byte

// A model of the Change object - its is an immutable collection of ChangeValues
// Change is one of the primary objects to be stored
type Change struct {
	Id     ChangeId
	Description string
	Created time.Time
	Config ChangeValueCollection
}

// Stringer method for the Change
func (c Change) String() string {
	jsonstr, err := json.Marshal(c)
	if err != nil {
		return err.Error()
	}
	return string(jsonstr)
}

// Check the contents of the Change against its hash
// This enforces the immutability of the Change
func (c Change) IsValid() error {
	h := sha1.New()
	jsonstr, _ := json.Marshal(c.Config)
	_, err1 := io.WriteString(h, string(jsonstr))
	if err1 != nil { return err1}

	_, err2 := io.WriteString(h, c.Description)
	if err2 != nil { return err1}

	_, err3 := io.WriteString(h, c.Created.String())
	if err3 != nil { return err1}

	hash := h.Sum(nil)
	if (hex.EncodeToString(hash) == hex.EncodeToString(c.Id)) {
		return nil
	} else {
		e := ErrInvalidChange(hash)
		return e
	}
}

func (c Change) GnmiChange() gnmi.SetRequest {
	var deletePaths = []*gnmi.Path{}
	var replacedPaths = []*gnmi.Update{}
	var updatedPaths = []*gnmi.Update{}

	for _, changeValue := range c.Config {
		elems := strings.Split(changeValue.Path, "/")
		pathElems := []*gnmi.PathElem{}
		for idx, elem := range elems {
			if idx == 0 { continue } //Skip first one
			partsNs := strings.Split(elem, ":")
			if (len(partsNs) == 2) {
				// We have to discard the namespace as gNMI doesn't handle it
				elem =  partsNs[1]
			}
			var pathElem = gnmi.PathElem{}
			partsIdx := strings.Split(elem, "=")
			if len(partsIdx) == 2 {
				pathElem.Name = partsIdx[0]
				pathElem.Key = make(map[string]string)
				pathElem.Key["name"] = partsIdx[1]
			} else {
				pathElem.Name = partsIdx[0]
			}
			pathElems = append(pathElems, &pathElem)
		}
		if (changeValue.Remove) {
			deletePaths = append(deletePaths, &gnmi.Path{Elem: pathElems})
		} else {
			typedValue := gnmi.TypedValue_StringVal{changeValue.Value}
			value := gnmi.TypedValue{Value: &typedValue}
			updatePath := gnmi.Path{Elem: pathElems}
			updatedPaths = append(updatedPaths, &gnmi.Update{Path: &updatePath, Val:&value})
		}
	}

	var setRequest = gnmi.SetRequest{
		Delete: deletePaths,
		Replace: replacedPaths,
		Update: updatedPaths,
	}

	return setRequest
}

// Create a Change object from ChangeValues
func CreateChange(config ChangeValueCollection, desc string) (Change, error) {
	h := sha1.New()
	t := time.Now()

	sort.Slice(config, func(i, j int) bool {
		return config[i].Path < config[j].Path
	});

	var pathList = make([]string, len(config))
	// If a path is repeated then reject
	for _, cv := range config {
		for _,p := range pathList {
			if strings.Compare(cv.Path, p) == 0 {
				return Change{}, ErrInvalidChange("Error Path " + p + " is repeated in change")
			}
		}
		pathList = append(pathList, cv.Path)
	}

	// Calculate a hash from the config, description and timestamp
	jsonstr, _ := json.Marshal(config)
	_, err1 := io.WriteString(h, string(jsonstr))
	if err1 != nil { return Change{}, err1}

	_, err2 := io.WriteString(h, desc)
	if err2 != nil { return Change{}, err1}

	_, err3 := io.WriteString(h, t.String())
	if err3 != nil { return Change{}, err1}

	hash := h.Sum(nil)

	return Change{
		Config: config,
		Id: hash,
		Description: desc,
		Created: t,
	}, nil
}
