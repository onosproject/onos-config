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

package southbound

import (
	"context"
	"strconv"
	"testing"
	"time"

	"github.com/golang/protobuf/proto"
	devicetype "github.com/onosproject/onos-api/go/onos/config/device"
	topodevice "github.com/onosproject/onos-config/pkg/device"
	"github.com/onosproject/onos-config/pkg/utils"
	"github.com/openconfig/gnmi/client"
	"github.com/openconfig/gnmi/proto/gnmi"
	"github.com/stretchr/testify/assert"
)

var (
	device                    topodevice.Device
	saveGnmiClientFactory     func(ctx context.Context, d client.Destination) (GnmiClient, error)
	saveGnmiBaseClientFactory func() BaseClientInterface
)

// Test Client to stub out the gnmiClient

type TestClientImpl struct {
}

func (TestClientImpl) Set(ctx context.Context, r *gnmi.SetRequest) (*gnmi.SetResponse, error) {
	setUpdateResult := make([]*gnmi.UpdateResult, 1)
	setUpdateResult[0] = &gnmi.UpdateResult{Op: gnmi.UpdateResult_DELETE}
	setResponse := gnmi.SetResponse{Response: setUpdateResult}
	return &setResponse, nil
}

func (TestClientImpl) Recv() error {
	return nil
}

func (TestClientImpl) Close() error {
	return nil
}

func (TestClientImpl) Poll() error {
	return nil
}

func (TestClientImpl) Capabilities(ctx context.Context, r *gnmi.CapabilityRequest) (*gnmi.CapabilityResponse, error) {
	model1 := gnmi.ModelData{
		Name:         "model 1",
		Organization: "ONF",
		Version:      "1.1",
	}
	models := make([]*gnmi.ModelData, 1)
	models[0] = &model1

	encodings := make([]gnmi.Encoding, 1)
	var encoding1 = gnmi.Encoding_ASCII
	encodings[0] = encoding1

	response := gnmi.CapabilityResponse{
		SupportedModels:    models,
		GNMIVersion:        "1.0",
		SupportedEncodings: encodings,
	}
	return &response, nil
}

func (TestClientImpl) Get(ctx context.Context, r *gnmi.GetRequest) (*gnmi.GetResponse, error) {
	response := gnmi.GetResponse{}
	response.Notification = make([]*gnmi.Notification, len(r.Path))

	for notificationIndex := range response.Notification {
		response.Notification[notificationIndex] = &gnmi.Notification{}
		response.Notification[notificationIndex].Update = make([]*gnmi.Update, 1)
		update := &gnmi.Update{}
		update.Path = r.Path[notificationIndex]
		update.Val = &gnmi.TypedValue{}
		update.Val.Value = &gnmi.TypedValue_StringVal{StringVal: strconv.Itoa(notificationIndex)}
		response.Notification[notificationIndex].Update[notificationIndex] = update
	}

	return &response, nil
}

func (TestClientImpl) Subscribe(context.Context, client.Query) error {
	return nil
}

//  Test client to stub out the gnmiCacheClient

type TestCacheClient struct {
}

func (TestCacheClient) Subscribe(context.Context, client.Query, ...string) error {
	return nil
}

func setUp(t *testing.T) {
	saveGnmiClientFactory = GnmiClientFactory
	GnmiClientFactory = func(ctx context.Context, d client.Destination) (GnmiClient, error) {
		return TestClientImpl{}, nil
	}

	saveGnmiBaseClientFactory = GnmiBaseClientFactory
	GnmiBaseClientFactory = func() BaseClientInterface {
		c := TestCacheClient{}
		return c
	}

	timeout := 10 * time.Second
	device = topodevice.Device{
		ID:      "localhost-1",
		Address: "localhost:10161",
		Version: "1.0.0",
		Credentials: topodevice.Credentials{
			User:     "devicesim",
			Password: "notused",
		},
		Timeout: &timeout,
	}

	targets[devicetype.NewVersionedID("dummyDevice1", "1.0.0")] = NewTarget()
	targets[devicetype.NewVersionedID("dummyDevice1", "2.0.0")] = NewTarget()
	targets[devicetype.NewVersionedID("dummyDevice2", "1.0.0")] = NewTarget()
}

func tearDown() {
	GnmiClientFactory = saveGnmiClientFactory
	GnmiBaseClientFactory = saveGnmiBaseClientFactory
	for t := range targets {
		delete(targets, t)
	}
}

func getDevice1Target(t *testing.T) (*Target, devicetype.VersionedID, context.Context) {
	target := &Target{}
	ctx := context.Background()
	key, err := target.ConnectTarget(ctx, device)
	assert.NoError(t, err)
	assert.NotNil(t, target.clt)
	assert.Equal(t, "localhost-1", string(key.GetID()))
	assert.NotNil(t, target.ctx)
	return target, key, ctx
}

func Test_ConnectTarget(t *testing.T) {
	setUp(t)

	target, key, _ := getDevice1Target(t)

	targetFetch, fetchError := GetTarget(key)
	assert.NoError(t, fetchError)
	assert.Equal(t, target.Destination().Addrs, targetFetch.Destination().Addrs)
	assert.Equal(t, target.Client(), targetFetch.Client())
	tearDown()
}

func Test_BadTarget(t *testing.T) {
	setUp(t)

	key := devicetype.NewVersionedID("dummyDevice1", "1.0.1")
	_, fetchError := GetTarget(key) // Should discriminate on different version
	t.Log(fetchError)
	assert.Error(t, fetchError)
	assert.Contains(t, fetchError.Error(), "gNMI client for dummyDevice1:1.0.1 does not exist")
	assert.Contains(t, fetchError.Error(), "dummyDevice1:1.0.0")
	assert.Contains(t, fetchError.Error(), "dummyDevice1:2.0.0")
	assert.Contains(t, fetchError.Error(), "dummyDevice2:1.0.0")

	key = devicetype.NewVersionedID("dummyDevice3", "1.0.0")
	_, fetchError = GetTarget(key) // Should discriminate on different name
	t.Log(fetchError)
	assert.Error(t, fetchError)
	assert.Contains(t, fetchError.Error(), "gNMI client for dummyDevice3:1.0.0 does not exist")
	assert.Contains(t, fetchError.Error(), "dummyDevice1:1.0.0")
	assert.Contains(t, fetchError.Error(), "dummyDevice1:2.0.0")
	assert.Contains(t, fetchError.Error(), "dummyDevice2:1.0.0")

	tearDown()
}

func Test_ConnectTargetUserPassword(t *testing.T) {
	setUp(t)

	device.TLS.Cert = "cert path"
	device.TLS.Key = ""
	device.Credentials.User = "User"
	device.Credentials.Password = "Password"
	target, key, _ := getDevice1Target(t)

	targetFetch, fetchError := GetTarget(key)
	assert.NoError(t, fetchError)
	assert.Equal(t, target.Destination().Credentials.Username, "User")
	assert.Equal(t, target.Destination().Credentials.Password, "Password")
	assert.Equal(t, target.clt, targetFetch.Client())

	tearDown()
}

func Test_ConnectTargetInsecurePaths(t *testing.T) {
	setUp(t)

	device.TLS.Cert = "cert path"
	device.TLS.Key = ""
	target, key, _ := getDevice1Target(t)

	targetFetch, fetchError := GetTarget(key)
	assert.NoError(t, fetchError)
	assert.Equal(t, targetFetch.Destination().TLS.InsecureSkipVerify, false)
	assert.Equal(t, target.clt, targetFetch.Client())

	tearDown()
}

func Test_ConnectTargetInsecureFlag(t *testing.T) {
	setUp(t)

	device.TLS.Insecure = true
	target, key, _ := getDevice1Target(t)

	targetFetch, fetchError := GetTarget(key)
	assert.NoError(t, fetchError)
	assert.Equal(t, targetFetch.Destination().TLS.InsecureSkipVerify, true)
	assert.Equal(t, target.clt, targetFetch.Client())

	tearDown()
}

func Test_ConnectTargetWithCert(t *testing.T) {
	setUp(t)

	device.TLS.Cert = "testdata/client1.crt"
	device.TLS.Key = "testdata/client1.key"
	device.TLS.CaCert = "testdata/onfca.crt"
	target, key, _ := getDevice1Target(t)

	targetFetch, fetchError := GetTarget(key)
	assert.NoError(t, fetchError)
	ca := getCertPool("testdata/onfca.crt")
	assert.Equal(t, targetFetch.Destination().TLS.RootCAs.Subjects()[0], ca.Subjects()[0])
	cert := setCertificate("testdata/client1.crt", "testdata/client1.key")
	assert.Equal(t, targetFetch.Destination().TLS.Certificates[0].Certificate, cert.Certificate)
	assert.Equal(t, target.clt, targetFetch.Client())

	tearDown()
}

func Test_Get(t *testing.T) {
	setUp(t)

	target, _, _ := getDevice1Target(t)

	allDevicesPath := gnmi.Path{Elem: make([]*gnmi.PathElem, 0), Target: "*"}

	request := gnmi.GetRequest{
		Path: []*gnmi.Path{&allDevicesPath},
	}

	response, err := target.Get(context.TODO(), &request)
	assert.NoError(t, err)
	assert.NotNil(t, response)
	assert.Equal(t, response.Notification[0].Update[0].Path.Target, "*", "Expected target")
	value := utils.StrVal(response.Notification[0].Update[0].Val)
	assert.Equal(t, value, "0", "Expected index as value")

	tearDown()
}

func Test_GetWithString(t *testing.T) {
	setUp(t)

	target, _, ctx := getDevice1Target(t)

	request := "path: <elem: <name: 'system'> elem:<name:'config'> elem: <name: 'hostname'>>"
	getResponse, getErr := target.GetWithString(ctx, request)

	assert.NoError(t, getErr)
	assert.NotNil(t, getResponse)
	assert.Equal(t, getResponse.Notification[0].Update[0].Path.Elem[0].Name, "system")
	value := utils.StrVal(getResponse.Notification[0].Update[0].Val)
	assert.Equal(t, value, "0")

	tearDown()
}

func Test_GetWithBadString(t *testing.T) {
	setUp(t)

	target, _, ctx := getDevice1Target(t)

	requestBadParse := "!!!path: <elem: <name: 'system'> elem:<name:'config'> elem: <name: 'hostname'>>"
	_, getParseErr := target.GetWithString(ctx, requestBadParse)
	assert.Error(t, getParseErr)
	assert.Contains(t, getParseErr.Error(), "unable to parse")

	requestNull := ""
	_, getEmptyErr := target.GetWithString(ctx, requestNull)
	assert.Error(t, getEmptyErr)
	assert.Contains(t, getEmptyErr.Error(), "empty request")

	tearDown()
}

func Test_Subscribe(t *testing.T) {
	setUp(t)

	target := Target{}
	target.Destination().Addrs = make([]string, 1)
	target.Destination().Addrs[0] = "127.0.0.1"
	ctx := context.Background()

	_, connectError := target.ConnectTarget(ctx, device)
	assert.NoError(t, connectError)

	paths := make([][]string, 1)
	paths[0] = make([]string, 3)
	paths[0][0] = "a"
	paths[0][1] = "b"
	paths[0][2] = "c"
	options := &SubscribeOptions{
		UpdatesOnly:       false,
		Prefix:            "",
		Mode:              "Stream",
		StreamMode:        "target_defined",
		SampleInterval:    15,
		HeartbeatInterval: 15,
		Paths:             paths,
		Origin:            "",
	}
	request, requestError := NewSubscribeRequest(options)
	assert.NotNil(t, request)
	assert.Equal(t, request.GetSubscribe().Subscription[0].Path.Elem[0].Name, "a")
	assert.Equal(t, request.GetSubscribe().Subscription[0].Path.Elem[1].Name, "b")
	assert.Equal(t, request.GetSubscribe().Subscription[0].Path.Elem[2].Name, "c")
	assert.NoError(t, requestError)

	subscribeError := target.Subscribe(ctx, request, handler)

	assert.NoError(t, subscribeError)

	tearDown()
}

func handler(msg proto.Message) error {
	return nil
}

func Test_NewSubscribeRequest(t *testing.T) {
	paths := make([][]string, 1)
	paths[0] = make([]string, 1)
	paths[0][0] = "/a/b/c"
	options := &SubscribeOptions{
		UpdatesOnly:       false,
		Prefix:            "",
		Mode:              "Stream",
		StreamMode:        "target_defined",
		SampleInterval:    15,
		HeartbeatInterval: 15,
		Paths:             paths,
		Origin:            "",
	}
	request, requestError := NewSubscribeRequest(options)
	assert.NoError(t, requestError)
	assert.Equal(t, request.GetSubscribe().Mode, gnmi.SubscriptionList_STREAM)
	assert.Equal(t, request.GetSubscribe().GetSubscription()[0].Mode, gnmi.SubscriptionMode_TARGET_DEFINED)

	options.Mode = "Once"
	options.StreamMode = "on_change"
	request, requestError = NewSubscribeRequest(options)
	assert.NoError(t, requestError)
	assert.Equal(t, request.GetSubscribe().Mode, gnmi.SubscriptionList_ONCE)
	assert.Equal(t, request.GetSubscribe().GetSubscription()[0].Mode, gnmi.SubscriptionMode_ON_CHANGE)

	options.Mode = "Poll"
	options.StreamMode = "sample"
	request, requestError = NewSubscribeRequest(options)
	assert.NoError(t, requestError)
	assert.Equal(t, request.GetSubscribe().Mode, gnmi.SubscriptionList_POLL)
	assert.Equal(t, request.GetSubscribe().GetSubscription()[0].Mode, gnmi.SubscriptionMode_SAMPLE)

	options.Mode = "Test_Error"
	_, requestError = NewSubscribeRequest(options)
	assert.Error(t, requestError)
	assert.Contains(t, requestError.Error(), "invalid")

	options.Mode = "Poll"
	options.StreamMode = "test_error"
	_, requestError = NewSubscribeRequest(options)
	assert.Error(t, requestError)
	assert.Contains(t, requestError.Error(), "invalid")

}

func Test_CapabilitiesWithString(t *testing.T) {
	setUp(t)

	target, _, ctx := getDevice1Target(t)

	capabilityResponse, capabilityErr := target.CapabilitiesWithString(ctx, "")

	assert.NoError(t, capabilityErr)
	assert.NotNil(t, capabilityResponse)
	assert.Equal(t, capabilityResponse.SupportedEncodings[0], gnmi.Encoding_ASCII)
	assert.Equal(t, capabilityResponse.GNMIVersion, "1.0")
	assert.Equal(t, capabilityResponse.SupportedModels[0].Organization, "ONF")

	tearDown()
}

func Test_CapabilitiesWithBadString(t *testing.T) {
	setUp(t)

	target, _, ctx := getDevice1Target(t)

	_, capabilityErr := target.CapabilitiesWithString(ctx, "not a valid string")

	assert.Error(t, capabilityErr)
	assert.Contains(t, capabilityErr.Error(), "unable to parse")

	tearDown()
}

func Test_SetWithString(t *testing.T) {
	setUp(t)

	target, _, ctx := getDevice1Target(t)

	request := "delete: <elem: <name: 'system'> elem:<name:'config'> elem: <name: 'hostname'>>"
	setResponse, setErr := target.SetWithString(ctx, request)

	assert.NoError(t, setErr)
	assert.NotNil(t, setResponse)
	assert.Equal(t, setResponse.Response[0].Op, gnmi.UpdateResult_DELETE)

	tearDown()
}
