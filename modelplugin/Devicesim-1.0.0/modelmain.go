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

// +build modelplugin

// A plugin for the YGOT model of Devicesim-1.0.0.
package main

import (
	"fmt"
	"github.com/onosproject/onos-config/modelplugin/Devicesim-1.0.0/devicesim_1_0_0"
	"github.com/openconfig/gnmi/proto/gnmi"
	"github.com/openconfig/goyang/pkg/yang"
	"github.com/openconfig/ygot/ygot"
)

type modelplugin string

const modeltype = "Devicesim"
const modelversion = "1.0.0"
const modulename = "devicesim.so.1.0.0"

var modelData = []*gnmi.ModelData{
	{Name: "openconfig-interfaces", Organization: "OpenConfig working group", Version: "2017-07-14"}, {Name: "openconfig-openflow", Organization: "OpenConfig working group", Version: "2017-06-01"}, {Name: "openconfig-platform", Organization: "OpenConfig working group", Version: "2016-12-22"}, {Name: "openconfig-system", Organization: "OpenConfig working group", Version: "2017-07-06"},
}

func (m modelplugin) ModelData() (string, string, []*gnmi.ModelData, string) {
	return modeltype, modelversion, modelData, modulename
}

// UnmarshallConfigValues allows Device to implement the Unmarshaller interface
func (m modelplugin) UnmarshalConfigValues(jsonTree []byte) (*ygot.ValidatedGoStruct, error) {
	device := &devicesim_1_0_0.Device{}
	vgs := ygot.ValidatedGoStruct(device)

	if err := devicesim_1_0_0.Unmarshal([]byte(jsonTree), device); err != nil {
		return nil, err
	}

	return &vgs, nil
}

func (m modelplugin) Validate(ygotModel *ygot.ValidatedGoStruct, opts ...ygot.ValidationOption) error {
	deviceDeref := *ygotModel
	device, ok := deviceDeref.(*devicesim_1_0_0.Device)
	if !ok {
		return fmt.Errorf("unable to convert model in to devicesim_1_0_0")
	}
	return device.Validate()
}

func (m modelplugin) Schema() (map[string]*yang.Entry, error) {
	return devicesim_1_0_0.UnzipSchema()
}

// ModelPlugin is the exported symbol that gives an entry point to this shared module
var ModelPlugin modelplugin
