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
	"github.com/onosproject/onos-config/api/admin"
	"github.com/spf13/cobra"
	"io"
	"os"
	"path/filepath"
	"text/template"
)

const modellistTemplate = "{{.Name}}: {{.Version}} from {{.Module}} containing:\n" +
	"GetStateMode: {{.GetStateMode}}\n" +
	"YANGS:\n" +
	"{{range .ModelData}}" +
	"\t{{.Name}}\t{{.Version}}\t{{.Organization}}\n" +
	"{{end}}" +
	"{{if .ReadOnlyPath}}" +
	"Read Only Paths (subpaths not shown):\n" +
	"{{range .ReadOnlyPath}}" +
	"\t{{.Path}}\n" +
	"{{end}}" +
	"{{end}}" +
	"{{if .ReadWritePath}}" +
	"Read Write Paths (details not shown):\n" +
	"{{range .ReadWritePath}}" +
	"\t{{.Path}}\n" +
	"{{end}}" +
	"{{end}}"

func getGetPluginsCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "plugins",
		Short: "Lists the loaded model plugins",
		Args:  cobra.MaximumNArgs(1),
		RunE:  runListPluginsCommand,
	}
	cmd.Flags().BoolP("verbose", "v", false, "display verbose output")
	return cmd
}

func runListPluginsCommand(cmd *cobra.Command, args []string) error {
	verbose, _ := cmd.Flags().GetBool("verbose")
	tmplModelList, _ := template.New("change").Parse(modellistTemplate)
	clientConnection, clientConnectionError := getConnection(cmd)

	if clientConnectionError != nil {
		return clientConnectionError
	}
	client := admin.CreateConfigAdminServiceClient(clientConnection)

	stream, err := client.ListRegisteredModels(context.Background(), &admin.ListModelsRequest{Verbose: verbose})
	if err != nil {
		return fmt.Errorf("Failed to send request: %v", err)
	}

	for {
		in, err := stream.Recv()
		if err == io.EOF {
			return nil
		}
		if err != nil {
			return err
		}
		_ = tmplModelList.Execute(GetOutput(), in)
		Output("\n")
	}
}

func getAddPluginCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "plugin <plugin path and filename>",
		Short: "Loads a new model plugin",
		Args:  cobra.ExactArgs(1),
		RunE:  runAddPluginCommand,
	}
	return cmd
}

func runAddPluginCommand(cmd *cobra.Command, args []string) error {
	clientConnection, clientConnectionError := getConnection(cmd)

	if clientConnectionError != nil {
		return clientConnectionError
	}

	pluginFileName := ""
	if len(args) == 1 {
		pluginFileName = args[0]
	}
	pluginFile, err := os.Open(pluginFileName)
	if err != nil {
		return err
	}
	defer pluginFile.Close()

	client := admin.CreateConfigAdminServiceClient(clientConnection)

	uploadClient, err := client.UploadRegisterModel(context.Background())
	if err != nil {
		return err
	}

	const chunkSize = 2000000
	data := make([]byte, chunkSize)
	var chunkNo int64 = 0
	for {
		count, err := pluginFile.ReadAt(data, chunkNo*chunkSize)
		if err == io.EOF {
			err := uploadClient.Send(&admin.Chunk{
				SoFile:  filepath.Base(pluginFileName),
				Content: data[:count],
			})
			if err != nil {
				return err
			}
			resp, err := uploadClient.CloseAndRecv()
			if err != nil {
				return err
			}
			Output("Plugin %s uploaded to server as %s:%s. %d chunks. %d bytes\n",
				pluginFileName, resp.GetName(), resp.GetVersion(),
				chunkNo, chunkNo*chunkSize+int64(count))
			return nil
		}
		if err != nil {
			return err
		}
		err = uploadClient.Send(&admin.Chunk{
			SoFile:  filepath.Base(pluginFileName),
			Content: data,
		})
		if err != nil {
			return err
		}
		chunkNo++
	}
}
