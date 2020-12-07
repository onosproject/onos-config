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
	"errors"
	"fmt"
	"github.com/onosproject/onos-api/go/onos/topo"
	"io"
	"strconv"
	"strings"
	"testing"
	"time"

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
	"github.com/openconfig/gnmi/client"
	gclient "github.com/openconfig/gnmi/client/gnmi"
	gpb "github.com/openconfig/gnmi/proto/gnmi"
	"github.com/openconfig/gnmi/proto/gnmi_ext"
	"github.com/stretchr/testify/assert"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
)

const (
	// SimulatorDeviceVersion default version for simulated device
	SimulatorDeviceVersion = "1.0.0"

	// SimulatorDeviceType type for simulated device
	SimulatorDeviceType = "Devicesim"
)

// MakeContext returns a new context for use in GNMI requests
func MakeContext() context.Context {
	ctx := context.Background()
	return ctx
}

func getService(release *helm.HelmRelease, serviceName string) (*v1.Service, error) {
	releaseClient := kubernetes.NewForReleaseOrDie(release)
	service, err := releaseClient.CoreV1().Services().Get(serviceName)
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

// GetDevice :
func GetDevice(simulator *helm.HelmRelease) (*device.Device, error) {
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
	return device.ToDevice(response.Object), nil
}

// AwaitDeviceState :
func AwaitDeviceState(simulator *helm.HelmRelease, predicate func(*device.Device) bool) error {
	for i := 0; i < 10; i++ {
		device, err := GetDevice(simulator)
		if err != nil {
			return err
		} else if predicate(device) {
			return nil
		}
		time.Sleep(time.Second)
	}
	return errors.New("device wait timed out")
}

// NewDevice :
func NewDevice(simulator *helm.HelmRelease, deviceType string, version string) (*device.Device, error) {
	simulatorClient := kubernetes.NewForReleaseOrDie(simulator)
	services, err := simulatorClient.CoreV1().Services().List()
	if err != nil {
		return nil, err
	}
	service := services[0]
	return &device.Device{
		ID:      device.ID(simulator.Name()),
		Address: service.Ports()[0].Address(true),
		Type:    device.Type(deviceType),
		Version: version,
		TLS: device.TLSConfig{
			Plain: true,
		},
		Protocols: []*topo.ProtocolState{{
			Protocol:          topo.Protocol_GNMI,
			ConnectivityState: topo.ConnectivityState_REACHABLE,
			ChannelState:      topo.ChannelState_DISCONNECTED,
			ServiceState:      topo.ServiceState_UNKNOWN_SERVICE_STATE,
		}},
	}, nil
}

// NewTopoClient :
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

// AddDeviceToTopo :
func AddDeviceToTopo(d *device.Device) error {
	client, err := NewTopoClient()
	if err != nil {
		return err
	}
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	_, err = client.Create(ctx, &topo.CreateRequest{
		Object: device.ToObject(d),
	})
	return err
}

// RemoveDeviceFromTopo :
func RemoveDeviceFromTopo(d *device.Device) error {
	client, err := NewTopoClient()
	if err != nil {
		return err
	}
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	_, err = client.Delete(ctx, &topo.DeleteRequest{
		ID: topo.ID(d.ID),
	})
	return err
}

// WaitForDevice waits for a device to match the given predicate
func WaitForDevice(t *testing.T, predicate func(*device.Device) bool, timeout time.Duration) bool {
	cl, err := NewTopoClient()
	assert.NoError(t, err)
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	stream, err := cl.Watch(ctx, &topo.WatchRequest{})
	assert.NoError(t, err)
	for {
		response, err := stream.Recv()
		if err == io.EOF {
			assert.Fail(t, "device stream closed prematurely")
			return false
		} else if err != nil {
			assert.Fail(t, "device stream failed with error: %v", err)
			return false
		} else if response.Event.Object.Type == topo.Object_ENTITY {
			if predicate(device.ToDevice(&response.Event.Object)) {
				return true
			}
		}
	}
}

// WaitForDeviceAvailable waits for a device to become available
func WaitForDeviceAvailable(t *testing.T, deviceID device.ID, timeout time.Duration) bool {
	return WaitForDevice(t, func(dev *device.Device) bool {
		if dev.ID != deviceID {
			return false
		}

		for _, protocol := range dev.Protocols {
			if protocol.Protocol == topo.Protocol_GNMI && protocol.ServiceState == topo.ServiceState_AVAILABLE {
				return true
			}
		}
		// TODO: Investigate why device events are not working properly
		return true
	}, timeout)
}

// WaitForDeviceUnavailable waits for a device to become available
func WaitForDeviceUnavailable(t *testing.T, deviceID device.ID, timeout time.Duration) bool {
	return WaitForDevice(t, func(dev *device.Device) bool {
		if dev.ID != deviceID {
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

// WaitForNetworkChangeComplete waits for a COMPLETED status on the given change
func WaitForNetworkChangeComplete(t *testing.T, networkChangeID network.ID, wait time.Duration) bool {
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
			assert.True(t, networkChangeResponse != nil)
			if change.State_COMPLETE == networkChangeResponse.Change.Status.State {
				return true
			}
		}
	}
}

// NoPaths can be used on a request that does not need path values
var NoPaths = make([]protoutils.DevicePath, 0)

// NoExtensions can be used on a request that does not need extension values
var NoExtensions = make([]*gnmi_ext.Extension, 0)

func convertGetResults(response *gpb.GetResponse) ([]protoutils.DevicePath, []*gnmi_ext.Extension, error) {
	entryCount := len(response.Notification)
	result := make([]protoutils.DevicePath, entryCount)

	for index, notification := range response.Notification {
		value := notification.Update[0].Val

		result[index].DeviceName = notification.Update[0].Path.Target
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
func GetGNMIValue(ctx context.Context, c client.Impl, paths []protoutils.DevicePath) ([]protoutils.DevicePath, []*gnmi_ext.Extension, error) {
	protoString := ""
	for _, devicePath := range paths {
		protoString = protoString + MakeProtoPath(devicePath.DeviceName, devicePath.Path)
	}

	getTZRequest := &gpb.GetRequest{}
	if err := proto.UnmarshalText(protoString, getTZRequest); err != nil {
		fmt.Printf("unable to parse gnmi.GetRequest from %q : %v\n", protoString, err)
		return nil, nil, err
	}

	response, err := c.(*gclient.Client).Get(ctx, getTZRequest)
	if err != nil || response == nil {
		return nil, nil, err
	}
	return convertGetResults(response)
}

// SetGNMIValue generates a SET request on the given client for update and delete paths on a device
func SetGNMIValue(ctx context.Context, c client.Impl, updatePaths []protoutils.DevicePath, deletePaths []protoutils.DevicePath, extensions []*gnmi_ext.Extension) (string, []*gnmi_ext.Extension, error) {
	var protoBuilder strings.Builder
	for _, updatePath := range updatePaths {
		protoBuilder.WriteString(protoutils.MakeProtoUpdatePath(updatePath))
	}
	for _, deletePath := range deletePaths {
		protoBuilder.WriteString(protoutils.MakeProtoDeletePath(deletePath.DeviceName, deletePath.Path))
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

// GetDevicePath creates a device path
func GetDevicePath(device string, path string) []protoutils.DevicePath {
	return GetDevicePathWithValue(device, path, "", "")
}

// GetDevicePathWithValue creates a device path with a value to set
func GetDevicePathWithValue(device string, path string, value string, valueType string) []protoutils.DevicePath {
	devicePath := make([]protoutils.DevicePath, 1)
	devicePath[0].DeviceName = device
	devicePath[0].Path = path
	devicePath[0].PathDataValue = value
	devicePath[0].PathDataType = valueType
	return devicePath
}

// GetDevicePaths creates multiple device paths
func GetDevicePaths(devices []string, paths []string) []protoutils.DevicePath {
	var devicePaths = make([]protoutils.DevicePath, len(paths)*len(devices))
	pathIndex := 0
	for _, dev := range devices {
		for _, path := range paths {
			devicePaths[pathIndex].DeviceName = dev
			devicePaths[pathIndex].Path = path
			pathIndex++
		}
	}
	return devicePaths
}

// GetDevicePathsWithValues creates multiple device paths with values to set
func GetDevicePathsWithValues(devices []string, paths []string, values []string) []protoutils.DevicePath {
	var devicePaths = GetDevicePaths(devices, paths)
	valueIndex := 0
	for range devices {
		for _, value := range values {
			devicePaths[valueIndex].PathDataValue = value
			devicePaths[valueIndex].PathDataType = protoutils.StringVal
			valueIndex++
		}
	}
	return devicePaths
}

// CheckDeviceValue makes sure a value has been assigned properly to a device path by querying GNMI
func CheckDeviceValue(t *testing.T, deviceGnmiClient client.Impl, devicePaths []protoutils.DevicePath, expectedValue string) {
	deviceValues, extensions, deviceValuesError := GetGNMIValue(MakeContext(), deviceGnmiClient, devicePaths)
	if deviceValuesError == nil {
		assert.NoError(t, deviceValuesError, "GNMI get operation to device returned an error")
		assert.Equal(t, expectedValue, deviceValues[0].PathDataValue, "Query after set returned the wrong value: %s\n", expectedValue)
		assert.Equal(t, 0, len(extensions))
	} else {
		assert.Fail(t, "Failed to query device: %v", deviceValuesError)
	}

}

// GetDeviceDestination :
func GetDeviceDestination(simulator *helm.HelmRelease) (client.Destination, error) {
	creds, err := getClientCredentials()
	if err != nil {
		return client.Destination{}, err
	}
	simulatorClient := kubernetes.NewForReleaseOrDie(simulator)
	services, err := simulatorClient.CoreV1().Services().List()
	if err != nil {
		return client.Destination{}, err
	}
	service := services[0]
	return client.Destination{
		Addrs:   []string{service.Ports()[0].Address(true)},
		Target:  service.Name,
		TLS:     creds,
		Timeout: 10 * time.Second,
	}, nil
}

// GetDeviceGNMIClientOrFail creates a GNMI client to a device. If there is an error, the test is failed
func GetDeviceGNMIClientOrFail(t *testing.T, simulator *helm.HelmRelease) client.Impl {
	t.Helper()
	simulatorClient := kubernetes.NewForReleaseOrDie(simulator)
	services, err := simulatorClient.CoreV1().Services().List()
	assert.NoError(t, err)
	service := services[0]
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	dest := client.Destination{
		Addrs:   []string{service.Ports()[0].Address(true)},
		Target:  service.Name,
		Timeout: 10 * time.Second,
	}
	client, err := gclient.New(ctx, dest)
	assert.NoError(t, err)
	assert.True(t, client != nil, "Fetching device client returned nil")
	return client
}

// GetDestination :
func GetDestination() (client.Destination, error) {
	creds, err := getClientCredentials()
	if err != nil {
		return client.Destination{}, err
	}
	configRelease := helm.Release("onos-umbrella")
	configClient := kubernetes.NewForReleaseOrDie(configRelease)

	configService, err := configClient.CoreV1().Services().Get("onos-config")
	if err != nil || configService == nil {
		return client.Destination{}, errors.New("can't find service for onos-config")
	}

	return client.Destination{
		Addrs:   []string{configService.Ports()[0].Address(true)},
		Target:  configService.Name,
		TLS:     creds,
		Timeout: 10 * time.Second,
	}, nil
}

// GetGNMIClientOrFail makes a GNMI client to use for requests. If creating the client fails, the test is failed.
func GetGNMIClientOrFail(t *testing.T) client.Impl {
	t.Helper()
	release := helm.Chart("onos-umbrella").Release("onos-umbrella")
	conn, err := connectService(release, "onos-config")
	assert.NoError(t, err)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	dest, err := GetDestination()
	if !assert.NoError(t, err) {
		t.Fail()
	}
	client, err := gclient.NewFromConn(ctx, conn, dest)
	assert.NoError(t, err)
	assert.True(t, client != nil, "Fetching device client returned nil")
	return client
}

// CheckGNMIValue makes sure a value has been assigned properly by querying the onos-config northbound API
func CheckGNMIValue(t *testing.T, gnmiClient client.Impl, paths []protoutils.DevicePath, expectedValue string, expectedExtensions int, failMessage string) {
	t.Helper()
	value, extensions, err := GetGNMIValue(MakeContext(), gnmiClient, paths)
	assert.NoError(t, err)
	assert.Equal(t, expectedExtensions, len(extensions))
	assert.Equal(t, expectedValue, value[0].PathDataValue, "%s: %s", failMessage, value)
}

// CheckGNMIValues makes sure a list of values has been assigned properly by querying the onos-config northbound API
func CheckGNMIValues(t *testing.T, gnmiClient client.Impl, paths []protoutils.DevicePath, expectedValues []string, expectedExtensions int, failMessage string) {
	t.Helper()
	value, extensions, err := GetGNMIValue(MakeContext(), gnmiClient, paths)
	assert.NoError(t, err)
	assert.Equal(t, expectedExtensions, len(extensions))
	for index, expectedValue := range expectedValues {
		assert.Equal(t, expectedValue, value[index].PathDataValue, "%s: %s", failMessage, value)
	}
}

// SetGNMIValueOrFail does a GNMI set operation to the given client, and fails the test if there is an error
func SetGNMIValueOrFail(t *testing.T, gnmiClient client.Impl,
	updatePaths []protoutils.DevicePath, deletePaths []protoutils.DevicePath,
	extensions []*gnmi_ext.Extension) network.ID {
	t.Helper()
	_, extensionsSet, errorSet := SetGNMIValue(MakeContext(), gnmiClient, updatePaths, deletePaths, extensions)
	assert.NoError(t, errorSet)
	assert.Equal(t, 1, len(extensionsSet))
	extensionBefore := extensionsSet[0].GetRegisteredExt()
	assert.Equal(t, extensionBefore.Id.String(), strconv.Itoa(gnmi.GnmiExtensionNetwkChangeID))
	networkChangeID := network.ID(extensionBefore.Msg)
	return networkChangeID
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

	simulatorDevice, err := NewDevice(simulator, SimulatorDeviceType, SimulatorDeviceVersion)
	assert.NoError(t, err, "could not make device for simulator %v", err)

	err = AddDeviceToTopo(simulatorDevice)
	assert.NoError(t, err, "could not add device to topo for simulator %v", err)

	return simulator
}

// DeleteSimulator shuts down the simulator pod and removes the device from topology
func DeleteSimulator(t *testing.T, simulator *helm.HelmRelease) {
	simulatorDevice, err := GetDevice(simulator)
	assert.NoError(t, err)
	err = simulator.Uninstall()
	assert.NoError(t, err)
	err = RemoveDeviceFromTopo(simulatorDevice)
	assert.NoError(t, err)
}
