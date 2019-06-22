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
	"errors"
	"fmt"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	log "k8s.io/klog"
	"os"
	"time"
)

func init() {
	log.SetOutput(os.Stdout)
}

// GetCommand returns a Cobra command for tests in the given test registry
func GetCommand(registry *TestRegistry) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "onit {list,test,run} [tests]",
		Short: "Run onos-config integration tests on Kubernetes",
	}
	cmd.AddCommand(getSetupCommand())
	cmd.AddCommand(getTeardownCommand())
	cmd.AddCommand(getRunCommand())
	cmd.AddCommand(getGetCommand(registry))
	cmd.AddCommand(getSetCommand())
	cmd.AddCommand(getTestCommand(registry))
	return cmd
}

// getSetupCommand returns a cobra "setup" command for setting up resources
func getSetupCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "setup {cluster,simulator} [args]",
		Short: "Setup a test resource on Kubernetes",
	}
	cmd.AddCommand(getSetupClusterCommand())
	cmd.AddCommand(getSetupSimulatorCommand())
	return cmd
}

// getSetupClusterCommand returns a cobra command for deploying a test cluster
func getSetupClusterCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "cluster",
		Short: "Setup a test cluster on Kubernetes",
		Run: func(cmd *cobra.Command, args []string) {
			nodes, _ := cmd.Flags().GetInt("nodes")
			partitions, _ := cmd.Flags().GetInt("partitions")
			partitionSize, _ := cmd.Flags().GetInt("partitionSize")
			configName, _ := cmd.Flags().GetString("config")

			changeStore, err := getChangeStoreConfig(configName)
			if err != nil {
				exitError(err)
			}
			deviceStore, err := getDeviceStoreConfig(configName)
			if err != nil {
				exitError(err)
			}
			configStore, err := getConfigStoreConfig(configName)
			if err != nil {
				exitError(err)
			}
			networkStore, err := getNetworkStoreConfig(configName)
			if err != nil {
				exitError(err)
			}

			config := &TestClusterConfig{
				ChangeStore:   changeStore,
				DeviceStore:   deviceStore,
				ConfigStore:   configStore,
				NetworkStore:  networkStore,
				Nodes:         nodes,
				Partitions:    partitions,
				PartitionSize: partitionSize,
			}

			controller, err := NewTestController(config)
			if err != nil {
				exitError(err)
			}

			if err = controller.SetupCluster(); err != nil {
				exitError(err)
			} else {
				addClusterConfig(controller.TestId, config)
				setDefaultCluster(controller.TestId)
			}
		},
	}
	cmd.Flags().StringP("config", "c", "default", "test cluster configuration")
	cmd.Flags().IntP("nodes", "n", 1, "the number of onos-config nodes to deploy")
	cmd.Flags().IntP("partitions", "p", 1, "the number of Raft partitions to deploy")
	cmd.Flags().IntP("partitionSize", "s", 1, "the size of each Raft partition")
	return cmd
}

// getSetupSimulatorCommand returns a cobra command for deploying a device simulator
func getSetupSimulatorCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "simulator <name>",
		Short: "Setup a device simulator on Kubernetes",
		Args:  cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			controller, err := getDefaultController()
			if err != nil {
				exitError(err)
			}

			name := args[0]
			configName, _ := cmd.Flags().GetString("config")
			simulatorConfig, err := getDeviceConfig(configName)
			if err != nil {
				exitError(err)
			}
			config := &TestSimulatorConfig{
				Config: simulatorConfig,
			}

			if err = controller.SetupSimulator(name, config); err != nil {
				exitError(err)
			} else {
				addSimulator(controller.TestId, name, config)
			}
		},
	}
	cmd.Flags().StringP("config", "c", "default", "simulator configuration")
	return cmd
}

// getTeardownCommand returns a cobra "teardown" command for tearing down Kubernetes test resources
func getTeardownCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "teardown",
		Short: "Teardown Kubernetes test resources",
	}
	cmd.AddCommand(getTeardownClusterCommand())
	cmd.AddCommand(getTeardownSimulatorCommand())
	return cmd
}

// getTeardownClusterCommand returns a cobra "teardown" command for tearing down a test cluster
func getTeardownClusterCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "cluster",
		Short: "Teardown a test cluster on Kubernetes",
		Run: func(cmd *cobra.Command, args []string) {
			controller, err := getDefaultController()
			if err != nil {
				exitError(err)
			}

			if err = controller.TeardownCluster(); err != nil {
				exitError(err)
			} else {
				unsetDefaultCluster(controller.TestId)
				removeClusterConfig(controller.TestId)
			}
		},
	}

	defaultCluster := getDefaultCluster()
	cmd.Flags().StringP("cluster", "c", defaultCluster, "the cluster to teardown")
	viper.BindPFlag(defaultClusterKey, cmd.Flags().Lookup("cluster"))
	return cmd
}

// getTeardownSimulatorCommand returns a cobra command for tearing down a device simulator
func getTeardownSimulatorCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "simulator <name>",
		Short: "Teardown a device simulator",
		Args:  cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			controller, err := getDefaultController()
			if err != nil {
				exitError(err)
			}

			name := args[0]
			if err = controller.TeardownSimulator(name); err != nil {
				exitError(err)
			} else {
				removeSimulator(controller.TestId, name)
			}
		},
	}
}

// getGetCommand returns a cobra "get" command to read test configurations
func getGetCommand(registry *TestRegistry) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "get {cluster,clusters,configs,tests}",
		Short: "Get test configurations",
	}
	cmd.AddCommand(getGetClusterCommand())
	cmd.AddCommand(getGetClustersCommand())
	cmd.AddCommand(getGetDeviceConfigsCommand())
	cmd.AddCommand(getGetStoreConfigsCommand())
	cmd.AddCommand(getGetTestsCommand(registry))
	return cmd
}

// getGetClusterCommand returns a cobra command to get the current cluster context
func getGetClusterCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "cluster",
		Short: "Get the currently configured cluster",
		Run: func(cmd *cobra.Command, args []string) {
			defaultCluster := getDefaultCluster()
			fmt.Println(defaultCluster)
			os.Exit(0)
		},
	}
}

// getGetClustersCommand returns a cobra command to get a list of available test clusters
func getGetClustersCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "clusters",
		Short: "Get a list of all deployed clusters",
		Run: func(cmd *cobra.Command, args []string) {
			clusters := getTestConfigMap(clustersKey)
			for name, _ := range clusters {
				fmt.Println(name)
			}
		},
	}
}

// getGetDeviceConfigsCommand returns a cobra command to get a list of available device simulator configurations
func getGetDeviceConfigsCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "device-presets",
		Short: "Get a list of device configurations",
		Run: func(cmd *cobra.Command, args []string) {
			for _, name := range getDeviceConfigs() {
				fmt.Println(name)
			}
		},
	}
}

// getGetStoreConfigsCommand returns a cobra command to get a list of available store configurations
func getGetStoreConfigsCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "store-presets",
		Short: "Get a list of store configurations",
		Run: func(cmd *cobra.Command, args []string) {
			for _, name := range getStoreConfigs() {
				fmt.Println(name)
			}
		},
	}
}

// getGetTestsCommand returns a cobra command to get a list of available tests
func getGetTestsCommand(registry *TestRegistry) *cobra.Command {
	return &cobra.Command{
		Use:   "tests",
		Short: "Get a list of integration tests",
		Run: func(cmd *cobra.Command, args []string) {
			for _, name := range registry.GetNames() {
				fmt.Println(name)
			}
		},
	}
}

// getSetCommand returns a cobra "set" command for setting configurations
func getSetCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "set {cluster} [args]",
		Short: "Set test configurations",
	}
	cmd.AddCommand(getSetClusterCommand())
	return cmd
}

// getSetClusterCommand returns a cobra command for setting the cluster context
func getSetClusterCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "cluster <name>",
		Args:  cobra.ExactArgs(1),
		Short: "Set cluster context",
		Run: func(cmd *cobra.Command, args []string) {
			clusterId := args[0]
			clusters := getTestConfigMap(clustersKey)
			if _, ok := clusters[clusterId]; !ok {
				exitError(errors.New("unknown test cluster " + clusterId))
			}

			if err := setTestConfig(defaultClusterKey, clusterId); err != nil {
				exitError(err)
			} else {
				fmt.Println(clusterId)
			}
		},
	}
}

// getRunCommand returns a cobra "run" command
func getRunCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "run [tests]",
		Short: "Run integration tests on Kubernetes",
		Run: func(cmd *cobra.Command, args []string) {
			controller, err := getDefaultController()
			if err != nil {
				exitError(err)
			}

			timeout, _ := cmd.Flags().GetInt("timeout")
			message, code, err := controller.RunTests(args, time.Duration(timeout)*time.Second)
			if err != nil {
				exitError(err)
			} else {
				fmt.Println(message)
				os.Exit(code)
			}
		},
	}
	cmd.Flags().IntP("timeout", "t", 60*10, "test timeout in seconds")
	return cmd
}

// getTestCommand returns a cobra "test" command for tests in the given registry
func getTestCommand(registry *TestRegistry) *cobra.Command {
	return &cobra.Command{
		Use:   "test [tests]",
		Short: "Run integration tests",
		Run: func(cmd *cobra.Command, args []string) {
			runner := &TestRunner{
				Registry: registry,
			}
			err := runner.RunTests(args)
			if err != nil {
				exitError(err)
			} else {
				os.Exit(0)
			}
		},
	}
}

// getDefaultController returns the currently configured test controller
func getDefaultController() (*TestController, error) {
	defaultCluster := getDefaultCluster()
	if defaultCluster == "" {
		return nil, errors.New("no default cluster set")
	}
	config, err := getClusterConfig(defaultCluster)
	if err != nil {
		return nil, err
	}
	return GetTestController(defaultCluster, config)
}

// exitError prints the given err to stdout and exits with exit code 1
func exitError(err error) {
	fmt.Println(err)
	os.Exit(1)
}
