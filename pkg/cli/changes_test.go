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

package cli

import (
	"gotest.tools/assert"
	"testing"
)

func Test_wrapline_over(t *testing.T) {
	const test1 = "abcdefghijklmnopqrstuvwxyz"

	test1Wrapped := wrapPath(test1, 20, 1)
	assert.Equal(t, test1Wrapped, "abcdefghijklmnopqrst\n\t  uvwxyz              ")
}

func Test_wrapline_under(t *testing.T) {
	const test1 = "abcdef"

	test1Wrapped := wrapPath(test1, 20, 1)
	assert.Equal(t, test1Wrapped, "abcdef              ")
}

func Test_wrapline_exact(t *testing.T) {
	const test1 = "abcdefghijklmnopqrst"

	test1Wrapped := wrapPath(test1, 20, 1)
	assert.Equal(t, test1Wrapped, test1)
}
