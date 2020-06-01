// Code generated by MockGen. DO NOT EDIT.
// Source: /Users/angeloskyratzakos/go/src/github.com/sirupsen/logrus/logrus.go

// Package mocks is a generated GoMock package.
package mocks

import (
	gomock "github.com/golang/mock/gomock"
	logrus "github.com/sirupsen/logrus"
	reflect "reflect"
)

// MockStdLogger is a mock of StdLogger interface
type MockStdLogger struct {
	ctrl     *gomock.Controller
	recorder *MockStdLoggerMockRecorder
}

// MockStdLoggerMockRecorder is the mock recorder for MockStdLogger
type MockStdLoggerMockRecorder struct {
	mock *MockStdLogger
}

// NewMockStdLogger creates a new mock instance
func NewMockStdLogger(ctrl *gomock.Controller) *MockStdLogger {
	mock := &MockStdLogger{ctrl: ctrl}
	mock.recorder = &MockStdLoggerMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use
func (m *MockStdLogger) EXPECT() *MockStdLoggerMockRecorder {
	return m.recorder
}

// Print mocks base method
func (m *MockStdLogger) Print(arg0 ...interface{}) {
	m.ctrl.T.Helper()
	varargs := []interface{}{}
	for _, a := range arg0 {
		varargs = append(varargs, a)
	}
	m.ctrl.Call(m, "Print", varargs...)
}

// Print indicates an expected call of Print
func (mr *MockStdLoggerMockRecorder) Print(arg0 ...interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Print", reflect.TypeOf((*MockStdLogger)(nil).Print), arg0...)
}

// Printf mocks base method
func (m *MockStdLogger) Printf(arg0 string, arg1 ...interface{}) {
	m.ctrl.T.Helper()
	varargs := []interface{}{arg0}
	for _, a := range arg1 {
		varargs = append(varargs, a)
	}
	m.ctrl.Call(m, "Printf", varargs...)
}

// Printf indicates an expected call of Printf
func (mr *MockStdLoggerMockRecorder) Printf(arg0 interface{}, arg1 ...interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	varargs := append([]interface{}{arg0}, arg1...)
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Printf", reflect.TypeOf((*MockStdLogger)(nil).Printf), varargs...)
}

// Println mocks base method
func (m *MockStdLogger) Println(arg0 ...interface{}) {
	m.ctrl.T.Helper()
	varargs := []interface{}{}
	for _, a := range arg0 {
		varargs = append(varargs, a)
	}
	m.ctrl.Call(m, "Println", varargs...)
}

// Println indicates an expected call of Println
func (mr *MockStdLoggerMockRecorder) Println(arg0 ...interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Println", reflect.TypeOf((*MockStdLogger)(nil).Println), arg0...)
}

// Fatal mocks base method
func (m *MockStdLogger) Fatal(arg0 ...interface{}) {
	m.ctrl.T.Helper()
	varargs := []interface{}{}
	for _, a := range arg0 {
		varargs = append(varargs, a)
	}
	m.ctrl.Call(m, "Fatal", varargs...)
}

// Fatal indicates an expected call of Fatal
func (mr *MockStdLoggerMockRecorder) Fatal(arg0 ...interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Fatal", reflect.TypeOf((*MockStdLogger)(nil).Fatal), arg0...)
}

// Fatalf mocks base method
func (m *MockStdLogger) Fatalf(arg0 string, arg1 ...interface{}) {
	m.ctrl.T.Helper()
	varargs := []interface{}{arg0}
	for _, a := range arg1 {
		varargs = append(varargs, a)
	}
	m.ctrl.Call(m, "Fatalf", varargs...)
}

// Fatalf indicates an expected call of Fatalf
func (mr *MockStdLoggerMockRecorder) Fatalf(arg0 interface{}, arg1 ...interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	varargs := append([]interface{}{arg0}, arg1...)
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Fatalf", reflect.TypeOf((*MockStdLogger)(nil).Fatalf), varargs...)
}

// Fatalln mocks base method
func (m *MockStdLogger) Fatalln(arg0 ...interface{}) {
	m.ctrl.T.Helper()
	varargs := []interface{}{}
	for _, a := range arg0 {
		varargs = append(varargs, a)
	}
	m.ctrl.Call(m, "Fatalln", varargs...)
}

// Fatalln indicates an expected call of Fatalln
func (mr *MockStdLoggerMockRecorder) Fatalln(arg0 ...interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Fatalln", reflect.TypeOf((*MockStdLogger)(nil).Fatalln), arg0...)
}

// Panic mocks base method
func (m *MockStdLogger) Panic(arg0 ...interface{}) {
	m.ctrl.T.Helper()
	varargs := []interface{}{}
	for _, a := range arg0 {
		varargs = append(varargs, a)
	}
	m.ctrl.Call(m, "Panic", varargs...)
}

// Panic indicates an expected call of Panic
func (mr *MockStdLoggerMockRecorder) Panic(arg0 ...interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Panic", reflect.TypeOf((*MockStdLogger)(nil).Panic), arg0...)
}

// Panicf mocks base method
func (m *MockStdLogger) Panicf(arg0 string, arg1 ...interface{}) {
	m.ctrl.T.Helper()
	varargs := []interface{}{arg0}
	for _, a := range arg1 {
		varargs = append(varargs, a)
	}
	m.ctrl.Call(m, "Panicf", varargs...)
}

// Panicf indicates an expected call of Panicf
func (mr *MockStdLoggerMockRecorder) Panicf(arg0 interface{}, arg1 ...interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	varargs := append([]interface{}{arg0}, arg1...)
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Panicf", reflect.TypeOf((*MockStdLogger)(nil).Panicf), varargs...)
}

// Panicln mocks base method
func (m *MockStdLogger) Panicln(arg0 ...interface{}) {
	m.ctrl.T.Helper()
	varargs := []interface{}{}
	for _, a := range arg0 {
		varargs = append(varargs, a)
	}
	m.ctrl.Call(m, "Panicln", varargs...)
}

// Panicln indicates an expected call of Panicln
func (mr *MockStdLoggerMockRecorder) Panicln(arg0 ...interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Panicln", reflect.TypeOf((*MockStdLogger)(nil).Panicln), arg0...)
}

// MockFieldLogger is a mock of FieldLogger interface
type MockFieldLogger struct {
	ctrl     *gomock.Controller
	recorder *MockFieldLoggerMockRecorder
}

// MockFieldLoggerMockRecorder is the mock recorder for MockFieldLogger
type MockFieldLoggerMockRecorder struct {
	mock *MockFieldLogger
}

// NewMockFieldLogger creates a new mock instance
func NewMockFieldLogger(ctrl *gomock.Controller) *MockFieldLogger {
	mock := &MockFieldLogger{ctrl: ctrl}
	mock.recorder = &MockFieldLoggerMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use
func (m *MockFieldLogger) EXPECT() *MockFieldLoggerMockRecorder {
	return m.recorder
}

// WithField mocks base method
func (m *MockFieldLogger) WithField(key string, value interface{}) *logrus.Entry {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "WithField", key, value)
	ret0, _ := ret[0].(*logrus.Entry)
	return ret0
}

// WithField indicates an expected call of WithField
func (mr *MockFieldLoggerMockRecorder) WithField(key, value interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "WithField", reflect.TypeOf((*MockFieldLogger)(nil).WithField), key, value)
}

// WithFields mocks base method
func (m *MockFieldLogger) WithFields(fields logrus.Fields) *logrus.Entry {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "WithFields", fields)
	ret0, _ := ret[0].(*logrus.Entry)
	return ret0
}

// WithFields indicates an expected call of WithFields
func (mr *MockFieldLoggerMockRecorder) WithFields(fields interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "WithFields", reflect.TypeOf((*MockFieldLogger)(nil).WithFields), fields)
}

// WithError mocks base method
func (m *MockFieldLogger) WithError(err error) *logrus.Entry {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "WithError", err)
	ret0, _ := ret[0].(*logrus.Entry)
	return ret0
}

// WithError indicates an expected call of WithError
func (mr *MockFieldLoggerMockRecorder) WithError(err interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "WithError", reflect.TypeOf((*MockFieldLogger)(nil).WithError), err)
}

// Debugf mocks base method
func (m *MockFieldLogger) Debugf(format string, args ...interface{}) {
	m.ctrl.T.Helper()
	varargs := []interface{}{format}
	for _, a := range args {
		varargs = append(varargs, a)
	}
	m.ctrl.Call(m, "Debugf", varargs...)
}

// Debugf indicates an expected call of Debugf
func (mr *MockFieldLoggerMockRecorder) Debugf(format interface{}, args ...interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	varargs := append([]interface{}{format}, args...)
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Debugf", reflect.TypeOf((*MockFieldLogger)(nil).Debugf), varargs...)
}

// Infof mocks base method
func (m *MockFieldLogger) Infof(format string, args ...interface{}) {
	m.ctrl.T.Helper()
	varargs := []interface{}{format}
	for _, a := range args {
		varargs = append(varargs, a)
	}
	m.ctrl.Call(m, "Infof", varargs...)
}

// Infof indicates an expected call of Infof
func (mr *MockFieldLoggerMockRecorder) Infof(format interface{}, args ...interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	varargs := append([]interface{}{format}, args...)
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Infof", reflect.TypeOf((*MockFieldLogger)(nil).Infof), varargs...)
}

// Printf mocks base method
func (m *MockFieldLogger) Printf(format string, args ...interface{}) {
	m.ctrl.T.Helper()
	varargs := []interface{}{format}
	for _, a := range args {
		varargs = append(varargs, a)
	}
	m.ctrl.Call(m, "Printf", varargs...)
}

// Printf indicates an expected call of Printf
func (mr *MockFieldLoggerMockRecorder) Printf(format interface{}, args ...interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	varargs := append([]interface{}{format}, args...)
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Printf", reflect.TypeOf((*MockFieldLogger)(nil).Printf), varargs...)
}

// Warnf mocks base method
func (m *MockFieldLogger) Warnf(format string, args ...interface{}) {
	m.ctrl.T.Helper()
	varargs := []interface{}{format}
	for _, a := range args {
		varargs = append(varargs, a)
	}
	m.ctrl.Call(m, "Warnf", varargs...)
}

// Warnf indicates an expected call of Warnf
func (mr *MockFieldLoggerMockRecorder) Warnf(format interface{}, args ...interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	varargs := append([]interface{}{format}, args...)
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Warnf", reflect.TypeOf((*MockFieldLogger)(nil).Warnf), varargs...)
}

// Warningf mocks base method
func (m *MockFieldLogger) Warningf(format string, args ...interface{}) {
	m.ctrl.T.Helper()
	varargs := []interface{}{format}
	for _, a := range args {
		varargs = append(varargs, a)
	}
	m.ctrl.Call(m, "Warningf", varargs...)
}

// Warningf indicates an expected call of Warningf
func (mr *MockFieldLoggerMockRecorder) Warningf(format interface{}, args ...interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	varargs := append([]interface{}{format}, args...)
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Warningf", reflect.TypeOf((*MockFieldLogger)(nil).Warningf), varargs...)
}

// Errorf mocks base method
func (m *MockFieldLogger) Errorf(format string, args ...interface{}) {
	m.ctrl.T.Helper()
	varargs := []interface{}{format}
	for _, a := range args {
		varargs = append(varargs, a)
	}
	m.ctrl.Call(m, "Errorf", varargs...)
}

// Errorf indicates an expected call of Errorf
func (mr *MockFieldLoggerMockRecorder) Errorf(format interface{}, args ...interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	varargs := append([]interface{}{format}, args...)
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Errorf", reflect.TypeOf((*MockFieldLogger)(nil).Errorf), varargs...)
}

// Fatalf mocks base method
func (m *MockFieldLogger) Fatalf(format string, args ...interface{}) {
	m.ctrl.T.Helper()
	varargs := []interface{}{format}
	for _, a := range args {
		varargs = append(varargs, a)
	}
	m.ctrl.Call(m, "Fatalf", varargs...)
}

// Fatalf indicates an expected call of Fatalf
func (mr *MockFieldLoggerMockRecorder) Fatalf(format interface{}, args ...interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	varargs := append([]interface{}{format}, args...)
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Fatalf", reflect.TypeOf((*MockFieldLogger)(nil).Fatalf), varargs...)
}

// Panicf mocks base method
func (m *MockFieldLogger) Panicf(format string, args ...interface{}) {
	m.ctrl.T.Helper()
	varargs := []interface{}{format}
	for _, a := range args {
		varargs = append(varargs, a)
	}
	m.ctrl.Call(m, "Panicf", varargs...)
}

// Panicf indicates an expected call of Panicf
func (mr *MockFieldLoggerMockRecorder) Panicf(format interface{}, args ...interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	varargs := append([]interface{}{format}, args...)
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Panicf", reflect.TypeOf((*MockFieldLogger)(nil).Panicf), varargs...)
}

// Debug mocks base method
func (m *MockFieldLogger) Debug(args ...interface{}) {
	m.ctrl.T.Helper()
	varargs := []interface{}{}
	for _, a := range args {
		varargs = append(varargs, a)
	}
	m.ctrl.Call(m, "Debug", varargs...)
}

// Debug indicates an expected call of Debug
func (mr *MockFieldLoggerMockRecorder) Debug(args ...interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Debug", reflect.TypeOf((*MockFieldLogger)(nil).Debug), args...)
}

// Info mocks base method
func (m *MockFieldLogger) Info(args ...interface{}) {
	m.ctrl.T.Helper()
	varargs := []interface{}{}
	for _, a := range args {
		varargs = append(varargs, a)
	}
	m.ctrl.Call(m, "Info", varargs...)
}

// Info indicates an expected call of Info
func (mr *MockFieldLoggerMockRecorder) Info(args ...interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Info", reflect.TypeOf((*MockFieldLogger)(nil).Info), args...)
}

// Print mocks base method
func (m *MockFieldLogger) Print(args ...interface{}) {
	m.ctrl.T.Helper()
	varargs := []interface{}{}
	for _, a := range args {
		varargs = append(varargs, a)
	}
	m.ctrl.Call(m, "Print", varargs...)
}

// Print indicates an expected call of Print
func (mr *MockFieldLoggerMockRecorder) Print(args ...interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Print", reflect.TypeOf((*MockFieldLogger)(nil).Print), args...)
}

// Warn mocks base method
func (m *MockFieldLogger) Warn(args ...interface{}) {
	m.ctrl.T.Helper()
	varargs := []interface{}{}
	for _, a := range args {
		varargs = append(varargs, a)
	}
	m.ctrl.Call(m, "Warn", varargs...)
}

// Warn indicates an expected call of Warn
func (mr *MockFieldLoggerMockRecorder) Warn(args ...interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Warn", reflect.TypeOf((*MockFieldLogger)(nil).Warn), args...)
}

// Warning mocks base method
func (m *MockFieldLogger) Warning(args ...interface{}) {
	m.ctrl.T.Helper()
	varargs := []interface{}{}
	for _, a := range args {
		varargs = append(varargs, a)
	}
	m.ctrl.Call(m, "Warning", varargs...)
}

// Warning indicates an expected call of Warning
func (mr *MockFieldLoggerMockRecorder) Warning(args ...interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Warning", reflect.TypeOf((*MockFieldLogger)(nil).Warning), args...)
}

// Error mocks base method
func (m *MockFieldLogger) Error(args ...interface{}) {
	m.ctrl.T.Helper()
	varargs := []interface{}{}
	for _, a := range args {
		varargs = append(varargs, a)
	}
	m.ctrl.Call(m, "Error", varargs...)
}

// Error indicates an expected call of Error
func (mr *MockFieldLoggerMockRecorder) Error(args ...interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Error", reflect.TypeOf((*MockFieldLogger)(nil).Error), args...)
}

// Fatal mocks base method
func (m *MockFieldLogger) Fatal(args ...interface{}) {
	m.ctrl.T.Helper()
	varargs := []interface{}{}
	for _, a := range args {
		varargs = append(varargs, a)
	}
	m.ctrl.Call(m, "Fatal", varargs...)
}

// Fatal indicates an expected call of Fatal
func (mr *MockFieldLoggerMockRecorder) Fatal(args ...interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Fatal", reflect.TypeOf((*MockFieldLogger)(nil).Fatal), args...)
}

// Panic mocks base method
func (m *MockFieldLogger) Panic(args ...interface{}) {
	m.ctrl.T.Helper()
	varargs := []interface{}{}
	for _, a := range args {
		varargs = append(varargs, a)
	}
	m.ctrl.Call(m, "Panic", varargs...)
}

// Panic indicates an expected call of Panic
func (mr *MockFieldLoggerMockRecorder) Panic(args ...interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Panic", reflect.TypeOf((*MockFieldLogger)(nil).Panic), args...)
}

// Debugln mocks base method
func (m *MockFieldLogger) Debugln(args ...interface{}) {
	m.ctrl.T.Helper()
	varargs := []interface{}{}
	for _, a := range args {
		varargs = append(varargs, a)
	}
	m.ctrl.Call(m, "Debugln", varargs...)
}

// Debugln indicates an expected call of Debugln
func (mr *MockFieldLoggerMockRecorder) Debugln(args ...interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Debugln", reflect.TypeOf((*MockFieldLogger)(nil).Debugln), args...)
}

// Infoln mocks base method
func (m *MockFieldLogger) Infoln(args ...interface{}) {
	m.ctrl.T.Helper()
	varargs := []interface{}{}
	for _, a := range args {
		varargs = append(varargs, a)
	}
	m.ctrl.Call(m, "Infoln", varargs...)
}

// Infoln indicates an expected call of Infoln
func (mr *MockFieldLoggerMockRecorder) Infoln(args ...interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Infoln", reflect.TypeOf((*MockFieldLogger)(nil).Infoln), args...)
}

// Println mocks base method
func (m *MockFieldLogger) Println(args ...interface{}) {
	m.ctrl.T.Helper()
	varargs := []interface{}{}
	for _, a := range args {
		varargs = append(varargs, a)
	}
	m.ctrl.Call(m, "Println", varargs...)
}

// Println indicates an expected call of Println
func (mr *MockFieldLoggerMockRecorder) Println(args ...interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Println", reflect.TypeOf((*MockFieldLogger)(nil).Println), args...)
}

// Warnln mocks base method
func (m *MockFieldLogger) Warnln(args ...interface{}) {
	m.ctrl.T.Helper()
	varargs := []interface{}{}
	for _, a := range args {
		varargs = append(varargs, a)
	}
	m.ctrl.Call(m, "Warnln", varargs...)
}

// Warnln indicates an expected call of Warnln
func (mr *MockFieldLoggerMockRecorder) Warnln(args ...interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Warnln", reflect.TypeOf((*MockFieldLogger)(nil).Warnln), args...)
}

// Warningln mocks base method
func (m *MockFieldLogger) Warningln(args ...interface{}) {
	m.ctrl.T.Helper()
	varargs := []interface{}{}
	for _, a := range args {
		varargs = append(varargs, a)
	}
	m.ctrl.Call(m, "Warningln", varargs...)
}

// Warningln indicates an expected call of Warningln
func (mr *MockFieldLoggerMockRecorder) Warningln(args ...interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Warningln", reflect.TypeOf((*MockFieldLogger)(nil).Warningln), args...)
}

// Errorln mocks base method
func (m *MockFieldLogger) Errorln(args ...interface{}) {
	m.ctrl.T.Helper()
	varargs := []interface{}{}
	for _, a := range args {
		varargs = append(varargs, a)
	}
	m.ctrl.Call(m, "Errorln", varargs...)
}

// Errorln indicates an expected call of Errorln
func (mr *MockFieldLoggerMockRecorder) Errorln(args ...interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Errorln", reflect.TypeOf((*MockFieldLogger)(nil).Errorln), args...)
}

// Fatalln mocks base method
func (m *MockFieldLogger) Fatalln(args ...interface{}) {
	m.ctrl.T.Helper()
	varargs := []interface{}{}
	for _, a := range args {
		varargs = append(varargs, a)
	}
	m.ctrl.Call(m, "Fatalln", varargs...)
}

// Fatalln indicates an expected call of Fatalln
func (mr *MockFieldLoggerMockRecorder) Fatalln(args ...interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Fatalln", reflect.TypeOf((*MockFieldLogger)(nil).Fatalln), args...)
}

// Panicln mocks base method
func (m *MockFieldLogger) Panicln(args ...interface{}) {
	m.ctrl.T.Helper()
	varargs := []interface{}{}
	for _, a := range args {
		varargs = append(varargs, a)
	}
	m.ctrl.Call(m, "Panicln", varargs...)
}

// Panicln indicates an expected call of Panicln
func (mr *MockFieldLoggerMockRecorder) Panicln(args ...interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Panicln", reflect.TypeOf((*MockFieldLogger)(nil).Panicln), args...)
}

// MockExt1FieldLogger is a mock of Ext1FieldLogger interface
type MockExt1FieldLogger struct {
	ctrl     *gomock.Controller
	recorder *MockExt1FieldLoggerMockRecorder
}

// MockExt1FieldLoggerMockRecorder is the mock recorder for MockExt1FieldLogger
type MockExt1FieldLoggerMockRecorder struct {
	mock *MockExt1FieldLogger
}

// NewMockExt1FieldLogger creates a new mock instance
func NewMockExt1FieldLogger(ctrl *gomock.Controller) *MockExt1FieldLogger {
	mock := &MockExt1FieldLogger{ctrl: ctrl}
	mock.recorder = &MockExt1FieldLoggerMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use
func (m *MockExt1FieldLogger) EXPECT() *MockExt1FieldLoggerMockRecorder {
	return m.recorder
}

// WithField mocks base method
func (m *MockExt1FieldLogger) WithField(key string, value interface{}) *logrus.Entry {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "WithField", key, value)
	ret0, _ := ret[0].(*logrus.Entry)
	return ret0
}

// WithField indicates an expected call of WithField
func (mr *MockExt1FieldLoggerMockRecorder) WithField(key, value interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "WithField", reflect.TypeOf((*MockExt1FieldLogger)(nil).WithField), key, value)
}

// WithFields mocks base method
func (m *MockExt1FieldLogger) WithFields(fields logrus.Fields) *logrus.Entry {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "WithFields", fields)
	ret0, _ := ret[0].(*logrus.Entry)
	return ret0
}

// WithFields indicates an expected call of WithFields
func (mr *MockExt1FieldLoggerMockRecorder) WithFields(fields interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "WithFields", reflect.TypeOf((*MockExt1FieldLogger)(nil).WithFields), fields)
}

// WithError mocks base method
func (m *MockExt1FieldLogger) WithError(err error) *logrus.Entry {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "WithError", err)
	ret0, _ := ret[0].(*logrus.Entry)
	return ret0
}

// WithError indicates an expected call of WithError
func (mr *MockExt1FieldLoggerMockRecorder) WithError(err interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "WithError", reflect.TypeOf((*MockExt1FieldLogger)(nil).WithError), err)
}

// Debugf mocks base method
func (m *MockExt1FieldLogger) Debugf(format string, args ...interface{}) {
	m.ctrl.T.Helper()
	varargs := []interface{}{format}
	for _, a := range args {
		varargs = append(varargs, a)
	}
	m.ctrl.Call(m, "Debugf", varargs...)
}

// Debugf indicates an expected call of Debugf
func (mr *MockExt1FieldLoggerMockRecorder) Debugf(format interface{}, args ...interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	varargs := append([]interface{}{format}, args...)
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Debugf", reflect.TypeOf((*MockExt1FieldLogger)(nil).Debugf), varargs...)
}

// Infof mocks base method
func (m *MockExt1FieldLogger) Infof(format string, args ...interface{}) {
	m.ctrl.T.Helper()
	varargs := []interface{}{format}
	for _, a := range args {
		varargs = append(varargs, a)
	}
	m.ctrl.Call(m, "Infof", varargs...)
}

// Infof indicates an expected call of Infof
func (mr *MockExt1FieldLoggerMockRecorder) Infof(format interface{}, args ...interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	varargs := append([]interface{}{format}, args...)
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Infof", reflect.TypeOf((*MockExt1FieldLogger)(nil).Infof), varargs...)
}

// Printf mocks base method
func (m *MockExt1FieldLogger) Printf(format string, args ...interface{}) {
	m.ctrl.T.Helper()
	varargs := []interface{}{format}
	for _, a := range args {
		varargs = append(varargs, a)
	}
	m.ctrl.Call(m, "Printf", varargs...)
}

// Printf indicates an expected call of Printf
func (mr *MockExt1FieldLoggerMockRecorder) Printf(format interface{}, args ...interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	varargs := append([]interface{}{format}, args...)
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Printf", reflect.TypeOf((*MockExt1FieldLogger)(nil).Printf), varargs...)
}

// Warnf mocks base method
func (m *MockExt1FieldLogger) Warnf(format string, args ...interface{}) {
	m.ctrl.T.Helper()
	varargs := []interface{}{format}
	for _, a := range args {
		varargs = append(varargs, a)
	}
	m.ctrl.Call(m, "Warnf", varargs...)
}

// Warnf indicates an expected call of Warnf
func (mr *MockExt1FieldLoggerMockRecorder) Warnf(format interface{}, args ...interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	varargs := append([]interface{}{format}, args...)
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Warnf", reflect.TypeOf((*MockExt1FieldLogger)(nil).Warnf), varargs...)
}

// Warningf mocks base method
func (m *MockExt1FieldLogger) Warningf(format string, args ...interface{}) {
	m.ctrl.T.Helper()
	varargs := []interface{}{format}
	for _, a := range args {
		varargs = append(varargs, a)
	}
	m.ctrl.Call(m, "Warningf", varargs...)
}

// Warningf indicates an expected call of Warningf
func (mr *MockExt1FieldLoggerMockRecorder) Warningf(format interface{}, args ...interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	varargs := append([]interface{}{format}, args...)
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Warningf", reflect.TypeOf((*MockExt1FieldLogger)(nil).Warningf), varargs...)
}

// Errorf mocks base method
func (m *MockExt1FieldLogger) Errorf(format string, args ...interface{}) {
	m.ctrl.T.Helper()
	varargs := []interface{}{format}
	for _, a := range args {
		varargs = append(varargs, a)
	}
	m.ctrl.Call(m, "Errorf", varargs...)
}

// Errorf indicates an expected call of Errorf
func (mr *MockExt1FieldLoggerMockRecorder) Errorf(format interface{}, args ...interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	varargs := append([]interface{}{format}, args...)
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Errorf", reflect.TypeOf((*MockExt1FieldLogger)(nil).Errorf), varargs...)
}

// Fatalf mocks base method
func (m *MockExt1FieldLogger) Fatalf(format string, args ...interface{}) {
	m.ctrl.T.Helper()
	varargs := []interface{}{format}
	for _, a := range args {
		varargs = append(varargs, a)
	}
	m.ctrl.Call(m, "Fatalf", varargs...)
}

// Fatalf indicates an expected call of Fatalf
func (mr *MockExt1FieldLoggerMockRecorder) Fatalf(format interface{}, args ...interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	varargs := append([]interface{}{format}, args...)
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Fatalf", reflect.TypeOf((*MockExt1FieldLogger)(nil).Fatalf), varargs...)
}

// Panicf mocks base method
func (m *MockExt1FieldLogger) Panicf(format string, args ...interface{}) {
	m.ctrl.T.Helper()
	varargs := []interface{}{format}
	for _, a := range args {
		varargs = append(varargs, a)
	}
	m.ctrl.Call(m, "Panicf", varargs...)
}

// Panicf indicates an expected call of Panicf
func (mr *MockExt1FieldLoggerMockRecorder) Panicf(format interface{}, args ...interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	varargs := append([]interface{}{format}, args...)
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Panicf", reflect.TypeOf((*MockExt1FieldLogger)(nil).Panicf), varargs...)
}

// Debug mocks base method
func (m *MockExt1FieldLogger) Debug(args ...interface{}) {
	m.ctrl.T.Helper()
	varargs := []interface{}{}
	for _, a := range args {
		varargs = append(varargs, a)
	}
	m.ctrl.Call(m, "Debug", varargs...)
}

// Debug indicates an expected call of Debug
func (mr *MockExt1FieldLoggerMockRecorder) Debug(args ...interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Debug", reflect.TypeOf((*MockExt1FieldLogger)(nil).Debug), args...)
}

// Info mocks base method
func (m *MockExt1FieldLogger) Info(args ...interface{}) {
	m.ctrl.T.Helper()
	varargs := []interface{}{}
	for _, a := range args {
		varargs = append(varargs, a)
	}
	m.ctrl.Call(m, "Info", varargs...)
}

// Info indicates an expected call of Info
func (mr *MockExt1FieldLoggerMockRecorder) Info(args ...interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Info", reflect.TypeOf((*MockExt1FieldLogger)(nil).Info), args...)
}

// Print mocks base method
func (m *MockExt1FieldLogger) Print(args ...interface{}) {
	m.ctrl.T.Helper()
	varargs := []interface{}{}
	for _, a := range args {
		varargs = append(varargs, a)
	}
	m.ctrl.Call(m, "Print", varargs...)
}

// Print indicates an expected call of Print
func (mr *MockExt1FieldLoggerMockRecorder) Print(args ...interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Print", reflect.TypeOf((*MockExt1FieldLogger)(nil).Print), args...)
}

// Warn mocks base method
func (m *MockExt1FieldLogger) Warn(args ...interface{}) {
	m.ctrl.T.Helper()
	varargs := []interface{}{}
	for _, a := range args {
		varargs = append(varargs, a)
	}
	m.ctrl.Call(m, "Warn", varargs...)
}

// Warn indicates an expected call of Warn
func (mr *MockExt1FieldLoggerMockRecorder) Warn(args ...interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Warn", reflect.TypeOf((*MockExt1FieldLogger)(nil).Warn), args...)
}

// Warning mocks base method
func (m *MockExt1FieldLogger) Warning(args ...interface{}) {
	m.ctrl.T.Helper()
	varargs := []interface{}{}
	for _, a := range args {
		varargs = append(varargs, a)
	}
	m.ctrl.Call(m, "Warning", varargs...)
}

// Warning indicates an expected call of Warning
func (mr *MockExt1FieldLoggerMockRecorder) Warning(args ...interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Warning", reflect.TypeOf((*MockExt1FieldLogger)(nil).Warning), args...)
}

// Error mocks base method
func (m *MockExt1FieldLogger) Error(args ...interface{}) {
	m.ctrl.T.Helper()
	varargs := []interface{}{}
	for _, a := range args {
		varargs = append(varargs, a)
	}
	m.ctrl.Call(m, "Error", varargs...)
}

// Error indicates an expected call of Error
func (mr *MockExt1FieldLoggerMockRecorder) Error(args ...interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Error", reflect.TypeOf((*MockExt1FieldLogger)(nil).Error), args...)
}

// Fatal mocks base method
func (m *MockExt1FieldLogger) Fatal(args ...interface{}) {
	m.ctrl.T.Helper()
	varargs := []interface{}{}
	for _, a := range args {
		varargs = append(varargs, a)
	}
	m.ctrl.Call(m, "Fatal", varargs...)
}

// Fatal indicates an expected call of Fatal
func (mr *MockExt1FieldLoggerMockRecorder) Fatal(args ...interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Fatal", reflect.TypeOf((*MockExt1FieldLogger)(nil).Fatal), args...)
}

// Panic mocks base method
func (m *MockExt1FieldLogger) Panic(args ...interface{}) {
	m.ctrl.T.Helper()
	varargs := []interface{}{}
	for _, a := range args {
		varargs = append(varargs, a)
	}
	m.ctrl.Call(m, "Panic", varargs...)
}

// Panic indicates an expected call of Panic
func (mr *MockExt1FieldLoggerMockRecorder) Panic(args ...interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Panic", reflect.TypeOf((*MockExt1FieldLogger)(nil).Panic), args...)
}

// Debugln mocks base method
func (m *MockExt1FieldLogger) Debugln(args ...interface{}) {
	m.ctrl.T.Helper()
	varargs := []interface{}{}
	for _, a := range args {
		varargs = append(varargs, a)
	}
	m.ctrl.Call(m, "Debugln", varargs...)
}

// Debugln indicates an expected call of Debugln
func (mr *MockExt1FieldLoggerMockRecorder) Debugln(args ...interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Debugln", reflect.TypeOf((*MockExt1FieldLogger)(nil).Debugln), args...)
}

// Infoln mocks base method
func (m *MockExt1FieldLogger) Infoln(args ...interface{}) {
	m.ctrl.T.Helper()
	varargs := []interface{}{}
	for _, a := range args {
		varargs = append(varargs, a)
	}
	m.ctrl.Call(m, "Infoln", varargs...)
}

// Infoln indicates an expected call of Infoln
func (mr *MockExt1FieldLoggerMockRecorder) Infoln(args ...interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Infoln", reflect.TypeOf((*MockExt1FieldLogger)(nil).Infoln), args...)
}

// Println mocks base method
func (m *MockExt1FieldLogger) Println(args ...interface{}) {
	m.ctrl.T.Helper()
	varargs := []interface{}{}
	for _, a := range args {
		varargs = append(varargs, a)
	}
	m.ctrl.Call(m, "Println", varargs...)
}

// Println indicates an expected call of Println
func (mr *MockExt1FieldLoggerMockRecorder) Println(args ...interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Println", reflect.TypeOf((*MockExt1FieldLogger)(nil).Println), args...)
}

// Warnln mocks base method
func (m *MockExt1FieldLogger) Warnln(args ...interface{}) {
	m.ctrl.T.Helper()
	varargs := []interface{}{}
	for _, a := range args {
		varargs = append(varargs, a)
	}
	m.ctrl.Call(m, "Warnln", varargs...)
}

// Warnln indicates an expected call of Warnln
func (mr *MockExt1FieldLoggerMockRecorder) Warnln(args ...interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Warnln", reflect.TypeOf((*MockExt1FieldLogger)(nil).Warnln), args...)
}

// Warningln mocks base method
func (m *MockExt1FieldLogger) Warningln(args ...interface{}) {
	m.ctrl.T.Helper()
	varargs := []interface{}{}
	for _, a := range args {
		varargs = append(varargs, a)
	}
	m.ctrl.Call(m, "Warningln", varargs...)
}

// Warningln indicates an expected call of Warningln
func (mr *MockExt1FieldLoggerMockRecorder) Warningln(args ...interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Warningln", reflect.TypeOf((*MockExt1FieldLogger)(nil).Warningln), args...)
}

// Errorln mocks base method
func (m *MockExt1FieldLogger) Errorln(args ...interface{}) {
	m.ctrl.T.Helper()
	varargs := []interface{}{}
	for _, a := range args {
		varargs = append(varargs, a)
	}
	m.ctrl.Call(m, "Errorln", varargs...)
}

// Errorln indicates an expected call of Errorln
func (mr *MockExt1FieldLoggerMockRecorder) Errorln(args ...interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Errorln", reflect.TypeOf((*MockExt1FieldLogger)(nil).Errorln), args...)
}

// Fatalln mocks base method
func (m *MockExt1FieldLogger) Fatalln(args ...interface{}) {
	m.ctrl.T.Helper()
	varargs := []interface{}{}
	for _, a := range args {
		varargs = append(varargs, a)
	}
	m.ctrl.Call(m, "Fatalln", varargs...)
}

// Fatalln indicates an expected call of Fatalln
func (mr *MockExt1FieldLoggerMockRecorder) Fatalln(args ...interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Fatalln", reflect.TypeOf((*MockExt1FieldLogger)(nil).Fatalln), args...)
}

// Panicln mocks base method
func (m *MockExt1FieldLogger) Panicln(args ...interface{}) {
	m.ctrl.T.Helper()
	varargs := []interface{}{}
	for _, a := range args {
		varargs = append(varargs, a)
	}
	m.ctrl.Call(m, "Panicln", varargs...)
}

// Panicln indicates an expected call of Panicln
func (mr *MockExt1FieldLoggerMockRecorder) Panicln(args ...interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Panicln", reflect.TypeOf((*MockExt1FieldLogger)(nil).Panicln), args...)
}

// Tracef mocks base method
func (m *MockExt1FieldLogger) Tracef(format string, args ...interface{}) {
	m.ctrl.T.Helper()
	varargs := []interface{}{format}
	for _, a := range args {
		varargs = append(varargs, a)
	}
	m.ctrl.Call(m, "Tracef", varargs...)
}

// Tracef indicates an expected call of Tracef
func (mr *MockExt1FieldLoggerMockRecorder) Tracef(format interface{}, args ...interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	varargs := append([]interface{}{format}, args...)
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Tracef", reflect.TypeOf((*MockExt1FieldLogger)(nil).Tracef), varargs...)
}

// Trace mocks base method
func (m *MockExt1FieldLogger) Trace(args ...interface{}) {
	m.ctrl.T.Helper()
	varargs := []interface{}{}
	for _, a := range args {
		varargs = append(varargs, a)
	}
	m.ctrl.Call(m, "Trace", varargs...)
}

// Trace indicates an expected call of Trace
func (mr *MockExt1FieldLoggerMockRecorder) Trace(args ...interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Trace", reflect.TypeOf((*MockExt1FieldLogger)(nil).Trace), args...)
}

// Traceln mocks base method
func (m *MockExt1FieldLogger) Traceln(args ...interface{}) {
	m.ctrl.T.Helper()
	varargs := []interface{}{}
	for _, a := range args {
		varargs = append(varargs, a)
	}
	m.ctrl.Call(m, "Traceln", varargs...)
}

// Traceln indicates an expected call of Traceln
func (mr *MockExt1FieldLoggerMockRecorder) Traceln(args ...interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Traceln", reflect.TypeOf((*MockExt1FieldLogger)(nil).Traceln), args...)
}
