// Copyright 2022-present Open Networking Foundation.
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

package gnmi

import (
	"github.com/onosproject/onos-lib-go/pkg/errors"

	configapi "github.com/onosproject/onos-api/go/onos/config/v2"
	"github.com/onosproject/onos-lib-go/pkg/logging"
	"github.com/openconfig/gnmi/proto/gnmi"
)

var log = logging.GetLogger("northbound", "gnmi")

const (
	// ExtensionTransactionID transaction ID extension
	ExtensionTransactionID = 100

	// ExtensionTransactionIndex transaction index extension
	ExtensionTransactionIndex = 102
)

// Extensions list of gNMI extensions
type Extensions struct {
	transactionID configapi.TransactionID
}

func extractExtensions(req interface{}) (Extensions, error) {
	var transactionID configapi.TransactionID
	switch v := req.(type) {
	case *gnmi.SetRequest:
		for _, ext := range v.GetExtension() {
			extID := ext.GetRegisteredExt().GetId()
			extMsg := ext.GetRegisteredExt().GetMsg()
			if extID == ExtensionTransactionID {
				transactionID = configapi.TransactionID(extMsg)
			} else {
				return Extensions{}, errors.NewInvalid("unexpected extension %d = '%s' in Set()", ext.GetRegisteredExt().GetId(), ext.GetRegisteredExt().GetMsg())
			}
		}
	case *gnmi.GetRequest:
	}

	extensions := Extensions{
		transactionID: transactionID,
	}

	return extensions, nil
}
