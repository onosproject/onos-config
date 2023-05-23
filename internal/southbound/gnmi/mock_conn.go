// Code generated by MockGen. DO NOT EDIT.
// Source: github.com/onosproject/onos-config/pkg/southbound/gnmi (interfaces: ConnManager,Conn)

// Package gnmi is a generated GoMock package.
package gnmi

import (
	context "context"
	reflect "reflect"

	gomock "github.com/golang/mock/gomock"
	topo "github.com/onosproject/onos-api/go/onos/topo"
	gnmi "github.com/onosproject/onos-config/pkg/southbound/gnmi"
	client "github.com/openconfig/gnmi/client"
	gnmi0 "github.com/openconfig/gnmi/proto/gnmi"
)

// MockConnManager is a mock of ConnManager interface.
type MockConnManager struct {
	ctrl     *gomock.Controller
	recorder *MockConnManagerMockRecorder
}

// MockConnManagerMockRecorder is the mock recorder for MockConnManager.
type MockConnManagerMockRecorder struct {
	mock *MockConnManager
}

// NewMockConnManager creates a new mock instance.
func NewMockConnManager(ctrl *gomock.Controller) *MockConnManager {
	mock := &MockConnManager{ctrl: ctrl}
	mock.recorder = &MockConnManagerMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use.
func (m *MockConnManager) EXPECT() *MockConnManagerMockRecorder {
	return m.recorder
}

// Connect mocks base method.
func (m *MockConnManager) Connect(arg0 context.Context, arg1 *topo.Object) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Connect", arg0, arg1)
	ret0, _ := ret[0].(error)
	return ret0
}

// Connect indicates an expected call of Connect.
func (mr *MockConnManagerMockRecorder) Connect(arg0, arg1 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Connect", reflect.TypeOf((*MockConnManager)(nil).Connect), arg0, arg1)
}

// Disconnect mocks base method.
func (m *MockConnManager) Disconnect(arg0 context.Context, arg1 topo.ID) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Disconnect", arg0, arg1)
	ret0, _ := ret[0].(error)
	return ret0
}

// Disconnect indicates an expected call of Disconnect.
func (mr *MockConnManagerMockRecorder) Disconnect(arg0, arg1 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Disconnect", reflect.TypeOf((*MockConnManager)(nil).Disconnect), arg0, arg1)
}

// Get mocks base method.
func (m *MockConnManager) Get(arg0 context.Context, arg1 gnmi.ConnID) (gnmi.Conn, bool) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Get", arg0, arg1)
	ret0, _ := ret[0].(gnmi.Conn)
	ret1, _ := ret[1].(bool)
	return ret0, ret1
}

// Get indicates an expected call of Get.
func (mr *MockConnManagerMockRecorder) Get(arg0, arg1 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Get", reflect.TypeOf((*MockConnManager)(nil).Get), arg0, arg1)
}

// GetByTarget mocks base method.
func (m *MockConnManager) GetByTarget(arg0 context.Context, arg1 topo.ID) (gnmi.Client, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetByTarget", arg0, arg1)
	ret0, _ := ret[0].(gnmi.Client)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// GetByTarget indicates an expected call of GetByTarget.
func (mr *MockConnManagerMockRecorder) GetByTarget(arg0, arg1 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetByTarget", reflect.TypeOf((*MockConnManager)(nil).GetByTarget), arg0, arg1)
}

// Watch mocks base method.
func (m *MockConnManager) Watch(arg0 context.Context, arg1 chan<- gnmi.Conn) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Watch", arg0, arg1)
	ret0, _ := ret[0].(error)
	return ret0
}

// Watch indicates an expected call of Watch.
func (mr *MockConnManagerMockRecorder) Watch(arg0, arg1 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Watch", reflect.TypeOf((*MockConnManager)(nil).Watch), arg0, arg1)
}

// MockConn is a mock of Conn interface.
type MockConn struct {
	ctrl     *gomock.Controller
	recorder *MockConnMockRecorder
}

// MockConnMockRecorder is the mock recorder for MockConn.
type MockConnMockRecorder struct {
	mock *MockConn
}

// NewMockConn creates a new mock instance.
func NewMockConn(ctrl *gomock.Controller) *MockConn {
	mock := &MockConn{ctrl: ctrl}
	mock.recorder = &MockConnMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use.
func (m *MockConn) EXPECT() *MockConnMockRecorder {
	return m.recorder
}

// Capabilities mocks base method.
func (m *MockConn) Capabilities(arg0 context.Context, arg1 *gnmi0.CapabilityRequest) (*gnmi0.CapabilityResponse, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Capabilities", arg0, arg1)
	ret0, _ := ret[0].(*gnmi0.CapabilityResponse)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// Capabilities indicates an expected call of Capabilities.
func (mr *MockConnMockRecorder) Capabilities(arg0, arg1 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Capabilities", reflect.TypeOf((*MockConn)(nil).Capabilities), arg0, arg1)
}

// CapabilitiesWithString mocks base method.
func (m *MockConn) CapabilitiesWithString(arg0 context.Context, arg1 string) (*gnmi0.CapabilityResponse, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "CapabilitiesWithString", arg0, arg1)
	ret0, _ := ret[0].(*gnmi0.CapabilityResponse)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// CapabilitiesWithString indicates an expected call of CapabilitiesWithString.
func (mr *MockConnMockRecorder) CapabilitiesWithString(arg0, arg1 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "CapabilitiesWithString", reflect.TypeOf((*MockConn)(nil).CapabilitiesWithString), arg0, arg1)
}

// Close mocks base method.
func (m *MockConn) Close() error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Close")
	ret0, _ := ret[0].(error)
	return ret0
}

// Close indicates an expected call of Close.
func (mr *MockConnMockRecorder) Close() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Close", reflect.TypeOf((*MockConn)(nil).Close))
}

// Get mocks base method.
func (m *MockConn) Get(arg0 context.Context, arg1 *gnmi0.GetRequest) (*gnmi0.GetResponse, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Get", arg0, arg1)
	ret0, _ := ret[0].(*gnmi0.GetResponse)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// Get indicates an expected call of Get.
func (mr *MockConnMockRecorder) Get(arg0, arg1 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Get", reflect.TypeOf((*MockConn)(nil).Get), arg0, arg1)
}

// GetWithString mocks base method.
func (m *MockConn) GetWithString(arg0 context.Context, arg1 string) (*gnmi0.GetResponse, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetWithString", arg0, arg1)
	ret0, _ := ret[0].(*gnmi0.GetResponse)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// GetWithString indicates an expected call of GetWithString.
func (mr *MockConnMockRecorder) GetWithString(arg0, arg1 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetWithString", reflect.TypeOf((*MockConn)(nil).GetWithString), arg0, arg1)
}

// ID mocks base method.
func (m *MockConn) ID() gnmi.ConnID {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "ID")
	ret0, _ := ret[0].(gnmi.ConnID)
	return ret0
}

// ID indicates an expected call of ID.
func (mr *MockConnMockRecorder) ID() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "ID", reflect.TypeOf((*MockConn)(nil).ID))
}

// Poll mocks base method.
func (m *MockConn) Poll() error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Poll")
	ret0, _ := ret[0].(error)
	return ret0
}

// Poll indicates an expected call of Poll.
func (mr *MockConnMockRecorder) Poll() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Poll", reflect.TypeOf((*MockConn)(nil).Poll))
}

// Set mocks base method.
func (m *MockConn) Set(arg0 context.Context, arg1 *gnmi0.SetRequest) (*gnmi0.SetResponse, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Set", arg0, arg1)
	ret0, _ := ret[0].(*gnmi0.SetResponse)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// Set indicates an expected call of Set.
func (mr *MockConnMockRecorder) Set(arg0, arg1 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Set", reflect.TypeOf((*MockConn)(nil).Set), arg0, arg1)
}

// SetWithString mocks base method.
func (m *MockConn) SetWithString(arg0 context.Context, arg1 string) (*gnmi0.SetResponse, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "SetWithString", arg0, arg1)
	ret0, _ := ret[0].(*gnmi0.SetResponse)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// SetWithString indicates an expected call of SetWithString.
func (mr *MockConnMockRecorder) SetWithString(arg0, arg1 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "SetWithString", reflect.TypeOf((*MockConn)(nil).SetWithString), arg0, arg1)
}

// Subscribe mocks base method.
func (m *MockConn) Subscribe(arg0 context.Context, arg1 client.Query) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Subscribe", arg0, arg1)
	ret0, _ := ret[0].(error)
	return ret0
}

// Subscribe indicates an expected call of Subscribe.
func (mr *MockConnMockRecorder) Subscribe(arg0, arg1 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Subscribe", reflect.TypeOf((*MockConn)(nil).Subscribe), arg0, arg1)
}

// TargetID mocks base method.
func (m *MockConn) TargetID() topo.ID {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "TargetID")
	ret0, _ := ret[0].(topo.ID)
	return ret0
}

// TargetID indicates an expected call of TargetID.
func (mr *MockConnMockRecorder) TargetID() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "TargetID", reflect.TypeOf((*MockConn)(nil).TargetID))
}