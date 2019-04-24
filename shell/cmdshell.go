package shell

import (
	"bufio"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"github.com/opennetworkinglab/onos-config/store"
	"os"
	"strconv"
	"strings"
	"time"
)

func printHelp() {
	fmt.Println("-------------------")
	fmt.Println("c) list changes")
	fmt.Println("s) Show configuration for a device")
	fmt.Println("x) Extract current config for a device")
	fmt.Println("m1) Make change 1 - set all tx-power values to 5")
	fmt.Println("m2) Make change 2 - remove leafs 2a,2b,2c add leaf 1b ")
	fmt.Println("?) show this message")
	fmt.Println("q) quit")
}

/**
 * This is a temporary utility that allows shell type access to the config
 * manager until a proper NBI is put in place
 */
func RunShell(configStore store.ConfigurationStore, changeStore store.ChangeStore) {
	reader := bufio.NewReader(os.Stdin)
	printHelp()

	for {
		fmt.Print("onos-config> ")
		text, _ := reader.ReadString('\n')
		// convert CRLF to LF
		text = strings.Replace(text, "\n", "", -1)

		switch text {
		case "c":
			changeId, err := selectChange(changeStore, reader)
			if err != nil {
				fmt.Println("Error invalid number given", err)
				continue
			}
			change := changeStore.Store[hex.EncodeToString(changeId)]
			fmt.Println("Change:", base64.StdEncoding.EncodeToString(changeId))
			fmt.Println("#\tPath\t\t\tValue\tRemove")
			for idx, c := range change.Config {
				fmt.Println(idx, "\t", c.Path, "\t", c.Value, "\t", c.Remove)
			}

		case "s":
			configId, err := selectDevice(configStore, reader)
			if err != nil {
				fmt.Println("Error invalid number given", err)
				continue
			}

			config := configStore.Store[configId]
			fmt.Println("Config\t\t Device\t\t Updated\t\t\t\t\t User\t Description")
			fmt.Println(configId, "\t", config.Device, "\t",
				config.Updated, "\t", config.User, "\t", config.Description)
			for _, change := range config.Changes {
				fmt.Println("\tChange:\t", base64.StdEncoding.EncodeToString([]byte(change)))
			}

		case "x":
			configId, err := selectDevice(configStore, reader)
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
				continue;
			}

			config := configStore.Store[configId]
			configValues := config.ExtractFullConfig(changeStore.Store, nBack)

			for _, configValue := range configValues {
				fmt.Println(configValue.Path, "\t", configValue.Value)
			}

		case "m1":
			configId, err := selectDevice(configStore, reader)
			if err != nil {
				fmt.Println("Error invalid number given", err)
				continue
			}

			config := configStore.Store[configId]
			var txPowerChange = make([]store.ChangeValue, 0)
			for _, cv := range config.ExtractFullConfig(changeStore.Store, 0) {
				if strings.Contains(cv.Path, "tx-power") {
					change, _ := store.CreateChangeValue(cv.Path, "5", false)
					txPowerChange = append(txPowerChange, change)
				}
			}

			change, err := store.CreateChange(txPowerChange, "Changing tx-power to 5")
			if err != nil {
				fmt.Println("Error creating tx-power change", err)
				continue
			}
			changeStore.Store[hex.EncodeToString(change.Id)] = change
			fmt.Println("Added change", base64.StdEncoding.EncodeToString(change.Id), "to ChangeStore (in memory)")

			config.Changes = append(config.Changes, change.Id)
			config.Updated = time.Now()
			configStore.Store[configId] = config
			fmt.Println("Added change", base64.StdEncoding.EncodeToString(change.Id),
				"to Config:", config.Name, "(in memory)")

		case "m2":
			configId, err := selectDevice(configStore, reader)
			if err != nil {
				fmt.Println("Error invalid number given", err)
				continue
			}

			changes := make([]store.ChangeValue, 0)
			c1, _ := store.CreateChangeValue("/test1:cont1a/cont2a/leaf2a", "", true)
			c2, _ := store.CreateChangeValue("/test1:cont1a/cont2a/leaf2b", "", true)
			c3, _ := store.CreateChangeValue("/test1:cont1a/cont2a/leaf2c", "", true)
			c4, _ := store.CreateChangeValue("/test1:cont1a/leaf1b", "Hello World", false)
			changes = append(changes, c1, c2, c3, c4)

			change, err := store.CreateChange(changes, "remove leafs 2a,2b,2c add leaf 1b")
			if err != nil {
				fmt.Println("Error creating m2 change", err)
				continue
			}
			changeStore.Store[hex.EncodeToString(change.Id)] = change
			fmt.Println("Added change", base64.StdEncoding.EncodeToString(change.Id), "to ChangeStore (in memory)")

			config := configStore.Store[configId]
			config.Changes = append(config.Changes, change.Id)
			config.Updated = time.Now()
			configStore.Store[configId] = config
			fmt.Println("Added change", base64.StdEncoding.EncodeToString(change.Id),
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
	var i int = 1
	configIds := make([]string, 1);
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
	var i int = 1
	changeIds := make([][]byte, 1);
	for _, c := range changeStore.Store {
		fmt.Println(i, ")\t", base64.StdEncoding.EncodeToString(c.Id), "\t", c.Description)
		changeIds = append(changeIds, c.Id)
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
