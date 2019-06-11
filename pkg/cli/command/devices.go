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
	"fmt"
	"github.com/spf13/cobra"
)

func newDevicesCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "devices {list,add,remove}",
		Short: "Manages the devices inventory",
	}
	cmd.AddCommand(newDevicesListCommand())
	cmd.AddCommand(newDevicesAddCommand())
	cmd.AddCommand(newDevicesRemoveCommand())
	return cmd
}

func newDevicesListCommand() *cobra.Command {
	return &cobra.Command{
		Use:  "list <device info protobuf>",
		Args: cobra.ExactArgs(0),
		Run:  runDevicesListCommand,
	}
}

func runDevicesListCommand(cmd *cobra.Command, args []string) {
	ExitWithOutput(fmt.Sprintf("List"))
}


func newDevicesAddCommand() *cobra.Command {
	return &cobra.Command{
		Use:  "add <device info protobuf>",
		Args: cobra.ExactArgs(1),
		Run:  runDeviceAddCommand,
	}
}

func runDeviceAddCommand(cmd *cobra.Command, args []string) {
	ExitWithOutput(fmt.Sprintf("Add %s", args[0]))
}

func newDevicesRemoveCommand() *cobra.Command {
	return &cobra.Command{
		Use:  "remove <device ID>",
		Args: cobra.ExactArgs(1),
		Run:  runDeviceRemoveCommand,
	}
}

func runDeviceRemoveCommand(cmd *cobra.Command, args []string) {
	ExitWithOutput(fmt.Sprintf("Remove %s", args[0]))
}
