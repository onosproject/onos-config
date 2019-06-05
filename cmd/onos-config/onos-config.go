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
Package main of cmd/onos-config is the main entry point to the system.

It connects to devices through a Southbound gNMI interface and
gives a gNMI interface northbound for other systems to connect to, and an
Admin service through gRPC

Arguments

-configStore <the location of a Configuration store> (stores/configStore-sample.json by default)

-changeStore <the location of a Change store> (stores/changeStore-sample.json by default)

-deviceStore <the location of a TopoCache store> (stores/deviceStore-sample.json by default)

-networkStore <the location of a Network store> (stores/networkStore-sample.json by default)

-caPath <the location of a CA certificate>

-keyPath <the location of a client private key>

-certPath <the location of a client certificate>


See ../../docs/run.md for how to run the application.
*/
package main

import (
	"flag"
	"github.com/onosproject/onos-config/pkg/manager"
	"github.com/onosproject/onos-config/pkg/northbound"
	"github.com/onosproject/onos-config/pkg/northbound/admin"
	"github.com/onosproject/onos-config/pkg/northbound/diags"
	"github.com/onosproject/onos-config/pkg/northbound/gnmi"
	log "k8s.io/klog"
	"time"
)

// Default locations of stores
const (
	configStoreDefaultFileName  = "../configs/configStore-sample.json"
	changeStoreDefaultFileName  = "../configs/changeStore-sample.json"
	deviceStoreDefaultFileName  = "../configs/deviceStore-sample.json"
	networkStoreDefaultFileName = "../configs/networkStore-sample.json"
)

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
	caPath := flag.String("caPath", "", "path to CA certificate")
	keyPath := flag.String("keyPath", "", "path to client private key")
	certPath := flag.String("certPath", "", "path to client certificate")

	flag.Parse()
	var err error

	log.Info("Starting onos-config")

	mgr, err := manager.LoadManager(*configStoreFile, *changeStoreFile, *deviceStoreFile, *networkStoreFile)
	if err != nil {
		log.Fatal("Unable to load onos-config ", err)
	} else {
		defer func() {
			close(mgr.TopoChannel)
			log.Info("Shutting down onos-config")
			time.Sleep(time.Second)
		}()

		mgr.Run()
		err = startServer(caPath, keyPath, certPath)
		if err != nil {
			log.Fatal("Unable to start onos-config ", err)
		}
	}
}

// Starts gRPC server and registers various services; then serves
func startServer(caPath *string, keyPath *string, certPath *string) error {
	cfg := &northbound.ServerConfig{
		Port:     5150,
		Insecure: true,
		CaPath:   caPath,
		KeyPath:  keyPath,
		CertPath: certPath,
	}

	serv := northbound.NewServer(cfg)
	serv.AddService(admin.Service{})
	serv.AddService(diags.Service{})
	serv.AddService(gnmi.Service{})

	return serv.Serve()
}
