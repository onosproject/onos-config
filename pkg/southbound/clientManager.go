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

// Package southbound implements configuration of network devices via gNMI clients.
package southbound

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"errors"
	"fmt"
	"github.com/onosproject/onos-config/pkg/certs"
	"github.com/onosproject/onos-config/pkg/utils"
	devicepb "github.com/onosproject/onos-topo/pkg/northbound/device"
	"io/ioutil"
	log "k8s.io/klog"
	"strings"

	"github.com/golang/protobuf/proto"
	"github.com/openconfig/gnmi/client"
	gpb "github.com/openconfig/gnmi/proto/gnmi"
)

var targets = make(map[DeviceID]interface{})

func createDestination(device devicepb.Device) (*client.Destination, DeviceID) {
	d := &client.Destination{}
	d.Addrs = []string{device.Address}
	d.Target = device.Target
	if device.Timeout != nil {
		d.Timeout = *device.Timeout
	}
	if device.TLS.Plain {
		log.Info("Plain connection connection to ", device.Address)
	} else if device.TLS.Insecure {
		log.Info("Insecure connection to ", device.Address)
		d.TLS = &tls.Config{InsecureSkipVerify: true}
	} else {
		d.TLS = &tls.Config{}
		if device.TLS.CaCert == "" {
			log.Info("Loading default CA onfca")
			d.TLS.RootCAs = getCertPoolDefault()
		} else {
			d.TLS.RootCAs = getCertPool(device.TLS.CaCert)
		}
		if device.TLS.Cert == "" && device.TLS.Key == "" {
			// Load default Certificates
			log.Info("Loading default certificates")
			clientCerts, err := tls.X509KeyPair([]byte(certs.DefaultClientCrt), []byte(certs.DefaultClientKey))
			if err != nil {
				log.Error("Error loading default certs")
			}
			d.TLS.Certificates = []tls.Certificate{clientCerts}
		} else if device.TLS.Cert != "" && device.TLS.Key != "" {
			// Load certs given for device
			d.TLS.Certificates = []tls.Certificate{setCertificate(device.TLS.Cert, device.TLS.Key)}
		} else if device.Credentials.User != "" && device.Credentials.Password != "" {
			cred := &client.Credentials{}
			cred.Username = device.Credentials.User
			cred.Password = device.Credentials.Password
			d.Credentials = cred
		} else {
			log.Errorf("Can't load Ca=%s , Cert=%s , key=%s for %v, trying with insecure connection",
				device.TLS.CaCert, device.TLS.Cert, device.TLS.Key, device.Address)
			d.TLS = &tls.Config{InsecureSkipVerify: true}
		}
	}
	return d, DeviceID{DeviceID: device.Address}
}

// GetTarget attempts to get a specific target from the targets cache
func GetTarget(key DeviceID) (*Target, error) {
	_, ok := targets[key].(*Target)
	if ok {
		return targets[key].(*Target), nil
	}
	return &Target{}, fmt.Errorf("Client for %v does not exist, create first", key)

}

// ConnectTarget connects to a given Device according to the passed information establishing a channel to it.
//TODO make asyc
//TODO lock channel to allow one request to device at each time
func (target *Target) ConnectTarget(ctx context.Context, device devicepb.Device) (DeviceID, error) {
	dest, key := createDestination(device)
	c, err := GnmiClientFactory(ctx, *dest)

	//c.handler := client.NotificationHandler{}
	if err != nil {
		return DeviceID{}, fmt.Errorf("could not create a gNMI client: %v", err)
	}
	target.Destination = *dest
	target.Clt = c
	target.Ctx = ctx
	targets[key] = target
	return key, err
}

func setCertificate(pathCert string, pathKey string) tls.Certificate {
	certificate, err := tls.LoadX509KeyPair(pathCert, pathKey)
	if err != nil {
		log.Error("could not load client key pair ", err)
	}
	return certificate
}

func getCertPool(CaPath string) *x509.CertPool {
	certPool := x509.NewCertPool()
	ca, err := ioutil.ReadFile(CaPath)
	if err != nil {
		log.Error("could not read ", CaPath, err)
	}
	if ok := certPool.AppendCertsFromPEM(ca); !ok {
		log.Error("failed to append CA certificates")
	}
	return certPool
}

func getCertPoolDefault() *x509.CertPool {
	certPool := x509.NewCertPool()
	if ok := certPool.AppendCertsFromPEM([]byte(certs.OnfCaCrt)); !ok {
		log.Error("failed to append CA certificates")
	}
	return certPool
}

// CapabilitiesWithString allows a request for the capabilities by a string - can be empty
func (target *Target) CapabilitiesWithString(ctx context.Context, request string) (*gpb.CapabilityResponse, error) {
	r := &gpb.CapabilityRequest{}
	reqProto := &request
	if err := proto.UnmarshalText(*reqProto, r); err != nil {
		return nil, fmt.Errorf("unable to parse gnmi.CapabilityRequest from %q : %v", *reqProto, err)
	}
	return target.Capabilities(ctx, r)
}

// Capabilities get capabilities according to a formatted request
func (target *Target) Capabilities(ctx context.Context, request *gpb.CapabilityRequest) (*gpb.CapabilityResponse, error) {
	response, err := target.Clt.Capabilities(ctx, request)
	if err != nil {
		return nil, fmt.Errorf("target returned RPC error for Capabilities(%q): %v", request.String(), err)
	}
	return response, nil
}

// GetWithString can make a get request according by a string - can be empty
func (target *Target) GetWithString(ctx context.Context, request string) (*gpb.GetResponse, error) {
	if request == "" {
		return nil, errors.New("cannot get and empty request")
	}
	r := &gpb.GetRequest{}
	reqProto := &request
	if err := proto.UnmarshalText(*reqProto, r); err != nil {
		return nil, fmt.Errorf("unable to parse gnmi.GetRequest from %q : %v", *reqProto, err)
	}
	return target.Get(ctx, r)
}

// Get can make a get request according to a formatted request
func (target *Target) Get(ctx context.Context, request *gpb.GetRequest) (*gpb.GetResponse, error) {
	response, err := target.Clt.Get(ctx, request)
	if err != nil {
		return nil, fmt.Errorf("target returned RPC error for Get(%q) : %v", request.String(), err)
	}
	return response, nil
}

// SetWithString can make a set request according by a string
func (target *Target) SetWithString(ctx context.Context, request string) (*gpb.SetResponse, error) {
	//TODO modify with key that gets target from map
	if request == "" {
		return nil, errors.New("cannot get and empty request")
	}
	r := &gpb.SetRequest{}
	reqProto := &request
	if err := proto.UnmarshalText(*reqProto, r); err != nil {
		return nil, fmt.Errorf("unable to parse gnmi.SetRequest from %q : %v", *reqProto, err)
	}
	return target.Set(ctx, r)
}

// Set can make a set request according to a formatted request
func (target *Target) Set(ctx context.Context, request *gpb.SetRequest) (*gpb.SetResponse, error) {
	response, err := target.Clt.Set(ctx, request)
	if err != nil {
		return nil, fmt.Errorf("target returned RPC error for Set(%q) : %v", request.String(), err)
	}
	return response, nil
}

// Subscribe initiates a subscription to a target and set of paths by establishing a new channel
func (target *Target) Subscribe(ctx context.Context, request *gpb.SubscribeRequest, handler client.ProtoHandler) error {
	//TODO currently establishing a throwaway client per each subscription request
	//this is due to the face that 1 NotificationHandler is allowed per client (1:1)
	//alternatively we could handle every connection request with one NotificationHandler
	//returing to the caller only the desired results.
	q, err := client.NewQuery(request)
	if err != nil {
		return err
	}
	q.Addrs = target.Destination.Addrs
	q.Timeout = target.Destination.Timeout
	q.Target = target.Destination.Target
	q.Credentials = target.Destination.Credentials
	q.TLS = target.Destination.TLS
	q.ProtoHandler = handler
	c := GnmiBaseClientFactory()
	err = c.Subscribe(ctx, q, "gnmi")
	if err != nil {
		return fmt.Errorf("could not create a gNMI for subscription: %v", err)
	}
	return err
}

// NewSubscribeRequest returns a SubscribeRequest for the given paths
func NewSubscribeRequest(subscribeOptions *SubscribeOptions) (*gpb.SubscribeRequest, error) {
	var mode gpb.SubscriptionList_Mode
	switch strings.ToUpper(subscribeOptions.Mode) {
	case gpb.SubscriptionList_ONCE.String():
		mode = gpb.SubscriptionList_ONCE
	case gpb.SubscriptionList_POLL.String():
		mode = gpb.SubscriptionList_POLL
	case "":
		fallthrough
	case gpb.SubscriptionList_STREAM.String():
		mode = gpb.SubscriptionList_STREAM
	default:
		return nil, fmt.Errorf("subscribe mode (%s) invalid", subscribeOptions.Mode)
	}

	var streamMode gpb.SubscriptionMode
	switch strings.ToUpper(subscribeOptions.StreamMode) {
	case gpb.SubscriptionMode_ON_CHANGE.String():
		streamMode = gpb.SubscriptionMode_ON_CHANGE
	case gpb.SubscriptionMode_SAMPLE.String():
		streamMode = gpb.SubscriptionMode_SAMPLE
	case "":
		fallthrough
	case gpb.SubscriptionMode_TARGET_DEFINED.String():
		streamMode = gpb.SubscriptionMode_TARGET_DEFINED
	default:
		return nil, fmt.Errorf("subscribe stream mode (%s) invalid", subscribeOptions.StreamMode)
	}

	prefixPath, err := utils.ParseGNMIElements(utils.SplitPath(subscribeOptions.Prefix))
	if err != nil {
		return nil, err
	}
	subList := &gpb.SubscriptionList{
		Subscription: make([]*gpb.Subscription, len(subscribeOptions.Paths)),
		Mode:         mode,
		UpdatesOnly:  subscribeOptions.UpdatesOnly,
		Prefix:       prefixPath,
	}
	for i, p := range subscribeOptions.Paths {
		gnmiPath, err := utils.ParseGNMIElements(p)
		if err != nil {
			return nil, err
		}
		gnmiPath.Origin = subscribeOptions.Origin
		subList.Subscription[i] = &gpb.Subscription{
			Path:              gnmiPath,
			Mode:              streamMode,
			SampleInterval:    subscribeOptions.SampleInterval,
			HeartbeatInterval: subscribeOptions.HeartbeatInterval,
		}
	}
	return &gpb.SubscribeRequest{Request: &gpb.SubscribeRequest_Subscribe{
		Subscribe: subList}}, nil
}
