// SPDX-FileCopyrightText: 2020-present Open Networking Foundation <info@opennetworking.org>
//
// SPDX-License-Identifier: Apache-2.0

package gnmi

import (
	"context"
	"github.com/gogo/protobuf/proto"
	configapi "github.com/onosproject/onos-api/go/onos/config/v2"
	"github.com/onosproject/onos-config/pkg/utils"
	"github.com/openconfig/gnmi/proto/gnmi"
	"github.com/openconfig/gnmi/proto/gnmi_ext"
	"github.com/stretchr/testify/assert"
	"testing"
)

func Test_BasicSetUpdate(t *testing.T) {
	test := createServer(t)
	defer test.atomix.Close()
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

func Test_BasicSetUpdateWithOverride(t *testing.T) {
	test := createServer(t)
	defer test.atomix.Close()
	defer test.mctl.Finish()

	setupTopoAndRegistry(test, "target-1", "devicesim", "1.0.0", false)

	test.startControllers(t)
	defer test.stopControllers()

	targetID := configapi.TargetID("target-1")
	tvoext, err := utils.TargetVersionOverrideExtension(targetID, "devicesim", "1.0.0")
	assert.NoError(t, err)

	request := gnmi.SetRequest{
		Update: []*gnmi.Update{
			{
				Path: targetPath(t, targetID, "foo"),
				Val:  &gnmi.TypedValue{Value: &gnmi.TypedValue_StringVal{StringVal: "Hello world!"}},
			},
		},
		Extension: []*gnmi_ext.Extension{tvoext},
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
	defer test.atomix.Close()
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
	defer test.atomix.Close()
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
	defer test.atomix.Close()
	defer test.mctl.Finish()

	setupTopoAndRegistry(test, "target-1", "devicesim", "1.0.0", false)

	request := gnmi.SetRequest{}

	_, err := test.server.Set(context.TODO(), &request)
	assert.NotNil(t, err)
	assert.Contains(t, err.Error(), "no updates, replace or deletes")
}

func Test_NoPlugin(t *testing.T) {
	test := createServer(t)
	defer test.atomix.Close()
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
	defer test.atomix.Close()
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
