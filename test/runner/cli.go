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
	"fmt"
	"github.com/google/uuid"
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

			// Load the onit configuration from disk
			config, err := LoadConfig()
			if err != nil {
				exitError(err)
			}

			// Load store presets from the given configuration name
			changeStore, err := getChangeStorePreset(configName)
			if err != nil {
				exitError(err)
			}
			deviceStore, err := getDeviceStorePreset(configName)
			if err != nil {
				exitError(err)
			}
			configStore, err := getConfigStorePreset(configName)
			if err != nil {
				exitError(err)
			}
			networkStore, err := getNetworkStorePreset(configName)
			if err != nil {
				exitError(err)
			}

			// Create the cluster configuration
			cluster := &ClusterConfig{
				ChangeStore:   changeStore,
				DeviceStore:   deviceStore,
				ConfigStore:   configStore,
				NetworkStore:  networkStore,
				Nodes:         nodes,
				Partitions:    partitions,
				PartitionSize: partitionSize,
			}

			// Create a new controller for the test cluster
			controller, err := NewTestController(cluster)
			if err != nil {
				exitError(err)
			}

			// Setup the cluster and update the onit configuration
			if err = controller.SetupCluster(); err != nil {
				exitError(err)
			} else {
				// Acquire a lock on the configuration
				if err := config.Lock(); err != nil {
					exitError(err)
				}

				// Release the configuration lock once complete
				defer config.Unlock()

				// Add the cluster to the configuration and update the default cluster
				config.Clusters[controller.ClusterId] = cluster
				config.DefaultCluster = controller.ClusterId

				// Write the configuration and exit
				if err = config.Write(); err != nil {
					exitError(err)
				}
				fmt.Println(controller.ClusterId)
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
		Use:   "simulator [name]",
		Short: "Setup a device simulator on Kubernetes",
		Args:  cobra.MaximumNArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			// Load the onit configuration from disk
			config, err := LoadConfig()
			if err != nil {
				exitError(err)
			}

			// Get the default cluster ID
			clusterId, err := config.getDefaultCluster()
			if err != nil {
				exitError(err)
			}

			// Fail if the default cluster configuration cannot be found
			cluster, err := config.getClusterConfig(clusterId)
			if err != nil {
				exitError(err)
			}

			// Get the test cluster controller from the cluster configuration
			controller, err := GetTestController(clusterId, cluster)
			if err != nil {
				exitError(err)
			}

			// If a simulator name was not provided, generate one from a UUID.
			var name string
			if len(args) > 0 {
				name = args[0]
			} else {
				id, err := uuid.NewUUID()
				if err != nil {
					exitError(err)
				}
				name = fmt.Sprintf("device-%s", id.String())
			}

			// Create the simulator configuration from the configured preset
			configName, _ := cmd.Flags().GetString("config")
			simulatorConfig, err := getSimulatorPreset(configName)
			if err != nil {
				exitError(err)
			}
			simulator := &SimulatorConfig{
				Config: simulatorConfig,
			}

			if err = controller.SetupSimulator(name, simulator); err != nil {
				exitError(err)
			} else {
				// Acquire a lock on the configuration
				if err := config.Lock(); err != nil {
					exitError(err)
				}

				// Release the configuration lock once complete
				defer config.Unlock()

				// Fail if the cluster does not exist
				cluster, err := config.getClusterConfig(clusterId)
				if err != nil {
					exitError(err)
				}

				// Fail if the simulator does not exist
				_, err = cluster.getSimulator(name)
				if err != nil {
					exitError(err)
				}

				// Add the simulator to the configuration and exit
				cluster.Simulators[name] = simulator
				if err = config.Write(); err != nil {
					exitError(err)
				}

				fmt.Println(name)
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
			// Load the onit configuration from disk
			config, err := LoadConfig()
			if err != nil {
				exitError(err)
			}

			// Get the default cluster ID
			clusterId, err := config.getDefaultCluster()
			if err != nil {
				exitError(err)
			}

			// Fail if the default cluster configuration cannot be found
			cluster, err := config.getClusterConfig(clusterId)
			if err != nil {
				exitError(err)
			}

			// Get the test cluster controller from the cluster configuration
			controller, err := GetTestController(clusterId, cluster)
			if err != nil {
				exitError(err)
			}

			if err = controller.TeardownCluster(); err != nil {
				exitError(err)
			} else {
				// Acquire a lock on the configuration
				if err := config.Lock(); err != nil {
					exitError(err)
				}

				// Release the configuration lock once complete
				defer config.Unlock()

				// Fail if the cluster does not exist
				_, err := config.getClusterConfig(clusterId)
				if err != nil {
					exitError(err)
				}

				// Remove the cluster from the clusters list
				delete(config.Clusters, clusterId)

				// If the cluster is the default cluster, unset the default
				if config.DefaultCluster == clusterId {
					config.DefaultCluster = ""
				}

				// Write the configuration and exit
				if err = config.Write(); err != nil {
					exitError(err)
				}
			}
		},
	}

	// Load the onit configuration from disk
	config, err := LoadConfig()
	if err != nil {
		exitError(err)
	}
	cmd.Flags().StringP("cluster", "c", config.DefaultCluster, "the cluster to teardown")
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
			// Load the onit configuration from disk
			config, err := LoadConfig()
			if err != nil {
				exitError(err)
			}

			// Get the default cluster ID
			clusterId, err := config.getDefaultCluster()
			if err != nil {
				exitError(err)
			}

			// Fail if the default cluster configuration cannot be found
			cluster, err := config.getClusterConfig(clusterId)
			if err != nil {
				exitError(err)
			}

			// Get the test cluster controller from the cluster configuration
			controller, err := GetTestController(clusterId, cluster)
			if err != nil {
				exitError(err)
			}

			name := args[0]
			if err = controller.TeardownSimulator(name); err != nil {
				exitError(err)
			} else {
				// Acquire a lock on the configuration
				if err := config.Lock(); err != nil {
					exitError(err)
				}

				// Release the configuration lock once complete
				defer config.Unlock()

				// Fail if the cluster does not exist
				cluster, err = config.getClusterConfig(clusterId)
				if err != nil {
					exitError(err)
				}

				// Fail if the simulator does not exist
				_, err := cluster.getSimulator(name)
				if err != nil {
					exitError(err)
				}

				// Remove the simulator from the cluster configuration
				delete(cluster.Simulators, name)

				// Write the configuration and exit
				if err = config.Write(); err != nil {
					exitError(err)
				}
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
	cmd.AddCommand(getGetSimulatorsCommand())
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
			// Load the onit configuration from disk
			config, err := LoadConfig()
			if err != nil {
				exitError(err)
			}
			fmt.Println(config.DefaultCluster)
		},
	}
}

// getGetSimulatorsCommand returns a cobra command to get the list of simulators deployed in the current cluster context
func getGetSimulatorsCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "simulators",
		Short: "Get the currently configured cluster's simulators",
		Run: func(cmd *cobra.Command, args []string) {
			// Load the onit configuration from disk
			config, err := LoadConfig()
			if err != nil {
				exitError(err)
			}

			// Get the default cluster ID
			clusterId, err := config.getDefaultCluster()
			if err != nil {
				exitError(err)
			}

			// Get the default cluster configuration
			cluster, err := config.getClusterConfig(clusterId)
			if err != nil {
				exitError(err)
			}

			// Iterate and print simulator names
			for name, _ := range cluster.Simulators {
				fmt.Println(name)
			}
		},
	}
}

// getGetClustersCommand returns a cobra command to get a list of available test clusters
func getGetClustersCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "clusters",
		Short: "Get a list of all deployed clusters",
		Run: func(cmd *cobra.Command, args []string) {
			// Load the onit configuration from disk
			config, err := LoadConfig()
			if err != nil {
				exitError(err)
			}

			// Iterate and print cluster names
			for name, _ := range config.Clusters {
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
			for _, name := range getSimulatorPresets() {
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
			for _, name := range getStorePresets() {
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

			// Load the onit configuration from disk
			config, err := LoadConfig()
			if err != nil {
				exitError(err)
			}

			// Acquire a lock on the configuration
			if err = config.Lock(); err != nil {
				exitError(err)
			}

			// Unlock the configuration lock once complete
			defer config.Unlock()

			// Verify that the cluster exists
			_, err = config.getClusterConfig(clusterId)
			if err != nil {
				exitError(err)
			}

			// Update and write the configuration
			config.DefaultCluster = clusterId
			if err = config.Write(); err != nil {
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
			// Load the onit configuration from disk
			config, err := LoadConfig()
			if err != nil {
				exitError(err)
			}

			// Get the default cluster ID
			clusterId, err := config.getDefaultCluster()
			if err != nil {
				exitError(err)
			}

			// Fail if the default cluster configuration cannot be found
			cluster, err := config.getClusterConfig(clusterId)
			if err != nil {
				exitError(err)
			}

			// Get the test cluster controller from the cluster configuration
			controller, err := GetTestController(clusterId, cluster)
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

// exitError prints the given err to stdout and exits with exit code 1
func exitError(err error) {
	fmt.Println(err)
	os.Exit(1)
}
