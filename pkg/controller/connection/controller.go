// Copyright 2021-present Open Networking Foundation.
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

package connection

import (
	"github.com/onosproject/onos-config/pkg/southbound/gnmi"
	"github.com/onosproject/onos-config/pkg/store/topo"
	"github.com/onosproject/onos-lib-go/pkg/controller"
	"github.com/onosproject/onos-lib-go/pkg/logging"
)

var log = logging.GetLogger("controller", "connection")

// NewController returns a new gNMI connection  controller
func NewController(topo topo.Store, conns gnmi.ConnManager) *controller.Controller {
	c := controller.NewController("connection")

	c.Watch(&ConnWatcher{
		conns: conns,
	})

	c.Watch(&TopoWatcher{
		topo:  topo,
		conns: conns,
	})

	c.Reconcile(&Reconciler{
		conns: conns,
		topo:  topo,
	})

	return c
}

// Reconciler reconciles gNMI connections
type Reconciler struct {
	conns gnmi.ConnManager
	topo  topo.Store
}

func (r *Reconciler) Reconcile(id controller.ID) (controller.Result, error) {
	panic("implement me")
}
