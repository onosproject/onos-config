// SPDX-FileCopyrightText: 2020-present Open Networking Foundation <info@opennetworking.org>
//
// SPDX-License-Identifier: Apache-2.0

package gnmi

import (
	"context"
	"strconv"
	"testing"
	"time"

	"github.com/onosproject/onos-config/pkg/utils"

	"google.golang.org/grpc"

	gpb "github.com/openconfig/gnmi/proto/gnmi"

	topoapi "github.com/onosproject/onos-api/go/onos/topo"
	"github.com/onosproject/onos-lib-go/pkg/northbound"
	"github.com/stretchr/testify/assert"
)

const (
	target1       = "device-1"
	target2       = "device-2"
	targetDisplay = "Device 1"
	targetKind    = "devicesim"
	targetType    = "Devicesim"
	targetAddress = targetHost + ":" + targetPort
	targetHost    = "localhost"
	targetPort    = "9339"
	targetVersion = "1.0.0"
	timeout       = 5 * time.Second
)

func newTestServer() *testServer {
	return &testServer{}
}

type testServer struct {
	northbound.Service
}

func (s testServer) Capabilities(ctx context.Context, request *gpb.CapabilityRequest) (*gpb.CapabilityResponse, error) {
	log.Infof("Capabilities request is received: %+v", request)
	model1 := gpb.ModelData{
		Name:         "model 1",
		Organization: "ONF",
		Version:      "1.1",
	}
	models := make([]*gpb.ModelData, 1)
	models[0] = &model1

	encodings := make([]gpb.Encoding, 1)
	var encoding1 = gpb.Encoding_ASCII
	encodings[0] = encoding1

	response := gpb.CapabilityResponse{
		SupportedModels:    models,
		GNMIVersion:        "0.7.0",
		SupportedEncodings: encodings,
	}
	return &response, nil
}

func (s testServer) Get(ctx context.Context, request *gpb.GetRequest) (*gpb.GetResponse, error) {
	log.Infof("Get request is received: %+v", request)
	response := gpb.GetResponse{}
	response.Notification = make([]*gpb.Notification, len(request.Path))

	for notificationIndex := range response.Notification {
		response.Notification[notificationIndex] = &gpb.Notification{}
		response.Notification[notificationIndex].Update = make([]*gpb.Update, 1)
		update := &gpb.Update{}
		update.Path = request.Path[notificationIndex]
		update.Val = &gpb.TypedValue{}
		update.Val.Value = &gpb.TypedValue_StringVal{StringVal: strconv.Itoa(notificationIndex)}
		response.Notification[notificationIndex].Update[notificationIndex] = update
	}

	return &response, nil
}

func (s testServer) Set(ctx context.Context, request *gpb.SetRequest) (*gpb.SetResponse, error) {
	log.Infof("Set request is received: %+v", request)
	setUpdateResult := make([]*gpb.UpdateResult, 1)
	setUpdateResult[0] = &gpb.UpdateResult{Op: gpb.UpdateResult_DELETE}
	setResponse := gpb.SetResponse{Response: setUpdateResult}
	return &setResponse, nil
}

func (s testServer) Subscribe(server gpb.GNMI_SubscribeServer) error {
	return nil
}

// Register registers the Service with the gRPC server.
func (s testServer) Register(r *grpc.Server) {
	testServer := &testServer{}
	gpb.RegisterGNMIServer(r, testServer)

}

func createTestTarget(t *testing.T, targetID string, insecure bool) *topoapi.Object {
	target := &topoapi.Object{
		ID:   topoapi.ID(targetID),
		Type: topoapi.Object_ENTITY,
		Obj: &topoapi.Object_Entity{
			Entity: &topoapi.Entity{
				KindID: topoapi.ID(targetKind),
			},
		},
	}
	tlsOptions := &topoapi.TLSOptions{}

	if insecure {
		tlsOptions.Insecure = insecure
	}
	err := target.SetAspect(tlsOptions)
	assert.NoError(t, err)

	err = target.SetAspect(&topoapi.Asset{
		Name: targetDisplay,
	})
	assert.NoError(t, err)

	err = target.SetAspect(&topoapi.MastershipState{})
	assert.NoError(t, err)

	timeout := time.Second * 30
	err = target.SetAspect(&topoapi.Configurable{
		Type:    targetType,
		Address: targetAddress,
		Version: targetVersion,
		Timeout: &timeout,
	})
	assert.NoError(t, err)

	err = target.SetAspect(&topoapi.Protocols{})
	assert.NoError(t, err)

	return target
}

func setup(t *testing.T, serverCfg *northbound.ServerConfig) *northbound.Server {
	s := northbound.NewServer(serverCfg)
	s.AddService(newTestServer())
	doneCh := make(chan error)
	go func() {
		_ = s.Serve(func(started string) {
			t.Log("Started NBI on ", started)
			close(doneCh)
		})
	}()
	<-doneCh
	return s
}

func getTLSServerConfig(t *testing.T) *northbound.ServerConfig {
	port, err := strconv.ParseInt(targetPort, 10, 16)
	assert.NoError(t, err)
	return northbound.NewServerCfg(
		"",
		"",
		"",
		int16(port),
		false,
		northbound.SecurityConfig{})
}

func getNonTLSServerConfig(t *testing.T) *northbound.ServerConfig {
	port, err := strconv.ParseInt(targetPort, 10, 16)
	assert.NoError(t, err)
	return northbound.NewServerCfg(
		"",
		"",
		"",
		int16(port),
		true,
		northbound.SecurityConfig{})
}

func TestGNMIConn_CapabilitiesWithString(t *testing.T) {
	s := setup(t, getTLSServerConfig(t))

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	target1 := createTestTarget(t, target1, true)

	destination, err := newDestination(target1)
	assert.NoError(t, err)
	conn, _, err := connect(ctx, *destination)
	assert.NoError(t, err)

	capabilityResponse, capabilityErr := conn.CapabilitiesWithString(ctx, "")

	assert.NoError(t, capabilityErr)
	assert.NotNil(t, capabilityResponse)
	assert.Equal(t, capabilityResponse.SupportedEncodings[0], gpb.Encoding_ASCII)
	assert.Equal(t, capabilityResponse.GNMIVersion, "0.7.0")
	assert.Equal(t, capabilityResponse.SupportedModels[0].Organization, "ONF")
	s.Stop()
}

func TestGNMIConn_GetWithString(t *testing.T) {
	s := setup(t, getTLSServerConfig(t))

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	target1 := createTestTarget(t, target1, true)

	destination, err := newDestination(target1)
	assert.NoError(t, err)
	conn, _, err := connect(ctx, *destination)
	assert.NoError(t, err)

	request := "path: <elem: <name: 'system'> elem:<name:'config'> elem: <name: 'hostname'>>"
	getResponse, getErr := conn.GetWithString(ctx, request)

	assert.NoError(t, getErr)
	assert.NotNil(t, getResponse)
	assert.Equal(t, getResponse.Notification[0].Update[0].Path.Elem[0].Name, "system")
	value := utils.StrVal(getResponse.Notification[0].Update[0].Val)
	assert.Equal(t, value, "0")
	s.Stop()
}

func TestGNMIConn_Get(t *testing.T) {
	s := setup(t, getTLSServerConfig(t))

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	target1 := createTestTarget(t, target1, true)

	destination, err := newDestination(target1)
	assert.NoError(t, err)
	conn, _, err := connect(ctx, *destination)
	assert.NoError(t, err)
	assert.NotNil(t, conn)
	assert.NoError(t, err)
	allDevicesPath := gpb.Path{Elem: make([]*gpb.PathElem, 0), Target: "*"}

	request := gpb.GetRequest{
		Path: []*gpb.Path{&allDevicesPath},
	}

	resp, err := conn.Get(ctx, &request)
	assert.NoError(t, err)
	assert.NotNil(t, resp)
	s.Stop()
}

func TestGNMIConn_SetWithString(t *testing.T) {
	s := setup(t, getTLSServerConfig(t))

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	target1 := createTestTarget(t, target1, true)

	destination, err := newDestination(target1)
	assert.NoError(t, err)
	conn, _, err := connect(ctx, *destination)
	assert.NoError(t, err)
	assert.NotNil(t, conn)
	assert.NoError(t, err)

	request := "delete: <elem: <name: 'system'> elem:<name:'config'> elem: <name: 'hostname'>>"
	setResponse, setErr := conn.SetWithString(ctx, request)

	assert.NoError(t, setErr)
	assert.NotNil(t, setResponse)
	assert.Equal(t, setResponse.Response[0].Op, gpb.UpdateResult_DELETE)
	s.Stop()
}

func TestGNMIConn_Set(t *testing.T) {
	s := setup(t, getTLSServerConfig(t))

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	target1 := createTestTarget(t, target1, true)

	destination, err := newDestination(target1)
	assert.NoError(t, err)
	conn, _, err := connect(ctx, *destination)
	assert.NoError(t, err)
	assert.NotNil(t, conn)
	assert.NoError(t, err)
	request := &gpb.SetRequest{}

	setResponse, err := conn.Set(ctx, request)
	assert.NoError(t, err)
	assert.NotNil(t, setResponse)
	s.Stop()

}

func TestGNMIConn_Capabilities(t *testing.T) {
	s := setup(t, getTLSServerConfig(t))

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	target1 := createTestTarget(t, target1, true)

	destination, err := newDestination(target1)
	assert.NoError(t, err)
	conn, _, err := connect(ctx, *destination)
	assert.NoError(t, err)
	assert.NotNil(t, conn)
	assert.NoError(t, err)
	capResponse, err := conn.Capabilities(ctx, &gpb.CapabilityRequest{})
	assert.NoError(t, err)
	assert.Equal(t, capResponse.GNMIVersion, "0.7.0")
	s.Stop()
}

func TestConnManager_List(t *testing.T) {
	s := setup(t, getTLSServerConfig(t))

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	target1 := createTestTarget(t, target1, true)
	target2 := createTestTarget(t, target2, true)

	destination1, err := newDestination(target1)
	assert.NoError(t, err)
	conn1, _, err := connect(ctx, *destination1)
	assert.NoError(t, err)
	assert.NotNil(t, conn1)

	destination2, err := newDestination(target2)
	assert.NoError(t, err)
	conn2, _, err := connect(ctx, *destination2)
	assert.NoError(t, err)
	assert.NotNil(t, conn2)
	s.Stop()
}

func TestGNMICon_GetBadRequest(t *testing.T) {
	s := setup(t, getTLSServerConfig(t))

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	target1 := createTestTarget(t, target1, true)

	destination, err := newDestination(target1)
	assert.NoError(t, err)
	conn, _, err := connect(ctx, *destination)
	assert.NoError(t, err)
	assert.NotNil(t, conn)
	assert.NoError(t, err)

	requestBadParse := "!!!path: <elem: <name: 'system'> elem:<name:'config'> elem: <name: 'hostname'>>"
	_, getParseErr := conn.GetWithString(ctx, requestBadParse)
	assert.Error(t, getParseErr)
	assert.Contains(t, getParseErr.Error(), "unable to unmarshal")

	requestNull := ""
	_, getEmptyErr := conn.GetWithString(ctx, requestNull)
	assert.Error(t, getEmptyErr)
	assert.Contains(t, getEmptyErr.Error(), "empty request")
	s.Stop()
}

func TestGNMIConn_NonTLS(t *testing.T) {
	s := setup(t, getNonTLSServerConfig(t))

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	target1 := createTestTarget(t, target1, true)

	destination, err := newDestination(target1)
	assert.NoError(t, err)
	conn, _, err := connect(ctx, *destination)
	assert.NoError(t, err)
	assert.NotNil(t, conn)
	assert.NoError(t, err)
	capResponse, err := conn.Capabilities(ctx, &gpb.CapabilityRequest{})
	assert.NoError(t, err)
	assert.Equal(t, capResponse.GNMIVersion, "0.7.0")
	s.Stop()
}
