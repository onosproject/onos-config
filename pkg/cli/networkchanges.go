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

const networkChangeTemplate = "CHANGE: {{.Type}} {{.Change.ID}} {{.Change.Index}} {{.Change.Revision}}\n" +
	"\tSTATE: Phase:{{.Change.Status.Phase}}\tState:{{.Change.Status.State}}\tReason:{{.Change.Status.Reason}}\tMsg:{{.Change.Status.Message}}\n" +
	"{{range .Change.Changes}}" +
	"\tDevice: {{.DeviceID}} ({{.DeviceVersion}})\n" +
	"{{range .Values}}" +
	"\t{{wrappath .Path 50 1| printf \"|%-50s|\"}}{{valuetostring .Value | printf \"(%s) %s\" .Value.Type | printf \"%-40s|\" }}{{printf \"%-7t|\" .Removed}}\n" +
	"{{end}}\n" +
	"{{end}}\n"

func getWatchNetworkChangesCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "network-changes <changeId>",
		Short: "Watch for network changes with updates",
		Args:  cobra.MaximumNArgs(1),
		RunE:  runWatchNetworkChangesCommand,
	}
	return cmd
}

func runWatchNetworkChangesCommand(cmd *cobra.Command, args []string) error {
	var id string
	if len(args) > 0 {
		id = args[0]
	}

	clientConnection, clientConnectionError := getConnection()

	if clientConnectionError != nil {
		return clientConnectionError
	}
	client := diags.CreateChangeServiceClient(clientConnection)
	changesReq := diags.ListNetworkChangeRequest{
		Subscribe: true,
		Changeid:  id,
	}

	tmplChanges, _ := template.New("change").Funcs(funcMapChanges).Parse(networkChangeTemplate)

	stream, err := client.ListNetworkChanges(context.Background(), &changesReq)
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
