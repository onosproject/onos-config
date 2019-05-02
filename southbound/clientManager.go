// Copyright 2019-present Open Networking Foundation
//
// Licensed under the Apache License, Configuration 2.0 (the "License");
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

package southbound

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"errors"
	"fmt"
	"github.com/opennetworkinglab/onos-config/southbound/topocache"
	"io/ioutil"
	"log"
	"time"

	"github.com/golang/protobuf/proto"
	"github.com/openconfig/gnmi/client"
	gclient "github.com/openconfig/gnmi/client/gnmi"
	gpb "github.com/openconfig/gnmi/proto/gnmi"
)

//Can be extended, for now is ip:port
type Key struct {
	Key string
}

type Target struct {
	Clt  client.Impl
	Ctxt context.Context
}

var targets = make(map[Key]Target)

func createDestination(device topocache.Device) (*client.Destination, Key) {
	d := &client.Destination{TLS: &tls.Config{}}
	d.Addrs = []string{device.Addr}
	d.Target = device.Target
	d.Timeout = time.Duration(device.Timeout * time.Second)
	if device.CaPath == "" {
		log.Println("Loading default CA onfca")
		d.TLS.RootCAs = getCertPoolDefault()
	} else {
		d.TLS.RootCAs = getCertPool(device.CaPath)
	}

	if device.CertPath == "" && device.KeyPath == "" {
		// Load default Certificates
		log.Println("Loading default certificates")
		clientCerts, err := tls.X509KeyPair([]byte(defaultClientCrt), []byte(defaultClientKey))
		if err != nil {
			log.Println("Error loading default certs")
		}

		d.TLS.Certificates = []tls.Certificate{clientCerts}
	} else if device.CertPath != "" && device.KeyPath != "" {
		// Load certs given for device
		d.TLS.Certificates = []tls.Certificate{setCertificate(device.CertPath, device.KeyPath)}

	} else if device.Usr != "" && device.Pwd != "" {
		//TODO implement
		// cred := &client.Credentials{}
		// cred.Username = "test"
		// cred.Password = "testpwd"
		//d.Credentials = cred
		//log.Println(*cred)
	} else {
		d.TLS = &tls.Config{InsecureSkipVerify: true}
	}
	return d, Key{Key: device.Addr}
}

func GetTarget(key Key) (Target, error) {
	_, ok := targets[key]
	if ok {
		return targets[key], nil
	}
	return Target{}, fmt.Errorf("Client for %v does not exist, create first", key)

}

func ConnectTarget(device topocache.Device) (Target, Key, error) {
	dest, key := createDestination(device)
	ctx := context.Background()
	c, err := gclient.New(ctx, *dest)
	if err != nil {
		return Target{}, Key{}, fmt.Errorf("could not create a gNMI client: %v", err)
	}
	target := Target{
		Clt:  c,
		Ctxt: ctx,
	}
	targets[key] = target
	return target, key, err
}

func setCertificate(pathCert string, pathKey string) tls.Certificate {
	certificate, err := tls.LoadX509KeyPair(pathCert, pathKey)
	if err != nil {
		log.Println("could not load client key pair: %v", err)
	}
	return certificate
}

func getCertPool(CaPath string) *x509.CertPool {
	certPool := x509.NewCertPool()
	ca, err := ioutil.ReadFile(CaPath)
	if err != nil {
		log.Println("could not read %q: %s", CaPath, err)
	}
	if ok := certPool.AppendCertsFromPEM(ca); !ok {
		log.Println("failed to append CA certificates")
	}
	return certPool
}

func getCertPoolDefault() *x509.CertPool {
	certPool := x509.NewCertPool()
	if ok := certPool.AppendCertsFromPEM([]byte(onfCaCrt)); !ok {
		log.Println("failed to append CA certificates")
	}
	return certPool
}

func CapabilitiesWithString(target Target, request string) (*gpb.CapabilityResponse, error) {
	r := &gpb.CapabilityRequest{}
	reqProto := &request
	if err := proto.UnmarshalText(*reqProto, r); err != nil {
		return nil, fmt.Errorf("unable to parse gnmi.CapabilityRequest from %q : %v", *reqProto, err)
	}
	return Capabilities(target, r)
}

func Capabilities(target Target, request *gpb.CapabilityRequest) (*gpb.CapabilityResponse, error) {
	response, err := target.Clt.(*gclient.Client).Capabilities(target.Ctxt, request)
	if err != nil {
		return nil, fmt.Errorf("target returned RPC error for Capabilities(%q): %v", request.String(), err)
	}
	return response, nil
}

func GetWithString(target Target, request string) (*gpb.GetResponse, error) {
	if request == "" {
		return nil, errors.New("cannot get and empty request")
	}
	r := &gpb.GetRequest{}
	reqProto := &request
	if err := proto.UnmarshalText(*reqProto, r); err != nil {
		return nil, fmt.Errorf("unable to parse gnmi.GetRequest from %q : %v", *reqProto, err)
	}
	return Get(target, r)
}

func Get(target Target, request *gpb.GetRequest) (*gpb.GetResponse, error) {
	response, err := target.Clt.(*gclient.Client).Get(target.Ctxt, request)
	if err != nil {
		return nil, fmt.Errorf("target returned RPC error for Get(%q) : %v", request.String(), err)
	}
	return response, nil
}

func SetWithString(target Target, request string) (*gpb.SetResponse, error) {
	//TODO modify with key that gets target from map
	if request == "" {
		return nil, errors.New("cannot get and empty request")
	}
	r := &gpb.SetRequest{}
	reqProto := &request
	if err := proto.UnmarshalText(*reqProto, r); err != nil {
		return nil, fmt.Errorf("unable to parse gnmi.SetRequest from %q : %v", *reqProto, err)
	}
	return Set(target, r)
}

func Set(target Target, request *gpb.SetRequest) (*gpb.SetResponse, error) {
	response, err := target.Clt.(*gclient.Client).Set(target.Ctxt, request)
	if err != nil {
		return nil, fmt.Errorf("target returned RPC error for Set(%q) : %v", request.String(), err)
	}
	return response, nil
}
