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
	"fmt"
	"strings"
)

/**
 * A model of a ChangeValue extends ConfigValue - path and a value and bool
 */
type ChangeValue struct {
	ConfigValue
	Remove bool
}

func (c ChangeValue) String() string {
	return fmt.Sprintf("%s %s %t", c.Path, c.Value, c.Remove)
}

/**
 * Decodes a path and value in to an object
 */
func CreateChangeValue(path string, value string, isRemove bool) (ChangeValue, error) {
	if len(path) < 2 || !strings.ContainsAny(path, "/") {
		e := ErrInvalidPath(path)
		return ChangeValue{}, &e
	}

	//v := ConfigValue{Value:value, Path:path}
	return ChangeValue{
		ConfigValue{path, value},
		isRemove,
	}, nil

}

type ChangeValueCollection []ChangeValue

