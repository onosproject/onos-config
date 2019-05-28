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
	"io"

	log "github.com/golang/glog"
	pb "github.com/openconfig/gnmi/proto/gnmi"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// Subscribe overrides the Subscribe function to implement it.
func (s *server) Subscribe(stream pb.GNMI_SubscribeServer) error {
	c := streamClient{stream: stream}
	var err error

	updateChan := make(chan *pb.Update)

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
		subscribe := c.sr.GetSubscribe()
		mode := c.sr.GetSubscribe().GetMode()
		done := make(chan struct{})

		switch mode {
		case pb.SubscriptionList_ONCE:
			log.Info("A Subscription ONCE request received")
			go s.collector(updateChan, subscribe)
		case pb.SubscriptionList_POLL:
			go s.collector(updateChan, subscribe)
			fmt.Println("A Subscription POLL request received")
			// TODO add a goroutine to process  subscription POLL.
		case pb.SubscriptionList_STREAM:
			fmt.Println("A Subscription STREAM request received")
			// TODO add a goroutine to process subscription STREAM.

		default:
			return status.Errorf(codes.InvalidArgument, "Subscription mode %v not recognized", mode)
		}
		go func() {
			for update := range updateChan {
				response, _ := buildSubResponse(update)
				sendResponse(response, stream)
				done <- struct{}{}
			}
		}()

		if mode == pb.SubscriptionList_ONCE {
			<-done
			return nil
		}
	}

}
