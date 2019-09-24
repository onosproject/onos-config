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
	"fmt"
	"io"
	"os"
)

var outputWriter io.Writer

// GetOutput returns the current output writer
func GetOutput() io.Writer {
	return outputWriter
}

// CaptureOutput allows a test harness to redirect output to an alternate source for testing
func CaptureOutput(capture io.Writer) {
	outputWriter = capture
}

func init() {
	CaptureOutput(os.Stdout)
}

// Output prints the specified format message with arguments to stdout.
func Output(msg string, args ...interface{}) {
	_, _ = fmt.Fprintf(outputWriter, msg, args...)
}
