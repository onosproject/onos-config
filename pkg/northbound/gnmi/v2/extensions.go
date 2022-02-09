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
	"github.com/onosproject/onos-lib-go/pkg/errors"
	"github.com/onosproject/onos-lib-go/pkg/logging"
	"github.com/openconfig/gnmi/proto/gnmi"
	"github.com/openconfig/gnmi/proto/gnmi_ext"
)

var log = logging.GetLogger("northbound", "gnmi")

func processExtension(ext *gnmi_ext.Extension) (configapi.TransactionStrategy, error) {
	var transactionStrategy configapi.TransactionStrategy
	if regExt, ok := ext.Ext.(*gnmi_ext.Extension_RegisteredExt); ok &&
		regExt.RegisteredExt.Id == configapi.TransactionStrategyExtensionID {
		bytes := regExt.RegisteredExt.Msg

		if err := proto.Unmarshal(bytes, &transactionStrategy); err != nil {
			return transactionStrategy, errors.NewInvalid(err.Error())
		}
	}
	return transactionStrategy, nil
}

func getExtensions(request interface{}) (configapi.TransactionStrategy, error) {
	switch req := request.(type) {
	case *gnmi.SetRequest:
		for _, ext := range req.GetExtension() {
			return processExtension(ext)
		}
	case *gnmi.GetRequest:
		for _, ext := range req.GetExtension() {
			return processExtension(ext)
		}
	}
	return configapi.TransactionStrategy{}, nil
}
