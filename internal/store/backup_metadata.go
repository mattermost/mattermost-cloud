package store

import (
	"database/sql"
	"encoding/json"

	sq "github.com/Masterminds/squirrel"
	"github.com/mattermost/mattermost-cloud/model"
	"github.com/pkg/errors"
)

const (
	backupMetadataTable = "BackupMetadata"
)

var backupMetadataSelect sq.SelectBuilder

func init() {
	backupMetadataSelect = sq.
		Select(
			"ID", "InstallationID", "ClusterInstallationID", "DataResidenceRaw", "State", "RequestAt", "StartAt", "DeleteAt", "APISecurityLock", "LockAcquiredBy", "LockAcquiredAt",
		).
		From(backupMetadataTable)
}

type rawBackupMetadata struct {
	*model.BackupMetadata
	DataResidenceRaw []byte
}

type rawBackupsMetadata []*rawBackupMetadata

func (r *rawBackupMetadata) toBackupMetadata() (*model.BackupMetadata, error) {
	// We only need to set values that are converted from a raw database format.
	var err error
	if len(r.DataResidenceRaw) > 0 {
		dataResidence := model.S3DataResidence{}
		err = json.Unmarshal(r.DataResidenceRaw, &dataResidence)
		if err != nil {
			return nil, err
		}
		r.BackupMetadata.DataResidence = &dataResidence
	}

	return r.BackupMetadata, nil
}

func (r *rawBackupsMetadata) toBackupsMetadata() ([]*model.BackupMetadata, error) {
	if r == nil {
		return []*model.BackupMetadata{}, nil
	}
	backupsMeta := make([]*model.BackupMetadata, 0, len(*r))

	for _, raw := range *r {
		metadata, err := raw.toBackupMetadata()
		if err != nil {
			return nil, errors.Wrap(err, "failed to create backup metadata from raw")
		}
		backupsMeta = append(backupsMeta, metadata)
	}
	return backupsMeta, nil
}

// IsBackupRunning checks if any backup is currently running or requested for specified installation.
func (sqlStore *SQLStore) IsBackupRunning(installationID string) (bool, error) {
	var totalResult countResult
	builder := sq.
		Select("Count (*)").
		From(backupMetadataTable).
		Where("InstallationID = ?", installationID).
		Where(sq.Eq{"State": model.AllBackupMetadataStatesPendingWork}).
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

// CreateBackupMetadata record backup metadata to the database, assigning it a unique ID.
func (sqlStore *SQLStore) CreateBackupMetadata(backupMeta *model.BackupMetadata) error {
	backupMeta.ID = model.NewID()
	backupMeta.RequestAt = GetMillis()

	_, err := sqlStore.execBuilder(sqlStore.db, sq.
		Insert(backupMetadataTable).
		SetMap(map[string]interface{}{
			"ID":                    backupMeta.ID,
			"InstallationID":        backupMeta.InstallationID,
			"ClusterInstallationID": backupMeta.ClusterInstallationID,
			"DataResidenceRaw":      nil,
			"State":                 backupMeta.State,
			"RequestAt":             backupMeta.RequestAt,
			"StartAt":               0,
			"DeleteAt":              0,
			"APISecurityLock":       backupMeta.APISecurityLock,
			"LockAcquiredBy":        nil,
			"LockAcquiredAt":        0,
		}),
	)
	if err != nil {
		return errors.Wrap(err, "failed to create backup metadata")
	}

	return nil
}

// GetBackupsMetadata fetches the given page of created backups metadata. The first page is 0.
func (sqlStore *SQLStore) GetBackupsMetadata(filter *model.BackupMetadataFilter) ([]*model.BackupMetadata, error) {
	builder := backupMetadataSelect.
		OrderBy("RequestAt DESC")
	builder = sqlStore.applyBackupMetadataFilter(builder, filter)

	var rawBackupsMetadata rawBackupsMetadata
	err := sqlStore.selectBuilder(sqlStore.db, &rawBackupsMetadata, builder)
	if err != nil {
		return nil, errors.Wrap(err, "failed to query for backups metadata")
	}

	backupsMetadata, err := rawBackupsMetadata.toBackupsMetadata()
	if err != nil {
		return nil, errors.Wrap(err, "failed to backup metadata from raw")
	}

	return backupsMetadata, nil
}

// GetBackupMetadata fetches the given backup metadata by id.
func (sqlStore *SQLStore) GetBackupMetadata(id string) (*model.BackupMetadata, error) {
	builder := backupMetadataSelect.Where("ID = ?", id)

	var rawMetadata rawBackupMetadata
	err := sqlStore.getBuilder(sqlStore.db, &rawMetadata, builder)
	if err == sql.ErrNoRows {
		return nil, nil
	} else if err != nil {
		return nil, errors.Wrap(err, "failed to get backup metadata by id")
	}

	backupMetadata, err := rawMetadata.toBackupMetadata()
	if err != nil {
		return backupMetadata, err
	}

	return backupMetadata, nil
}

// GetUnlockedBackupMetadataPendingWork returns an unlocked backups metadata in a pending state.
func (sqlStore *SQLStore) GetUnlockedBackupMetadataPendingWork() ([]*model.BackupMetadata, error) {
	builder := backupMetadataSelect.
		Where(sq.Eq{
			"State": model.AllBackupMetadataStatesPendingWork,
		}).
		Where("LockAcquiredAt = 0").
		OrderBy("RequestAt ASC")

	var rawBackupsMeta rawBackupsMetadata
	err := sqlStore.selectBuilder(sqlStore.db, &rawBackupsMeta, builder)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get backup metadata pending work")
	}

	backupsMeta, err := rawBackupsMeta.toBackupsMetadata()
	if err != nil {
		return nil, err
	}

	return backupsMeta, nil
}

// UpdateBackupSchedulingData updates the given backup metadata data residency and ClusterInstallationID.
func (sqlStore *SQLStore) UpdateBackupSchedulingData(backupMeta *model.BackupMetadata) error {
	data, err := json.Marshal(backupMeta.DataResidence)
	if err != nil {
		return errors.Wrap(err, "failed to marshal data residency")
	}

	return sqlStore.updateBackupMetadataFields(
		backupMeta.ID, map[string]interface{}{
			"DataResidenceRaw":      data,
			"ClusterInstallationID": backupMeta.ClusterInstallationID,
		})
}

// UpdateBackupStartTime updates the given backup start time.
func (sqlStore *SQLStore) UpdateBackupStartTime(backupMeta *model.BackupMetadata) error {
	return sqlStore.updateBackupMetadataFields(
		backupMeta.ID, map[string]interface{}{
			"StartAt": backupMeta.StartAt,
		})
}

// UpdateBackupMetadataState updates the given backup metadata to a new state.
func (sqlStore *SQLStore) UpdateBackupMetadataState(backupMeta *model.BackupMetadata) error {
	return sqlStore.updateBackupMetadataFields(
		backupMeta.ID, map[string]interface{}{
			"State": backupMeta.State,
		})
}

// DeleteBackupMetadata marks the given backup metadata as deleted, but does not remove
// the record from the database.
func (sqlStore *SQLStore) DeleteBackupMetadata(id string) error {
	_, err := sqlStore.execBuilder(sqlStore.db, sq.
		Update(backupMetadataTable).
		Set("DeleteAt", GetMillis()).
		Where("ID = ?", id).
		Where("DeleteAt = ?", 0))
	if err != nil {
		return errors.Wrapf(err, "failed to to mark backup metadata as deleted")
	}

	return nil
}

func (sqlStore *SQLStore) updateBackupMetadataFields(id string, fields map[string]interface{}) error {
	_, err := sqlStore.execBuilder(sqlStore.db, sq.
		Update(backupMetadataTable).
		SetMap(fields).
		Where("ID = ?", id))
	if err != nil {
		return errors.Wrapf(err, "failed to update backup metadata fields: %s", getMapKeys(fields))
	}

	return nil
}

// LockBackupMetadata marks the backup metadata as locked for exclusive use by the caller.
func (sqlStore *SQLStore) LockBackupMetadata(backupMetadataID, lockerID string) (bool, error) {
	return sqlStore.lockRows(backupMetadataTable, []string{backupMetadataID}, lockerID)
}

// UnlockBackupMetadata releases a lock previously acquired against a caller.
func (sqlStore *SQLStore) UnlockBackupMetadata(backupMetadataID, lockerID string, force bool) (bool, error) {
	return sqlStore.unlockRows(backupMetadataTable, []string{backupMetadataID}, lockerID, force)
}

func (sqlStore *SQLStore) applyBackupMetadataFilter(builder sq.SelectBuilder, filter *model.BackupMetadataFilter) sq.SelectBuilder {
	if filter.PerPage != model.AllPerPage {
		builder = builder.
			Limit(uint64(filter.PerPage)).
			Offset(uint64(filter.Page * filter.PerPage))
	}

	if filter.InstallationID != "" {
		builder = builder.Where("InstallationID = ?", filter.InstallationID)
	}
	if filter.ClusterInstallationID != "" {
		builder = builder.Where("ClusterInstallationID = ?", filter.ClusterInstallationID)
	}
	if filter.State != "" {
		builder = builder.Where("State = ?", filter.State)
	}
	if !filter.IncludeDeleted {
		builder = builder.Where("DeleteAt = 0")
	}

	return builder
}

// LockBackupAPI locks updates to the backup from the API.
func (sqlStore *SQLStore) LockBackupAPI(backupMetadataID string) error {
	return sqlStore.updateBackupMetadataFields(
		backupMetadataID, map[string]interface{}{
			"APISecurityLock": true,
		})
}

// UnlockBackupAPI unlocks updates to the backup from the API.
func (sqlStore *SQLStore) UnlockBackupAPI(backupMetadataID string) error {
	return sqlStore.updateBackupMetadataFields(
		backupMetadataID, map[string]interface{}{
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
