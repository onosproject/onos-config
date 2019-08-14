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
	"github.com/onosproject/onos-config/pkg/northbound/diags"
	"github.com/spf13/cobra"
	"io"
	"os"
	"text/template"
)

const opstateTemplate = "{{wrappath .Pathvalue.Path 80 0| printf \"%-80s|\"}}" +
	"{{nativeType .Pathvalue | printf \"(%s) %s\" .Pathvalue.ValueType | printf \"%-20s|\" }}"

func getGetOpstateCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "opstate <deviceid> [--subscribe]",
		Short: "Gets the Opstate cache for a device",
		Args:  cobra.ExactArgs(1),
		Run:   runOpstateCommand,
	}
	cmd.Flags().BoolP("subscribe", "s", false, "subscribe for subsequent changes")
	return cmd
}

func runOpstateCommand(cmd *cobra.Command, args []string) {
	deviceID := args[0]
	subscribe, _ := cmd.Flags().GetBool("subscribe")
	tmplGetOpState, _ := template.New("change").Funcs(funcMapChanges).Parse(opstateTemplate)
	client := diags.NewOpStateDiagsClient(getConnection())

	fmt.Printf("OPSTATE CACHE: %s\n", deviceID)
	fmt.Printf("%-82s|%-20s|\n", "PATH", "VALUE")

	stream, err := client.GetOpState(context.Background(), &diags.OpStateRequest{DeviceId: deviceID, Subscribe: subscribe})
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
			_ = tmplGetOpState.Execute(os.Stdout, in)
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
