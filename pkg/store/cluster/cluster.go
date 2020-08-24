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

package cluster

import "os"

const nodeIDEnv = "POD_NAME"

// NodeID is a node identifier provided by the environment
type NodeID string

// GetNodeID gets the local node identifier
func GetNodeID() NodeID {
	nodeID := os.Getenv(nodeIDEnv)
	if nodeID == "" {
		return NodeID("onos-config")
	}
	return NodeID(nodeID)
}
