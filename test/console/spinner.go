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
	"io"
	"sync"
	"time"
)

// custom CLI loading spinner for kind
var spinnerFrames = []string{
	"⠈⠁",
	"⠈⠑",
	"⠈⠱",
	"⠈⡱",
	"⢀⡱",
	"⢄⡱",
	"⢄⡱",
	"⢆⡱",
	"⢎⡱",
	"⢎⡰",
	"⢎⡠",
	"⢎⡀",
	"⢎⠁",
	"⠎⠁",
	"⠊⠁",
}

// Spinner is a simple and efficient CLI loading spinner used by kind
// It is simplistic and assumes that the line length will not change.
// It is best used indirectly via log.Status (see parent package)
type Spinner struct {
	frames  []string
	stop    chan struct{}
	ticker  *time.Ticker
	writer  io.Writer
	mu      *sync.Mutex
	message string
}

// newSpinner initializes and returns a new Spinner that will write to
func newSpinner(w io.Writer) *Spinner {
	return &Spinner{
		frames: spinnerFrames,
		stop:   make(chan struct{}, 1),
		ticker: time.NewTicker(time.Millisecond * 100),
		mu:     &sync.Mutex{},
		writer: w,
	}
}

// SetMessage sets the message to print after the spinner
func (s *Spinner) SetMessage(message string) {
	s.mu.Lock()
	s.message = message
	s.mu.Unlock()
}

// Spin starts the spinner running
func (s *Spinner) Spin() {
	go func() {
		for {
			for _, frame := range s.frames {
				select {
				case <-s.stop:
					return
				case <-s.ticker.C:
					func() {
						s.mu.Lock()
						defer s.mu.Unlock()
						fmt.Fprintf(s.writer, "\r%s%s", frame, s.message)
					}()
				}
			}
		}
	}()
}

// Stop signals the spinner to stop
func (s *Spinner) Stop() {
	s.stop <- struct{}{}
}
