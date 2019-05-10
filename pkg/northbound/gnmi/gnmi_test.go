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

package gnmi

import (
	"fmt"
	"github.com/onosproject/onos-config/pkg/manager"
	"github.com/onosproject/onos-config/pkg/store"
	"github.com/onosproject/onos-config/pkg/utils"
	"github.com/openconfig/gnmi/proto/gnmi"
	"gotest.tools/assert"
	"os"
	"testing"
)

var (
	server *Server
	mgr *manager.Manager
	configurationStore map[string]store.Configuration
)

func TestMain(m *testing.M) {
	//var err error

	server = &Server{}

	cfgStore, err := store.LoadConfigStore("../../../configs/configStore-sample.json")
	if err != nil {
		fmt.Println("Unexpected config store loading error", err)
		os.Exit(-1)
	}
	if cfgStore.Version == "" {
		fmt.Println("Expected Config store to be loaded")
		os.Exit(-1)
	}

	mgr = manager.GetManager()
	mgr.ConfigStore = &cfgStore

	os.Exit(m.Run())
}

func Test_getalldevices(t *testing.T) {

	allDevicesPath := gnmi.Path{Elem:make([]*gnmi.PathElem, 0), Target:"*"}

	request := gnmi.GetRequest{
		Path: []*gnmi.Path{&allDevicesPath},
	}

	result, err := server.Get(nil, &request)
	assert.NilError(t, err, "Unexpected error calling gNMI Set")

	assert.Equal(t, len(result.Notification), 1, "Expected 1 notification")
	assert.Equal(t, len(result.Notification[0].Update), 1, "Expected 1 update")

	assert.Equal(t, result.Notification[0].Update[0].Path.Target, "*", "Expected target")

	deviceListStr := utils.StrVal(result.Notification[0].Update[0].Val)

	assert.Equal(t, deviceListStr, "[Device1, Device2]", "Expected value")
}
