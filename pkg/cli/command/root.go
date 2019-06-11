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

// Package command holds ONOS command-line command implementations.
package command

import (
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

// GetRootCommand returns the root CLI command.
func GetRootCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "onos",
		Short: "ONOS command line client",
	}

	viper.SetDefault("controller", ":5150")

	cmd.PersistentFlags().StringP("controller", "c", viper.GetString("controller"), "the controller address")
	cmd.PersistentFlags().String("config", "", "config file (default: $HOME/.onos/config.yaml)")

	viper.BindPFlag("controller", cmd.PersistentFlags().Lookup("controller"))

	cmd.AddCommand(newDevicesCommand())
	return cmd
}
