// Copyright 2022-present Open Networking Foundation.
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

package gnmi

import (
	"context"
	"fmt"
	configapi "github.com/onosproject/onos-api/go/onos/config/v2"
	"github.com/openconfig/gnmi/proto/gnmi"
	"github.com/stretchr/testify/assert"
	"testing"
)

func Test_BasicSetUpdate(t *testing.T) {
	test := createServer(t)
	defer test.atomix.Stop()
	defer test.mctl.Finish()

	test.startControllers(t)
	defer test.stopControllers()

	setupTopoAndRegistry(test, "target-1", "devicesim", "1.0.0")

	targetID := configapi.TargetID("target-1")
	request := gnmi.SetRequest{
		Update: []*gnmi.Update{
			{
				Path: targetPath(t, targetID, "foo"),
				Val:  &gnmi.TypedValue{Value: &gnmi.TypedValue_StringVal{StringVal: "Hello world!"}},
			},
		},
	}

	result, err := test.server.Set(context.TODO(), &request)
	assert.NoError(t, err)
	fmt.Printf("%+v\n", result)
}
