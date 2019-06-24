package runner

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/mitchellh/go-homedir"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"io/ioutil"
	"os"
	"path/filepath"
	"runtime"
)

var (
	configFile = ""
)

const (
	defaultClusterKey = "default"
	clustersKey       = "clusters"
)

var (
	_, path, _, _     = runtime.Caller(0)
	certsPath         = filepath.Join(filepath.Dir(filepath.Dir(path)), "certs")
	deviceConfigsPath = filepath.Join(filepath.Join(filepath.Dir(filepath.Dir(path)), "configs"), "device")
	storeConfigsPath  = filepath.Join(filepath.Join(filepath.Dir(filepath.Dir(path)), "configs"), "store")
)

func init() {
	cobra.OnInitialize(initConfig)
}

// getDeviceConfig gets a device configuration by name
func getDeviceConfig(name string) (map[string]interface{}, error) {
	file, err := os.Open(filepath.Join(deviceConfigsPath, name+".json"))
	if err != nil {
		return nil, err
	}
	defer file.Close()

	jsonBytes, err := ioutil.ReadAll(file)
	if err != nil {
		return nil, err
	}

	var jsonObj map[string]interface{}
	err = json.Unmarshal(jsonBytes, &jsonObj)
	return jsonObj, err
}

// getDeviceConfigs returns a list of store configurations from the configs/store directory
func getDeviceConfigs() []string {
	configs := []string{}
	filepath.Walk(deviceConfigsPath, func(path string, info os.FileInfo, err error) error {
		if info.IsDir() {
			return nil
		}

		ext := filepath.Ext(info.Name())
		if ext == ".json" {
			configs = append(configs, info.Name()[:len(info.Name())-len(ext)])
		}
		return nil
	})
	return configs
}

// getStoreConfig returns a named store configuration from the given stores configuration
func getStoreConfig(configName string, storeType string) (map[string]interface{}, error) {
	file, err := os.Open(filepath.Join(storeConfigsPath, configName+".json"))
	if err != nil {
		return nil, err
	}
	defer file.Close()

	jsonBytes, err := ioutil.ReadAll(file)
	if err != nil {
		return nil, err
	}

	var jsonObj map[string]interface{}
	err = json.Unmarshal(jsonBytes, &jsonObj)
	if err != nil {
		return nil, err
	}

	storeObj, ok := jsonObj[storeType]
	if !ok {
		return nil, errors.New("malformed store configuration: " + storeType + " key not found")
	}

	return storeObj.(map[string]interface{}), nil
}

// getChangeStoreConfig returns the change store from the given stores configuration
func getChangeStoreConfig(name string) (map[string]interface{}, error) {
	return getStoreConfig(name, "changeStore")
}

// getConfigStoreConfig returns the config store from the given stores configuration
func getConfigStoreConfig(name string) (map[string]interface{}, error) {
	return getStoreConfig(name, "configStore")
}

// getNetworkStoreConfig returns the network store from the given stores configuration
func getNetworkStoreConfig(name string) (map[string]interface{}, error) {
	return getStoreConfig(name, "networkStore")
}

// getDeviceStoreConfig returns the device store from the given stores configuration
func getDeviceStoreConfig(name string) (map[string]interface{}, error) {
	return getStoreConfig(name, "deviceStore")
}

// getStoreConfigs returns a list of store configurations from the configs/store directory
func getStoreConfigs() []string {
	configs := []string{}
	filepath.Walk(storeConfigsPath, func(path string, info os.FileInfo, err error) error {
		if info.IsDir() {
			return nil
		}

		ext := filepath.Ext(info.Name())
		if ext == ".json" {
			configs = append(configs, info.Name()[:len(info.Name())-len(ext)])
		}
		return nil
	})
	return configs
}

func addClusterConfig(name string, config *TestClusterConfig) error {
	return setTestConfig(fmt.Sprintf("%s.%s", clustersKey, name), config)
}

func getClusterConfig(name string) (*TestClusterConfig, error) {
	config := &TestClusterConfig{}
	if err := viper.UnmarshalKey(fmt.Sprintf("%s.%s", clustersKey, name), config); err != nil {
		return nil, err
	}
	return config, nil
}

func removeClusterConfig(name string) error {
	return unsetTestConfig(fmt.Sprintf("%s.%s", clustersKey, name))
}

func setDefaultCluster(name string) error {
	return setTestConfig(defaultClusterKey, name)
}

func unsetDefaultCluster(name string) error {
	return setTestConfig(defaultClusterKey, "")
}

func addSimulator(cluster string, name string, config *TestSimulatorConfig) error {
	return setTestConfig(fmt.Sprintf("%s.%s.simulators.%s", clustersKey, cluster, name), config)
}

func removeSimulator(cluster string, name string) error {
	return setTestConfig(fmt.Sprintf("%s.%s.simulators.%s", clustersKey, cluster, name), nil)
}

func getDefaultCluster() string {
	return getTestConfig(defaultClusterKey).(string)
}

func setTestConfig(key string, value interface{}) error {
	viper.Set(key, value)
	return viper.WriteConfig()
}

func getTestConfig(key string) interface{} {
	return viper.GetString(key)
}

func unsetTestConfig(key string) error {
	viper.Set(key, struct{}{})
	return viper.WriteConfig()
}

func getTestConfigMap(key string) map[string]interface{} {
	return viper.GetStringMap(key)
}

func initConfig() {
	if configFile != "" {
		viper.SetConfigFile(configFile)
	} else {
		home, err := homedir.Dir()
		if err != nil {
			exitError(err)
		}

		viper.SetConfigName("onit")
		viper.AddConfigPath(home + "/.onos")
		viper.AddConfigPath("/etc/onos")
		viper.AddConfigPath(".")
	}

	// If the configuration file is not found, initialize a configuration in the home dir.
	if err := viper.ReadInConfig(); err != nil {
		if _, ok := err.(*viper.ConfigFileNotFoundError); !ok {
			home, err := homedir.Dir()
			if err != nil {
				exitError(err)
			}

			err = os.MkdirAll(home+"/.onos", 0777)
			if err != nil {
				exitError(err)
			}

			f, err := os.Create(home + "/.onos/onit.yaml")
			if err != nil {
				exitError(err)
			} else {
				f.Close()
			}

			err = viper.WriteConfig()
			if err != nil {
				exitError(err)
			}
		} else {
			exitError(err)
		}
	}
}
