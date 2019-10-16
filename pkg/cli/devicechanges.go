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
	"github.com/spf13/cobra"
	"io"
	"text/template"
)

const deviceChangeTemplate = "CHANGE: {{.Type}} {{.Change.ID}} {{.Change.Index}} {{.Change.Revision}}\n" +
	"\tSTATE: Phase:{{.Change.Status.Phase}}\tState:{{.Change.Status.State}}\tReason:{{.Change.Status.Reason}}\tMsg:{{.Change.Status.Message}}\n" +
	"\tDevice: {{.Change.Change.DeviceID}} ({{.Change.Change.DeviceVersion}})\n" +
	"{{range .Change.Change.Values}}" +
	"\t{{wrappath .Path 50 1| printf \"|%-50s|\"}}{{valuetostring .Value | printf \"(%s) %s\" .Value.Type | printf \"%-40s|\" }}{{printf \"%-7t|\" .Removed}}\n" +
	"{{end}}\n"

func getWatchDeviceChangesCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "device-changes <changeId>",
		Short: "Watch for device changes with updates",
		Args:  cobra.MaximumNArgs(1),
		RunE:  runWatchDeviceChangesCommand,
	}
	return cmd
}

func runWatchDeviceChangesCommand(cmd *cobra.Command, args []string) error {
	var id string
	if len(args) > 0 {
		id = args[0]
	}

	clientConnection, clientConnectionError := getConnection()

	if clientConnectionError != nil {
		return clientConnectionError
	}
	client := diags.CreateChangeServiceClient(clientConnection)
	changesReq := diags.ListDeviceChangeRequest{
		Subscribe: true,
		Changeid:  id,
	}

	tmplChanges, _ := template.New("change").Funcs(funcMapChanges).Parse(deviceChangeTemplate)

	stream, err := client.ListDeviceChanges(context.Background(), &changesReq)
	if err != nil {
		return err
	}

	for {
		in, err := stream.Recv()
		if err == io.EOF {
			return nil
		}
		if err != nil {
			return err
		}
		_ = tmplChanges.Execute(GetOutput(), in)
	}
}
