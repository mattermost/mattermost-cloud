package store

import (
	"os"
	"strings"
	"testing"

	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/require"
)

// testingWriter is an io.Writer that writes through t.Log
type testingWriter struct {
	tb testing.TB
}

func (tw *testingWriter) Write(b []byte) (int, error) {
	tw.tb.Log(strings.TrimSpace(string(b)))
	return len(b), nil
}

func makeLogger(tb testing.TB) logrus.FieldLogger {
	logger := logrus.New()
	logger.SetOutput(&testingWriter{tb})

	return logger
}

func makeUnmigratedSQLStore(tb testing.TB) *SQLStore {
	dsn := os.Getenv("CLOUD_DATABASE")
	if dsn == "" {
		dsn = "sqlite://:memory:/"
	}

	sqlStore, err := New(dsn, makeLogger(tb))
	require.NoError(tb, err)

	if sqlStore.db.DriverName() == "postgres" {
		// Force the use of the current session's temporary-table schema, simplifying
		// cleanup.
		sqlStore.db.Exec("SET search_path TO pg_temp")
	}

	return sqlStore
}

func makeSQLStore(tb testing.TB) *SQLStore {
	sqlStore := makeUnmigratedSQLStore(tb)
	err := sqlStore.Migrate()
	require.NoError(tb, err)

	return sqlStore
}
