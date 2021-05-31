// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.
//

package store

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"math"

	"github.com/mattermost/mattermost-cloud/model"

	sq "github.com/Masterminds/squirrel"
	"github.com/pkg/errors"
)

var multitenantDatabaseSelect sq.SelectBuilder

func init() {
	multitenantDatabaseSelect = sq.
		Select(
			"ID",
			"VpcID",
			"DatabaseType",
			"State",
			"InstallationsRaw",
			"MigratedInstallationsRaw",
			"SharedLogicalDatabaseMappingsRaw",
			"MaxInstallationsPerLogicalDatabase",
			"WriterEndpoint",
			"ReaderEndpoint",
			"CreateAt",
			"DeleteAt",
			"LockAcquiredBy",
			"LockAcquiredAt").
		From("MultitenantDatabase")
}

type rawMultitenantDatabase struct {
	*model.MultitenantDatabase
	InstallationsRaw                 []byte
	MigratedInstallationsRaw         []byte
	SharedLogicalDatabaseMappingsRaw []byte
}

type rawMultitenantDatabases []*rawMultitenantDatabase

func (r *rawMultitenantDatabase) toMultitenantDatabase() (*model.MultitenantDatabase, error) {
	// We only need to set values that are converted from a raw database format.
	if r.InstallationsRaw != nil {
		err := json.Unmarshal(r.InstallationsRaw, &r.MultitenantDatabase.Installations)
		if err != nil {
			return nil, err
		}
	}
	if r.MigratedInstallationsRaw != nil {
		err := json.Unmarshal(r.MigratedInstallationsRaw, &r.MultitenantDatabase.MigratedInstallations)
		if err != nil {
			return nil, err
		}
	}
	if r.SharedLogicalDatabaseMappingsRaw != nil {
		err := json.Unmarshal(r.SharedLogicalDatabaseMappingsRaw, &r.MultitenantDatabase.SharedLogicalDatabaseMappings)
		if err != nil {
			return nil, err
		}
	}

	return r.MultitenantDatabase, nil
}

func (rs *rawMultitenantDatabases) toMultitenantDatabases() ([]*model.MultitenantDatabase, error) {
	var databases []*model.MultitenantDatabase
	for _, rawInstallation := range *rs {
		database, err := rawInstallation.toMultitenantDatabase()
		if err != nil {
			return nil, err
		}
		databases = append(databases, database)
	}

	return databases, nil
}

// GetMultitenantDatabase fetches the given multitenant database by id.
func (sqlStore *SQLStore) GetMultitenantDatabase(id string) (*model.MultitenantDatabase, error) {
	var rawDatabase rawMultitenantDatabase
	err := sqlStore.getBuilder(sqlStore.db, &rawDatabase, multitenantDatabaseSelect.Where("ID = ?", id))
	if err == sql.ErrNoRows {
		return nil, nil
	} else if err != nil {
		return nil, errors.Wrap(err, "failed to get multitenant database by id")
	}

	return rawDatabase.toMultitenantDatabase()
}

// GetMultitenantDatabases fetches the given page of created multitenant database. The first page is 0.
func (sqlStore *SQLStore) GetMultitenantDatabases(filter *model.MultitenantDatabaseFilter) ([]*model.MultitenantDatabase, error) {
	builder := multitenantDatabaseSelect.
		OrderBy("CreateAt ASC")

	builder = applyPagingFilter(builder, filter.Paging)

	if len(filter.InstallationID) > 0 {
		builder = builder.
			Where(sq.Like{"InstallationsRaw": fmt.Sprint("%", filter.InstallationID, "%")})
	}
	if len(filter.MigratedInstallationID) > 0 {
		builder = builder.
			Where(sq.Like{"MigratedInstallationsRaw": fmt.Sprint("%", filter.MigratedInstallationID, "%")})
	}
	if len(filter.LockerID) > 0 {
		builder = builder.Where(sq.Eq{"LockAcquiredBy": filter.LockerID})
	}
	if len(filter.VpcID) > 0 {
		builder = builder.Where(sq.Eq{"VpcID": filter.VpcID})
	}
	if len(filter.DatabaseType) > 0 {
		builder = builder.Where(sq.Eq{"DatabaseType": filter.DatabaseType})
	}

	var rawDatabases rawMultitenantDatabases

	err := sqlStore.selectBuilder(sqlStore.db, &rawDatabases, builder)
	if err != nil {
		return nil, errors.Wrap(err, "failed to query multitenant databases")
	}

	databases, err := rawDatabases.toMultitenantDatabases()
	if err != nil {
		return nil, errors.Wrap(err, "failed to convert from raw multitenant database")
	}

	if filter.MaxInstallationsLimit != model.NoInstallationsLimit {
		var filteredDatabases []*model.MultitenantDatabase
		for _, database := range databases {
			totalWeight, err := sqlStore.GetInstallationsTotalDatabaseWeight(database.Installations)
			if err != nil {
				return nil, errors.Wrap(err, "failed to calculate total weight for database")
			}

			if int(math.Ceil(totalWeight)) < filter.MaxInstallationsLimit {
				filteredDatabases = append(filteredDatabases, database)
			}
		}
		databases = filteredDatabases
	}

	return databases, nil
}

// GetMultitenantDatabaseForInstallationID fetches the multitenant database associated with an installation ID.
// If more than one multitenant database per installation exists, this function returns an error.
func (sqlStore *SQLStore) GetMultitenantDatabaseForInstallationID(installationID string) (*model.MultitenantDatabase, error) {
	multitenantDatabases, err := sqlStore.GetMultitenantDatabases(&model.MultitenantDatabaseFilter{
		InstallationID:        installationID,
		MaxInstallationsLimit: model.NoInstallationsLimit,
		Paging:                model.AllPagesNotDeleted(),
	})
	if err != nil {
		return nil, errors.Wrap(err, "failed to query for multitenant databases")
	}
	if len(multitenantDatabases) != 1 {
		return nil, errors.Errorf("expected exactly one multitenant database, but found %d", len(multitenantDatabases))
	}

	return multitenantDatabases[0], nil
}

// CreateMultitenantDatabase records the supplied multitenant database to the datastore.
func (sqlStore *SQLStore) CreateMultitenantDatabase(multitenantDatabase *model.MultitenantDatabase) error {
	if multitenantDatabase.ID == "" {
		return errors.New("multitenant database ID must not be empty")
	}

	multitenantDatabase.CreateAt = GetMillis()

	installationsJSON, err := json.Marshal(multitenantDatabase.Installations)
	if err != nil {
		return errors.Wrap(err, "unable to marshal installation IDs")
	}
	migratedInstallationsJSON, err := json.Marshal(multitenantDatabase.MigratedInstallations)
	if err != nil {
		return errors.Wrap(err, "unable to marshal migrated installation IDs")
	}
	logicalDatabaseJSON, err := json.Marshal(multitenantDatabase.SharedLogicalDatabaseMappings)
	if err != nil {
		return errors.Wrap(err, "unable to marshal migrated installation IDs")
	}

	_, err = sqlStore.execBuilder(sqlStore.db, sq.
		Insert("MultitenantDatabase").
		SetMap(map[string]interface{}{
			"ID":                               multitenantDatabase.ID,
			"VpcID":                            multitenantDatabase.VpcID,
			"DatabaseType":                     multitenantDatabase.DatabaseType,
			"State":                            multitenantDatabase.State,
			"InstallationsRaw":                 installationsJSON,
			"MigratedInstallationsRaw":         migratedInstallationsJSON,
			"SharedLogicalDatabaseMappingsRaw": logicalDatabaseJSON,
			"WriterEndpoint":                   multitenantDatabase.WriterEndpoint,
			"ReaderEndpoint":                   multitenantDatabase.ReaderEndpoint,
			"LockAcquiredBy":                   nil,
			"LockAcquiredAt":                   0,
			"CreateAt":                         multitenantDatabase.CreateAt,
			"DeleteAt":                         0,
		}),
	)
	if err != nil {
		return errors.Wrap(err, "failed to store multitenant database")
	}

	return nil
}

// UpdateMultitenantDatabase updates a already existent multitenant database in the datastore.
func (sqlStore *SQLStore) UpdateMultitenantDatabase(multitenantDatabase *model.MultitenantDatabase) error {
	return sqlStore.updateMultitenantDatabase(sqlStore.db, multitenantDatabase)
}

func (sqlStore *SQLStore) updateMultitenantDatabase(db execer, multitenantDatabase *model.MultitenantDatabase) error {
	installationsJSON, err := json.Marshal(multitenantDatabase.Installations)
	if err != nil {
		return errors.Wrap(err, "unable to marshal installation IDs")
	}
	migratedInstallationsJSON, err := json.Marshal(multitenantDatabase.MigratedInstallations)
	if err != nil {
		return errors.Wrap(err, "unable to marshal migrated installation IDs")
	}
	logicalDatabaseJSON, err := json.Marshal(multitenantDatabase.SharedLogicalDatabaseMappings)
	if err != nil {
		return errors.Wrap(err, "unable to marshal migrated installation IDs")
	}

	_, err = sqlStore.execBuilder(db, sq.
		Update("MultitenantDatabase").
		SetMap(map[string]interface{}{
			"State":                              multitenantDatabase.State,
			"MaxInstallationsPerLogicalDatabase": multitenantDatabase.MaxInstallationsPerLogicalDatabase,
			"InstallationsRaw":                   []byte(installationsJSON),
			"MigratedInstallationsRaw":           []byte(migratedInstallationsJSON),
			"SharedLogicalDatabaseMappingsRaw":   logicalDatabaseJSON,
		}).
		Where(sq.Eq{"ID": multitenantDatabase.ID}),
	)
	if err != nil {
		return errors.Wrap(err, "failed to store updated multitenant database")
	}

	return nil
}

// LockMultitenantDatabase marks the database cluster as locked for exclusive use by the caller.
func (sqlStore *SQLStore) LockMultitenantDatabase(multitenantDatabaseID, lockerID string) (bool, error) {
	return sqlStore.lockRows("MultitenantDatabase", []string{multitenantDatabaseID}, lockerID)
}

// UnlockMultitenantDatabase releases a lock previously acquired against a caller.
func (sqlStore *SQLStore) UnlockMultitenantDatabase(multitenantDatabaseID, lockerID string, force bool) (bool, error) {
	return sqlStore.unlockRows("MultitenantDatabase", []string{multitenantDatabaseID}, lockerID, force)
}

// LockMultitenantDatabases marks MultitenantDatabases as locked for exclusive use by the caller.
func (sqlStore *SQLStore) LockMultitenantDatabases(ids []string, lockerID string) (bool, error) {
	return sqlStore.lockRows("MultitenantDatabase", ids, lockerID)
}

// UnlockMultitenantDatabases releases a locks previously acquired against a caller.
func (sqlStore *SQLStore) UnlockMultitenantDatabases(ids []string, lockerID string, force bool) (bool, error) {
	return sqlStore.unlockRows("MultitenantDatabase", ids, lockerID, force)
}
