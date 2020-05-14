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
		Select("ID", "RawInstallationIDs", "CreateAt", "DeleteAt", "LockAcquiredBy", "LockAcquiredAt").
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

	if filter != nil && len(filter.InstallationID) > 0 {
		builder = builder.
			Where("RawInstallationIDs LIKE ?", fmt.Sprint("%", filter.InstallationID, "%"))
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
	multitenantDatabase.CreateAt = GetMillis()

	if multitenantDatabase == nil {
		return errors.New("multitenant database must not be nil")
	}

	if multitenantDatabase.ID == "" {
		return errors.New("multitenant database ID must not be empty")
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
		return nil, errors.Wrap(err, "failed to get installations from database")
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
		InstallationID: installationID,
		PerPage:        model.AllPerPage,
	})
	if err != nil {
		return nil, errors.Wrapf(err, "unable to find multitenant database for installation ID %s", installationID)
	}
	if len(multitenantDatabases) != 1 {
		return nil, errors.Errorf("expected exactly one multitenant database per installation (found %d)", len(multitenantDatabases))
	}

	return multitenantDatabases[0], nil
}

// LockMultitenantDatabase marks the database cluster as locked for exclusive use by the caller.
func (sqlStore *SQLStore) LockMultitenantDatabase(installationID, lockerID string) (bool, error) {
	return sqlStore.lockRows("MultitenantDatabase", []string{installationID}, lockerID)
}

// UnlockMultitenantDatabase releases a lock previously acquired against a caller.
func (sqlStore *SQLStore) UnlockMultitenantDatabase(installationID, lockerID string, force bool) (bool, error) {
	return sqlStore.unlockRows("MultitenantDatabase", []string{installationID}, lockerID, force)
}
