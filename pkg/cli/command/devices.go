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
	"context"
	"github.com/golang/protobuf/proto"
	devices "github.com/onosproject/onos-config/pkg/northbound/proto"
	"github.com/spf13/cobra"
	"io"
	"os"
	"text/template"
)

const devicelistTemplate = "{{printf \"%-23s \" .Id}}{{printf \"%-23s \" .Address}}{{printf \"%-23s \" .Version}}\n"
const devicelistTemplateVerbose = "\tUSER\t\tPASSWORD\tTIMEOUT\tPLAIN\tINSECURE\n" +
	"\t{{printf \"%-15s \" .User}}{{printf \"%-15s \" .Password}}{{printf \"%-7d \" .Timeout}}{{.Plain}}\t{{.Insecure}}\n\n"

func newDevicesCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "devices {list,add,remove}",
		Short: "Manages inventory of network devices",
	}
	cmd.AddCommand(newDevicesListCommand())
	cmd.AddCommand(newDevicesAddCommand())
	cmd.AddCommand(newDevicesRemoveCommand())
	return cmd
}

func newDevicesListCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "list",
		Short: "Lists devices in the inventory",
		Args:  cobra.ExactArgs(0),
		Run:   runDevicesListCommand,
	}
	cmd.Flags().BoolP("verbose", "v", false, "display verbose output")
	return cmd
}

func runDevicesListCommand(cmd *cobra.Command, args []string) {
	client := devices.NewDeviceInventoryServiceClient(getConnection(cmd))
	stream, err := client.GetDevices(context.Background(), &devices.GetDevicesRequest{})
	verbose, _ := cmd.Flags().GetBool("verbose")

	tmplDevicesList, _ := template.New("devices").Parse(devicelistTemplate)
	tmplDevicesListVerbose, _ := template.New("devicesverbose").Parse(devicelistTemplateVerbose)
	Output("NAME\t\t\tADDRESS\t\t\tVERSION\n")

	if err != nil {
		ExitWithErrorMessage("Failed to get devices: %v\n", err)
	}
	waitc := make(chan struct{})
	go func() {
		for {
			in, err := stream.Recv()
			if err == io.EOF {
				// read done.
				close(waitc)
				return
			}
			if err != nil {
				ExitWithErrorMessage("Failed to receive device: %v\n", err)
			}
			_ = tmplDevicesList.Execute(os.Stdout, in)
			if verbose {
				_ = tmplDevicesListVerbose.Execute(os.Stdout, in)
			}
		}
	}()
	err = stream.CloseSend()
	if err != nil {
		ExitWithErrorMessage("Failed to close: %v", err)
	}
	<-waitc
	ExitWithSuccess()
}

func newDevicesAddCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "add <deviceInfoProtobuf>",
		Short: "Adds a new device to the inventory",
		Args:  cobra.ExactArgs(1),
		Run:   runDeviceAddCommand,
	}
}

func runDeviceAddCommand(cmd *cobra.Command, args []string) {
	client := devices.NewDeviceInventoryServiceClient(getConnection(cmd))
	deviceInfo := &devices.DeviceInfo{}
	if err := proto.UnmarshalText(args[0], deviceInfo); err == nil {
		_, err = client.AddOrUpdateDevice(context.Background(), deviceInfo)
		if err != nil {
			ExitWithErrorMessage("Unable to add device: %v\n", err)
		}
	} else {
		ExitWithErrorMessage("Unable to parse device info from %s: %v\n", args[0], err)
	}
	ExitWithOutput("Added device %s\n", deviceInfo.Id)
}

func newDevicesRemoveCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "remove <deviceId>",
		Short: "Removes a device from the inventory",
		Args:  cobra.ExactArgs(1),
		Run:   runDeviceRemoveCommand,
	}
}

func runDeviceRemoveCommand(cmd *cobra.Command, args []string) {
	client := devices.NewDeviceInventoryServiceClient(getConnection(cmd))
	_, err := client.RemoveDevice(context.Background(), &devices.DeviceInfo{Id: args[0]})
	if err != nil {
		ExitWithErrorMessage("Unable to remove device: %v\n", err)
	}
	ExitWithOutput("Removed device %s\n", args[0])
}
