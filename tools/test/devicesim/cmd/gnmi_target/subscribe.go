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
	"io"

	log "github.com/golang/glog"
	"github.com/openconfig/gnmi/proto/gnmi"
	pb "github.com/openconfig/gnmi/proto/gnmi"
)

// Subscribe overrides the Subscribe function to implement it.
func (s *server) Subscribe(stream pb.GNMI_SubscribeServer) error {
	c := streamClient{stream: stream}
	var err error
	updateChan := make(chan *pb.Update)
	var subscribe *gnmi.SubscriptionList
	var mode gnmi.SubscriptionList_Mode

	for {
		c.sr, err = stream.Recv()
		switch {
		case err == io.EOF:
			log.Error("No more input is available, subscription terminated")
			return nil
		case err != nil:
			log.Error("Error in subscription", err)
			return err

		}

		if c.sr.GetPoll() != nil {
			mode = gnmi.SubscriptionList_POLL
		} else {
			subscribe = c.sr.GetSubscribe()
			mode = subscribe.Mode
		}
		done := make(chan struct{})

		//If the subscription mode is ONCE or POLL we immediately start a routine to collect the data
		if mode != pb.SubscriptionList_STREAM {
			go s.collector(updateChan, subscribe)
		}
		go s.listenForUpdates(updateChan, stream, mode, done)

		if mode == pb.SubscriptionList_ONCE {
			<-done
			return nil
		} else if mode == pb.SubscriptionList_STREAM {
			subs := subscribe.Subscription
			for _, sub := range subs {
				s.PathToChannels[sub.Path] = updateChan
			}

			return nil
		}

	}

}
