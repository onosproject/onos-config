// SPDX-FileCopyrightText: 2020-present Open Networking Foundation <info@opennetworking.org>
//
// SPDX-License-Identifier: Apache-2.0

package test

import (
	"crypto/tls"
	"fmt"
	"github.com/onosproject/onos-api/go/onos/config/admin"
	"github.com/onosproject/onos-lib-go/pkg/certs"
	"github.com/onosproject/onos-lib-go/pkg/grpc/retry"
	toposdk "github.com/onosproject/onos-ric-sdk-go/pkg/topo"
	gnmiclient "github.com/openconfig/gnmi/client"
	gclient "github.com/openconfig/gnmi/client/gnmi"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/credentials/insecure"
	"math"
	"time"
)

// RetryOption specifies if a client should retry request errors
type RetryOption int

const (
	// NoRetry do not attempt to retry
	NoRetry RetryOption = iota

	// WithRetry adds a retry option to the client
	WithRetry
)

const (
	onosConfigName = "onos-config"
	onosConfigPort = "5150"
	onosConfig     = onosConfigName + ":" + onosConfigPort
)

// NewTopoClient creates a topology client
func (s *Suite) NewTopoClient() (toposdk.Client, error) {
	return toposdk.NewClient()
}

// NewAdminServiceClient :
func (s *Suite) NewAdminServiceClient() (admin.ConfigAdminServiceClient, error) {
	conn, err := s.getOnosConfigConnection()
	if err != nil {
		return nil, err
	}
	return admin.NewConfigAdminServiceClient(conn), nil
}

// NewTransactionServiceClient :
func (s *Suite) NewTransactionServiceClient() (admin.TransactionServiceClient, error) {
	conn, err := s.getOnosConfigConnection()
	if err != nil {
		return nil, err
	}
	return admin.NewTransactionServiceClient(conn), nil
}

// NewConfigurationServiceClient returns configuration store client
func (s *Suite) NewConfigurationServiceClient() (admin.ConfigurationServiceClient, error) {
	conn, err := s.getOnosConfigConnection()
	if err != nil {
		return nil, err
	}
	return admin.NewConfigurationServiceClient(conn), nil
}

// GetOnosConfigDestination returns a gnmi Destination for the onos-config service
func (s *Suite) GetOnosConfigDestination() (gnmiclient.Destination, error) {
	creds, err := s.getClientCredentials()
	if err != nil {
		return gnmiclient.Destination{}, err
	}

	return gnmiclient.Destination{
		Addrs:   []string{onosConfig},
		Target:  onosConfigName,
		TLS:     creds,
		Timeout: 10 * time.Second,
	}, nil
}

func (s *Suite) getOnosConfigConnection() (*grpc.ClientConn, error) {
	tlsConfig, err := s.getClientCredentials()
	if err != nil {
		return nil, err
	}
	return grpc.DialContext(s.Context(), onosConfig, grpc.WithTransportCredentials(credentials.NewTLS(tlsConfig)))
}

// getClientCredentials returns the credentials for a service client
func (s *Suite) getClientCredentials() (*tls.Config, error) {
	cert, err := tls.X509KeyPair([]byte(certs.DefaultClientCrt), []byte(certs.DefaultClientKey))
	if err != nil {
		return nil, err
	}
	return &tls.Config{
		Certificates:       []tls.Certificate{cert},
		InsecureSkipVerify: true,
	}, nil
}

// NewSimulatorGNMIClientOrFail creates a GNMI client to a target. If there is an error, the test is failed
func (s *Suite) NewSimulatorGNMIClientOrFail(simulator string) gnmiclient.Impl {
	s.T().Helper()
	dest := gnmiclient.Destination{
		Addrs:   []string{fmt.Sprintf("%s-device-simulator:11161", simulator)},
		Target:  simulator,
		Timeout: 10 * time.Second,
	}
	opts := []grpc.DialOption{grpc.WithTransportCredentials(insecure.NewCredentials())}
	return s.newGNMIClientOrFail(dest, opts)
}

// NewOnosConfigGNMIClientOrFail makes a GNMI client to use for requests. If creating the client fails, the test is failed.
func (s *Suite) NewOnosConfigGNMIClientOrFail(retryOption RetryOption) gnmiclient.Impl {
	s.T().Helper()
	dest, err := s.GetOnosConfigDestination()
	s.NoError(err)
	opts := make([]grpc.DialOption, 0)
	opts = append(opts, grpc.WithTransportCredentials(credentials.NewTLS(dest.TLS)))
	if retryOption == WithRetry {
		opts = append(opts, grpc.WithUnaryInterceptor(retry.RetryingUnaryClientInterceptor()))
	}

	return s.newGNMIClientOrFail(dest, opts)
}

// newGNMIClientOrFail returns a gnmi client
func (s *Suite) newGNMIClientOrFail(dest gnmiclient.Destination, opts []grpc.DialOption) gnmiclient.Impl {
	opts = append(opts, grpc.WithBlock())
	opts = append(opts, grpc.WithDefaultCallOptions(grpc.MaxCallRecvMsgSize(math.MaxInt32)))
	conn, err := grpc.DialContext(s.Context(), dest.Addrs[0], opts...)
	s.NoError(err)
	client, err := gclient.NewFromConn(s.Context(), conn, dest)
	s.NoError(err)
	return client
}
