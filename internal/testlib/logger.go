package testlib

import (
	"strings"
	"testing"

	log "github.com/sirupsen/logrus"
)

// testingWriter is an io.Writer that writes through t.Log
type testingWriter struct {
	tb testing.TB
}

func (tw *testingWriter) Write(b []byte) (int, error) {
	tw.tb.Log(strings.TrimSpace(string(b)))
	return len(b), nil
}

// MakeLogger creates a log.FieldLogger tthat routes to tb.Log.
func MakeLogger(tb testing.TB) log.FieldLogger {
	logger := log.New()
	logger.SetOutput(&testingWriter{tb})

	return logger
}
