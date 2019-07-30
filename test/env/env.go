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

package env

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"errors"
	"fmt"
	"github.com/onosproject/onos-config/pkg/northbound"
	"github.com/onosproject/onos-config/pkg/northbound/proto"
	"github.com/openconfig/gnmi/client"
	gnmi "github.com/openconfig/gnmi/client/gnmi"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"io/ioutil"
	"os"
	"strings"
	"time"
)

const (
	// TestDevicesEnv : environment variable name for devices
	TestDevicesEnv = "ONOS_CONFIG_TEST_DEVICES"
)

const (
	clientKeyPath = "/etc/onos-config/certs/client1.key"
	clientCrtPath = "/etc/onos-config/certs/client1.crt"
	caCertPath    = "/etc/onos-config/certs/onf.cacrt"
	address       = "onos-config:5150"
)

// GetCredentials returns gNMI client credentials for the test environment
func GetCredentials() (*tls.Config, error) {
	certPool := x509.NewCertPool()
	ca, err := ioutil.ReadFile(caCertPath)
	if err != nil {
		return nil, err
	}

	if !certPool.AppendCertsFromPEM(ca) {
		return nil, errors.New("failed to append CA certificates")
	}

	cert, err := tls.LoadX509KeyPair(clientCrtPath, clientKeyPath)
	if err != nil {
		return nil, err
	}

	return &tls.Config{
		RootCAs:            certPool,
		Certificates:       []tls.Certificate{cert},
		InsecureSkipVerify: true,
	}, nil
}

// GetDestination returns a gNMI client destination for the test environment
func GetDestination(target string) (client.Destination, error) {
	tlsConfig, err := GetCredentials()
	if err != nil {
		return client.Destination{}, err
	}
	return client.Destination{
		Addrs:   []string{address},
		Target:  target,
		TLS:     tlsConfig,
		Timeout: 10 * time.Second,
	}, nil
}

// GetDestinationForDevice returns a gNMI client destination for the test environment
func GetDestinationForDevice(addr string, target string) (client.Destination, error) {
	tlsConfig, err := GetCredentials()
	if err != nil {
		return client.Destination{}, err
	}
	return client.Destination{
		Addrs:   []string{addr},
		Target:  target,
		TLS:     tlsConfig,
		Timeout: 10 * time.Second,
	}, nil
}

// NewGnmiClient returns a new gNMI client for the test environment
func NewGnmiClient(ctx context.Context, target string) (client.Impl, error) {
	dest, err := GetDestination(target)
	if err != nil {
		return nil, err
	}
	return gnmi.New(ctx, dest)
}

// NewGnmiClientForDevice returns a new gNMI client for the test environment
func NewGnmiClientForDevice(ctx context.Context, address string, target string) (client.Impl, error) {
	dest, err := GetDestinationForDevice(address, target)
	if err != nil {
		return nil, err
	}
	return gnmi.New(ctx, dest)
}

// GetDevices returns a slice of device names for the test environment
func GetDevices() []string {
	devices := os.Getenv(TestDevicesEnv)
	return strings.Split(devices, ",")
}

func handleCertArgs() ([]grpc.DialOption, error) {
	var opts = []grpc.DialOption{}
	var cert tls.Certificate
	var err error

	// Load default Certificates
	cert, err = tls.LoadX509KeyPair(clientCrtPath, clientKeyPath)
	if err != nil {
		return nil, err
	}

	tlsConfig := &tls.Config{
		Certificates:       []tls.Certificate{cert},
		InsecureSkipVerify: true,
	}
	opts = append(opts, grpc.WithTransportCredentials(credentials.NewTLS(tlsConfig)))

	return opts, nil
}

// GetAdminClient returns a client that can be used for the admin APIs
func GetAdminClient() (*grpc.ClientConn, proto.ConfigAdminServiceClient) {
	opts, err := handleCertArgs()
	if err != nil {
		fmt.Printf("Error loading cert %s", err)
	}
	conn := northbound.Connect(address, opts...)
	return conn, proto.NewConfigAdminServiceClient(conn)
}
