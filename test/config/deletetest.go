// SPDX-FileCopyrightText: 2020-present Open Networking Foundation <info@opennetworking.org>
//
// SPDX-License-Identifier: Apache-2.0

package config

import (
	"context"
	"github.com/onosproject/onos-api/go/onos/topo"
	"github.com/onosproject/onos-config/test"
	"github.com/onosproject/onos-config/test/utils/proto"
	gnmiapi "github.com/openconfig/gnmi/proto/gnmi"

	"github.com/onosproject/onos-api/go/onos/config/admin"
	gnmiutils "github.com/onosproject/onos-config/test/utils/gnmi"
)

// TestDeleteAndRollback tests target deletion and rollback
func (s *TestSuite) TestDeleteAndRollback() {
	const (
		newValue = "new-value"
		newPath  = "/system/config/login-banner"
	)

	// Wait for config to connect to the target
	s.WaitForTargetAvailable(topo.ID(s.simulator1.Name))

	// Make a GNMI client to use for requests
	gnmiClient := s.NewOnosConfigGNMIClientOrFail(test.NoRetry)

	// Set values
	var targetPath = gnmiutils.GetTargetPathWithValue(s.simulator1.Name, newPath, newValue, proto.StringVal)
	var setReq = &gnmiutils.SetRequest{
		Ctx:         s.Context(),
		Client:      gnmiClient,
		Extensions:  s.SyncExtension(),
		Encoding:    gnmiapi.Encoding_PROTO,
		UpdatePaths: targetPath,
	}
	_, transactionIndex := setReq.SetOrFail(s.T())

	// Check that the values were set correctly
	var getConfigReq = &gnmiutils.GetRequest{
		Ctx:      s.Context(),
		Client:   gnmiClient,
		Encoding: gnmiapi.Encoding_PROTO,
		Paths:    targetPath,
	}
	getConfigReq.CheckValues(s.T(), newValue)

	// Check that the values are set on the target
	target1GnmiClient := s.NewSimulatorGNMIClientOrFail(s.simulator1.Name)
	var getTargetReq = &gnmiutils.GetRequest{
		Ctx:      s.Context(),
		Client:   target1GnmiClient,
		Encoding: gnmiapi.Encoding_JSON,
		Paths:    targetPath,
	}
	getTargetReq.CheckValues(s.T(), newValue)

	// Now rollback the change
	adminClient, err := s.NewAdminServiceClient()
	s.NoError(err)
	rollbackResponse, rollbackError := adminClient.RollbackTransaction(context.Background(), &admin.RollbackRequest{Index: transactionIndex})

	s.NoError(rollbackError, "Rollback returned an error")
	s.NotNil(rollbackResponse, "Response for rollback is nil")

	// Check that the value was really rolled back- should be an error here since the node was deleted
	_, err = getTargetReq.Get()
	s.Error(err)
}
