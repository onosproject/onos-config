/*
 * SPDX-FileCopyrightText: 2022-present Intel Corporation
 *
 * SPDX-License-Identifier: Apache-2.0
 */

// Code generated by MockGen. DO NOT EDIT.
// Source: registry.go

// Package test is a generated GoMock package.
package pluginregistry

import (
	context "context"
	reflect "reflect"

	gomock "github.com/golang/mock/gomock"
	admin "github.com/onosproject/onos-api/go/onos/config/admin"
	v2 "github.com/onosproject/onos-api/go/onos/config/v2"
	pluginregistry "github.com/onosproject/onos-config/pkg/pluginregistry"
	gnmi "github.com/openconfig/gnmi/proto/gnmi"
)

// MockModelPlugin is a mock of ModelPlugin interface.
type MockModelPlugin struct {
	ctrl     *gomock.Controller
	recorder *MockModelPluginMockRecorder
}

// MockModelPluginMockRecorder is the mock recorder for MockModelPlugin.
type MockModelPluginMockRecorder struct {
	mock *MockModelPlugin
}

// NewMockModelPlugin creates a new mock instance.
func NewMockModelPlugin(ctrl *gomock.Controller) *MockModelPlugin {
	mock := &MockModelPlugin{ctrl: ctrl}
	mock.recorder = &MockModelPluginMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use.
func (m *MockModelPlugin) EXPECT() *MockModelPluginMockRecorder {
	return m.recorder
}

// Capabilities mocks base method.
func (m *MockModelPlugin) Capabilities(ctx context.Context) *gnmi.CapabilityResponse {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Capabilities", ctx)
	ret0, _ := ret[0].(*gnmi.CapabilityResponse)
	return ret0
}

// Capabilities indicates an expected call of Capabilities.
func (mr *MockModelPluginMockRecorder) Capabilities(ctx interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Capabilities", reflect.TypeOf((*MockModelPlugin)(nil).Capabilities), ctx)
}

// GetInfo mocks base method.
func (m *MockModelPlugin) GetInfo() *pluginregistry.ModelPluginInfo {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetInfo")
	ret0, _ := ret[0].(*pluginregistry.ModelPluginInfo)
	return ret0
}

// GetInfo indicates an expected call of GetInfo.
func (mr *MockModelPluginMockRecorder) GetInfo() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetInfo", reflect.TypeOf((*MockModelPlugin)(nil).GetInfo))
}

// GetPathValues mocks base method.
func (m *MockModelPlugin) GetPathValues(ctx context.Context, pathPrefix string, jsonData []byte) ([]*v2.PathValue, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetPathValues", ctx, pathPrefix, jsonData)
	ret0, _ := ret[0].([]*v2.PathValue)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// GetPathValues indicates an expected call of GetPathValues.
func (mr *MockModelPluginMockRecorder) GetPathValues(ctx, pathPrefix, jsonData interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetPathValues", reflect.TypeOf((*MockModelPlugin)(nil).GetPathValues), ctx, pathPrefix, jsonData)
}

// LeafValueSelection mocks base method.
func (m *MockModelPlugin) LeafValueSelection(ctx context.Context, selectionPath string, jsonData []byte) ([]string, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "LeafValueSelection", ctx, selectionPath, jsonData)
	ret0, _ := ret[0].([]string)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// LeafValueSelection indicates an expected call of LeafValueSelection.
func (mr *MockModelPluginMockRecorder) LeafValueSelection(ctx, selectionPath, jsonData interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "LeafValueSelection", reflect.TypeOf((*MockModelPlugin)(nil).LeafValueSelection), ctx, selectionPath, jsonData)
}

// Validate mocks base method.
func (m *MockModelPlugin) Validate(ctx context.Context, jsonData []byte) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Validate", ctx, jsonData)
	ret0, _ := ret[0].(error)
	return ret0
}

// Validate indicates an expected call of Validate.
func (mr *MockModelPluginMockRecorder) Validate(ctx, jsonData interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Validate", reflect.TypeOf((*MockModelPlugin)(nil).Validate), ctx, jsonData)
}

// MockPluginRegistry is a mock of PluginRegistry interface.
type MockPluginRegistry struct {
	ctrl     *gomock.Controller
	recorder *MockPluginRegistryMockRecorder
}

// MockPluginRegistryMockRecorder is the mock recorder for MockPluginRegistry.
type MockPluginRegistryMockRecorder struct {
	mock *MockPluginRegistry
}

// NewMockPluginRegistry creates a new mock instance.
func NewMockPluginRegistry(ctrl *gomock.Controller) *MockPluginRegistry {
	mock := &MockPluginRegistry{ctrl: ctrl}
	mock.recorder = &MockPluginRegistryMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use.
func (m *MockPluginRegistry) EXPECT() *MockPluginRegistryMockRecorder {
	return m.recorder
}

// GetPlugin mocks base method.
func (m *MockPluginRegistry) GetPlugin(model v2.TargetType, version v2.TargetVersion) (pluginregistry.ModelPlugin, bool) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetPlugin", model, version)
	ret0, _ := ret[0].(pluginregistry.ModelPlugin)
	ret1, _ := ret[1].(bool)
	return ret0, ret1
}

// GetPlugin indicates an expected call of GetPlugin.
func (mr *MockPluginRegistryMockRecorder) GetPlugin(model, version interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetPlugin", reflect.TypeOf((*MockPluginRegistry)(nil).GetPlugin), model, version)
}

// GetPlugins mocks base method.
func (m *MockPluginRegistry) GetPlugins() []pluginregistry.ModelPlugin {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetPlugins")
	ret0, _ := ret[0].([]pluginregistry.ModelPlugin)
	return ret0
}

// GetPlugins indicates an expected call of GetPlugins.
func (mr *MockPluginRegistryMockRecorder) GetPlugins() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetPlugins", reflect.TypeOf((*MockPluginRegistry)(nil).GetPlugins))
}

// NewClientFn mocks base method.
func (m *MockPluginRegistry) NewClientFn(arg0 func(string) (admin.ModelPluginServiceClient, error)) {
	m.ctrl.T.Helper()
	m.ctrl.Call(m, "NewClientFn", arg0)
}

// NewClientFn indicates an expected call of NewClientFn.
func (mr *MockPluginRegistryMockRecorder) NewClientFn(arg0 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "NewClientFn", reflect.TypeOf((*MockPluginRegistry)(nil).NewClientFn), arg0)
}

// Start mocks base method.
func (m *MockPluginRegistry) Start() {
	m.ctrl.T.Helper()
	m.ctrl.Call(m, "Start")
}

// Start indicates an expected call of Start.
func (mr *MockPluginRegistryMockRecorder) Start() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Start", reflect.TypeOf((*MockPluginRegistry)(nil).Start))
}

// Stop mocks base method.
func (m *MockPluginRegistry) Stop() {
	m.ctrl.T.Helper()
	m.ctrl.Call(m, "Stop")
}

// Stop indicates an expected call of Stop.
func (mr *MockPluginRegistryMockRecorder) Stop() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Stop", reflect.TypeOf((*MockPluginRegistry)(nil).Stop))
}
