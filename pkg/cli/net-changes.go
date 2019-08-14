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
	"github.com/onosproject/onos-config/pkg/northbound/admin"
	"github.com/spf13/cobra"
	"io"
)

func getGetNetChangesCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "net-changes",
		Short: "Lists network configuration changes",
		Args:  cobra.ExactArgs(0),
		Run:   runNetChangesCommand,
	}
	return cmd
}

func runNetChangesCommand(cmd *cobra.Command, args []string) {
	client := admin.NewConfigAdminServiceClient(getConnection())

	stream, err := client.GetNetworkChanges(context.Background(), &admin.NetworkChangesRequest{})
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
			Output("%s: %s (%s)\n", in.Time,
				in.Name, in.User)
			for _, c := range in.Changes {
				Output("\t%s: %s\n", c.Id, c.Hash)
			}
		}
	}()
	err = stream.CloseSend()
	if err != nil {
		ExitWithErrorMessage("Failed to close: %v", err)
	}
	<-waitc
	ExitWithSuccess()
}
