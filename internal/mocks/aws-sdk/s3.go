// Copyright (c) Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.
//
// Code generated by MockGen. DO NOT EDIT.
// Source: github.com/mattermost/mattermost-cloud/internal/tools/aws (interfaces: S3API)

// Package mockawssdk is a generated GoMock package.
package mockawssdk

import (
	context "context"
	s3 "github.com/aws/aws-sdk-go-v2/service/s3"
	gomock "github.com/golang/mock/gomock"
	reflect "reflect"
)

// MockS3API is a mock of S3API interface
type MockS3API struct {
	ctrl     *gomock.Controller
	recorder *MockS3APIMockRecorder
}

// MockS3APIMockRecorder is the mock recorder for MockS3API
type MockS3APIMockRecorder struct {
	mock *MockS3API
}

// NewMockS3API creates a new mock instance
func NewMockS3API(ctrl *gomock.Controller) *MockS3API {
	mock := &MockS3API{ctrl: ctrl}
	mock.recorder = &MockS3APIMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use
func (m *MockS3API) EXPECT() *MockS3APIMockRecorder {
	return m.recorder
}

// CompleteMultipartUpload mocks base method
func (m *MockS3API) CompleteMultipartUpload(arg0 context.Context, arg1 *s3.CompleteMultipartUploadInput, arg2 ...func(*s3.Options)) (*s3.CompleteMultipartUploadOutput, error) {
	m.ctrl.T.Helper()
	varargs := []interface{}{arg0, arg1}
	for _, a := range arg2 {
		varargs = append(varargs, a)
	}
	ret := m.ctrl.Call(m, "CompleteMultipartUpload", varargs...)
	ret0, _ := ret[0].(*s3.CompleteMultipartUploadOutput)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// CompleteMultipartUpload indicates an expected call of CompleteMultipartUpload
func (mr *MockS3APIMockRecorder) CompleteMultipartUpload(arg0, arg1 interface{}, arg2 ...interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	varargs := append([]interface{}{arg0, arg1}, arg2...)
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "CompleteMultipartUpload", reflect.TypeOf((*MockS3API)(nil).CompleteMultipartUpload), varargs...)
}

// CreateBucket mocks base method
func (m *MockS3API) CreateBucket(arg0 context.Context, arg1 *s3.CreateBucketInput, arg2 ...func(*s3.Options)) (*s3.CreateBucketOutput, error) {
	m.ctrl.T.Helper()
	varargs := []interface{}{arg0, arg1}
	for _, a := range arg2 {
		varargs = append(varargs, a)
	}
	ret := m.ctrl.Call(m, "CreateBucket", varargs...)
	ret0, _ := ret[0].(*s3.CreateBucketOutput)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// CreateBucket indicates an expected call of CreateBucket
func (mr *MockS3APIMockRecorder) CreateBucket(arg0, arg1 interface{}, arg2 ...interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	varargs := append([]interface{}{arg0, arg1}, arg2...)
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "CreateBucket", reflect.TypeOf((*MockS3API)(nil).CreateBucket), varargs...)
}

// CreateMultipartUpload mocks base method
func (m *MockS3API) CreateMultipartUpload(arg0 context.Context, arg1 *s3.CreateMultipartUploadInput, arg2 ...func(*s3.Options)) (*s3.CreateMultipartUploadOutput, error) {
	m.ctrl.T.Helper()
	varargs := []interface{}{arg0, arg1}
	for _, a := range arg2 {
		varargs = append(varargs, a)
	}
	ret := m.ctrl.Call(m, "CreateMultipartUpload", varargs...)
	ret0, _ := ret[0].(*s3.CreateMultipartUploadOutput)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// CreateMultipartUpload indicates an expected call of CreateMultipartUpload
func (mr *MockS3APIMockRecorder) CreateMultipartUpload(arg0, arg1 interface{}, arg2 ...interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	varargs := append([]interface{}{arg0, arg1}, arg2...)
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "CreateMultipartUpload", reflect.TypeOf((*MockS3API)(nil).CreateMultipartUpload), varargs...)
}

// DeleteBucket mocks base method
func (m *MockS3API) DeleteBucket(arg0 context.Context, arg1 *s3.DeleteBucketInput, arg2 ...func(*s3.Options)) (*s3.DeleteBucketOutput, error) {
	m.ctrl.T.Helper()
	varargs := []interface{}{arg0, arg1}
	for _, a := range arg2 {
		varargs = append(varargs, a)
	}
	ret := m.ctrl.Call(m, "DeleteBucket", varargs...)
	ret0, _ := ret[0].(*s3.DeleteBucketOutput)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// DeleteBucket indicates an expected call of DeleteBucket
func (mr *MockS3APIMockRecorder) DeleteBucket(arg0, arg1 interface{}, arg2 ...interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	varargs := append([]interface{}{arg0, arg1}, arg2...)
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "DeleteBucket", reflect.TypeOf((*MockS3API)(nil).DeleteBucket), varargs...)
}

// DeleteObject mocks base method
func (m *MockS3API) DeleteObject(arg0 context.Context, arg1 *s3.DeleteObjectInput, arg2 ...func(*s3.Options)) (*s3.DeleteObjectOutput, error) {
	m.ctrl.T.Helper()
	varargs := []interface{}{arg0, arg1}
	for _, a := range arg2 {
		varargs = append(varargs, a)
	}
	ret := m.ctrl.Call(m, "DeleteObject", varargs...)
	ret0, _ := ret[0].(*s3.DeleteObjectOutput)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// DeleteObject indicates an expected call of DeleteObject
func (mr *MockS3APIMockRecorder) DeleteObject(arg0, arg1 interface{}, arg2 ...interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	varargs := append([]interface{}{arg0, arg1}, arg2...)
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "DeleteObject", reflect.TypeOf((*MockS3API)(nil).DeleteObject), varargs...)
}

// DeleteObjects mocks base method
func (m *MockS3API) DeleteObjects(arg0 context.Context, arg1 *s3.DeleteObjectsInput, arg2 ...func(*s3.Options)) (*s3.DeleteObjectsOutput, error) {
	m.ctrl.T.Helper()
	varargs := []interface{}{arg0, arg1}
	for _, a := range arg2 {
		varargs = append(varargs, a)
	}
	ret := m.ctrl.Call(m, "DeleteObjects", varargs...)
	ret0, _ := ret[0].(*s3.DeleteObjectsOutput)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// DeleteObjects indicates an expected call of DeleteObjects
func (mr *MockS3APIMockRecorder) DeleteObjects(arg0, arg1 interface{}, arg2 ...interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	varargs := append([]interface{}{arg0, arg1}, arg2...)
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "DeleteObjects", reflect.TypeOf((*MockS3API)(nil).DeleteObjects), varargs...)
}

// GetBucketTagging mocks base method
func (m *MockS3API) GetBucketTagging(arg0 context.Context, arg1 *s3.GetBucketTaggingInput, arg2 ...func(*s3.Options)) (*s3.GetBucketTaggingOutput, error) {
	m.ctrl.T.Helper()
	varargs := []interface{}{arg0, arg1}
	for _, a := range arg2 {
		varargs = append(varargs, a)
	}
	ret := m.ctrl.Call(m, "GetBucketTagging", varargs...)
	ret0, _ := ret[0].(*s3.GetBucketTaggingOutput)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// GetBucketTagging indicates an expected call of GetBucketTagging
func (mr *MockS3APIMockRecorder) GetBucketTagging(arg0, arg1 interface{}, arg2 ...interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	varargs := append([]interface{}{arg0, arg1}, arg2...)
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetBucketTagging", reflect.TypeOf((*MockS3API)(nil).GetBucketTagging), varargs...)
}

// GetBucketVersioning mocks base method
func (m *MockS3API) GetBucketVersioning(arg0 context.Context, arg1 *s3.GetBucketVersioningInput, arg2 ...func(*s3.Options)) (*s3.GetBucketVersioningOutput, error) {
	m.ctrl.T.Helper()
	varargs := []interface{}{arg0, arg1}
	for _, a := range arg2 {
		varargs = append(varargs, a)
	}
	ret := m.ctrl.Call(m, "GetBucketVersioning", varargs...)
	ret0, _ := ret[0].(*s3.GetBucketVersioningOutput)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// GetBucketVersioning indicates an expected call of GetBucketVersioning
func (mr *MockS3APIMockRecorder) GetBucketVersioning(arg0, arg1 interface{}, arg2 ...interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	varargs := append([]interface{}{arg0, arg1}, arg2...)
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetBucketVersioning", reflect.TypeOf((*MockS3API)(nil).GetBucketVersioning), varargs...)
}

// HeadBucket mocks base method
func (m *MockS3API) HeadBucket(arg0 context.Context, arg1 *s3.HeadBucketInput, arg2 ...func(*s3.Options)) (*s3.HeadBucketOutput, error) {
	m.ctrl.T.Helper()
	varargs := []interface{}{arg0, arg1}
	for _, a := range arg2 {
		varargs = append(varargs, a)
	}
	ret := m.ctrl.Call(m, "HeadBucket", varargs...)
	ret0, _ := ret[0].(*s3.HeadBucketOutput)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// HeadBucket indicates an expected call of HeadBucket
func (mr *MockS3APIMockRecorder) HeadBucket(arg0, arg1 interface{}, arg2 ...interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	varargs := append([]interface{}{arg0, arg1}, arg2...)
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "HeadBucket", reflect.TypeOf((*MockS3API)(nil).HeadBucket), varargs...)
}

// HeadObject mocks base method
func (m *MockS3API) HeadObject(arg0 context.Context, arg1 *s3.HeadObjectInput, arg2 ...func(*s3.Options)) (*s3.HeadObjectOutput, error) {
	m.ctrl.T.Helper()
	varargs := []interface{}{arg0, arg1}
	for _, a := range arg2 {
		varargs = append(varargs, a)
	}
	ret := m.ctrl.Call(m, "HeadObject", varargs...)
	ret0, _ := ret[0].(*s3.HeadObjectOutput)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// HeadObject indicates an expected call of HeadObject
func (mr *MockS3APIMockRecorder) HeadObject(arg0, arg1 interface{}, arg2 ...interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	varargs := append([]interface{}{arg0, arg1}, arg2...)
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "HeadObject", reflect.TypeOf((*MockS3API)(nil).HeadObject), varargs...)
}

// ListObjectVersions mocks base method
func (m *MockS3API) ListObjectVersions(arg0 context.Context, arg1 *s3.ListObjectVersionsInput, arg2 ...func(*s3.Options)) (*s3.ListObjectVersionsOutput, error) {
	m.ctrl.T.Helper()
	varargs := []interface{}{arg0, arg1}
	for _, a := range arg2 {
		varargs = append(varargs, a)
	}
	ret := m.ctrl.Call(m, "ListObjectVersions", varargs...)
	ret0, _ := ret[0].(*s3.ListObjectVersionsOutput)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// ListObjectVersions indicates an expected call of ListObjectVersions
func (mr *MockS3APIMockRecorder) ListObjectVersions(arg0, arg1 interface{}, arg2 ...interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	varargs := append([]interface{}{arg0, arg1}, arg2...)
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "ListObjectVersions", reflect.TypeOf((*MockS3API)(nil).ListObjectVersions), varargs...)
}

// ListObjectsV2 mocks base method
func (m *MockS3API) ListObjectsV2(arg0 context.Context, arg1 *s3.ListObjectsV2Input, arg2 ...func(*s3.Options)) (*s3.ListObjectsV2Output, error) {
	m.ctrl.T.Helper()
	varargs := []interface{}{arg0, arg1}
	for _, a := range arg2 {
		varargs = append(varargs, a)
	}
	ret := m.ctrl.Call(m, "ListObjectsV2", varargs...)
	ret0, _ := ret[0].(*s3.ListObjectsV2Output)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// ListObjectsV2 indicates an expected call of ListObjectsV2
func (mr *MockS3APIMockRecorder) ListObjectsV2(arg0, arg1 interface{}, arg2 ...interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	varargs := append([]interface{}{arg0, arg1}, arg2...)
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "ListObjectsV2", reflect.TypeOf((*MockS3API)(nil).ListObjectsV2), varargs...)
}

// PutBucketEncryption mocks base method
func (m *MockS3API) PutBucketEncryption(arg0 context.Context, arg1 *s3.PutBucketEncryptionInput, arg2 ...func(*s3.Options)) (*s3.PutBucketEncryptionOutput, error) {
	m.ctrl.T.Helper()
	varargs := []interface{}{arg0, arg1}
	for _, a := range arg2 {
		varargs = append(varargs, a)
	}
	ret := m.ctrl.Call(m, "PutBucketEncryption", varargs...)
	ret0, _ := ret[0].(*s3.PutBucketEncryptionOutput)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// PutBucketEncryption indicates an expected call of PutBucketEncryption
func (mr *MockS3APIMockRecorder) PutBucketEncryption(arg0, arg1 interface{}, arg2 ...interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	varargs := append([]interface{}{arg0, arg1}, arg2...)
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "PutBucketEncryption", reflect.TypeOf((*MockS3API)(nil).PutBucketEncryption), varargs...)
}

// PutBucketVersioning mocks base method
func (m *MockS3API) PutBucketVersioning(arg0 context.Context, arg1 *s3.PutBucketVersioningInput, arg2 ...func(*s3.Options)) (*s3.PutBucketVersioningOutput, error) {
	m.ctrl.T.Helper()
	varargs := []interface{}{arg0, arg1}
	for _, a := range arg2 {
		varargs = append(varargs, a)
	}
	ret := m.ctrl.Call(m, "PutBucketVersioning", varargs...)
	ret0, _ := ret[0].(*s3.PutBucketVersioningOutput)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// PutBucketVersioning indicates an expected call of PutBucketVersioning
func (mr *MockS3APIMockRecorder) PutBucketVersioning(arg0, arg1 interface{}, arg2 ...interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	varargs := append([]interface{}{arg0, arg1}, arg2...)
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "PutBucketVersioning", reflect.TypeOf((*MockS3API)(nil).PutBucketVersioning), varargs...)
}

// PutPublicAccessBlock mocks base method
func (m *MockS3API) PutPublicAccessBlock(arg0 context.Context, arg1 *s3.PutPublicAccessBlockInput, arg2 ...func(*s3.Options)) (*s3.PutPublicAccessBlockOutput, error) {
	m.ctrl.T.Helper()
	varargs := []interface{}{arg0, arg1}
	for _, a := range arg2 {
		varargs = append(varargs, a)
	}
	ret := m.ctrl.Call(m, "PutPublicAccessBlock", varargs...)
	ret0, _ := ret[0].(*s3.PutPublicAccessBlockOutput)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// PutPublicAccessBlock indicates an expected call of PutPublicAccessBlock
func (mr *MockS3APIMockRecorder) PutPublicAccessBlock(arg0, arg1 interface{}, arg2 ...interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	varargs := append([]interface{}{arg0, arg1}, arg2...)
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "PutPublicAccessBlock", reflect.TypeOf((*MockS3API)(nil).PutPublicAccessBlock), varargs...)
}

// UploadPartCopy mocks base method
func (m *MockS3API) UploadPartCopy(arg0 context.Context, arg1 *s3.UploadPartCopyInput, arg2 ...func(*s3.Options)) (*s3.UploadPartCopyOutput, error) {
	m.ctrl.T.Helper()
	varargs := []interface{}{arg0, arg1}
	for _, a := range arg2 {
		varargs = append(varargs, a)
	}
	ret := m.ctrl.Call(m, "UploadPartCopy", varargs...)
	ret0, _ := ret[0].(*s3.UploadPartCopyOutput)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// UploadPartCopy indicates an expected call of UploadPartCopy
func (mr *MockS3APIMockRecorder) UploadPartCopy(arg0, arg1 interface{}, arg2 ...interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	varargs := append([]interface{}{arg0, arg1}, arg2...)
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "UploadPartCopy", reflect.TypeOf((*MockS3API)(nil).UploadPartCopy), varargs...)
}
