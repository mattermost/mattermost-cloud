// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.
//

package common

import (
	"testing"

	"github.com/mattermost/mattermost-cloud/internal/store"
	"github.com/mattermost/mattermost-cloud/internal/testlib"
	"github.com/mattermost/mattermost-cloud/internal/testutil"
	"github.com/mattermost/mattermost-cloud/model"
	"github.com/stretchr/testify/require"
)

func TestValidateDBMigrationDestination(t *testing.T) {
	logger := testlib.MakeLogger(t)
	sqlStore := store.MakeTestSQLStore(t, logger)
	defer store.CloseConnection(t, sqlStore)

	installation := &model.Installation{
		Name:  "dns",
		State: model.InstallationStateStable,
	}
	err := sqlStore.CreateInstallation(installation, nil, testutil.DNSForInstallation("dns.com"))
	require.NoError(t, err)

	database := &model.MultitenantDatabase{
		ID:                    "database1",
		DatabaseType:          model.InstallationDatabaseMultiTenantRDSPostgres,
		Installations:         model.MultitenantDatabaseInstallations{installation.ID},
		MigratedInstallations: model.MultitenantDatabaseInstallations{"migrated"},
	}
	err = sqlStore.CreateMultitenantDatabase(database)
	require.NoError(t, err)

	err = ValidateDBMigrationDestination(sqlStore, database, "installation", 10)
	require.NoError(t, err)

	t.Run("max weight reached", func(t *testing.T) {
		err = ValidateDBMigrationDestination(sqlStore, database, "installation", 1)
		require.Error(t, err)
	})

	t.Run("installation already in migrated installations", func(t *testing.T) {
		err = ValidateDBMigrationDestination(sqlStore, database, "migrated", 10)
		require.Error(t, err)
	})
}
