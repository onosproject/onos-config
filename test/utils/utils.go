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

package utils

import (
	"context"
	"github.com/onosproject/onos-config/api/diags"
	"github.com/onosproject/onos-config/api/types/change"
	"github.com/onosproject/onos-config/api/types/change/network"
	"github.com/onosproject/onos-test/pkg/onit/env"
	"github.com/onosproject/onos-topo/api/device"
	"github.com/stretchr/testify/assert"
	"io"
	"testing"
	"time"
)

// MakeContext returns a new context for use in GNMI requests
func MakeContext() context.Context {
	// TODO: Investigate using context.WithCancel() here
	ctx := context.Background()
	return ctx
}

// WaitForDevice waits for a device to match the given predicate
func WaitForDevice(t *testing.T, predicate func(*device.Device) bool, timeout time.Duration) bool {
	client, err := env.Topo().NewDeviceServiceClient()
	assert.NoError(t, err)
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	stream, err := client.List(ctx, &device.ListRequest{
		Subscribe: true,
	})
	assert.NoError(t, err)
	for {
		response, err := stream.Recv()
		if err == io.EOF {
			assert.Fail(t, "device stream closed prematurely")
			return false
		} else if err != nil {
			assert.Fail(t, "device stream failed with error: %v", err)
			return false
		} else if predicate(response.Device) {
			return true
		}
	}
}

// WaitForDeviceAvailable waits for a device to become available
func WaitForDeviceAvailable(t *testing.T, deviceID device.ID, timeout time.Duration) bool {
	return WaitForDevice(t, func(dev *device.Device) bool {
		if dev.ID != deviceID {
			return false
		}

		for _, protocol := range dev.Protocols {
			if protocol.Protocol == device.Protocol_GNMI && protocol.ServiceState == device.ServiceState_AVAILABLE {
				return true
			}
		}
		return false
	}, timeout)
}

// WaitForDeviceUnavailable waits for a device to become available
func WaitForDeviceUnavailable(t *testing.T, deviceID device.ID, timeout time.Duration) bool {
	return WaitForDevice(t, func(dev *device.Device) bool {
		if dev.ID != deviceID {
			return false
		}

		for _, protocol := range dev.Protocols {
			if protocol.Protocol == device.Protocol_GNMI && protocol.ServiceState == device.ServiceState_UNAVAILABLE {
				return true
			}
		}
		return false
	}, timeout)
}

// WaitForNetworkChangeComplete waits for a COMPLETED status on the given change
func WaitForNetworkChangeComplete(t *testing.T, networkChangeID network.ID, wait time.Duration) bool {
	listNetworkChangeRequest := &diags.ListNetworkChangeRequest{
		Subscribe:     true,
		ChangeID:      networkChangeID,
		WithoutReplay: false,
	}

	changeServiceClient, changeServiceClientErr := env.Config().NewChangeServiceClient()
	assert.Nil(t, changeServiceClientErr)
	assert.True(t, changeServiceClient != nil)

	ctx, cancel := context.WithTimeout(context.Background(), wait)
	defer cancel()

	listNetworkChangesClient, listNetworkChangesClientErr := changeServiceClient.ListNetworkChanges(ctx, listNetworkChangeRequest)
	assert.Nil(t, listNetworkChangesClientErr)
	assert.True(t, listNetworkChangesClient != nil)

	for {
		// Check if the network change has completed
		networkChangeResponse, networkChangeResponseErr := listNetworkChangesClient.Recv()
		if networkChangeResponseErr == io.EOF {
			assert.Fail(t, "change stream closed prematurely")
			return false
		} else if networkChangeResponseErr != nil {
			assert.Fail(t, "change stream failed with error: %v", networkChangeResponseErr)
			return false
		} else {
			assert.True(t, networkChangeResponse != nil)
			if change.State_COMPLETE == networkChangeResponse.Change.Status.State {
				return true
			}
		}
	}
}
