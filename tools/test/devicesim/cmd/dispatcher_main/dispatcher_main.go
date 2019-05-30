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

package main

import (
	"fmt"
	"strconv"
	"time"

	log "github.com/golang/glog"
	"github.com/onosproject/onos-config/tools/test/devicesim/pkg/dispatcher"
	"github.com/onosproject/onos-config/tools/test/devicesim/pkg/events"
)

func eventProducer(dispatcher *dispatcher.Dispatcher) {
	counter := 0
	for {
		subject := "test" + strconv.Itoa(counter)
		event := &events.ConfigEvent{
			Subject: subject,
			Time:    time.Now(),
		}
		dispatcher.Dispatch(event)
		counter = counter + 1
		time.Sleep(1 * time.Second)
	}
}

func main() {

	dispatcher := dispatcher.NewDispatcher()
	ok := dispatcher.RegisterEvent((*events.ConfigEvent)(nil))

	if !ok {
		log.Error("Cannot register an event")
	}

	ch := make(chan events.ConfigEvent, 2)
	ok = dispatcher.RegisterListener(ch)

	if !ok {
		log.Error("Cannot register the listener")
	}

	go eventProducer(dispatcher)

	for result := range ch {
		fmt.Println(result.Time, result.Subject)
	}

}
