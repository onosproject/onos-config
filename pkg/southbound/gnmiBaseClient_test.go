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

package southbound

import (
	"context"
	"github.com/openconfig/gnmi/client"
	"gotest.tools/assert"
	"testing"
)

func Test_GnmiBaseClient(t *testing.T) {
	baseClient := GnmiBaseClientFactory()
	assert.Assert(t, baseClient != nil)
	query := client.Query{Type: client.Unknown}
	err := baseClient.Subscribe(context.TODO(), query, "XXX")
	assert.ErrorContains(t, err, "Destination.Addrs is empty")
}
