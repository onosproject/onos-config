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
	"os"
)

const (
	configDir  = ".onos"
	configName = "config"

	addressKey     = "service-address"
	defaultAddress = "onos-config:5150"

	tlsCertPathKey = "tls.certPath"
	tlsKeyPathKey  = "tls.keyPath"
	noTLSKey       = "no-tls"

	addressFlag     = "service-address"
	tlsCertPathFlag = "tls-cert-path"
	tlsKeyPathFlag  = "tls-key-path"
	noTLSFlag       = "no-tls"
)

var configOptions = []string{
	addressKey,
	tlsCertPathKey,
	tlsKeyPathKey,
	noTLSKey,
}

func addConfigFlags(cmd *cobra.Command) {
	viper.SetDefault(addressKey, defaultAddress)

	cmd.PersistentFlags().String(addressFlag, viper.GetString(addressKey), "the onos-config service address")
	cmd.PersistentFlags().String(tlsCertPathFlag, viper.GetString(tlsCertPathKey), "the path to the TLS certificate")
	cmd.PersistentFlags().String(tlsKeyPathFlag, viper.GetString(tlsKeyPathKey), "the path to the TLS key")
	cmd.PersistentFlags().Bool(noTLSFlag, viper.GetBool(noTLSKey), "if present, do not use TLS")
}

func getConfigCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "config {set,get,delete,init} [args]",
		Short: "Manage the CLI configuration",
	}
	cmd.AddCommand(getConfigGetCommand())
	cmd.AddCommand(getConfigSetCommand())
	cmd.AddCommand(getConfigDeleteCommand())
	cmd.AddCommand(getConfigInitCommand())
	return cmd
}

func getConfigGetCommand() *cobra.Command {
	return &cobra.Command{
		Use:       "get <key>",
		Args:      cobra.ExactArgs(1),
		ValidArgs: configOptions,
		RunE:      runConfigGetCommand,
	}
}

func runConfigGetCommand(_ *cobra.Command, args []string) error {
	value := viper.Get(args[0])
	fmt.Fprintln(GetOutput(), value)
	return nil
}

func getConfigSetCommand() *cobra.Command {
	return &cobra.Command{
		Use:       "set <key> <value>",
		Args:      cobra.ExactArgs(2),
		ValidArgs: configOptions,
		RunE:      runConfigSetCommand,
	}
}

func runConfigSetCommand(_ *cobra.Command, args []string) error {
	viper.Set(args[0], args[1])
	if err := viper.WriteConfig(); err != nil {
		return err
	}

	value := viper.Get(args[0])
	fmt.Fprintln(GetOutput(), value)
	return nil
}

func getConfigDeleteCommand() *cobra.Command {
	return &cobra.Command{
		Use:       "delete <key>",
		Args:      cobra.ExactArgs(1),
		ValidArgs: configOptions,
		RunE:      runConfigDeleteCommand,
	}
}

func runConfigDeleteCommand(_ *cobra.Command, args []string) error {
	viper.Set(args[0], nil)
	if err := viper.WriteConfig(); err != nil {
		return err
	}

	value := viper.Get(args[0])
	fmt.Fprintln(GetOutput(), value)
	return nil
}

func getConfigInitCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "init",
		Short: "Initialize the topo CLI configuration",
		RunE:  runConfigInitCommand,
	}
}

func runConfigInitCommand(_ *cobra.Command, _ []string) error {
	if err := viper.ReadInConfig(); err == nil {
		return nil
	}

	home, err := homedir.Dir()
	if err != nil {
		return err
	}

	if err := os.MkdirAll(home+"/"+configDir, 0777); err != nil {
		return err
	}

	filePath := home + "/" + configDir + "/" + configName + ".yaml"
	f, err := os.Create(filePath)
	if err != nil {
		return err
	}
	f.Close()

	if err := viper.WriteConfig(); err != nil {
		return err
	}
	fmt.Fprintln(GetOutput(), "Created "+filePath)
	return nil
}

func getAddress(cmd *cobra.Command) string {
	address, _ := cmd.Flags().GetString(addressFlag)
	if address == "" {
		return defaultAddress
	}
	return address
}

func getCertPath(cmd *cobra.Command) string {
	certPath, _ := cmd.Flags().GetString(tlsCertPathFlag)
	return certPath
}

func getKeyPath(cmd *cobra.Command) string {
	keyPath, _ := cmd.Flags().GetString(tlsKeyPathFlag)
	return keyPath
}

func noTLS(cmd *cobra.Command) bool {
	tls, _ := cmd.Flags().GetBool("no-tls")
	return tls
}

func initConfig() {
	home, err := homedir.Dir()
	if err != nil {
		panic(err)
	}

	viper.SetConfigName(configName)
	viper.AddConfigPath(home + "/" + configDir)
	viper.AddConfigPath("/etc/onos")
	viper.AddConfigPath(".")

	_ = viper.ReadInConfig()
}
