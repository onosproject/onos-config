// Code generated by MockGen. DO NOT EDIT.
// Source: github.com/onosproject/onos-config/pkg/store/configuration (interfaces: Store)

// Package configuration is a generated GoMock package.
package configuration

import (
	context "context"
	reflect "reflect"

	gomock "github.com/golang/mock/gomock"
	v2 "github.com/onosproject/onos-api/go/onos/config/v2"
)

// MockStore is a mock of Store interface.
type MockStore struct {
	ctrl     *gomock.Controller
	recorder *MockStoreMockRecorder
}

// MockStoreMockRecorder is the mock recorder for MockStore.
type MockStoreMockRecorder struct {
	mock *MockStore
}

// NewMockStore creates a new mock instance.
func NewMockStore(ctrl *gomock.Controller) *MockStore {
	mock := &MockStore{ctrl: ctrl}
	mock.recorder = &MockStoreMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use.
func (m *MockStore) EXPECT() *MockStoreMockRecorder {
	return m.recorder
}

// Close mocks base method.
func (m *MockStore) Close(arg0 context.Context) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Close", arg0)
	ret0, _ := ret[0].(error)
	return ret0
}

// Close indicates an expected call of Close.
func (mr *MockStoreMockRecorder) Close(arg0 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Close", reflect.TypeOf((*MockStore)(nil).Close), arg0)
}

// Create mocks base method.
func (m *MockStore) Create(arg0 context.Context, arg1 *v2.Configuration) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Create", arg0, arg1)
	ret0, _ := ret[0].(error)
	return ret0
}

// Create indicates an expected call of Create.
func (mr *MockStoreMockRecorder) Create(arg0, arg1 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Create", reflect.TypeOf((*MockStore)(nil).Create), arg0, arg1)
}

// Get mocks base method.
func (m *MockStore) Get(arg0 context.Context, arg1 v2.ConfigurationID) (*v2.Configuration, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Get", arg0, arg1)
	ret0, _ := ret[0].(*v2.Configuration)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// Get indicates an expected call of Get.
func (mr *MockStoreMockRecorder) Get(arg0, arg1 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Get", reflect.TypeOf((*MockStore)(nil).Get), arg0, arg1)
}

// List mocks base method.
func (m *MockStore) List(arg0 context.Context) ([]*v2.Configuration, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "List", arg0)
	ret0, _ := ret[0].([]*v2.Configuration)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// List indicates an expected call of List.
func (mr *MockStoreMockRecorder) List(arg0 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "List", reflect.TypeOf((*MockStore)(nil).List), arg0)
}

// Update mocks base method.
func (m *MockStore) Update(arg0 context.Context, arg1 *v2.Configuration) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Update", arg0, arg1)
	ret0, _ := ret[0].(error)
	return ret0
}

// Update indicates an expected call of Update.
func (mr *MockStoreMockRecorder) Update(arg0, arg1 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Update", reflect.TypeOf((*MockStore)(nil).Update), arg0, arg1)
}

// UpdateStatus mocks base method.
func (m *MockStore) UpdateStatus(arg0 context.Context, arg1 *v2.Configuration) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "UpdateStatus", arg0, arg1)
	ret0, _ := ret[0].(error)
	return ret0
}

// UpdateStatus indicates an expected call of UpdateStatus.
func (mr *MockStoreMockRecorder) UpdateStatus(arg0, arg1 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "UpdateStatus", reflect.TypeOf((*MockStore)(nil).UpdateStatus), arg0, arg1)
}

// Watch mocks base method.
func (m *MockStore) Watch(arg0 context.Context, arg1 chan<- v2.ConfigurationEvent, arg2 ...WatchOption) error {
	m.ctrl.T.Helper()
	varargs := []interface{}{arg0, arg1}
	for _, a := range arg2 {
		varargs = append(varargs, a)
	}
	ret := m.ctrl.Call(m, "Watch", varargs...)
	ret0, _ := ret[0].(error)
	return ret0
}

// Watch indicates an expected call of Watch.
func (mr *MockStoreMockRecorder) Watch(arg0, arg1 interface{}, arg2 ...interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	varargs := append([]interface{}{arg0, arg1}, arg2...)
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Watch", reflect.TypeOf((*MockStore)(nil).Watch), varargs...)
}