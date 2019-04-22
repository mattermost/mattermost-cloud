package api_test

import (
	"net/url"
	"os"
	"strings"
	"testing"

	"github.com/mattermost/mattermost-cloud/internal/store"
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

func makeUnmigratedSQLStore(tb testing.TB) *store.SQLStore {
	dsn := os.Getenv("CLOUD_DATABASE")
	if dsn == "" {
		dsn = "sqlite://:memory:/"
	}

	dsnURL, err := url.Parse(dsn)
	require.NoError(tb, err)

	// Unconditionally add the pg_temp flag for PostgreSQL databases. It won't affect sqlite.
	q := dsnURL.Query()
	q.Add("pg_temp", "true")
	dsnURL.RawQuery = q.Encode()
	dsn = dsnURL.String()

	sqlStore, err := store.New(dsn, makeLogger(tb))
	require.NoError(tb, err)

	return sqlStore
}

func makeSQLStore(tb testing.TB) *store.SQLStore {
	sqlStore := makeUnmigratedSQLStore(tb)
	err := sqlStore.Migrate()
	require.NoError(tb, err)

	return sqlStore
}
