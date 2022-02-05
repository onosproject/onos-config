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
	"github.com/gogo/protobuf/proto"
	configapi "github.com/onosproject/onos-api/go/onos/config/v2"
	"github.com/onosproject/onos-lib-go/pkg/logging"
	"github.com/openconfig/gnmi/proto/gnmi"
	"github.com/openconfig/gnmi/proto/gnmi_ext"
)

var log = logging.GetLogger("northbound", "gnmi")

func getSetExtensions(request *gnmi.SetRequest) (configapi.TransactionMode, error) {
	var transactionMode configapi.TransactionMode
	for _, ext := range request.GetExtension() {
		if regExt, ok := ext.Ext.(*gnmi_ext.Extension_RegisteredExt); ok &&
			regExt.RegisteredExt.Id == configapi.TransactionModeExtensionID {
			bytes := regExt.RegisteredExt.Msg
			if err := proto.Unmarshal(bytes, &transactionMode); err != nil {
				return transactionMode, err
			}
		}
	}
	return transactionMode, nil
}
