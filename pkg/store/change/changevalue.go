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

package change

import (
	"fmt"
)

// Value is a model that extends ConfigValue - path and a value and bool
type Value struct {
	ConfigValue
	Remove bool
}

// Value.String is the stringer for Value
func (c Value) String() string {
	if c.Path == "" {
		return "InvalidChange"
	}
	return fmt.Sprintf("%s %v %t", c.Path, c.Value, c.Remove)
}

// CreateChangeValue decodes a path and value in to an object
func CreateChangeValue(path string, value *TypedValue, isRemove bool) (*Value, error) {
	cv := Value{
		ConfigValue{path, *value},
		isRemove,
	}
	err := cv.ConfigValue.IsPathValid()
	if err != nil {
		return nil, err
	}
	return &cv, nil
}

// ValueCollections is an alias for a slice of ChangeValues
type ValueCollections []*Value
