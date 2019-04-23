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
)

type ErrInvalidPath string

func (e ErrInvalidPath) Error() string {
	return fmt.Sprintf("Invalid path: %s", e)
}

/**
 * A model of a ConfigValue - path and a value
 */
type ConfigValue struct {
	Path string // Can be changed to *gpb.Path
	Value string // Can be changed to *gpb.TypedValue
}
