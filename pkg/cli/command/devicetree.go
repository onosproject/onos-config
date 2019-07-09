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
	"encoding/base64"
	diags "github.com/onosproject/onos-config/pkg/northbound/proto"
	"github.com/onosproject/onos-config/pkg/store"
	"github.com/onosproject/onos-config/pkg/store/change"
	"github.com/spf13/cobra"
	"io"
	"time"
)

func newDeviceTreeCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "devicetree [-version #] [<deviceId>]",
		Short: "Lists devices and their configuration in tree format",
		Args:  cobra.MaximumNArgs(1),
		Run:   runDeviceTreeCommand,
	}
	cmd.Flags().Int32("version", 0, "version of the configuration to retrieve; 0 is the latest, -1 is the previous")
	return cmd
}

func runDeviceTreeCommand(cmd *cobra.Command, args []string) {
	client := diags.NewConfigDiagsClient(getConnection(cmd))
	configReq := &diags.ConfigRequest{DeviceIds: make([]string, 0)}
	if len(args) > 0 {
		configReq.DeviceIds = append(configReq.DeviceIds, args[0])
	}

	version, _ := cmd.Flags().GetInt("version")

	stream, err := client.GetConfigurations(context.Background(), configReq)
	if err != nil {
		ExitWithErrorMessage("Failed to send request: %v", err)
	}

	configurations := make([]store.Configuration, 0)

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

			changes := make([]change.ID, len(in.ChangeIDs))
			for idx, ch := range in.ChangeIDs {
				changes[idx] = change.ID(ch)
			}

			configuration, _ := store.CreateConfiguration(
				in.Deviceid, in.Version, in.Devicetype, changes)

			configuration.Updated = time.Unix(in.Updated.Seconds, int64(in.Updated.Nanos))
			configuration.Created = time.Unix(in.Updated.Seconds, int64(in.Updated.Nanos))

			for _, cid := range in.ChangeIDs {
				idBytes, _ := base64.StdEncoding.DecodeString(cid)
				configuration.Changes = append(configuration.Changes, change.ID(idBytes))
			}

			configurations = append(configurations, *configuration)
		}
	}()
	err = stream.CloseSend()
	if err != nil {
		ExitWithErrorMessage("Failed to close: %v", err)
	}
	<-waitc

	changes := make(map[string]*change.Change)

	changesReq := &diags.ChangesRequest{ChangeIds: make([]string, 0)}
	if len(args) == 1 {
		// Only add the changes for a specific device
		for _, ch := range configurations[0].Changes {
			changesReq.ChangeIds = append(changesReq.ChangeIds, base64.StdEncoding.EncodeToString(ch))
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
			Output("Received change %s", in.Id)
			idBytes, _ := base64.StdEncoding.DecodeString(in.Id)
			changeObj := change.Change{
				ID:          change.ID(idBytes),
				Description: in.Desc,
				Created:     time.Unix(in.Time.Seconds, int64(in.Time.Nanos)),
				Config:      make([]*change.Value, 0),
			}
			for _, cv := range in.Changevalues {
				var tv *change.TypedValue
				typeOptInt32 := make([]int, len(cv.Typeopts))
				for i, v := range cv.Typeopts {
					typeOptInt32[i] = int(v)
				}
				tv = &change.TypedValue{
					Value:    cv.Value,
					Type:     change.ValueType(cv.Valuetype),
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

	for _, configuration := range configurations {
		Output("Config %s (Device: %s)\n", configuration.Name, configuration.Device)
		fullDeviceConfigValues := configuration.ExtractFullConfig(changes, version)
		jsonTree, _ := store.BuildTree(fullDeviceConfigValues, false)
		Output("%s\n", string(jsonTree))
	}

	ExitWithSuccess()
}
