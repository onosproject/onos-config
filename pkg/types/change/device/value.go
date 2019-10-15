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

package device

import (
	"errors"
	"regexp"
)

// NewChangeValue decodes a path and value in to an object
func NewChangeValue(path string, value *TypedValue, isRemove bool) (*ChangeValue, error) {
	cv := ChangeValue{
		Path:    path,
		Value:   value,
		Removed: isRemove,
	}
	err := IsPathValid(cv.GetPath())
	if err != nil {
		return nil, err
	}
	return &cv, nil
}

// IsPathValid tests for valid paths. Path is valid if it
// 1) starts with a slash
// 2) is followed by at least one of alphanumeric or any of : = - _ [ ]
// 3) and any further combinations of 1+2
// Two contiguous slashes are not allowed
// Paths not starting with slash are not allowed
func IsPathValid(path string) error {
	r1 := regexp.MustCompile(`(/[a-zA-Z0-9:=\-_,[\]]+)+`)

	match := r1.FindString(path)
	if path != match {
		return errors.New(path)
	}
	return nil
}
