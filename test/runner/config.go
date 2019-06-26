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

package runner

import (
	"encoding/json"
	"github.com/gofrs/flock"
	"github.com/mitchellh/go-homedir"
	"github.com/spf13/viper"
	"io/ioutil"
	"os"
	"path/filepath"
	"runtime"
)

var (
	_, path, _, _     = runtime.Caller(0)
	certsPath         = filepath.Join(filepath.Dir(filepath.Dir(path)), "certs")
	deviceConfigsPath = filepath.Join(filepath.Join(filepath.Dir(filepath.Dir(path)), "configs"), "device")
	storeConfigsPath  = filepath.Join(filepath.Join(filepath.Dir(filepath.Dir(path)), "configs"), "store")
)

var (
	configLock *flock.Flock
)

// getSimulatorPresets returns a list of store configurations from the configs/store directory
func getSimulatorPresets() []string {
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

// getStorePresets returns a list of store configurations from the configs/store directory
func getStorePresets() []string {
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

// setDefaultCluster sets the default cluster
func setDefaultCluster(clusterID string) error {
	if err := initConfig(); err != nil {
		return err
	}
	viper.Set("cluster", clusterID)
	return viper.WriteConfig()
}

// getDefaultCluster returns the default cluster
func getDefaultCluster() string {
	return viper.GetString("cluster")
}

// ClusterConfig provides the configuration for the Kubernetes test cluster
type ClusterConfig struct {
	Preset        string `yaml:"preset" mapstructure:"preset"`
	Nodes         int    `yaml:"nodes" mapstructure:"nodes"`
	Partitions    int    `yaml:"partitions" mapstructure:"partitions"`
	PartitionSize int    `yaml:"partitionSize" mapstructure:"partitionSize"`
}

// load loads the preset configuration for the cluster
func (c *ClusterConfig) load() (map[string]interface{}, error) {
	file, err := os.Open(filepath.Join(storeConfigsPath, c.Preset+".json"))
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
	return jsonObj, nil
}

// SimulatorConfig provides the configuration for a device simulator
type SimulatorConfig struct {
	Config string `yaml:"config" mapstructure:"config"`
}

// load loads the simulator configuration
func (c *SimulatorConfig) load() (map[string]interface{}, error) {
	file, err := os.Open(filepath.Join(deviceConfigsPath, c.Config+".json"))
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

func initConfig() error {
	// If the configuration file is not found, initialize a configuration in the home dir.
	if err := viper.ReadInConfig(); err != nil {
		if _, ok := err.(*viper.ConfigFileNotFoundError); !ok {
			home, err := homedir.Dir()
			if err != nil {
				return err
			}

			err = os.MkdirAll(home+"/.onos", 0777)
			if err != nil {
				return err
			}

			f, err := os.Create(home + "/.onos/onit.yaml")
			if err != nil {
				return err
			}
			f.Close()
		} else {
			return err
		}
	}
	return nil
}

func init() {
	home, err := homedir.Dir()
	if err != nil {
		exitError(err)
	}

	viper.SetConfigName("onit")
	viper.AddConfigPath(home + "/.onos")
	viper.AddConfigPath("/etc/onos")
	viper.AddConfigPath(".")

	viper.ReadInConfig()
}
