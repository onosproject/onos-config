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
	"github.com/onosproject/onos-config/pkg/northbound/diags"
	types "github.com/onosproject/onos-config/pkg/types/change/device"
	"github.com/onosproject/onos-topo/pkg/northbound/device"
	"github.com/spf13/cobra"
	"io"
	"text/template"
)

const deviceChangeTemplate = changeHeaderFormat +
	"\t{{.NetworkChange.ID}}\t{{.NetworkChange.Index}}\t{{.Change.DeviceID}}\t{{.Change.DeviceVersion}}\n" +
	"{{range .Change.Values}}" + typedValueFormat + "{{end}}\n"

func getWatchDeviceChangesCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "device-changes <deviceid>",
		Short: "Watch for device changes with updates",
		Args:  cobra.ExactArgs(1),
		RunE:  runWatchDeviceChangesCommand,
	}
	cmd.Flags().StringP("version", "v", "", "an optional device version")
	cmd.Flags().Bool("no-headers", false, "disables output headers")
	return cmd
}

func runWatchDeviceChangesCommand(cmd *cobra.Command, args []string) error {
	id := types.ID(args[0]) // Argument is mandatory
	version, _ := cmd.Flags().GetString("version")
	noHeaders, _ := cmd.Flags().GetBool("no-headers")

	clientConnection, clientConnectionError := getConnection()

	if clientConnectionError != nil {
		return clientConnectionError
	}
	client := diags.CreateChangeServiceClient(clientConnection)
	changesReq := diags.ListDeviceChangeRequest{
		Subscribe:     true,
		DeviceID:      device.ID(id),
		DeviceVersion: version,
	}

	var tmplChanges *template.Template
	tmplChanges, _ = template.New("device").Funcs(funcMapChanges).Parse(deviceChangeTemplate)

	stream, err := client.ListDeviceChanges(context.Background(), &changesReq)
	if err != nil {
		return err
	}
	if !noHeaders {
		GetOutput().Write([]byte(changeHeader))
	}
	for {
		in, err := stream.Recv()
		if err == io.EOF {
			return nil
		}
		if err != nil {
			return err
		}
		_ = tmplChanges.Execute(GetOutput(), in.Change)
	}
}
