// Copyright 2020-present Open Networking Foundation.
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

package load

import (
	"github.com/openconfig/gnmi/proto/gnmi"
	"github.com/openconfig/gnmi/proto/gnmi_ext"
)

// SetRequest - this is a simplification of the gnmi.SetRequest that can be used
// when forming configurations - it cuts out all of the unused parts like XXX_
// and so results in a smaller file
type SetRequest struct {
	Prefix  *gnmi.Path   `protobuf:"bytes,1,opt,name=prefix,proto3" json:"prefix,omitempty"`
	Delete  []*gnmi.Path `protobuf:"bytes,2,rep,name=delete,proto3" json:"delete,omitempty"`
	Replace []*Update    `protobuf:"bytes,3,rep,name=replace,proto3" json:"replace,omitempty"`
	Update  []*Update    `protobuf:"bytes,4,rep,name=update,proto3" json:"update,omitempty"`
	// Extension messages associated with the SetRequest. See the
	// gNMI extension specification for further definition.
	Extension []*gnmi_ext.Extension `protobuf:"bytes,5,rep,name=extension,proto3" json:"extension,omitempty"`
}

// Update - a simplified version of the gnmi.Update
type Update struct {
	Path       *gnmi.Path  `protobuf:"bytes,1,opt,name=path,proto3" json:"path,omitempty"`
	Val        *TypedValue `protobuf:"bytes,3,opt,name=val,proto3" json:"val,omitempty"`
	Duplicates uint32      `protobuf:"varint,4,opt,name=duplicates,proto3" json:"duplicates,omitempty"`
}

// TypedValue - a simplified version of the gnmi.TypedValue
type TypedValue struct {
	StringValue   *gnmi.TypedValue_StringVal
	IntValue      *gnmi.TypedValue_IntVal
	UIntValue     *gnmi.TypedValue_UintVal
	BoolValue     *gnmi.TypedValue_BoolVal
	BytesValue    *gnmi.TypedValue_BytesVal
	FloatValue    *gnmi.TypedValue_FloatVal
	DecimalValue  *gnmi.TypedValue_DecimalVal
	LeaflistValue *gnmi.TypedValue_LeaflistVal
	AnyValue      *gnmi.TypedValue_AnyVal
	JSONValue     *gnmi.TypedValue_JsonVal
	JSONIetfValue *gnmi.TypedValue_JsonIetfVal
	ASCIIValue    *gnmi.TypedValue_AsciiVal
	ProtoBytes    *gnmi.TypedValue_ProtoBytes
}

// ToGnmiSetRequest -- convert an internal SetRequest to Gnmi format
func ToGnmiSetRequest(sr *ConfigGnmiSimple) *gnmi.SetRequest {
	gnmiSr := gnmi.SetRequest{
		Prefix: &gnmi.Path{
			Elem:   sr.SetRequest.Prefix.Elem,
			Target: sr.SetRequest.Prefix.Target,
		},
		Update:    make([]*gnmi.Update, 0),
		Replace:   make([]*gnmi.Update, 0),
		Delete:    sr.SetRequest.Delete,
		Extension: sr.SetRequest.Extension,
	}
	for _, up := range sr.SetRequest.Update {
		gnmiSr.Update = append(gnmiSr.Update, &gnmi.Update{
			Path:       up.Path,
			Duplicates: up.Duplicates,
			Val:        fromStructTypeValueToGnmi(up.Val),
		})
	}

	return &gnmiSr
}

func fromStructTypeValueToGnmi(value *TypedValue) *gnmi.TypedValue {
	var gnmiVal gnmi.TypedValue
	if value.StringValue != nil {
		gnmiVal = gnmi.TypedValue{
			Value: &gnmi.TypedValue_StringVal{StringVal: value.StringValue.StringVal},
		}
	} else if value.IntValue != nil {
		gnmiVal = gnmi.TypedValue{
			Value: &gnmi.TypedValue_IntVal{IntVal: value.IntValue.IntVal},
		}
	} else if value.UIntValue != nil {
		gnmiVal = gnmi.TypedValue{
			Value: &gnmi.TypedValue_UintVal{UintVal: value.UIntValue.UintVal},
		}
	} else if value.BoolValue != nil {
		gnmiVal = gnmi.TypedValue{
			Value: &gnmi.TypedValue_BoolVal{BoolVal: value.BoolValue.BoolVal},
		}
	} else if value.BytesValue != nil {
		gnmiVal = gnmi.TypedValue{
			Value: &gnmi.TypedValue_BytesVal{BytesVal: value.BytesValue.BytesVal},
		}
	} else if value.FloatValue != nil {
		gnmiVal = gnmi.TypedValue{
			Value: &gnmi.TypedValue_FloatVal{FloatVal: value.FloatValue.FloatVal},
		}
	} else if value.DecimalValue != nil {
		gnmiVal = gnmi.TypedValue{
			Value: &gnmi.TypedValue_DecimalVal{DecimalVal: value.DecimalValue.DecimalVal},
		}
	} else if value.LeaflistValue != nil {
		gnmiVal = gnmi.TypedValue{
			Value: &gnmi.TypedValue_LeaflistVal{LeaflistVal: value.LeaflistValue.LeaflistVal},
		}
	} else if value.AnyValue != nil {
		gnmiVal = gnmi.TypedValue{
			Value: &gnmi.TypedValue_AnyVal{AnyVal: value.AnyValue.AnyVal},
		}
	} else if value.JSONValue != nil {
		gnmiVal = gnmi.TypedValue{
			Value: &gnmi.TypedValue_JsonVal{JsonVal: value.JSONValue.JsonVal},
		}
	} else if value.JSONIetfValue != nil {
		gnmiVal = gnmi.TypedValue{
			Value: &gnmi.TypedValue_JsonIetfVal{JsonIetfVal: value.JSONIetfValue.JsonIetfVal},
		}
	} else if value.ASCIIValue != nil {
		gnmiVal = gnmi.TypedValue{
			Value: &gnmi.TypedValue_AsciiVal{AsciiVal: value.ASCIIValue.AsciiVal},
		}
	} else if value.ProtoBytes != nil {
		gnmiVal = gnmi.TypedValue{
			Value: &gnmi.TypedValue_ProtoBytes{ProtoBytes: value.ProtoBytes.ProtoBytes},
		}
	}

	return &gnmiVal
}
