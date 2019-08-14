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
	"fmt"
	"github.com/mitchellh/go-homedir"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"os"
)

// getConfigCommand returns a config configuration command
func getConfigCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "config",
		Short: "Read and update the config configuration",
	}
	cmd.AddCommand(getConfigGetCommand())
	cmd.AddCommand(getConfigSetCommand())
	return cmd
}

// getConfigGetCommand returns a config configuration get command
func getConfigGetCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "get <key>",
		Args:  cobra.ExactArgs(1),
		Short: "Get a configuration value",
		RunE:  runConfigGetCommand,
	}
}

func runConfigGetCommand(cmd *cobra.Command, args []string) error {
	value := getConfig(args[0])
	if value != nil {
		fmt.Printf("%+v", value)
	}
	return nil
}

// getConfigSetCommand returns a config configuration set command
func getConfigSetCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "set <key> <value>",
		Args:  cobra.ExactArgs(2),
		Short: "Set a configuration key/value pair",
		RunE:  runConfigSetCommand,
	}
}

func runConfigSetCommand(cmd *cobra.Command, args []string) error {
	return setConfig(args[0], args[1])
}

// setConfig sets a configuration key/value
func setConfig(key string, value interface{}) error {
	if err := initConfig(); err != nil {
		return err
	}
	viper.Set(key, value)
	return viper.WriteConfig()
}

// getConfig gets a configuration value
func getConfig(key string) interface{} {
	return viper.Get(key)
}

// getConfigString gets a configuration value as a string
func getConfigString(key string) string {
	return viper.GetString(key)
}

// getConfigOrDefault gets a configuration value or returns the default if the configuration is not set
func getConfigOrDefault(key string, def interface{}) interface{} {
	value := getConfig(key)
	if value != nil {
		return value
	}
	return def
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

			f, err := os.Create(home + "/.onos/config.yaml")
			if err != nil {
				return err
			}
			defer f.Close()
		} else {
			return err
		}
	}
	return nil
}

func init() {
	home, err := homedir.Dir()
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	viper.SetConfigName("config")
	viper.AddConfigPath(home + "/.onos")
	viper.AddConfigPath("/etc/onos")
	viper.AddConfigPath(".")

	// Read in the configuration and ignore the error if the configuration file is not found.
	if err = viper.ReadInConfig(); err != nil {
		if _, ok := err.(*viper.ConfigFileNotFoundError); ok {
			fmt.Println(err)
			os.Exit(1)
		}
	}
}
