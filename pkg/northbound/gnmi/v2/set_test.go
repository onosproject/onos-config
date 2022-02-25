// Copyright 2022-present Open Networking Foundation.
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

package gnmi

import (
	"context"
	"github.com/gogo/protobuf/proto"
	configapi "github.com/onosproject/onos-api/go/onos/config/v2"
	"github.com/openconfig/gnmi/proto/gnmi"
	"github.com/stretchr/testify/assert"
	"testing"
)

func Test_BasicSetUpdate(t *testing.T) {
	test := createServer(t)
	defer test.atomix.Stop()
	defer test.mctl.Finish()

	setupTopoAndRegistry(test, "target-1", "devicesim", "1.0.0", false)

	test.startControllers(t)
	defer test.stopControllers()

	targetID := configapi.TargetID("target-1")
	request := gnmi.SetRequest{
		Update: []*gnmi.Update{
			{
				Path: targetPath(t, targetID, "foo"),
				Val:  &gnmi.TypedValue{Value: &gnmi.TypedValue_StringVal{StringVal: "Hello world!"}},
			},
		},
	}

	result, err := test.server.Set(context.TODO(), &request)
	assert.NoError(t, err)
	assert.Len(t, result.Extension, 1)

	transactionInfo := &configapi.TransactionInfo{}
	assert.NoError(t, proto.Unmarshal(result.Extension[0].GetRegisteredExt().GetMsg(), transactionInfo))
	tx, err := test.transaction.Get(context.TODO(), transactionInfo.ID)
	assert.NoError(t, err)
	assert.NotNil(t, tx.Status.Phases.Commit)
	assert.Equal(t, configapi.TransactionCommitPhase_COMMITTED, tx.Status.Phases.Commit.State)
}

func Test_SetJsonUpdate(t *testing.T) {
	test := createServer(t)
	defer test.atomix.Stop()
	defer test.mctl.Finish()

	setupTopoAndRegistry(test, "target-1", "devicesim", "1.0.0", false)

	test.startControllers(t)
	defer test.stopControllers()

	targetID := configapi.TargetID("target-1")
	request := gnmi.SetRequest{
		Update: []*gnmi.Update{
			{
				Path: targetPath(t, targetID),
				Val:  &gnmi.TypedValue{Value: &gnmi.TypedValue_JsonVal{JsonVal: []byte("{\"foo\": \"Hello world!\"}")}},
			},
		},
	}

	result, err := test.server.Set(context.TODO(), &request)
	assert.NoError(t, err)
	assert.Len(t, result.Extension, 1)

	transactionInfo := &configapi.TransactionInfo{}
	assert.NoError(t, proto.Unmarshal(result.Extension[0].GetRegisteredExt().GetMsg(), transactionInfo))
	tx, err := test.transaction.Get(context.TODO(), transactionInfo.ID)
	assert.NoError(t, err)
	assert.NotNil(t, tx.Status.Phases.Commit)
	assert.Equal(t, configapi.TransactionCommitPhase_COMMITTED, tx.Status.Phases.Commit.State)
}

func Test_SetUpdateReplaceDelete(t *testing.T) {
	test := createServer(t)
	defer test.atomix.Stop()
	defer test.mctl.Finish()

	setupTopoAndRegistry(test, "target-1", "devicesim", "1.0.0", false)

	test.startControllers(t)
	defer test.stopControllers()

	targetID := configapi.TargetID("target-1")
	request := gnmi.SetRequest{
		Update: []*gnmi.Update{
			{
				Path: targetPath(t, targetID, "foo"),
				Val:  &gnmi.TypedValue{Value: &gnmi.TypedValue_StringVal{StringVal: "Hello world!"}},
			},
		},
		Replace: []*gnmi.Update{
			{
				Path: targetPath(t, targetID, "bar"),
				Val:  &gnmi.TypedValue{Value: &gnmi.TypedValue_StringVal{StringVal: "Bye world!"}},
			},
		},
		Delete: []*gnmi.Path{targetPath(t, targetID, "goo")},
	}

	result, err := test.server.Set(context.TODO(), &request)
	assert.NoError(t, err)
	assert.Len(t, result.Extension, 1)

	transactionInfo := &configapi.TransactionInfo{}
	assert.NoError(t, proto.Unmarshal(result.Extension[0].GetRegisteredExt().GetMsg(), transactionInfo))
	tx, err := test.transaction.Get(context.TODO(), transactionInfo.ID)
	assert.NoError(t, err)
	assert.NotNil(t, tx.Status.Phases.Commit)
	assert.Equal(t, configapi.TransactionCommitPhase_COMMITTED, tx.Status.Phases.Commit.State)
}

func Test_NoUpdateSet(t *testing.T) {
	test := createServer(t)
	defer test.atomix.Stop()
	defer test.mctl.Finish()

	setupTopoAndRegistry(test, "target-1", "devicesim", "1.0.0", false)

	request := gnmi.SetRequest{}

	_, err := test.server.Set(context.TODO(), &request)
	assert.NotNil(t, err)
	assert.Contains(t, err.Error(), "no updates, replace or deletes")
}

func Test_NoPlugin(t *testing.T) {
	test := createServer(t)
	defer test.atomix.Stop()
	defer test.mctl.Finish()

	setupTopoAndRegistry(test, "target-1", "devicesim", "1.0.0", true)

	targetID := configapi.TargetID("target-1")
	request := gnmi.SetRequest{
		Update: []*gnmi.Update{
			{
				Path: targetPath(t, targetID, "foo"),
				Val:  &gnmi.TypedValue{Value: &gnmi.TypedValue_StringVal{StringVal: "Hello world!"}},
			},
		},
	}

	_, err := test.server.Set(context.TODO(), &request)
	assert.NotNil(t, err)
	assert.Contains(t, err.Error(), "plugin not found")
}

func Test_SetDeleteSet(t *testing.T) {
	test := createServer(t)
	defer test.atomix.Stop()
	defer test.mctl.Finish()

	setupTopoAndRegistry(test, "target-1", "devicesim", "1.0.0", false)

	test.startControllers(t)
	defer test.stopControllers()

	targetID := configapi.TargetID("target-1")
	deleteNestedPath(t, targetID, test)
	setNestedPath(t, targetID, test)
	validateNestedPath(t, targetID, test)
}

func validateNestedPath(t *testing.T, targetID configapi.TargetID, test *testContext) {
	request := gnmi.GetRequest{
		Path:     []*gnmi.Path{targetPath(t, targetID, "some", "nested", "path")},
		Encoding: gnmi.Encoding_JSON,
	}

	result, err := test.server.Get(context.TODO(), &request)
	assert.NoError(t, err)
	assert.Len(t, result.Notification, 1)
	assert.Len(t, result.Notification[0].Update, 1)
	assert.Equal(t, "{\n  \"some\": {\n    \"nested\": {\n      \"path\": \"Hello Again!\"\n    }\n  }\n}",
		string(result.Notification[0].Update[0].GetVal().GetJsonVal()))
}

func setNestedPath(t *testing.T, targetID configapi.TargetID, test *testContext) {
	request := gnmi.SetRequest{
		Update: []*gnmi.Update{
			{
				Path: targetPath(t, targetID, "some", "nested", "path"),
				Val:  &gnmi.TypedValue{Value: &gnmi.TypedValue_StringVal{StringVal: "Hello Again!"}},
			},
		},
	}

	result, err := test.server.Set(context.TODO(), &request)
	assert.NoError(t, err)
	assert.Len(t, result.Extension, 1)

	transactionInfo := &configapi.TransactionInfo{}
	assert.NoError(t, proto.Unmarshal(result.Extension[0].GetRegisteredExt().GetMsg(), transactionInfo))
	tx, err := test.transaction.Get(context.TODO(), transactionInfo.ID)
	assert.NoError(t, err)
	assert.NotNil(t, tx.Status.Phases.Commit)
	assert.Equal(t, configapi.TransactionCommitPhase_COMMITTED, tx.Status.Phases.Commit.State)
}

func deleteNestedPath(t *testing.T, targetID configapi.TargetID, test *testContext) {
	request := gnmi.SetRequest{
		Delete: []*gnmi.Path{targetPath(t, targetID, "some")},
	}

	result, err := test.server.Set(context.TODO(), &request)
	assert.NoError(t, err)
	assert.Len(t, result.Extension, 1)

	transactionInfo := &configapi.TransactionInfo{}
	assert.NoError(t, proto.Unmarshal(result.Extension[0].GetRegisteredExt().GetMsg(), transactionInfo))
	tx, err := test.transaction.Get(context.TODO(), transactionInfo.ID)
	assert.NoError(t, err)
	assert.NotNil(t, tx.Status.Phases.Commit)
	assert.Equal(t, configapi.TransactionCommitPhase_COMMITTED, tx.Status.Phases.Commit.State)
}
