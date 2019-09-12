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
	"github.com/onosproject/onos-config/pkg/certs"
	"github.com/onosproject/onos-config/pkg/manager"
	"github.com/onosproject/onos-config/pkg/northbound"
	"github.com/onosproject/onos-config/pkg/northbound/admin"
	"github.com/onosproject/onos-config/pkg/northbound/diags"
	"github.com/onosproject/onos-config/pkg/northbound/gnmi"
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

type arrayFlags []string

func (i *arrayFlags) String() string {
	return "my string representation"
}

func (i *arrayFlags) Set(value string) error {
	*i = append(*i, value)
	return nil
}

// The main entry point
func main() {
	var modelPlugins arrayFlags

	configStoreFile := flag.String("configStore", configStoreDefaultFileName,
		"path to config store file")
	changeStoreFile := flag.String("changeStore", changeStoreDefaultFileName,
		"path to change store file")
	networkStoreFile := flag.String("networkStore", networkStoreDefaultFileName,
		"path to network store file")

	// TODO: This flag is preserved for backwards compatibility
	_ = flag.String("deviceStore", deviceStoreDefaultFileName,
		"path to device store file")

	flag.Var(&modelPlugins, "modelPlugin", "names of model plugins to load (repeated)")
	caPath := flag.String("caPath", "", "path to CA certificate")
	keyPath := flag.String("keyPath", "", "path to client private key")
	certPath := flag.String("certPath", "", "path to client certificate")

	//lines 93-109 are implemented according to
	// https://github.com/kubernetes/klog/blob/master/examples/coexist_glog/coexist_glog.go
	// because of libraries importing glog. With glog import we can't call log.InitFlags(nil) as per klog readme
	// thus the alsologtostderr is not set properly and we issue multiple logs.
	// Calling log.InitFlags(nil) throws panic with error `flag redefined: log_dir`
	err := flag.Set("alsologtostderr", "true")
	if err != nil {
		log.Error("Cant' avoid double Error logging ", err)
	}
	flag.Parse()

	klogFlags := flag.NewFlagSet("klog", flag.ExitOnError)
	log.InitFlags(klogFlags)

	// Sync the glog and klog flags.
	flag.CommandLine.VisitAll(func(f1 *flag.Flag) {
		f2 := klogFlags.Lookup(f1.Name)
		if f2 != nil {
			value := f1.Value.String()
			_ = f2.Value.Set(value)
		}
	})
	log.Info("Starting onos-config")

	opts, err := certs.HandleCertArgs(keyPath, certPath)
	if err != nil {
		log.Fatal(err)
	}

	mgr, err := manager.LoadManager(*configStoreFile, *changeStoreFile, *networkStoreFile, opts...)
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
		err = mgr.ValidateStores()
		if err != nil {
			log.Error("Error when validating configurations against model plugins: ", err)
			os.Exit(-1)
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
	s := northbound.NewServer(northbound.NewServerConfig(caPath, keyPath, certPath))
	s.AddService(admin.Service{})
	s.AddService(diags.Service{})
	s.AddService(gnmi.Service{})

	return s.Serve(func(started string) {
		log.Info("Started NBI on ", started)
	})
}
