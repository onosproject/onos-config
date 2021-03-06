// Code generated by MockGen. DO NOT EDIT.
// Source: pkg/store/device/cache.go

// Package device is a generated GoMock package.
package cache

import (
	gomock "github.com/golang/mock/gomock"
	device "github.com/onosproject/onos-api/go/onos/config/device"
	cache "github.com/onosproject/onos-config/pkg/store/device/cache"
	stream "github.com/onosproject/onos-config/pkg/store/stream"
	reflect "reflect"
)

// MockCache is a mock of Cache interface
type MockCache struct {
	ctrl     *gomock.Controller
	recorder *MockCacheMockRecorder
}

// MockCacheMockRecorder is the mock recorder for MockCache
type MockCacheMockRecorder struct {
	mock *MockCache
}

// NewMockCache creates a new mock instance
func NewMockCache(ctrl *gomock.Controller) *MockCache {
	mock := &MockCache{ctrl: ctrl}
	mock.recorder = &MockCacheMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use
func (m *MockCache) EXPECT() *MockCacheMockRecorder {
	return m.recorder
}

// Close mocks base method
func (m *MockCache) Close() error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Close")
	ret0, _ := ret[0].(error)
	return ret0
}

// Close indicates an expected call of Close
func (mr *MockCacheMockRecorder) Close() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Close", reflect.TypeOf((*MockCache)(nil).Close))
}

// GetDevicesByID mocks base method
func (m *MockCache) GetDevicesByID(id device.ID) []*cache.Info {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetDevicesByID", id)
	ret0, _ := ret[0].([]*cache.Info)
	return ret0
}

// GetDevicesByID indicates an expected call of GetDevicesByID
func (mr *MockCacheMockRecorder) GetDevicesByID(id interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetDevicesByID", reflect.TypeOf((*MockCache)(nil).GetDevicesByID), id)
}

// GetDevicesByType mocks base method
func (m *MockCache) GetDevicesByType(deviceType device.Type) []*cache.Info {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetDevicesByType", deviceType)
	ret0, _ := ret[0].([]*cache.Info)
	return ret0
}

// GetDevicesByType indicates an expected call of GetDevicesByType
func (mr *MockCacheMockRecorder) GetDevicesByType(deviceType interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetDevicesByType", reflect.TypeOf((*MockCache)(nil).GetDevicesByType), deviceType)
}

// GetDevicesByVersion mocks base method
func (m *MockCache) GetDevicesByVersion(deviceType device.Type, deviceVersion device.Version) []*cache.Info {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetDevicesByVersion", deviceType, deviceVersion)
	ret0, _ := ret[0].([]*cache.Info)
	return ret0
}

// GetDevicesByVersion indicates an expected call of GetDevicesByVersion
func (mr *MockCacheMockRecorder) GetDevicesByVersion(deviceType, deviceVersion interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetDevicesByVersion", reflect.TypeOf((*MockCache)(nil).GetDevicesByVersion), deviceType, deviceVersion)
}

// GetDevices mocks base method
func (m *MockCache) GetDevices() []*cache.Info {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetDevices")
	ret0, _ := ret[0].([]*cache.Info)
	return ret0
}

// GetDevices indicates an expected call of GetDevices
func (mr *MockCacheMockRecorder) GetDevices() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetDevices", reflect.TypeOf((*MockCache)(nil).GetDevices))
}

// Watch mocks base method
func (m *MockCache) Watch(ch chan<- stream.Event, replay bool) (stream.Context, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Watch", ch, replay)
	ret0, _ := ret[0].(stream.Context)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// Watch indicates an expected call of Watch
func (mr *MockCacheMockRecorder) Watch(ch, replay interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Watch", reflect.TypeOf((*MockCache)(nil).Watch), ch, replay)
}
