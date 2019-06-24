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
	"github.com/onosproject/onos-config/pkg/southbound"
	"github.com/onosproject/onos-config/pkg/southbound/topocache"
	"github.com/onosproject/onos-config/pkg/store"
	"github.com/onosproject/onos-config/pkg/utils"
	"github.com/openconfig/gnmi/client"
	"github.com/openconfig/gnmi/proto/gnmi"
	log "k8s.io/klog"
)

// Synchronizer enables proper configuring of a device based on store events and cache of operational data
type Synchronizer struct {
	context.Context
	*store.ChangeStore
	*topocache.Device
	deviceConfigChan     <-chan events.ConfigEvent
	operationalStateChan chan<- events.OperationalStateEvent
	key                  southbound.DeviceID
	query                client.Query
	operationalCache     map[string]string
}

// New Build a new Synchronizer given the parameters, starts the connection with the device and polls the capabilities
func New(context context.Context, changeStore *store.ChangeStore, device *topocache.Device,
	deviceCfgChan <-chan events.ConfigEvent, opStateChan chan<- events.OperationalStateEvent) (*Synchronizer, error) {
	sync := &Synchronizer{
		Context:              context,
		ChangeStore:          changeStore,
		Device:               device,
		deviceConfigChan:     deviceCfgChan,
		operationalStateChan: opStateChan,
		operationalCache:     make(map[string]string),
	}
	log.Info("Connecting to ", sync.Device.Addr, " over gNMI")
	target := southbound.Target{}
	key, err := target.ConnectTarget(context, *sync.Device)
	sync.key = key
	if err != nil {
		log.Warning(err)
		return nil, err
	}
	log.Info(sync.Device.Addr, " connected over gNMI")

	// Get the device capabilities
	capResponse, capErr := target.CapabilitiesWithString(context, "")
	if capErr != nil {
		log.Error(sync.Device.Addr, " capabilities ", err)
		return nil, err
	}

	log.Info(sync.Device.Addr, " capabilities ", capResponse)

	return sync, nil
}

// syncConfigEventsToDevice is a go routine that listens out for configuration events specific
// to a device and propagates them downwards through southbound interface
func (sync *Synchronizer) syncConfigEventsToDevice() {

	for deviceConfigEvent := range sync.deviceConfigChan {
		change := sync.ChangeStore.Store[deviceConfigEvent.ChangeID()]
		err := change.IsValid()
		if err != nil {
			log.Warning("Event discarded because change is invalid ", err)
			continue
		}
		gnmiChange, parseError := change.GnmiChange()

		if parseError != nil {
			log.Error("Parsing error for Gnmi change ", parseError)
			continue
		}

		log.Info("Change formatted to gNMI setRequest ", gnmiChange)
		target, err := southbound.GetTarget(sync.key)
		if err != nil {
			log.Warning(err)
			continue
		}
		setResponse, err := target.Set(sync.Context, gnmiChange)
		if err != nil {
			log.Error("SetResponse ", err)
			continue
		}
		log.Info(sync.Device.Addr, " SetResponse ", setResponse)

	}
}

func (sync Synchronizer) syncOperationalState() error {

	target, err := southbound.GetTarget(sync.key)

	if err != nil {
		log.Error("Can't find target for key ", sync.key)
		return err
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

	subscribePaths := make([][]string, 0)

	if len(notifications) != 0 {
		for _, notification := range notifications {
			for _, update := range notification.Update {
				pathStr := utils.StrPath(update.Path)
				val := utils.StrVal(update.Val)
				sync.operationalCache[pathStr] = val
				subscribePaths = append(subscribePaths, utils.SplitPath(pathStr))
			}

		}
	} else {
		//TODO implement getting paths here leaving commented for examples of what works.
		//path := make([]string, 0)
		//Stratum
		//path = append(path, "interfaces")
		//path = append(path, "interface[name=s1-eth1]")
		//path = append(path, "state")
		//path = append(path, "admin-status")
		//path = append(path, "config")
		//path = append(path, "enabled")

		//Simulator
		//path = append(path, "system")
		//path = append(path, "openflow")
		//path = append(path, "controllers")
		//path = append(path, "controller[name=main]")
		//path = append(path, "connections")
		//path = append(path, "connection[aux-id=0]")
		//path = append(path, "state")
		//path = append(path, "address")
		//subscribePaths = append(subscribePaths, path)
	}

	if len(subscribePaths) == 0 {
		noPathErr := fmt.Errorf("target %#v has no paths to subscribe to", sync.ID)
		log.Warning(noPathErr)
		return noPathErr
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

	req, err := southbound.NewSubscribeRequest(options)
	if err != nil {
		return err
	}
	return target.Subscribe(sync.Context, req, sync.handler)

}

func (sync *Synchronizer) handler(msg proto.Message) error {

	log.Info("Proto Message ", msg)

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
			val := utils.StrVal(update.Val)
			eventValues[pathStr] = val
			sync.operationalCache[pathStr] = val
		}
		for _, del := range notification.Delete {
			if del.Elem == nil {
				return fmt.Errorf("invalid nil path in update: %v", del)
			}
			//deletedPaths := delete.Elem
			pathStr := utils.StrPathElem(del.Elem)
			eventValues[pathStr] = ""
			sync.operationalCache[pathStr] = ""
		}
		sync.operationalStateChan <- events.CreateOperationalStateEvent(string(sync.Device.ID), eventValues)
	}
	return nil
}
