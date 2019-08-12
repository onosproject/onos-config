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
	"os"
)

const (
	// ExitSuccess means nominal status
	ExitSuccess = iota

	// ExitError means general error
	ExitError

	// ExitBadConnection means failed connection to remote service
	ExitBadConnection

	// ExitBadArgs means invalid argument values were given
	ExitBadArgs = 128
)

// Output prints the specified format message with arguments to stdout.
func Output(msg string, args ...interface{}) {
	fmt.Fprintf(os.Stdout, msg, args...)
}

// ExitWithOutput prints the specified entity and exits program with success.
func ExitWithOutput(msg string, output ...interface{}) {
	fmt.Fprintf(os.Stdout, msg, output...)
	os.Exit(ExitSuccess)
}

// ExitWithSuccess exits program with success without any output.
func ExitWithSuccess() {
	os.Exit(ExitSuccess)
}

// ExitWithError prints the specified error and exits program with the given error code.
func ExitWithError(code int, err error) {
	fmt.Fprintln(os.Stderr, "Error:", err)
	os.Exit(code)
}

// ExitWithErrorMessage prints the specified message and exits program with the given error code.
func ExitWithErrorMessage(msg string, args ...interface{}) {
	fmt.Fprintf(os.Stderr, msg, args...)
	os.Exit(ExitError)
}
