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
	"github.com/google/uuid"
	"github.com/spf13/cobra"
	log "k8s.io/klog"
	"os"
	"text/tabwriter"
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
	cmd.AddCommand(getCreateCommand())
	cmd.AddCommand(getAddCommand())
	cmd.AddCommand(getRemoveCommand())
	cmd.AddCommand(getDeleteCommand())
	cmd.AddCommand(getRunCommand())
	cmd.AddCommand(getGetCommand(registry))
	cmd.AddCommand(getSetCommand())
	cmd.AddCommand(getTestCommand(registry))
	return cmd
}

// getCreateCommand returns a cobra "setup" command for setting up resources
func getCreateCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "create {cluster} [args]",
		Short: "Create a test resource on Kubernetes",
	}
	cmd.AddCommand(getCreateClusterCommand())
	return cmd
}

// getCreateClusterCommand returns a cobra command for deploying a test cluster
func getCreateClusterCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "cluster [id]",
		Short: "Setup a test cluster on Kubernetes",
		Args:  cobra.MaximumNArgs(1),
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
				Simulators:    make(map[string]*SimulatorConfig),
			}

			// Get or create a cluster ID
			var clusterId string
			if len(args) > 0 {
				clusterId = args[0]
			} else {
				clusterId = newUuid()
			}

			// Acquire a lock on the configuration to add the cluster
			if err := config.Lock(); err != nil {
				exitError(err)
			}

			// If the cluster is already present in the configuration, fail
			if _, ok := config.Clusters[clusterId]; ok {
				exitError(errors.New("cluster already exists"))
			}

			// Add the cluster to the configuration and update the default cluster
			config.Clusters[clusterId] = cluster
			config.DefaultCluster = clusterId

			// Write the configuration and exit
			err = config.Write()
			config.Unlock()
			if err != nil {
				exitError(err)
			}

			// Create a new controller for the test cluster
			controller, err := GetClusterController(clusterId, cluster)
			if err != nil {
				exitError(err)
			}

			// Setup the cluster and update the onit configuration
			if err = controller.SetupCluster(); err != nil {
				exitError(err)
			}
		},
	}
	cmd.Flags().StringP("config", "c", "default", "test cluster configuration")
	cmd.Flags().IntP("nodes", "n", 1, "the number of onos-config nodes to deploy")
	cmd.Flags().IntP("partitions", "p", 1, "the number of Raft partitions to deploy")
	cmd.Flags().IntP("partitionSize", "s", 1, "the size of each Raft partition")
	return cmd
}

// getAddCommand returns a cobra "add" command for adding resources to the cluster
func getAddCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "add {simulator} [args]",
		Short: "Add resources to the cluster",
	}
	cmd.AddCommand(getAddSimulatorCommand())
	return cmd
}

// getAddSimulatorCommand returns a cobra command for deploying a device simulator
func getAddSimulatorCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "simulator [name]",
		Short: "Add a device simulator to the test cluster",
		Args:  cobra.MaximumNArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			// If a simulator name was not provided, generate one from a UUID.
			var name string
			if len(args) > 0 {
				name = args[0]
			} else {
				name = fmt.Sprintf("device-%s", newUuid())
			}

			// Create the simulator configuration from the configured preset
			configName, _ := cmd.Flags().GetString("preset")
			simulatorConfig, err := getSimulatorPreset(configName)
			if err != nil {
				exitError(err)
			}
			simulator := &SimulatorConfig{
				Config: simulatorConfig,
			}

			// Load the onit configuration from disk
			config, err := LoadConfig()
			if err != nil {
				exitError(err)
			}

			// Get the cluster ID
			clusterId, err := cmd.Flags().GetString("cluster")
			if err != nil {
				exitError(err)
			}

			// Acquire a lock on the configuration
			if err := config.Lock(); err != nil {
				exitError(err)
			}

			// Fail if the default cluster configuration cannot be found
			cluster, err := config.getClusterConfig(clusterId)
			if err != nil {
				config.Unlock()
				exitError(err)
			}

			// Add the simulator to the configuration
			cluster.Simulators[name] = simulator

			// Write the configuration before setting up the simulator
			err = config.Write()
			config.Unlock()
			if err != nil {
				exitError(err)
			}

			// Get the test cluster controller from the cluster configuration
			controller, err := GetClusterController(clusterId, cluster)
			if err != nil {
				exitError(err)
			}

			// Setup the simulator and output the simulator name once complete
			if err = controller.SetupSimulator(name, simulator); err != nil {
				exitError(err)
			} else {
				fmt.Println(name)
			}
		},
	}

	cmd.Flags().StringP("cluster", "c", getDefaultCluster(), "the cluster to which to add the simulator")
	cmd.Flags().StringP("preset", "p", "default", "simulator preset to apply")
	return cmd
}

// getDeleteCommand returns a cobra "teardown" command for tearing down Kubernetes test resources
func getDeleteCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "delete",
		Short: "Delete Kubernetes test resources",
	}
	cmd.AddCommand(getDeleteClusterCommand())
	return cmd
}

// getDeleteClusterCommand returns a cobra "teardown" command for tearing down a test cluster
func getDeleteClusterCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "cluster [id]",
		Short: "Delete a test cluster on Kubernetes",
		Args:  cobra.MaximumNArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			// Load the onit configuration from disk
			config, err := LoadConfig()
			if err != nil {
				exitError(err)
			}

			// Get the cluster ID
			var clusterId string
			if len(args) > 0 {
				clusterId = args[0]
			} else {
				clusterId = getDefaultCluster()
			}

			// Fail if the default cluster configuration cannot be found
			cluster, err := config.getClusterConfig(clusterId)
			if err != nil {
				exitError(err)
			}

			// Get the test cluster controller from the cluster configuration
			controller, err := GetClusterController(clusterId, cluster)
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

				// Fail if the cluster does not exist
				_, err := config.getClusterConfig(clusterId)
				if err != nil {
					config.Unlock()
					exitError(err)
				}

				// Remove the cluster from the clusters list
				delete(config.Clusters, clusterId)

				// If the cluster is the default cluster, unset the default
				if config.DefaultCluster == clusterId {
					config.DefaultCluster = ""
				}

				// Write the configuration and exit
				err = config.Write()
				config.Unlock()
				if err != nil {
					exitError(err)
				}
			}
		},
	}
	return cmd
}

// getRemoveCommand returns a cobra "remove" command for removing resources from the cluster
func getRemoveCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "remove {simulator} [args]",
		Short: "Remove resources from the cluster",
	}
	cmd.AddCommand(getRemoveSimulatorCommand())
	return cmd
}

// getRemoveSimulatorCommand returns a cobra command for tearing down a device simulator
func getRemoveSimulatorCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "simulator <name>",
		Short: "Remove a device simulator from the cluster",
		Args:  cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			name := args[0]

			// Load the onit configuration from disk
			config, err := LoadConfig()
			if err != nil {
				exitError(err)
			}

			// Get the cluster ID
			clusterId, err := cmd.Flags().GetString("cluster")
			if err != nil {
				exitError(err)
			}

			// Acquire a lock on the configuration
			if err := config.Lock(); err != nil {
				exitError(err)
			}

			// Fail if the default cluster configuration cannot be found
			cluster, err := config.getClusterConfig(clusterId)
			if err != nil {
				config.Unlock()
				exitError(err)
			}

			// Remove the simulator from the configuration
			delete(cluster.Simulators, name)

			// Write the configuration before tearing down the simulator
			err = config.Write()
			config.Unlock()
			if err != nil {
				exitError(err)
			}

			// Get the test cluster controller from the cluster configuration
			controller, err := GetClusterController(clusterId, cluster)
			if err != nil {
				exitError(err)
			}

			// Teardown the simulator and exit
			if err = controller.TeardownSimulator(name); err != nil {
				exitError(err)
			}
		},
	}

	cmd.Flags().StringP("cluster", "c", getDefaultCluster(), "the cluster to which to add the simulator")
	return cmd
}

// getGetCommand returns a cobra "get" command to read test configurations
func getGetCommand(registry *TestRegistry) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "get {cluster,clusters,simulators,device-presets,store-presets,tests}",
		Short: "Get test configurations",
	}
	cmd.AddCommand(getGetClusterCommand())
	cmd.AddCommand(getGetSimulatorsCommand())
	cmd.AddCommand(getGetClustersCommand())
	cmd.AddCommand(getGetDevicePresetsCommand())
	cmd.AddCommand(getGetStorePresetsCommand())
	cmd.AddCommand(getGetTestsCommand(registry))
	return cmd
}

// getGetClusterCommand returns a cobra command to get the current cluster context
func getGetClusterCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "cluster",
		Short: "Get the currently configured cluster",
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Println(getDefaultCluster())
		},
	}
}

// getGetSimulatorsCommand returns a cobra command to get the list of simulators deployed in the current cluster context
func getGetSimulatorsCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "simulators",
		Short: "Get the currently configured cluster's simulators",
		Run: func(cmd *cobra.Command, args []string) {
			// Load the onit configuration from disk
			config, err := LoadConfig()
			if err != nil {
				exitError(err)
			}

			// Get the cluster ID
			clusterId, err := cmd.Flags().GetString("cluster")
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

	cmd.Flags().StringP("cluster", "c", getDefaultCluster(), "the cluster to query")
	return cmd
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
			printClusters(config.Clusters)
		},
	}
}

func printClusters(clusters map[string]*ClusterConfig) {
	writer := new(tabwriter.Writer)
	writer.Init(os.Stdout, 0, 0, 3, ' ', tabwriter.FilterHTML)
	fmt.Fprintln(writer, "ID\tSIZE\tPARTITIONS")
	for id, config := range clusters {
		fmt.Fprintln(writer, fmt.Sprintf("%s\t%d\t%d", id, config.Nodes, config.Partitions))
	}
	writer.Flush()
}

// getGetDevicePresetsCommand returns a cobra command to get a list of available device simulator configurations
func getGetDevicePresetsCommand() *cobra.Command {
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

// getGetStorePresetsCommand returns a cobra command to get a list of available store configurations
func getGetStorePresetsCommand() *cobra.Command {
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

			// Verify that the cluster exists
			_, err = config.getClusterConfig(clusterId)
			if err != nil {
				config.Unlock()
				exitError(err)
			}

			// Update and write the configuration
			config.DefaultCluster = clusterId
			err = config.Write()
			config.Unlock()
			if err != nil {
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

			// Get the cluster ID
			clusterId, err := cmd.Flags().GetString("cluster")
			if err != nil {
				exitError(err)
			}

			// Fail if the default cluster configuration cannot be found
			cluster, err := config.getClusterConfig(clusterId)
			if err != nil {
				exitError(err)
			}

			// Get the test cluster controller from the cluster configuration
			controller, err := GetClusterController(clusterId, cluster)
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

	cmd.Flags().StringP("cluster", "c", getDefaultCluster(), "the cluster on which to run the test")
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

// getDefaultCluster returns the default cluster ID
func getDefaultCluster() string {
	// Load the onit configuration from disk
	config, err := LoadConfig()
	if err != nil {
		exitError(err)
	}
	return config.DefaultCluster
}

// newUuid returns a new UUID
func newUuid() string {
	id, err := uuid.NewUUID()
	if err != nil {
		exitError(err)
	}
	return id.String()
}

// exitError prints the given err to stdout and exits with exit code 1
func exitError(err error) {
	fmt.Println(err)
	os.Exit(1)
}
