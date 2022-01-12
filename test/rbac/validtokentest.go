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

package rbac

import (
	"context"
	"encoding/json"
	"github.com/onosproject/onos-lib-go/pkg/errors"
	gpb "github.com/openconfig/gnmi/proto/gnmi"
	"github.com/prometheus/common/log"
	"github.com/stretchr/testify/assert"
	"golang.org/x/oauth2"
	"google.golang.org/grpc/metadata"
	"net/http"
	"net/url"
	"strings"
	"testing"

	"github.com/onosproject/onos-config/test/utils/gnmi"
	"github.com/onosproject/onos-config/test/utils/proto"
)

// fetchATokenViaKeyCloak Get the token via keycloak using curl
func fetchATokenViaKeyCloak(openIDIssuer string, user string, passwd string) (string, error) {

	data := url.Values{}
	data.Set("username", user)
	data.Set("password", passwd)
	data.Set("grant_type", "password")
	data.Set("client_id", "onos-config-test")
	data.Set("scope", "openid profile email groups")

	req, err := http.NewRequest("POST", openIDIssuer+"/protocol/openid-connect/token", strings.NewReader(data.Encode()))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", err
	}

	log.Debug("Response Code : ", resp.StatusCode)

	defer resp.Body.Close()
	if resp.StatusCode == http.StatusOK {
		target := new(oauth2.Token)
		err = json.NewDecoder(resp.Body).Decode(target)
		if err != nil {
			return "", err
		}
		return target.AccessToken, nil
	}

	return "", errors.NewInvalid("Error HTTP response code : ", resp.StatusCode)

}

func getContext(ctx context.Context, token string) context.Context {
	const (
		authorization = "Authorization"
	)
	token = "Bearer " + token
	md := make(metadata.MD)
	md.Set(authorization, token)
	ctx = metadata.NewOutgoingContext(ctx, md)
	return ctx
}

// TestValidToken tests access to a protected API with a valid token supplied
func (s *TestSuite) TestValidToken(t *testing.T) {
	const (
		tzValue = "Europe/Dublin"
		tzPath  = "/system/clock/config/timezone-name"
	)
	// Create a simulated device
	simulator := gnmi.CreateSimulator(t)
	defer gnmi.DeleteSimulator(t, simulator)

	// get an access token
	token, err := fetchATokenViaKeyCloak("https://keycloak-dev.onlab.us/auth/realms/master", "alicea", "KDVMw3xvk2mu")
	assert.NoError(t, err)
	assert.NotNil(t, token)

	// Make a GNMI client to use for requests
	ctx := getContext(context.Background(), token)
	gnmiClient := gnmi.GetGNMIClientWithContextOrFail(ctx, t)

	// Try to fetch a value from the GNMI client
	devicePath := gnmi.GetDevicePathWithValue(simulator.Name(), tzPath, tzValue, proto.StringVal)
	_, _, err = gnmi.GetGNMIValue(ctx, gnmiClient, devicePath, gpb.Encoding_PROTO)

	assert.NoError(t, err)
}
