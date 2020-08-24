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
Package onos-config is the main entry point to the ONOS configuration subsystem.

It connects to devices through a Southbound gNMI interface and
gives a gNMI interface northbound for other systems to connect to, and an
Admin service through gRPC

Arguments
-allowUnvalidatedConfig <allow configuration for devices without a corresponding model plugin>

-modelPlugin (repeated) <the location of a shared object library that implements the Model Plugin interface>

-caPath <the location of a CA certificate>

-keyPath <the location of a client private key>

-certPath <the location of a client certificate>


See ../../docs/run.md for how to run the application.
*/
package main

import (
	"flag"
	"time"

	"github.com/onosproject/onos-config/pkg/manager"
	"github.com/onosproject/onos-config/pkg/northbound/admin"
	"github.com/onosproject/onos-config/pkg/northbound/diags"
	"github.com/onosproject/onos-config/pkg/northbound/gnmi"
	"github.com/onosproject/onos-lib-go/pkg/certs"
	"github.com/onosproject/onos-lib-go/pkg/logging"
	"github.com/onosproject/onos-lib-go/pkg/northbound"
)

type arrayFlags []string

func (i *arrayFlags) String() string {
	return "my string representation"
}

func (i *arrayFlags) Set(value string) error {
	*i = append(*i, value)
	return nil
}

var log = logging.GetLogger("main")

// The main entry point
func main() {
	var modelPlugins arrayFlags
	allowUnvalidatedConfig := flag.Bool("allowUnvalidatedConfig", false, "allow configuration for devices without a corresponding model plugin")
	flag.Var(&modelPlugins, "modelPlugin", "names of model plugins to load (repeated)")
	caPath := flag.String("caPath", "", "path to CA certificate")
	keyPath := flag.String("keyPath", "", "path to client private key")
	certPath := flag.String("certPath", "", "path to client certificate")
	topoEndpoint := flag.String("topoEndpoint", "onos-topo:5150", "topology service endpoint")
	//This flag is used in logging.init()
	flag.Bool("debug", false, "enable debug logging")
	flag.Parse()

	log.Info("Starting onos-config")

	opts, err := certs.HandleCertPaths(*caPath, *keyPath, *certPath, true)
	if err != nil {
		log.Fatal(err)
	}

	mgr, err := manager.NewManager(topoEndpoint, opts, *allowUnvalidatedConfig)
	log.Info("Manager started")

	if err != nil {
		log.Fatal("Unable to load onos-config ", err)
	} else {
		defer func() {
			close(mgr.TopoChannel)
			log.Info("Shutting down onos-config")
			time.Sleep(time.Second)
		}()

		for _, modelPlugin := range modelPlugins {
			if modelPlugin == "" {
				continue
			}
			_, _, err := mgr.ModelRegistry.RegisterModelPlugin(modelPlugin)
			if err != nil {
				log.Fatal("Unable to start onos-config ", err)
			}
		}

		mgr.Run()
		err = startServer(*caPath, *keyPath, *certPath)
		if err != nil {
			log.Fatal("Unable to start onos-config ", err)
		}
	}
}

// Creates gRPC server and registers various services; then serves.
func startServer(caPath string, keyPath string, certPath string) error {
	s := northbound.NewServer(northbound.NewServerCfg(caPath, keyPath, certPath, 5150, true, northbound.SecurityConfig{}))
	s.AddService(admin.Service{})
	s.AddService(diags.Service{})
	s.AddService(gnmi.Service{})
	s.AddService(logging.Service{})

	return s.Serve(func(started string) {
		log.Info("Started NBI on ", started)
	})
}
