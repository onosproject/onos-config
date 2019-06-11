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
	diags "github.com/onosproject/onos-config/pkg/northbound/proto"
	"github.com/spf13/cobra"
	"io"
	"time"
)

func newChangesCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "changes [<changeId>]",
		Short: "Lists records of configuration changes",
		Args:  cobra.MaximumNArgs(0),
		Run:   runChangesCommand,
	}
	return cmd
}

func runChangesCommand(cmd *cobra.Command, args []string) {
	client := diags.NewConfigDiagsClient(getConnection(cmd))
	changesReq := &diags.ChangesRequest{ChangeIds: make([]string, 0)}
	if len(args) == 1 {
		changesReq.ChangeIds = append(changesReq.ChangeIds, args[0])
	}

	stream, err := client.GetChanges(context.Background(), changesReq)
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
			Output("%s\t(%s)\t%s\n", in.Id, in.Desc,
				time.Unix(in.Time.Seconds, int64(in.Time.Nanos)).Format(time.RFC3339))
			for _, cv := range in.Changevalues {
				Output("\t%s\t%s\t%v\n", cv.Path, cv.Value, cv.Removed)
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
