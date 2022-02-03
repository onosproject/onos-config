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

package gnmi

import (
	"bytes"
	"context"
	"fmt"
	topoapi "github.com/onosproject/onos-api/go/onos/topo"
	"github.com/onosproject/onos-config/pkg/store/configuration"
	"io/ioutil"
	"net/http"
	"os"
	"regexp"
	"strings"
	"time"

	"github.com/onosproject/onos-config/pkg/utils/tree"

	configapi "github.com/onosproject/onos-api/go/onos/config/v2"

	"github.com/onosproject/onos-lib-go/pkg/errors"

	"github.com/grpc-ecosystem/go-grpc-middleware/util/metautils"
	"github.com/onosproject/onos-config/pkg/utils"
	"github.com/openconfig/gnmi/proto/gnmi"
)

// OIDCServerURL - the ENV var that signified security is turned on - no groups will
// be extracted from request without this
const OIDCServerURL = "OIDC_SERVER_URL"

// Get implements gNMI Get
func (s *Server) Get(ctx context.Context, req *gnmi.GetRequest) (*gnmi.GetResponse, error) {
	notifications := make([]*gnmi.Notification, 0)
	groups := make([]string, 0)
	if md := metautils.ExtractIncoming(ctx); md != nil && md.Get("name") != "" {
		groups = append(groups, strings.Split(md.Get("groups"), ";")...)
		log.Debugf("gNMI Get() called by '%s (%s)'. Groups %v. Token %s",
			md.Get("name"), md.Get("email"), groups, md.Get("at_hash"))
	}
	if req == nil || (req.GetEncoding() != gnmi.Encoding_PROTO && req.GetEncoding() != gnmi.Encoding_JSON_IETF && req.GetEncoding() != gnmi.Encoding_JSON) {
		err := errors.NewInvalid("invalid encoding format in Get request. Only JSON_IETF and PROTO accepted. %v", req.Encoding)
		log.Warn(err)
		return nil, errors.Status(err).Err()
	}

	targetInfo := targetInfo{}
	prefix := req.GetPrefix()
	for _, path := range req.GetPath() {
		updates, err := s.getUpdate(ctx, targetInfo, prefix, path, req.GetEncoding(), groups)
		if err != nil {
			return nil, errors.Status(err).Err()
		}
		notification := &gnmi.Notification{
			Timestamp: time.Now().Unix(),
			Update:    updates,
			Prefix:    prefix,
		}

		notifications = append(notifications, notification)
	}
	// Alternatively - if there's only the prefix
	if len(req.GetPath()) == 0 {
		updates, err := s.getUpdate(ctx, targetInfo, prefix, nil, req.GetEncoding(), groups)
		if err != nil {
			return nil, errors.Status(err).Err()
		}
		notification := &gnmi.Notification{
			Timestamp: time.Now().Unix(),
			Update:    updates,
			Prefix:    prefix,
		}

		notifications = append(notifications, notification)
	}

	response := gnmi.GetResponse{
		Notification: notifications,
	}
	return &response, nil
}

// getUpdate utility method for getting an Update for a given path
func (s *Server) getUpdate(ctx context.Context, targetInfo targetInfo, prefix *gnmi.Path, path *gnmi.Path,
	encoding gnmi.Encoding, groups []string) ([]*gnmi.Update, error) {
	if (path == nil || path.Target == "") && (prefix == nil || prefix.Target == "") {
		return nil, errors.NewInvalid("invalid request - Path %s has no target", utils.StrPath(path))
	}

	// If a target exists on the path, use it. If not use target of Prefix
	targetID := configapi.TargetID(path.GetTarget())
	if targetID == "" {
		targetInfo.targetID = configapi.TargetID(prefix.Target)
	} else {
		targetInfo.targetID = targetID
	}

	pathAsString := utils.StrPath(path)
	if prefix != nil && prefix.Elem != nil {
		pathAsString = utils.StrPath(prefix) + pathAsString
	}
	pathAsString = strings.TrimSuffix(pathAsString, "/")

	// TODO: Add target type and version to configuration ID
	targetConfig, err := s.configurations.Get(ctx, configuration.NewID(targetInfo.targetID, "", ""))
	if err != nil {
		return nil, err
	}

	var configValues []*configapi.PathValue
	for _, configValue := range targetConfig.Values {
		if configValue.Path == pathAsString {
			configValues = append(configValues, configValue)
		}
	}

	var configValuesAllowed []*configapi.PathValue
	// Filter config values using open policy agent
	if len(os.Getenv(OIDCServerURL)) > 0 {
		configValuesAllowed, err = s.checkOpaAllowed(ctx, targetInfo, configValues, groups)
		if err != nil {
			return nil, err
		}
	} else {
		configValuesAllowed = make([]*configapi.PathValue, len(configValues))
		copy(configValuesAllowed, configValues)
	}

	return createUpdate(prefix, path, configValuesAllowed, encoding)
}

func filterTargetForURL(target string) string {
	re := regexp.MustCompile(`[.-]`)
	return re.ReplaceAllString(target, "_")
}

func (s *Server) checkOpaAllowed(ctx context.Context, targetInfo targetInfo, configValues []*configapi.PathValue, groups []string) ([]*configapi.PathValue, error) {
	modelPlugin, err := s.getModelPlugin(ctx, topoapi.ID(targetInfo.targetID))
	if err != nil {
		log.Warn(err)
		return nil, err
	}

	targetVersion := filterTargetForURL(modelPlugin.Info.Version)
	targetType := filterTargetForURL(modelPlugin.Info.Name)

	jsonTree, err := tree.BuildTree(configValues, true)
	if err != nil {
		return nil, err
	}
	// add 'input' and `groups` objects to the JSON
	jsonTreeInput := utils.FormatInput(jsonTree, groups)
	log.Debugf("OPA Input:\n%s", jsonTreeInput)

	client := &http.Client{}
	// POST to OPA sidecar
	opaURL := fmt.Sprintf("http://localhost:%d/v1/data/%s_%s/allowed?pretty=%v&metrics=%v", 8181,
		targetType, targetVersion, false, true)

	log.Debugf("OPA URL is %s", opaURL)
	resp, err := client.Post(opaURL, "application/json", bytes.NewBuffer([]byte(jsonTreeInput)))
	if err != nil {
		log.Warnf("Error sending request to OPA sidecar %s %s", opaURL, err.Error())
		return nil, err
	}
	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	bodyText, err := utils.FormatOutput(body)
	if err != nil {
		return nil, err
	}
	if bodyText == "" {
		return nil, nil
	}

	log.Debugf("body text of response from OPA:\n%s", bodyText)
	return modelPlugin.GetPathValues(ctx, "", []byte(bodyText))
}
