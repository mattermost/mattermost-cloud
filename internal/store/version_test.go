package store

import (
	"testing"

	"github.com/blang/semver"
	"github.com/stretchr/testify/require"
)

func TestCurrentVersion(t *testing.T) {
	sqlStore := makeUnmigratedSQLStore(t)

	currentVersion, err := sqlStore.GetCurrentVersion()
	require.NoError(t, err)
	require.Equal(t, semver.Version{}, currentVersion)

	currentVersion, err = sqlStore.getCurrentVersion(sqlStore.db)
	require.NoError(t, err)
	require.Equal(t, semver.Version{}, currentVersion)

	err = sqlStore.Migrate()
	require.NoError(t, err)

	currentVersion, err = sqlStore.GetCurrentVersion()
	require.NoError(t, err)
	require.Equal(t, LatestVersion(), currentVersion)

	currentVersion, err = sqlStore.getCurrentVersion(sqlStore.db)
	require.NoError(t, err)
	require.Equal(t, LatestVersion(), currentVersion)

	err = sqlStore.setCurrentVersion(sqlStore.db, "5.0.0")
	require.NoError(t, err)

	currentVersion, err = sqlStore.GetCurrentVersion()
	require.NoError(t, err)
	require.Equal(t, semver.MustParse("5.0.0"), currentVersion)

	currentVersion, err = sqlStore.getCurrentVersion(sqlStore.db)
	require.NoError(t, err)
	require.Equal(t, semver.MustParse("5.0.0"), currentVersion)
}
