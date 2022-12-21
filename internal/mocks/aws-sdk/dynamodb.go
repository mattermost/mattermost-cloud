// Copyright (c) Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.
//
// Code generated by MockGen. DO NOT EDIT.
// Source: github.com/mattermost/mattermost-cloud/internal/tools/aws (interfaces: DynamoDBAPI)

// Package mockawssdk is a generated GoMock package.
package mockawssdk

import (
	context "context"
	dynamodb "github.com/aws/aws-sdk-go-v2/service/dynamodb"
	gomock "github.com/golang/mock/gomock"
	reflect "reflect"
)

// MockDynamoDBAPI is a mock of DynamoDBAPI interface
type MockDynamoDBAPI struct {
	ctrl     *gomock.Controller
	recorder *MockDynamoDBAPIMockRecorder
}

// MockDynamoDBAPIMockRecorder is the mock recorder for MockDynamoDBAPI
type MockDynamoDBAPIMockRecorder struct {
	mock *MockDynamoDBAPI
}

// NewMockDynamoDBAPI creates a new mock instance
func NewMockDynamoDBAPI(ctrl *gomock.Controller) *MockDynamoDBAPI {
	mock := &MockDynamoDBAPI{ctrl: ctrl}
	mock.recorder = &MockDynamoDBAPIMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use
func (m *MockDynamoDBAPI) EXPECT() *MockDynamoDBAPIMockRecorder {
	return m.recorder
}

// DeleteTable mocks base method
func (m *MockDynamoDBAPI) DeleteTable(arg0 context.Context, arg1 *dynamodb.DeleteTableInput, arg2 ...func(*dynamodb.Options)) (*dynamodb.DeleteTableOutput, error) {
	m.ctrl.T.Helper()
	varargs := []interface{}{arg0, arg1}
	for _, a := range arg2 {
		varargs = append(varargs, a)
	}
	ret := m.ctrl.Call(m, "DeleteTable", varargs...)
	ret0, _ := ret[0].(*dynamodb.DeleteTableOutput)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// DeleteTable indicates an expected call of DeleteTable
func (mr *MockDynamoDBAPIMockRecorder) DeleteTable(arg0, arg1 interface{}, arg2 ...interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	varargs := append([]interface{}{arg0, arg1}, arg2...)
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "DeleteTable", reflect.TypeOf((*MockDynamoDBAPI)(nil).DeleteTable), varargs...)
}

// DescribeTable mocks base method
func (m *MockDynamoDBAPI) DescribeTable(arg0 context.Context, arg1 *dynamodb.DescribeTableInput, arg2 ...func(*dynamodb.Options)) (*dynamodb.DescribeTableOutput, error) {
	m.ctrl.T.Helper()
	varargs := []interface{}{arg0, arg1}
	for _, a := range arg2 {
		varargs = append(varargs, a)
	}
	ret := m.ctrl.Call(m, "DescribeTable", varargs...)
	ret0, _ := ret[0].(*dynamodb.DescribeTableOutput)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// DescribeTable indicates an expected call of DescribeTable
func (mr *MockDynamoDBAPIMockRecorder) DescribeTable(arg0, arg1 interface{}, arg2 ...interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	varargs := append([]interface{}{arg0, arg1}, arg2...)
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "DescribeTable", reflect.TypeOf((*MockDynamoDBAPI)(nil).DescribeTable), varargs...)
}
