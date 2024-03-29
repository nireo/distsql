// Code generated by MockGen. DO NOT EDIT.
// Source: github.com/nireo/distsql/service (interfaces: Store)

// Package service is a generated GoMock package.
package service

import (
	reflect "reflect"

	gomock "github.com/golang/mock/gomock"
	pb "github.com/nireo/distsql/pb"
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

// Exec mocks base method.
func (m *MockStore) Exec(arg0 *pb.Request) ([]*pb.ExecRes, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Exec", arg0)
	ret0, _ := ret[0].([]*pb.ExecRes)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// Exec indicates an expected call of Exec.
func (mr *MockStoreMockRecorder) Exec(arg0 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Exec", reflect.TypeOf((*MockStore)(nil).Exec), arg0)
}

// GetServers mocks base method.
func (m *MockStore) GetServers() ([]*pb.Server, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetServers")
	ret0, _ := ret[0].([]*pb.Server)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// GetServers indicates an expected call of GetServers.
func (mr *MockStoreMockRecorder) GetServers() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetServers", reflect.TypeOf((*MockStore)(nil).GetServers))
}

// Join mocks base method.
func (m *MockStore) Join(arg0, arg1 string) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Join", arg0, arg1)
	ret0, _ := ret[0].(error)
	return ret0
}

// Join indicates an expected call of Join.
func (mr *MockStoreMockRecorder) Join(arg0, arg1 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Join", reflect.TypeOf((*MockStore)(nil).Join), arg0, arg1)
}

// LeaderAddr mocks base method.
func (m *MockStore) LeaderAddr() string {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "LeaderAddr")
	ret0, _ := ret[0].(string)
	return ret0
}

// LeaderAddr indicates an expected call of LeaderAddr.
func (mr *MockStoreMockRecorder) LeaderAddr() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "LeaderAddr", reflect.TypeOf((*MockStore)(nil).LeaderAddr))
}

// Leave mocks base method.
func (m *MockStore) Leave(arg0 string) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Leave", arg0)
	ret0, _ := ret[0].(error)
	return ret0
}

// Leave indicates an expected call of Leave.
func (mr *MockStoreMockRecorder) Leave(arg0 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Leave", reflect.TypeOf((*MockStore)(nil).Leave), arg0)
}

// Metrics mocks base method.
func (m *MockStore) Metrics() (map[string]interface{}, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Metrics")
	ret0, _ := ret[0].(map[string]interface{})
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// Metrics indicates an expected call of Metrics.
func (mr *MockStoreMockRecorder) Metrics() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Metrics", reflect.TypeOf((*MockStore)(nil).Metrics))
}

// Query mocks base method.
func (m *MockStore) Query(arg0 *pb.QueryReq) ([]*pb.QueryRes, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Query", arg0)
	ret0, _ := ret[0].([]*pb.QueryRes)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// Query indicates an expected call of Query.
func (mr *MockStoreMockRecorder) Query(arg0 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Query", reflect.TypeOf((*MockStore)(nil).Query), arg0)
}
