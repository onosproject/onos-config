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

// Package synchronizer synchronizes configurations down to devices
package synchronizer

import (
	"context"
	"fmt"
	"github.com/golang/protobuf/proto"
	"github.com/onosproject/onos-config/pkg/events"
	"github.com/onosproject/onos-config/pkg/modelregistry"
	"github.com/onosproject/onos-config/pkg/modelregistry/jsonvalues"
	"github.com/onosproject/onos-config/pkg/southbound"
	"github.com/onosproject/onos-config/pkg/store"
	"github.com/onosproject/onos-config/pkg/store/change"
	devicechangetypes "github.com/onosproject/onos-config/pkg/types/change/device"
	"github.com/onosproject/onos-config/pkg/utils"
	"github.com/onosproject/onos-config/pkg/utils/values"
	devicetopo "github.com/onosproject/onos-topo/pkg/northbound/device"
	"github.com/openconfig/gnmi/client"
	"github.com/openconfig/gnmi/proto/gnmi"
	"google.golang.org/grpc/status"
	log "k8s.io/klog"
	"regexp"
	"strings"
)

const matchOnIndex = `(\=.*?]).*?`

// Synchronizer enables proper configuring of a device based on store events and cache of operational data
type Synchronizer struct {
	context.Context
	*store.ChangeStore
	*store.ConfigurationStore
	*devicetopo.Device
	deviceConfigChan     <-chan events.ConfigEvent
	operationalStateChan chan<- events.OperationalStateEvent
	key                  southbound.DeviceID
	query                client.Query
	modelReadOnlyPaths   modelregistry.ReadOnlyPathMap
	operationalCache     devicechangetypes.TypedValueMap
	encoding             gnmi.Encoding
	getStateMode         modelregistry.GetStateMode
}

// New Build a new Synchronizer given the parameters, starts the connection with the device and polls the capabilities
func New(context context.Context, changeStore *store.ChangeStore, configStore *store.ConfigurationStore,
	device *devicetopo.Device, deviceCfgChan <-chan events.ConfigEvent, opStateChan chan<- events.OperationalStateEvent,
	errChan chan<- events.DeviceResponse, opStateCache devicechangetypes.TypedValueMap,
	mReadOnlyPaths modelregistry.ReadOnlyPathMap, target southbound.TargetIf, getStateMode modelregistry.GetStateMode) (*Synchronizer, error) {
	sync := &Synchronizer{
		Context:              context,
		ChangeStore:          changeStore,
		ConfigurationStore:   configStore,
		Device:               device,
		deviceConfigChan:     deviceCfgChan,
		operationalStateChan: opStateChan,
		operationalCache:     opStateCache,
		modelReadOnlyPaths:   mReadOnlyPaths,
		getStateMode:         getStateMode,
	}
	log.Info("Connecting to ", sync.Device.Address, " over gNMI")
	key, err := target.ConnectTarget(context, *sync.Device)
	sync.key = key
	if err != nil {
		log.Warning(err)
		return nil, err
	}
	log.Info(sync.Device.Address, " connected over gNMI")

	// Get the device capabilities
	capResponse, capErr := target.CapabilitiesWithString(context, "")
	if capErr != nil {
		log.Error(sync.Device.Address, " capabilities: ", capErr)
		errChan <- events.NewErrorEventNoChangeID(events.EventTypeErrorDeviceCapabilities,
			string(device.ID), capErr)
		return nil, capErr
	}
	sync.encoding = gnmi.Encoding_PROTO // Default
	if capResponse != nil {
		for _, enc := range capResponse.SupportedEncodings {
			if enc == gnmi.Encoding_PROTO {
				sync.encoding = enc
				break // We prefer PROTO if possible
			}
			sync.encoding = enc // Will take alternatives or last
		}
	}
	log.Info(sync.Device.Address, " Encoding:", sync.encoding, " Capabilities ", capResponse)

	config, err := getNetworkConfig(sync, string(sync.Device.ID), "", 0)

	//Device does not have any stored config at the moment, skip initial set
	if err != nil {
		log.Info(sync.Device.Address, " has no initial configuration")
	} else {
		//Device has initial configuration saved in onos-config, trying to apply
		initialConfig, err := change.NewChangeValuesNoRemoval(config, "Initial set to device")

		if err != nil {
			log.Errorf("Can't translate the initial config for %s due to: %s", sync.Device.Address, err)
			return sync, nil
		}

		gnmiChange, err := values.NativeChangeToGnmiChange(initialConfig)

		if err != nil {
			log.Errorf("Can't obtain GnmiChange for %s due to: %s", sync.Device.Address, err)
			return sync, nil
		}

		resp, err := target.Set(context, gnmiChange)

		if err != nil {
			errGnmi, _ := status.FromError(err)
			//Hack because the desc field is not available.
			//Splitting at the desc string and getting the second element which is the description.
			log.Errorf("Can't set initial configuration for %s due to: %s", sync.Device.Address,
				strings.Split(errGnmi.Message(), " desc = ")[1])
			errChan <- events.NewErrorEvent(events.EventTypeErrorSetInitialConfig,
				string(device.ID), initialConfig.ID, err)
		} else {
			log.Infof("Loaded initial config %s for device %s", store.B64(initialConfig.ID), sync.key.DeviceID)
			errChan <- events.NewResponseEvent(events.EventTypeAchievedSetConfig,
				sync.key.DeviceID, initialConfig.ID, resp.String())
		}
	}

	return sync, nil
}

// syncConfigEventsToDevice is a go routine that listens out for configuration events specific
// to a device and propagates them downwards through southbound interface
func (sync *Synchronizer) syncConfigEventsToDevice(target southbound.TargetIf, respChan chan<- events.DeviceResponse) {

	for deviceConfigEvent := range sync.deviceConfigChan {
		c := sync.ChangeStore.Store[deviceConfigEvent.ChangeID()]
		err := c.IsValid()
		if err != nil {
			log.Warning("Event discarded because change is invalid: ", err)
			respChan <- events.NewErrorEvent(events.EventTypeErrorParseConfig,
				sync.key.DeviceID, c.ID, err)
			continue
		}
		gnmiChange, parseError := values.NativeChangeToGnmiChange(c)

		if parseError != nil {
			log.Error("Parsing error for gNMI change: ", parseError)
			respChan <- events.NewErrorEvent(events.EventTypeErrorParseConfig,
				sync.key.DeviceID, c.ID, err)
			continue
		}

		log.Info("Change formatted to gNMI setRequest ", gnmiChange)
		setResponse, err := target.Set(sync.Context, gnmiChange)
		if err != nil {
			log.Error("Error while doing set: ", err)
			respChan <- events.NewErrorEvent(events.EventTypeErrorSetConfig,
				sync.key.DeviceID, c.ID, err)
			continue
		}
		log.Info(sync.Device.Address, " SetResponse ", setResponse)
		respChan <- events.NewResponseEvent(events.EventTypeAchievedSetConfig,
			sync.key.DeviceID, c.ID, setResponse.String())

	}
}

// For use when device model has modelregistry.GetStateOpState
func (sync Synchronizer) syncOperationalStateByPartition(ctx context.Context, target southbound.TargetIf,
	errChan chan<- events.DeviceResponse) {

	log.Infof("Syncing Op & State of %s started. Mode %v", sync.key.DeviceID, sync.getStateMode)
	notifications := make([]*gnmi.Notification, 0)
	stateNotif, errState := sync.getOpStatePathsByType(ctx, target, gnmi.GetRequest_STATE)
	if errState != nil {
		log.Warning("Can't request read-only state paths to target ", sync.key, errState)
	} else {
		notifications = append(notifications, stateNotif...)
	}

	operNotif, errOp := sync.getOpStatePathsByType(ctx, target, gnmi.GetRequest_OPERATIONAL)
	if errState != nil {
		log.Warning("Can't request read-only operational paths to target ", sync.key, errOp)
	} else {
		notifications = append(notifications, operNotif...)
	}

	sync.opCacheUpdate(notifications, errChan)

	// Now try the subscribe with the read only paths and the expanded wildcard
	// paths (if any) from above
	sync.subscribeOpState(target, errChan)
}

// For use when device model has
// * modelregistry.GetStateExplicitRoPathsExpandWildcards (like Stratum) or
// * modelregistry.GetStateExplicitRoPaths
func (sync Synchronizer) syncOperationalStateByPaths(ctx context.Context, target southbound.TargetIf,
	errChan chan<- events.DeviceResponse) {

	log.Infof("Syncing Op & State of %s started. Mode %v", sync.key.DeviceID, sync.getStateMode)
	if sync.modelReadOnlyPaths == nil {
		errMp := fmt.Errorf("no model plugin, cant work in operational state cache")
		log.Error(errMp)
		errChan <- events.NewErrorEventNoChangeID(events.EventTypeErrorMissingModelPlugin,
			sync.key.DeviceID, errMp)
		return
	} else if len(sync.modelReadOnlyPaths) == 0 {
		noPathErr := fmt.Errorf("target %#v has no paths to subscribe to", sync.ID)
		errChan <- events.NewErrorEventNoChangeID(events.EventTypeErrorSubscribe,
			sync.key.DeviceID, noPathErr)
		log.Warning(noPathErr)
		return
	}
	log.Infof("Getting state by %d ReadOnly paths for %s", len(sync.modelReadOnlyPaths), sync.key.DeviceID)
	getPaths := make([]*gnmi.Path, 0)
	for _, path := range sync.modelReadOnlyPaths.JustPaths() {
		if sync.getStateMode == modelregistry.GetStateExplicitRoPathsExpandWildcards &&
			strings.Contains(path, "*") {
			// Don't add in wildcards here - they will be expanded later
			continue
		}
		gnmiPath, err := utils.ParseGNMIElements(utils.SplitPath(path))
		if err != nil {
			log.Warning("Error converting RO path to gNMI")
			errChan <- events.NewErrorEventNoChangeID(events.EventTypeErrorTranslation,
				sync.key.DeviceID, err)
			return
		}
		getPaths = append(getPaths, gnmiPath)
	}

	if sync.getStateMode == modelregistry.GetStateExplicitRoPathsExpandWildcards {
		ewStringPaths := make(map[string]interface{})
		ewGetPaths := make([]*gnmi.Path, 0)
		for roPath := range sync.modelReadOnlyPaths {
			// Some devices e.g. Stratum does not fully support wild-carded Gets
			// instead this allows a wildcarded Get of a state container
			// e.g. /interfaces/interface[name=*]/state
			// and from the response a concrete set of instance names can be
			// retrieved which can then be used in the OpState get
			// These are called Expanded Wildcards
			if strings.Contains(roPath, "*") {
				ewPath, err := utils.ParseGNMIElements(utils.SplitPath(roPath))
				if err != nil {
					log.Warningf("Unable to parse %s", roPath)
					continue
				}
				ewStringPaths[roPath] = nil // Just holding the keys
				ewGetPaths = append(ewGetPaths, ewPath)
			}
		}
		requestEwRoPaths := &gnmi.GetRequest{
			Encoding: sync.encoding,
			Path:     ewGetPaths,
		}

		log.Infof("Calling Get again for %s with expanded %d wildcard read-only paths", sync.key, len(ewGetPaths))
		if len(ewGetPaths) > 0 {
			responseEwRoPaths, errRoPaths := target.Get(ctx, requestEwRoPaths)
			if errRoPaths != nil {
				log.Warning("Error on request for expanded wildcard read-only paths", sync.key, errRoPaths)
				errChan <- events.NewErrorEventNoChangeID(events.EventTypeErrorGetWithRoPaths,
					sync.key.DeviceID, errRoPaths)
				return
			}
			for _, n := range responseEwRoPaths.Notification {
				for _, u := range n.Update {
					if sync.encoding == gnmi.Encoding_JSON || sync.encoding == gnmi.Encoding_JSON_IETF {
						configValues, err := sync.getValuesFromJSON(u)
						if err != nil {
							errChan <- events.NewErrorEventNoChangeID(events.EventTypeErrorTranslation,
								sync.key.DeviceID, err)
							continue
						}
						for _, cv := range configValues {
							matched, err := pathMatchesWildcard(ewStringPaths, cv.Path)
							if err != nil {
								errChan <- events.NewErrorEventNoChangeID(events.EventTypeErrorTranslation,
									sync.key.DeviceID, err)
								continue
							}
							p, err := utils.ParseGNMIElements(utils.SplitPath(matched))
							if err != nil {
								errChan <- events.NewErrorEventNoChangeID(events.EventTypeErrorTranslation,
									sync.key.DeviceID, err)
								continue
							}
							getPaths = append(getPaths, p)
						}
					} else {
						matched, err := pathMatchesWildcard(ewStringPaths, utils.StrPath(u.Path))
						if err != nil {
							errChan <- events.NewErrorEventNoChangeID(events.EventTypeErrorTranslation,
								sync.key.DeviceID, err)
							continue
						}
						matchedAsPath, err := utils.ParseGNMIElements(utils.SplitPath(matched))
						if err != nil {
							errChan <- events.NewErrorEventNoChangeID(events.EventTypeErrorTranslation,
								sync.key.DeviceID, err)
							continue
						}
						getPaths = append(getPaths, matchedAsPath)
					}
				}
			}
		}
	}

	requestRoPaths := &gnmi.GetRequest{
		Encoding: sync.encoding,
		Path:     getPaths,
	}

	responseRoPaths, errRoPaths := target.Get(ctx, requestRoPaths)
	if errRoPaths != nil {
		log.Warning("Error on request for read-only paths", sync.key, errRoPaths)
		errChan <- events.NewErrorEventNoChangeID(events.EventTypeErrorGetWithRoPaths,
			sync.key.DeviceID, errRoPaths)
		return
	}
	sync.opCacheUpdate(responseRoPaths.Notification, errChan)

	// Now try the subscribe with the read only paths and the expanded wildcard
	// paths (if any) from above
	sync.subscribeOpState(target, errChan)
}

/**
 * Process the returned path
 * Request might have been /interfaces/interface[name=*]/state
 * Result might be like /interfaces/interface[name=s1-eth2]/state/ifindex
 * Have to cater for many scenarios
 */
func pathMatchesWildcard(wildcards map[string]interface{}, path string) (string, error) {
	if len(wildcards) == 0 || path == "" {
		return "", fmt.Errorf("empty")
	}
	rOnIndex := regexp.MustCompile(matchOnIndex)

	idxMatches := rOnIndex.FindAllStringSubmatch(path, -1)
	pathWildIndex := path
	for _, m := range idxMatches {
		pathWildIndex = strings.Replace(pathWildIndex, m[1], "=*]", 1)
	}
	_, exactMatch := wildcards[pathWildIndex]
	if exactMatch {
		return path, nil
	}
	// Else iterate through paths for see if any match
	for key := range wildcards {
		if strings.HasPrefix(pathWildIndex, key) {
			remainder := pathWildIndex[len(key):]
			return path[:len(path)-len(remainder)], nil
		}
	}

	return "", fmt.Errorf("no match for %s", path)
}

func (sync Synchronizer) opCacheUpdate(notifications []*gnmi.Notification,
	errChan chan<- events.DeviceResponse) {

	log.Infof("Handling %d received OpState paths. %s", len(notifications), sync.key.DeviceID)
	for _, notification := range notifications {
		for _, update := range notification.Update {
			if sync.encoding == gnmi.Encoding_JSON || sync.encoding == gnmi.Encoding_JSON_IETF {
				configValues, err := sync.getValuesFromJSON(update)
				if err != nil {
					errChan <- events.NewErrorEventNoChangeID(events.EventTypeErrorTranslation,
						sync.key.DeviceID, err)
					continue
				}
				for _, cv := range configValues {
					value := cv.GetValue()
					sync.operationalCache[cv.Path] = value
				}
			} else if sync.encoding == gnmi.Encoding_PROTO {
				typedVal, err := values.GnmiTypedValueToNativeType(update.Val)
				if err != nil {
					log.Warning("Error converting gnmi value to Typed"+
						" Value", update.Val, " for ", update.Path)
				} else {
					sync.operationalCache[utils.StrPath(update.Path)] = typedVal
				}
			}
		}
	}
}

func (sync Synchronizer) getValuesFromJSON(update *gnmi.Update) ([]*devicechangetypes.PathValue, error) {
	jsonVal := update.Val.GetJsonVal()
	if jsonVal == nil {
		jsonVal = update.Val.GetJsonIetfVal()
	}
	configValuesUnparsed, err := store.DecomposeTree(jsonVal)
	if err != nil {
		return nil, err
	}
	configValues, err := jsonvalues.CorrectJSONPaths("", configValuesUnparsed, sync.modelReadOnlyPaths, true)
	if err != nil {
		return nil, err
	}
	return configValues, nil
}

/**
 *	subscribeOpState only subscribes to the paths that were successfully retrieved
 *	with Get (of state - which ever method was successful).
 *  This can be found from the OpStateCache
 *  At this stage the wildcards will have been expanded and the ReadOnly paths traversed
 */
func (sync *Synchronizer) subscribeOpState(target southbound.TargetIf, errChan chan<- events.DeviceResponse) {
	subscribePaths := make([][]string, 0)
	for p := range sync.operationalCache {
		subscribePaths = append(subscribePaths, utils.SplitPath(p))
	}

	options := &southbound.SubscribeOptions{
		UpdatesOnly:       false,
		Prefix:            "",
		Mode:              "stream",
		StreamMode:        "target_defined",
		SampleInterval:    15,
		HeartbeatInterval: 15,
		Paths:             subscribePaths,
		Origin:            "",
	}

	log.Infof("Subscribing to %d paths. %s", len(subscribePaths), sync.key.DeviceID)
	req, err := southbound.NewSubscribeRequest(options)
	if err != nil {
		errChan <- events.NewErrorEventNoChangeID(events.EventTypeErrorParseConfig,
			sync.key.DeviceID, err)
		return
	}

	subErr := target.Subscribe(sync.Context, req, sync.opStateSubHandler)
	if subErr != nil {
		log.Warning("Error in subscribe", subErr)
		errChan <- events.NewErrorEventNoChangeID(events.EventTypeErrorSubscribe,
			sync.key.DeviceID, subErr)
		return
	}
	log.Info("Subscribe for OpState notifications on ", sync.key.DeviceID, " started")
}

func (sync *Synchronizer) getOpStatePathsByType(ctx context.Context,
	target southbound.TargetIf,
	reqtype gnmi.GetRequest_DataType) ([]*gnmi.Notification, error) {

	log.Infof("Getting %s partition for %s", reqtype, sync.key.DeviceID)
	requestState := &gnmi.GetRequest{
		Type:     reqtype,
		Encoding: sync.encoding,
	}

	responseState, err := target.Get(ctx, requestState)
	if err != nil {
		return nil, err
	}

	return responseState.Notification, nil
}

func (sync *Synchronizer) opStateSubHandler(msg proto.Message) error {

	resp, ok := msg.(*gnmi.SubscribeResponse)
	if !ok {
		return fmt.Errorf("failed to type assert message %#v", msg)
	}
	switch v := resp.Response.(type) {
	default:
		return fmt.Errorf("unknown response %T: %s", v, v)
	case *gnmi.SubscribeResponse_Error:
		return fmt.Errorf("error in response: %s", v)
	case *gnmi.SubscribeResponse_SyncResponse:
		if sync.query.Type == client.Poll || sync.query.Type == client.Once {
			return client.ErrStopReading
		}
	case *gnmi.SubscribeResponse_Update:
		notification := v.Update
		for _, update := range notification.Update {
			if update.Path == nil {
				return fmt.Errorf("invalid nil path in update: %v", update)
			}
			pathStr := utils.StrPath(update.Path)

			//TODO this currently supports only leaf values, and no * paths,
			// parsing of json is needed and a per path storage
			valStr := utils.StrVal(update.Val)

			// FIXME: this is a hack to ignore bogus values in phantom notifications coming from Stratum for some reason
			if valStr != "unsupported yet" {
				val, err := values.GnmiTypedValueToNativeType(update.Val)
				if err != nil {
					return fmt.Errorf("can't translate to Typed value %s", err)
				}
				sync.operationalStateChan <- events.NewOperationalStateEvent(string(sync.Device.ID), pathStr, val, events.EventItemUpdated)

				sync.operationalCache[pathStr] = val
			}
		}

		for _, del := range notification.Delete {
			if del.Elem == nil {
				return fmt.Errorf("invalid nil path in update: %v", del)
			}
			pathStr := utils.StrPathElem(del.Elem)
			log.Info("Delete path ", pathStr, " for device ", sync.ID)
			sync.operationalStateChan <- events.NewOperationalStateEvent(string(sync.Device.ID), pathStr, nil, events.EventItemDeleted)
			delete(sync.operationalCache, pathStr)
		}

	}
	return nil
}

func getNetworkConfig(sync *Synchronizer, target string, configname string, layer int) ([]*devicechangetypes.PathValue, error) {
	log.Info("Getting saved config for ", target)
	//TODO the key of the config store should be a tuple of (devicename, configname) use the param
	var config store.Configuration
	if target != "" {
		for _, cfg := range sync.ConfigurationStore.Store {
			if cfg.Device == target {
				config = cfg
				break
			}
		}
		if config.Name == "" {
			return make([]*devicechangetypes.PathValue, 0),
				fmt.Errorf("No Configuration found for %s", target)
		}
	} else if configname != "" {
		config = sync.ConfigurationStore.Store[store.ConfigName(configname)]
		if config.Name == "" {
			return make([]*devicechangetypes.PathValue, 0),
				fmt.Errorf("No Configuration found for %s", configname)
		}
	}
	configValues := config.ExtractFullConfig(nil, sync.ChangeStore.Store, layer)
	if len(configValues) != 0 {
		return configValues, nil
	}

	return nil, fmt.Errorf("No Configuration for Device %s", target)
}
