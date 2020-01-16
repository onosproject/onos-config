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

package stream

import "context"

// Context is a stream context
type Context interface {
	// Close closes the stream. The stream channel will be closed and resources used by the
	// stream will be released.
	Close()
}

// CloseFunc is a Context Close function
type CloseFunc func()

// NewContext creates a new context with a close function
func NewContext(close CloseFunc) Context {
	return &closerContext{
		close: close,
	}
}

// NewCancelContext creates a new stream context from a context CancelFunc
func NewCancelContext(cancel context.CancelFunc) Context {
	return NewContext(func() {
		cancel()
	})
}

type closerContext struct {
	close CloseFunc
}

func (c *closerContext) Close() {
	c.close()
}

// EventType is a stream event type
type EventType string

const (
	// None indicates an object was not changed
	None EventType = ""
	// Created indicates an object was created
	Created EventType = "Created"
	// Updated indicates an object was updated
	Updated EventType = "Updated"
	// Deleted indicates an object was deleted
	Deleted EventType = "Deleted"
)

// Event is a stream event
type Event struct {
	// Type is the stream event type
	Type EventType

	// Object is the event object
	Object interface{}
}
