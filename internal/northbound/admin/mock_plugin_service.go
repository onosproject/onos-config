// Code generated by MockGen. DO NOT EDIT.
// Source: registry.go

// Package test is a generated GoMock package.
package admin

import (
	context "context"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
	reflect "reflect"

	gomock "github.com/golang/mock/gomock"
	admin "github.com/onosproject/onos-api/go/onos/config/admin"
)

// MockModelPluginServiceClient is a mock of ModelPluginServiceClient interface.
type MockModelPluginServiceClient struct {
	ctrl     *gomock.Controller
	recorder *MockModelPluginServiceClientMockRecorder
}

// MockModelPluginServiceClientMockRecorder is the mock recorder for MockModelPluginServiceClient.
type MockModelPluginServiceClientMockRecorder struct {
	mock *MockModelPluginServiceClient
}

// NewMockModelPluginServiceClient creates a new mock instance.
func NewMockModelPluginServiceClient(ctrl *gomock.Controller) *MockModelPluginServiceClient {
	mock := &MockModelPluginServiceClient{ctrl: ctrl}
	mock.recorder = &MockModelPluginServiceClientMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use.
func (m *MockModelPluginServiceClient) EXPECT() *MockModelPluginServiceClientMockRecorder {
	return m.recorder
}

// GetModelInfo mocks base method.
func (m *MockModelPluginServiceClient) GetModelInfo(ctx context.Context, in *admin.ModelInfoRequest, opts ...grpc.CallOption) (*admin.ModelInfoResponse, error) {
	m.ctrl.T.Helper()
	varargs := []interface{}{ctx, in}
	for _, a := range opts {
		varargs = append(varargs, a)
	}
	ret := m.ctrl.Call(m, "GetModelInfo", varargs...)
	ret0, _ := ret[0].(*admin.ModelInfoResponse)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// GetModelInfo indicates an expected call of GetModelInfo.
func (mr *MockModelPluginServiceClientMockRecorder) GetModelInfo(ctx, in interface{}, opts ...interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	varargs := append([]interface{}{ctx, in}, opts...)
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetModelInfo", reflect.TypeOf((*MockModelPluginServiceClient)(nil).GetModelInfo), varargs...)
}

// GetPathValues mocks base method.
func (m *MockModelPluginServiceClient) GetPathValues(ctx context.Context, in *admin.PathValuesRequest, opts ...grpc.CallOption) (*admin.PathValuesResponse, error) {
	m.ctrl.T.Helper()
	varargs := []interface{}{ctx, in}
	for _, a := range opts {
		varargs = append(varargs, a)
	}
	ret := m.ctrl.Call(m, "GetPathValues", varargs...)
	ret0, _ := ret[0].(*admin.PathValuesResponse)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// GetPathValues indicates an expected call of GetPathValues.
func (mr *MockModelPluginServiceClientMockRecorder) GetPathValues(ctx, in interface{}, opts ...interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	varargs := append([]interface{}{ctx, in}, opts...)
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetPathValues", reflect.TypeOf((*MockModelPluginServiceClient)(nil).GetPathValues), varargs...)
}

// GetValueSelection mocks base method.
func (m *MockModelPluginServiceClient) GetValueSelection(ctx context.Context, in *admin.ValueSelectionRequest, opts ...grpc.CallOption) (*admin.ValueSelectionResponse, error) {
	m.ctrl.T.Helper()
	varargs := []interface{}{ctx, in}
	for _, a := range opts {
		varargs = append(varargs, a)
	}
	ret := m.ctrl.Call(m, "GetValueSelection", varargs...)
	ret0, _ := ret[0].(*admin.ValueSelectionResponse)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// GetValueSelection indicates an expected call of GetValueSelection.
func (mr *MockModelPluginServiceClientMockRecorder) GetValueSelection(ctx, in interface{}, opts ...interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	varargs := append([]interface{}{ctx, in}, opts...)
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetValueSelection", reflect.TypeOf((*MockModelPluginServiceClient)(nil).GetValueSelection), varargs...)
}

// GetValueSelectionChunked mocks base method.
func (m *MockModelPluginServiceClient) GetValueSelectionChunked(ctx context.Context, opts ...grpc.CallOption) (admin.ModelPluginService_GetValueSelectionChunkedClient, error) {
	m.ctrl.T.Helper()
	varargs := []interface{}{ctx}
	for _, a := range opts {
		varargs = append(varargs, a)
	}
	ret := m.ctrl.Call(m, "GetValueSelectionChunked", varargs...)
	ret0, _ := ret[0].(admin.ModelPluginService_GetValueSelectionChunkedClient)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// GetValueSelectionChunked indicates an expected call of GetValueSelectionChunked.
func (mr *MockModelPluginServiceClientMockRecorder) GetValueSelectionChunked(ctx interface{}, opts ...interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	varargs := append([]interface{}{ctx}, opts...)
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetValueSelectionChunked", reflect.TypeOf((*MockModelPluginServiceClient)(nil).GetValueSelectionChunked), varargs...)
}

// ValidateConfig mocks base method.
func (m *MockModelPluginServiceClient) ValidateConfig(ctx context.Context, in *admin.ValidateConfigRequest, opts ...grpc.CallOption) (*admin.ValidateConfigResponse, error) {
	m.ctrl.T.Helper()
	varargs := []interface{}{ctx, in}
	for _, a := range opts {
		varargs = append(varargs, a)
	}
	ret := m.ctrl.Call(m, "ValidateConfig", varargs...)
	ret0, _ := ret[0].(*admin.ValidateConfigResponse)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// ValidateConfig indicates an expected call of ValidateConfig.
func (mr *MockModelPluginServiceClientMockRecorder) ValidateConfig(ctx, in interface{}, opts ...interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	varargs := append([]interface{}{ctx, in}, opts...)
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "ValidateConfig", reflect.TypeOf((*MockModelPluginServiceClient)(nil).ValidateConfig), varargs...)
}

// ValidateConfigChunked mocks base method.
func (m *MockModelPluginServiceClient) ValidateConfigChunked(ctx context.Context, opts ...grpc.CallOption) (admin.ModelPluginService_ValidateConfigChunkedClient, error) {
	m.ctrl.T.Helper()
	varargs := []interface{}{ctx}
	for _, a := range opts {
		varargs = append(varargs, a)
	}
	ret := m.ctrl.Call(m, "ValidateConfigChunked", varargs...)
	ret0, _ := ret[0].(admin.ModelPluginService_ValidateConfigChunkedClient)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// ValidateConfigChunked indicates an expected call of ValidateConfigChunked.
func (mr *MockModelPluginServiceClientMockRecorder) ValidateConfigChunked(ctx interface{}, opts ...interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	varargs := append([]interface{}{ctx}, opts...)
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "ValidateConfigChunked", reflect.TypeOf((*MockModelPluginServiceClient)(nil).ValidateConfigChunked), varargs...)
}

// MockModelPluginService_ValidateConfigChunkedClient is a mock of ModelPluginService_ValidateConfigChunkedClient interface.
type MockModelPluginService_ValidateConfigChunkedClient struct {
	ctrl     *gomock.Controller
	recorder *MockModelPluginService_ValidateConfigChunkedClientMockRecorder
}

// MockModelPluginService_ValidateConfigChunkedClientMockRecorder is the mock recorder for MockModelPluginService_ValidateConfigChunkedClient.
type MockModelPluginService_ValidateConfigChunkedClientMockRecorder struct {
	mock *MockModelPluginService_ValidateConfigChunkedClient
}

// NewMockModelPluginService_ValidateConfigChunkedClient creates a new mock instance.
func NewMockModelPluginService_ValidateConfigChunkedClient(ctrl *gomock.Controller) *MockModelPluginService_ValidateConfigChunkedClient {
	mock := &MockModelPluginService_ValidateConfigChunkedClient{ctrl: ctrl}
	mock.recorder = &MockModelPluginService_ValidateConfigChunkedClientMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use.
func (m *MockModelPluginService_ValidateConfigChunkedClient) EXPECT() *MockModelPluginService_ValidateConfigChunkedClientMockRecorder {
	return m.recorder
}

// CloseAndRecv mocks base method.
func (m *MockModelPluginService_ValidateConfigChunkedClient) CloseAndRecv() (*admin.ValidateConfigResponse, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "CloseAndRecv")
	ret0, _ := ret[0].(*admin.ValidateConfigResponse)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// CloseAndRecv indicates an expected call of CloseAndRecv.
func (mr *MockModelPluginService_ValidateConfigChunkedClientMockRecorder) CloseAndRecv() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "CloseAndRecv", reflect.TypeOf((*MockModelPluginService_ValidateConfigChunkedClient)(nil).CloseAndRecv))
}

// CloseSend mocks base method.
func (m *MockModelPluginService_ValidateConfigChunkedClient) CloseSend() error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "CloseSend")
	ret0, _ := ret[0].(error)
	return ret0
}

// CloseSend indicates an expected call of CloseSend.
func (mr *MockModelPluginService_ValidateConfigChunkedClientMockRecorder) CloseSend() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "CloseSend", reflect.TypeOf((*MockModelPluginService_ValidateConfigChunkedClient)(nil).CloseSend))
}

// Context mocks base method.
func (m *MockModelPluginService_ValidateConfigChunkedClient) Context() context.Context {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Context")
	ret0, _ := ret[0].(context.Context)
	return ret0
}

// Context indicates an expected call of Context.
func (mr *MockModelPluginService_ValidateConfigChunkedClientMockRecorder) Context() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Context", reflect.TypeOf((*MockModelPluginService_ValidateConfigChunkedClient)(nil).Context))
}

// Header mocks base method.
func (m *MockModelPluginService_ValidateConfigChunkedClient) Header() (metadata.MD, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Header")
	ret0, _ := ret[0].(metadata.MD)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// Header indicates an expected call of Header.
func (mr *MockModelPluginService_ValidateConfigChunkedClientMockRecorder) Header() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Header", reflect.TypeOf((*MockModelPluginService_ValidateConfigChunkedClient)(nil).Header))
}

// RecvMsg mocks base method.
func (m_2 *MockModelPluginService_ValidateConfigChunkedClient) RecvMsg(m interface{}) error {
	m_2.ctrl.T.Helper()
	ret := m_2.ctrl.Call(m_2, "RecvMsg", m)
	ret0, _ := ret[0].(error)
	return ret0
}

// RecvMsg indicates an expected call of RecvMsg.
func (mr *MockModelPluginService_ValidateConfigChunkedClientMockRecorder) RecvMsg(m interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "RecvMsg", reflect.TypeOf((*MockModelPluginService_ValidateConfigChunkedClient)(nil).RecvMsg), m)
}

// Send mocks base method.
func (m *MockModelPluginService_ValidateConfigChunkedClient) Send(arg0 *admin.ValidateConfigRequestChunk) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Send", arg0)
	ret0, _ := ret[0].(error)
	return ret0
}

// Send indicates an expected call of Send.
func (mr *MockModelPluginService_ValidateConfigChunkedClientMockRecorder) Send(arg0 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Send", reflect.TypeOf((*MockModelPluginService_ValidateConfigChunkedClient)(nil).Send), arg0)
}

// SendMsg mocks base method.
func (m_2 *MockModelPluginService_ValidateConfigChunkedClient) SendMsg(m interface{}) error {
	m_2.ctrl.T.Helper()
	ret := m_2.ctrl.Call(m_2, "SendMsg", m)
	ret0, _ := ret[0].(error)
	return ret0
}

// SendMsg indicates an expected call of SendMsg.
func (mr *MockModelPluginService_ValidateConfigChunkedClientMockRecorder) SendMsg(m interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "SendMsg", reflect.TypeOf((*MockModelPluginService_ValidateConfigChunkedClient)(nil).SendMsg), m)
}

// Trailer mocks base method.
func (m *MockModelPluginService_ValidateConfigChunkedClient) Trailer() metadata.MD {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Trailer")
	ret0, _ := ret[0].(metadata.MD)
	return ret0
}

// Trailer indicates an expected call of Trailer.
func (mr *MockModelPluginService_ValidateConfigChunkedClientMockRecorder) Trailer() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Trailer", reflect.TypeOf((*MockModelPluginService_ValidateConfigChunkedClient)(nil).Trailer))
}

// MockModelPluginService_GetValueSelectionChunkedClient is a mock of ModelPluginService_GetValueSelectionChunkedClient interface.
type MockModelPluginService_GetValueSelectionChunkedClient struct {
	ctrl     *gomock.Controller
	recorder *MockModelPluginService_GetValueSelectionChunkedClientMockRecorder
}

// MockModelPluginService_GetValueSelectionChunkedClientMockRecorder is the mock recorder for MockModelPluginService_GetValueSelectionChunkedClient.
type MockModelPluginService_GetValueSelectionChunkedClientMockRecorder struct {
	mock *MockModelPluginService_GetValueSelectionChunkedClient
}

// NewMockModelPluginService_GetValueSelectionChunkedClient creates a new mock instance.
func NewMockModelPluginService_GetValueSelectionChunkedClient(ctrl *gomock.Controller) *MockModelPluginService_GetValueSelectionChunkedClient {
	mock := &MockModelPluginService_GetValueSelectionChunkedClient{ctrl: ctrl}
	mock.recorder = &MockModelPluginService_GetValueSelectionChunkedClientMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use.
func (m *MockModelPluginService_GetValueSelectionChunkedClient) EXPECT() *MockModelPluginService_GetValueSelectionChunkedClientMockRecorder {
	return m.recorder
}

// CloseAndRecv mocks base method.
func (m *MockModelPluginService_GetValueSelectionChunkedClient) CloseAndRecv() (*admin.ValueSelectionResponse, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "CloseAndRecv")
	ret0, _ := ret[0].(*admin.ValueSelectionResponse)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// CloseAndRecv indicates an expected call of CloseAndRecv.
func (mr *MockModelPluginService_GetValueSelectionChunkedClientMockRecorder) CloseAndRecv() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "CloseAndRecv", reflect.TypeOf((*MockModelPluginService_GetValueSelectionChunkedClient)(nil).CloseAndRecv))
}

// CloseSend mocks base method.
func (m *MockModelPluginService_GetValueSelectionChunkedClient) CloseSend() error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "CloseSend")
	ret0, _ := ret[0].(error)
	return ret0
}

// CloseSend indicates an expected call of CloseSend.
func (mr *MockModelPluginService_GetValueSelectionChunkedClientMockRecorder) CloseSend() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "CloseSend", reflect.TypeOf((*MockModelPluginService_GetValueSelectionChunkedClient)(nil).CloseSend))
}

// Context mocks base method.
func (m *MockModelPluginService_GetValueSelectionChunkedClient) Context() context.Context {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Context")
	ret0, _ := ret[0].(context.Context)
	return ret0
}

// Context indicates an expected call of Context.
func (mr *MockModelPluginService_GetValueSelectionChunkedClientMockRecorder) Context() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Context", reflect.TypeOf((*MockModelPluginService_GetValueSelectionChunkedClient)(nil).Context))
}

// Header mocks base method.
func (m *MockModelPluginService_GetValueSelectionChunkedClient) Header() (metadata.MD, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Header")
	ret0, _ := ret[0].(metadata.MD)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// Header indicates an expected call of Header.
func (mr *MockModelPluginService_GetValueSelectionChunkedClientMockRecorder) Header() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Header", reflect.TypeOf((*MockModelPluginService_GetValueSelectionChunkedClient)(nil).Header))
}

// RecvMsg mocks base method.
func (m_2 *MockModelPluginService_GetValueSelectionChunkedClient) RecvMsg(m interface{}) error {
	m_2.ctrl.T.Helper()
	ret := m_2.ctrl.Call(m_2, "RecvMsg", m)
	ret0, _ := ret[0].(error)
	return ret0
}

// RecvMsg indicates an expected call of RecvMsg.
func (mr *MockModelPluginService_GetValueSelectionChunkedClientMockRecorder) RecvMsg(m interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "RecvMsg", reflect.TypeOf((*MockModelPluginService_GetValueSelectionChunkedClient)(nil).RecvMsg), m)
}

// Send mocks base method.
func (m *MockModelPluginService_GetValueSelectionChunkedClient) Send(arg0 *admin.ValueSelectionRequestChunk) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Send", arg0)
	ret0, _ := ret[0].(error)
	return ret0
}

// Send indicates an expected call of Send.
func (mr *MockModelPluginService_GetValueSelectionChunkedClientMockRecorder) Send(arg0 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Send", reflect.TypeOf((*MockModelPluginService_GetValueSelectionChunkedClient)(nil).Send), arg0)
}

// SendMsg mocks base method.
func (m_2 *MockModelPluginService_GetValueSelectionChunkedClient) SendMsg(m interface{}) error {
	m_2.ctrl.T.Helper()
	ret := m_2.ctrl.Call(m_2, "SendMsg", m)
	ret0, _ := ret[0].(error)
	return ret0
}

// SendMsg indicates an expected call of SendMsg.
func (mr *MockModelPluginService_GetValueSelectionChunkedClientMockRecorder) SendMsg(m interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "SendMsg", reflect.TypeOf((*MockModelPluginService_GetValueSelectionChunkedClient)(nil).SendMsg), m)
}

// Trailer mocks base method.
func (m *MockModelPluginService_GetValueSelectionChunkedClient) Trailer() metadata.MD {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Trailer")
	ret0, _ := ret[0].(metadata.MD)
	return ret0
}

// Trailer indicates an expected call of Trailer.
func (mr *MockModelPluginService_GetValueSelectionChunkedClientMockRecorder) Trailer() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Trailer", reflect.TypeOf((*MockModelPluginService_GetValueSelectionChunkedClient)(nil).Trailer))
}

// MockModelPluginServiceServer is a mock of ModelPluginServiceServer interface.
type MockModelPluginServiceServer struct {
	ctrl     *gomock.Controller
	recorder *MockModelPluginServiceServerMockRecorder
}

// MockModelPluginServiceServerMockRecorder is the mock recorder for MockModelPluginServiceServer.
type MockModelPluginServiceServerMockRecorder struct {
	mock *MockModelPluginServiceServer
}

// NewMockModelPluginServiceServer creates a new mock instance.
func NewMockModelPluginServiceServer(ctrl *gomock.Controller) *MockModelPluginServiceServer {
	mock := &MockModelPluginServiceServer{ctrl: ctrl}
	mock.recorder = &MockModelPluginServiceServerMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use.
func (m *MockModelPluginServiceServer) EXPECT() *MockModelPluginServiceServerMockRecorder {
	return m.recorder
}

// GetModelInfo mocks base method.
func (m *MockModelPluginServiceServer) GetModelInfo(arg0 context.Context, arg1 *admin.ModelInfoRequest) (*admin.ModelInfoResponse, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetModelInfo", arg0, arg1)
	ret0, _ := ret[0].(*admin.ModelInfoResponse)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// GetModelInfo indicates an expected call of GetModelInfo.
func (mr *MockModelPluginServiceServerMockRecorder) GetModelInfo(arg0, arg1 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetModelInfo", reflect.TypeOf((*MockModelPluginServiceServer)(nil).GetModelInfo), arg0, arg1)
}

// GetPathValues mocks base method.
func (m *MockModelPluginServiceServer) GetPathValues(arg0 context.Context, arg1 *admin.PathValuesRequest) (*admin.PathValuesResponse, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetPathValues", arg0, arg1)
	ret0, _ := ret[0].(*admin.PathValuesResponse)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// GetPathValues indicates an expected call of GetPathValues.
func (mr *MockModelPluginServiceServerMockRecorder) GetPathValues(arg0, arg1 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetPathValues", reflect.TypeOf((*MockModelPluginServiceServer)(nil).GetPathValues), arg0, arg1)
}

// GetValueSelection mocks base method.
func (m *MockModelPluginServiceServer) GetValueSelection(arg0 context.Context, arg1 *admin.ValueSelectionRequest) (*admin.ValueSelectionResponse, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetValueSelection", arg0, arg1)
	ret0, _ := ret[0].(*admin.ValueSelectionResponse)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// GetValueSelection indicates an expected call of GetValueSelection.
func (mr *MockModelPluginServiceServerMockRecorder) GetValueSelection(arg0, arg1 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetValueSelection", reflect.TypeOf((*MockModelPluginServiceServer)(nil).GetValueSelection), arg0, arg1)
}

// GetValueSelectionChunked mocks base method.
func (m *MockModelPluginServiceServer) GetValueSelectionChunked(arg0 admin.ModelPluginService_GetValueSelectionChunkedServer) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetValueSelectionChunked", arg0)
	ret0, _ := ret[0].(error)
	return ret0
}

// GetValueSelectionChunked indicates an expected call of GetValueSelectionChunked.
func (mr *MockModelPluginServiceServerMockRecorder) GetValueSelectionChunked(arg0 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetValueSelectionChunked", reflect.TypeOf((*MockModelPluginServiceServer)(nil).GetValueSelectionChunked), arg0)
}

// ValidateConfig mocks base method.
func (m *MockModelPluginServiceServer) ValidateConfig(arg0 context.Context, arg1 *admin.ValidateConfigRequest) (*admin.ValidateConfigResponse, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "ValidateConfig", arg0, arg1)
	ret0, _ := ret[0].(*admin.ValidateConfigResponse)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// ValidateConfig indicates an expected call of ValidateConfig.
func (mr *MockModelPluginServiceServerMockRecorder) ValidateConfig(arg0, arg1 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "ValidateConfig", reflect.TypeOf((*MockModelPluginServiceServer)(nil).ValidateConfig), arg0, arg1)
}

// ValidateConfigChunked mocks base method.
func (m *MockModelPluginServiceServer) ValidateConfigChunked(arg0 admin.ModelPluginService_ValidateConfigChunkedServer) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "ValidateConfigChunked", arg0)
	ret0, _ := ret[0].(error)
	return ret0
}

// ValidateConfigChunked indicates an expected call of ValidateConfigChunked.
func (mr *MockModelPluginServiceServerMockRecorder) ValidateConfigChunked(arg0 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "ValidateConfigChunked", reflect.TypeOf((*MockModelPluginServiceServer)(nil).ValidateConfigChunked), arg0)
}

// MockModelPluginService_ValidateConfigChunkedServer is a mock of ModelPluginService_ValidateConfigChunkedServer interface.
type MockModelPluginService_ValidateConfigChunkedServer struct {
	ctrl     *gomock.Controller
	recorder *MockModelPluginService_ValidateConfigChunkedServerMockRecorder
}

// MockModelPluginService_ValidateConfigChunkedServerMockRecorder is the mock recorder for MockModelPluginService_ValidateConfigChunkedServer.
type MockModelPluginService_ValidateConfigChunkedServerMockRecorder struct {
	mock *MockModelPluginService_ValidateConfigChunkedServer
}

// NewMockModelPluginService_ValidateConfigChunkedServer creates a new mock instance.
func NewMockModelPluginService_ValidateConfigChunkedServer(ctrl *gomock.Controller) *MockModelPluginService_ValidateConfigChunkedServer {
	mock := &MockModelPluginService_ValidateConfigChunkedServer{ctrl: ctrl}
	mock.recorder = &MockModelPluginService_ValidateConfigChunkedServerMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use.
func (m *MockModelPluginService_ValidateConfigChunkedServer) EXPECT() *MockModelPluginService_ValidateConfigChunkedServerMockRecorder {
	return m.recorder
}

// Context mocks base method.
func (m *MockModelPluginService_ValidateConfigChunkedServer) Context() context.Context {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Context")
	ret0, _ := ret[0].(context.Context)
	return ret0
}

// Context indicates an expected call of Context.
func (mr *MockModelPluginService_ValidateConfigChunkedServerMockRecorder) Context() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Context", reflect.TypeOf((*MockModelPluginService_ValidateConfigChunkedServer)(nil).Context))
}

// Recv mocks base method.
func (m *MockModelPluginService_ValidateConfigChunkedServer) Recv() (*admin.ValidateConfigRequestChunk, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Recv")
	ret0, _ := ret[0].(*admin.ValidateConfigRequestChunk)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// Recv indicates an expected call of Recv.
func (mr *MockModelPluginService_ValidateConfigChunkedServerMockRecorder) Recv() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Recv", reflect.TypeOf((*MockModelPluginService_ValidateConfigChunkedServer)(nil).Recv))
}

// RecvMsg mocks base method.
func (m_2 *MockModelPluginService_ValidateConfigChunkedServer) RecvMsg(m interface{}) error {
	m_2.ctrl.T.Helper()
	ret := m_2.ctrl.Call(m_2, "RecvMsg", m)
	ret0, _ := ret[0].(error)
	return ret0
}

// RecvMsg indicates an expected call of RecvMsg.
func (mr *MockModelPluginService_ValidateConfigChunkedServerMockRecorder) RecvMsg(m interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "RecvMsg", reflect.TypeOf((*MockModelPluginService_ValidateConfigChunkedServer)(nil).RecvMsg), m)
}

// SendAndClose mocks base method.
func (m *MockModelPluginService_ValidateConfigChunkedServer) SendAndClose(arg0 *admin.ValidateConfigResponse) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "SendAndClose", arg0)
	ret0, _ := ret[0].(error)
	return ret0
}

// SendAndClose indicates an expected call of SendAndClose.
func (mr *MockModelPluginService_ValidateConfigChunkedServerMockRecorder) SendAndClose(arg0 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "SendAndClose", reflect.TypeOf((*MockModelPluginService_ValidateConfigChunkedServer)(nil).SendAndClose), arg0)
}

// SendHeader mocks base method.
func (m *MockModelPluginService_ValidateConfigChunkedServer) SendHeader(arg0 metadata.MD) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "SendHeader", arg0)
	ret0, _ := ret[0].(error)
	return ret0
}

// SendHeader indicates an expected call of SendHeader.
func (mr *MockModelPluginService_ValidateConfigChunkedServerMockRecorder) SendHeader(arg0 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "SendHeader", reflect.TypeOf((*MockModelPluginService_ValidateConfigChunkedServer)(nil).SendHeader), arg0)
}

// SendMsg mocks base method.
func (m_2 *MockModelPluginService_ValidateConfigChunkedServer) SendMsg(m interface{}) error {
	m_2.ctrl.T.Helper()
	ret := m_2.ctrl.Call(m_2, "SendMsg", m)
	ret0, _ := ret[0].(error)
	return ret0
}

// SendMsg indicates an expected call of SendMsg.
func (mr *MockModelPluginService_ValidateConfigChunkedServerMockRecorder) SendMsg(m interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "SendMsg", reflect.TypeOf((*MockModelPluginService_ValidateConfigChunkedServer)(nil).SendMsg), m)
}

// SetHeader mocks base method.
func (m *MockModelPluginService_ValidateConfigChunkedServer) SetHeader(arg0 metadata.MD) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "SetHeader", arg0)
	ret0, _ := ret[0].(error)
	return ret0
}

// SetHeader indicates an expected call of SetHeader.
func (mr *MockModelPluginService_ValidateConfigChunkedServerMockRecorder) SetHeader(arg0 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "SetHeader", reflect.TypeOf((*MockModelPluginService_ValidateConfigChunkedServer)(nil).SetHeader), arg0)
}

// SetTrailer mocks base method.
func (m *MockModelPluginService_ValidateConfigChunkedServer) SetTrailer(arg0 metadata.MD) {
	m.ctrl.T.Helper()
	m.ctrl.Call(m, "SetTrailer", arg0)
}

// SetTrailer indicates an expected call of SetTrailer.
func (mr *MockModelPluginService_ValidateConfigChunkedServerMockRecorder) SetTrailer(arg0 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "SetTrailer", reflect.TypeOf((*MockModelPluginService_ValidateConfigChunkedServer)(nil).SetTrailer), arg0)
}

// MockModelPluginService_GetValueSelectionChunkedServer is a mock of ModelPluginService_GetValueSelectionChunkedServer interface.
type MockModelPluginService_GetValueSelectionChunkedServer struct {
	ctrl     *gomock.Controller
	recorder *MockModelPluginService_GetValueSelectionChunkedServerMockRecorder
}

// MockModelPluginService_GetValueSelectionChunkedServerMockRecorder is the mock recorder for MockModelPluginService_GetValueSelectionChunkedServer.
type MockModelPluginService_GetValueSelectionChunkedServerMockRecorder struct {
	mock *MockModelPluginService_GetValueSelectionChunkedServer
}

// NewMockModelPluginService_GetValueSelectionChunkedServer creates a new mock instance.
func NewMockModelPluginService_GetValueSelectionChunkedServer(ctrl *gomock.Controller) *MockModelPluginService_GetValueSelectionChunkedServer {
	mock := &MockModelPluginService_GetValueSelectionChunkedServer{ctrl: ctrl}
	mock.recorder = &MockModelPluginService_GetValueSelectionChunkedServerMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use.
func (m *MockModelPluginService_GetValueSelectionChunkedServer) EXPECT() *MockModelPluginService_GetValueSelectionChunkedServerMockRecorder {
	return m.recorder
}

// Context mocks base method.
func (m *MockModelPluginService_GetValueSelectionChunkedServer) Context() context.Context {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Context")
	ret0, _ := ret[0].(context.Context)
	return ret0
}

// Context indicates an expected call of Context.
func (mr *MockModelPluginService_GetValueSelectionChunkedServerMockRecorder) Context() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Context", reflect.TypeOf((*MockModelPluginService_GetValueSelectionChunkedServer)(nil).Context))
}

// Recv mocks base method.
func (m *MockModelPluginService_GetValueSelectionChunkedServer) Recv() (*admin.ValueSelectionRequestChunk, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Recv")
	ret0, _ := ret[0].(*admin.ValueSelectionRequestChunk)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// Recv indicates an expected call of Recv.
func (mr *MockModelPluginService_GetValueSelectionChunkedServerMockRecorder) Recv() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Recv", reflect.TypeOf((*MockModelPluginService_GetValueSelectionChunkedServer)(nil).Recv))
}

// RecvMsg mocks base method.
func (m_2 *MockModelPluginService_GetValueSelectionChunkedServer) RecvMsg(m interface{}) error {
	m_2.ctrl.T.Helper()
	ret := m_2.ctrl.Call(m_2, "RecvMsg", m)
	ret0, _ := ret[0].(error)
	return ret0
}

// RecvMsg indicates an expected call of RecvMsg.
func (mr *MockModelPluginService_GetValueSelectionChunkedServerMockRecorder) RecvMsg(m interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "RecvMsg", reflect.TypeOf((*MockModelPluginService_GetValueSelectionChunkedServer)(nil).RecvMsg), m)
}

// SendAndClose mocks base method.
func (m *MockModelPluginService_GetValueSelectionChunkedServer) SendAndClose(arg0 *admin.ValueSelectionResponse) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "SendAndClose", arg0)
	ret0, _ := ret[0].(error)
	return ret0
}

// SendAndClose indicates an expected call of SendAndClose.
func (mr *MockModelPluginService_GetValueSelectionChunkedServerMockRecorder) SendAndClose(arg0 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "SendAndClose", reflect.TypeOf((*MockModelPluginService_GetValueSelectionChunkedServer)(nil).SendAndClose), arg0)
}

// SendHeader mocks base method.
func (m *MockModelPluginService_GetValueSelectionChunkedServer) SendHeader(arg0 metadata.MD) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "SendHeader", arg0)
	ret0, _ := ret[0].(error)
	return ret0
}

// SendHeader indicates an expected call of SendHeader.
func (mr *MockModelPluginService_GetValueSelectionChunkedServerMockRecorder) SendHeader(arg0 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "SendHeader", reflect.TypeOf((*MockModelPluginService_GetValueSelectionChunkedServer)(nil).SendHeader), arg0)
}

// SendMsg mocks base method.
func (m_2 *MockModelPluginService_GetValueSelectionChunkedServer) SendMsg(m interface{}) error {
	m_2.ctrl.T.Helper()
	ret := m_2.ctrl.Call(m_2, "SendMsg", m)
	ret0, _ := ret[0].(error)
	return ret0
}

// SendMsg indicates an expected call of SendMsg.
func (mr *MockModelPluginService_GetValueSelectionChunkedServerMockRecorder) SendMsg(m interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "SendMsg", reflect.TypeOf((*MockModelPluginService_GetValueSelectionChunkedServer)(nil).SendMsg), m)
}

// SetHeader mocks base method.
func (m *MockModelPluginService_GetValueSelectionChunkedServer) SetHeader(arg0 metadata.MD) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "SetHeader", arg0)
	ret0, _ := ret[0].(error)
	return ret0
}

// SetHeader indicates an expected call of SetHeader.
func (mr *MockModelPluginService_GetValueSelectionChunkedServerMockRecorder) SetHeader(arg0 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "SetHeader", reflect.TypeOf((*MockModelPluginService_GetValueSelectionChunkedServer)(nil).SetHeader), arg0)
}

// SetTrailer mocks base method.
func (m *MockModelPluginService_GetValueSelectionChunkedServer) SetTrailer(arg0 metadata.MD) {
	m.ctrl.T.Helper()
	m.ctrl.Call(m, "SetTrailer", arg0)
}

// SetTrailer indicates an expected call of SetTrailer.
func (mr *MockModelPluginService_GetValueSelectionChunkedServerMockRecorder) SetTrailer(arg0 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "SetTrailer", reflect.TypeOf((*MockModelPluginService_GetValueSelectionChunkedServer)(nil).SetTrailer), arg0)
}