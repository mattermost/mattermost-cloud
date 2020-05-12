package store

import (
	"database/sql"
	"fmt"

	sq "github.com/Masterminds/squirrel"
	"github.com/mattermost/mattermost-cloud/model"
	"github.com/pkg/errors"
)

var multitenantDatabaseSelect sq.SelectBuilder

func init() {
	multitenantDatabaseSelect = sq.
		Select("ID", "RawInstallationIDs", "LockAcquiredBy", "LockAcquiredAt").
		From("MultitenantDatabase")
}

// GetMultitenantDatabase fetches the given database cluster by id.
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

// GetMultitenantDatabases fetches the given page of created database clusters. The first page is 0.
func (sqlStore *SQLStore) GetMultitenantDatabases(filter *model.MultitenantDatabaseFilter) ([]*model.MultitenantDatabase, error) {
	builder := multitenantDatabaseSelect

	if filter != nil && len(filter.InstallationID) > 0 {
		builder = builder.Where("RawInstallationIDs LIKE ?", fmt.Sprint("%", filter.InstallationID, "%"))
	}

	var multitenantDatabases []*model.MultitenantDatabase

	err := sqlStore.selectBuilder(sqlStore.db, &multitenantDatabases, builder)
	if err != nil {
		return nil, errors.Wrap(err, "failed to query multitenant databases")
	}

	if filter != nil && filter.NumOfInstallationsLimit > 0 {
		for i, database := range multitenantDatabases {
			installationIDs, err := database.GetInstallationIDs()
			if err != nil {
				return nil, errors.Wrap(err, "failed to query multitenant databases")
			}

			if len(installationIDs) > int(filter.NumOfInstallationsLimit) {
				multitenantDatabases = append(multitenantDatabases[:i], multitenantDatabases[i+1:]...)
			}
		}
	}

	return multitenantDatabases, nil
}

// CreateMultitenantDatabase records the supplied multitenant database to the datastore.
func (sqlStore *SQLStore) CreateMultitenantDatabase(multitenantDatabase *model.MultitenantDatabase) error {
	if multitenantDatabase == nil {
		return errors.New("multitenant database cannot be nil")
	}

	if len(multitenantDatabase.ID) < 1 {
		return errors.New("multitenant database ID cannot be nil")
	}

	if len(multitenantDatabase.RawInstallationIDs) < 1 {
		multitenantDatabase.RawInstallationIDs = make([]byte, 0)
	}

	_, err := sqlStore.execBuilder(sqlStore.db, sq.
		Insert("MultitenantDatabase").
		SetMap(map[string]interface{}{
			"ID":                 multitenantDatabase.ID,
			"RawInstallationIDs": multitenantDatabase.RawInstallationIDs,
			"LockAcquiredBy":     nil,
			"LockAcquiredAt":     0,
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

	if len(multitenantDatabase.RawInstallationIDs) < 1 {
		multitenantDatabase.RawInstallationIDs = make([]byte, 0)
	}

	_, err := sqlStore.execBuilder(sqlStore.db, sq.
		Update("MultitenantDatabase").
		SetMap(map[string]interface{}{
			"RawInstallationIDs": multitenantDatabase.RawInstallationIDs,
		}).
		Where("ID = ?", multitenantDatabase.ID),
	)
	if err != nil {
		return errors.Wrap(err, "failed to update multitenant database")
	}

	return nil
}

// AddMultitenantDatabaseInstallationID adds an installation ID to a multitenant database.
func (sqlStore *SQLStore) AddMultitenantDatabaseInstallationID(multitenantID, installationID string) (model.MultitenantDatabaseInstallationIDs, error) {
	multitenantDatabase, err := sqlStore.GetMultitenantDatabase(multitenantID)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to add installation ID %s", installationID)
	}

	multitenantDatabaseInstallationIDs, err := multitenantDatabase.GetInstallationIDs()
	if err != nil {
		return nil, errors.Wrapf(err, "failed to add installation ID %s", installationID)
	}

	if !multitenantDatabaseInstallationIDs.Contains(installationID) {
		multitenantDatabaseInstallationIDs.Add(installationID)

		err = multitenantDatabase.SetInstallationIDs(multitenantDatabaseInstallationIDs)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to add installation ID %s", installationID)
		}

		err = sqlStore.UpdateMultitenantDatabase(multitenantDatabase)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to add installation ID %s", installationID)
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

// LockMultitenantDatabase marks the database cluster as locked for exclusive use by the caller.
func (sqlStore *SQLStore) LockMultitenantDatabase(installationID, lockerID string) (bool, error) {
	return sqlStore.lockRows("MultitenantDatabase", []string{installationID}, lockerID)
}

// UnlockMultitenantDatabase releases a lock previously acquired against a caller.
func (sqlStore *SQLStore) UnlockMultitenantDatabase(installationID, lockerID string, force bool) (bool, error) {
	return sqlStore.unlockRows("MultitenantDatabase", []string{installationID}, lockerID, force)
}
