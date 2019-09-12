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
	"encoding/base64"
	"github.com/onosproject/onos-config/pkg/northbound/diags"
	"github.com/onosproject/onos-config/pkg/store"
	"github.com/onosproject/onos-config/pkg/store/change"
	"github.com/spf13/cobra"
	"io"
	"os"
	"text/template"
)

const devicetreeTemplate = "DEVICE\t\t\tCONFIGURATION\t\tTYPE\t\tVERSION\n" +
	"{{printf \"%-22s  \" .Device}}{{printf \"%-22s  \" .Name}}{{printf \"%-14s  \" .Type}}{{printf \"%-6s  \" .Version}}\n" +
	"{{range .Changes}}" +
	"CHANGE:\t{{b64 .}}\n" +
	"{{end}}"

var funcMapDeviceTree = template.FuncMap{
	"b64": base64.StdEncoding.EncodeToString,
}

func getGetDeviceTreeCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "devicetree [--layer #] [<deviceId>]",
		Short: "Lists devices and their configuration in tree format",
		Args:  cobra.MaximumNArgs(1),
		Run:   runDeviceTreeCommand,
	}
	cmd.Flags().Int16("layer", 0, "layer of the configuration to retrieve; 0 is all including the latest, -1 is all up to previous")
	return cmd
}

func runDeviceTreeCommand(cmd *cobra.Command, args []string) {
	client := diags.NewConfigDiagsClient(getConnection())
	configReq := &diags.ConfigRequest{DeviceIDs: make([]string, 0)}
	if len(args) > 0 {
		configReq.DeviceIDs = append(configReq.DeviceIDs, args[0])
	}

	layer, err := cmd.Flags().GetInt16("layer")
	if err != nil {
		ExitWithErrorMessage("Failed to parse 'layer': %v\n", err)
	} else if layer > 0 {
		ExitWithErrorMessage("Layer must be less than or equal 0.\n")
	}

	stream, err := client.GetConfigurations(context.Background(), configReq)
	if err != nil {
		ExitWithErrorMessage("Failed to send request: %v", err)
	}

	configurations := make([]store.Configuration, 0)
	allChangeIds := make([]string, 0)

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
				ExitWithErrorMessage("Failed to receive response : %v", err)
			}

			changes := make([]change.ID, 0)
			for idx, ch := range in.ChangeIDs {
				if idx >= len(in.ChangeIDs)+int(layer) {
					continue
				}
				idBytes, _ := base64.StdEncoding.DecodeString(ch)
				changes = append(changes, change.ID(idBytes))
				allChangeIds = append(allChangeIds, store.B64(idBytes))
			}

			configuration, _ := store.CreateConfiguration(
				in.DeviceID, in.Version, in.DeviceType, changes)

			configuration.Updated = *in.Updated
			configuration.Created = *in.Updated

			configurations = append(configurations, *configuration)
		}
	}()
	err = stream.CloseSend()
	if err != nil {
		ExitWithErrorMessage("Failed to close: %v", err)
	}
	<-waitc

	if len(configurations) == 0 {
		ExitWithErrorMessage("Device(s) not found: %v\n", configReq.DeviceIDs)
	}

	changes := make(map[string]*change.Change)

	changesReq := &diags.ChangesRequest{ChangeIDs: allChangeIds}
	if len(args) == 1 {
		// Only add the changes for a specific device
		for _, ch := range configurations[0].Changes {
			changesReq.ChangeIDs = append(changesReq.ChangeIDs, base64.StdEncoding.EncodeToString(ch))
		}
	}

	stream2, err := client.GetChanges(context.Background(), changesReq)
	if err != nil {
		ExitWithErrorMessage("Failed to send request: %v", err)
	}
	waitc2 := make(chan struct{})
	go func() {
		for {
			in, err := stream2.Recv()
			if err == io.EOF {
				// read done.
				close(waitc2)
				return
			}
			if err != nil {
				ExitWithErrorMessage("Failed to receive response : %v", err)
			}
			idBytes, _ := base64.StdEncoding.DecodeString(in.Id)
			changeObj := change.Change{
				ID:          change.ID(idBytes),
				Description: in.Desc,
				Created:     *in.Time,
				Config:      make([]*change.Value, 0),
			}
			for _, cv := range in.ChangeValues {
				var tv *change.TypedValue
				typeOptInt32 := make([]int, len(cv.TypeOpts))
				for i, v := range cv.TypeOpts {
					typeOptInt32[i] = int(v)
				}
				tv = &change.TypedValue{
					Value:    cv.Value,
					Type:     change.ValueType(cv.ValueType),
					TypeOpts: typeOptInt32,
				}

				value, _ := change.CreateChangeValue(cv.Path, tv, cv.Removed)
				changeObj.Config = append(changeObj.Config, value)
			}
			changes[in.Id] = &changeObj
		}
	}()
	err = stream2.CloseSend()
	if err != nil {
		ExitWithErrorMessage("Failed to close: %v", err)
	}
	<-waitc2

	tmplDevicetreeList, _ := template.New("devices").Funcs(funcMapDeviceTree).Parse(devicetreeTemplate)
	for _, configuration := range configurations {
		_ = tmplDevicetreeList.Execute(os.Stdout, configuration)
		fullDeviceConfigValues := configuration.ExtractFullConfig(nil, changes, 0) // Passing 0 as change set has already been reduced by n='layer'
		jsonTree, _ := store.BuildTree(fullDeviceConfigValues, false)
		Output("TREE:\n%s\n\n", string(jsonTree))
	}

	ExitWithSuccess()
}
