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
	"time"
)

func getGetConfigsCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "configs [<deviceId>]",
		Short: "Lists details of device configuration changes",
		Args:  cobra.MaximumNArgs(0),
		Run:   runGetConfigsCommand,
	}
	return cmd
}

func runGetConfigsCommand(cmd *cobra.Command, args []string) {
	client := diags.NewConfigDiagsClient(getConnection())

	configReq := &diags.ConfigRequest{DeviceIDs: make([]string, 0)}
	if len(args) == 1 {
		configReq.DeviceIDs = append(configReq.DeviceIDs, args[0])
	}
	stream, err := client.GetConfigurations(context.Background(), configReq)
	if err != nil {
		ExitWithErrorMessage("Failed to send request: %v", err)
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
				ExitWithErrorMessage("Failed to receive response : %v", err)
			}
			Output("%s\t(%s)\t%s\t%s\t%s\n", in.Name, in.DeviceID, in.Version, in.DeviceType,
				in.Updated.Format(time.RFC3339))
			for _, cid := range in.ChangeIDs {
				Output("\t%s", cid)
			}
			Output("\n")
		}
	}()
	err = stream.CloseSend()
	if err != nil {
		ExitWithErrorMessage("Failed to close: %v", err)
	}
	<-waitc
	ExitWithSuccess()
}
