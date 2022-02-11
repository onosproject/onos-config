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
//

package gnmi

import (
	"context"
	gclient "github.com/openconfig/gnmi/client/gnmi"
	"testing"
	"time"

	toposdk "github.com/onosproject/onos-ric-sdk-go/pkg/topo"

	"github.com/onosproject/helmit/pkg/helm"
	"github.com/onosproject/helmit/pkg/kubernetes"
	v1 "github.com/onosproject/helmit/pkg/kubernetes/core/v1"
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

func getService(ctx context.Context, release *helm.HelmRelease, serviceName string) (*v1.Service, error) {
	releaseClient := kubernetes.NewForReleaseOrDie(release)
	service, err := releaseClient.CoreV1().Services().Get(ctx, serviceName)
	if err != nil {
		return nil, err
	}

	return service, nil
}

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
func GetOnosConfigDestination(ctx context.Context) (gnmiclient.Destination, error) {
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

// GetTargetGNMIClientOrFail creates a GNMI client to a target. If there is an error, the test is failed
func GetTargetGNMIClientOrFail(ctx context.Context, t *testing.T, simulator *helm.HelmRelease) gnmiclient.Impl {
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
	return newClientOrFail(ctx, t, dest, opts)
}

// GetGNMIClientOrFail makes a GNMI client to use for requests. If creating the client fails, the test is failed.
func GetGNMIClientOrFail(ctx context.Context, t *testing.T, retryOption RetryOption) gnmiclient.Impl {
	t.Helper()
	dest, err := GetOnosConfigDestination(ctx)
	assert.NoError(t, err)
	opts := make([]grpc.DialOption, 0)
	opts = append(opts, grpc.WithTransportCredentials(credentials.NewTLS(dest.TLS)))
	if retryOption == WithRetry {
		opts = append(opts, grpc.WithUnaryInterceptor(retry.RetryingUnaryClientInterceptor()))
	}

	return newClientOrFail(ctx, t, dest, opts)
}

// newClientOrFail returns a gnmi client
func newClientOrFail(ctx context.Context, t *testing.T, dest gnmiclient.Destination, opts []grpc.DialOption) gnmiclient.Impl {
	conn, err := grpc.DialContext(ctx, dest.Addrs[0], opts...)
	assert.NoError(t, err)
	client, err := gclient.NewFromConn(ctx, conn, dest)
	assert.NoError(t, err)
	return client
}
