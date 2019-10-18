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
	types "github.com/onosproject/onos-config/pkg/types/change/network"
	"github.com/spf13/cobra"
	"io"
	"text/template"
)

const changeHeader = "CHANGE                          INDEX  REVISION  PHASE    STATE     REASON   MESSAGE\n"

const changeHeaderFormat = "{{printf \"%-31v %-7d %-8d %-8s %-9s %-8s %s\" .ID .Index .Revision .Status.Phase .Status.State .Status.Reason .Status.Message}}\n"

const typedValueFormat = "\t{{wrappath .Path 50 1| printf \"|%-50s|\"}}{{valuetostring .Value | printf \"(%s) %s\" .Value.Type | printf \"%-40s|\" }}{{printf \"%-7t|\" .Removed}}\n"

const deviceIDFormat = "Device: {{.DeviceID}} ({{.DeviceVersion}})"

const networkChangeTemplate = changeHeaderFormat +
	"{{range .Changes}}\t" + deviceIDFormat + "\n{{end}}\n"

const networkChangeTemplateVerbose = changeHeaderFormat +
	"{{range .Changes}}\t" + deviceIDFormat + "\n" +
	"{{range .Values}}" + typedValueFormat + "{{end}}\n" +
	"{{end}}\n"

func getWatchNetworkChangesCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "network-changes <changeId>",
		Short: "Watch for network changes with updates",
		Args:  cobra.MaximumNArgs(1),
		RunE:  runWatchNetworkChangesCommand,
	}
	cmd.Flags().BoolP("verbose", "v", false, "whether to print the device with verbose output")
	cmd.Flags().Bool("no-headers", false, "disables output headers")
	return cmd
}

func runWatchNetworkChangesCommand(cmd *cobra.Command, args []string) error {
	var id types.ID
	if len(args) > 0 {
		id = types.ID(args[0])
	}
	verbose, _ := cmd.Flags().GetBool("verbose")
	noHeaders, _ := cmd.Flags().GetBool("no-headers")

	clientConnection, clientConnectionError := getConnection()

	if clientConnectionError != nil {
		return clientConnectionError
	}
	client := diags.CreateChangeServiceClient(clientConnection)
	changesReq := diags.ListNetworkChangeRequest{
		Subscribe: true,
		ChangeID:  id,
	}

	var tmplChanges *template.Template
	tmplChanges, _ = template.New("change").Funcs(funcMapChanges).Parse(networkChangeTemplate)
	if verbose {
		tmplChanges, _ = template.New("change").Funcs(funcMapChanges).Parse(networkChangeTemplateVerbose)
	}

	stream, err := client.ListNetworkChanges(context.Background(), &changesReq)
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
