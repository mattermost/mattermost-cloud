// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.
//

package components

import (
	"github.com/mattermost/mattermost-cloud/model"
	"github.com/pkg/errors"
)

type dbMigrationValidationStore interface {
	GetInstallationsTotalDatabaseWeight(installationIDs []string) (float64, error)
}

// ValidateDBMigrationDestination validates if installation can be migrated to destinationDB.
func ValidateDBMigrationDestination(store dbMigrationValidationStore, destinationDB *model.MultitenantDatabase, installationID string, maxWeight float64) error {
	if Contains(destinationDB.MigratedInstallations, installationID) {
		return errors.Errorf("installation %q still exists in migrated installations for %q database, clean it up before migration", installationID, destinationDB.ID)
	}

	weight, err := store.GetInstallationsTotalDatabaseWeight(destinationDB.Installations)
	if err != nil {
		return errors.Wrap(err, "failed to check total weight of installations in destination database")
	}
	if weight >= maxWeight {
		return errors.Errorf("cannot migrate to database, installations weight reached the limit: %f", weight)
	}

	return nil
}
