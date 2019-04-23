package shell

import (
	"bufio"
	"encoding/base64"
	"fmt"
	"github.com/opennetworkinglab/onos-config/store"
	"os"
	"strconv"
	"strings"
)

/**
 * This is a temporary utility that allows shell type access to the config
 * manager until a proper NBI is put in place
 */
func RunShell(configStore store.ConfigurationStore, changeStore store.ChangeStore) {
	reader := bufio.NewReader(os.Stdin)
	fmt.Println("-------------------")
	fmt.Println("c) list changes")
	fmt.Println("l) List configurations")
	fmt.Println("s) Show config details for a device")
	fmt.Println("x) Extract current config for a device")
	fmt.Println("q) quit")

	for {
		fmt.Print("onos-config> ")
		text, _ := reader.ReadString('\n')
		// convert CRLF to LF
		text = strings.Replace(text, "\n", "", -1)

		if strings.Compare("c", text) == 0 {
			fmt.Println("Change\tId\t# changes\tDescription")
			for idx, c := range changeStore.Store {
				fmt.Println(idx, "\t", len(c.Config), "\t", c.Description)
			}
		}

		if strings.Compare("l", text) == 0 {
			fmt.Println("Config\tDevice\tDescription")
			for idx, c := range configStore.Store {
				fmt.Println(idx, "\t", c.Device, "\t", c.Description)
			}
		}

		if strings.Compare("s", text) == 0 {
			configIdMap, deviceNum, err := selectDevice(configStore, reader)
			if err != nil {
				fmt.Println("Error invalid number given", err)
				continue
			}

			config := configStore.Store[configIdMap[deviceNum]]
			fmt.Println("Config\t\t Device\t\t Updated\t\t\t\t\t User\t Description")
			fmt.Println(configIdMap[deviceNum], "\t", config.Device, "\t",
				config.Created, "\t", config.User, "\t", config.Description)
			for _, change := range config.Changes {
				fmt.Println("\tChange:\t", base64.StdEncoding.EncodeToString([]byte(change)))
			}
		}

		if strings.Compare("x", text) == 0 {
			configIdMap, deviceNum, err := selectDevice(configStore, reader)
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

			config := configStore.Store[configIdMap[deviceNum]]
			configValues := config.ExtractFullConfig(changeStore.Store, nBack)

			for _, configValue := range configValues {
				fmt.Println(configValue.Path, "\t", configValue.Value)
			}
		}

		if strings.Compare("q", text) == 0 {
			break;
		}
	}
}

func selectDevice(configStore store.ConfigurationStore, reader *bufio.Reader) (map[int]string, int, error) {
	fmt.Println("Select Device")
	var i int = 1
	configIdMap := make(map[int]string);
	for _, c := range configStore.Store {
		fmt.Println(i, ")\t", c.Device)
		configIdMap[i] = c.Name
		i++
	}
	fmt.Printf(">>([1]-%d\n", len(configStore.Store))
	deviceNumStr, _ := reader.ReadString('\n')
	deviceNumStr = strings.Replace(deviceNumStr, "\n", "", -1)
	deviceNum, err := strconv.Atoi(deviceNumStr)
	if err != nil {
		fmt.Println("Error invalid number given", deviceNumStr)
		return make(map[int]string), -1, err
	}

	return configIdMap, deviceNum, nil
}
