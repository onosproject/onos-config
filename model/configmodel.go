// Copyright 2020-present Open Networking Foundation.
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

package configmodel

import (
	"fmt"
	"github.com/openconfig/gnmi/proto/gnmi"
	"github.com/openconfig/goyang/pkg/yang"
	"github.com/openconfig/ygot/ygot"
)

// Name is a config model name
type Name string

// Version is a config model version
type Version string

// Revision is a config module revision
type Revision string

// GetStateMode defines the Getstate handling
type GetStateMode string

const (
	// GetStateNone - device type does not support Operational State at all
	GetStateNone GetStateMode = "GetStateNone"
	// GetStateOpState - device returns all its op state attributes by querying
	// GetRequest_STATE and GetRequest_OPERATIONAL
	GetStateOpState GetStateMode = "GetStateOpState"
	// GetStateExplicitRoPaths - device returns all its op state attributes by querying
	// exactly what the ReadOnly paths from YANG - wildcards are handled by device
	GetStateExplicitRoPaths GetStateMode = "GetStateExplicitRoPaths"
	// GetStateExplicitRoPathsExpandWildcards - where there are wildcards in the
	// ReadOnly paths 2 calls have to be made - 1) to expand the wildcards in to
	// real paths (since the device doesn't do it) and 2) to query those expanded
	// wildcard paths - this is the Stratum 1.0.0 method
	GetStateExplicitRoPathsExpandWildcards GetStateMode = "GetStateExplicitRoPathsExpandWildcards"
)

// ModelInfo is config model info
type ModelInfo struct {
	Name         Name         `json:"name"`
	Version      Version      `json:"version"`
	GetStateMode GetStateMode `json:"getStateMode"`
	Files        []FileInfo   `json:"files"`
	Modules      []ModuleInfo `json:"modules"`
	Plugin       PluginInfo   `json:"plugin"`
}

func (m ModelInfo) String() string {
	return fmt.Sprintf("%s@%s", m.Name, m.Version)
}

// ModuleInfo is a config module info
type ModuleInfo struct {
	Name         Name     `json:"name"`
	File         string   `json:"file"`
	Organization string   `json:"organization"`
	Revision     Revision `json:"revision"`
}

// FileInfo is a config file info
type FileInfo struct {
	Path string `json:"path"`
	Data []byte `json:"data"`
}

// PluginInfo is config model plugin info
type PluginInfo struct {
	Name    Name    `json:"name"`
	Version Version `json:"version"`
}

// ConfigModel is a configuration model data
type ConfigModel interface {
	// Info returns the config model info
	Info() ModelInfo

	// Data returns the config model data
	Data() []*gnmi.ModelData

	// Schema returns the config model schema
	Schema() (map[string]*yang.Entry, error)

	// GetStateMode returns the get state mode
	GetStateMode() GetStateMode

	// Unmarshaler returns the config model unmarshaler function
	Unmarshaler() Unmarshaler

	// Validator returns the config model validator function
	Validator() Validator
}

// Unmarshaler is a config model unmarshaler function
type Unmarshaler func([]byte) (*ygot.ValidatedGoStruct, error)

// Validator is a config model validator function
type Validator func(model *ygot.ValidatedGoStruct, opts ...ygot.ValidationOption) error
