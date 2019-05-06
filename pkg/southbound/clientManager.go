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
	"io/ioutil"
	"time"

	"github.com/golang/protobuf/proto"
	"github.com/openconfig/gnmi/client"
	gclient "github.com/openconfig/gnmi/client/gnmi"
	gpb "github.com/openconfig/gnmi/proto/gnmi"
)

type Device struct {
	Addr, Target, Usr, Pwd, CaPath, CertPath, KeyPath string
	Timeout                                           time.Duration
}

//Can be extended, for now is ip:port
type Key struct {
	Key string
}

type Target struct {
	Destination client.Destination
	Clt         client.Impl
	Ctxt        context.Context
}

// SubscribeOptions is the gNMI subscription request options
type SubscribeOptions struct {
	UpdatesOnly       bool
	Prefix            string
	Mode              string
	StreamMode        string
	SampleInterval    uint64
	HeartbeatInterval uint64
	Paths             [][]string
	Origin            string
}

var targets = make(map[Key]Target)

func createDestination(device Device) (*client.Destination, Key) {
	d := &client.Destination{TLS: &tls.Config{}}
	d.Addrs = []string{device.Addr}
	d.Target = device.Target
	d.Timeout = time.Duration(device.Timeout * time.Second)
	if device.CertPath != "" && device.KeyPath != "" {
		if device.CaPath != "" {
			certPool := getCertPool(device.CaPath)
			d.TLS.RootCAs = certPool
		} else {
			fmt.Println("Assuming well know CA for certificate signature")
		}
		// Certificates
		d.TLS.Certificates = []tls.Certificate{setCertificate(device.CertPath, device.KeyPath)}
	} else if device.Usr != "" && device.Pwd != "" {
		//TODO implement
		// cred := &client.Credentials{}
		// cred.Username = "test"
		// cred.Password = "testpwd"
		//d.Credentials = cred
		//fmt.Println(*cred)
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

//TODO make asyc
//TODO lock channel to allow one request to device at each time
func ConnectTarget(device Device) (Target, Key, error) {
	dest, key := createDestination(device)
	ctx := context.Background()
	c, err := gclient.New(ctx, *dest)
	//c.handler := client.NotificationHandler{}
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
		fmt.Println("could not load client key pair: %v", err)
	}
	return certificate
}

func getCertPool(CaPath string) *x509.CertPool {
	certPool := x509.NewCertPool()
	ca, err := ioutil.ReadFile(CaPath)
	if err != nil {
		fmt.Println("could not read %q: %s", CaPath, err)
	}
	if ok := certPool.AppendCertsFromPEM(ca); !ok {
		fmt.Println("failed to append CA certificates")
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

func Subscribe(target Target, request *gpb.SubscribeRequest, handler client.NotificationHandler) error {
	//TODO currently establishing a throwaway client per each subscription request
	//this is due to the face that 1 NotificationHandler is allowed per client (1:1)
	//alternatively we could handle every connection request with one NotificationHandler
	//returing to the caller only the desired results.
	q, err := client.NewQuery(request)
	if err != nil {
		//TODO handle
	}
	q.Addrs = target.Destination.Addrs
	q.Timeout = target.Destination.Timeout
	q.Target = target.Destination.Target
	q.Credentials = target.Destination.Credentials
	q.TLS = target.Destination.TLS
	q.NotificationHandler = handler
	c := client.New()
	err = c.Subscribe(target.Ctxt, q, "")
	if err != nil {
		return fmt.Errorf("could not create a gNMI for subscription: %v", err)
	}
	target.Clt.(*gclient.Client).Subscribe(target.Ctxt, q)
	return nil

}

// NewSubscribeRequest returns a SubscribeRequest for the given paths
func NewSubscribeRequest(subscribeOptions *SubscribeOptions) (*gpb.SubscribeRequest, error) {
	var mode gpb.SubscriptionList_Mode
	switch subscribeOptions.Mode {
	case "once":
		mode = gpb.SubscriptionList_ONCE
	case "poll":
		mode = gpb.SubscriptionList_POLL
	case "":
		fallthrough
	case "stream":
		mode = gpb.SubscriptionList_STREAM
	default:
		return nil, fmt.Errorf("subscribe mode (%s) invalid", subscribeOptions.Mode)
	}

	var streamMode gpb.SubscriptionMode
	switch subscribeOptions.StreamMode {
	case "on_change":
		streamMode = gpb.SubscriptionMode_ON_CHANGE
	case "sample":
		streamMode = gpb.SubscriptionMode_SAMPLE
	case "":
		fallthrough
	case "target_defined":
		streamMode = gpb.SubscriptionMode_TARGET_DEFINED
	default:
		return nil, fmt.Errorf("subscribe stream mode (%s) invalid", subscribeOptions.StreamMode)
	}

	prefixPath, err := ParseGNMIElements(SplitPath(subscribeOptions.Prefix))
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
		gnmiPath, err := ParseGNMIElements(p)
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
