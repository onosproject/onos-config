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
//

package config

import (
	"context"
	"testing"
	"time"

	configapi "github.com/onosproject/onos-api/go/onos/config/v2"

	"github.com/onosproject/onos-test/pkg/onostest"
	"github.com/stretchr/testify/assert"

	"github.com/onosproject/helmit/pkg/helm"
	topoapi "github.com/onosproject/onos-api/go/onos/topo"

	gnmiutils "github.com/onosproject/onos-config/test/utils/gnmi"
	"github.com/onosproject/onos-config/test/utils/proto"
	"github.com/openconfig/gnmi/proto/gnmi_ext"
)

const (
	modPath           = "/system/clock/config/timezone-name"
	modValue          = "Europe/Rome"
	offlineTargetName = "offline-target-device-simulator"
)

// TestOfflineTarget tests set/query of a single GNMI path to a single target that is initially not connected to onos-config
func (s *TestSuite) TestOfflineTarget(t *testing.T) {
	// create a target entity in topo
	createOfflineTarget(t, offlineTargetName, "devicesim-1.0.x", "1.0.0", offlineTargetName+":11161")

	// Make a GNMI client to use for requests
	gnmiClient := gnmiutils.GetGNMIClientOrFail(t)

	// Sends a set request using onos-config NB
	targetPath := gnmiutils.GetTargetPathWithValue(offlineTargetName, modPath, modValue, proto.StringVal)
	gnmiutils.SetGNMIValueOrFail(t, gnmiClient, targetPath, gnmiutils.NoPaths, []*gnmi_ext.Extension{})

	// Install simulator
	simulator := helm.
		Chart("device-simulator", onostest.OnosChartRepo).
		Release(offlineTargetName).
		Set("image.tag", "latest")
	err := simulator.Install(true)
	assert.NoError(t, err, "could not install device simulator %v", err)

	// Wait for config to connect to the target
	gnmiutils.WaitForTargetAvailable(t, topoapi.ID(simulator.Name()), time.Minute)
	err = gnmiutils.WaitForConfigurationCompleteOrFail(t, configapi.ConfigurationID(simulator.Name()), time.Minute)
	assert.NoError(t, err)

	gnmiutils.CheckGNMIValue(t, gnmiClient, targetPath, modValue, 0, "Query after set returned the wrong value")

	// Check that the value was set properly on the target, wait for configuration gets completed
	targetGnmiClient := gnmiutils.GetTargetGNMIClientOrFail(t, simulator)
	gnmiutils.CheckTargetValue(t, targetGnmiClient, targetPath, modValue)

	err = simulator.Uninstall()
	assert.NoError(t, err)
}

func createOfflineTarget(t *testing.T, targetID topoapi.ID, targetType string, targetVersion string, targetAddress string) {
	topoClient, err := gnmiutils.NewTopoClient()
	assert.NotNil(t, topoClient)
	assert.Nil(t, err)

	newTarget := &topoapi.Object{
		ID:   targetID,
		Type: topoapi.Object_ENTITY,
		Obj: &topoapi.Object_Entity{
			Entity: &topoapi.Entity{
				KindID: topoapi.ID(targetType),
			},
		},
	}
	err = newTarget.SetAspect(&topoapi.TLSOptions{
		Insecure: true,
		Plain:    true,
	})
	assert.NoError(t, err)

	err = newTarget.SetAspect(&topoapi.Configurable{
		Type:    targetType,
		Address: targetAddress,
		Version: targetVersion,
		Timeout: uint64(3000000 * time.Second),
	})
	assert.NoError(t, err)
	err = topoClient.Create(context.Background(), newTarget)
	assert.NoError(t, err)
}
