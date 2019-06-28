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

package console

import (
	"fmt"
	"github.com/fatih/color"
	"io"
	"os"
)

var (
	success = color.GreenString("✓")
	failure = color.RedString("✗")
)

// ErrorStatus tracks the errors that occurred for a long-running operation
type ErrorStatus interface {
	// Failed returns a boolean indicating whether any failures have occurred
	Failed() bool

	// Errors returns a list of errors that occurred
	Errors() []error
}

// StatusWriter provides real-time status output during onit setup operations
type StatusWriter struct {
	ErrorStatus
	spinner *Spinner
	status  string
	writer  io.Writer
	errors  []error
}

// NewStatusWriter creates a new default StatusWriter
func NewStatusWriter() *StatusWriter {
	writer := os.Stdout
	spinner := newSpinner(writer)
	s := &StatusWriter{
		spinner: spinner,
		writer:  writer,
		errors:  []error{},
	}
	return s
}

// Start starts a new status and begins a loading spinner
func (s *StatusWriter) Start(status string) {
	s.Succeed()
	// set new status
	s.status = status
	s.spinner.SetMessage(fmt.Sprintf(" %s ", s.status))
	s.spinner.Spin()
}

// Succeed completes the current status successfully
func (s *StatusWriter) Succeed() *StatusWriter {
	if s.status == "" {
		return s
	}

	s.spinner.Stop()
	fmt.Fprint(s.writer, "\r")
	fmt.Fprintf(s.writer, " %s %s\n", success, s.status)

	s.status = ""
	return s
}

// Fail fails the current status
func (s *StatusWriter) Fail(err error) *StatusWriter {
	if s.status == "" {
		return s
	}

	s.spinner.Stop()
	fmt.Fprint(s.writer, "\r")
	fmt.Fprintf(s.writer, " %s %-40s %s\n", failure, s.status, err)

	s.status = ""
	s.errors = append(s.errors, err)
	return s
}

// Failed returns a boolean indicating whether errors occurred
func (s *StatusWriter) Failed() bool {
	return len(s.errors) > 0
}

// Errors returns a list of errors that occurred
func (s *StatusWriter) Errors() []error {
	return s.errors
}
