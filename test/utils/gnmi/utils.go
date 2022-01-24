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
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/onosproject/onos-api/go/onos/topo"
	"github.com/onosproject/onos-lib-go/pkg/errors"

	"github.com/onosproject/onos-test/pkg/onostest"

	"github.com/golang/protobuf/proto"
	"github.com/onosproject/helmit/pkg/helm"
	"github.com/onosproject/helmit/pkg/kubernetes"
	v1 "github.com/onosproject/helmit/pkg/kubernetes/core/v1"
	"github.com/onosproject/helmit/pkg/util/random"
	"github.com/onosproject/onos-api/go/onos/config/admin"
	"github.com/onosproject/onos-api/go/onos/config/change"
	"github.com/onosproject/onos-api/go/onos/config/change/network"
	"github.com/onosproject/onos-api/go/onos/config/diags"
	"github.com/onosproject/onos-config/pkg/device"
	"github.com/onosproject/onos-config/pkg/northbound/gnmi"
	"github.com/onosproject/onos-config/pkg/utils"
	protoutils "github.com/onosproject/onos-config/test/utils/proto"
	gnmiclient "github.com/openconfig/gnmi/client"
	gclient "github.com/openconfig/gnmi/client/gnmi"
	gpb "github.com/openconfig/gnmi/proto/gnmi"
	"github.com/openconfig/gnmi/proto/gnmi_ext"
	"github.com/stretchr/testify/assert"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
)

const (
	// SimulatorTargetVersion default version for simulated device
	SimulatorTargetVersion = "1.0.0"

	// SimulatorTargetType type for simulated device
	SimulatorTargetType = "Devicesim"
)

// MakeContext returns a new context for use in GNMI requests
func MakeContext() context.Context {
	ctx := context.Background()
	return ctx
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
	response, err := client.Get(ctx, &topo.GetRequest{
		ID: topo.ID(simulator.Name()),
	})
	if err != nil {
		return nil, err
	}
	return response.Object, nil
}

// NewSimulatorTargetEntity creates a topo entity for a device simulator target
func NewSimulatorTargetEntity(simulator *helm.HelmRelease, targetType string, targetVersion string) (*topo.Object, error) {
	simulatorClient := kubernetes.NewForReleaseOrDie(simulator)
	services, err := simulatorClient.CoreV1().Services().List(context.Background())
	if err != nil {
		return nil, err
	}
	service := services[0]

	o := topo.Object{
		ID:   topo.ID(simulator.Name()),
		Type: topo.Object_ENTITY,
		Obj: &topo.Object_Entity{
			Entity: &topo.Entity{
				KindID: topo.ID(targetType),
			},
		},
	}

	err = o.SetAspect(&topo.TLSOptions{
		Insecure: true,
		Plain:    true,
	})
	if err != nil {
		return nil, err
	}

	err = o.SetAspect(&topo.Asset{
		Name: simulator.Name(),
	})
	if err != nil {
		return nil, err
	}

	if err := o.SetAspect(&topo.MastershipState{}); err != nil {
		return nil, err
	}

	err = o.SetAspect(&topo.Configurable{
		Type:    targetType,
		Address: service.Ports()[0].Address(true),
		Version: targetVersion,
		Timeout: uint64(time.Second * 30),
	})
	if err != nil {
		return nil, err
	}

	protocolState := &topo.ProtocolState{
		Protocol:          topo.Protocol_GNMI,
		ConnectivityState: topo.ConnectivityState_REACHABLE,
		ChannelState:      topo.ChannelState_CONNECTED,
		ServiceState:      topo.ServiceState_AVAILABLE,
	}
	protocolStates := []*topo.ProtocolState{protocolState}
	err = o.SetAspect(&topo.Protocols{State: protocolStates})
	if err != nil {
		return nil, err
	}

	return &o, nil
}

// NewTopoClient creates a topology client
func NewTopoClient() (topo.TopoClient, error) {
	conn, err := connectComponent("onos-umbrella", "onos-topo")
	if err != nil {
		return nil, err
	}
	return topo.NewTopoClient(conn), nil
}

// NewChangeServiceClient :
func NewChangeServiceClient() (diags.ChangeServiceClient, error) {
	conn, err := connectComponent("onos-umbrella", "onos-config")
	if err != nil {
		return nil, err
	}
	return diags.NewChangeServiceClient(conn), nil
}

// NewAdminServiceClient :
func NewAdminServiceClient() (admin.ConfigAdminServiceClient, error) {
	conn, err := connectComponent("onos-umbrella", "onos-config")
	if err != nil {
		return nil, err
	}
	return admin.NewConfigAdminServiceClient(conn), nil
}

// NewOpStateDiagsClient :
func NewOpStateDiagsClient() (diags.OpStateDiagsClient, error) {
	conn, err := connectComponent("onos-umbrella", "onos-config")
	if err != nil {
		return nil, err
	}
	return diags.NewOpStateDiagsClient(conn), nil
}

// AddTargetToTopo adds a new target to topo
func AddTargetToTopo(targetEntity *topo.Object) error {
	client, err := NewTopoClient()
	if err != nil {
		return err
	}
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	_, err = client.Create(ctx, &topo.CreateRequest{
		Object: targetEntity,
	})
	return err
}

// RemoveTargetFromTopo removes a target from topo
func RemoveTargetFromTopo(d *topo.Object) error {
	client, err := NewTopoClient()
	if err != nil {
		return err
	}
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	_, err = client.Delete(ctx, &topo.DeleteRequest{
		ID: d.ID,
	})
	return err
}

// WaitForTarget waits for a target to match the given predicate
func WaitForTarget(t *testing.T, predicate func(*device.Device, topo.EventType) bool, timeout time.Duration) bool {
	cl, err := NewTopoClient()
	assert.NoError(t, err)
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	stream, err := cl.Watch(ctx, &topo.WatchRequest{})
	assert.NoError(t, err)
	for {
		response, err := stream.Recv() // Wait here for topo events
		if err == io.EOF {
			assert.Fail(t, "topo stream closed prematurely")
			return false
		} else if err != nil {
			assert.Fail(t, fmt.Sprintf("topo stream failed with error %s", err.Error()))
			return false
		} else if response.Event.Object.Type == topo.Object_ENTITY {
			err = response.Event.Object.GetAspect(&topo.Configurable{})
			if err == nil {
				topoDevice, err := device.ToDevice(&response.Event.Object)
				assert.Nil(t, err, "error converting entity to topo Device")
				if predicate(topoDevice, response.Event.GetType()) {
					return true
				} // Otherwise loop and wait for the next topo event
			}
		}
	}
}

// WaitForTargetAvailable waits for a target to become available
func WaitForTargetAvailable(t *testing.T, deviceID device.ID, timeout time.Duration) bool {
	return WaitForTarget(t, func(dev *device.Device, eventType topo.EventType) bool {
		if dev.ID != deviceID {
			fmt.Printf("Topo %s event from %s (expected %s). Discarding\n", eventType, dev.ID, deviceID)
			return false
		}

		for _, protocol := range dev.Protocols {
			if protocol.Protocol == topo.Protocol_GNMI && protocol.ServiceState == topo.ServiceState_AVAILABLE {
				//fmt.Printf("Topo %s on %s is AVAILABLE. Has: %v\n", eventType, dev.ID, protocol)
				return true
			}
		}
		//fmt.Printf("Topo %s on %s not AVAILABLE. Protocols: %v\n", eventType, dev.ID, dev.Protocols)
		return false
	}, timeout)
}

// WaitForTargetUnavailable waits for a target to become available
func WaitForTargetUnavailable(t *testing.T, deviceID device.ID, timeout time.Duration) bool {
	return WaitForTarget(t, func(dev *device.Device, eventType topo.EventType) bool {
		if dev.ID != deviceID {
			fmt.Printf("Topo %s event from %s (expected %s). Discarding\n", eventType, dev.ID, deviceID)
			return false
		}

		for _, protocol := range dev.Protocols {
			if protocol.Protocol == topo.Protocol_GNMI && protocol.ServiceState == topo.ServiceState_UNAVAILABLE {
				return true
			}
		}
		return false
	}, timeout)
}

// WaitForTransactionComplete waits for a COMPLETED status on the given change
func WaitForTransactionComplete(t *testing.T, networkChangeID network.ID, wait time.Duration) bool {
	listNetworkChangeRequest := &diags.ListNetworkChangeRequest{
		Subscribe:     true,
		ChangeID:      networkChangeID,
		WithoutReplay: false,
	}

	client, err := NewChangeServiceClient()
	assert.NoError(t, err)

	ctx, cancel := context.WithTimeout(context.Background(), wait)
	defer cancel()

	listNetworkChangesClient, listNetworkChangesClientErr := client.ListNetworkChanges(ctx, listNetworkChangeRequest)
	assert.Nil(t, listNetworkChangesClientErr)
	assert.True(t, listNetworkChangesClient != nil)

	for {
		// Check if the network change has completed
		networkChangeResponse, networkChangeResponseErr := listNetworkChangesClient.Recv()
		if networkChangeResponseErr == io.EOF {
			assert.Fail(t, "change stream closed prematurely")
			return false
		} else if networkChangeResponseErr != nil {
			assert.Fail(t, fmt.Sprintf("change stream failed with error: %v", networkChangeResponseErr))
			return false
		} else {
			assert.NotNil(t, networkChangeResponse)
			if change.State_COMPLETE == networkChangeResponse.Change.Status.State {
				return true
			}
		}
	}
}

// NoPaths can be used on a request that does not need path values
var NoPaths = make([]protoutils.TargetPath, 0)

// NoExtensions can be used on a request that does not need extension values
var NoExtensions = make([]*gnmi_ext.Extension, 0)

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

// GetGNMIValue generates a GET request on the given client for a Path on a device
func GetGNMIValue(ctx context.Context, c gnmiclient.Impl, paths []protoutils.TargetPath, encoding gpb.Encoding) ([]protoutils.TargetPath, []*gnmi_ext.Extension, error) {
	protoString := ""
	for _, devicePath := range paths {
		protoString = protoString + MakeProtoPath(devicePath.TargetName, devicePath.Path)
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

// SetGNMIValue generates a SET request on the given client for update and delete paths on a device
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
	setResult, setError := c.(*gclient.Client).Set(ctx, setTZRequest)
	if setError != nil {
		return "", nil, setError
	}
	return extractSetTransactionID(setResult), setResult.Extension, nil
}

// GetTargetPath creates a device path
func GetTargetPath(target string, path string) []protoutils.TargetPath {
	return GetTargetPathWithValue(target, path, "", "")
}

// GetTargetPathWithValue creates a device path with a value to set
func GetTargetPathWithValue(target string, path string, value string, valueType string) []protoutils.TargetPath {
	targetPath := make([]protoutils.TargetPath, 1)
	targetPath[0].TargetName = target
	targetPath[0].Path = path
	targetPath[0].PathDataValue = value
	targetPath[0].PathDataType = valueType
	return targetPath
}

// GetTargetPaths creates multiple device paths
func GetTargetPaths(devices []string, paths []string) []protoutils.TargetPath {
	var devicePaths = make([]protoutils.TargetPath, len(paths)*len(devices))
	pathIndex := 0
	for _, dev := range devices {
		for _, path := range paths {
			devicePaths[pathIndex].TargetName = dev
			devicePaths[pathIndex].Path = path
			pathIndex++
		}
	}
	return devicePaths
}

// GetTargetPathsWithValues creates multiple device paths with values to set
func GetTargetPathsWithValues(devices []string, paths []string, values []string) []protoutils.TargetPath {
	var targetPaths = GetTargetPaths(devices, paths)
	valueIndex := 0
	for range devices {
		for _, value := range values {
			targetPaths[valueIndex].PathDataValue = value
			targetPaths[valueIndex].PathDataType = protoutils.StringVal
			valueIndex++
		}
	}
	return targetPaths
}

// CheckTargetValue makes sure a value has been assigned properly to a device path by querying GNMI
func CheckTargetValue(t *testing.T, targetGnmiClient gnmiclient.Impl, targetPaths []protoutils.TargetPath, expectedValue string) {
	targetValues, extensions, err := GetGNMIValue(MakeContext(), targetGnmiClient, targetPaths, gpb.Encoding_JSON)
	if err == nil {
		assert.NoError(t, err, "GNMI get operation to device returned an error")
		assert.Equal(t, expectedValue, targetValues[0].PathDataValue, "Query after set returned the wrong value: %s\n", expectedValue)
		assert.Equal(t, 0, len(extensions))
	} else {
		assert.Fail(t, "Failed to query device: %v", err)
	}

}

// GetTargetGNMIClientOrFail creates a GNMI client to a device. If there is an error, the test is failed
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
	assert.True(t, client != nil, "Fetching device client returned nil")
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
func GetGNMIClientWithContextOrFail(ctx context.Context, t *testing.T) gnmiclient.Impl {
	t.Helper()
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	dest, err := GetOnosConfigDestination()
	if !assert.NoError(t, err) {
		t.Fail()
	}
	client, err := gclient.New(ctx, dest)
	assert.NoError(t, err)
	assert.True(t, client != nil, "Fetching device client returned nil")
	return client
}

// GetGNMIClientOrFail makes a GNMI client to use for requests. If creating the client fails, the test is failed.
func GetGNMIClientOrFail(t *testing.T) gnmiclient.Impl {
	t.Helper()
	return GetGNMIClientWithContextOrFail(context.Background(), t)
}

// CheckGNMIValueWithContext makes sure a value has been assigned properly by querying the onos-config northbound API
func CheckGNMIValueWithContext(ctx context.Context, t *testing.T, gnmiClient gnmiclient.Impl, paths []protoutils.TargetPath, expectedValue string, expectedExtensions int, failMessage string) {
	t.Helper()
	value, extensions, err := GetGNMIValue(ctx, gnmiClient, paths, gpb.Encoding_PROTO)
	assert.NoError(t, err, "Get operation returned an unexpected error")
	assert.Equal(t, expectedExtensions, len(extensions))
	assert.Equal(t, expectedValue, value[0].PathDataValue, "%s: %s", failMessage, value)
}

// CheckGNMIValue makes sure a value has been assigned properly by querying the onos-config northbound API
func CheckGNMIValue(t *testing.T, gnmiClient gnmiclient.Impl, paths []protoutils.TargetPath, expectedValue string, expectedExtensions int, failMessage string) {
	t.Helper()
	CheckGNMIValueWithContext(MakeContext(), t, gnmiClient, paths, expectedValue, expectedExtensions, failMessage)
}

// CheckGNMIValues makes sure a list of values has been assigned properly by querying the onos-config northbound API
func CheckGNMIValues(t *testing.T, gnmiClient gnmiclient.Impl, paths []protoutils.TargetPath, expectedValues []string, expectedExtensions int, failMessage string) {
	t.Helper()
	value, extensions, err := GetGNMIValue(MakeContext(), gnmiClient, paths, gpb.Encoding_PROTO)
	assert.NoError(t, err, "Get operation returned unexpected error")
	assert.Equal(t, expectedExtensions, len(extensions))
	for index, expectedValue := range expectedValues {
		assert.Equal(t, expectedValue, value[index].PathDataValue, "%s: %s", failMessage, value)
	}
}

// SetGNMIValueWithContextOrFail does a GNMI set operation to the given client, and fails the test if there is an error
func SetGNMIValueWithContextOrFail(ctx context.Context, t *testing.T, gnmiClient gnmiclient.Impl,
	updatePaths []protoutils.TargetPath, deletePaths []protoutils.TargetPath,
	extensions []*gnmi_ext.Extension) network.ID {
	t.Helper()
	networkChangeID, errorSet := SetGNMIValueWithContext(ctx, t, gnmiClient, updatePaths, deletePaths, extensions)
	assert.NoError(t, errorSet, "Set operation returned unexpected error")

	return networkChangeID
}

// SetGNMIValueWithContext does a GNMI set operation to the given client, and fails the test if there is an error
func SetGNMIValueWithContext(ctx context.Context, t *testing.T, gnmiClient gnmiclient.Impl,
	updatePaths []protoutils.TargetPath, deletePaths []protoutils.TargetPath,
	extensions []*gnmi_ext.Extension) (network.ID, error) {
	t.Helper()
	_, extensionsSet, errorSet := SetGNMIValue(ctx, gnmiClient, updatePaths, deletePaths, extensions)
	if errorSet != nil {
		return "", errorSet
	}
	if len(extensionsSet) != 1 {
		return "", errors.NewNotFound("extension set not found")
	}
	extensionBefore := extensionsSet[0].GetRegisteredExt()
	if extensionBefore.Id.String() != strconv.Itoa(gnmi.GnmiExtensionNetwkChangeID) {
		return "", errors.NewNotFound("network change ID extension not found")
	}

	networkChangeID := network.ID(extensionBefore.Msg)
	return networkChangeID, errorSet
}

// SetGNMIValueOrFail does a GNMI set operation to the given client, and fails the test if there is an error
func SetGNMIValueOrFail(t *testing.T, gnmiClient gnmiclient.Impl,
	updatePaths []protoutils.TargetPath, deletePaths []protoutils.TargetPath,
	extensions []*gnmi_ext.Extension) network.ID {
	return SetGNMIValueWithContextOrFail(MakeContext(), t, gnmiClient, updatePaths, deletePaths, extensions)
}

// GetSimulatorExtensions creates the default set of extensions for a simulated device
func GetSimulatorExtensions() []*gnmi_ext.Extension {
	const (
		deviceVersion = "1.0.0"
		deviceType    = "Devicesim"
	)

	extDeviceType := gnmi_ext.Extension_RegisteredExt{
		RegisteredExt: &gnmi_ext.RegisteredExtension{
			Id:  gnmi.GnmiExtensionDeviceType,
			Msg: []byte(deviceType),
		},
	}
	extDeviceVersion := gnmi_ext.Extension_RegisteredExt{
		RegisteredExt: &gnmi_ext.RegisteredExtension{
			Id:  gnmi.GnmiExtensionVersion,
			Msg: []byte(deviceVersion),
		},
	}
	return []*gnmi_ext.Extension{{Ext: &extDeviceType}, {Ext: &extDeviceVersion}}
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
	return CreateSimulatorWithName(t, random.NewPetName(2))
}

// CreateSimulatorWithName creates a device simulator
func CreateSimulatorWithName(t *testing.T, name string) *helm.HelmRelease {
	simulator := helm.
		Chart("device-simulator", onostest.OnosChartRepo).
		Release(name).
		Set("image.tag", "latest")
	err := simulator.Install(true)
	assert.NoError(t, err, "could not install device simulator %v", err)

	time.Sleep(2 * time.Second)

	simulatorDevice, err := NewSimulatorTargetEntity(simulator, SimulatorTargetType, SimulatorTargetVersion)
	assert.NoError(t, err, "could not make device for simulator %v", err)

	err = AddTargetToTopo(simulatorDevice)
	assert.NoError(t, err, "could not add device to topo for simulator %v", err)

	return simulator
}

// DeleteSimulator shuts down the simulator pod and removes the device from topology
func DeleteSimulator(t *testing.T, simulator *helm.HelmRelease) {
	simulatorDevice, err := GetSimulatorTarget(simulator)
	assert.NoError(t, err)
	err = simulator.Uninstall()
	assert.NoError(t, err)
	err = RemoveTargetFromTopo(simulatorDevice)
	assert.NoError(t, err)
}
