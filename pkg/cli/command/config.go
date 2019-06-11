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

package command

import (
	"github.com/mitchellh/go-homedir"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"os"
)

var (
	configFile = ""
	validArgs  = []string{
		"address",
		"keyPath",
		"certPath",
	}
)

func init() {
	cobra.OnInitialize(initConfig)
}

func newConfigCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "config <subcommand>",
		Short: "Read and update CLI configuration options",
	}
	cmd.AddCommand(newConfigGetCommand())
	cmd.AddCommand(newConfigSetCommand())
	cmd.AddCommand(newConfigDeleteCommand())
	return cmd
}

func newConfigGetCommand() *cobra.Command {
	return &cobra.Command{
		Use:       "get <key>",
		Args:      cobra.ExactArgs(1),
		ValidArgs: validArgs,
		Run:       runConfigGetCommand,
	}
}

func runConfigGetCommand(cmd *cobra.Command, args []string) {
	value := viper.Get(args[0])
	ExitWithOutput("%s\n", value)
}

func newConfigSetCommand() *cobra.Command {
	return &cobra.Command{
		Use:       "set <key> <value>",
		Args:      cobra.ExactArgs(2),
		ValidArgs: validArgs,
		Run:       runConfigSetCommand,
	}
}

func runConfigSetCommand(cmd *cobra.Command, args []string) {
	viper.Set(args[0], args[1])
	if err := viper.WriteConfig(); err != nil {
		ExitWithError(ExitError, err)
	} else {
		value := viper.Get(args[0])
		ExitWithOutput("%s\n", value)
	}
}

func newConfigDeleteCommand() *cobra.Command {
	validArgs := []string{
		"controller",
		"namespace",
		"group",
		"app",
	}
	return &cobra.Command{
		Use:       "delete <key>",
		Args:      cobra.ExactArgs(1),
		ValidArgs: validArgs,
		Run:       runConfigDeleteCommand,
	}
}

func runConfigDeleteCommand(cmd *cobra.Command, args []string) {
	viper.Set(args[0], nil)
	if err := viper.WriteConfig(); err != nil {
		ExitWithError(ExitError, err)
	} else {
		value := viper.Get(args[0])
		ExitWithOutput("%s\n", value)
	}
}

func newInitCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "init",
		Short: "Initialize the ONOS CLI configuration",
		Run: func(cmd *cobra.Command, args []string) {
			if err := viper.ReadInConfig(); err == nil {
				ExitWithSuccess()
			}

			home, err := homedir.Dir()
			if err != nil {
				ExitWithError(ExitError, err)
			}

			err = os.MkdirAll(home+"/.onos", 0777)
			if err != nil {
				ExitWithError(ExitError, err)
			}

			f, err := os.Create(home + "/.onos/config.yaml")
			if err != nil {
				ExitWithError(ExitError, err)
			} else {
				f.Close()
			}

			err = viper.WriteConfig()
			if err != nil {
				ExitWithError(ExitError, err)
			} else {
				ExitWithOutput("Created ~/.onos/config.yaml\n")
			}
		},
	}
}

func setConfig(key string, value string) error {
	viper.Set(key, value)
	return viper.WriteConfig()
}

func getConfig(key string) string {
	return viper.GetString(key)
}

func initConfig() {
	if configFile != "" {
		viper.SetConfigFile(configFile)
	} else {
		home, err := homedir.Dir()
		if err != nil {
			ExitWithError(ExitError, err)
		}

		viper.SetConfigName("config")
		viper.AddConfigPath(home + "/.onos")
		viper.AddConfigPath("/etc/onos")
		viper.AddConfigPath(".")
	}

	viper.ReadInConfig()
}
