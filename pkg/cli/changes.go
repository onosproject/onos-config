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
	"github.com/onosproject/onos-config/pkg/northbound/admin"
	"github.com/onosproject/onos-config/pkg/northbound/diags"
	"github.com/onosproject/onos-config/pkg/store/change"
	"github.com/spf13/cobra"
	"io"
	"os"
	"strings"
	"text/template"
)

const changeTemplate = "CHANGE: {{.Id}} ({{.Desc}})\n" +
	"\t{{printf \"|%-50s|%-40s|%-7s|\" \"PATH\" \"VALUE\" \"REMOVED\"}}\n" +
	"{{range .ChangeValues}}" +
	"\t{{wrappath .Path 50 1| printf \"|%-50s|\"}}{{nativeType . | printf \"(%s) %s\" .ValueType | printf \"%-40s|\" }}{{printf \"%-7t|\" .Removed}}\n" +
	"{{end}}\n"

var funcMapChanges = template.FuncMap{
	"wrappath":   wrapPath,
	"nativeType": nativeType,
}

func getGetChangesCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "changes [<changeId>]",
		Short: "Lists records of configuration changes",
		Args:  cobra.MaximumNArgs(1),
		Run:   runChangesCommand,
	}
	return cmd
}

func runChangesCommand(cmd *cobra.Command, args []string) {
	client := diags.NewConfigDiagsClient(getConnection())
	changesReq := &diags.ChangesRequest{ChangeIDs: make([]string, 0)}
	if len(args) == 1 {
		changesReq.ChangeIDs = append(changesReq.ChangeIDs, args[0])
	}

	tmplChanges, _ := template.New("change").Funcs(funcMapChanges).Parse(changeTemplate)

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
			_ = tmplChanges.Execute(os.Stdout, in)
		}
	}()
	err = stream.CloseSend()
	if err != nil {
		ExitWithErrorMessage("Failed to close: %v", err)
	}
	<-waitc
	ExitWithSuccess()
}

func wrapPath(path string, lineLen int, tabs int) string {
	over := len(path) % lineLen
	linesC := len(path) / lineLen
	var wrapped []string
	if over > 0 {
		wrapped = make([]string, linesC+1)
	} else {
		wrapped = make([]string, linesC)
	}
	var i int
	for i = 0; i < linesC; i++ {
		wrapped[i] = path[i*lineLen : i*lineLen+lineLen]
	}
	if over > 0 {
		overFmt := fmt.Sprintf("%s-%ds", "%", lineLen)
		wrapped[linesC] = fmt.Sprintf(overFmt, path[linesC*lineLen:])
	}

	tabsArr := make([]string, tabs+1)
	sep := fmt.Sprintf("\n%s  ", strings.Join(tabsArr, "\t"))
	return strings.Join(wrapped, sep)
}

func nativeType(cv admin.ChangeValue) string {
	to := make([]int, len(cv.TypeOpts))
	for i, t := range cv.TypeOpts {
		to[i] = int(t)
	}
	tv, err := change.CreateTypedValue(cv.Value, change.ValueType(cv.ValueType), to)
	if err != nil {
		ExitWithErrorMessage("Failed to convert value to TypedValue %s", err)
	}
	return tv.String()
}
