// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.
//

package store

import (
	"github.com/mattermost/mattermost-cloud/model"

	sq "github.com/Masterminds/squirrel"
	"github.com/pkg/errors"
)

// GetOrCreateProxyDatabaseResourcesForInstallation returns DatabaseResourceGrouping
// for a given installation ID or creates the necessary resources if they
// don't exist.
func (sqlStore *SQLStore) GetOrCreateProxyDatabaseResourcesForInstallation(installationID, multitenantDatabaseID string) (*model.DatabaseResourceGrouping, error) {
	databaseResources, err := sqlStore.GetProxyDatabaseResourcesForInstallation(installationID)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get database schema for installation")
	}
	if databaseResources != nil {
		return databaseResources, nil
	}

	// Begin assigning installation to a logical database.

	multitenantDatabase, err := sqlStore.GetMultitenantDatabase(multitenantDatabaseID)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get multitenant database")
	}
	if multitenantDatabase == nil {
		return nil, errors.Errorf("multitenant database %s doesn't exist", multitenantDatabaseID)
	}
	if multitenantDatabase.DeleteAt > 0 {
		return nil, errors.Errorf("multitenant database %s has been deleted", multitenantDatabaseID)
	}

	logicalDatabases, err := sqlStore.GetLogicalDatabases(&model.LogicalDatabaseFilter{
		MultitenantDatabaseID: multitenantDatabaseID,
		Paging:                model.AllPagesNotDeleted(),
	})
	if err != nil {
		return nil, errors.Wrap(err, "failed to get logical databases for multitenant database")
	}
	var logicalDatabaseIDs []string
	for _, logicalDatabase := range logicalDatabases {
		logicalDatabaseIDs = append(logicalDatabaseIDs, logicalDatabase.ID)
	}

	countResult, err := sqlStore.getSchemaCountsPerLogicalDatabase(logicalDatabaseIDs)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get database schema counts")
	}

	// Maximize logical database density while not exceeding the maximum value.
	var selectedLogicalDatabase *model.LogicalDatabase
	for _, logicalDatabase := range logicalDatabases {
		if countResult[logicalDatabase.ID] >= multitenantDatabase.MaxInstallationsPerLogicalDatabase {
			continue
		}
		if selectedLogicalDatabase == nil {
			selectedLogicalDatabase = logicalDatabase
			continue
		}
		if countResult[logicalDatabase.ID] > countResult[selectedLogicalDatabase.ID] {
			selectedLogicalDatabase = logicalDatabase
		}
	}
	if selectedLogicalDatabase == nil {
		// None of the existing logical databases are valid, so create a new one.
		newLogicalDatabase := &model.LogicalDatabase{
			MultitenantDatabaseID: multitenantDatabase.ID,
		}
		err = sqlStore.CreateLogicalDatabase(newLogicalDatabase)
		if err != nil {
			return nil, errors.Wrap(err, "failed to create new logical database")
		}
		selectedLogicalDatabase = newLogicalDatabase
	}

	databaseSchema := &model.DatabaseSchema{
		LogicalDatabaseID: selectedLogicalDatabase.ID,
		InstallationID:    installationID,
	}
	err = sqlStore.commitInstallationToProxyDatabaseResources(multitenantDatabase, databaseSchema)
	if err != nil {
		return nil, errors.Wrap(err, "failed to create new database schema")
	}

	return &model.DatabaseResourceGrouping{
		MultitenantDatabase: multitenantDatabase,
		LogicalDatabase:     selectedLogicalDatabase,
		DatabaseSchema:      databaseSchema,
	}, nil
}

// GetProxyDatabaseResourcesForInstallation returns the DatabaseResourceGrouping
// for a given installation ID.
func (sqlStore *SQLStore) GetProxyDatabaseResourcesForInstallation(installationID string) (*model.DatabaseResourceGrouping, error) {
	schema, err := sqlStore.GetDatabaseSchemaForInstallationID(installationID)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get database schema for installation")
	}
	if schema == nil {
		return nil, nil
	}
	logicalDatabase, err := sqlStore.GetLogicalDatabase(schema.LogicalDatabaseID)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get logical database for installation")
	}
	multitenantDatabase, err := sqlStore.GetMultitenantDatabase(logicalDatabase.MultitenantDatabaseID)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get database schema for installation")
	}

	return &model.DatabaseResourceGrouping{
		MultitenantDatabase: multitenantDatabase,
		LogicalDatabase:     logicalDatabase,
		DatabaseSchema:      schema,
	}, nil
}

// commitInstallationToDatabaseResources makes the final database changes for
// installation assignment synchronously.
func (sqlStore *SQLStore) commitInstallationToProxyDatabaseResources(multitenantDatabase *model.MultitenantDatabase, databaseSchema *model.DatabaseSchema) error {
	tx, err := sqlStore.beginTransaction(sqlStore.db)
	if err != nil {
		return errors.Wrap(err, "failed to start transaction")
	}
	defer tx.RollbackUnlessCommitted()

	err = sqlStore.createDatabaseSchema(tx, databaseSchema)
	if err != nil {
		return errors.Wrap(err, "failed to create database schema")
	}

	if !multitenantDatabase.Installations.Contains(databaseSchema.InstallationID) {
		multitenantDatabase.Installations.Add(databaseSchema.InstallationID)
		err = sqlStore.updateMultitenantDatabase(tx, multitenantDatabase)
		if err != nil {
			return errors.Wrap(err, "failed to update multitenant database")
		}
	}

	err = tx.Commit()
	if err != nil {
		return errors.Wrap(err, "failed to commit transaction")
	}

	return nil
}

// DeleteInstallationProxyDatabaseResources makes the final database resource
// cleanup for installatoin deletion.
func (sqlStore *SQLStore) DeleteInstallationProxyDatabaseResources(multitenantDatabase *model.MultitenantDatabase, databaseSchema *model.DatabaseSchema) error {
	tx, err := sqlStore.beginTransaction(sqlStore.db)
	if err != nil {
		return errors.Wrap(err, "failed to start transaction")
	}
	defer tx.RollbackUnlessCommitted()

	err = sqlStore.deleteDatabaseSchema(tx, databaseSchema.ID)
	if err != nil {
		return errors.Wrap(err, "failed to delete database schema")
	}

	multitenantDatabase.Installations.Remove(databaseSchema.InstallationID)
	err = sqlStore.updateMultitenantDatabase(tx, multitenantDatabase)
	if err != nil {
		return errors.Wrap(err, "failed to update multitenant database")
	}

	err = tx.Commit()
	if err != nil {
		return errors.Wrap(err, "failed to commit transaction")
	}

	return nil
}

// getSchemaCountsPerLogicalDatabase returns the number of database schemas for
// each logical database ID.
func (sqlStore *SQLStore) getSchemaCountsPerLogicalDatabase(logicalDatabaseIDs []string) (map[string]int64, error) {
	var output []struct {
		LogicalDatabaseID string
		Count             int64
	}

	installationBuilder := sq.
		Select("Count (*) as Count, LogicalDatabaseID").
		From("DatabaseSchema").
		Where("DeleteAt = 0").
		Where(sq.Eq{"LogicalDatabaseID": logicalDatabaseIDs}).
		GroupBy("LogicalDatabaseID")
	err := sqlStore.selectBuilder(sqlStore.db, &output, installationBuilder)
	if err != nil {
		return nil, errors.Wrap(err, "failed to query for installations by state")
	}

	result := make(map[string]int64, len(logicalDatabaseIDs))
	for _, logicalDatabaseID := range logicalDatabaseIDs {
		// Initialize the map with zeroes in case some logical databases are empty.
		result[logicalDatabaseID] = 0
	}
	for _, entry := range output {
		result[entry.LogicalDatabaseID] = entry.Count
	}

	return result, nil
}
