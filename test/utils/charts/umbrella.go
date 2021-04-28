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

package charts

import (
	"github.com/onosproject/helmit/pkg/helm"
	"github.com/onosproject/onos-test/pkg/onostest"
)

// CreateUmbrellaRelease creates a helm release for an onos-umbrella instance
func CreateUmbrellaRelease() *helm.HelmRelease {
	return helm.Chart("onos-umbrella", onostest.OnosChartRepo).
		Release("onos-umbrella").
		Set("import.onos-gui.enabled", false).
		Set("import.onos-cli.enabled", false). // not needed - can enabled be through Helm for investigations
		Set("onos-topo.image.tag", "latest").
		Set("onos-config.image.tag", "latest")
}
