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
	devicechange "github.com/onosproject/onos-config/api/types/change/device"
	"github.com/onosproject/onos-config/api/types/device"
	"github.com/onosproject/onos-lib-go/pkg/cli"
	"github.com/spf13/cobra"
	"io"
	"strings"
	"text/template"
)

const deviceChangeTemplate = changeHeaderFormat +
	"\t{{.NetworkChange.ID}}\t{{.NetworkChange.Index}}\t{{.Change.DeviceID}}\t{{.Change.DeviceVersion}}\n" +
	"{{range .Change.Values}}" + typedValueFormat + "{{end}}\n"

func getWatchDeviceChangesCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "device-changes <deviceid>",
		Short: "Watch for changes on a device with updates",
		Args:  cobra.ExactArgs(1),
		RunE:  runWatchDeviceChangesCommand,
	}
	cmd.Flags().StringP("version", "v", "", "an optional version")
	cmd.Flags().Bool("no-headers", false, "disables output headers")
	return cmd
}

func getListDeviceChangesCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "device-changes <deviceid>",
		Short: "List current changes on a device",
		Args:  cobra.ExactArgs(1),
		RunE:  runListDeviceChangesCommand,
	}
	cmd.Flags().StringP("version", "v", "", "device version")
	cmd.Flags().Bool("no-headers", false, "disables output headers")
	return cmd
}

func runWatchDeviceChangesCommand(cmd *cobra.Command, args []string) error {
	return deviceChangesCommand(cmd, true, args)
}

func runListDeviceChangesCommand(cmd *cobra.Command, args []string) error {
	return deviceChangesCommand(cmd, false, args)
}

func deviceChangesCommand(cmd *cobra.Command, subscribe bool, args []string) error {
	id := args[0] // Argument is mandatory
	version, _ := cmd.Flags().GetString("version")
	noHeaders, _ := cmd.Flags().GetBool("no-headers")

	clientConnection, clientConnectionError := cli.GetConnection(cmd)

	if clientConnectionError != nil {
		return clientConnectionError
	}
	client := diags.CreateChangeServiceClient(clientConnection)
	changesReq := diags.ListDeviceChangeRequest{
		Subscribe:     subscribe,
		DeviceID:      device.ID(id),
		DeviceVersion: device.Version(version),
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

func wrapPath(path string, lineLen int, tabs int) string {
	over := len(path) % lineLen
	linesC := len(path) / lineLen
	var wrapped []string
	if over > 0 {
		wrapped = make([]string, linesC+1)
	} else {
		wrapped = make([]string, linesC)
	}
	var i int
	for i = 0; i < linesC; i++ {
		wrapped[i] = path[i*lineLen : i*lineLen+lineLen]
	}
	if over > 0 {
		overFmt := fmt.Sprintf("%s-%ds", "%", lineLen)
		wrapped[linesC] = fmt.Sprintf(overFmt, path[linesC*lineLen:])
	}

	tabsArr := make([]string, tabs+1)
	sep := fmt.Sprintf("\n%s  ", strings.Join(tabsArr, "\t"))
	return strings.Join(wrapped, sep)
}

func valueToSstring(cv devicechange.TypedValue) string {
	return cv.ValueToString()
}
