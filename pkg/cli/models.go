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
	"os"
	"text/template"
)

const modellistTemplate = "{{.Name}}: {{.Version}} from {{.Module}} containing:\n" +
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
		Run:   runListPluginsCommand,
	}
	cmd.Flags().BoolP("verbose", "v", false, "display verbose output")
	return cmd
}

func runListPluginsCommand(cmd *cobra.Command, args []string) {
	verbose, _ := cmd.Flags().GetBool("verbose")
	tmplModelList, _ := template.New("change").Parse(modellistTemplate)
	client := admin.NewConfigAdminServiceClient(getConnection())

	stream, err := client.ListRegisteredModels(context.Background(), &admin.ListModelsRequest{Verbose: verbose})
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
			_ = tmplModelList.Execute(os.Stdout, in)
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

func getAddPluginCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "plugin <plugin path and filename>",
		Short: "Loads a new model plugin from server",
		Args:  cobra.ExactArgs(1),
		Run:   runAddPluginCommand,
	}
	return cmd
}

func runAddPluginCommand(cmd *cobra.Command, args []string) {
	client := admin.NewConfigAdminServiceClient(getConnection())
	pluginFileName := ""
	if len(args) == 1 {
		pluginFileName = args[0]
	}

	resp, err := client.RegisterModel(
		context.Background(), &admin.RegisterRequest{SoFile: pluginFileName})
	if err != nil {
		ExitWithErrorMessage("Failed to send request: %v", err)
	}
	Output("load plugin success %s %s\n", resp.Name, resp.Version)
	ExitWithSuccess()
}
