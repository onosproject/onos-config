// SPDX-FileCopyrightText: 2020-present Open Networking Foundation <info@opennetworking.org>
//
// SPDX-License-Identifier: Apache-2.0

package load

import (
	"github.com/openconfig/gnmi/proto/gnmi"
	"github.com/openconfig/gnmi/proto/gnmi_ext"
)

// SetRequest - this is a simplification of the gnmi.SetRequest that can be used
// when forming configurations - it cuts out all of the unused parts like XXX_
// and so results in a smaller file
type SetRequest struct {
	Prefix  *gnmi.Path
	Delete  []*gnmi.Path
	Replace []*Update
	Update  []*Update
	// Extension messages associated with the SetRequest. See the
	// gNMI extension specification for further definition.
	Extension []*Extension
}

// Extension - a simplified version of gnmi.Extension
type Extension struct {
	ID    int
	Value string
}

// Update - a simplified version of the gnmi.Update
type Update struct {
	Path       *gnmi.Path
	Val        *TypedValue
	Duplicates uint32
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
		Extension: make([]*gnmi_ext.Extension, 0),
	}
	for _, up := range sr.SetRequest.Update {
		gnmiSr.Update = append(gnmiSr.Update, &gnmi.Update{
			Path:       up.Path,
			Duplicates: up.Duplicates,
			Val:        fromStructTypeValueToGnmi(up.Val),
		})
	}
	for _, e := range sr.SetRequest.Extension {
		gnmiRegExt := fromStructExtensionToGnmi(e)
		gnmiSr.Extension = append(gnmiSr.Extension, gnmiRegExt)
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

func fromStructExtensionToGnmi(ext *Extension) *gnmi_ext.Extension {
	regExt := gnmi_ext.Extension_RegisteredExt{
		RegisteredExt: &gnmi_ext.RegisteredExtension{
			Id:  gnmi_ext.ExtensionID(ext.ID),
			Msg: []byte(ext.Value),
		},
	}

	return &gnmi_ext.Extension{
		Ext: &regExt,
	}
}
