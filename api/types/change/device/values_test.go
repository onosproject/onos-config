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
//

package device

import (
	"gotest.tools/assert"
	"strings"
	"testing"
)

func Test_NewChangedValue(t *testing.T) {
	path := "/a/b/c"
	badPath := "a@b@c"
	value := NewTypedValueUint64(64)
	const isRemove = false
	changeValueBad, errorBad := NewChangeValue(badPath, value, isRemove)
	assert.Assert(t, errorBad != nil)
	assert.Assert(t, strings.Contains(errorBad.Error(), badPath))
	assert.Assert(t, changeValueBad == nil)
	changeValue, err := NewChangeValue(path, value, isRemove)
	assert.Assert(t, err == nil)
	assert.Assert(t, changeValue != nil)
	assert.Assert(t, changeValue.Path == path)
	assert.Assert(t, changeValue.Value.ValueToString() == "64")
}
