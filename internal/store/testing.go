// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.
//

package store

import (
	"fmt"
	"net/url"
	"os"
	"testing"

	"github.com/mattermost/mattermost-cloud/model"
	log "github.com/sirupsen/logrus"
	"github.com/stretchr/testify/require"
)

func makeUnmigratedTestSQLStore(tb testing.TB, logger log.FieldLogger) *SQLStore {
	// TODO: fix this dirty workaround
	// https://github.com/golang/go/issues/33633
	dsn := os.Getenv("CLOUD_DATABASE")
	if dsn == "" {
		dsn = fmt.Sprintf("sqlite3://file:%s.db?mode=memory&cache=shared", model.NewID())
	} else {
		dsnURL, err := url.Parse(dsn)
		require.NoError(tb, err)

		switch dsnURL.Scheme {
		case "sqlite", "sqlite3":
		case "postgres", "postgresql":
			q := dsnURL.Query()
			q.Add("pg_temp", "true")
			dsnURL.RawQuery = q.Encode()
			dsn = dsnURL.String()
		}
	}

	sqlStore, err := New(dsn, logger)
	require.NoError(tb, err)

	// For testing with mode=memory and pg_temp above, restrict to a single connection,
	// otherwise multiple goroutines may not see consistent views / have consistent access.
	// Technically, this is redundant for sqlite3, given that we force this anyway.
	sqlStore.db.SetMaxOpenConns(1)

	return sqlStore
}

// MakeTestSQLStore creates a SQLStore for use with unit tests.
func MakeTestSQLStore(tb testing.TB, logger log.FieldLogger) *SQLStore {
	sqlStore := makeUnmigratedTestSQLStore(tb, logger)
	err := sqlStore.Migrate()
	require.NoError(tb, err)

	return sqlStore
}
