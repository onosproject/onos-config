// Code generated by MockGen. DO NOT EDIT.
// Source: pkg/store/change/device/store.go

// Package store is a generated GoMock package.
package store

import (
	indexedmap "github.com/atomix/atomix-go-client/pkg/client/indexedmap"
	gomock "github.com/golang/mock/gomock"
	device "github.com/onosproject/onos-config/pkg/store/change/device"
	device0 "github.com/onosproject/onos-config/pkg/types/change/device"
	device1 "github.com/onosproject/onos-topo/pkg/northbound/device"
	reflect "reflect"
)

// MockStore is a mock of Store interface
type MockDeviceChangesStore struct {
	ctrl     *gomock.Controller
	recorder *MockDeviceChangesStoreMockRecorder
}

// MockStoreMockRecorder is the mock recorder for MockStore
type MockDeviceChangesStoreMockRecorder struct {
	mock *MockDeviceChangesStore
}

// NewMockDeviceChangesStore creates a new mock instance
func NewMockDeviceChangesStore(ctrl *gomock.Controller) *MockDeviceChangesStore {
	mock := &MockDeviceChangesStore{ctrl: ctrl}
	mock.recorder = &MockDeviceChangesStoreMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use
func (m *MockDeviceChangesStore) EXPECT() *MockDeviceChangesStoreMockRecorder {
	return m.recorder
}

// Close mocks base method
func (m *MockDeviceChangesStore) Close() error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Close")
	ret0, _ := ret[0].(error)
	return ret0
}

// Close indicates an expected call of Close
func (mr *MockDeviceChangesStoreMockRecorder) Close() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Close", reflect.TypeOf((*MockDeviceChangesStore)(nil).Close))
}

// Get mocks base method
func (m *MockDeviceChangesStore) Get(id device0.ID) (*device0.DeviceChange, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Get", id)
	ret0, _ := ret[0].(*device0.DeviceChange)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// Get indicates an expected call of Get
func (mr *MockDeviceChangesStoreMockRecorder) Get(id interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Get", reflect.TypeOf((*MockDeviceChangesStore)(nil).Get), id)
}

// Create mocks base method
func (m *MockDeviceChangesStore) Create(config *device0.DeviceChange) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Create", config)
	ret0, _ := ret[0].(error)
	return ret0
}

// Create indicates an expected call of Create
func (mr *MockDeviceChangesStoreMockRecorder) Create(config interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Create", reflect.TypeOf((*MockDeviceChangesStore)(nil).Create), config)
}

// Update mocks base method
func (m *MockDeviceChangesStore) Update(config *device0.DeviceChange) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Update", config)
	ret0, _ := ret[0].(error)
	return ret0
}

// Update indicates an expected call of Update
func (mr *MockDeviceChangesStoreMockRecorder) Update(config interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Update", reflect.TypeOf((*MockDeviceChangesStore)(nil).Update), config)
}

// Delete mocks base method
func (m *MockDeviceChangesStore) Delete(config *device0.DeviceChange) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Delete", config)
	ret0, _ := ret[0].(error)
	return ret0
}

// Delete indicates an expected call of Delete
func (mr *MockDeviceChangesStoreMockRecorder) Delete(config interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Delete", reflect.TypeOf((*MockDeviceChangesStore)(nil).Delete), config)
}

// List mocks base method
func (m *MockDeviceChangesStore) List(arg0 device1.ID, arg1 chan<- *device0.DeviceChange) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "List", arg0, arg1)
	ret0, _ := ret[0].(error)
	return ret0
}

// List indicates an expected call of List
func (mr *MockDeviceChangesStoreMockRecorder) List(arg0, arg1 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "List", reflect.TypeOf((*MockDeviceChangesStore)(nil).List), arg0, arg1)
}

// Watch mocks base method
func (m *MockDeviceChangesStore) Watch(arg0 device1.ID, arg1 chan<- *device0.DeviceChange, arg2 ...device.WatchOption) error {
	m.ctrl.T.Helper()
	varargs := []interface{}{arg0, arg1}
	for _, a := range arg2 {
		varargs = append(varargs, a)
	}
	ret := m.ctrl.Call(m, "Watch", varargs...)
	ret0, _ := ret[0].(error)
	return ret0
}

// Watch indicates an expected call of Watch
func (mr *MockDeviceChangesStoreMockRecorder) Watch(arg0, arg1 interface{}, arg2 ...interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	varargs := append([]interface{}{arg0, arg1}, arg2...)
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Watch", reflect.TypeOf((*MockDeviceChangesStore)(nil).Watch), varargs...)
}

// MockWatchOption is a mock of WatchOption interface
type MockWatchOption struct {
	ctrl     *gomock.Controller
	recorder *MockWatchOptionMockRecorder
}

// MockWatchOptionMockRecorder is the mock recorder for MockWatchOption
type MockWatchOptionMockRecorder struct {
	mock *MockWatchOption
}

// NewMockWatchOption creates a new mock instance
func NewMockWatchOption(ctrl *gomock.Controller) *MockWatchOption {
	mock := &MockWatchOption{ctrl: ctrl}
	mock.recorder = &MockWatchOptionMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use
func (m *MockWatchOption) EXPECT() *MockWatchOptionMockRecorder {
	return m.recorder
}

// apply mocks base method
func (m *MockWatchOption) apply(arg0 []indexedmap.WatchOption) []indexedmap.WatchOption {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "apply", arg0)
	ret0, _ := ret[0].([]indexedmap.WatchOption)
	return ret0
}

// apply indicates an expected call of apply
func (mr *MockWatchOptionMockRecorder) apply(arg0 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "apply", reflect.TypeOf((*MockWatchOption)(nil).apply), arg0)
}
