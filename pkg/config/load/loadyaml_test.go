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

package load

import (
	"github.com/openconfig/gnmi/proto/gnmi"
	"gotest.tools/assert"
	"testing"
)

func Test_LoadConfig1(t *testing.T) {
	config, err := GetConfigGnmi("onos-config-load-sample-gnmi.yaml")
	assert.NilError(t, err, "Unexpected error loading gnmi setrequest yaml")

	assert.Equal(t, "", config.SetRequest.Prefix.Target)
	assert.Equal(t, 1, len(config.SetRequest.Prefix.Elem), "Unexpected number of Prefix Elems")
	assert.Equal(t, "e2node", config.SetRequest.Prefix.Elem[0].Name)

	assert.Equal(t, 6, len(config.SetRequest.Update), "Unexpected number of Updates")
	assert.Equal(t, 2, len(config.SetRequest.Update[0].Path.Elem), "Unexpected number of elems in path of update 0")
	assert.Equal(t, "intervals", config.SetRequest.Update[0].Path.Elem[0].Name)
	assert.Equal(t, "RadioMeasReportPerUe", config.SetRequest.Update[0].Path.Elem[1].Name)

	assert.Equal(t, uint64(20), config.SetRequest.Update[0].Val.UIntValue.UintVal)
}

func Test_ConvertConfig(t *testing.T) {
	config, err := GetConfigGnmi("onos-config-load-sample-gnmi.yaml")
	assert.NilError(t, err, "Unexpected error loading gnmi setrequest yaml")

	gnmiSr := ToGnmiSetRequest(&config)
	assert.Equal(t, "", gnmiSr.Prefix.Target)
	assert.Equal(t, 1, len(gnmiSr.Prefix.Elem), "Unexpected number of Prefix Elems")
	assert.Equal(t, "e2node", gnmiSr.Prefix.Elem[0].Name)

	assert.Equal(t, 6, len(gnmiSr.Update), "Unexpected number of Updates")
	assert.Equal(t, 2, len(gnmiSr.Update[0].Path.Elem), "Unexpected number of elems in path of update 0")
	assert.Equal(t, "intervals", gnmiSr.Update[0].Path.Elem[0].Name)
	assert.Equal(t, "RadioMeasReportPerUe", gnmiSr.Update[0].Path.Elem[1].Name)

	assert.Equal(t, uint64(20), gnmiSr.Update[0].Val.Value.(*gnmi.TypedValue_UintVal).UintVal)
}
