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
	"github.com/onosproject/onos-config/pkg/dispatcher"
	"github.com/onosproject/onos-config/pkg/events"
	"github.com/onosproject/onos-config/pkg/manager"
	log "k8s.io/klog"
	"os"
	"testing"
)

// TestMain should only contain static data.
// It is run once for all tests - each test is then run on its own thread, so if
// anything is shared the order of it's modification is not deterministic
// Also there can only be one TestMain per package
func TestMain(m *testing.M) {
	log.SetOutput(os.Stdout)
	os.Exit(m.Run())
}

// setUp should not depend on any global variables
func setUp() (*Server, *manager.Manager) {
	var server = &Server{}

	mgr, err := manager.LoadManager(
		"../../../configs/configStore-sample.json",
		"../../../configs/changeStore-sample.json",
		"../../../configs/deviceStore-sample.json",
		"../../../configs/networkStore-sample.json",
	)
	if err != nil {
		log.Error("Expected manager to be loaded ", err)
		os.Exit(-1)
	}

	mgr = manager.GetManager()
	mgr.Dispatcher = dispatcher.NewDispatcher()
	mgr.TopoChannel = make(chan events.TopoEvent)
	go listenToTopoLoading(mgr.TopoChannel)
	mgr.ChangesChannel = make(chan events.ConfigEvent)
	go mgr.Dispatcher.Listen(mgr.ChangesChannel)

	log.Info("Finished setUp()")
	return server, mgr
}

func listenToTopoLoading(deviceChan <-chan events.TopoEvent) {
	for deviceConfigEvent := range deviceChan {
		log.Info("Ignoring event for testing ", deviceConfigEvent)
	}
}
