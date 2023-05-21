// Code generated by MockGen. DO NOT EDIT.
// Source: github.com/onosproject/onos-config/pkg/pluginregistry (interfaces: PluginRegistry)

// Package pluginregistry is a generated GoMock package.
package pluginregistry

import (
	"context"
	"github.com/openconfig/gnmi/proto/gnmi"
	reflect "reflect"

	gomock "github.com/golang/mock/gomock"
	admin "github.com/onosproject/onos-api/go/onos/config/admin"
	v2 "github.com/onosproject/onos-api/go/onos/config/v2"
)

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
func (m *MockPluginRegistry) GetPlugin(arg0 v2.TargetType, arg1 v2.TargetVersion) (ModelPlugin, bool) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetPlugin", arg0, arg1)
	ret0, _ := ret[0].(ModelPlugin)
	ret1, _ := ret[1].(bool)
	return ret0, ret1
}

// GetPlugin indicates an expected call of GetPlugin.
func (mr *MockPluginRegistryMockRecorder) GetPlugin(arg0, arg1 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetPlugin", reflect.TypeOf((*MockPluginRegistry)(nil).GetPlugin), arg0, arg1)
}

// GetPlugins mocks base method.
func (m *MockPluginRegistry) GetPlugins() []ModelPlugin {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetPlugins")
	ret0, _ := ret[0].([]ModelPlugin)
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
func (m *MockModelPlugin) Capabilities(arg0 context.Context) *gnmi.CapabilityResponse {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Capabilities", arg0)
	ret0, _ := ret[0].(*gnmi.CapabilityResponse)
	return ret0
}

// Capabilities indicates an expected call of Capabilities.
func (mr *MockModelPluginMockRecorder) Capabilities(arg0 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Capabilities", reflect.TypeOf((*MockModelPlugin)(nil).Capabilities), arg0)
}

// GetInfo mocks base method.
func (m *MockModelPlugin) GetInfo() *ModelPluginInfo {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetInfo")
	ret0, _ := ret[0].(*ModelPluginInfo)
	return ret0
}

// GetInfo indicates an expected call of GetInfo.
func (mr *MockModelPluginMockRecorder) GetInfo() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetInfo", reflect.TypeOf((*MockModelPlugin)(nil).GetInfo))
}

// GetPathValues mocks base method.
func (m *MockModelPlugin) GetPathValues(arg0 context.Context, arg1 string, arg2 []byte) ([]*v2.PathValue, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetPathValues", arg0, arg1, arg2)
	ret0, _ := ret[0].([]*v2.PathValue)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// GetPathValues indicates an expected call of GetPathValues.
func (mr *MockModelPluginMockRecorder) GetPathValues(arg0, arg1, arg2 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetPathValues", reflect.TypeOf((*MockModelPlugin)(nil).GetPathValues), arg0, arg1, arg2)
}

// LeafValueSelection mocks base method.
func (m *MockModelPlugin) LeafValueSelection(arg0 context.Context, arg1 string, arg2 []byte) ([]string, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "LeafValueSelection", arg0, arg1, arg2)
	ret0, _ := ret[0].([]string)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// LeafValueSelection indicates an expected call of LeafValueSelection.
func (mr *MockModelPluginMockRecorder) LeafValueSelection(arg0, arg1, arg2 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "LeafValueSelection", reflect.TypeOf((*MockModelPlugin)(nil).LeafValueSelection), arg0, arg1, arg2)
}

// Validate mocks base method.
func (m *MockModelPlugin) Validate(arg0 context.Context, arg1 []byte) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Validate", arg0, arg1)
	ret0, _ := ret[0].(error)
	return ret0
}

// Validate indicates an expected call of Validate.
func (mr *MockModelPluginMockRecorder) Validate(arg0, arg1 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Validate", reflect.TypeOf((*MockModelPlugin)(nil).Validate), arg0, arg1)
}