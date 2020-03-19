package testlib

import (
	"strings"
	"testing"

	"github.com/golang/mock/gomock"
	mocks "github.com/mattermost/mattermost-cloud/internal/mocks/logger"
	"github.com/sirupsen/logrus"
	log "github.com/sirupsen/logrus"
)

// testingWriter is an io.Writer that writes through t.Log.
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

// MockedFieldLogger supplies a mocked library for testing logs.
type MockedFieldLogger struct {
	Logger *mocks.MockFieldLogger
}

// NewMockedFieldLogger returns a instance of FieldLogger for testing.
func NewMockedFieldLogger(ctrl *gomock.Controller) *MockedFieldLogger {
	return &MockedFieldLogger{
		Logger: mocks.NewMockFieldLogger(ctrl),
	}
}

// NewLoggerEntry returns a new logger entry instance.
func NewLoggerEntry() *logrus.Entry {
	return logrus.NewEntry(logrus.New())
}
