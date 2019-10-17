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
	"github.com/onosproject/onos-topo/pkg/northbound/device"
	"github.com/spf13/cobra"
	"io"
)

func getSnapshotCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "snapshot",
		Aliases: []string{"snapshots"},
		Short:   "Commands for managing snapshots",
		Args:    cobra.MaximumNArgs(1),
		RunE:    runSnapshotCommand,
	}
	return cmd
}

func runSnapshotCommand(cmd *cobra.Command, args []string) error {
	clientConnection, clientConnectionError := getConnection()

	if clientConnectionError != nil {
		return clientConnectionError
	}
	client := admin.CreateConfigAdminServiceClient(clientConnection)

	if len(args) == 0 {
		stream, err := client.ListSnapshots(context.Background(), &admin.ListSnapshotsRequest{})
		if err != nil {
			return err
		}

		for {
			response, err := stream.Recv()
			if err == io.EOF {
				return nil
			} else if err != nil {
				return err
			}
			Output("%s\n", response.DeviceID)
		}
	} else {
		snapshot, err := client.GetSnapshot(context.Background(), &admin.GetSnapshotRequest{
			DeviceID: device.ID(args[0]),
		})
		if err != nil {
			return err
		}
		Output("%v", snapshot)
	}
	return nil
}

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
	clientConnection, clientConnectionError := getConnection()

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
	Output("Completed compaction\n")
	return nil
}
