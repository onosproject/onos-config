/*
 * Copyright 2019-present Open Networking Foundation
 *
 * Licensed under the Apache License, Configuration 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */
package main

import (
	"flag"
	"fmt"
	"github.com/opennetworkinglab/onos-config/shell"
	"github.com/opennetworkinglab/onos-config/store"
	"log"
	"log/syslog"
	"os"
)

const (
	CONFIG_STORE_FILE = "stores/configStore-sample.json"
	CHANGE_STORE_FILE = "stores/changeStore-sample.json"
)
var (
	configStore store.ConfigurationStore
	changeStore store.ChangeStore
)


// This is purely experimental for the moment and is a placeholder for bigger
// things to come
func main() {
	configStoreFile := flag.String("configStore", CONFIG_STORE_FILE,
			"path to config store file (default=" + CONFIG_STORE_FILE + ")")
	changeStoreFile := 	flag.String("changeStore", CHANGE_STORE_FILE,
			"path to change store file (default=" + CHANGE_STORE_FILE + ")")

	flag.Parse()
	var err error;
	sysLog, err := syslog.New(syslog.LOG_INFO|syslog.LOG_LOCAL7, "onos-config-manager")

	if err != nil {
		log.Fatal(err)
	} else {
		log.SetOutput(sysLog)
	}
	log.Println("onos-config-manager started")

	fmt.Println("Welcome to onos-config-manager. Messages are logged to syslog")


	configStore, err = store.LoadConfigStore(*configStoreFile)
	if err != nil {
		log.Fatal("Cannot load config store ", err)
	}
	log.Println("Configuration store loaded from", *configStoreFile)

	changeStore, err = store.LoadChangeStore(*changeStoreFile)
	if err != nil {
		log.Fatal("Cannot load change store ", err)
	}
	log.Println("Change store loaded from", *changeStoreFile)

	// Run a shell as a temporary solution to not having an NBI
	shell.RunShell(configStore, changeStore)

	log.Println("Shutting down")
	os.Exit(0);
}
