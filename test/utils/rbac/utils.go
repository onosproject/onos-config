// SPDX-FileCopyrightText: 2020-present Open Networking Foundation <info@opennetworking.org>
//
// SPDX-License-Identifier: Apache-2.0

package rbac

import (
	"context"
	"encoding/json"
	"github.com/onosproject/onos-lib-go/pkg/errors"
	"github.com/onosproject/onos-lib-go/pkg/logging"
	"golang.org/x/oauth2"
	"google.golang.org/grpc/metadata"
	"net/http"
	"net/url"
	"strings"
)

var log = logging.GetLogger()

// FetchATokenViaKeyCloak Get the token via keycloak using curl
func FetchATokenViaKeyCloak(openIDIssuer string, user string, passwd string) (string, error) {

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

// GetBearerContext gets a context that usesthe given token as the Bearer in the Authorization header
func GetBearerContext(ctx context.Context, token string) context.Context {
	const (
		authorization = "Authorization"
	)
	token = "Bearer " + token
	md := make(metadata.MD)
	md.Set(authorization, token)
	ctx = metadata.NewOutgoingContext(ctx, md)
	return ctx
}
