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
	"github.com/onosproject/onos-config/api/admin"
	"github.com/onosproject/onos-lib-go/pkg/cli"
	"github.com/spf13/cobra"
)

func getRollbackCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "rollback <changeId>",
		Short: "Rolls-back a network change",
		Args:  cobra.MaximumNArgs(1),
		RunE:  runRollbackCommand,
	}
	return cmd
}

func runRollbackCommand(cmd *cobra.Command, args []string) error {
	clientConnection, clientConnectionError := cli.GetConnection(cmd)

	if clientConnectionError != nil {
		return clientConnectionError
	}
	client := admin.CreateConfigAdminServiceClient(clientConnection)

	changeID := ""
	if len(args) == 1 {
		changeID = args[0]
	}

	resp, err := client.RollbackNetworkChange(
		context.Background(), &admin.RollbackRequest{Name: changeID})
	if err != nil {
		return err
	}
	Output("Rollback success %s\n", resp.Message)
	return nil
}
