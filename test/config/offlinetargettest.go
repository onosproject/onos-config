// SPDX-FileCopyrightText: 2020-present Open Networking Foundation <info@opennetworking.org>
//
// SPDX-License-Identifier: Apache-2.0
//

package config

import (
	"context"
	"github.com/onosproject/onos-config/test"
	gnmiapi "github.com/openconfig/gnmi/proto/gnmi"
	"time"

	topoapi "github.com/onosproject/onos-api/go/onos/topo"

	gnmiutils "github.com/onosproject/onos-config/test/utils/gnmi"
	"github.com/onosproject/onos-config/test/utils/proto"
)

const (
	modPath           = "/system/clock/config/timezone-name"
	modValue          = "Europe/Rome"
	offlineTargetName = "offline-target-device-simulator"
)

// TestOfflineTarget tests set/query of a single GNMI path to a single target that is initially not connected to onos-config
func (s *TestSuite) TestOfflineTarget(ctx context.Context) {
	// create a target entity in topo
	s.createOfflineTarget(offlineTargetName, "devicesim", "1.0.0", offlineTargetName+":11161")

	// Make a GNMI client to use for requests
	gnmiClient := s.NewOnosConfigGNMIClientOrFail(ctx, test.WithRetry)

	// Sends a set request using onos-config NB
	targetPath := gnmiutils.GetTargetPathWithValue(offlineTargetName, modPath, modValue, proto.StringVal)
	var setReq = &gnmiutils.SetRequest{
		Ctx:         ctx,
		Client:      gnmiClient,
		Encoding:    gnmiapi.Encoding_PROTO,
		UpdatePaths: targetPath,
	}
	setReq.SetOrFail(s.T())

	// Install and start target simulator
	s.SetupSimulator(ctx, offlineTargetName, false)
	defer s.TearDownSimulator(ctx, offlineTargetName)

	// Wait for config to connect to the target
	s.WaitForTargetAvailable(ctx, topoapi.ID(offlineTargetName), time.Minute)
	var getConfigReq = &gnmiutils.GetRequest{
		Ctx:        ctx,
		Client:     gnmiClient,
		Paths:      targetPath,
		Extensions: s.SyncExtension(),
		Encoding:   gnmiapi.Encoding_PROTO,
	}
	getConfigReq.CheckValues(s.T(), modValue)

	// Check that the value was set properly on the target, wait for configuration gets completed
	targetGnmiClient := s.NewSimulatorGNMIClientOrFail(ctx, offlineTargetName)
	var getTargetReq = &gnmiutils.GetRequest{
		Ctx:      ctx,
		Client:   targetGnmiClient,
		Encoding: gnmiapi.Encoding_JSON,
		Paths:    targetPath,
	}
	getTargetReq.CheckValues(s.T(), modValue)
}

func (s *TestSuite) createOfflineTarget(targetID topoapi.ID, targetType string, targetVersion string, targetAddress string) {
	topoClient, err := gnmiutils.NewTopoClient()
	s.NotNil(topoClient)
	s.Nil(err)

	newTarget, err := s.NewTargetEntity(string(targetID), targetType, targetVersion, targetAddress)
	s.NoError(err)
	err = topoClient.Create(context.Background(), newTarget)
	s.NoError(err)
}
