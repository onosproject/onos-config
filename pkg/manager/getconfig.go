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

package manager

import (
	"bytes"
	"fmt"
	devicechange "github.com/onosproject/onos-api/go/onos/config/change/device"
	networkchange "github.com/onosproject/onos-api/go/onos/config/change/network"
	devicetype "github.com/onosproject/onos-api/go/onos/config/device"
	"github.com/onosproject/onos-config/pkg/modelregistry/jsonvalues"
	"github.com/onosproject/onos-config/pkg/store"
	"github.com/onosproject/onos-config/pkg/utils"
	"github.com/onosproject/onos-lib-go/pkg/errors"
	"io/ioutil"
	"net/http"
	"os"
	"strings"
)

// OIDCServerURL - the ENV var that signified security is turned on - no groups will
// be extracted from request without this
const OIDCServerURL = "OIDC_SERVER_URL"

// GetTargetConfig returns a set of change values given a target, a configuration name, a path and a layer.
// The layer is the numbers of config changes we want to go back in time for. 0 is the latest (Atomix based)
func (m *Manager) GetTargetConfig(deviceID devicetype.ID, version devicetype.Version, deviceType devicetype.Type,
	path string, revision networkchange.Revision, groups []string) ([]*devicechange.PathValue, error) {
	configValues, errGetTargetCfg := m.DeviceStateStore.Get(devicetype.NewVersionedID(deviceID, version), revision)
	if errGetTargetCfg != nil {
		log.Error("Error while extracting config", errGetTargetCfg)
		return nil, errGetTargetCfg
	}
	if len(configValues) == 0 {
		return configValues, nil
	}
	var configValuesAllowed []*devicechange.PathValue
	var err error
	if len(os.Getenv(OIDCServerURL)) > 0 {
		configValuesAllowed, err = m.checkOpaAllowed(version, deviceType, configValues, groups)
		if err != nil {
			return nil, err
		}
	} else {
		configValuesAllowed = make([]*devicechange.PathValue, len(configValues))
		copy(configValuesAllowed, configValues)
	}

	filteredValues := make([]*devicechange.PathValue, 0)
	pathRegexp := utils.MatchWildcardRegexp(path, false)
	for _, cv := range configValuesAllowed {
		if pathRegexp.MatchString(cv.Path) {
			filteredValues = append(filteredValues, cv)
		}
	}
	//TODO if filteredValue is empty return error
	return filteredValues, nil
}

// GetAllDeviceIds returns a list of just DeviceIDs from the device cache
func (m *Manager) GetAllDeviceIds() *[]string {

	var deviceIds = make([]string, 0)
	for _, dev := range m.DeviceCache.GetDevices() {
		deviceIds = append(deviceIds, string(dev.DeviceID))
	}

	return &deviceIds
}

// checkOpaAllowed - call the OPA sidecar with the configuration to filter only
// those items allowed for an enterprise
func (m *Manager) checkOpaAllowed(version devicetype.Version, deviceType devicetype.Type,
	configValues []*devicechange.PathValue, groups []string) ([]*devicechange.PathValue, error) {
	log.Infof("Querying OPA sidecar for allowed configuration for %s:%s", deviceType, version)

	jsonTree, err := store.BuildTree(configValues, true)
	if err != nil {
		log.Error("Error building JSON tree from Config Values ", err, jsonTree)
		return nil, err
	}
	// add 'input' and `groups` objects to the JSON
	jsonTreeInput := utils.FormatInput(jsonTree, groups)
	log.Debugf("OPA Input:\n%s", jsonTreeInput)
	client := &http.Client{}
	// POST to OPA sidecar
	opaURL := fmt.Sprintf("http://localhost:%d/v1/data/%s_%s/allowed?pretty=%v&metrics=%v", 8181,
		strings.ToLower(string(deviceType)), strings.ReplaceAll(string(version), ".", "_"), false, true)
	resp, err := client.Post(opaURL, "application/json", bytes.NewBuffer([]byte(jsonTreeInput)))
	if err != nil {
		log.Errorf("Error sending request to OPA sidecar %s %s", opaURL, err.Error())
		return nil, err
	}
	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Error("Error reading response from OPA sidecar %s", err.Error())
		return nil, err
	}

	bodyText, err := utils.FormatOutput(body)
	if err != nil {
		return nil, err
	}
	if bodyText == "" {
		return nil, nil
	}

	// Have to have the model plugin to determine the types of each attribute when unmarshalling JSON
	modelName := utils.ToModelName(deviceType, version)
	deviceModelYgotPlugin, err := m.ModelRegistry.GetPlugin(modelName)
	if err != nil {
		if errors.IsNotFound(err) {
			log.Warn("No model ", modelName, " available as a plugin")
			if !mgr.allowUnvalidatedConfig {
				return nil, errors.NewNotFound("no model %s available as a plugin", modelName)
			}
			return nil, errors.NewNotSupported("allowUnvalidatedConfig is not supported")
		}
		return nil, err
	}
	// Unmarshal JSON in to PathValues
	return jsonvalues.DecomposeJSONWithPaths("", []byte(bodyText),
		deviceModelYgotPlugin.ReadOnlyPaths, deviceModelYgotPlugin.ReadWritePaths)
}
