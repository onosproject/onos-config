// SPDX-FileCopyrightText: 2020-present Open Networking Foundation <info@opennetworking.org>
//
// SPDX-License-Identifier: Apache-2.0
//

package gnmi

import (
	"context"
	gclient "github.com/openconfig/gnmi/client/gnmi"
	"math"
	"testing"
	"time"

	toposdk "github.com/onosproject/onos-ric-sdk-go/pkg/topo"

	"github.com/onosproject/helmit/pkg/helm"
	"github.com/onosproject/helmit/pkg/kubernetes"
	"github.com/onosproject/onos-api/go/onos/config/admin"
	"github.com/onosproject/onos-lib-go/pkg/grpc/retry"
	gnmiclient "github.com/openconfig/gnmi/client"
	"github.com/stretchr/testify/assert"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
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

func getOnosConfigConnection() (*grpc.ClientConn, error) {
	tlsConfig, err := getClientCredentials()
	if err != nil {
		return nil, err
	}
	return grpc.Dial(onosConfig, grpc.WithTransportCredentials(credentials.NewTLS(tlsConfig)))
}

// NewTopoClient creates a topology client
func NewTopoClient() (toposdk.Client, error) {
	return toposdk.NewClient()
}

// NewAdminServiceClient :
func NewAdminServiceClient(ctx context.Context) (admin.ConfigAdminServiceClient, error) {
	conn, err := getOnosConfigConnection()
	if err != nil {
		return nil, err
	}
	return admin.NewConfigAdminServiceClient(conn), nil
}

// NewTransactionServiceClient :
func NewTransactionServiceClient(ctx context.Context) (admin.TransactionServiceClient, error) {
	conn, err := getOnosConfigConnection()
	if err != nil {
		return nil, err
	}
	return admin.NewTransactionServiceClient(conn), nil
}

// NewConfigurationServiceClient returns configuration store client
func NewConfigurationServiceClient(ctx context.Context) (admin.ConfigurationServiceClient, error) {
	conn, err := getOnosConfigConnection()
	if err != nil {
		return nil, err
	}
	return admin.NewConfigurationServiceClient(conn), nil
}

// GetOnosConfigDestination returns a gnmi Destination for the onos-config service
func GetOnosConfigDestination() (gnmiclient.Destination, error) {
	creds, err := getClientCredentials()
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

// NewSimulatorGNMIClientOrFail creates a GNMI client to a target. If there is an error, the test is failed
func NewSimulatorGNMIClientOrFail(ctx context.Context, t *testing.T, simulator *helm.HelmRelease) gnmiclient.Impl {
	t.Helper()
	simulatorClient := kubernetes.NewForReleaseOrDie(simulator)
	services, err := simulatorClient.CoreV1().Services().List(context.Background())
	assert.NoError(t, err)
	service := services[0]
	dest := gnmiclient.Destination{
		Addrs:   []string{service.Ports()[0].Address(true)},
		Target:  service.Name,
		Timeout: 10 * time.Second,
	}
	opts := []grpc.DialOption{grpc.WithInsecure()}
	return newGNMIClientOrFail(ctx, t, dest, opts)
}

// NewOnosConfigGNMIClientOrFail makes a GNMI client to use for requests. If creating the client fails, the test is failed.
func NewOnosConfigGNMIClientOrFail(ctx context.Context, t *testing.T, retryOption RetryOption) gnmiclient.Impl {
	t.Helper()
	dest, err := GetOnosConfigDestination()
	assert.NoError(t, err)
	opts := make([]grpc.DialOption, 0)
	opts = append(opts, grpc.WithTransportCredentials(credentials.NewTLS(dest.TLS)))
	if retryOption == WithRetry {
		opts = append(opts, grpc.WithUnaryInterceptor(retry.RetryingUnaryClientInterceptor()))
	}

	return newGNMIClientOrFail(ctx, t, dest, opts)
}

// newGNMIClientOrFail returns a gnmi client
func newGNMIClientOrFail(ctx context.Context, t *testing.T, dest gnmiclient.Destination, opts []grpc.DialOption) gnmiclient.Impl {
	opts = append(opts, grpc.WithBlock())
	opts = append(opts, grpc.WithDefaultCallOptions(grpc.MaxCallRecvMsgSize(math.MaxInt32)))
	conn, err := grpc.DialContext(ctx, dest.Addrs[0], opts...)
	assert.NoError(t, err)
	client, err := gclient.NewFromConn(ctx, conn, dest)
	assert.NoError(t, err)
	return client
}
