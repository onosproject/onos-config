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
	topopb "github.com/onosproject/onos-topo/pkg/northbound/proto"
	"github.com/openconfig/gnmi/client"
	"github.com/openconfig/gnmi/proto/gnmi"
	"google.golang.org/grpc/status"
	log "k8s.io/klog"
	"strings"
)

// Synchronizer enables proper configuring of a device based on store events and cache of operational data
type Synchronizer struct {
	context.Context
	*store.ChangeStore
	*store.ConfigurationStore
	*topopb.Device
	deviceConfigChan     <-chan events.ConfigEvent
	operationalStateChan chan<- events.OperationalStateEvent
	key                  southbound.DeviceID
	query                client.Query
	modelReadOnlyPaths   modelregistry.ReadOnlyPathMap
	operationalCache     change.TypedValueMap
}

// New Build a new Synchronizer given the parameters, starts the connection with the device and polls the capabilities
func New(context context.Context, changeStore *store.ChangeStore, configStore *store.ConfigurationStore,
	device *topopb.Device, deviceCfgChan <-chan events.ConfigEvent, opStateChan chan<- events.OperationalStateEvent,
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
	key, err := target.ConnectTarget(context, sync.Device)
	sync.key = key
	if err != nil {
		log.Warning(err)
		return nil, err
	}
	log.Info(sync.Device.Address, " connected over gNMI")

	// Get the device capabilities
	capResponse, capErr := target.CapabilitiesWithString(context, "")
	if capErr != nil {
		log.Error(sync.Device.Address, " capabilities ", err)
		errChan <- events.CreateErrorEventNoChangeID(events.EventTypeErrorDeviceCapabilities, device.Id, err)
		return nil, err
	}

	log.Info(sync.Device.Address, " capabilities ", capResponse)

	config, err := getNetworkConfig(sync, sync.Device.Id, "", 0)

	//Device does not have any stored config at the moment, skip initial set
	if err != nil {
		log.Info(sync.Device.Address, " has no initial configuration")
	} else {
		//Device has initial configuration saved in onos-config, trying to apply
		initialConfig, err := change.CreateChangeValuesNoRemoval(config, "Initial set to device")

		if err != nil {
			log.Error("Can't translate the initial config for ", sync.Device.Address, err)
			return sync, nil
		}

		gnmiChange, err := values.NativeChangeToGnmiChange(initialConfig)

		if err != nil {
			log.Error("Can't obtain GnmiChange for ", sync.Device.Address, err)
			return sync, nil
		}

		resp, err := target.Set(context, gnmiChange)

		if err != nil {
			errGnmi, _ := status.FromError(err)
			//Hack because the desc field is not available.
			//Splitting at the desc string and getting the second element which is the description.
			log.Errorf("Can't set initial configuration for %s due to %s", sync.Device.Address,
				strings.Split(errGnmi.Message(), " desc = ")[1])
			errChan <- events.CreateErrorEvent(events.EventTypeErrorSetInitialConfig, device.Id, initialConfig.ID, err)
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
			log.Warning("Event discarded because change is invalid ", err)
			respChan <- events.CreateErrorEvent(events.EventTypeErrorParseConfig,
				sync.key.DeviceID, c.ID, err)
			continue
		}
		gnmiChange, parseError := values.NativeChangeToGnmiChange(c)

		if parseError != nil {
			log.Error("Parsing error for Gnmi change ", parseError)
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
			log.Error("Error while doing set ", err)
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
		log.Error("Can't find target for key ", sync.key)
		errChan <- events.CreateErrorEventNoChangeID(events.EventTypeErrorDeviceConnect,
			sync.key.DeviceID, err)
		return
	}

	subscribePaths := make([][]string, 0)

	if sync.modelReadOnlyPaths != nil {
		for _, path := range modelregistry.Paths(sync.modelReadOnlyPaths) {
			subscribePaths = append(subscribePaths, utils.SplitPath(path))
		}
	} else {
		errMp := fmt.Errorf("No model plugin, cant work in operational state cache")
		log.Error(errMp)
		errChan <- events.CreateErrorEventNoChangeID(events.EventTypeErrorMissingModelPlugin,
			sync.key.DeviceID, errMp)
		return
	}

	if len(subscribePaths) == 0 {
		noPathErr := fmt.Errorf("target %#v has no paths to subscribe to", sync.Id)
		errChan <- events.CreateErrorEventNoChangeID(events.EventTypeErrorSubscribe,
			sync.key.DeviceID, noPathErr)
		log.Warning(noPathErr)
		return
	}

	notifications := make([]*gnmi.Notification, 0)

	requestState := &gnmi.GetRequest{
		Type: gnmi.GetRequest_STATE,
	}

	responseState, errState := target.Get(target.Ctx, requestState)

	if errState != nil {
		log.Warning("Can't request read-only state paths to target ", sync.key, errState)
	} else {
		notifications = append(notifications, responseState.Notification...)
	}

	requestOperational := &gnmi.GetRequest{
		Type: gnmi.GetRequest_OPERATIONAL,
	}

	responseOperational, errOp := target.Get(target.Ctx, requestOperational)

	if errOp != nil {
		log.Warning("Can't request read-only operational paths to target ", sync.key, errOp)
	} else {
		notifications = append(notifications, responseOperational.Notification...)
	}

	if len(notifications) != 0 {
		for _, notification := range notifications {
			for _, update := range notification.Update {
				//TODO check that this is json
				jsonVal := update.Val.GetJsonVal()
				configValuesUnparsed, err := store.DecomposeTree(jsonVal)
				if err != nil {
					log.Error("Can't translate from json to values, skipping to next update", err)
					errChan <- events.CreateErrorEventNoChangeID(events.EventTypeErrorTranslation,
						sync.key.DeviceID, err)
					break
				}
				configValues, err := jsonvalues.CorrectJSONPaths(configValuesUnparsed, sync.modelReadOnlyPaths, true)
				if err != nil {
					log.Error("Can't translate from config values to typed values, skipping to next update", err)
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
			}
		}
	}

	//TODO do get for subscribePaths

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

	req, err := southbound.NewSubscribeRequest(options)
	if err != nil {
		errChan <- events.CreateErrorEventNoChangeID(events.EventTypeErrorParseConfig,
			sync.key.DeviceID, err)
		return
	}

	subErr := target.Subscribe(sync.Context, req, sync.handler)
	if subErr != nil {
		errChan <- events.CreateErrorEventNoChangeID(events.EventTypeErrorSubscribe,
			sync.key.DeviceID, subErr)
	}

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
			val, err := values.GnmiTypedValueToNativeType(update.Val)
			if err != nil {
				return fmt.Errorf("can't translate to Typed value %s", err)
			}
			eventValues[pathStr] = valStr
			log.Info("Added ", val, " for path ", pathStr, " for device ", sync.Id)
			sync.operationalCache[pathStr] = val
		}
		for _, del := range notification.Delete {
			if del.Elem == nil {
				return fmt.Errorf("invalid nil path in update: %v", del)
			}
			pathStr := utils.StrPathElem(del.Elem)
			eventValues[pathStr] = ""
			log.Info("Delete path ", pathStr, " for device ", sync.Id)
			delete(sync.operationalCache, pathStr)
		}
		sync.operationalStateChan <- events.CreateOperationalStateEvent(sync.Device.Id, eventValues)
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
	configValues := config.ExtractFullConfig(sync.ChangeStore.Store, layer)
	if len(configValues) != 0 {
		return configValues, nil
	}

	return nil, fmt.Errorf("No Configuration for Device %s", target)
}
