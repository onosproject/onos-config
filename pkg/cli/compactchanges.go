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

func getCompactCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "compact-changes",
		Short: "Takes a snapshot of network and device changes",
		Args:  cobra.NoArgs,
		RunE:  runCompactCommand,
	}
	cmd.Flags().DurationP("retention-period", "r", 0, "the period for which to retain all changes")
	return cmd
}

func runCompactCommand(cmd *cobra.Command, args []string) error {
	clientConnection, clientConnectionError := cli.GetConnection(cmd)

	if clientConnectionError != nil {
		return clientConnectionError
	}
	client := admin.CreateConfigAdminServiceClient(clientConnection)

	retentionPeriod, _ := cmd.Flags().GetDuration("retention-period")
	request := &admin.CompactChangesRequest{}
	if retentionPeriod != 0 {
		request.RetentionPeriod = &retentionPeriod
	}
	_, err := client.CompactChanges(context.Background(), request)
	if err != nil {
		return err
	}
	cli.Output("Completed compaction\n")
	return nil
}
