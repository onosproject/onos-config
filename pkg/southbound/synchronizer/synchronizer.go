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
	"github.com/onosproject/onos-config/pkg/utils"
	"github.com/onosproject/onos-config/pkg/utils/values"
	"github.com/onosproject/onos-topo/pkg/northbound/device"
	"github.com/openconfig/gnmi/client"
	"github.com/openconfig/gnmi/proto/gnmi"
	"google.golang.org/grpc/status"
	log "k8s.io/klog"
	"strings"
	"time"
)

// Synchronizer enables proper configuring of a device based on store events and cache of operational data
type Synchronizer struct {
	context.Context
	*store.ChangeStore
	*store.ConfigurationStore
	*device.Device
	deviceConfigChan     <-chan events.ConfigEvent
	operationalStateChan chan<- events.OperationalStateEvent
	key                  southbound.DeviceID
	query                client.Query
	modelReadOnlyPaths   modelregistry.ReadOnlyPathMap
	operationalCache     change.TypedValueMap
	encoding             gnmi.Encoding
}

// New Build a new Synchronizer given the parameters, starts the connection with the device and polls the capabilities
func New(context context.Context, changeStore *store.ChangeStore, configStore *store.ConfigurationStore,
	device *device.Device, deviceCfgChan <-chan events.ConfigEvent, opStateChan chan<- events.OperationalStateEvent,
	errChan chan<- events.DeviceResponse, opStateCache change.TypedValueMap,
	mReadOnlyPaths modelregistry.ReadOnlyPathMap) (*Synchronizer, error) {
	sync := &Synchronizer{
		Context:              context,
		ChangeStore:          changeStore,
		ConfigurationStore:   configStore,
		Device:               device,
		deviceConfigChan:     deviceCfgChan,
		operationalStateChan: opStateChan,
		operationalCache:     opStateCache,
		modelReadOnlyPaths:   mReadOnlyPaths,
	}
	log.Info("Connecting to ", sync.Device.Address, " over gNMI")
	target := southbound.Target{}
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
		errChan <- events.CreateErrorEventNoChangeID(events.EventTypeErrorDeviceCapabilities,
			string(device.ID), capErr)
		return nil, capErr
	}
	sync.encoding = gnmi.Encoding_PROTO // Default
	// Currently stratum does not return capabilities (Aug 19)
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
		initialConfig, err := change.CreateChangeValuesNoRemoval(config, "Initial set to device")

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
			errChan <- events.CreateErrorEvent(events.EventTypeErrorSetInitialConfig,
				string(device.ID), initialConfig.ID, err)
		} else {
			log.Infof("Loaded initial config %s for device %s", store.B64(initialConfig.ID), sync.key.DeviceID)
			errChan <- events.CreateResponseEvent(events.EventTypeAchievedSetConfig,
				sync.key.DeviceID, initialConfig.ID, resp.String())
		}
	}

	return sync, nil
}

// syncConfigEventsToDevice is a go routine that listens out for configuration events specific
// to a device and propagates them downwards through southbound interface
func (sync *Synchronizer) syncConfigEventsToDevice(respChan chan<- events.DeviceResponse) {

	for deviceConfigEvent := range sync.deviceConfigChan {
		c := sync.ChangeStore.Store[deviceConfigEvent.ChangeID()]
		err := c.IsValid()
		if err != nil {
			log.Warning("Event discarded because change is invalid: ", err)
			respChan <- events.CreateErrorEvent(events.EventTypeErrorParseConfig,
				sync.key.DeviceID, c.ID, err)
			continue
		}
		gnmiChange, parseError := values.NativeChangeToGnmiChange(c)

		if parseError != nil {
			log.Error("Parsing error for gNMI change: ", parseError)
			respChan <- events.CreateErrorEvent(events.EventTypeErrorParseConfig,
				sync.key.DeviceID, c.ID, err)
			continue
		}

		log.Info("Change formatted to gNMI setRequest ", gnmiChange)
		target, err := southbound.GetTarget(sync.key)
		if err != nil {
			log.Warning(err)
			respChan <- events.CreateErrorEvent(events.EventTypeErrorDeviceConnect,
				sync.key.DeviceID, c.ID, err)
			continue
		}
		setResponse, err := target.Set(sync.Context, gnmiChange)
		if err != nil {
			log.Error("Error while doing set: ", err)
			respChan <- events.CreateErrorEvent(events.EventTypeErrorSetConfig,
				sync.key.DeviceID, c.ID, err)
			continue
		}
		log.Info(sync.Device.Address, " SetResponse ", setResponse)
		respChan <- events.CreateResponseEvent(events.EventTypeAchievedSetConfig,
			sync.key.DeviceID, c.ID, setResponse.String())

	}
}

func (sync Synchronizer) syncOperationalState(errChan chan<- events.DeviceResponse) {
	target, err := southbound.GetTarget(sync.key)
	if err != nil {
		log.Error("Can't find target for key: ", sync.key)
		errChan <- events.CreateErrorEventNoChangeID(events.EventTypeErrorDeviceConnect,
			sync.key.DeviceID, err)
		return
	}
	log.Info("Syncing Op & State of  ", sync.key.DeviceID, " started")

	if sync.modelReadOnlyPaths == nil {
		errMp := fmt.Errorf("no model plugin, cant work in operational state cache")
		log.Error(errMp)
		errChan <- events.CreateErrorEventNoChangeID(events.EventTypeErrorMissingModelPlugin,
			sync.key.DeviceID, errMp)
		return
	}

	notifications := make([]*gnmi.Notification, 0)
	stateNotif, errState := sync.getOpStatePathsByType(target, gnmi.GetRequest_STATE)
	if errState != nil {
		log.Warning("Can't request read-only state paths to target ", sync.key, errState)
	} else {
		notifications = append(notifications, stateNotif...)
	}

	operNotif, errOp := sync.getOpStatePathsByType(target, gnmi.GetRequest_OPERATIONAL)
	if errState != nil {
		log.Warning("Can't request read-only operational paths to target ", sync.key, errOp)
	} else {
		notifications = append(notifications, operNotif...)
	}

	//TODO do get for subscribePaths (the paths we get out of the model)
	// This is because the previous 2 gets we have done might be empty or unimplemented
	// There is a risk that the device might return everything (including config)
	// If the device does not support wildcards GET of the whole tree and we need
	// to take the burden of parsing it
	//   so the following should be considered in the if statement:
	// 1) GET of STATE with no path - assuming this will return JSON of RO attributes
	// * Could be empty (device might not have any state - valid but unlikely)
	// * or this qualified GET is not supported by device - do we get error to this effect?
	// * These paths are compared to RO paths and any that should not be there are quietly dropped
	// * THIS IS CURRENTLY IMPLEMENTED 01/Aug
	// 2) GET of OPERATIONAL with no path - assuming this will return JSON of RO attributes
	// * same as above
	// * THIS IS CURRENTLY IMPLEMENTED 01/Aug
	// 3) GET of paths as extracted from the RO paths of the model
	// * This is a specific set of gNMI Paths in a GetRequest with wildcards for list keys
	// * This has wildcards because we don't know the instances
	// * This requires the device to support wildcards
	// * This gives us the instances of RO lists that currently exist
	// * The result of this will be several gNMI notifications or updates and anyone
	//   of these could be an int or string value instead of JSON_VAL
	// * THIS IS NOT CURRENTLY IMPLEMENTED 01/Aug
	// * Do we need to call this at all if the result from 1) and 2) is valid? The
	//    only reason to do this is if we don't trust the device.
	// 4) GET against the whole tree without a path (empty)
	// * this is needed if we get no response from 1), 2) or 3)
	// * This will include Config and RO values
	// * We assume this is needed for Stratum but need to prove it
	// * In this case we parse the JSON result and weed out only the RO values
	// * This gives us the instances of RO lists that currently exist
	// * THIS IS NOT CURRENTLY IMPLEMENTED 01/Aug

	// TODO Parse the GET result to get actual instances of list items (replacing
	//  wildcarded paths - assuming that stratum does not support wildcarded GET
	//  and SUBSCRIBE)
	log.Infof("%d ReadOnly paths for %s", len(sync.modelReadOnlyPaths), sync.key.DeviceID)
	if len(sync.modelReadOnlyPaths) == 0 {
		noPathErr := fmt.Errorf("target %#v has no paths to subscribe to", sync.ID)
		errChan <- events.CreateErrorEventNoChangeID(events.EventTypeErrorSubscribe,
			sync.key.DeviceID, noPathErr)
		log.Warning(noPathErr)
		return
	}

	wildExpandedPaths := make([]string, 0)
	if len(notifications) == 0 {
		log.Infof("No notifications received - trying Get on the readonly paths instead %s", sync.key.DeviceID)
		getPaths := make([]*gnmi.Path, 0)
		for _, path := range modelregistry.Paths(sync.modelReadOnlyPaths) {
			gnmiPath, err := utils.ParseGNMIElements(utils.SplitPath(path))
			if err != nil {
				log.Warning("Error converting RO path to gNMI")
				errChan <- events.CreateErrorEventNoChangeID(events.EventTypeErrorTranslation,
					sync.key.DeviceID, err)
				return
			}
			getPaths = append(getPaths, gnmiPath)
		}

		requestRoPaths := &gnmi.GetRequest{
			Encoding: sync.encoding,
			Path:     getPaths,
		}

		responseRoPaths, errRoPaths := target.Get(target.Ctx, requestRoPaths)
		if errRoPaths != nil {
			log.Warning("Error on request for read-only paths", sync.key, errRoPaths)
			errChan <- events.CreateErrorEventNoChangeID(events.EventTypeErrorGetWithRoPaths,
				sync.key.DeviceID, errRoPaths)
			return
		}

		notifications = append(notifications, responseRoPaths.Notification...)

		////////////////////////////////////////////////////////////////////////
		// Special case for Stratum Aug'19 - the notification will only contain
		// the ifindex and name of the interface - to get the other state attributes
		// it has to be called again with the wildcard values expanded out. This
		// is because Stratum does not support wildcards properly.
		// Fortunately these can be also used for subscribe
		////////////////////////////////////////////////////////////////////////
		if sync.Type == "Stratum" {
			ewGetPaths := make([]*gnmi.Path, 0)
			for _, n := range responseRoPaths.Notification {
				for _, u := range n.Update {
					updatepath := utils.StrPath(u.Path)
					if strings.HasPrefix(updatepath, "/interfaces/interface[") {
						newPathStr := updatepath[:strings.LastIndex(updatepath, "/ifindex")]
						wildExpandedPaths = append(wildExpandedPaths, newPathStr)
						newGnmiPath, errNewPath := utils.ParseGNMIElements(utils.SplitPath(newPathStr))
						if errNewPath != nil {
							log.Warning("Error on request for read-only paths", sync.key, errNewPath)
							errChan <- events.CreateErrorEventNoChangeID(events.EventTypeErrorGetWithRoPaths,
								sync.key.DeviceID, errNewPath)
							return
						}
						log.Info("Making special expanded wildcard call to ", newPathStr, " for ", sync.key)
						ewGetPaths = append(ewGetPaths, newGnmiPath)
					}
				}
			}

			requestEwRoPaths := &gnmi.GetRequest{
				Encoding: sync.encoding,
				Path:     ewGetPaths,
			}

			log.Infof("Calling Get again for %s with expanded %d wildcard read-only paths", sync.key, len(ewGetPaths))
			responseEwRoPaths, errRoPaths := target.Get(target.Ctx, requestEwRoPaths)
			if errRoPaths != nil {
				log.Warning("Error on request for expanded wildcard read-only paths", sync.key, errRoPaths)
				errChan <- events.CreateErrorEventNoChangeID(events.EventTypeErrorGetWithRoPaths,
					sync.key.DeviceID, errRoPaths)
				return
			}
			notifications = append(notifications, responseEwRoPaths.Notification...)
		}

	}

	log.Infof("Handling %d received OpState paths. %s", len(notifications), sync.key.DeviceID)
	if len(notifications) != 0 {
		for _, notification := range notifications {
			for _, update := range notification.Update {
				if sync.encoding == gnmi.Encoding_JSON || sync.encoding == gnmi.Encoding_JSON_IETF {
					jsonVal := update.Val.GetJsonVal()
					if jsonVal == nil {
						jsonVal = update.Val.GetJsonIetfVal()
					}
					configValuesUnparsed, err := store.DecomposeTree(jsonVal)
					if err != nil {
						log.Error("Can't translate from json to values, skipping to next update: ", err, jsonVal)
						errChan <- events.CreateErrorEventNoChangeID(events.EventTypeErrorTranslation,
							sync.key.DeviceID, err)
						break
					}
					configValues, err := jsonvalues.CorrectJSONPaths("", configValuesUnparsed, sync.modelReadOnlyPaths, true)
					if err != nil {
						log.Error("Can't translate from config values to typed values, skipping to next update: ", err)
						errChan <- events.CreateErrorEventNoChangeID(events.EventTypeErrorTranslation,
							sync.key.DeviceID, err)
						break
					}
					pathsAndValues := make(change.TypedValueMap)
					for _, cv := range configValues {
						pathsAndValues[cv.Path] = &cv.TypedValue
					}
					for path, value := range pathsAndValues {
						if value != nil {
							sync.operationalCache[path] = value
						}
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
	} else {
		log.Warning("No Op or State results received from ", sync.key.DeviceID)
	}

	// Now try the subscribe with the read only paths and the expanded wildcard
	// paths (if any) from above
	sync.subscribeOpState(wildExpandedPaths, errChan)
}

// TODO We do a subscribe here based on wildcards and the RO paths calculated
//  from the model - subscribePaths
//  * The subscribe must go down to the leaf, so the subpaths of the
//  readonly paths must be included too
//  * If wildcards are supported by the device they will notify us about the
//  insertion of new items in a list in this case
//  * If the device does NOT support wildcards in subscribe we
//  will not get an error response - so how do we know it doesn't support wildcards?
//  //
//  Instead we should
//  ** Use this list of paths in the subscribe below to replace
//  the wildcards in subscribePaths
//  ** The caveat with that is that we will will not get notified of
//   items that are added to lists after the initial subscribe is done

// TODO check if it's possible to do a Subscribe with a qualifier of
//  OPERATIONAL or STATE like you can do for GET (unlikely). This is a moot
//  point if we had to go down to the leaf anyway.
//  //
//  The easiest fix here for the subscribe is to use the paths and values
//  from above and populate it as best you can from the GET options above
func (sync *Synchronizer) subscribeOpState(wildExpandedPaths []string, errChan chan<- events.DeviceResponse) {
	target, err := southbound.GetTarget(sync.key)
	if err != nil {
		log.Error("Can't find target for key ", sync.key)
		errChan <- events.CreateErrorEventNoChangeID(events.EventTypeErrorDeviceConnect,
			sync.key.DeviceID, err)
		return
	}
	log.Info("Syncing Op & State of  ", sync.key.DeviceID, " started")

	subscribePaths := make([][]string, 0)
	for _, path := range modelregistry.Paths(sync.modelReadOnlyPaths) {
		subscribePaths = append(subscribePaths, utils.SplitPath(path))
	}
	for _, w := range wildExpandedPaths {
		subscribePaths = append(subscribePaths, utils.SplitPath(w))
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
		errChan <- events.CreateErrorEventNoChangeID(events.EventTypeErrorParseConfig,
			sync.key.DeviceID, err)
		return
	}

	subErr := target.Subscribe(sync.Context, req, sync.handler)
	if subErr != nil {
		log.Warning("Error in subscribe", subErr)
		errChan <- events.CreateErrorEventNoChangeID(events.EventTypeErrorSubscribe,
			sync.key.DeviceID, subErr)
		return
	}
	log.Info("Subscribe for OpState notifications on ", sync.key.DeviceID, " started")
}

func (sync *Synchronizer) getOpStatePathsByType(target *southbound.Target, reqtype gnmi.GetRequest_DataType) ([]*gnmi.Notification, error) {
	log.Infof("Getting %s partition for %s", reqtype, sync.key.DeviceID)
	requestState := &gnmi.GetRequest{
		Type:     reqtype,
		Encoding: sync.encoding,
	}

	responseState, err := target.Get(target.Ctx, requestState)
	if err != nil {
		return nil, err
	}

	return responseState.Notification, nil
}

func (sync *Synchronizer) handler(msg proto.Message) error {

	_, err := southbound.GetTarget(sync.key)
	if err != nil {
		return fmt.Errorf("target not connected %#v", msg)
	}

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
		eventValues := make(map[string]string)
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
				eventValues[pathStr] = valStr
				log.Info("Added ", val, " for path ", pathStr, " for device ", sync.ID)

				//TODO this is a hack for Stratum Sept 19
				if strings.HasPrefix(pathStr, "/interfaces/interface[") && strings.HasSuffix(pathStr, "]/state/name") {
					pathChange := strings.Replace(pathStr, "/state/", "/config/", 1)
					log.Infof("Adding placeholder config node for %s as %s", pathStr, pathChange)
					var newChanges = make([]*change.Value, 0)
					changeValue, _ := change.CreateChangeValue(pathChange, val, false)
					newChanges = append(newChanges, changeValue)
					chg, err := change.CreateChange(newChanges, "Setting config name")
					if err != nil {
						log.Error("Unable to add placeholder node due to: ", err)
						continue
					}
					sync.ChangeStore.Store[store.B64(chg.ID)] = chg
					var newConfig *store.Configuration
					for _, cfg := range sync.ConfigurationStore.Store {
						if cfg.Device == sync.Target {
							log.Infof("Tying placeholder config node to %s", sync.GetTarget())
							newConfig = &cfg
							break
						}
					}
					if newConfig == nil {
						log.Infof("Creating new configuration for %s-%s to hold the placeholder config change %s",
							sync.GetTarget(), sync.GetVersion(), store.B64(chg.ID))
						newConfig, _ = store.CreateConfiguration(sync.GetTarget(), sync.GetVersion(), string(sync.GetType()), []change.ID{})
					}
					if newConfig != nil {
						found := false
						chID := store.B64(chg.ID)
						for _, id := range newConfig.Changes {
							if store.B64(id) == chID {
								found = true
								break
							}
						}
						if !found {
							log.Infof("Appending placeholder config change %s to config store", chID)
							newConfig.Changes = append(newConfig.Changes, chg.ID)
							newConfig.Updated = time.Now()
							sync.ConfigurationStore.Store[newConfig.Name] = *newConfig
						}
					}
				}
				sync.operationalCache[pathStr] = val
			}
		}

		for _, del := range notification.Delete {
			if del.Elem == nil {
				return fmt.Errorf("invalid nil path in update: %v", del)
			}
			pathStr := utils.StrPathElem(del.Elem)
			eventValues[pathStr] = ""
			log.Info("Delete path ", pathStr, " for device ", sync.ID)
			delete(sync.operationalCache, pathStr)
		}
		sync.operationalStateChan <- events.CreateOperationalStateEvent(string(sync.Device.ID), eventValues)
	}
	return nil
}

func getNetworkConfig(sync *Synchronizer, target string, configname string, layer int) ([]*change.ConfigValue, error) {
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
			return make([]*change.ConfigValue, 0),
				fmt.Errorf("No Configuration found for %s", target)
		}
	} else if configname != "" {
		config = sync.ConfigurationStore.Store[store.ConfigName(configname)]
		if config.Name == "" {
			return make([]*change.ConfigValue, 0),
				fmt.Errorf("No Configuration found for %s", configname)
		}
	}
	configValues := config.ExtractFullConfig(nil, sync.ChangeStore.Store, layer)
	if len(configValues) != 0 {
		return configValues, nil
	}

	return nil, fmt.Errorf("No Configuration for Device %s", target)
}
