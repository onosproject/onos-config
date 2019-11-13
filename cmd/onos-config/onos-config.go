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

-modelPlugin (repeated) <the location of a shared object library that implements the Model Plugin interface>

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
	"github.com/onosproject/onos-config/pkg/store/change/device"
	"github.com/onosproject/onos-config/pkg/store/change/network"
	devicestore "github.com/onosproject/onos-config/pkg/store/device"
	"github.com/onosproject/onos-config/pkg/store/device/cache"
	"github.com/onosproject/onos-config/pkg/store/leadership"
	"github.com/onosproject/onos-config/pkg/store/mastership"
	devicesnap "github.com/onosproject/onos-config/pkg/store/snapshot/device"
	networksnap "github.com/onosproject/onos-config/pkg/store/snapshot/network"
	log "k8s.io/klog"
	"time"
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

	leadershipStore, err := leadership.NewAtomixStore()
	if err != nil {
		log.Error("Cannot load leadership atomix store ", err)
	}

	mastershipStore, err := mastership.NewAtomixStore()
	if err != nil {
		log.Error("Cannot load mastership atomix store ", err)
	}

	deviceChangesStore, err := device.NewAtomixStore()
	if err != nil {
		log.Error("Cannot load device atomix store ", err)
	}

	networkChangesStore, err := network.NewAtomixStore()
	if err != nil {
		log.Error("Cannot load network atomix store ", err)
	}
	log.Info("Network Configuration store connected")

	deviceCache, err := cache.NewCache(networkChangesStore)
	if err != nil {
		log.Error("Cannot load device cache", err)
	}

	networkSnapshotStore, err := networksnap.NewAtomixStore()
	if err != nil {
		log.Error("Cannot load network snapshot atomix store ", err)
	}

	deviceSnapshotStore, err := devicesnap.NewAtomixStore()
	if err != nil {
		log.Error("Cannot load network atomix store ", err)
	}

	deviceStore, err := devicestore.NewTopoStore(opts...)
	if err != nil {
		log.Error("Cannot load device store ", err)
	}

	mgr, err := manager.LoadManager(leadershipStore, mastershipStore, deviceChangesStore, deviceCache,
		networkChangesStore, networkSnapshotStore, deviceSnapshotStore, deviceStore)
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
	s := northbound.NewServer(northbound.NewServerConfig(caPath, keyPath, certPath))
	s.AddService(admin.Service{})
	s.AddService(diags.Service{})
	s.AddService(gnmi.Service{})

	return s.Serve(func(started string) {
		log.Info("Started NBI on ", started)
	})
}
