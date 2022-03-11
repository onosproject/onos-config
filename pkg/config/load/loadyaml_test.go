// SPDX-FileCopyrightText: 2020-present Open Networking Foundation <info@opennetworking.org>
//
// SPDX-License-Identifier: Apache-2.0

package load

import (
	"github.com/openconfig/gnmi/proto/gnmi"
	"gopkg.in/yaml.v2"
	"gotest.tools/assert"
	"testing"
)

func Test_LoadConfig1(t *testing.T) {
	Clear()
	config, err := GetConfigGnmi("onos-config-load-sample-gnmi.yaml")
	assert.NilError(t, err, "Unexpected error loading gnmi setrequest yaml")

	assert.Equal(t, "", config.SetRequest.Prefix.Target)
	assert.Equal(t, 1, len(config.SetRequest.Prefix.Elem), "Unexpected number of Prefix Elems")
	assert.Equal(t, "e2node", config.SetRequest.Prefix.Elem[0].Name)

	assert.Equal(t, 6, len(config.SetRequest.Update), "Unexpected number of Updates")
	assert.Equal(t, 2, len(config.SetRequest.Update[0].Path.Elem), "Unexpected number of elems in path of update 0")
	assert.Equal(t, "intervals", config.SetRequest.Update[0].Path.Elem[0].Name)
	assert.Equal(t, "RadioMeasReportPerUe", config.SetRequest.Update[0].Path.Elem[1].Name)

	assert.Equal(t, uint64(20), config.SetRequest.Update[0].Val.UIntValue.UintVal)

	assert.Equal(t, 2, len(config.SetRequest.Extension))
	assert.Equal(t, 101, config.SetRequest.Extension[0].ID)
	assert.Equal(t, "1.0.0", config.SetRequest.Extension[0].Value)
	assert.Equal(t, 102, config.SetRequest.Extension[1].ID)
	assert.Equal(t, "E2Node", config.SetRequest.Extension[1].Value)
}

func Test_ConvertConfig(t *testing.T) {
	Clear()
	config, err := GetConfigGnmi("onos-config-load-sample-gnmi.yaml")
	assert.NilError(t, err, "Unexpected error loading gnmi setrequest yaml")

	gnmiSr := ToGnmiSetRequest(&config)
	assert.Equal(t, "", gnmiSr.Prefix.Target)
	assert.Equal(t, 1, len(gnmiSr.Prefix.Elem), "Unexpected number of Prefix Elems")
	assert.Equal(t, "e2node", gnmiSr.Prefix.Elem[0].Name)

	assert.Equal(t, 6, len(gnmiSr.Update), "Unexpected number of Updates")
	assert.Equal(t, 2, len(gnmiSr.Update[0].Path.Elem), "Unexpected number of elems in path of update 0")
	assert.Equal(t, "intervals", gnmiSr.Update[0].Path.Elem[0].Name)
	assert.Equal(t, "RadioMeasReportPerUe", gnmiSr.Update[0].Path.Elem[1].Name)

	assert.Equal(t, uint64(20), gnmiSr.Update[0].Val.Value.(*gnmi.TypedValue_UintVal).UintVal)
}

func Test_LeafList(t *testing.T) {
	gnmisr := &gnmi.SetRequest{
		Update: make([]*gnmi.Update, 0),
	}
	upd0 := gnmi.Update{
		Val: &gnmi.TypedValue{
			Value: &gnmi.TypedValue_StringVal{StringVal: "abc"},
		},
	}
	gnmisr.Update = append(gnmisr.Update, &upd0)

	upd1 := gnmi.Update{
		Val: &gnmi.TypedValue{
			Value: &gnmi.TypedValue_LeaflistVal{
				LeaflistVal: &gnmi.ScalarArray{
					Element: []*gnmi.TypedValue{
						{
							Value: &gnmi.TypedValue_StringVal{StringVal: "def"},
						},
						{
							Value: &gnmi.TypedValue_StringVal{StringVal: "ghi"},
						},
					},
				},
			},
		},
	}
	gnmisr.Update = append(gnmisr.Update, &upd1)

	yamlBytes, err := yaml.Marshal(gnmisr)
	assert.NilError(t, err, "Unexpected error marshalling Gnmi SetRequest")
	t.Log(string(yamlBytes))
}

func Test_LoadLeafList(t *testing.T) {
	Clear()
	config, err := GetConfigGnmi("set.role-aether-ops.yaml")
	assert.NilError(t, err, "Unexpected error loading set.role-aether-ops yaml")

	gnmiSr := ToGnmiSetRequest(&config)
	assert.Equal(t, "internal", gnmiSr.Prefix.Target)
	assert.Equal(t, 2, len(gnmiSr.Prefix.Elem), "Unexpected number of Prefix Elems")
	assert.Equal(t, "rbac", gnmiSr.Prefix.Elem[0].Name)
	assert.Equal(t, "role", gnmiSr.Prefix.Elem[1].Name)

	assert.Equal(t, 4, len(gnmiSr.Update), "Unexpected number of Updates")
	assert.Equal(t, 1, len(gnmiSr.Update[0].Path.Elem), "Unexpected number of elems in path of update 0")
	assert.Equal(t, "description", gnmiSr.Update[0].Path.Elem[0].Name)

	assert.Equal(t, 2, len(gnmiSr.Update[3].Path.Elem), "Unexpected number of elems in path of update 3")
	assert.Equal(t, "permission", gnmiSr.Update[3].Path.Elem[0].Name)
	assert.Equal(t, "noun", gnmiSr.Update[3].Path.Elem[1].Name)
	llVal, ok := gnmiSr.Update[3].Val.Value.(*gnmi.TypedValue_LeaflistVal)
	assert.Assert(t, ok, "unexpected error casting to TypedValue_LeaflistVal %v", gnmiSr.Update[3].Val.Value)
	assert.Equal(t, 1, len(llVal.LeaflistVal.Element))
	elem0Val := llVal.LeaflistVal.Element[0]
	_, ok = elem0Val.Value.(*gnmi.TypedValue_StringVal)
	assert.Assert(t, !ok, "Fails here - no way to pass string value in from YAML %v", llVal.LeaflistVal.Element[0].Value)
	//assert.Equal(t, "abc", strVal1.StringVal)
}
