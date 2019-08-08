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
	"errors"
	"regexp"
)

// ConfigValue is a of a path, a value and a type
type ConfigValue struct {
	Path string
	TypedValue
}

// IsPathValid tests for valid paths. Path is valid if it
// 1) starts with a slash
// 2) is followed by at least one of alphanumeric or any of : = - _ [ ]
// 3) and any further combinations of 1+2
// Two contiguous slashes are not allowed
// Paths not starting with slash are not allowed
func (c ConfigValue) IsPathValid() error {
	r1 := regexp.MustCompile(`(/[a-zA-Z0-9:=\-_,[\]]+)+`)

	match := r1.FindString(c.Path)
	if c.Path != match {
		return errors.New(c.Path)
	}
	return nil
}
