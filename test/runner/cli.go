package runner

import (
	"fmt"
	"github.com/spf13/cobra"
	"os"
	"time"
)

// GetCommand returns a Cobra command for tests in the given test registry
func GetCommand(registry *TestRegistry) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "onos-it {list,test,run} [tests]",
		Short: "Run onos-config integration tests on Kubernetes",
	}
	cmd.AddCommand(getRunCommand(registry))
	cmd.AddCommand(getTestCommand(registry))
	cmd.AddCommand(getListCommand(registry))
	return cmd
}

// getRunCommand returns a cobra "run" command
func getRunCommand(registry *TestRegistry) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "run [tests]",
		Short: "Run integration tests on Kubernetes",
		Run: func(cmd *cobra.Command, args []string) {
			configName, _ := cmd.Flags().GetString("config")
			nodes, _ := cmd.Flags().GetInt("nodes")
			partitions, _ := cmd.Flags().GetInt("partitions")
			partitionSize, _ := cmd.Flags().GetInt("partitionSize")
			timeout, _ := cmd.Flags().GetInt("timeout")
			config := &KubeControllerConfig{
				Config:        configName,
				Nodes:         nodes,
				Partitions:    partitions,
				PartitionSize: partitionSize,
				Timeout:       time.Duration(timeout) * time.Second,
			}
			controller, err := NewKubeController(config)
			if err != nil {
				fmt.Println(err)
				os.Exit(1)
			}
			controller.Run(args)
		},
	}
	cmd.Flags().StringP("config", "c", "default", "test cluster configuration")
	cmd.Flags().IntP("timeout", "t", 60*10, "test timeout in seconds")
	cmd.Flags().IntP("nodes", "n", 1, "the number of onos-config nodes to deploy")
	cmd.Flags().IntP("partitions", "p", 1, "the number of Raft partitions to deploy")
	cmd.Flags().IntP("partitionSize", "r", 1, "the size of each Raft partition")
	cmd.Flags().IntP("simulators", "s", 1, "the number of simulators to deploy")
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
				fmt.Println(err)
				os.Exit(1)
			} else {
				os.Exit(0)
			}
		},
	}
}

// getListCommand returns a cobra "list" command for tests in the given registry
func getListCommand(registry *TestRegistry) *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List integration tests",
		Run: func(cmd *cobra.Command, args []string) {
			for _, name := range registry.GetNames() {
				fmt.Println(name)
			}
		},
	}
}
