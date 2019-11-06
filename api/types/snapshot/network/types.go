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

package network

import (
	"fmt"
	"github.com/onosproject/onos-config/api/types"
	"strconv"
	"strings"
)

const separator = ":"

// ID is a network snapshot identifier type
type ID types.ID

// GetIndex returns the Index
func (i ID) GetIndex() Index {
	indexStr := string(i)[strings.LastIndex(string(i), separator)+1:]
	index, _ := strconv.Atoi(indexStr)
	return Index(index)
}

// Index is the index of a network snapshot
type Index uint64

// GetSnapshotID returns the network snapshot ID for the index
func (i Index) GetSnapshotID() ID {
	return ID(fmt.Sprintf("snapshot%s%d", separator, i))
}

// Revision is a network snapshot revision number
type Revision types.Revision
