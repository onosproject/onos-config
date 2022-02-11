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
	"io/ioutil"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"

	topoapi "github.com/onosproject/onos-api/go/onos/topo"
	"github.com/onosproject/onos-config/pkg/store/configuration"

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
	log.Infof("Received gNMI Get Request: %+v", req)
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

	transactionStrategy, err := getExtensions(req)
	if err != nil {
		log.Warn(err)
		return nil, errors.Status(err).Err()
	}

	prefix := req.GetPrefix()
	targets := make(map[configapi.TargetID]*targetInfo)
	var paths []*pathInfo

	// Get configuration for each target and forms targets info map
	// and process paths in the request and forms a map of paths info
	for _, path := range req.GetPath() {
		targetID := configapi.TargetID(path.Target)
		if targetID == "" && prefix != nil {
			targetID = configapi.TargetID(prefix.Target)
		}
		if targetID == "" {
			return nil, errors.Status(errors.NewInvalid("has no target")).Err()
		}

		if _, ok := targets[targetID]; !ok {
			err := s.addTarget(ctx, targetID, targets)
			if err != nil {
				log.Warn(err)
				return nil, errors.Status(err).Err()

			}
		}
		paths = append(paths, &pathInfo{
			targetID: targetID,
			path:     path,
		})
	}

	// if there's only the prefix
	if len(req.GetPath()) == 0 && prefix != nil {
		targetID := configapi.TargetID(prefix.Target)
		if targetID == "" {
			return nil, errors.Status(errors.NewInvalid("has no target")).Err()
		}
		if _, ok := targets[targetID]; !ok {
			err := s.addTarget(ctx, targetID, targets)
			if err != nil {
				return nil, errors.Status(errors.NewInvalid(err.Error())).Err()
			}
		}

		updates, err := s.getUpdate(ctx, targets[targetID], prefix, nil, req.GetEncoding(), groups)
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

	for _, pathInfo := range paths {
		if targetInfo, ok := targets[pathInfo.targetID]; ok {
			updates, err := s.getUpdate(ctx, targetInfo, prefix, pathInfo.path, req.GetEncoding(), groups)
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
	}

	switch transactionStrategy.Synchronicity {
	case configapi.TransactionStrategy_SYNCHRONOUS:
		log.Debugf("Processing synchronous get request %+v", req)
		wg := &sync.WaitGroup{}
		for _, target := range targets {
			if target.configuration.Status.Applied.Index != target.configuration.Status.Committed.Index {
				ch := make(chan configapi.ConfigurationEvent)
				err = s.configurations.Watch(ctx, ch, configuration.WithConfigurationID(configuration.NewID(target.targetID)), configuration.WithReplay())
				if err != nil {
					return nil, errors.Status(err).Err()
				}
				wg.Add(1)
				go func(id configapi.ConfigurationID) {
					for event := range ch {
						if event.Configuration.Status.Applied.Index >= target.configuration.Status.Committed.Index &&
							event.Configuration.Status.Applied.Term == event.Configuration.Status.Term {
							wg.Done()
							return
						}
					}
				}(target.configuration.ID)
			}
		}
		wg.Wait()
	}

	response := gnmi.GetResponse{
		Notification: notifications,
	}
	return &response, nil
}

func (s *Server) addTarget(ctx context.Context, targetID configapi.TargetID, targets map[configapi.TargetID]*targetInfo) error {
	modelPlugin, err := s.getModelPlugin(ctx, topoapi.ID(targetID))
	if err != nil {
		log.Warn(err)
		return err
	}
	targetInfo := &targetInfo{
		targetID:      targetID,
		targetVersion: configapi.TargetVersion(modelPlugin.GetInfo().Info.Version),
		targetType:    configapi.TargetType(modelPlugin.GetInfo().Info.Name),
	}

	targetConfig, err := s.configurations.Get(ctx, configuration.NewID(targetInfo.targetID))
	if err != nil {
		return err
	}
	targetInfo.configuration = targetConfig
	targets[targetID] = targetInfo
	return nil
}

// getUpdate utility method for getting an Update for a given path
func (s *Server) getUpdate(ctx context.Context, targetInfo *targetInfo, prefix *gnmi.Path, path *gnmi.Path,
	encoding gnmi.Encoding, groups []string) ([]*gnmi.Update, error) {
	if (path == nil || path.Target == "") && (prefix == nil || prefix.Target == "") {
		return nil, errors.NewInvalid("invalid request - Path %s has no target", utils.StrPath(path))
	}

	pathAsString := utils.StrPath(path)
	if prefix != nil && prefix.Elem != nil {
		pathAsString = utils.StrPath(prefix) + pathAsString
	}
	pathAsString = strings.TrimSuffix(pathAsString, "/")
	targetConfig := targetInfo.configuration

	var configValues []*configapi.PathValue
	for _, configValue := range targetConfig.Values {
		configValues = append(configValues, configValue)
	}

	var configValuesAllowed []*configapi.PathValue
	var err error
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

	filteredValues := make([]*configapi.PathValue, 0)
	pathRegexp := utils.MatchWildcardRegexp(pathAsString, false)
	for _, cv := range configValuesAllowed {
		if pathRegexp.MatchString(cv.Path) && !cv.Deleted {
			filteredValues = append(filteredValues, cv)
		}
	}

	return createUpdate(prefix, path, filteredValues, encoding)
}

func (s *Server) checkOpaAllowed(ctx context.Context, targetInfo *targetInfo, configValues []*configapi.PathValue, groups []string) ([]*configapi.PathValue, error) {
	targetVersion := filterTargetForURL(string(targetInfo.targetVersion))
	targetType := filterTargetForURL(string(targetInfo.targetType))

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
	modelPlugin, err := s.getModelPlugin(ctx, topoapi.ID(targetInfo.targetID))
	if err != nil {
		log.Warn(err)
		return nil, err
	}
	return modelPlugin.GetPathValues(ctx, "", []byte(bodyText))
}
