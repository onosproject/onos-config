package env

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"errors"
	"github.com/openconfig/gnmi/client"
	gnmi "github.com/openconfig/gnmi/client/gnmi"
	"io/ioutil"
	"os"
	"strings"
)

const (
	TestDevicesEnv = "ONOS_CONFIG_TEST_DEVICES"
)

const (
	clientKeyPath = "/etc/onos-config/certs/tls.key"
	clientCrtPath = "/etc/onos-config/certs/tls.crt"
	caCertPath    = "/etc/onos-config/certs/tls.cacrt"
	Address       = "onos-config:5150"
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
		RootCAs:      certPool,
		Certificates: []tls.Certificate{cert},
	}, nil
}

// GetDestination returns a gNMI client destination for the test environment
func GetDestination(target string) (client.Destination, error) {
	tlsConfig, err := GetCredentials()
	if err != nil {
		return client.Destination{}, err
	}
	return client.Destination{
		Addrs:  []string{Address},
		Target: target,
		TLS:    tlsConfig,
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

// GetDevices returns a slice of device names for the test environment
func GetDevices() []string {
	devices := os.Getenv(TestDevicesEnv)
	return strings.Split(devices, ",")
}
