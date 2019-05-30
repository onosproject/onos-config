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

package dispatcher

import (
	"fmt"
	"reflect"
	"sync"

	"github.com/onosproject/onos-config/tools/test/devicesim/pkg/events"
)

//Dispatcher dispatches the events
type Dispatcher struct {
	handlers map[reflect.Type][]reflect.Value
	lock     *sync.RWMutex
}

// NewDispatcher creates an instance of Dispatcher struct
func NewDispatcher() *Dispatcher {
	return &Dispatcher{
		handlers: make(map[reflect.Type][]reflect.Value),
		lock:     &sync.RWMutex{},
	}
}

// RegisterEvent registers custom events making it possible to register listeners for them.
func (d *Dispatcher) RegisterEvent(event events.Event) bool {
	d.lock.Lock()
	defer d.lock.Unlock()
	typ := reflect.TypeOf(event).Elem()
	fmt.Println(typ)
	if _, ok := d.handlers[typ]; ok {
		return false
	}
	var chanArr []reflect.Value
	d.handlers[typ] = chanArr
	return true
}

// RegisterListener registers chanel accepting desired event - a listener.
func (d *Dispatcher) RegisterListener(pipe interface{}) bool {
	d.lock.Lock()
	defer d.lock.Unlock()
	channelValue := reflect.ValueOf(pipe)
	channelType := channelValue.Type()
	if channelType.Kind() != reflect.Chan {
		panic("Trying to register a non-channel listener")
	}
	channelIn := channelType.Elem()
	if arr, ok := d.handlers[channelIn]; ok {
		d.handlers[channelIn] = append(arr, channelValue)
		return true
	}
	return false
}

// Dispatch provides thread safe method to send event to all listeners
// Returns true if succeded and false if event was not registered
func (d *Dispatcher) Dispatch(event events.Event) bool {
	d.lock.RLock()
	defer d.lock.RUnlock()

	eventType := reflect.TypeOf(event).Elem()
	if listeners, ok := d.handlers[eventType]; ok {
		for _, listener := range listeners {
			listener.TrySend(reflect.ValueOf(event.Clone()).Elem())
		}
		return true
	}
	return false
}
