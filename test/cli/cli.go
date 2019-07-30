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
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"text/tabwriter"
	"time"

	"github.com/google/uuid"
	"github.com/onosproject/onos-config/test/console"
	"github.com/onosproject/onos-config/test/runner"
	"github.com/spf13/cobra"
	corev1 "k8s.io/api/core/v1"
)

// Contains tells whether array contains x.
func Contains(array []string, elem string) bool {
	for _, n := range array {
		if elem == n {
			return true
		}
	}
	return false
}

// GetOnitCommand returns a Cobra command for tests in the given test registry
func GetOnitCommand(registry *runner.TestRegistry) *cobra.Command {
	cmd := &cobra.Command{
		Use:                    "onit",
		Short:                  "Run onos-config integration tests on Kubernetes",
		BashCompletionFunction: bashCompletion,
	}
	cmd.AddCommand(getCreateCommand())
	cmd.AddCommand(getAddCommand())
	cmd.AddCommand(getRemoveCommand())
	cmd.AddCommand(getDeleteCommand())
	cmd.AddCommand(getRunCommand(registry))
	cmd.AddCommand(getGetCommand(registry))
	cmd.AddCommand(getSetCommand())
	cmd.AddCommand(getDebugCommand())
	cmd.AddCommand(getFetchCommand())
	cmd.AddCommand(getCompletionCommand())

	return cmd
}

// GetOnitK8sCommand returns a Cobra command for running tests on k8s
func GetOnitK8sCommand(registry *runner.TestRegistry) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "onit-k8s",
		Short: "This command is only intended to be used in a k8s instance for running integration tests.",
	}
	cmd.AddCommand(getTestTestLocalCommand(registry))
	cmd.AddCommand(getTestSuiteLocalCommand(registry))
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
			configNodes, _ := cmd.Flags().GetInt("config-nodes")
			topoNodes, _ := cmd.Flags().GetInt("topo-nodes")
			partitions, _ := cmd.Flags().GetInt("partitions")
			partitionSize, _ := cmd.Flags().GetInt("partition-size")
			configName, _ := cmd.Flags().GetString("config")

			// Get the onit controller
			controller, err := runner.NewController()
			if err != nil {
				exitError(err)
			}

			// Get or create a cluster ID
			var clusterID string
			if len(args) > 0 {
				clusterID = args[0]
			} else {
				clusterID = fmt.Sprintf("cluster-%s", newUUIDString())
			}

			// Create the cluster configuration
			config := &runner.ClusterConfig{
				Preset:        configName,
				ConfigNodes:   configNodes,
				TopoNodes:     topoNodes,
				Partitions:    partitions,
				PartitionSize: partitionSize,
			}

			// Create the cluster controller
			cluster, status := controller.NewCluster(clusterID, config)
			if status.Failed() {
				exitStatus(status)
			}

			// Store the cluster before setting it up to ensure other shell sessions can debug setup
			_ = setDefaultCluster(clusterID)

			// Setup the cluster
			if status := cluster.Setup(); status.Failed() {
				exitStatus(status)
			} else {
				fmt.Println(clusterID)
			}
		},
	}
	cmd.Flags().StringP("config", "c", "default", "test cluster configuration")
	cmd.Flags().IntP("config-nodes", "", 1, "the number of onos-config nodes to deploy")
	cmd.Flags().IntP("topo-nodes", "", 0, "the number of onos-topo nodes to deploy")
	cmd.Flags().IntP("partitions", "p", 1, "the number of Raft partitions to deploy")
	cmd.Flags().IntP("partition-size", "s", 1, "the size of each Raft partition")
	return cmd
}

// getAddCommand returns a cobra "add" command for adding resources to the cluster
func getAddCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "add {simulator,network} [args]",
		Short: "Add resources to the cluster",
	}
	cmd.AddCommand(getAddSimulatorCommand())
	cmd.AddCommand(getAddNetworkCommand())
	return cmd
}

// getAddNetworkCommand returns a cobra command for deploying a stratum network
func getAddNetworkCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "network [name]",
		Short: "Add a stratum network to the test cluster",
		Args:  cobra.MaximumNArgs(10),
		Run: func(cmd *cobra.Command, args []string) {
			// If a network name was not provided, generate one from a UUID.
			var name string
			if len(args) > 0 {
				name = args[0]
			} else {
				name = fmt.Sprintf("network-%d", newUUIDInt())
			}

			// Create the simulator configuration from the configured preset
			configName, _ := cmd.Flags().GetString("preset")

			// Get the onit controller
			controller, err := runner.NewController()
			if err != nil {
				exitError(err)
			}

			// Get the cluster ID
			clusterID, err := cmd.Flags().GetString("cluster")
			if err != nil {
				exitError(err)
			}

			// Get the cluster controller
			cluster, err := controller.GetCluster(clusterID)
			if err != nil {
				exitError(err)
			}

			// Create the network configuration

			config := &runner.NetworkConfig{
				Config: configName,
			}
			if len(args) > 1 {
				config.MininetOptions = args[1:]
			}

			// Update number of devices in the network configuration
			runner.ParseMininetOptions(config)

			if err != nil {
				exitError(err)
			}

			// Add the network to the cluster
			if status := cluster.AddNetwork(name, config); status.Failed() {
				exitStatus(status)
			} else {
				fmt.Println(name)
			}
		},
	}

	cmd.Flags().StringP("cluster", "c", getDefaultCluster(), "the cluster to which to add the simulator")
	cmd.Flags().Lookup("cluster").Annotations = map[string][]string{
		cobra.BashCompCustom: {"__onit_get_clusters"},
	}
	cmd.Flags().StringP("preset", "p", "default", "simulator preset to apply")
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
				name = fmt.Sprintf("device-%d", newUUIDInt())
			}

			// Create the simulator configuration from the configured preset
			configName, _ := cmd.Flags().GetString("preset")

			// Get the onit controller
			controller, err := runner.NewController()
			if err != nil {
				exitError(err)
			}

			// Get the cluster ID
			clusterID, err := cmd.Flags().GetString("cluster")
			if err != nil {
				exitError(err)
			}

			// Get the cluster controller
			cluster, err := controller.GetCluster(clusterID)
			if err != nil {
				exitError(err)
			}

			// Create the simulator configuration
			config := &runner.SimulatorConfig{
				Config: configName,
			}

			// Add the simulator to the cluster
			if status := cluster.AddSimulator(name, config); status.Failed() {
				exitStatus(status)
			} else {
				fmt.Println(name)
			}
		},
	}

	cmd.Flags().StringP("cluster", "c", getDefaultCluster(), "the cluster to which to add the simulator")
	cmd.Flags().Lookup("cluster").Annotations = map[string][]string{
		cobra.BashCompCustom: {"__onit_get_clusters"},
	}
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
			// Create the onit controller
			controller, err := runner.NewController()
			if err != nil {
				exitError(err)
			}

			// Get the cluster ID
			var clusterID string
			if len(args) > 0 {
				clusterID = args[0]
			} else {
				clusterID = getDefaultCluster()
			}

			// Delete the cluster
			status := controller.DeleteCluster(clusterID)
			_ = setDefaultCluster("")
			if status.Failed() {
				exitStatus(status)
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
	cmd.AddCommand(getRemoveNetworkCommand())
	return cmd
}

// getRemoveNetworkCommand returns a cobra command for tearing down a stratum network
func getRemoveNetworkCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "network [name]",
		Short: "Remove a stratum network from the cluster",
		Args:  cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			name := args[0]

			// Get the onit controller
			controller, err := runner.NewController()
			if err != nil {
				exitError(err)
			}

			// Get the cluster ID
			clusterID, err := cmd.Flags().GetString("cluster")
			if err != nil {
				exitError(err)
			}

			// Get the cluster controller
			cluster, err := controller.GetCluster(clusterID)
			if err != nil {
				exitError(err)
			}

			networks, err := cluster.GetNetworks()
			if err != nil {
				exitError(err)
			}
			if !Contains(networks, name) {
				exitError(errors.New("The given network name does not exist"))
			}

			// Remove the network from the cluster
			if status := cluster.RemoveNetwork(name); status.Failed() {
				exitStatus(status)
			}
		},
	}

	cmd.Flags().StringP("cluster", "c", getDefaultCluster(), "the cluster to which to add the network")
	cmd.Flags().Lookup("cluster").Annotations = map[string][]string{
		cobra.BashCompCustom: {"__onit_get_clusters"},
	}
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

			// Get the onit controller
			controller, err := runner.NewController()
			if err != nil {
				exitError(err)
			}

			// Get the cluster ID
			clusterID, err := cmd.Flags().GetString("cluster")
			if err != nil {
				exitError(err)
			}

			// Get the cluster controller
			cluster, err := controller.GetCluster(clusterID)
			if err != nil {
				exitError(err)
			}

			simulators, err := cluster.GetSimulators()
			if err != nil {
				exitError(err)
			}

			if !Contains(simulators, name) {
				exitError(errors.New("The given simulator name does not exist"))
			}

			// Remove the simulator from the cluster
			if status := cluster.RemoveSimulator(name); status.Failed() {
				exitStatus(status)
			}
		},
	}

	cmd.Flags().StringP("cluster", "c", getDefaultCluster(), "the cluster to which to add the simulator")
	cmd.Flags().Lookup("cluster").Annotations = map[string][]string{
		cobra.BashCompCustom: {"__onit_get_clusters"},
	}
	return cmd
}

// getGetCommand returns a cobra "get" command to read test configurations
func getGetCommand(registry *runner.TestRegistry) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "get {cluster,clusters,networks,simulators,device-presets,store-presets,tests,suites}",
		Short: "Get test configurations",
	}
	cmd.AddCommand(getGetClusterCommand())
	cmd.AddCommand(getGetNodesCommand())
	cmd.AddCommand(getGetPartitionsCommand())
	cmd.AddCommand(getGetPartitionCommand())
	cmd.AddCommand(getGetSimulatorsCommand())
	cmd.AddCommand(getGetNetworksCommand())
	cmd.AddCommand(getGetClustersCommand())
	cmd.AddCommand(getGetDevicePresetsCommand())
	cmd.AddCommand(getGetStorePresetsCommand())
	cmd.AddCommand(getGetTestsCommand(registry))
	cmd.AddCommand(getGetTestSuitesCommand(registry))
	cmd.AddCommand(getGetHistoryCommand())
	cmd.AddCommand(getGetLogsCommand())
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

// getGetNetworksCommand returns a cobra command to get the list of networks deployed in the current cluster context
func getGetNetworksCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "networks",
		Short: "Get the currently configured cluster's networks",
		Run: func(cmd *cobra.Command, args []string) {
			// Get the onit controller
			controller, err := runner.NewController()
			if err != nil {
				exitError(err)
			}

			// Get the cluster ID
			clusterID, err := cmd.Flags().GetString("cluster")
			if err != nil {
				exitError(err)
			}

			// Get the cluster controller
			cluster, err := controller.GetCluster(clusterID)
			if err != nil {
				exitError(err)
			}

			// Get the list of networks and output
			networks, err := cluster.GetNetworks()
			if err != nil {
				exitError(err)
			} else {
				for _, name := range networks {
					fmt.Println(name)
				}
			}
		},
	}

	cmd.Flags().StringP("cluster", "c", getDefaultCluster(), "the cluster to query")
	cmd.Flags().Lookup("cluster").Annotations = map[string][]string{
		cobra.BashCompCustom: {"__onit_get_clusters"},
	}
	return cmd
}

// getGetSimulatorsCommand returns a cobra command to get the list of simulators deployed in the current cluster context
func getGetSimulatorsCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "simulators",
		Short: "Get the currently configured cluster's simulators",
		Run: func(cmd *cobra.Command, args []string) {
			// Get the onit controller
			controller, err := runner.NewController()
			if err != nil {
				exitError(err)
			}

			// Get the cluster ID
			clusterID, err := cmd.Flags().GetString("cluster")
			if err != nil {
				exitError(err)
			}

			// Get the cluster controller
			cluster, err := controller.GetCluster(clusterID)
			if err != nil {
				exitError(err)
			}

			// Get the list of simulators and output
			simulators, err := cluster.GetSimulators()
			if err != nil {
				exitError(err)
			} else {
				for _, name := range simulators {
					fmt.Println(name)
				}
			}
		},
	}

	cmd.Flags().StringP("cluster", "c", getDefaultCluster(), "the cluster to query")
	cmd.Flags().Lookup("cluster").Annotations = map[string][]string{
		cobra.BashCompCustom: {"__onit_get_clusters"},
	}
	return cmd
}

// getGetClustersCommand returns a cobra command to get a list of available test clusters
func getGetClustersCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "clusters",
		Short: "Get a list of all deployed clusters",
		Run: func(cmd *cobra.Command, args []string) {
			// Get the onit controller
			controller, err := runner.NewController()
			if err != nil {
				exitError(err)
			}

			// Get the list of clusters and output
			clusters, err := controller.GetClusters()
			if err != nil {
				exitError(err)
			} else {
				noHeaders, _ := cmd.Flags().GetBool("no-headers")
				printClusters(clusters, !noHeaders)
			}
		},
	}
	cmd.Flags().Bool("no-headers", false, "whether to print column headers")
	return cmd
}

func printClusters(clusters map[string]*runner.ClusterConfig, includeHeaders bool) {
	writer := new(tabwriter.Writer)
	writer.Init(os.Stdout, 0, 0, 3, ' ', tabwriter.FilterHTML)
	if includeHeaders {
		_, _ = fmt.Fprintln(writer, "ID\tSIZE\tPARTITIONS")
	}
	for id, config := range clusters {
		fmt.Fprintln(writer, fmt.Sprintf("%s\t%d\t%d\t%d", id, config.ConfigNodes, config.TopoNodes, config.Partitions))
	}
	_ = writer.Flush()
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

// getGetPartitionsCommand returns a cobra command to get a list of Raft partitions in the cluster
func getGetPartitionsCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "partitions",
		Short: "Get a list of partitions in the cluster",
		Run: func(cmd *cobra.Command, args []string) {
			// Get the onit controller
			controller, err := runner.NewController()
			if err != nil {
				exitError(err)
			}

			// Get the cluster ID
			clusterID, err := cmd.Flags().GetString("cluster")
			if err != nil {
				exitError(err)
			}

			// Get the cluster controller
			cluster, err := controller.GetCluster(clusterID)
			if err != nil {
				exitError(err)
			}

			// Get the list of partitions and output
			partitions, err := cluster.GetPartitions()
			if err != nil {
				exitError(err)
			} else {
				printPartitions(partitions)
			}
		},
	}
	cmd.Flags().StringP("cluster", "c", getDefaultCluster(), "the cluster to query")
	cmd.Flags().Lookup("cluster").Annotations = map[string][]string{
		cobra.BashCompCustom: {"__onit_get_clusters"},
	}
	return cmd
}

func printPartitions(partitions []runner.PartitionInfo) {
	writer := new(tabwriter.Writer)
	writer.Init(os.Stdout, 0, 0, 3, ' ', tabwriter.FilterHTML)
	_, _ = fmt.Fprintln(writer, "ID\tGROUP\tNODES")
	for _, partition := range partitions {
		_, _ = fmt.Fprintln(writer, fmt.Sprintf("%d\t%s\t%s", partition.Partition, partition.Group, strings.Join(partition.Nodes, ",")))
	}
	_ = writer.Flush()
}

// getGetPartitionCommand returns a cobra command to get the nodes in a partition
func getGetPartitionCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "partition <partition>",
		Short: "Get a list of nodes in a partition",
		Args:  cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			// Get the onit controller
			controller, err := runner.NewController()
			if err != nil {
				exitError(err)
			}

			// Get the cluster ID
			clusterID, err := cmd.Flags().GetString("cluster")
			if err != nil {
				exitError(err)
			}

			// Get the cluster controller
			cluster, err := controller.GetCluster(clusterID)
			if err != nil {
				exitError(err)
			}

			// Parse the partition argument
			partition, err := strconv.ParseInt(args[0], 0, 32)
			if err != nil {
				exitError(err)
			}

			// Get the list of nodes and output
			nodes, err := cluster.GetPartitionNodes(int(partition))
			if err != nil {
				exitError(err)
			} else {
				printNodes(nodes, true)
			}
		},
	}
	cmd.Flags().StringP("cluster", "c", getDefaultCluster(), "the cluster to query")
	cmd.Flags().Lookup("cluster").Annotations = map[string][]string{
		cobra.BashCompCustom: {"__onit_get_clusters"},
	}
	return cmd
}

// getGetNodesCommand returns a cobra command to get a list of onos nodes in the cluster
func getGetNodesCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "nodes",
		Short: "Get a list of nodes in the cluster",
		Run: func(cmd *cobra.Command, args []string) {
			// Get the onit controller
			controller, err := runner.NewController()
			if err != nil {
				exitError(err)
			}

			// Get the cluster ID
			clusterID, err := cmd.Flags().GetString("cluster")
			if err != nil {
				exitError(err)
			}

			nodeType, err := cmd.Flags().GetString("type")
			if err != nil {
				exitError(err)
			}

			// Get the cluster controller
			cluster, err := controller.GetCluster(clusterID)
			if err != nil {
				exitError(err)
			}

			// Get the list of nodes and output
			if strings.Compare(nodeType, string(runner.OnosAll)) == 0 {
				nodes, err := cluster.GetNodes()
				if err != nil {
					exitError(err)
				} else {
					noHeaders, _ := cmd.Flags().GetBool("no-headers")
					printNodes(nodes, !noHeaders)
				}
			} else if strings.Compare(nodeType, string(runner.OnosConfig)) == 0 {
				nodes, err := cluster.GetOnosConfigNodes()
				if err != nil {
					exitError(err)
				} else {
					noHeaders, _ := cmd.Flags().GetBool("no-headers")
					printNodes(nodes, !noHeaders)
				}

			} else if strings.Compare(nodeType, string(runner.OnosTopo)) == 0 {
				nodes, err := cluster.GetOnosTopoNodes()
				if err != nil {
					exitError(err)
				} else {
					noHeaders, _ := cmd.Flags().GetBool("no-headers")
					printNodes(nodes, !noHeaders)
				}

			}
		},
	}
	cmd.Flags().StringP("cluster", "c", getDefaultCluster(), "the cluster to query")
	cmd.Flags().StringP("type", "t", "all", "To get list of nodes based on their types")
	cmd.Flags().Lookup("cluster").Annotations = map[string][]string{
		cobra.BashCompCustom: {"__onit_get_clusters"},
	}
	cmd.Flags().Bool("no-headers", false, "whether to print column headers")
	return cmd
}

func printNodes(nodes []runner.NodeInfo, includeHeaders bool) {
	writer := new(tabwriter.Writer)
	writer.Init(os.Stdout, 0, 0, 3, ' ', tabwriter.FilterHTML)
	if includeHeaders {
		fmt.Fprintln(writer, "ID\tTYPE\tSTATUS")
	}
	for _, node := range nodes {
		fmt.Fprintln(writer, fmt.Sprintf("%s\t%s\t%s", node.ID, node.Type, node.Status))
	}
	_ = writer.Flush()
}

// getGetTestsCommand returns a cobra command to get a list of available tests
func getGetTestsCommand(registry *runner.TestRegistry) *cobra.Command {
	return &cobra.Command{
		Use:   "tests",
		Short: "Get a list of integration tests",
		Run: func(cmd *cobra.Command, args []string) {
			for _, name := range registry.GetTestNames() {
				fmt.Println(name)
			}
		},
	}
}

// getGetTestsCommand returns a cobra command to get a list of available tests
func getGetTestSuitesCommand(registry *runner.TestRegistry) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "suites",
		Short: "Get a list of integration testing suites",
		Run: func(cmd *cobra.Command, args []string) {
			noHeaders, _ := cmd.Flags().GetBool("no-headers")
			printTestSuites(registry, !noHeaders)
		},
	}

	cmd.Flags().Bool("no-headers", false, "whether to print column headers")
	return cmd
}

//PrintTestSuites prints test suites in a table
func printTestSuites(registry *runner.TestRegistry, includeHeaders bool) {
	writer := new(tabwriter.Writer)
	writer.Init(os.Stdout, 0, 0, 3, ' ', tabwriter.FilterHTML)
	if includeHeaders {
		_, _ = fmt.Fprintln(writer, "SUITE\tTESTS")
	}
	for name, suite := range registry.TestSuites {
		_, _ = fmt.Fprintln(writer, fmt.Sprintf("%s\t%s", name, strings.Join(suite.GetTestNames(), ", ")))
	}
	_ = writer.Flush()
}

// getGetHistoryCommand returns a cobra command to get the history of tests
func getGetHistoryCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "history",
		Short: "Get the history of test runs",
		Run: func(cmd *cobra.Command, args []string) {
			// Get the onit controller
			controller, err := runner.NewController()
			if err != nil {
				exitError(err)
			}

			// Get the cluster ID
			clusterID, err := cmd.Flags().GetString("cluster")
			if err != nil {
				exitError(err)
			}

			// Get the cluster controller
			cluster, err := controller.GetCluster(clusterID)
			if err != nil {
				exitError(err)
			}

			// Get the history of test runs for the cluster
			records, err := cluster.GetHistory()
			if err != nil {
				exitError(err)
			}

			printHistory(records)
		},
	}

	cmd.Flags().StringP("cluster", "c", getDefaultCluster(), "the cluster for which to load the history")
	cmd.Flags().Lookup("cluster").Annotations = map[string][]string{
		cobra.BashCompCustom: {"__onit_get_clusters"},
	}
	return cmd
}

// printHistory prints a test history in table format
func printHistory(records []runner.TestRecord) {
	writer := new(tabwriter.Writer)
	writer.Init(os.Stdout, 0, 0, 3, ' ', tabwriter.FilterHTML)
	_, _ = fmt.Fprintln(writer, "ID\tTESTS\tSTATUS\tEXIT CODE\tMESSAGE")
	for _, record := range records {
		var args string
		if len(record.Args) > 0 {
			args = strings.Join(record.Args, ",")
		} else {
			args = "*"
		}
		_, _ = fmt.Fprintln(writer, fmt.Sprintf("%s\t%s\t%s\t%d\t%s", record.TestID, args, record.Status, record.ExitCode, record.Message))
	}
	_ = writer.Flush()
}

// getGetLogsCommand returns a cobra command to output the logs for a specific resource
func getGetLogsCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "logs <id> [options]",
		Short: "Get the logs for a test resource",
		Long: `Outputs the complete logs for any test resource.
To output the logs from an onos-config node, get the node ID via 'onit get nodes'
To output the logs from a test, get the test ID from the test run or from 'onit get history'`,
		Args: cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			stream, _ := cmd.Flags().GetBool("stream")

			// Get the onit controller
			controller, err := runner.NewController()
			if err != nil {
				exitError(err)
			}

			// Get the cluster ID
			clusterID, err := cmd.Flags().GetString("cluster")
			if err != nil {
				exitError(err)
			}

			// Get the cluster controller
			cluster, err := controller.GetCluster(clusterID)
			if err != nil {
				exitError(err)
			}

			// If streaming is enabled, stream the logs to stdout. Otherwise, get the logs for all resources and print them.
			if stream {
				reader, err := cluster.StreamLogs(args[0])
				if err != nil {
					exitError(err)
				}
				defer reader.Close()
				if err = printStream(reader); err != nil {
					exitError(err)
				}
			} else {
				resources, err := cluster.GetResources(args[0])
				if err != nil {
					exitError(err)
				}

				// Iterate through resources and get/print logs
				numResources := len(resources)

				if err != nil {
					exitError(err)
				}

				options := parseLogOptions(cmd)

				for i, resource := range resources {
					logs, err := cluster.GetLogs(resource, options)
					if err != nil {
						exitError(err)
					}
					_, _ = os.Stdout.Write(logs)
					if i+1 < numResources {
						fmt.Println("----")
					}
				}
			}
		},
	}

	cmd.Flags().DurationP("since", "", -1, "Only return logs newer than a relative "+
		"duration like 5s, 2m, or 3h. Defaults to all logs. Only one of since-time / since may be used")
	cmd.Flags().Int64P("tail", "t", -1, "If set, the number of bytes to read from the "+
		"server before terminating the log output. This may not display a complete final line of logging, and may return "+
		"slightly more or slightly less than the specified limit.")

	cmd.Flags().StringP("cluster", "c", getDefaultCluster(), "the cluster for which to load the history")
	cmd.Flags().Lookup("cluster").Annotations = map[string][]string{
		cobra.BashCompCustom: {"__onit_get_clusters"},
	}
	cmd.Flags().BoolP("stream", "s", false, "stream logs to stdout")
	return cmd
}

func parseLogOptions(cmd *cobra.Command) corev1.PodLogOptions {
	// Parse log options from CLI
	options := corev1.PodLogOptions{}
	since, err := cmd.Flags().GetDuration("since")
	sinceSeconds := int64(since / time.Second)
	if sinceSeconds > 0 {
		options.SinceSeconds = &sinceSeconds
	}
	if err != nil {
		exitError(err)
	}
	tail, err := cmd.Flags().GetInt64("tail")
	if tail > 0 && err != nil {
		options.TailLines = &tail
	}
	return options
}

// printStream prints a stream to stdout from the given reader
func printStream(reader io.Reader) error {
	buf := make([]byte, 1024)
	for {
		n, err := reader.Read(buf)
		if err != nil {
			if err == io.EOF {
				break
			}
			return err
		}
		fmt.Print(string(buf[:n]))
	}
	return nil
}

// getFetchCommand returns a cobra "download" command for downloading resources from a test cluster
func getFetchCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "fetch",
		Short: "Fetch resources from the cluster",
	}
	cmd.AddCommand(getFetchLogsCommand())
	return cmd
}

// getFetchLogsCommand returns a cobra command for downloading the logs from a node
func getFetchLogsCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "logs [node]",
		Short: "Download logs from a node",
		Args:  cobra.MaximumNArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			// Get the onit controller
			controller, err := runner.NewController()
			if err != nil {
				exitError(err)
			}

			// Get the cluster ID
			clusterID, err := cmd.Flags().GetString("cluster")
			if err != nil {
				exitError(err)
			}

			// Get the cluster controller
			cluster, err := controller.GetCluster(clusterID)
			if err != nil {
				exitError(err)
			}

			options := parseLogOptions(cmd)

			destination, _ := cmd.Flags().GetString("destination")
			if len(args) > 0 {
				resourceID := args[0]
				resources, err := cluster.GetResources(resourceID)
				if err != nil {
					exitError(err)
				}

				var status console.ErrorStatus
				for _, resource := range resources {
					path := filepath.Join(destination, fmt.Sprintf("%s.log", resource))
					status = cluster.DownloadLogs(resource, path, options)
				}

				if status.Failed() {
					exitStatus(status)
				}
			} else {
				nodes, err := cluster.GetNodes()
				if err != nil {
					exitError(err)
				}

				var status console.ErrorStatus
				for _, node := range nodes {
					path := filepath.Join(destination, fmt.Sprintf("%s.log", node.ID))
					status = cluster.DownloadLogs(node.ID, path, options)
				}

				if status.Failed() {
					exitStatus(status)
				}
			}
		},
	}

	cmd.Flags().DurationP("since", "", -1, "Only return logs newer than a relative "+
		"duration like 5s, 2m, or 3h. Defaults to all logs. Only one of since-time / since may be used")
	cmd.Flags().Int64P("tail", "t", -1, "If set, the number of bytes to read from the "+
		"server before terminating the log output. This may not display a complete final line of logging, and may return "+
		"slightly more or slightly less than the specified limit.")

	cmd.Flags().StringP("cluster", "c", getDefaultCluster(), "the cluster for which to load the history")
	cmd.Flags().Lookup("cluster").Annotations = map[string][]string{
		cobra.BashCompCustom: {"__onit_get_clusters"},
	}
	cmd.Flags().StringP("destination", "d", ".", "the destination to which to write the logs")
	return cmd
}

// getDebugCommand returns a cobra "debug" command to open a debugger port to the given resource
func getDebugCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "debug <resource>",
		Short: "Open a debugger port to the given resource",
		Args:  cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			// Get the onit controller
			controller, err := runner.NewController()
			if err != nil {
				exitError(err)
			}

			// Get the cluster ID
			clusterID, err := cmd.Flags().GetString("cluster")
			if err != nil {
				exitError(err)
			}

			// Get the cluster controller
			cluster, err := controller.GetCluster(clusterID)
			if err != nil {
				exitError(err)
			}

			port, _ := cmd.Flags().GetInt("port")
			if err := cluster.PortForward(args[0], port, 40000); err != nil {
				exitError(err)
			} else {
				fmt.Println(port)
			}
		},
	}
	cmd.Flags().StringP("cluster", "c", getDefaultCluster(), "the cluster for which to load the history")
	cmd.Flags().Lookup("cluster").Annotations = map[string][]string{
		cobra.BashCompCustom: {"__onit_get_clusters"},
	}
	cmd.Flags().IntP("port", "p", 40000, "the local port to forward to the given resource")
	return cmd
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
			clusterID := args[0]

			// Get the onit controller
			controller, err := runner.NewController()
			if err != nil {
				exitError(err)
			}

			// Get the cluster controller
			_, err = controller.GetCluster(clusterID)
			if err != nil {
				exitError(err)
			}

			// If we've made it this far, update the default cluster
			if err := setDefaultCluster(clusterID); err != nil {
				exitError(err)
			} else {
				fmt.Println(clusterID)
			}
		},
	}
}

// getRunCommand returns a cobra run command to run integration tests
func getRunCommand(registry *runner.TestRegistry) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "run {test,suite}",
		Short: "Run integration tests",
	}
	cmd.AddCommand(getRunSuiteRemoteCommand(registry))
	cmd.AddCommand(getRunTestRemoteCommand())
	return cmd
}

// getRunCommand returns a cobra "run" command
func getRunTestRemoteCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "test [tests]",
		Short: "Run integration tests on Kubernetes",
		Run: func(cmd *cobra.Command, args []string) {
			runTestsRemote(cmd, "test", args)
		},
	}
	cmd.Flags().StringP("cluster", "c", getDefaultCluster(), "the cluster on which to run the test")
	cmd.Flags().Lookup("cluster").Annotations = map[string][]string{
		cobra.BashCompCustom: {"__onit_get_clusters"},
	}
	cmd.Flags().IntP("timeout", "t", 60*10, "test timeout in seconds")
	return cmd
}

func getRunSuiteRemoteCommand(registry *runner.TestRegistry) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "suite [suite]",
		Short: "Run integration tests",
		Run: func(cmd *cobra.Command, args []string) {
			runTestsRemote(cmd, "suite", args)
		},
	}
	cmd.Flags().StringP("cluster", "c", getDefaultCluster(), "the cluster on which to run the test")
	cmd.Flags().Lookup("cluster").Annotations = map[string][]string{
		cobra.BashCompCustom: {"__onit_get_clusters"},
	}
	cmd.Flags().IntP("timeout", "t", 60*10, "test timeout in seconds")

	return cmd
}

func runTestsRemote(cmd *cobra.Command, commandType string, tests []string) {
	testID := fmt.Sprintf("test-%d", newUUIDInt())

	// Get the onit controller
	controller, err := runner.NewController()
	if err != nil {
		exitError(err)
	}

	// Get the cluster ID
	clusterID, err := cmd.Flags().GetString("cluster")
	if err != nil {
		exitError(err)
	}

	// Get the cluster controller
	cluster, err := controller.GetCluster(clusterID)
	if err != nil {
		exitError(err)
	}

	timeout, _ := cmd.Flags().GetInt("timeout")
	message, code, status := cluster.RunTests(testID, append([]string{commandType}, tests...), time.Duration(timeout)*time.Second)
	if status.Failed() {
		exitStatus(status)
	} else {
		fmt.Println(message)
		os.Exit(code)
	}
}

// getTestTestLocalCommand returns a cobra "test" command for tests in the given registry
func getTestTestLocalCommand(registry *runner.TestRegistry) *cobra.Command {
	return &cobra.Command{
		Use:   "test [tests]",
		Short: "Run integration tests",
		Run: func(cmd *cobra.Command, args []string) {
			testRunner := &runner.TestRunner{
				Registry: registry,
			}
			err := testRunner.RunTests(args)
			if err != nil {
				exitError(err)
			} else {
				os.Exit(0)
			}
		},
	}
}

// getTestSuiteLocalCommand returns a cobra "test" command for tests in the given registry
func getTestSuiteLocalCommand(registry *runner.TestRegistry) *cobra.Command {
	return &cobra.Command{
		Use:   "suite [suite]",
		Short: "Run integration test suites on Kubernetes",
		Run: func(cmd *cobra.Command, args []string) {
			testRunner := &runner.TestRunner{
				Registry: registry,
			}
			err := testRunner.RunTestSuites(args)
			if err != nil {
				exitError(err)
			} else {
				os.Exit(0)
			}
		},
	}
}

// newUUIDString returns a new string UUID
func newUUIDString() string {
	id, err := uuid.NewUUID()
	if err != nil {
		exitError(err)
	}
	return id.String()
}

// newUuidInt returns a numeric UUID
func newUUIDInt() uint32 {
	id, err := uuid.NewUUID()
	if err != nil {
		exitError(err)
	}
	return id.ID()
}

// exitStatus prints the errors from the given status and exits
func exitStatus(status console.ErrorStatus) {
	for _, err := range status.Errors() {
		fmt.Println(err)
	}
	os.Exit(1)
}

// exitError prints the given errors to stdout and exits with exit code 1
func exitError(err error) {
	fmt.Println(err)
	os.Exit(1)
}
