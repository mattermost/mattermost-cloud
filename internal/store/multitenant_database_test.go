package store

import (
	"testing"
	"time"

	"github.com/mattermost/mattermost-cloud/internal/testlib"
	"github.com/mattermost/mattermost-cloud/model"
	"github.com/stretchr/testify/require"
)

func TestMultitenantDatabase(t *testing.T) {
	logger := testlib.MakeLogger(t)
	sqlStore := MakeTestSQLStore(t, logger)

	installationID1 := model.NewID()
	installationID2 := model.NewID()
	installationID3 := model.NewID()
	installationID4 := model.NewID()
	installationID5 := model.NewID()

	database1 := model.MultitenantDatabase{
		ID: model.NewID(),
	}
	database1.SetInstallationIDs(model.MultitenantDatabaseInstallationIDs{installationID1, installationID2})

	err := sqlStore.CreateMultitenantDatabase(&database1)
	require.NoError(t, err)

	err = sqlStore.CreateMultitenantDatabase(&database1)
	require.Error(t, err)
	require.Equal(t, "failed to create multitenant database: UNIQUE constraint failed: MultitenantDatabase.ID", err.Error())

	time.Sleep(1 * time.Millisecond)

	database2 := model.MultitenantDatabase{
		ID: model.NewID(),
	}
	database2.SetInstallationIDs(model.MultitenantDatabaseInstallationIDs{installationID3, installationID4, installationID5})

	err = sqlStore.CreateMultitenantDatabase(&database2)
	require.NoError(t, err)

	t.Run("get multitenant database", func(t *testing.T) {
		database, err := sqlStore.GetMultitenantDatabase(database1.ID)
		require.NoError(t, err)
		require.NotNil(t, database)
		require.Equal(t, database1, *database)
	})

	t.Run("get multitenant databases", func(t *testing.T) {
		databases, err := sqlStore.GetMultitenantDatabases(&model.MultitenantDatabaseFilter{
			PerPage: model.AllPerPage,
		})
		require.NoError(t, err)
		require.NotNil(t, databases)
		require.Equal(t, 2, len(databases))
	})
}
