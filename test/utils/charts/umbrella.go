// SPDX-FileCopyrightText: 2020-present Open Networking Foundation <info@opennetworking.org>
//
// SPDX-License-Identifier: Apache-2.0

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
		Set("onos-topo.image.tag", "latest").
		Set("onos-config.image.tag", "latest").
		Set("onos-config-model.image.tag", "latest")
}
