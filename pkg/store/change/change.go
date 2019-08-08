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

// Package change defines change records for tracking device configuration changes.
package change

import (
	"crypto/sha1"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"sort"
	"time"
)

// B64 is an alias for the function encoding a byte array to a Base64 string
var b64 = base64.StdEncoding.EncodeToString

// ID is an alias for the ID of the change
type ID []byte

// Change is one of the primary objects to be stored
// A model of the Change object - its is an immutable collection of ChangeValues
type Change struct {
	ID          ID
	Description string
	Created     time.Time
	Config      ValueCollections
}

// Stringer method for the Change
func (c Change) String() string {
	jsonstr, _ := json.Marshal(c)
	return string(jsonstr)
}

// IsValid checks the contents of the Change against its hash
// This enforces the immutability of the Change. The hash is only on the changes
// and not the description or the time
func (c Change) IsValid() error {
	if len(c.ID) == 0 {
		return fmt.Errorf("Empty Change")
	}

	h := sha1.New()
	jsonstr, _ := json.Marshal(c.Config)
	_, err1 := io.WriteString(h, string(jsonstr))
	if err1 != nil {
		return err1
	}

	hash := h.Sum(nil)
	if b64(hash) == b64(c.ID) {
		return nil
	}
	return fmt.Errorf("Change '%s': Calculated hash '%s' does not match. Current: %s",
		b64(c.ID), b64(hash), jsonstr)
}

// CreateChange creates a Change object from ChangeValues
// The ID is a has generated from the change values,and does not include
// the description or the time. This way changes that have an identical meaning
// can be identified
func CreateChange(config ValueCollections, desc string) (*Change, error) {
	h := sha1.New()
	t := time.Now()

	if len(config) == 0 {
		return nil, fmt.Errorf("invalid change %s - no config values found", desc)
	}

	sort.Slice(config, func(i, j int) bool {
		return (*config[i]).Path < (*config[j]).Path
	})

	var pathList = make([]string, len(config))
	// If a path is repeated then reject
	for _, cv := range config {
		for _, p := range pathList {
			if cv.Path == p { // Suspend for leaf-list
				return nil, errors.New("Error Path " + p + " is repeated in change")
			}
		}
		pathList = append(pathList, cv.Path)
	}

	// Calculate a hash from the config, description and timestamp
	jsonstr, _ := json.Marshal(config)
	_, err1 := io.WriteString(h, string(jsonstr))
	if err1 != nil {
		return nil, err1
	}

	hash := h.Sum(nil)

	return &Change{
		Config:      config,
		ID:          hash,
		Description: desc,
		Created:     t,
	}, nil
}

// CreateChangeValuesNoRemoval creates a Change object from ConfigValue
// The ID is a has generated from the change values,and does not include
// the description or the time. This way changes that have an identical meaning
// can be identified
func CreateChangeValuesNoRemoval(config []*ConfigValue, desc string) (*Change, error) {
	h := sha1.New()
	t := time.Now()

	sort.Slice(config, func(i, j int) bool {
		return (*config[i]).Path < (*config[j]).Path
	})

	var pathList = make([]string, len(config))
	// If a path is repeated then reject
	for _, cv := range config {
		for _, p := range pathList {
			if cv.Path == p {
				return nil, errors.New("Error Path " + p + " is repeated in change")
			}
		}
		pathList = append(pathList, cv.Path)
	}

	// Calculate a hash from the config, description and timestamp
	jsonstr, _ := json.Marshal(config)
	_, err1 := io.WriteString(h, string(jsonstr))
	if err1 != nil {
		return nil, err1
	}

	hash := h.Sum(nil)

	configColl := make(ValueCollections, 0)

	for _, c := range config {
		configColl = append(configColl, &Value{
			ConfigValue: *c,
			Remove:      false})
	}

	return &Change{
		Config:      configColl,
		ID:          hash,
		Description: desc,
		Created:     t,
	}, nil
}
