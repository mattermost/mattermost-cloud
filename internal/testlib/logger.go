package testlib

import (
	"strings"
	"testing"

	"github.com/sirupsen/logrus"
	log "github.com/sirupsen/logrus"
	"github.com/stretchr/testify/mock"

	mocks "github.com/mattermost/mattermost-cloud/internal/mocks/logger"
)

// testingWriter is an io.Writer that writes through t.Log
type testingWriter struct {
	tb testing.TB
}

func (tw *testingWriter) Write(b []byte) (int, error) {
	tw.tb.Log(strings.TrimSpace(string(b)))
	return len(b), nil
}

// MakeLogger creates a log.FieldLogger that routes to tb.Log.
func MakeLogger(tb testing.TB) log.FieldLogger {
	logger := log.New()
	logger.SetOutput(&testingWriter{tb})
	logger.SetLevel(log.TraceLevel)

	return logger
}

// MockedFieldLogger supplies a mocked library for testing logs
type MockedFieldLogger struct {
	Logger *mocks.FieldLogger
}

// NewMockedFieldLogger returns a instance of FieldLogger for testing.
func NewMockedFieldLogger() *MockedFieldLogger {
	return &MockedFieldLogger{
		Logger: &mocks.FieldLogger{},
	}
}

// WithFieldArgs set expectations for WithField by passing name and arguments
func (m *MockedFieldLogger) WithFieldArgs(name string, args ...string) *mock.Call {
	return m.Logger.Mock.On("WithField", name, args).Return(logrus.NewEntry(&logrus.Logger{}))
}

// WithFields set expectations for WithFields by passing logrus.Fields.
func (m *MockedFieldLogger) WithFields(fields logrus.Fields) *mock.Call {
	return m.Logger.Mock.On("WithFields", fields).Return(logrus.NewEntry(&logrus.Logger{}))
}

// WithFieldString set expectations for WithField by passing name and value
func (m *MockedFieldLogger) WithFieldString(name string, value string) *mock.Call {
	return m.Logger.Mock.On("WithField", name, value).Return(logrus.NewEntry(&logrus.Logger{}))
}

// InfofString set expectations for Infof by passing name and value
func (m *MockedFieldLogger) InfofString(name string, value string) *mock.Call {
	return m.Logger.Mock.On("Infof", name, value).Return(logrus.NewEntry(&logrus.Logger{}))
}
