// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.
//

package store

import (
	"database/sql"
	"encoding/json"

	sq "github.com/Masterminds/squirrel"
	"github.com/mattermost/mattermost-cloud/model"
	"github.com/pkg/errors"
)

const (
	backupTable = "InstallationBackup"
)

var installationBackupSelect sq.SelectBuilder

func init() {
	installationBackupSelect = sq.
		Select("ID",
			"InstallationID",
			"ClusterInstallationID",
			"DataResidenceRaw",
			"State",
			"RequestAt",
			"StartAt",
			"DeleteAt",
			"APISecurityLock",
			"LockAcquiredBy",
			"LockAcquiredAt",
		).
		From(backupTable)
}

type rawInstallationBackup struct {
	*model.InstallationBackup
	DataResidenceRaw []byte
}

type rawInstallationBackups []*rawInstallationBackup

func (r *rawInstallationBackup) toInstallationBackup() (*model.InstallationBackup, error) {
	// We only need to set values that are converted from a raw database format.
	var err error
	if len(r.DataResidenceRaw) > 0 {
		dataResidence := model.S3DataResidence{}
		err = json.Unmarshal(r.DataResidenceRaw, &dataResidence)
		if err != nil {
			return nil, err
		}
		r.InstallationBackup.DataResidence = &dataResidence
	}

	return r.InstallationBackup, nil
}

func (r *rawInstallationBackups) toInstallationBackups() ([]*model.InstallationBackup, error) {
	if r == nil {
		return []*model.InstallationBackup{}, nil
	}
	backups := make([]*model.InstallationBackup, 0, len(*r))

	for _, raw := range *r {
		backup, err := raw.toInstallationBackup()
		if err != nil {
			return nil, errors.Wrap(err, "failed to create backup from raw")
		}
		backups = append(backups, backup)
	}
	return backups, nil
}

// IsInstallationBackupRunning checks if any backup is currently running or requested for specified installation.
func (sqlStore *SQLStore) IsInstallationBackupRunning(installationID string) (bool, error) {
	var totalResult countResult
	builder := sq.
		Select("Count (*)").
		From(backupTable).
		Where("InstallationID = ?", installationID).
		Where(sq.Eq{"State": model.AllInstallationBackupsStatesRunning}).
		Where("DeleteAt = 0")
	err := sqlStore.selectBuilder(sqlStore.db, &totalResult, builder)
	if err != nil {
		return false, errors.Wrap(err, "failed to count ongoing backups")
	}

	ongoingBackups, err := totalResult.value()
	if err != nil {
		return false, errors.Wrap(err, "failed to value of ongoing backups")
	}

	return ongoingBackups > 0, nil
}

// CreateInstallationBackup records installation backup to the database, assigning it a unique ID.
func (sqlStore *SQLStore) CreateInstallationBackup(backup *model.InstallationBackup) error {
	backup.ID = model.NewID()
	backup.RequestAt = GetMillis()

	_, err := sqlStore.execBuilder(sqlStore.db, sq.
		Insert(backupTable).
		SetMap(map[string]interface{}{
			"ID":                    backup.ID,
			"InstallationID":        backup.InstallationID,
			"ClusterInstallationID": backup.ClusterInstallationID,
			"DataResidenceRaw":      nil,
			"State":                 backup.State,
			"RequestAt":             backup.RequestAt,
			"StartAt":               0,
			"DeleteAt":              0,
			"APISecurityLock":       backup.APISecurityLock,
			"LockAcquiredBy":        nil,
			"LockAcquiredAt":        0,
		}),
	)
	if err != nil {
		return errors.Wrap(err, "failed to create backup")
	}

	return nil
}

// GetInstallationBackups fetches the given page of created installation backups. The first page is 0.
func (sqlStore *SQLStore) GetInstallationBackups(filter *model.InstallationBackupFilter) ([]*model.InstallationBackup, error) {
	builder := installationBackupSelect.
		OrderBy("RequestAt DESC")
	builder = sqlStore.applyInstallationBackupFilter(builder, filter)

	var rawInstallationBackupsData rawInstallationBackups
	err := sqlStore.selectBuilder(sqlStore.db, &rawInstallationBackupsData, builder)
	if err != nil {
		return nil, errors.Wrap(err, "failed to query for backups metadata")
	}

	backups, err := rawInstallationBackupsData.toInstallationBackups()
	if err != nil {
		return nil, errors.Wrap(err, "failed to backup from raw")
	}

	return backups, nil
}

// GetInstallationBackup fetches the given installation backup by id.
func (sqlStore *SQLStore) GetInstallationBackup(id string) (*model.InstallationBackup, error) {
	builder := installationBackupSelect.Where("ID = ?", id)

	var rawBackupData rawInstallationBackup
	err := sqlStore.getBuilder(sqlStore.db, &rawBackupData, builder)
	if err == sql.ErrNoRows {
		return nil, nil
	} else if err != nil {
		return nil, errors.Wrap(err, "failed to get installation backup by id")
	}

	backup, err := rawBackupData.toInstallationBackup()
	if err != nil {
		return nil, errors.Wrap(err, "failed to convert installation backup from raw")
	}

	return backup, nil
}

// GetUnlockedInstallationBackupPendingWork returns an unlocked installation backups in a pending state.
func (sqlStore *SQLStore) GetUnlockedInstallationBackupPendingWork() ([]*model.InstallationBackup, error) {
	builder := installationBackupSelect.
		Where(sq.Eq{
			"State": model.AllInstallationBackupStatesPendingWork,
		}).
		Where("LockAcquiredAt = 0").
		OrderBy("RequestAt ASC")

	var rawBackupsData rawInstallationBackups
	err := sqlStore.selectBuilder(sqlStore.db, &rawBackupsData, builder)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get installation backup pending work")
	}

	backups, err := rawBackupsData.toInstallationBackups()
	if err != nil {
		return nil, err
	}

	return backups, nil
}

// UpdateInstallationBackupSchedulingData updates the given backup data residency and ClusterInstallationID.
func (sqlStore *SQLStore) UpdateInstallationBackupSchedulingData(backup *model.InstallationBackup) error {
	data, err := json.Marshal(backup.DataResidence)
	if err != nil {
		return errors.Wrap(err, "failed to marshal data residency")
	}

	return sqlStore.updateBackupFields(
		backup.ID, map[string]interface{}{
			"DataResidenceRaw":      data,
			"ClusterInstallationID": backup.ClusterInstallationID,
		})
}

// UpdateInstallationBackupStartTime updates the given backup start time.
func (sqlStore *SQLStore) UpdateInstallationBackupStartTime(backup *model.InstallationBackup) error {
	return sqlStore.updateBackupFields(
		backup.ID, map[string]interface{}{
			"StartAt": backup.StartAt,
		})
}

// UpdateInstallationBackupState updates the given backup to a new state.
func (sqlStore *SQLStore) UpdateInstallationBackupState(backup *model.InstallationBackup) error {
	return sqlStore.updateBackupFields(
		backup.ID, map[string]interface{}{
			"State": backup.State,
		})
}

// DeleteInstallationBackup marks the given backup as deleted, but does not remove
// the record from the database.
func (sqlStore *SQLStore) DeleteInstallationBackup(id string) error {
	_, err := sqlStore.execBuilder(sqlStore.db, sq.
		Update(backupTable).
		Set("DeleteAt", GetMillis()).
		Where("ID = ?", id).
		Where("DeleteAt = ?", 0))
	if err != nil {
		return errors.Wrapf(err, "failed to to mark backup as deleted")
	}

	return nil
}

func (sqlStore *SQLStore) updateBackupFields(id string, fields map[string]interface{}) error {
	_, err := sqlStore.execBuilder(sqlStore.db, sq.
		Update(backupTable).
		SetMap(fields).
		Where("ID = ?", id))
	if err != nil {
		return errors.Wrapf(err, "failed to update installation backup fields: %s", getMapKeys(fields))
	}

	return nil
}

// LockInstallationBackup marks the backup as locked for exclusive use by the caller.
func (sqlStore *SQLStore) LockInstallationBackup(backupID, lockerID string) (bool, error) {
	return sqlStore.lockRows(backupTable, []string{backupID}, lockerID)
}

// LockInstallationBackups marks backups as locked for exclusive use by the caller.
func (sqlStore *SQLStore) LockInstallationBackups(backupIDs []string, lockerID string) (bool, error) {
	return sqlStore.lockRows(backupTable, backupIDs, lockerID)
}

// UnlockInstallationBackup releases a lock previously acquired against a caller.
func (sqlStore *SQLStore) UnlockInstallationBackup(backupID, lockerID string, force bool) (bool, error) {
	return sqlStore.unlockRows(backupTable, []string{backupID}, lockerID, force)
}

// UnlockInstallationBackups releases a locks previously acquired against a caller.
func (sqlStore *SQLStore) UnlockInstallationBackups(backupIDs []string, lockerID string, force bool) (bool, error) {
	return sqlStore.unlockRows(backupTable, backupIDs, lockerID, force)
}

func (sqlStore *SQLStore) applyInstallationBackupFilter(builder sq.SelectBuilder, filter *model.InstallationBackupFilter) sq.SelectBuilder {
	if filter.PerPage != model.AllPerPage {
		builder = builder.
			Limit(uint64(filter.PerPage)).
			Offset(uint64(filter.Page * filter.PerPage))
	}

	if len(filter.IDs) > 0 {
		builder = builder.Where(sq.Eq{"ID": filter.IDs})
	}
	if filter.InstallationID != "" {
		builder = builder.Where("InstallationID = ?", filter.InstallationID)
	}
	if filter.ClusterInstallationID != "" {
		builder = builder.Where("ClusterInstallationID = ?", filter.ClusterInstallationID)
	}
	if len(filter.States) > 0 {
		builder = builder.Where(sq.Eq{
			"State": filter.States,
		})
	}
	if !filter.IncludeDeleted {
		builder = builder.Where("DeleteAt = 0")
	}

	return builder
}

// LockInstallationBackupAPI locks updates to the backup from the API.
func (sqlStore *SQLStore) LockInstallationBackupAPI(backupID string) error {
	return sqlStore.updateBackupFields(
		backupID, map[string]interface{}{
			"APISecurityLock": true,
		})
}

// UnlockInstallationBackupAPI unlocks updates to the backup from the API.
func (sqlStore *SQLStore) UnlockInstallationBackupAPI(backupID string) error {
	return sqlStore.updateBackupFields(
		backupID, map[string]interface{}{
			"APISecurityLock": false,
		})
}

func getMapKeys(m map[string]interface{}) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	return keys
}
