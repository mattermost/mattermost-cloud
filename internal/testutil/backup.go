// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.
//

package testutil

import (
	"fmt"
	"testing"

	"github.com/pborman/uuid"

	"github.com/mattermost/mattermost-cloud/internal/store"
	"github.com/mattermost/mattermost-cloud/model"
	"github.com/stretchr/testify/require"
)

// CreateBackupCompatibleInstallation is a helper function for tests which creates Installation compatible with backup.
func CreateBackupCompatibleInstallation(t *testing.T, sqlStore *store.SQLStore) *model.Installation {
	installation := &model.Installation{
		Database:  model.InstallationDatabaseMultiTenantRDSPostgres,
		Filestore: model.InstallationFilestoreBifrost,
		State:     model.InstallationStateHibernating,
		DNS:       fmt.Sprintf("dns-%s", uuid.NewRandom().String()[:6]),
	}
	err := sqlStore.CreateInstallation(installation, nil)
	require.NoError(t, err)
	return installation
}
