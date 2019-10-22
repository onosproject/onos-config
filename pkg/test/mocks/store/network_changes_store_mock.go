// Code generated by MockGen. DO NOT EDIT.
// Source: pkg/store/change/network/store.go

// Package store is a generated GoMock package.
package store

import (
	indexedmap "github.com/atomix/atomix-go-client/pkg/client/indexedmap"
	gomock "github.com/golang/mock/gomock"
	network "github.com/onosproject/onos-config/pkg/store/change/network"
	network0 "github.com/onosproject/onos-config/pkg/types/change/network"
	reflect "reflect"
)

// MockStore is a mock of Store interface
type MockNetworkChangesStore struct {
	ctrl     *gomock.Controller
	recorder *MockNetworkChangesStoreMockRecorder
}

// MockStoreMockRecorder is the mock recorder for MockStore
type MockNetworkChangesStoreMockRecorder struct {
	mock *MockNetworkChangesStore
}

// NewMockNetworkChangesStore creates a new mock instance
func NewMockNetworkChangesStore(ctrl *gomock.Controller) *MockNetworkChangesStore {
	mock := &MockNetworkChangesStore{ctrl: ctrl}
	mock.recorder = &MockNetworkChangesStoreMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use
func (m *MockNetworkChangesStore) EXPECT() *MockNetworkChangesStoreMockRecorder {
	return m.recorder
}

// Close mocks base method
func (m *MockNetworkChangesStore) Close() error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Close")
	ret0, _ := ret[0].(error)
	return ret0
}

// Close indicates an expected call of Close
func (mr *MockNetworkChangesStoreMockRecorder) Close() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Close", reflect.TypeOf((*MockNetworkChangesStore)(nil).Close))
}

// Get mocks base method
func (m *MockNetworkChangesStore) Get(id network0.ID) (*network0.NetworkChange, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Get", id)
	ret0, _ := ret[0].(*network0.NetworkChange)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// Get indicates an expected call of Get
func (mr *MockNetworkChangesStoreMockRecorder) Get(id interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Get", reflect.TypeOf((*MockNetworkChangesStore)(nil).Get), id)
}

// GetByIndex mocks base method
func (m *MockNetworkChangesStore) GetByIndex(index network0.Index) (*network0.NetworkChange, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetByIndex", index)
	ret0, _ := ret[0].(*network0.NetworkChange)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// GetByIndex indicates an expected call of GetByIndex
func (mr *MockNetworkChangesStoreMockRecorder) GetByIndex(index interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetByIndex", reflect.TypeOf((*MockNetworkChangesStore)(nil).GetByIndex), index)
}

// GetPrev mocks base method
func (m *MockNetworkChangesStore) GetPrev(index network0.Index) (*network0.NetworkChange, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetPrev", index)
	ret0, _ := ret[0].(*network0.NetworkChange)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// GetPrev indicates an expected call of GetPrev
func (mr *MockNetworkChangesStoreMockRecorder) GetPrev(index interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetPrev", reflect.TypeOf((*MockNetworkChangesStore)(nil).GetPrev), index)
}

// GetNext mocks base method
func (m *MockNetworkChangesStore) GetNext(index network0.Index) (*network0.NetworkChange, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetNext", index)
	ret0, _ := ret[0].(*network0.NetworkChange)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// GetNext indicates an expected call of GetNext
func (mr *MockNetworkChangesStoreMockRecorder) GetNext(index interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetNext", reflect.TypeOf((*MockNetworkChangesStore)(nil).GetNext), index)
}

// Create mocks base method
func (m *MockNetworkChangesStore) Create(config *network0.NetworkChange) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Create", config)
	ret0, _ := ret[0].(error)
	return ret0
}

// Create indicates an expected call of Create
func (mr *MockNetworkChangesStoreMockRecorder) Create(config interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Create", reflect.TypeOf((*MockNetworkChangesStore)(nil).Create), config)
}

// Update mocks base method
func (m *MockNetworkChangesStore) Update(config *network0.NetworkChange) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Update", config)
	ret0, _ := ret[0].(error)
	return ret0
}

// Update indicates an expected call of Update
func (mr *MockNetworkChangesStoreMockRecorder) Update(config interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Update", reflect.TypeOf((*MockNetworkChangesStore)(nil).Update), config)
}

// Delete mocks base method
func (m *MockNetworkChangesStore) Delete(config *network0.NetworkChange) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Delete", config)
	ret0, _ := ret[0].(error)
	return ret0
}

// Delete indicates an expected call of Delete
func (mr *MockNetworkChangesStoreMockRecorder) Delete(config interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Delete", reflect.TypeOf((*MockNetworkChangesStore)(nil).Delete), config)
}

// List mocks base method
func (m *MockNetworkChangesStore) List(arg0 chan<- *network0.NetworkChange) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "List", arg0)
	ret0, _ := ret[0].(error)
	return ret0
}

// List indicates an expected call of List
func (mr *MockNetworkChangesStoreMockRecorder) List(arg0 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "List", reflect.TypeOf((*MockNetworkChangesStore)(nil).List), arg0)
}

// Watch mocks base method
func (m *MockNetworkChangesStore) Watch(arg0 chan<- *network0.NetworkChange, arg1 ...network.WatchOption) error {
	m.ctrl.T.Helper()
	varargs := []interface{}{arg0}
	for _, a := range arg1 {
		varargs = append(varargs, a)
	}
	ret := m.ctrl.Call(m, "Watch", varargs...)
	ret0, _ := ret[0].(error)
	return ret0
}

// Watch indicates an expected call of Watch
func (mr *MockNetworkChangesStoreMockRecorder) Watch(arg0 interface{}, arg1 ...interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	varargs := append([]interface{}{arg0}, arg1...)
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Watch", reflect.TypeOf((*MockNetworkChangesStore)(nil).Watch), varargs...)
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
