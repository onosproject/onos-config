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
	topoapi "github.com/onosproject/onos-api/go/onos/topo"
	"time"
)

// Type represents device type
type Type string

// Role represents device role in the network
type Role string

// Device structure provide a shim for topo.Object
type Device struct {
	// globally unique device identifier; maps to Object.ID
	ID topoapi.ID

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

	Protocols []*topoapi.ProtocolState
	// user-friendly tag
	Displayname string

	// arbitrary mapping of attribute keys/values
	Attributes map[string]string

	// revision of the underlying Object
	Revision topoapi.Revision
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
