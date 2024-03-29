// Code generated by MockGen. DO NOT EDIT.
// Source: pathio.go

// Package pathio is a generated GoMock package.
package pathio

import (
	io "io"
	reflect "reflect"

	s3 "github.com/aws/aws-sdk-go/service/s3"
	gomock "github.com/golang/mock/gomock"
)

// MockPathio is a mock of Pathio interface.
type MockPathio struct {
	ctrl     *gomock.Controller
	recorder *MockPathioMockRecorder
}

// MockPathioMockRecorder is the mock recorder for MockPathio.
type MockPathioMockRecorder struct {
	mock *MockPathio
}

// NewMockPathio creates a new mock instance.
func NewMockPathio(ctrl *gomock.Controller) *MockPathio {
	mock := &MockPathio{ctrl: ctrl}
	mock.recorder = &MockPathioMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use.
func (m *MockPathio) EXPECT() *MockPathioMockRecorder {
	return m.recorder
}

// Delete mocks base method.
func (m *MockPathio) Delete(path string) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Delete", path)
	ret0, _ := ret[0].(error)
	return ret0
}

// Delete indicates an expected call of Delete.
func (mr *MockPathioMockRecorder) Delete(path interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Delete", reflect.TypeOf((*MockPathio)(nil).Delete), path)
}

// Exists mocks base method.
func (m *MockPathio) Exists(path string) (bool, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Exists", path)
	ret0, _ := ret[0].(bool)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// Exists indicates an expected call of Exists.
func (mr *MockPathioMockRecorder) Exists(path interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Exists", reflect.TypeOf((*MockPathio)(nil).Exists), path)
}

// ListFiles mocks base method.
func (m *MockPathio) ListFiles(path string) ([]string, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "ListFiles", path)
	ret0, _ := ret[0].([]string)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// ListFiles indicates an expected call of ListFiles.
func (mr *MockPathioMockRecorder) ListFiles(path interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "ListFiles", reflect.TypeOf((*MockPathio)(nil).ListFiles), path)
}

// Reader mocks base method.
func (m *MockPathio) Reader(path string) (io.ReadCloser, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Reader", path)
	ret0, _ := ret[0].(io.ReadCloser)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// Reader indicates an expected call of Reader.
func (mr *MockPathioMockRecorder) Reader(path interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Reader", reflect.TypeOf((*MockPathio)(nil).Reader), path)
}

// Write mocks base method.
func (m *MockPathio) Write(path string, input []byte) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Write", path, input)
	ret0, _ := ret[0].(error)
	return ret0
}

// Write indicates an expected call of Write.
func (mr *MockPathioMockRecorder) Write(path, input interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Write", reflect.TypeOf((*MockPathio)(nil).Write), path, input)
}

// WriteReader mocks base method.
func (m *MockPathio) WriteReader(path string, input io.ReadSeeker) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "WriteReader", path, input)
	ret0, _ := ret[0].(error)
	return ret0
}

// WriteReader indicates an expected call of WriteReader.
func (mr *MockPathioMockRecorder) WriteReader(path, input interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "WriteReader", reflect.TypeOf((*MockPathio)(nil).WriteReader), path, input)
}

// Mocks3Handler is a mock of s3Handler interface.
type Mocks3Handler struct {
	ctrl     *gomock.Controller
	recorder *Mocks3HandlerMockRecorder
}

// Mocks3HandlerMockRecorder is the mock recorder for Mocks3Handler.
type Mocks3HandlerMockRecorder struct {
	mock *Mocks3Handler
}

// NewMocks3Handler creates a new mock instance.
func NewMocks3Handler(ctrl *gomock.Controller) *Mocks3Handler {
	mock := &Mocks3Handler{ctrl: ctrl}
	mock.recorder = &Mocks3HandlerMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use.
func (m *Mocks3Handler) EXPECT() *Mocks3HandlerMockRecorder {
	return m.recorder
}

// DeleteObject mocks base method.
func (m *Mocks3Handler) DeleteObject(input *s3.DeleteObjectInput) (*s3.DeleteObjectOutput, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "DeleteObject", input)
	ret0, _ := ret[0].(*s3.DeleteObjectOutput)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// DeleteObject indicates an expected call of DeleteObject.
func (mr *Mocks3HandlerMockRecorder) DeleteObject(input interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "DeleteObject", reflect.TypeOf((*Mocks3Handler)(nil).DeleteObject), input)
}

// GetBucketLocation mocks base method.
func (m *Mocks3Handler) GetBucketLocation(input *s3.GetBucketLocationInput) (*s3.GetBucketLocationOutput, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetBucketLocation", input)
	ret0, _ := ret[0].(*s3.GetBucketLocationOutput)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// GetBucketLocation indicates an expected call of GetBucketLocation.
func (mr *Mocks3HandlerMockRecorder) GetBucketLocation(input interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetBucketLocation", reflect.TypeOf((*Mocks3Handler)(nil).GetBucketLocation), input)
}

// GetObject mocks base method.
func (m *Mocks3Handler) GetObject(input *s3.GetObjectInput) (*s3.GetObjectOutput, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetObject", input)
	ret0, _ := ret[0].(*s3.GetObjectOutput)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// GetObject indicates an expected call of GetObject.
func (mr *Mocks3HandlerMockRecorder) GetObject(input interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetObject", reflect.TypeOf((*Mocks3Handler)(nil).GetObject), input)
}

// HeadObject mocks base method.
func (m *Mocks3Handler) HeadObject(input *s3.HeadObjectInput) (*s3.HeadObjectOutput, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "HeadObject", input)
	ret0, _ := ret[0].(*s3.HeadObjectOutput)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// HeadObject indicates an expected call of HeadObject.
func (mr *Mocks3HandlerMockRecorder) HeadObject(input interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "HeadObject", reflect.TypeOf((*Mocks3Handler)(nil).HeadObject), input)
}

// ListObjects mocks base method.
func (m *Mocks3Handler) ListObjects(input *s3.ListObjectsInput) (*s3.ListObjectsOutput, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "ListObjects", input)
	ret0, _ := ret[0].(*s3.ListObjectsOutput)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// ListObjects indicates an expected call of ListObjects.
func (mr *Mocks3HandlerMockRecorder) ListObjects(input interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "ListObjects", reflect.TypeOf((*Mocks3Handler)(nil).ListObjects), input)
}

// PutObject mocks base method.
func (m *Mocks3Handler) PutObject(input *s3.PutObjectInput) (*s3.PutObjectOutput, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "PutObject", input)
	ret0, _ := ret[0].(*s3.PutObjectOutput)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// PutObject indicates an expected call of PutObject.
func (mr *Mocks3HandlerMockRecorder) PutObject(input interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "PutObject", reflect.TypeOf((*Mocks3Handler)(nil).PutObject), input)
}
