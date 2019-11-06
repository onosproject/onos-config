// Code generated by MockGen. DO NOT EDIT.
// Source: pkg/store/snapshot/network/store.go

// Package store is a generated GoMock package.
package store

import (
	gomock "github.com/golang/mock/gomock"
	network "github.com/onosproject/onos-config/api/types/snapshot/network"
	stream "github.com/onosproject/onos-config/pkg/store/stream"
	reflect "reflect"
)

// MockNetworkSnapshotStore is a mock of Store interface
type MockNetworkSnapshotStore struct {
	ctrl     *gomock.Controller
	recorder *MockNetworkSnapshotStoreMockRecorder
}

// MockNetworkSnapshotStoreMockRecorder is the mock recorder for MockStore
type MockNetworkSnapshotStoreMockRecorder struct {
	mock *MockNetworkSnapshotStore
}

// NewMockNetworkSnapshotStore creates a new mock instance
func NewMockNetworkSnapshotStore(ctrl *gomock.Controller) *MockNetworkSnapshotStore {
	mock := &MockNetworkSnapshotStore{ctrl: ctrl}
	mock.recorder = &MockNetworkSnapshotStoreMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use
func (m *MockNetworkSnapshotStore) EXPECT() *MockNetworkSnapshotStoreMockRecorder {
	return m.recorder
}

// Close mocks base method
func (m *MockNetworkSnapshotStore) Close() error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Close")
	ret0, _ := ret[0].(error)
	return ret0
}

// Close indicates an expected call of Close
func (mr *MockNetworkSnapshotStoreMockRecorder) Close() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Close", reflect.TypeOf((*MockNetworkSnapshotStore)(nil).Close))
}

// Get mocks base method
func (m *MockNetworkSnapshotStore) Get(id network.ID) (*network.NetworkSnapshot, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Get", id)
	ret0, _ := ret[0].(*network.NetworkSnapshot)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// Get indicates an expected call of Get
func (mr *MockNetworkSnapshotStoreMockRecorder) Get(id interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Get", reflect.TypeOf((*MockNetworkSnapshotStore)(nil).Get), id)
}

// GetByIndex mocks base method
func (m *MockNetworkSnapshotStore) GetByIndex(index network.Index) (*network.NetworkSnapshot, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetByIndex", index)
	ret0, _ := ret[0].(*network.NetworkSnapshot)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// GetByIndex indicates an expected call of GetByIndex
func (mr *MockNetworkSnapshotStoreMockRecorder) GetByIndex(index interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetByIndex", reflect.TypeOf((*MockNetworkSnapshotStore)(nil).GetByIndex), index)
}

// Create mocks base method
func (m *MockNetworkSnapshotStore) Create(snapshot *network.NetworkSnapshot) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Create", snapshot)
	ret0, _ := ret[0].(error)
	return ret0
}

// Create indicates an expected call of Create
func (mr *MockNetworkSnapshotStoreMockRecorder) Create(snapshot interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Create", reflect.TypeOf((*MockNetworkSnapshotStore)(nil).Create), snapshot)
}

// Update mocks base method
func (m *MockNetworkSnapshotStore) Update(snapshot *network.NetworkSnapshot) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Update", snapshot)
	ret0, _ := ret[0].(error)
	return ret0
}

// Update indicates an expected call of Update
func (mr *MockNetworkSnapshotStoreMockRecorder) Update(snapshot interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Update", reflect.TypeOf((*MockNetworkSnapshotStore)(nil).Update), snapshot)
}

// Delete mocks base method
func (m *MockNetworkSnapshotStore) Delete(snapshot *network.NetworkSnapshot) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Delete", snapshot)
	ret0, _ := ret[0].(error)
	return ret0
}

// Delete indicates an expected call of Delete
func (mr *MockNetworkSnapshotStoreMockRecorder) Delete(snapshot interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Delete", reflect.TypeOf((*MockNetworkSnapshotStore)(nil).Delete), snapshot)
}

// List mocks base method
func (m *MockNetworkSnapshotStore) List(arg0 chan<- *network.NetworkSnapshot) (stream.Context, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "List", arg0)
	ret0, _ := ret[0].(stream.Context)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// List indicates an expected call of List
func (mr *MockNetworkSnapshotStoreMockRecorder) List(arg0 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "List", reflect.TypeOf((*MockNetworkSnapshotStore)(nil).List), arg0)
}

// Watch mocks base method
func (m *MockNetworkSnapshotStore) Watch(arg0 chan<- stream.Event) (stream.Context, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Watch", arg0)
	ret0, _ := ret[0].(stream.Context)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// Watch indicates an expected call of Watch
func (mr *MockNetworkSnapshotStoreMockRecorder) Watch(arg0 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Watch", reflect.TypeOf((*MockNetworkSnapshotStore)(nil).Watch), arg0)
}
