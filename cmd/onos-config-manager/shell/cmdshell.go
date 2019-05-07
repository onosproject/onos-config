// Copyright 2019-present Open Networking Foundation
//
// Licensed under the Apache License, Configuration 2.0 (the "License");
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

// Package shell is a temporary minimal shell interface to the config manager
package shell

import (
	"bufio"
	"fmt"
	"github.com/onosproject/onos-config/pkg/events"
	"github.com/onosproject/onos-config/pkg/southbound/topocache"
	"github.com/onosproject/onos-config/pkg/store"
	"github.com/onosproject/onos-config/pkg/store/change"
	"html/template"
	"os"
	"strconv"
	"strings"
	"time"
)

const deviceListTemplate = "Addr\t\t\tTarget\t\t\tTimeout\n{{ range . }}" +
	"{{printf \"%-24s\" .Addr}}{{printf \"%-24s\" .Target}}{{.Timeout}}\n{{end}}"

const configListTemplate = "#\tPath\t\t\t\t\tValue\t\tRemove\n{{ range $i, $e := .}}" +
	"{{$i}}\t{{printf \"%-40s\" .Path}}{{printf \"%-16s\" .Value}}\t{{.Remove}}\n{{end}}"

const networkConfigHeaderTemplate = "Name\tCreated\n"

const networkConfigListTemplate = "{{.Name}}\t{{.Created}}\n" +
	//"{{.ConfigurationChanges}}"
	"{{ range $k, $v := .ConfigurationChanges}}" +
	"\t{{printf \"%-20s\" $k}}\t{{printf \"%x\" $v}}\n" +
	"{{end}}"

func printHelp() {
	fmt.Println("-------------------")
	fmt.Println("c) list changes")
	fmt.Println("d) list connected devices")
	fmt.Println("n) list network changes")
	fmt.Println("s) Show configuration for a device")
	fmt.Println("x) Extract current config for a device")
	fmt.Println("m1) Make change 1 - set all tx-power values to 5")
	fmt.Println("m2) Make change 2 - remove leafs 2a,2b,2c add leaf 1b ")
	fmt.Println("m3) Make a change to timezone on openconfig device")
	fmt.Println("?) show this message")
	fmt.Println("q) quit")
}

// RunShell is a temporary utility that allows shell type access to the config
// manager until a proper NBI is put in place
func RunShell(configStore store.ConfigurationStore, changeStore store.ChangeStore,
	deviceStore *topocache.DeviceStore, networkStore *store.NetworkStore,
	changesChannel chan events.Event) {
	reader := bufio.NewReader(os.Stdin)
	printHelp()

	for {
		fmt.Print("onos-config> ")
		text, _ := reader.ReadString('\n')
		// convert CRLF to LF
		text = strings.Replace(text, "\n", "", -1)

		switch text {
		case "c":
			changeID, err := selectChange(changeStore, reader)
			if err != nil {
				fmt.Println("Error invalid number given", err)
				continue
			}
			change := changeStore.Store[store.B64(changeID)]
			t, err := template.New("configlist.txt").Parse(configListTemplate)
			if err != nil {
				fmt.Println("Error parsing template", err)
				continue
			}
			fmt.Println("Change:", store.B64(changeID))
			t.Execute(os.Stdout, change.Config)

		case "d":
			t, err := template.New("devicelist.txt").Parse(deviceListTemplate)
			if err != nil {
				fmt.Println("Error parsing template", deviceListTemplate)
				continue
			}
			err = t.Execute(os.Stdout, deviceStore.Store)

		case "n":
			t1, err := template.New("networkconfig.txt").Parse(networkConfigHeaderTemplate)
			if err != nil {
				fmt.Println("Error parsing template", networkConfigHeaderTemplate)
				continue
			}
			t2, err := template.New("networklist.txt").Parse(networkConfigListTemplate)
			if err != nil {
				fmt.Println("Error parsing template", networkConfigListTemplate)
				continue
			}
			err = t1.Execute(os.Stdout, networkStore.Store)
			for _, nw := range networkStore.Store {
				err = t2.Execute(os.Stdout, nw)
			}

		case "s":
			configID, err := selectDevice(configStore, reader)
			if err != nil {
				fmt.Println("Error invalid number given", err)
				continue
			}

			config := configStore.Store[configID]
			fmt.Println("Config\t\t Device\t\t Updated\t\t\t User\t Description")
			fmt.Println(configID, "\t", config.Device, "\t",
				config.Updated.Format(time.RFC3339), "\t", config.User, "\t", config.Description)
			for _, changeID := range config.Changes {
				change := changeStore.Store[store.B64(changeID)]
				fmt.Println("\tChange:\t", store.B64(changeID),
					"\t", change.Description)
			}

		case "x":
			configID, err := selectDevice(configStore, reader)
			if err != nil {
				fmt.Println("Error invalid number given", err)
				continue
			}
			fmt.Println("Number of changes to go back (default 0 - the latest)")
			nBackStr, _ := reader.ReadString('\n')
			nBackStr = strings.Replace(nBackStr, "\n", "", -1)
			nBack, err := strconv.Atoi(nBackStr)
			if err != nil && nBackStr != "" {
				fmt.Println("Error invalid number given", nBackStr)
				continue
			}

			config := configStore.Store[configID]
			configValues := config.ExtractFullConfig(changeStore.Store, nBack)

			for _, configValue := range configValues {
				fmt.Println(configValue.Path, "\t", configValue.Value)
			}
		case "m1":
			configID, err := selectDevice(configStore, reader)
			if err != nil {
				fmt.Println("Error invalid number given", err)
				continue
			}

			config := configStore.Store[configID]
			var txPowerChange = make([]*change.Value, 0)
			for _, cv := range config.ExtractFullConfig(changeStore.Store, 0) {
				if strings.Contains(cv.Path, "tx-power") {
					change, _ := change.CreateChangeValue(cv.Path, "5", false)
					txPowerChange = append(txPowerChange, change)
				}
			}

			change, err := change.CreateChange(txPowerChange, "Changing tx-power to 5")
			if err != nil {
				fmt.Println("Error creating tx-power change", err)
				continue
			}
			changeStore.Store[store.B64(change.ID)] = change
			fmt.Println("Added change", store.B64(change.ID), "to ChangeStore (in memory)")

			config.Changes = append(config.Changes, change.ID)
			config.Updated = time.Now()
			configStore.Store[configID] = config
			eventValues := make(map[string]string)
			eventValues[events.ChangeID] = store.B64(change.ID)
			eventValues[events.Committed] = "true"
			changesChannel <- events.CreateEvent(config.Device,
				events.EventTypeConfiguration, eventValues)
			fmt.Println("Added change", store.B64(change.ID),
				"to Config:", config.Name, "(in memory)")

		case "m2":
			configID, err := selectDevice(configStore, reader)
			if err != nil {
				fmt.Println("Error invalid number given", err)
				continue
			}

			changes := make([]*change.Value, 0)
			c1, _ := change.CreateChangeValue("/test1:cont1a/cont2a/leaf2a", "", true)
			c2, _ := change.CreateChangeValue("/test1:cont1a/cont2a/leaf2b", "", true)
			c3, _ := change.CreateChangeValue("/test1:cont1a/cont2a/leaf2c", "", true)
			c4, _ := change.CreateChangeValue("/test1:cont1a/leaf1b", "Hello World", false)
			changes = append(changes, c1, c2, c3, c4)

			change, err := change.CreateChange(changes, "remove leafs 2a,2b,2c add leaf 1b")
			if err != nil {
				fmt.Println("Error creating m2 change", err)
				continue
			}
			changeStore.Store[store.B64(change.ID)] = change
			fmt.Println("Added change", store.B64(change.ID), "to ChangeStore (in memory)")

			config := configStore.Store[configID]
			config.Changes = append(config.Changes, change.ID)
			config.Updated = time.Now()
			configStore.Store[configID] = config

			eventValues := make(map[string]string)
			eventValues[events.ChangeID] = store.B64(change.ID)
			eventValues[events.Committed] = "true"
			changesChannel <- events.CreateEvent(config.Device,
				events.EventTypeConfiguration, eventValues)

			fmt.Println("Added change", store.B64(change.ID),
				"to Config:", config.Name, "(in memory)")

		case "m3":
			configID, err := selectDevice(configStore, reader)
			if err != nil {
				fmt.Println("Error invalid number given", err)
				continue
			}
			changes := make([]*change.Value, 0)
			c1, _ := change.CreateChangeValue("/system/clock/config/timezone-name", "Europe/Milan", false)
			changes = append(changes, c1)
			change, err := change.CreateChange(changes, "Chanage timezone")
			if err != nil {
				fmt.Println("Error creating m3 change", err)
				continue
			}
			changeStore.Store[store.B64(change.ID)] = change
			fmt.Println("Added change", store.B64(change.ID), "to ChangeStore (in memory)")

			config := configStore.Store[configID]
			config.Changes = append(config.Changes, change.ID)
			config.Updated = time.Now()
			configStore.Store[configID] = config

			eventValues := make(map[string]string)
			eventValues[events.ChangeID] = store.B64(change.ID)
			eventValues[events.Committed] = "true"
			changesChannel <- events.CreateEvent(config.Device,
				events.EventTypeConfiguration, eventValues)

			fmt.Println("Added change", store.B64(change.ID),
				"to Config:", config.Name, "(in memory)")

		case "?":
			printHelp()

		case "q":
			return

		case "":
			continue

		default:
			fmt.Println("Unexpected command", text)

		}
	}
}

func selectDevice(configStore store.ConfigurationStore, reader *bufio.Reader) (string, error) {
	fmt.Println("Select Device")
	var i = 1
	configIds := make([]string, 1)
	for _, c := range configStore.Store {
		fmt.Println(i, ")\t", c.Device)
		configIds = append(configIds, c.Name)
		i++
	}
	fmt.Printf(">>([1]-%d\n", len(configStore.Store))
	deviceNumStr, _ := reader.ReadString('\n')
	deviceNumStr = strings.Replace(deviceNumStr, "\n", "", -1)
	if deviceNumStr == "" {
		deviceNumStr = "1"
	}
	deviceNum, err := strconv.Atoi(deviceNumStr)
	if err != nil {
		fmt.Println("Error invalid number given", deviceNumStr)
		return "", err
	}

	return configIds[deviceNum], nil
}

func selectChange(changeStore store.ChangeStore, reader *bufio.Reader) ([]byte, error) {
	fmt.Println("Select Change")
	var i = 1
	changeIds := make([][]byte, 1)
	for _, c := range changeStore.Store {
		fmt.Println(i, ")\t", store.B64(c.ID), "\t", c.Description)
		changeIds = append(changeIds, c.ID)
		i++
	}
	fmt.Printf(">>([1]-%d\n", len(changeStore.Store))
	changeNumStr, _ := reader.ReadString('\n')
	changeNumStr = strings.Replace(changeNumStr, "\n", "", -1)
	if changeNumStr == "" {
		changeNumStr = "1"
	}
	changeNum, err := strconv.Atoi(changeNumStr)
	if err != nil {
		fmt.Println("Error invalid number given", changeNumStr)
		return make([]byte, 0), err
	}

	return changeIds[changeNum], nil
}
