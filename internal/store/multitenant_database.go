// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.
//

package store

import (
	"database/sql"
	"fmt"

	"github.com/mattermost/mattermost-cloud/model"

	sq "github.com/Masterminds/squirrel"
	"github.com/pkg/errors"
)

var multitenantDatabaseSelect sq.SelectBuilder

func init() {
	multitenantDatabaseSelect = sq.
		Select("ID", "VpcID", "RawInstallationIDs", "CreateAt", "DeleteAt", "LockAcquiredBy", "LockAcquiredAt").
		From("MultitenantDatabase")
}

// GetMultitenantDatabase fetches the given multitenant database by id.
func (sqlStore *SQLStore) GetMultitenantDatabase(id string) (*model.MultitenantDatabase, error) {
	var multitenantDatabase model.MultitenantDatabase
	err := sqlStore.getBuilder(sqlStore.db, &multitenantDatabase, multitenantDatabaseSelect.Where("ID = ?", id))
	if err == sql.ErrNoRows {
		return nil, nil
	} else if err != nil {
		return nil, errors.Wrap(err, "failed to get multitenant database by id")
	}

	return &multitenantDatabase, nil
}

// GetMultitenantDatabases fetches the given page of created multitenant database. The first page is 0.
func (sqlStore *SQLStore) GetMultitenantDatabases(filter *model.MultitenantDatabaseFilter) ([]*model.MultitenantDatabase, error) {
	builder := multitenantDatabaseSelect.
		OrderBy("CreateAt ASC")

	if filter.PerPage != model.AllPerPage {
		builder = builder.
			Limit(uint64(filter.PerPage)).
			Offset(uint64(filter.Page * filter.PerPage))
	}

	if filter != nil {
		if len(filter.InstallationID) > 0 {
			builder = builder.
				Where(sq.Like{"RawInstallationIDs": fmt.Sprint("%", filter.InstallationID, "%")})
		}

		if len(filter.LockerID) > 0 {
			builder = builder.
				Where(sq.Eq{"LockAcquiredBy": filter.LockerID})
		}

		if len(filter.VpcID) > 0 {
			builder = builder.
				Where(sq.Eq{"VpcID": filter.VpcID})
		}
	}

	var databases []*model.MultitenantDatabase

	err := sqlStore.selectBuilder(sqlStore.db, &databases, builder)
	if err != nil {
		return nil, errors.Wrap(err, "failed to query multitenant databases")
	}

	if filter.NumOfInstallationsLimit != model.NoInstallationsLimit {
		filteredDatabases := databases[:0]
		for _, database := range databases {
			installationIDs, err := database.GetInstallationIDs()
			if err != nil {
				return nil, errors.Wrap(err, "failed to query multitenant databases")
			}

			if len(installationIDs) < int(filter.NumOfInstallationsLimit) {
				filteredDatabases = append(filteredDatabases, database)
			}
		}
		databases = filteredDatabases
	}

	return databases, nil
}

// CreateMultitenantDatabase records the supplied multitenant database to the datastore.
func (sqlStore *SQLStore) CreateMultitenantDatabase(multitenantDatabase *model.MultitenantDatabase) error {
	if multitenantDatabase == nil {
		return errors.New("multitenant database must not be nil")
	}

	if multitenantDatabase.ID == "" {
		return errors.New("multitenant database ID must not be empty")
	}

	multitenantDatabase.CreateAt = GetMillis()

	if len(multitenantDatabase.RawInstallationIDs) < 1 {
		multitenantDatabase.RawInstallationIDs = make([]byte, 0)
	}

	_, err := sqlStore.execBuilder(sqlStore.db, sq.
		Insert("MultitenantDatabase").
		SetMap(map[string]interface{}{
			"ID":                 multitenantDatabase.ID,
			"VpcID":              multitenantDatabase.VpcID,
			"RawInstallationIDs": multitenantDatabase.RawInstallationIDs,
			"LockAcquiredBy":     nil,
			"LockAcquiredAt":     0,
			"CreateAt":           multitenantDatabase.CreateAt,
			"DeleteAt":           0,
		}),
	)
	if err != nil {
		return errors.Wrap(err, "failed to create multitenant database")
	}

	return nil
}

// UpdateMultitenantDatabase updates a already existent multitenant database in the datastore.
func (sqlStore *SQLStore) UpdateMultitenantDatabase(multitenantDatabase *model.MultitenantDatabase) error {
	if multitenantDatabase == nil {
		return errors.New("multitenant database cannot be nil")
	}
	if multitenantDatabase.LockAcquiredBy == nil {
		return errors.New("multitenant database is not locked")
	}

	if len(multitenantDatabase.RawInstallationIDs) < 1 {
		multitenantDatabase.RawInstallationIDs = make([]byte, 0)
	} else {
		_, err := multitenantDatabase.GetInstallationIDs()
		if err != nil {
			return errors.Wrap(err, "failed to parse raw installation ids")
		}
	}

	res, err := sqlStore.execBuilder(sqlStore.db, sq.
		Update("MultitenantDatabase").
		SetMap(map[string]interface{}{
			"RawInstallationIDs": multitenantDatabase.RawInstallationIDs,
		}).
		Where(sq.Eq{"ID": multitenantDatabase.ID}).
		Where(sq.Eq{"LockAcquiredBy": *multitenantDatabase.LockAcquiredBy}),
	)
	if err != nil {
		return errors.Wrap(err, "failed to update multitenant database")
	}

	rowsAffected, err := res.RowsAffected()
	if err != nil {
		return errors.Wrap(err, "failed to update multitenant database")
	}
	if rowsAffected < 1 {
		return errors.New("failed to update multitenant database: no rows affected")
	}

	return nil
}

// AddMultitenantDatabaseInstallationID adds an installation ID to a multitenant database.
func (sqlStore *SQLStore) AddMultitenantDatabaseInstallationID(multitenantID, installationID string) (model.MultitenantDatabaseInstallationIDs, error) {
	multitenantDatabase, err := sqlStore.GetMultitenantDatabase(multitenantID)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get installations from database")
	}
	if multitenantDatabase == nil {
		return nil, errors.Errorf("unable to find multitenant database ID %s", multitenantID)
	}

	multitenantDatabaseInstallationIDs, err := multitenantDatabase.GetInstallationIDs()
	if err != nil {
		return nil, errors.Wrap(err, "failed to get installations from database")
	}

	if !multitenantDatabaseInstallationIDs.Contains(installationID) {
		multitenantDatabaseInstallationIDs.Add(installationID)

		err = multitenantDatabase.SetInstallationIDs(multitenantDatabaseInstallationIDs)
		if err != nil {
			return nil, errors.Wrap(err, "failed to get installations from database")
		}

		err = sqlStore.UpdateMultitenantDatabase(multitenantDatabase)
		if err != nil {
			return nil, errors.Wrap(err, "failed to get installations from database")
		}
	}

	return multitenantDatabaseInstallationIDs, nil
}

// RemoveMultitenantDatabaseInstallationID removes an installation ID from a multitenant database.
func (sqlStore *SQLStore) RemoveMultitenantDatabaseInstallationID(multitenantID, installationID string) (model.MultitenantDatabaseInstallationIDs, error) {
	multitenantDatabase, err := sqlStore.GetMultitenantDatabase(multitenantID)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to remove installation ID %s", installationID)
	}
	if multitenantDatabase == nil {
		return nil, errors.Errorf("unable to find multitenant database ID %s", multitenantID)
	}

	multitenantDatabaseInstallationIDs, err := multitenantDatabase.GetInstallationIDs()
	if err != nil {
		return nil, errors.Wrapf(err, "failed to remove installation ID %s", installationID)
	}

	if multitenantDatabaseInstallationIDs.Contains(installationID) {
		multitenantDatabaseInstallationIDs.Remove(installationID)

		err = multitenantDatabase.SetInstallationIDs(multitenantDatabaseInstallationIDs)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to remove installation ID %s", installationID)
		}

		err = sqlStore.UpdateMultitenantDatabase(multitenantDatabase)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to remove installation ID %s", installationID)
		}
	}

	return multitenantDatabaseInstallationIDs, nil
}

// GetMultitenantDatabaseForInstallationID fetches the multitenant database associated with an installation ID.
// If more than one multitenant database per installation exists, this function returns an error.
func (sqlStore *SQLStore) GetMultitenantDatabaseForInstallationID(installationID string) (*model.MultitenantDatabase, error) {
	multitenantDatabases, err := sqlStore.GetMultitenantDatabases(&model.MultitenantDatabaseFilter{
		InstallationID:          installationID,
		NumOfInstallationsLimit: model.NoInstallationsLimit,
		PerPage:                 model.AllPerPage,
	})
	if err != nil {
		return nil, errors.Wrapf(err, "unable to get multitenant database for installation ID %s in sql store", installationID)
	}
	if len(multitenantDatabases) != 1 {
		return nil, errors.Errorf("expected exactly one multitenant database per installation (found %d)", len(multitenantDatabases))
	}

	return multitenantDatabases[0], nil
}

// LockMultitenantDatabase marks the database cluster as locked for exclusive use by the caller.
func (sqlStore *SQLStore) LockMultitenantDatabase(multitenantDatabaseID, instanceID string) (bool, error) {
	return sqlStore.lockRows("MultitenantDatabase", []string{multitenantDatabaseID}, instanceID)
}

// UnlockMultitenantDatabase releases a lock previously acquired against a caller.
func (sqlStore *SQLStore) UnlockMultitenantDatabase(multitenantDatabaseID, instanceID string, force bool) (bool, error) {
	return sqlStore.unlockRows("MultitenantDatabase", []string{multitenantDatabaseID}, instanceID, force)
}
