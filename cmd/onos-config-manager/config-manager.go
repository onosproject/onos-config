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

/*
Package main is for the command onos-config-manager

It connects to devices through a Southbound gNMI interface and
gives a gNMI interface northbound for other systems to connect to, and an
Admin service through gRPC

Arguments

-configStore <the location of a Configuration store> (stores/configStore-sample.json by default)

-changeStore <the location of a Change store> (stores/changeStore-sample.json by default)

-deviceStore <the location of a TopoCache store> (stores/deviceStore-sample.json by default)

-networkStore <the location of a Network store> (stores/networkStore-sample.json by default)

To run from anywhere

	go run github.com/opennetworkinglab/onos-config/onos-config-manager \
	-configStore=$HOME/go/src/github.com/opennetworkinglab/onos-config/onos-config-manager/stores/configStore-sample.json \
	-changeStore=$HOME/go/src/github.com/opennetworkinglab/onos-config/onos-config-manager/stores/changeStore-sample.json \
    -deviceStore=$HOME/go/src/github.com/onosproject/onos-config/configs/deviceStore-sample.json \
    -networkStore=$HOME/go/src/github.com/onosproject/onos-config/configs/networkStore-sample.json


*/
package main

import (
	"flag"
	"github.com/onosproject/onos-config/cmd/onos-config-manager/shell"
	"github.com/onosproject/onos-config/pkg/events"
	"github.com/onosproject/onos-config/pkg/listener"
	"github.com/onosproject/onos-config/pkg/northbound"
	"github.com/onosproject/onos-config/pkg/northbound/admin"
	"github.com/onosproject/onos-config/pkg/northbound/gnmi"
	"github.com/onosproject/onos-config/pkg/manager"
	"github.com/onosproject/onos-config/pkg/southbound/synchronizer"
	"github.com/onosproject/onos-config/pkg/southbound/topocache"
	"github.com/onosproject/onos-config/pkg/store"
	log "k8s.io/klog"
	"os"
	"time"
)

// Default locations of stores
const (
	configStoreDefaultFileName  = "../configs/configStore-sample.json"
	changeStoreDefaultFileName  = "../configs/changeStore-sample.json"
	deviceStoreDefaultFileName  = "../configs/deviceStore-sample.json"
	networkStoreDefaultFileName = "../configs/networkStore-sample.json"
)

var (
	configStore    store.ConfigurationStore
	changeStore    store.ChangeStore
	deviceStore    *topocache.DeviceStore
	networkStore   *store.NetworkStore
	changesChannel chan events.Event
	topoChannel    chan events.Event
)

func init() {
	// Start the main listener system
	changesChannel = make(chan events.Event, 10)
	go listener.Listen(changesChannel)
	topoChannel = make(chan events.Event, 10)

}

// The main entry point
func main() {
	configStoreFile := flag.String("configStore", configStoreDefaultFileName,
		"path to config store file")
	changeStoreFile := flag.String("changeStore", changeStoreDefaultFileName,
		"path to change store file")
	deviceStoreFile := flag.String("deviceStore", deviceStoreDefaultFileName,
		"path to device store file")
	networkStoreFile := flag.String("networkStore", networkStoreDefaultFileName,
		"path to network store file")

	flag.Parse()
	var err error

	if err != nil {
		log.Fatal(err)
	}
	log.Info("onos-config-manager started")

	configStore, err = store.LoadConfigStore(*configStoreFile)
	if err != nil {
		log.Fatal("Cannot load config store ", err)
	}
	log.Info("Configuration store loaded from", *configStoreFile)

	changeStore, err = store.LoadChangeStore(*changeStoreFile)
	if err != nil {
		log.Fatal("Cannot load change store ", err)
	}
	log.Info("Change store loaded from", *changeStoreFile)

	deviceStore, err = topocache.LoadDeviceStore(*deviceStoreFile, topoChannel)
	if err != nil {
		log.Fatal("Cannot load device store ", err)
	}
	log.Info("Device store loaded from", *deviceStoreFile)

	networkStore, err = store.LoadNetworkStore(*networkStoreFile)
	if err != nil {
		log.Fatal("Cannot load network store ", err)
	}
	log.Info("Network store loaded from", *networkStoreFile)

	go synchronizer.Factory(&changeStore, deviceStore, topoChannel)

	go manager.Manager(&configStore, &changeStore, deviceStore, networkStore)

	startServer()

	// Run a shell as a temporary solution to not having an NBI
	shell.RunShell(configStore, changeStore, deviceStore, networkStore, changesChannel)

	close(changesChannel)
	close(topoChannel)

	log.Info("Shutting down")
	time.Sleep(time.Second)
	os.Exit(0)
}

func startServer() {
	cfg := &northbound.ServerConfig{
		Port : 5150,
		Insecure: true,
	}

	serv := northbound.NewServer(cfg)
	serv.AddService(admin.Service{})
	serv.AddService(gnmi.Service{})

	go func() {
		err := serv.Serve()
		if err != nil {
			log.Error("Serve failed", err)
		}
	}()
}
