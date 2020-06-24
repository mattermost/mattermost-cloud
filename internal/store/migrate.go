// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.
//

package store

import (
	"github.com/blang/semver"
	"github.com/pkg/errors"
)

// LatestVersion returns the version to which the last migration migrates.
func LatestVersion() semver.Version {
	return migrations[len(migrations)-1].toVersion
}

// Migrate advances the schema of the configured database to the latest version.
func (sqlStore *SQLStore) Migrate() error {
	var currentVersion semver.Version
	if systemTableExists, err := sqlStore.tableExists("System"); err != nil {
		return errors.Wrap(err, "failed to check if system table exists")
	} else if systemTableExists {
		currentVersion, err = sqlStore.getCurrentVersion(sqlStore.db)
		if err != nil {
			return err
		}
	}

	sqlStore.logger.Infof(
		"Schema version is %s, latest version is %s",
		currentVersion,
		LatestVersion(),
	)

	applied := 0
	for _, migration := range migrations {
		if !currentVersion.EQ(migration.fromVersion) {
			continue
		}

		err := func() error {
			sqlStore.logger.Infof("Migrating schema from %s to %s", currentVersion, migration.toVersion)
			tx, err := sqlStore.db.Beginx()
			if err != nil {
				return errors.Wrapf(err, "failed to begin applying target version %s", migration.toVersion)
			}
			defer tx.Rollback()

			err = migration.migrationFunc(tx)
			if err != nil {
				return errors.Wrapf(err, "failed to migrate to target version %s", migration.toVersion)
			}

			currentVersion = migration.toVersion
			err = sqlStore.setCurrentVersion(tx, currentVersion.String())
			if err != nil {
				return errors.Wrap(err, "failed to record target version")
			}

			applied++
			err = tx.Commit()
			if err != nil {
				return errors.Wrapf(err, "failed to commit target version %s", migration.toVersion)
			}

			return nil
		}()

		if err != nil {
			return err
		}
	}

	if applied == 1 {
		sqlStore.logger.Info("Applied 1 migration")
	} else {
		sqlStore.logger.Infof("Applied %d migrations", applied)
	}

	return nil
}
