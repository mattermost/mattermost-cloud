// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.
//

package store

import (
	"github.com/blang/semver"
	"github.com/pkg/errors"
)

const systemDatabaseVersionKey = "DatabaseVersion"

// GetCurrentVersion queries the System table for the current database version.
func (sqlStore *SQLStore) GetCurrentVersion() (semver.Version, error) {
	return sqlStore.getCurrentVersion(sqlStore.db)
}

// getCurrentVersion queries the System table for the current database version against the given
// queryer.
func (sqlStore *SQLStore) getCurrentVersion(q queryer) (semver.Version, error) {
	currentVersionStr, err := sqlStore.getSystemValue(q, systemDatabaseVersionKey)
	if currentVersionStr == "" {
		return semver.Version{}, nil
	}

	currentVersion, err := semver.Parse(currentVersionStr)
	if err != nil {
		return semver.Version{}, errors.Wrapf(err, "failed to parse current version %s", currentVersionStr)
	}

	return currentVersion, nil
}

// setCurrentVersion updates the System table with the given database version.
func (sqlStore *SQLStore) setCurrentVersion(e execer, version string) error {
	return sqlStore.setSystemValue(e, systemDatabaseVersionKey, version)
}
