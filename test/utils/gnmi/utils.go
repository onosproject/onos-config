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
	"fmt"
	"io"
	"strings"
	"testing"
	"time"

	configapi "github.com/onosproject/onos-api/go/onos/config/v2"
	toposdk "github.com/onosproject/onos-ric-sdk-go/pkg/topo"

	"github.com/onosproject/onos-api/go/onos/topo"
	"github.com/onosproject/onos-lib-go/pkg/errors"

	"github.com/onosproject/onos-test/pkg/onostest"

	"github.com/golang/protobuf/proto"
	"github.com/onosproject/helmit/pkg/helm"
	"github.com/onosproject/helmit/pkg/kubernetes"
	v1 "github.com/onosproject/helmit/pkg/kubernetes/core/v1"
	"github.com/onosproject/helmit/pkg/util/random"
	"github.com/onosproject/onos-api/go/onos/config/admin"
	"github.com/onosproject/onos-api/go/onos/config/v2"
	"github.com/onosproject/onos-config/pkg/utils"
	protoutils "github.com/onosproject/onos-config/test/utils/proto"
	"github.com/onosproject/onos-lib-go/pkg/grpc/retry"
	gnmiclient "github.com/openconfig/gnmi/client"
	gclient "github.com/openconfig/gnmi/client/gnmi"
	gpb "github.com/openconfig/gnmi/proto/gnmi"
	"github.com/openconfig/gnmi/proto/gnmi_ext"
	"github.com/stretchr/testify/assert"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
)

const (
	// SimulatorTargetVersion default version for simulated target
	SimulatorTargetVersion = "1.0.0"
	// SimulatorTargetType type for simulated target
	SimulatorTargetType = "devicesim"

	defaultGNMITimeout = time.Second * 30

	// Maximum time for an entire test to complete
	defaultTestTimeout = 2 * time.Minute
)

// RetryOption specifies if a client should retry request errors
type RetryOption int

const (
	// NoRetry do not attempt to retry
	NoRetry RetryOption = iota

	// WithRetry adds a retry option to the client
	WithRetry
)

// MakeContext returns a new context for use in GNMI requests
func MakeContext() (context.Context, context.CancelFunc) {
	ctx := context.Background()
	return context.WithTimeout(ctx, defaultTestTimeout)
}

func getService(release *helm.HelmRelease, serviceName string) (*v1.Service, error) {
	releaseClient := kubernetes.NewForReleaseOrDie(release)
	service, err := releaseClient.CoreV1().Services().Get(context.Background(), serviceName)
	if err != nil {
		return nil, err
	}

	return service, nil
}

func connectComponent(releaseName string, deploymentName string) (*grpc.ClientConn, error) {
	release := helm.Chart(releaseName).Release(releaseName)
	return connectService(release, deploymentName)
}

func connectService(release *helm.HelmRelease, deploymentName string) (*grpc.ClientConn, error) {
	service, err := getService(release, deploymentName)
	if err != nil {
		return nil, err
	}
	tlsConfig, err := getClientCredentials()
	if err != nil {
		return nil, err
	}
	return grpc.Dial(service.Ports()[0].Address(true), grpc.WithTransportCredentials(credentials.NewTLS(tlsConfig)))
}

// GetSimulatorTarget queries topo to find the topo object for a simulator target
func GetSimulatorTarget(simulator *helm.HelmRelease) (*topo.Object, error) {
	client, err := NewTopoClient()
	if err != nil {
		return nil, err
	}
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	obj, err := client.Get(ctx, topo.ID(simulator.Name()))
	if err != nil {
		return nil, err
	}
	return obj, nil
}

// NewSimulatorTargetEntity creates a topo entity for a device simulator target
func NewSimulatorTargetEntity(simulator *helm.HelmRelease, targetType string, targetVersion string) (*topo.Object, error) {
	simulatorClient := kubernetes.NewForReleaseOrDie(simulator)
	services, err := simulatorClient.CoreV1().Services().List(context.Background())
	if err != nil {
		return nil, err
	}
	service := services[0]
	return NewTargetEntity(simulator.Name(), targetType, targetVersion, service.Ports()[0].Address(true))
}

// NewTargetEntity creates a topo entity with the specified target name, type, version and service address
func NewTargetEntity(name string, targetType string, targetVersion string, serviceAddress string) (*topo.Object, error) {
	o := topo.Object{
		ID:   topo.ID(name),
		Type: topo.Object_ENTITY,
		Obj: &topo.Object_Entity{
			Entity: &topo.Entity{
				KindID: topo.ID(targetType),
			},
		},
	}

	if err := o.SetAspect(&topo.TLSOptions{Insecure: true, Plain: true}); err != nil {
		return nil, err
	}

	timeout := defaultGNMITimeout
	if err := o.SetAspect(&topo.Configurable{
		Type:    targetType,
		Address: serviceAddress,
		Version: targetVersion,
		Timeout: &timeout,
	}); err != nil {
		return nil, err
	}

	return &o, nil
}

// NewTopoClient creates a topology client
func NewTopoClient() (toposdk.Client, error) {
	return toposdk.NewClient()
}

// NewAdminServiceClient :
func NewAdminServiceClient() (admin.ConfigAdminServiceClient, error) {
	conn, err := connectComponent("onos-umbrella", "onos-config")
	if err != nil {
		return nil, err
	}
	return admin.NewConfigAdminServiceClient(conn), nil
}

// NewTransactionServiceClient :
func NewTransactionServiceClient() (admin.TransactionServiceClient, error) {
	conn, err := connectComponent("onos-umbrella", "onos-config")
	if err != nil {
		return nil, err
	}
	return admin.NewTransactionServiceClient(conn), nil
}

// NewConfigurationServiceClient returns configuration store client
func NewConfigurationServiceClient() (admin.ConfigurationServiceClient, error) {
	conn, err := connectComponent("onos-umbrella", "onos-config")
	if err != nil {
		return nil, err
	}
	return admin.NewConfigurationServiceClient(conn), nil
}

// AddTargetToTopo adds a new target to topo
func AddTargetToTopo(targetEntity *topo.Object) error {
	client, err := NewTopoClient()
	if err != nil {
		return err
	}
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	err = client.Create(ctx, targetEntity)
	return err
}

func getKindFilter(kind string) *topo.Filters {
	kindFilter := &topo.Filters{
		KindFilter: &topo.Filter{
			Filter: &topo.Filter_Equal_{
				Equal_: &topo.EqualFilter{
					Value: kind,
				},
			},
		},
	}
	return kindFilter

}

func getControlRelationFilter() *topo.Filters {
	return getKindFilter(topo.CONTROLS)
}

// WaitForControlRelation waits to create control relation for a given target
func WaitForControlRelation(t *testing.T, predicate func(*topo.Relation, topo.Event) bool, timeout time.Duration) bool {
	cl, err := NewTopoClient()
	assert.NoError(t, err)
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	stream := make(chan topo.Event)
	err = cl.Watch(ctx, stream, toposdk.WithWatchFilters(getControlRelationFilter()))
	assert.NoError(t, err)
	for event := range stream {
		if predicate(event.Object.GetRelation(), event) {
			return true
		} // Otherwise, loop and wait for the next topo event
	}

	return false
}

// WaitForTargetAvailable waits for a target to become available
func WaitForTargetAvailable(t *testing.T, objectID topo.ID, timeout time.Duration) bool {
	return WaitForControlRelation(t, func(rel *topo.Relation, event topo.Event) bool {
		if rel.TgtEntityID != objectID {
			t.Logf("Topo %v event from %s (expected %s). Discarding\n", event.Type, rel.TgtEntityID, objectID)
			return false
		}

		if event.Type == topo.EventType_ADDED || event.Type == topo.EventType_UPDATED || event.Type == topo.EventType_NONE {
			cl, err := NewTopoClient()
			assert.NoError(t, err)
			ctx, cancel := context.WithTimeout(context.Background(), timeout)
			defer cancel()
			_, err = cl.Get(ctx, event.Object.ID)
			if err == nil {
				t.Logf("Target %s is available", objectID)
				return true
			}
		}

		return false
	}, timeout)
}

// WaitForTargetUnavailable waits for a target to become available
func WaitForTargetUnavailable(t *testing.T, objectID topo.ID, timeout time.Duration) bool {
	return WaitForControlRelation(t, func(rel *topo.Relation, event topo.Event) bool {
		if rel.TgtEntityID != objectID {
			t.Logf("Topo %v event from %s (expected %s). Discarding\n", event, rel.TgtEntityID, objectID)
			return false
		}

		if event.Type == topo.EventType_REMOVED || event.Type == topo.EventType_NONE {
			cl, err := NewTopoClient()
			assert.NoError(t, err)
			ctx, cancel := context.WithTimeout(context.Background(), timeout)
			defer cancel()
			_, err = cl.Get(ctx, event.Object.ID)
			if errors.IsNotFound(err) {
				t.Logf("Target %s is unavailable", objectID)
				return true
			}
		}
		return false
	}, timeout)
}

// WaitForConfigurationCompleteOrFail wait for a configuration to complete or fail
func WaitForConfigurationCompleteOrFail(t *testing.T, configurationID configapi.ConfigurationID, wait time.Duration) error {
	client, err := NewConfigurationServiceClient()
	assert.NoError(t, err)

	ctx, cancel := context.WithTimeout(context.Background(), wait)
	defer cancel()
	response, err := client.WatchConfigurations(ctx, &admin.WatchConfigurationsRequest{
		ConfigurationID: configurationID,
		Noreplay:        false,
	})
	assert.NoError(t, err)
	for {
		resp, err := response.Recv()
		configuration := resp.ConfigurationEvent.GetConfiguration()
		if err == io.EOF {
			break
		} else if err != nil {
			return errors.NewInvalid(err.Error())
		} else {
			configStatus := configuration.GetStatus()
			if configStatus.GetState() == configapi.ConfigurationStatus_SYNCHRONIZED {
				return nil
			}
		}
	}
	return errors.NewInvalid("configuration %s  failed", configurationID)
}

// WaitForRollback waits for a COMPLETED status on the most recent rollback transaction
func WaitForRollback(t *testing.T, transactionIndex v2.Index, wait time.Duration) bool {
	client, err := NewTransactionServiceClient()
	assert.NoError(t, err)

	ctx, cancel := context.WithTimeout(context.Background(), wait)
	defer cancel()

	stream, err := client.WatchTransactions(ctx, &admin.WatchTransactionsRequest{})
	assert.NoError(t, err)
	assert.NotNil(t, stream)

	start := time.Now()

	for {
		resp, err := stream.Recv()
		if err != nil {
			return false
		}
		assert.NotNil(t, resp)
		fmt.Printf("%v\n", resp.TransactionEvent)

		t := resp.TransactionEvent.Transaction
		if rt := t.GetRollback(); rt != nil {
			if rt.RollbackIndex == transactionIndex {
				return true
			}
		}

		if time.Since(start) > wait {
			return false
		}
	}
}

// NoPaths can be used on a request that does not need path values
var NoPaths = make([]protoutils.TargetPath, 0)

// NoExtensions can be used on a request that does not need extension values
var NoExtensions = make([]*gnmi_ext.Extension, 0)

// SyncExtension returns list of extensions with just the transaction mode extension set to sync and atomic.
func SyncExtension(t *testing.T) []*gnmi_ext.Extension {
	return []*gnmi_ext.Extension{TransactionStrategyExtension(t, configapi.TransactionStrategy_SYNCHRONOUS, 0)}
}

// TransactionStrategyExtension returns a transaction strategy extension populated with the specified fields
func TransactionStrategyExtension(t *testing.T,
	synchronicity configapi.TransactionStrategy_Synchronicity,
	isolation configapi.TransactionStrategy_Isolation) *gnmi_ext.Extension {
	ext := v2.TransactionStrategy{
		Synchronicity: synchronicity,
		Isolation:     isolation,
	}
	b, err := ext.Marshal()
	assert.NoError(t, err)
	return &gnmi_ext.Extension{
		Ext: &gnmi_ext.Extension_RegisteredExt{
			RegisteredExt: &gnmi_ext.RegisteredExtension{
				Id:  v2.TransactionStrategyExtensionID,
				Msg: b,
			},
		},
	}
}

func convertGetResults(response *gpb.GetResponse) ([]protoutils.TargetPath, []*gnmi_ext.Extension, error) {
	entryCount := len(response.Notification)
	result := make([]protoutils.TargetPath, entryCount)

	for index, notification := range response.Notification {
		value := notification.Update[0].Val

		result[index].TargetName = notification.Update[0].Path.Target
		pathString := ""

		for _, elem := range notification.Update[0].Path.Elem {
			pathString = pathString + "/" + elem.Name
		}
		result[index].Path = pathString

		result[index].PathDataType = "string_val"
		if value != nil {
			result[index].PathDataValue = utils.StrVal(value)
		} else {
			result[index].PathDataValue = ""
		}
	}

	return result, response.Extension, nil
}

func extractSetTransactionID(response *gpb.SetResponse) string {
	return string(response.Extension[0].GetRegisteredExt().Msg)
}

// GetGNMIValue generates a GET request on the given client for a Path on a target
func GetGNMIValue(ctx context.Context, c gnmiclient.Impl, paths []protoutils.TargetPath, encoding gpb.Encoding) ([]protoutils.TargetPath, []*gnmi_ext.Extension, error) {
	protoString := ""
	for _, targetPath := range paths {
		protoString = protoString + MakeProtoPath(targetPath.TargetName, targetPath.Path)
	}
	getTZRequest := &gpb.GetRequest{}
	if err := proto.UnmarshalText(protoString, getTZRequest); err != nil {
		fmt.Printf("unable to parse gnmi.GetRequest from %q : %v\n", protoString, err)
		return nil, nil, err
	}
	getTZRequest.Encoding = encoding
	response, err := c.(*gclient.Client).Get(ctx, getTZRequest)
	if err != nil || response == nil {
		return nil, nil, err
	}
	return convertGetResults(response)
}

// SetGNMIValue generates a SET request on the given client for update and delete paths on a target
func SetGNMIValue(ctx context.Context, c gnmiclient.Impl, updatePaths []protoutils.TargetPath, deletePaths []protoutils.TargetPath, extensions []*gnmi_ext.Extension) (string, []*gnmi_ext.Extension, error) {
	var protoBuilder strings.Builder
	for _, updatePath := range updatePaths {
		protoBuilder.WriteString(protoutils.MakeProtoUpdatePath(updatePath))
	}
	for _, deletePath := range deletePaths {
		protoBuilder.WriteString(protoutils.MakeProtoDeletePath(deletePath.TargetName, deletePath.Path))
	}

	setTZRequest := &gpb.SetRequest{}

	if err := proto.UnmarshalText(protoBuilder.String(), setTZRequest); err != nil {
		return "", nil, err
	}

	setTZRequest.Extension = extensions
	setResult, err := c.(*gclient.Client).Set(ctx, setTZRequest)
	if err != nil {
		return "", nil, err
	}
	return extractSetTransactionID(setResult), setResult.Extension, nil
}

// GetTargetPath creates a target path
func GetTargetPath(target string, path string) []protoutils.TargetPath {
	return GetTargetPathWithValue(target, path, "", "")
}

// GetTargetPathWithValue creates a target path with a value to set
func GetTargetPathWithValue(target string, path string, value string, valueType string) []protoutils.TargetPath {
	targetPath := make([]protoutils.TargetPath, 1)
	targetPath[0].TargetName = target
	targetPath[0].Path = path
	targetPath[0].PathDataValue = value
	targetPath[0].PathDataType = valueType
	return targetPath
}

// GetTargetPaths creates multiple target paths
func GetTargetPaths(targets []string, paths []string) []protoutils.TargetPath {
	var targetPaths = make([]protoutils.TargetPath, len(paths)*len(targets))
	pathIndex := 0
	for _, dev := range targets {
		for _, path := range paths {
			targetPaths[pathIndex].TargetName = dev
			targetPaths[pathIndex].Path = path
			pathIndex++
		}
	}
	return targetPaths
}

// GetTargetPathsWithValues creates multiple target paths with values to set
func GetTargetPathsWithValues(targets []string, paths []string, values []string) []protoutils.TargetPath {
	var targetPaths = GetTargetPaths(targets, paths)
	valueIndex := 0
	for range targets {
		for _, value := range values {
			targetPaths[valueIndex].PathDataValue = value
			targetPaths[valueIndex].PathDataType = protoutils.StringVal
			valueIndex++
		}
	}
	return targetPaths
}

// CheckTargetValue makes sure a value has been assigned properly to a target path by querying GNMI
func CheckTargetValue(ctx context.Context, t *testing.T, targetGnmiClient gnmiclient.Impl, targetPaths []protoutils.TargetPath, expectedValue string) {
	targetValues, extensions, err := GetGNMIValue(ctx, targetGnmiClient, targetPaths, gpb.Encoding_JSON)
	if err == nil {
		assert.NoError(t, err, "GNMI get operation to target returned an error")
		assert.Equal(t, expectedValue, targetValues[0].PathDataValue, "Query after set returned the wrong value: %s\n", expectedValue)
		assert.Equal(t, 0, len(extensions))
	} else {
		assert.Fail(t, "Failed to query target: %v", err)
	}
}

// CheckTargetValueDeleted makes sure target path is missing when queried via GNMI
func CheckTargetValueDeleted(ctx context.Context, t *testing.T, targetGnmiClient gnmiclient.Impl, targetPaths []protoutils.TargetPath) {
	_, _, err := GetGNMIValue(ctx, targetGnmiClient, targetPaths, gpb.Encoding_JSON)
	if err == nil {
		assert.Fail(t, "Path not deleted", targetPaths)
	} else if !strings.Contains(err.Error(), "NotFound") {
		assert.Fail(t, "Incorrect error received", err)
	}
}

// GetTargetGNMIClientOrFail creates a GNMI client to a target. If there is an error, the test is failed
func GetTargetGNMIClientOrFail(t *testing.T, simulator *helm.HelmRelease) gnmiclient.Impl {
	t.Helper()
	simulatorClient := kubernetes.NewForReleaseOrDie(simulator)
	services, err := simulatorClient.CoreV1().Services().List(context.Background())
	assert.NoError(t, err)
	service := services[0]
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
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
func GetOnosConfigDestination() (gnmiclient.Destination, error) {
	creds, err := getClientCredentials()
	if err != nil {
		return gnmiclient.Destination{}, err
	}
	configRelease := helm.Release("onos-umbrella")
	configClient := kubernetes.NewForReleaseOrDie(configRelease)

	configService, err := configClient.CoreV1().Services().Get(context.Background(), "onos-config")
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

// GetGNMIClientWithContextOrFail makes a GNMI client to use for requests. If creating the client fails, the test is failed.
func GetGNMIClientWithContextOrFail(ctx context.Context, t *testing.T, retryOption RetryOption) gnmiclient.Impl {
	t.Helper()
	gCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	dest, err := GetOnosConfigDestination()
	if !assert.NoError(t, err) {
		t.Fail()
	}
	opts := make([]grpc.DialOption, 0)
	opts = append(opts, grpc.WithTransportCredentials(credentials.NewTLS(dest.TLS)))
	if retryOption == WithRetry {
		opts = append(opts, grpc.WithUnaryInterceptor(retry.RetryingUnaryClientInterceptor()))
	}

	conn, err := grpc.DialContext(gCtx, dest.Addrs[0], opts...)
	assert.NoError(t, err)
	client, err := gclient.NewFromConn(gCtx, conn, dest)
	assert.NoError(t, err)
	assert.True(t, client != nil, "Fetching target client returned nil")
	return client
}

// CheckGNMIValueWithContext makes sure a value has been assigned properly by querying the onos-config northbound API
func CheckGNMIValueWithContext(ctx context.Context, t *testing.T, gnmiClient gnmiclient.Impl, paths []protoutils.TargetPath, expectedValue string, expectedExtensions int, failMessage string) {
	t.Helper()
	value, extensions, err := GetGNMIValue(ctx, gnmiClient, paths, gpb.Encoding_PROTO)
	assert.NoError(t, err, "Get operation returned an unexpected error")
	assert.Equal(t, expectedExtensions, len(extensions))
	assert.Equal(t, expectedValue, value[0].PathDataValue, "%s: %s", failMessage, value)
}

// CheckGNMIValues makes sure a list of values has been assigned properly by querying the onos-config northbound API
func CheckGNMIValues(ctx context.Context, t *testing.T, gnmiClient gnmiclient.Impl, paths []protoutils.TargetPath, expectedValues []string, expectedExtensions int, failMessage string) {
	t.Helper()
	value, extensions, err := GetGNMIValue(ctx, gnmiClient, paths, gpb.Encoding_PROTO)
	assert.NoError(t, err, "Get operation returned unexpected error")
	assert.Equal(t, expectedExtensions, len(extensions))
	for index, expectedValue := range expectedValues {
		assert.Equal(t, expectedValue, value[index].PathDataValue, "%s: %s", failMessage, value)
	}
}

// SetGNMIValueWithContextOrFail does a GNMI set operation to the given client, and fails the test if there is an error
func SetGNMIValueWithContextOrFail(ctx context.Context, t *testing.T, gnmiClient gnmiclient.Impl,
	updatePaths []protoutils.TargetPath, deletePaths []protoutils.TargetPath,
	extensions []*gnmi_ext.Extension) (configapi.TransactionID, v2.Index) {
	t.Helper()
	transactionID, transactionIndex, err := SetGNMIValueWithContext(ctx, t, gnmiClient, updatePaths, deletePaths, extensions)
	assert.NoError(t, err, "Set operation returned unexpected error")

	return transactionID, transactionIndex
}

// SetGNMIValueWithContext does a GNMI set operation to the given client, and fails the test if there is an error
func SetGNMIValueWithContext(ctx context.Context, t *testing.T, gnmiClient gnmiclient.Impl,
	updatePaths []protoutils.TargetPath, deletePaths []protoutils.TargetPath,
	extensions []*gnmi_ext.Extension) (configapi.TransactionID, v2.Index, error) {
	t.Helper()
	_, extensionsSet, err := SetGNMIValue(ctx, gnmiClient, updatePaths, deletePaths, extensions)
	if err != nil {
		return "", 0, err
	}

	var transactionInfo *configapi.TransactionInfo
	for _, extension := range extensionsSet {
		if ext, ok := extension.Ext.(*gnmi_ext.Extension_RegisteredExt); ok &&
			ext.RegisteredExt.Id == configapi.TransactionInfoExtensionID {
			bytes := ext.RegisteredExt.Msg
			transactionInfo = &configapi.TransactionInfo{}
			err := proto.Unmarshal(bytes, transactionInfo)
			assert.NoError(t, err)
		}
	}

	if transactionInfo == nil {
		return "", 0, errors.NewNotFound("transaction ID extension not found")
	}

	return transactionInfo.ID, transactionInfo.Index, err
}

// MakeProtoPath returns a Path: element for a given target and Path
func MakeProtoPath(target string, path string) string {
	var protoBuilder strings.Builder

	protoBuilder.WriteString("path: ")
	gnmiPath := protoutils.MakeProtoTarget(target, path)
	protoBuilder.WriteString(gnmiPath)
	return protoBuilder.String()
}

// CreateSimulator creates a device simulator
func CreateSimulator(t *testing.T) *helm.HelmRelease {
	return CreateSimulatorWithName(t, random.NewPetName(2), true)
}

// CreateSimulatorWithName creates a device simulator
func CreateSimulatorWithName(t *testing.T, name string, createTopoEntity bool) *helm.HelmRelease {
	simulator := helm.
		Chart("device-simulator", onostest.OnosChartRepo).
		Release(name).
		Set("image.tag", "latest")
	err := simulator.Install(true)
	assert.NoError(t, err, "could not install device simulator %v", err)

	time.Sleep(2 * time.Second)

	if createTopoEntity {
		simulatorTarget, err := NewSimulatorTargetEntity(simulator, SimulatorTargetType, SimulatorTargetVersion)
		assert.NoError(t, err, "could not make target for simulator %v", err)

		err = AddTargetToTopo(simulatorTarget)
		assert.NoError(t, err, "could not add target to topo for simulator %v", err)
	}

	return simulator
}

// DeleteSimulator shuts down the simulator pod and removes the target from topology
func DeleteSimulator(t *testing.T, simulator *helm.HelmRelease) {
	assert.NoError(t, simulator.Uninstall())
}
