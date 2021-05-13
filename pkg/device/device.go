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

// Package device contains a shim definition of Device to insulate the onos-config subsystem from
// the deprecation of onos-topo/api/device.
package device

import (
	"fmt"
	"github.com/gogo/protobuf/proto"
	"github.com/onosproject/onos-api/go/onos/topo"
	"time"
)

// ID represents device globally unique ID
type ID topo.ID

// Type represents device type
type Type string

// Role represents device role in the network
type Role string

// Device structure provide a shim for topo.Object
type Device struct {
	// globally unique device identifier; maps to Object.ID
	ID ID

	// host:port of the device
	Address string
	// device target
	Target string
	// device software version
	Version string

	// timeout indicates the device request timeout
	Timeout *time.Duration
	// credentials for connecting to the device
	Credentials Credentials
	// TLS configuration for connecting to the device
	TLS TLSConfig

	// type of the device
	Type Type
	// role for the device
	Role Role

	Protocols []*topo.ProtocolState
	// user-friendly tag
	Displayname string

	// Mastership state
	MastershipTerm uint64
	MasterKey      string

	// revision of the underlying Object
	Revision topo.Revision

	// Backing entity
	Object *topo.Object
}

// Credentials is the device credentials
type Credentials struct {
	// user with which to connect to the device
	User string
	// password for connecting to the device
	Password string
}

// TLSConfig contains information pertinent to establishing a secure connection
type TLSConfig struct {
	// name of the device's CA certificate
	CaCert string
	// name of the device's certificate
	Cert string
	// name of the device's TLS key
	Key string
	// indicates whether to connect to the device over plaintext
	Plain bool
	// indicates whether to connect to the device with insecure communication
	Insecure bool
}

// ListResponse carries a single device event
type ListResponse struct {
	// type of the event
	Type ListResponseType
	// device is the device on which the event occurred
	Device *Device
}

// ListResponseType is a device event type
type ListResponseType int32

const (
	// ListResponseNONE obviously
	ListResponseNONE ListResponseType = 0
	// ListResponseADDED obviously
	ListResponseADDED ListResponseType = 1
	// ListResponseUPDATED obviously
	ListResponseUPDATED ListResponseType = 2
	// ListResponseREMOVED obviously
	ListResponseREMOVED ListResponseType = 3
)

// ListresponsetypeName - map of enumerations
var ListresponsetypeName = map[int32]string{
	0: "NONE",
	1: "ADDED",
	2: "UPDATED",
	3: "REMOVED",
}

// String - convert to string
func (x ListResponseType) String() string {
	return proto.EnumName(ListresponsetypeName, int32(x))
}

// ToObject converts local device to a topology object entity.
func ToObject(device *Device) *topo.Object {
	o := device.Object
	if o == nil {
		o = &topo.Object{
			ID:       topo.ID(device.ID),
			Revision: device.Revision,
			Type:     topo.Object_ENTITY,
			Obj: &topo.Object_Entity{
				Entity: &topo.Entity{
					KindID:    topo.ID(device.Type),
					Protocols: device.Protocols,
				},
			},
		}
	}

	var timeout uint64
	if device.Timeout != nil {
		timeout = uint64(device.Timeout.Milliseconds())
	}

	// FIXME: preserve other aspect values; presently this transformation is lossy
	_ = o.SetAspect(&topo.Asset{
		Name: device.Displayname,
		Role: string(device.Role),
	})

	_ = o.SetAspect(&topo.Configurable{
		Type:    string(device.Type),
		Address: device.Address,
		Target:  device.Target,
		Version: device.Version,
		Timeout: timeout,
	})

	_ = o.SetAspect(&topo.TLSOptions{
		Insecure: device.TLS.Insecure,
		Plain:    device.TLS.Plain,
		Key:      device.TLS.Key,
		CaCert:   device.TLS.CaCert,
		Cert:     device.TLS.Cert,
	})

	_ = o.SetAspect(&topo.MastershipState{
		Term:   device.MastershipTerm,
		NodeId: device.MasterKey,
	})
	return o
}

// ToDevice converts topology object entity to a local device object
func ToDevice(object *topo.Object) (*Device, error) {
	if object.Type != topo.Object_ENTITY {
		return nil, fmt.Errorf("object is not a topo entity %v+", object)
	}

	typeKindID := Type(object.GetEntity().KindID)
	if len(typeKindID) == 0 {
		return nil, fmt.Errorf("topo entity %s must have a 'kindid' to work with onos-config", object.ID)
	}

	asset := object.GetAspect(&topo.Asset{}).(*topo.Asset)
	if asset == nil {
		return nil, fmt.Errorf("topo entity %s must have 'asset' attribute to work with onos-config", object.ID)
	}
	configurable := object.GetAspect(&topo.Configurable{}).(*topo.Configurable)
	if configurable == nil {
		return nil, fmt.Errorf("topo entity %s must have 'configurable' attribute to work with onos-config", object.ID)
	}

	mastership := object.GetAspect(&topo.MastershipState{}).(*topo.MastershipState)
	if mastership == nil {
		return nil, fmt.Errorf("topo entity %s must have 'mastership' attribute to work with onos-config", object.ID)
	}

	tlsInfo := object.GetAspect(&topo.TLSOptions{}).(*topo.TLSOptions)
	if tlsInfo == nil {
		return nil, fmt.Errorf("topo entity %s must have 'tls-info' attribute to work with onos-config", object.ID)
	}

	timeout := time.Millisecond * time.Duration(configurable.Timeout)

	d := &Device{
		ID:          ID(object.ID),
		Revision:    object.Revision,
		Protocols:   object.GetEntity().Protocols,
		Type:        typeKindID,
		Role:        Role(asset.Role),
		Displayname: asset.Name,
		Address:     configurable.Address,
		Target:      configurable.Target,
		Version:     configurable.Version,
		Timeout:     &timeout,
		TLS: TLSConfig{
			Plain:    tlsInfo.Plain,
			Insecure: tlsInfo.Insecure,
			Cert:     tlsInfo.Cert,
			CaCert:   tlsInfo.CaCert,
			Key:      tlsInfo.Key,
		},
		MastershipTerm: mastership.Term,
		MasterKey:      mastership.NodeId,
		Object:         object,
	}
	return d, nil
}
