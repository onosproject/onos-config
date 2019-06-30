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

package cli

import (
	"github.com/mitchellh/go-homedir"
	"github.com/spf13/viper"
	"os"
	"path/filepath"
	"runtime"
)

var (
	_, path, _, _     = runtime.Caller(0)
	deviceConfigsPath = filepath.Join(filepath.Join(filepath.Dir(filepath.Dir(path)), "configs"), "device")
	storeConfigsPath  = filepath.Join(filepath.Join(filepath.Dir(filepath.Dir(path)), "configs"), "store")
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
