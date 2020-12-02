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
	"github.com/onosproject/onos-api/go/onos/config/admin"
	device_snapshot "github.com/onosproject/onos-api/go/onos/config/snapshot/device"
	"github.com/onosproject/onos-lib-go/pkg/cli"
	"github.com/spf13/cobra"
	"io"
	"text/template"
)

const snapshotHeader = "ID                      DEVICE          VERSION TYPE        INDEX SNAPSHOTID\n"

const snapshotsHeaderFormat = "{{printf \"%-24s %-16s %-8s %-12s %-6d %s\" .ID .DeviceID .DeviceVersion .DeviceType .ChangeIndex .SnapshotID}}\n"

const pathValueTemplate = "{{wrappath .Path 80 0| printf \"%-80s|\"}}" +
	"{{valuetostring .Value | printf \"(%s) %s\" .Value.Type | printf \"%-40s|\" }}"

const snapshotsTemplate = snapshotsHeaderFormat

const snapshotsTemplateVerbose = snapshotsHeaderFormat +
	"{{range .Values}}" + pathValueTemplate + "\n{{end}}\n"

func getWatchSnapshotsCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "snapshots [id wildcard]",
		Short: "Watch for snapshots with updates",
		Args:  cobra.MaximumNArgs(1),
		RunE:  runWatchSnapshotsCommand,
	}
	cmd.Flags().BoolP("verbose", "v", false, "whether to print the change with verbose output")
	cmd.Flags().Bool("no-headers", false, "disables output headers")
	return cmd
}

func getListSnapshotsCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "snapshots [id wildcard]",
		Short: "List snapshots",
		Args:  cobra.MaximumNArgs(1),
		RunE:  runListSnapshotsCommand,
	}
	cmd.Flags().BoolP("verbose", "v", false, "whether to print the change with verbose output")
	cmd.Flags().Bool("no-headers", false, "disables output headers")
	return cmd
}

func runWatchSnapshotsCommand(cmd *cobra.Command, args []string) error {
	return snapshotsCommand(cmd, true, args)
}

func runListSnapshotsCommand(cmd *cobra.Command, args []string) error {
	return snapshotsCommand(cmd, false, args)
}

func snapshotsCommand(cmd *cobra.Command, subscribe bool, args []string) error {
	var id device_snapshot.ID
	if len(args) > 0 {
		id = device_snapshot.ID(args[0])
	}
	verbose, _ := cmd.Flags().GetBool("verbose")
	noHeaders, _ := cmd.Flags().GetBool("no-headers")

	clientConnection, clientConnectionError := cli.GetConnection(cmd)

	if clientConnectionError != nil {
		return clientConnectionError
	}
	client := admin.CreateConfigAdminServiceClient(clientConnection)
	snapshotsRequest := admin.ListSnapshotsRequest{
		Subscribe: subscribe,
		ID:        id,
	}

	var tmplSnapshots *template.Template
	tmplSnapshots, _ = template.New("snapshots").Funcs(funcMapChanges).Parse(snapshotsTemplate)
	if verbose {
		tmplSnapshots, _ = template.New("snapshots").Funcs(funcMapChanges).Parse(snapshotsTemplateVerbose)
	}

	stream, err := client.ListSnapshots(context.Background(), &snapshotsRequest)
	if err != nil {
		return err
	}

	if !noHeaders {
		cli.GetOutput().Write([]byte(snapshotHeader))
	}
	for {
		in, err := stream.Recv()
		if err == io.EOF {
			return nil
		}
		if err != nil {
			return err
		}
		err = tmplSnapshots.Execute(cli.GetOutput(), in)
		if err != nil {
			cli.Output("ERROR on template: %s", snapshotsTemplate)
			return err
		}
	}
}
