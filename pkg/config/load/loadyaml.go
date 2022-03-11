// SPDX-FileCopyrightText: 2020-present Open Networking Foundation <info@opennetworking.org>
//
// SPDX-License-Identifier: Apache-2.0

package load

import (
	"fmt"
	configlib "github.com/onosproject/onos-lib-go/pkg/config"
	"github.com/openconfig/gnmi/proto/gnmi"
)

var configGnmi *ConfigGnmiSimple

// ConfigGnmi - a wrapper around a gnmi SetRequest
type ConfigGnmi struct {
	SetRequest gnmi.SetRequest `yaml:"setrequest"`
}

// ConfigGnmiSimple - a wrapper around a simple native SetRequest
type ConfigGnmiSimple struct {
	SetRequest SetRequest
}

// Clear - reset the config - needed for tests
func Clear() {
	configGnmi = nil
}

// GetConfigGnmi gets the onos-config configuration
func GetConfigGnmi(location string) (ConfigGnmiSimple, error) {
	if configGnmi == nil {
		configGnmi = &ConfigGnmiSimple{}
		if err := configlib.LoadNamedConfig(location, configGnmi); err != nil {
			return ConfigGnmiSimple{}, err
		}
		if err := Checker(configGnmi); err != nil {
			return ConfigGnmiSimple{}, err
		}
	}
	return *configGnmi, nil
}

// Checker - check everything is within bounds
func Checker(config *ConfigGnmiSimple) error {

	updateLen := len(config.SetRequest.Update)
	replaceLen := len(config.SetRequest.Replace)
	deleteLen := len(config.SetRequest.Delete)

	if updateLen+replaceLen+deleteLen == 0 {
		return fmt.Errorf("no update, replace or delete found")
	}

	for _, u := range config.SetRequest.Update {
		if err := checkOnlyOneVal(u); err != nil {
			return err
		}
	}

	return nil
}

func checkOnlyOneVal(u *Update) error {
	count := 0
	if u.Val == nil {
		return fmt.Errorf("no value found for %s", u.Path)
	}
	if u.Val.StringValue != nil {
		count++
	}
	if u.Val.IntValue != nil {
		count++
	}
	if u.Val.UIntValue != nil {
		count++
	}
	if u.Val.BoolValue != nil {
		count++
	}
	if u.Val.BytesValue != nil {
		count++
	}
	if u.Val.FloatValue != nil {
		count++
	}
	if u.Val.DecimalValue != nil {
		count++
	}
	if u.Val.LeaflistValue != nil {
		count++
	}
	if u.Val.AnyValue != nil {
		count++
	}
	if u.Val.JSONValue != nil {
		count++
	}
	if u.Val.JSONIetfValue != nil {
		count++
	}
	if u.Val.ASCIIValue != nil {
		count++
	}
	if u.Val.ProtoBytes != nil {
		count++
	}
	if count > 1 {
		return fmt.Errorf("more than 1 value type set on %v", u.Path)
	}
	return nil
}
