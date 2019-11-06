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
	"context"
	"fmt"
	"github.com/onosproject/onos-config/api/diags"
	"github.com/spf13/cobra"
	"io"
	"text/template"
)

const opstateTemplate = "{{wrappath .Pathvalue.Path 80 0| printf \"%-80s|\"}}" +
	"{{valuetostring .Pathvalue.Value | printf \"(%s) %s\" .Pathvalue.Value.Type | printf \"%-20s|\" }}"

func getGetOpstateCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "opstate <deviceid>",
		Short: "Get the Opstate cache for a device",
		Args:  cobra.ExactArgs(1),
		RunE:  runGetOpstateCommand,
	}
	cmd.Flags().Bool("no-headers", false, "disables output headers")
	return cmd
}

func getWatchOpstateCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "opstate <deviceid>",
		Short: "Watch the Opstate cache for a device",
		Args:  cobra.ExactArgs(1),
		RunE:  runWatchOpstateCommand,
	}
	cmd.Flags().Bool("no-headers", false, "disables output headers")
	return cmd
}

func runGetOpstateCommand(cmd *cobra.Command, args []string) error {
	return opstateCommand(cmd, false, args)
}

func runWatchOpstateCommand(cmd *cobra.Command, args []string) error {
	return opstateCommand(cmd, true, args)
}

func opstateCommand(cmd *cobra.Command, subscribe bool, args []string) error {
	deviceID := args[0]
	noHeaders, _ := cmd.Flags().GetBool("no-headers")
	tmplGetOpState, _ := template.New("change").Funcs(funcMapChanges).Parse(opstateTemplate)
	clientConnection, clientConnectionError := getConnection()

	if clientConnectionError != nil {
		return clientConnectionError
	}
	client := diags.CreateOpStateDiagsClient(clientConnection)

	if !noHeaders {
		Output("OPSTATE CACHE: %s\n", deviceID)
		Output("%-82s|%-20s|\n", "PATH", "VALUE")
	}

	stream, err := client.GetOpState(context.Background(), &diags.OpStateRequest{DeviceId: deviceID, Subscribe: subscribe})
	if err != nil {
		return fmt.Errorf("failed to send request: %v", err)
	}

	for {
		in, err := stream.Recv()
		if err == io.EOF {
			// read done.
			return nil
		}
		if err != nil {
			return err
		}
		_ = tmplGetOpState.Execute(GetOutput(), in)
		Output("\n")
	}
}
