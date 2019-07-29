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
	admin "github.com/onosproject/onos-config/pkg/northbound/proto"
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
	"Containers & Lists:\n" +
	"{{range .SchemaEntry}}" +
	"\t{{.SchemaPath}}\n" +
	"{{end}}" +
	"{{end}}" +
	"{{if .ReadOnlyPath}}" +
	"Read Only Paths:\n" +
	"{{range .ReadOnlyPath}}" +
	"\t{{.}}\n" +
	"{{end}}" +
	"{{end}}"

func newModelsCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "models {list|load}",
		Short: "Manages model plugins",
	}
	cmd.AddCommand(newListPluginsCommand())
	cmd.AddCommand(newLoadPluginCommand())
	return cmd
}

func newListPluginsCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "list <plugin path and filename>",
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
	client := admin.ConfigNewAdminServiceClient(getConnection(cmd))

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
			tmplModelList.Execute(os.Stdout, in)
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

func newLoadPluginCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "load <plugin path and filename>",
		Short: "Loads a new model plugin",
		Args:  cobra.MaximumNArgs(1),
		Run:   runLoadPluginCommand,
	}
	return cmd
}

func runLoadPluginCommand(cmd *cobra.Command, args []string) {
	client := admin.NewConfigAdminServiceClient(getConnection(cmd))
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
