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

package store

import (
	"fmt"
	devicechange "github.com/onosproject/onos-api/go/onos/config/change/device"
	"gotest.tools/assert"
	"os"
	"strconv"
	"testing"
)

const (
	Test1Cont1ACont2ALeaf2A = "/cont1a/cont2a/leaf2a"
	Test1Cont1ACont2ALeaf2B = "/cont1a/cont2a/leaf2b"
	Test1Cont1ACont2ALeaf2C = "/cont1a/cont2a/leaf2c"
	Test1Cont1ACont2ALeaf2D = "/cont1a/cont2a/leaf2d"
	Test1Cont1ACont2ALeaf2E = "/cont1a/cont2a/leaf2e"
	Test1Cont1ACont2ALeaf2F = "/cont1a/cont2a/leaf2f"
	Test1Cont1ACont2ALeaf2G = "/cont1a/cont2a/leaf2g"
	Test1Leaftoplevel       = "/leafAtTopLevel"
)

const (
	ValueLeaftopWxy1234 = "WXY-1234"
)

func TestMain(m *testing.M) {
	os.Exit(m.Run())
}

func BenchmarkCreateChangeValue(b *testing.B) {

	for i := 0; i < b.N; i++ {
		path := fmt.Sprintf("/test-%s", strconv.Itoa(b.N))
		cv, _ := devicechange.NewChangeValue(path, devicechange.NewTypedValueUint(uint(i), 32), false)
		err := devicechange.IsPathValid(cv.Path)
		assert.NilError(b, err, "path not valid %s", err)

	}
}
