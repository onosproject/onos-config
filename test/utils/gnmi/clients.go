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

	"github.com/onosproject/onos-lib-go/pkg/errors"

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

func getService(ctx context.Context, release *helm.HelmRelease, serviceName string) (*v1.Service, error) {
	releaseClient := kubernetes.NewForReleaseOrDie(release)
	service, err := releaseClient.CoreV1().Services().Get(ctx, serviceName)
	if err != nil {
		return nil, err
	}

	return service, nil
}

func connectComponent(ctx context.Context, releaseName string, deploymentName string) (*grpc.ClientConn, error) {
	release := helm.Chart(releaseName).Release(releaseName)
	return connectService(ctx, release, deploymentName)
}

func connectService(ctx context.Context, release *helm.HelmRelease, deploymentName string) (*grpc.ClientConn, error) {
	service, err := getService(ctx, release, deploymentName)
	if err != nil {
		return nil, err
	}
	tlsConfig, err := getClientCredentials()
	if err != nil {
		return nil, err
	}
	return grpc.Dial(service.Ports()[0].Address(true), grpc.WithTransportCredentials(credentials.NewTLS(tlsConfig)))
}

// NewTopoClient creates a topology client
func NewTopoClient() (toposdk.Client, error) {
	return toposdk.NewClient()
}

// NewAdminServiceClient :
func NewAdminServiceClient(ctx context.Context) (admin.ConfigAdminServiceClient, error) {
	conn, err := connectComponent(ctx, "onos-umbrella", "onos-config")
	if err != nil {
		return nil, err
	}
	return admin.NewConfigAdminServiceClient(conn), nil
}

// NewTransactionServiceClient :
func NewTransactionServiceClient(ctx context.Context) (admin.TransactionServiceClient, error) {
	conn, err := connectComponent(ctx, "onos-umbrella", "onos-config")
	if err != nil {
		return nil, err
	}
	return admin.NewTransactionServiceClient(conn), nil
}

// NewConfigurationServiceClient returns configuration store client
func NewConfigurationServiceClient(ctx context.Context) (admin.ConfigurationServiceClient, error) {
	conn, err := connectComponent(ctx, "onos-umbrella", "onos-config")
	if err != nil {
		return nil, err
	}
	return admin.NewConfigurationServiceClient(conn), nil
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
	client, err := gclient.New(ctx, dest)
	assert.NoError(t, err)
	assert.True(t, client != nil, "Fetching target client returned nil")
	return client
}

// GetOnosConfigDestination :
func GetOnosConfigDestination(ctx context.Context) (gnmiclient.Destination, error) {
	creds, err := getClientCredentials()
	if err != nil {
		return gnmiclient.Destination{}, err
	}
	configRelease := helm.Release("onos-umbrella")
	configClient := kubernetes.NewForReleaseOrDie(configRelease)

	configService, err := configClient.CoreV1().Services().Get(ctx, "onos-config")
	if err != nil || configService == nil {
		return gnmiclient.Destination{}, errors.NewNotFound("can't find service for onos-config")
	}

	return gnmiclient.Destination{
		Addrs:   []string{configService.Ports()[0].Address(true)},
		Target:  configService.Name,
		TLS:     creds,
		Timeout: 10 * time.Second,
	}, nil
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

	conn, err := grpc.DialContext(ctx, dest.Addrs[0], opts...)
	assert.NoError(t, err)
	client, err := gclient.NewFromConn(ctx, conn, dest)
	assert.NoError(t, err)
	return client
}
