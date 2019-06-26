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
	log "k8s.io/klog"
	"os"
	"strconv"
	"strings"
	"text/tabwriter"
	"time"
)

func init() {
	log.SetOutput(os.Stdout)
}

// GetCommand returns a Cobra command for tests in the given test registry
func GetCommand(registry *TestRegistry) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "onit {create,add,remove,delete,get,set,run} [args]",
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
			partitionSize, _ := cmd.Flags().GetInt("partition-size")
			configName, _ := cmd.Flags().GetString("config")

			// Get the onit controller
			controller, err := NewController()
			if err != nil {
				exitError(err)
			}

			// Get or create a cluster ID
			var clusterId string
			if len(args) > 0 {
				clusterId = args[0]
			} else {
				clusterId = fmt.Sprintf("cluster-%s", newUuidString())
			}

			// Create the cluster configuration
			config := &ClusterConfig{
				Preset:        configName,
				Nodes:         nodes,
				Partitions:    partitions,
				PartitionSize: partitionSize,
			}

			// Create the cluster controller
			cluster, err := controller.NewCluster(clusterId, config)
			if err != nil {
				exitError(err)
			}

			// Setup the cluster
			if err = cluster.Setup(); err != nil {
				exitError(err)
			} else {
				setDefaultCluster(clusterId)
				fmt.Println(clusterId)
			}
		},
	}
	cmd.Flags().StringP("config", "c", "default", "test cluster configuration")
	cmd.Flags().IntP("nodes", "n", 1, "the number of onos-config nodes to deploy")
	cmd.Flags().IntP("partitions", "p", 1, "the number of Raft partitions to deploy")
	cmd.Flags().IntP("partition-size", "s", 1, "the size of each Raft partition")
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
				name = fmt.Sprintf("device-%d", newUuidInt())
			}

			// Create the simulator configuration from the configured preset
			configName, _ := cmd.Flags().GetString("preset")

			// Get the onit controller
			controller, err := NewController()
			if err != nil {
				exitError(err)
			}

			// Get the cluster ID
			clusterId, err := cmd.Flags().GetString("cluster")
			if err != nil {
				exitError(err)
			}

			// Get the cluster controller
			cluster, err := controller.GetCluster(clusterId)
			if err != nil {
				exitError(err)
			}

			// Create the simulator configuration
			config := &SimulatorConfig{
				Config: configName,
			}

			// Add the simulator to the cluster
			if err = cluster.AddSimulator(name, config); err != nil {
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
			// Create the onit controller
			controller, err := NewController()
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

			// Delete the cluster
			if err = controller.DeleteCluster(clusterId); err != nil {
				exitError(err)
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

			// Get the onit controller
			controller, err := NewController()
			if err != nil {
				exitError(err)
			}

			// Get the cluster ID
			clusterId, err := cmd.Flags().GetString("cluster")
			if err != nil {
				exitError(err)
			}

			// Get the cluster controller
			cluster, err := controller.GetCluster(clusterId)
			if err != nil {
				exitError(err)
			}

			// Remove the simulator from the cluster
			if err = cluster.RemoveSimulator(name); err != nil {
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
	cmd.AddCommand(getGetNodesCommand())
	cmd.AddCommand(getGetPartitionsCommand())
	cmd.AddCommand(getGetPartitionCommand())
	cmd.AddCommand(getGetSimulatorsCommand())
	cmd.AddCommand(getGetClustersCommand())
	cmd.AddCommand(getGetDevicePresetsCommand())
	cmd.AddCommand(getGetStorePresetsCommand())
	cmd.AddCommand(getGetTestsCommand(registry))
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

// getGetSimulatorsCommand returns a cobra command to get the list of simulators deployed in the current cluster context
func getGetSimulatorsCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "simulators",
		Short: "Get the currently configured cluster's simulators",
		Run: func(cmd *cobra.Command, args []string) {
			// Get the onit controller
			controller, err := NewController()
			if err != nil {
				exitError(err)
			}

			// Get the cluster ID
			clusterId, err := cmd.Flags().GetString("cluster")
			if err != nil {
				exitError(err)
			}

			// Get the cluster controller
			cluster, err := controller.GetCluster(clusterId)
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
	return cmd
}

// getGetClustersCommand returns a cobra command to get a list of available test clusters
func getGetClustersCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "clusters",
		Short: "Get a list of all deployed clusters",
		Run: func(cmd *cobra.Command, args []string) {
			// Get the onit controller
			controller, err := NewController()
			if err != nil {
				exitError(err)
			}

			// Get the list of clusters and output
			clusters, err := controller.GetClusters()
			if err != nil {
				exitError(err)
			} else {
				printClusters(clusters)
			}
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

// getGetPartitionsCommand returns a cobra command to get a list of Raft partitions in the cluster
func getGetPartitionsCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "partitions",
		Short: "Get a list of partitions in the cluster",
		Run: func(cmd *cobra.Command, args []string) {
			// Get the onit controller
			controller, err := NewController()
			if err != nil {
				exitError(err)
			}

			// Get the cluster ID
			clusterId, err := cmd.Flags().GetString("cluster")
			if err != nil {
				exitError(err)
			}

			// Get the cluster controller
			cluster, err := controller.GetCluster(clusterId)
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
	return cmd
}

func printPartitions(partitions []PartitionInfo) {
	writer := new(tabwriter.Writer)
	writer.Init(os.Stdout, 0, 0, 3, ' ', tabwriter.FilterHTML)
	fmt.Fprintln(writer, "ID\tGROUP\tNODES")
	for _, partition := range partitions {
		fmt.Fprintln(writer, fmt.Sprintf("%d\t%s\t%s", partition.Partition, partition.Group, strings.Join(partition.Nodes, ",")))
	}
	writer.Flush()
}

// getGetPartitionCommand returns a cobra command to get the nodes in a partition
func getGetPartitionCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "partition <partition>",
		Short: "Get a list of nodes in a partition",
		Args:  cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			// Get the onit controller
			controller, err := NewController()
			if err != nil {
				exitError(err)
			}

			// Get the cluster ID
			clusterId, err := cmd.Flags().GetString("cluster")
			if err != nil {
				exitError(err)
			}

			// Get the cluster controller
			cluster, err := controller.GetCluster(clusterId)
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
				printNodes(nodes)
			}
		},
	}
	cmd.Flags().StringP("cluster", "c", getDefaultCluster(), "the cluster to query")
	return cmd
}

// getGetNodesCommand returns a cobra command to get a list of onos-config nodes in the cluster
func getGetNodesCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "nodes",
		Short: "Get a list of nodes in the cluster",
		Run: func(cmd *cobra.Command, args []string) {
			// Get the onit controller
			controller, err := NewController()
			if err != nil {
				exitError(err)
			}

			// Get the cluster ID
			clusterId, err := cmd.Flags().GetString("cluster")
			if err != nil {
				exitError(err)
			}

			// Get the cluster controller
			cluster, err := controller.GetCluster(clusterId)
			if err != nil {
				exitError(err)
			}

			// Get the list of nodes and output
			nodes, err := cluster.GetNodes()
			if err != nil {
				exitError(err)
			} else {
				printNodes(nodes)
			}
		},
	}
	cmd.Flags().StringP("cluster", "c", getDefaultCluster(), "the cluster to query")
	return cmd
}

func printNodes(nodes []NodeInfo) {
	writer := new(tabwriter.Writer)
	writer.Init(os.Stdout, 0, 0, 3, ' ', tabwriter.FilterHTML)
	fmt.Fprintln(writer, "ID\tSTATUS")
	for _, node := range nodes {
		fmt.Fprintln(writer, fmt.Sprintf("%s\t%s", node.Id, node.Status))
	}
	writer.Flush()
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

// getGetHistoryCommand returns a cobra command to get the history of tests
func getGetHistoryCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "history",
		Short: "Get the history of test runs",
		Run: func(cmd *cobra.Command, args []string) {
			// Get the onit controller
			controller, err := NewController()
			if err != nil {
				exitError(err)
			}

			// Get the cluster ID
			clusterId, err := cmd.Flags().GetString("cluster")
			if err != nil {
				exitError(err)
			}

			// Get the cluster controller
			cluster, err := controller.GetCluster(clusterId)
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
	return cmd
}

// printHistory prints a test history in table format
func printHistory(records []TestRecord) {
	writer := new(tabwriter.Writer)
	writer.Init(os.Stdout, 0, 0, 3, ' ', tabwriter.FilterHTML)
	fmt.Fprintln(writer, "ID\tTESTS\tSTATUS\tEXIT CODE\tMESSAGE")
	for _, record := range records {
		var args string
		if len(record.Args) > 0 {
			args = strings.Join(record.Args, ",")
		} else {
			args = "*"
		}
		fmt.Fprintln(writer, fmt.Sprintf("%s\t%s\t%s\t%d\t%s", record.TestId, args, record.Status, record.ExitCode, record.Message))
	}
	writer.Flush()
}

// getGetLogsCommand returns a cobra command to output the logs for a specific resource
func getGetLogsCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "logs <id>",
		Short: "Get the logs for a test resource",
		Long: `Outputs the complete logs for any test resource.
To output the logs from an onos-config node, get the node ID via 'onit get nodes'
To output the logs from a test, get the test ID from the test run or from 'onit get history'`,
		Args: cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			// Get the onit controller
			controller, err := NewController()
			if err != nil {
				exitError(err)
			}

			// Get the cluster ID
			clusterId, err := cmd.Flags().GetString("cluster")
			if err != nil {
				exitError(err)
			}

			// Get the cluster controller
			cluster, err := controller.GetCluster(clusterId)
			if err != nil {
				exitError(err)
			}

			// Get the logs for the resource and print to stdout
			logs, err := cluster.GetLogs(args[0])
			if err != nil {
				exitError(err)
			} else {
				numLogs := len(logs)
				for i, lines := range logs {
					for _, line := range lines {
						fmt.Println(line)
					}
					if i+1 < numLogs {
						fmt.Println("----")
					}
				}
			}
		},
	}

	cmd.Flags().StringP("cluster", "c", getDefaultCluster(), "the cluster for which to load the history")
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
			clusterId := args[0]

			// Get the onit controller
			controller, err := NewController()
			if err != nil {
				exitError(err)
			}

			// Get the cluster controller
			_, err = controller.GetCluster(clusterId)
			if err != nil {
				exitError(err)
			}

			// If we've made it this far, update the default cluster
			if err := setDefaultCluster(clusterId); err != nil {
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
			testId := fmt.Sprintf("test-%d", newUuidInt())

			// Get the onit controller
			controller, err := NewController()
			if err != nil {
				exitError(err)
			}

			// Get the cluster ID
			clusterId, err := cmd.Flags().GetString("cluster")
			if err != nil {
				exitError(err)
			}

			// Get the cluster controller
			cluster, err := controller.GetCluster(clusterId)
			if err != nil {
				exitError(err)
			}

			timeout, _ := cmd.Flags().GetInt("timeout")
			message, code, err := cluster.RunTests(testId, args, time.Duration(timeout)*time.Second)
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

// newUuidString returns a new string UUID
func newUuidString() string {
	id, err := uuid.NewUUID()
	if err != nil {
		exitError(err)
	}
	return id.String()
}

// newUuidInt returns a numeric UUID
func newUuidInt() uint32 {
	id, err := uuid.NewUUID()
	if err != nil {
		exitError(err)
	}
	return id.ID()
}

// exitError prints the given err to stdout and exits with exit code 1
func exitError(err error) {
	fmt.Println(err)
	os.Exit(1)
}
