// Code generated by MockGen. DO NOT EDIT.
// Source: pkg/store/change/device/state/store.go

// Package store is a generated GoMock package.
package store

import (
	gomock "github.com/golang/mock/gomock"
	device "github.com/onosproject/onos-api/go/onos/config/change/device"
	network "github.com/onosproject/onos-api/go/onos/config/change/network"
	device0 "github.com/onosproject/onos-api/go/onos/config/device"
	reflect "reflect"
)

// MockDeviceStateStore is a mock of Store interface
type MockDeviceStateStore struct {
	ctrl     *gomock.Controller
	recorder *MockDeviceStateStoreMockRecorder
}

// MockDeviceStateStoreMockRecorder is the mock recorder for MockDeviceStateStore
type MockDeviceStateStoreMockRecorder struct {
	mock *MockDeviceStateStore
}

// NewMockDeviceStateStore creates a new mock instance
func NewMockDeviceStateStore(ctrl *gomock.Controller) *MockDeviceStateStore {
	mock := &MockDeviceStateStore{ctrl: ctrl}
	mock.recorder = &MockDeviceStateStoreMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use
func (m *MockDeviceStateStore) EXPECT() *MockDeviceStateStoreMockRecorder {
	return m.recorder
}

// Get mocks base method
func (m *MockDeviceStateStore) Get(id device0.VersionedID, revision network.Revision) ([]*device.PathValue, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Get", id, revision)
	ret0, _ := ret[0].([]*device.PathValue)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// Get indicates an expected call of Get
func (mr *MockDeviceStateStoreMockRecorder) Get(id, revision interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Get", reflect.TypeOf((*MockDeviceStateStore)(nil).Get), id, revision)
}
