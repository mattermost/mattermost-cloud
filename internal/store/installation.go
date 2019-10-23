package store

import (
	"database/sql"

	sq "github.com/Masterminds/squirrel"
	"github.com/mattermost/mattermost-cloud/model"
	"github.com/pkg/errors"
)

var installationSelect sq.SelectBuilder

func init() {
	installationSelect = sq.
		Select(
			"ID", "OwnerID", "Version", "DNS", "Database", "Filestore", "Size",
			"Affinity", "GroupID", "State", "License", "CreateAt", "DeleteAt",
			"LockAcquiredBy", "LockAcquiredAt",
		).
		From("Installation")
}

// GetInstallation fetches the given installation by id.
func (sqlStore *SQLStore) GetInstallation(id string) (*model.Installation, error) {
	var installation model.Installation
	err := sqlStore.getBuilder(sqlStore.db, &installation,
		installationSelect.Where("ID = ?", id),
	)
	if err == sql.ErrNoRows {
		return nil, nil
	} else if err != nil {
		return nil, errors.Wrap(err, "failed to get installation by id")
	}

	return &installation, nil
}

// GetUnlockedInstallationsPendingWork returns an unlocked installation in a pending state.
func (sqlStore *SQLStore) GetUnlockedInstallationsPendingWork() ([]*model.Installation, error) {
	builder := installationSelect.
		Where(sq.Eq{
			"State": []string{
				model.InstallationStateCreationRequested,
				model.InstallationStateCreationPreProvisioning,
				model.InstallationStateCreationInProgress,
				model.InstallationStateCreationNoCompatibleClusters,
				model.InstallationStateCreationDNS,
				model.InstallationStateUpgradeRequested,
				model.InstallationStateUpgradeInProgress,
				model.InstallationStateDeletionRequested,
				model.InstallationStateDeletionInProgress,
				model.InstallationStateDeletionFinalCleanup,
			},
		}).
		Where("LockAcquiredAt = 0").
		OrderBy("CreateAt ASC")

	var installations []*model.Installation
	err := sqlStore.selectBuilder(sqlStore.db, &installations, builder)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get installations pending work")
	}

	return installations, nil
}

// LockInstallation marks the installation as locked for exclusive use by the caller.
func (sqlStore *SQLStore) LockInstallation(installationID, lockerID string) (bool, error) {
	return sqlStore.lockRows("Installation", []string{installationID}, lockerID)
}

// UnlockInstallation releases a lock previously acquired against a caller.
func (sqlStore *SQLStore) UnlockInstallation(installationID, lockerID string, force bool) (bool, error) {
	return sqlStore.unlockRows("Installation", []string{installationID}, lockerID, force)
}

// GetInstallations fetches the given page of created installations. The first page is 0.
func (sqlStore *SQLStore) GetInstallations(filter *model.InstallationFilter) ([]*model.Installation, error) {
	builder := installationSelect.
		OrderBy("CreateAt ASC")

	if filter.PerPage != model.AllPerPage {
		builder = builder.
			Limit(uint64(filter.PerPage)).
			Offset(uint64(filter.Page * filter.PerPage))
	}

	if filter.OwnerID != "" {
		builder = builder.Where("OwnerID = ?", filter.OwnerID)
	}
	if !filter.IncludeDeleted {
		builder = builder.Where("DeleteAt = 0")
	}

	var installations []*model.Installation
	err := sqlStore.selectBuilder(sqlStore.db, &installations, builder)
	if err != nil {
		return nil, errors.Wrap(err, "failed to query for installations")
	}

	return installations, nil
}

// CreateInstallation records the given installation to the database, assigning it a unique ID.
func (sqlStore *SQLStore) CreateInstallation(installation *model.Installation) error {
	installation.ID = model.NewID()
	installation.CreateAt = GetMillis()

	_, err := sqlStore.execBuilder(sqlStore.db, sq.
		Insert("Installation").
		SetMap(map[string]interface{}{
			"ID":             installation.ID,
			"OwnerID":        installation.OwnerID,
			"Version":        installation.Version,
			"DNS":            installation.DNS,
			"Database":       installation.Database,
			"Filestore":      installation.Filestore,
			"Size":           installation.Size,
			"Affinity":       installation.Affinity,
			"GroupID":        installation.GroupID,
			"State":          installation.State,
			"CreateAt":       installation.CreateAt,
			"License":        installation.License,
			"DeleteAt":       0,
			"LockAcquiredBy": nil,
			"LockAcquiredAt": 0,
		}),
	)
	if err != nil {
		return errors.Wrap(err, "failed to create installation")
	}

	return nil
}

// UpdateInstallation updates the given installation in the database.
func (sqlStore *SQLStore) UpdateInstallation(installation *model.Installation) error {
	_, err := sqlStore.execBuilder(sqlStore.db, sq.
		Update("Installation").
		SetMap(map[string]interface{}{
			"OwnerID":   installation.OwnerID,
			"Version":   installation.Version,
			"DNS":       installation.DNS,
			"Database":  installation.Database,
			"Filestore": installation.Filestore,
			"Size":      installation.Size,
			"Affinity":  installation.Affinity,
			"GroupID":   installation.GroupID,
			"License":   installation.License,
			"State":     installation.State,
		}).
		Where("ID = ?", installation.ID),
	)
	if err != nil {
		return errors.Wrap(err, "failed to update installation")
	}

	return nil
}

// DeleteInstallation marks the given installation as deleted, but does not remove the record from the
// database.
func (sqlStore *SQLStore) DeleteInstallation(id string) error {
	_, err := sqlStore.execBuilder(sqlStore.db, sq.
		Update("Installation").
		Set("DeleteAt", GetMillis()).
		Where("ID = ?", id).
		Where("DeleteAt = 0"),
	)
	if err != nil {
		return errors.Wrap(err, "failed to mark installation as deleted")
	}

	return nil
}
