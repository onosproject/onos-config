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

package store

import (
	"fmt"
)

// ChangeValue is a model that extends ConfigValue - path and a value and bool
type ChangeValue struct {
	ConfigValue
	Remove bool
}

// ChangeValue.String is the stringer for ChangeValue
func (c ChangeValue) String() string {
	if c.Path == "" {
		return "InvalidChange"
	}
	return fmt.Sprintf("%s %s %t", c.Path, c.Value, c.Remove)
}

// CreateChangeValue decodes a path and value in to an object
func CreateChangeValue(path string, value string, isRemove bool) (ChangeValue, error) {
	cv := ChangeValue{
		ConfigValue{path, value},
		isRemove,
	}
	err := cv.ConfigValue.IsPathValid()
	if err != nil {
		return ChangeValue{}, err
	}
	return cv, nil
}

// ChangeValueCollection is an alias for a slice of ChangeValues
type ChangeValueCollection []ChangeValue
