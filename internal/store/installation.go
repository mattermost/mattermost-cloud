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

var installationSelect sq.SelectBuilder

func init() {
	installationSelect = sq.
		Select(
			"ID", "OwnerID", "Version", "Image", "DNS", "Database", "Filestore", "Size",
			"Affinity", "GroupID", "GroupSequence", "State", "License",
			"MattermostEnvRaw", "CreateAt", "DeleteAt", "APISecurityLock",
			"LockAcquiredBy", "LockAcquiredAt",
		).
		From("Installation")
}

type rawInstallation struct {
	*model.Installation
	MattermostEnvRaw []byte
}

type rawInstallations []*rawInstallation

func (r *rawInstallation) toInstallation() (*model.Installation, error) {
	// We only need to set values that are converted from a raw database format.
	var err error
	mattermostEnv := &model.EnvVarMap{}
	if r.MattermostEnvRaw != nil {
		mattermostEnv, err = model.EnvVarFromJSON(r.MattermostEnvRaw)
		if err != nil {
			return nil, err
		}
	}

	r.Installation.MattermostEnv = *mattermostEnv
	return r.Installation, nil
}

func (rs *rawInstallations) toInstallations() ([]*model.Installation, error) {
	var installations []*model.Installation
	for _, rawInstallation := range *rs {
		installation, err := rawInstallation.toInstallation()
		if err != nil {
			return nil, err
		}
		installations = append(installations, installation)
	}

	return installations, nil
}

// GetInstallation fetches the given installation by id.
func (sqlStore *SQLStore) GetInstallation(id string, includeGroupConfig, includeGroupConfigOverrides bool) (*model.Installation, error) {
	var rawInstallation rawInstallation
	err := sqlStore.getBuilder(sqlStore.db, &rawInstallation,
		installationSelect.Where("ID = ?", id),
	)
	if err == sql.ErrNoRows {
		return nil, nil
	} else if err != nil {
		return nil, errors.Wrap(err, "failed to get installation by id")
	}

	installation, err := rawInstallation.toInstallation()
	if err != nil {
		return installation, err
	}
	if !installation.IsInGroup() || !includeGroupConfig {
		return installation, nil
	}

	// Installation is in a group and the request is for the merged config,
	// so get group config and perform a merge.
	group, err := sqlStore.GetGroup(*installation.GroupID)
	if err != nil {
		return installation, err
	}
	installation.MergeWithGroup(group, includeGroupConfigOverrides)

	return installation, nil
}

// GetInstallations fetches the given page of created installations. The first page is 0.
func (sqlStore *SQLStore) GetInstallations(filter *model.InstallationFilter, includeGroupConfig, includeGroupConfigOverrides bool) ([]*model.Installation, error) {
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
	if filter.GroupID != "" {
		builder = builder.Where("GroupID = ?", filter.GroupID)
	}
	if !filter.IncludeDeleted {
		builder = builder.Where("DeleteAt = 0")
	}
	if filter.DNS != "" {
		builder = builder.Where("DNS = ?", filter.DNS)
	}

	var rawInstallations rawInstallations
	err := sqlStore.selectBuilder(sqlStore.db, &rawInstallations, builder)
	if err != nil {
		return nil, errors.Wrap(err, "failed to query for installations")
	}

	installations, err := rawInstallations.toInstallations()
	if err != nil {
		return nil, err
	}

	for _, installation := range installations {
		if !installation.IsInGroup() || !includeGroupConfig {
			continue
		}

		// Installation is in a group and the request is for the merged config,
		// so get group config and perform a merge.
		group, err := sqlStore.GetGroup(*installation.GroupID)
		if err != nil {
			return nil, err
		}
		installation.MergeWithGroup(group, includeGroupConfigOverrides)
	}

	return installations, nil
}

// GetInstallationsCount returns the number of installations filtered by the deletedat
// field
func (sqlStore *SQLStore) GetInstallationsCount(includeDeleted bool) (int, error) {
	builder := sq.Select("COUNT(*) as InstallationsNumber").From("Installation")
	if !includeDeleted {
		builder = builder.Where("DeleteAt = 0")
	}
	var numberOfInstallations int
	query, _, err := builder.ToSql()
	if err != nil {
		return 0, errors.Wrap(err, "failed to parse query for installations count")
	}
	err = sqlStore.get(sqlStore.db, &numberOfInstallations, query)
	if err != nil {
		return 0, errors.Wrap(err, "failed to query for installations count")
	}
	return numberOfInstallations, nil
}

// GetUnlockedInstallationsPendingWork returns an unlocked installation in a pending state.
func (sqlStore *SQLStore) GetUnlockedInstallationsPendingWork() ([]*model.Installation, error) {
	builder := installationSelect.
		Where(sq.Eq{
			"State": model.AllInstallationStatesPendingWork,
		}).
		Where("LockAcquiredAt = 0").
		OrderBy("CreateAt ASC")

	var rawInstallations rawInstallations
	err := sqlStore.selectBuilder(sqlStore.db, &rawInstallations, builder)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get installations pending work")
	}

	installations, err := rawInstallations.toInstallations()
	if err != nil {
		return nil, err
	}

	for _, installation := range installations {
		if !installation.IsInGroup() {
			continue
		}

		group, err := sqlStore.GetGroup(*installation.GroupID)
		if err != nil {
			return nil, err
		}
		installation.MergeWithGroup(group, false)
	}

	return installations, nil
}

// CreateInstallation records the given installation to the database, assigning it a unique ID.
func (sqlStore *SQLStore) CreateInstallation(installation *model.Installation) error {
	installation.ID = model.NewID()
	installation.CreateAt = GetMillis()

	envJSON, err := json.Marshal(installation.MattermostEnv)
	if err != nil {
		return errors.Wrap(err, "unable to marshal MattermostEnv")
	}

	_, err = sqlStore.execBuilder(sqlStore.db, sq.
		Insert("Installation").
		SetMap(map[string]interface{}{
			"ID":               installation.ID,
			"OwnerID":          installation.OwnerID,
			"GroupID":          installation.GroupID,
			"GroupSequence":    nil,
			"Version":          installation.Version,
			"Image":            installation.Image,
			"DNS":              installation.DNS,
			"Database":         installation.Database,
			"Filestore":        installation.Filestore,
			"Size":             installation.Size,
			"Affinity":         installation.Affinity,
			"State":            installation.State,
			"License":          installation.License,
			"MattermostEnvRaw": []byte(envJSON),
			"CreateAt":         installation.CreateAt,
			"DeleteAt":         0,
			"APISecurityLock":  installation.APISecurityLock,
			"LockAcquiredBy":   nil,
			"LockAcquiredAt":   0,
		}),
	)
	if err != nil {
		return errors.Wrap(err, "failed to create installation")
	}

	return nil
}

// UpdateInstallation updates the given installation in the database.
func (sqlStore *SQLStore) UpdateInstallation(installation *model.Installation) error {
	if installation.ConfigMergedWithGroup() {
		return errors.New("unable to save installations that have merged group config")
	}
	envJSON, err := json.Marshal(installation.MattermostEnv)
	if err != nil {
		return errors.Wrap(err, "unable to marshal MattermostEnv")
	}

	_, err = sqlStore.execBuilder(sqlStore.db, sq.
		Update("Installation").
		SetMap(map[string]interface{}{
			"OwnerID":          installation.OwnerID,
			"GroupID":          installation.GroupID,
			"GroupSequence":    installation.GroupSequence,
			"Version":          installation.Version,
			"Image":            installation.Image,
			"DNS":              installation.DNS,
			"Database":         installation.Database,
			"Filestore":        installation.Filestore,
			"Size":             installation.Size,
			"Affinity":         installation.Affinity,
			"License":          installation.License,
			"MattermostEnvRaw": []byte(envJSON),
			"State":            installation.State,
		}).
		Where("ID = ?", installation.ID),
	)
	if err != nil {
		return errors.Wrap(err, "failed to update installation")
	}

	return nil
}

// UpdateInstallationGroupSequence updates the given installation GroupSequence
// to the value stored in the merged group config. The provided installation must
// have been merged with group config before passing it in.
func (sqlStore *SQLStore) UpdateInstallationGroupSequence(installation *model.Installation) error {
	if !installation.ConfigMergedWithGroup() {
		return errors.New("installation is not merged with a group")
	}

	_, err := sqlStore.execBuilder(sqlStore.db, sq.
		Update("Installation").
		SetMap(map[string]interface{}{
			"GroupSequence": installation.GroupSequence,
		}).
		Where("ID = ?", installation.ID),
	)
	if err != nil {
		return errors.Wrap(err, "failed to update installation")
	}

	return nil
}

// UpdateInstallationState updates the given installation to a new state.
func (sqlStore *SQLStore) UpdateInstallationState(installation *model.Installation) error {
	_, err := sqlStore.execBuilder(sqlStore.db, sq.
		Update("Installation").
		SetMap(map[string]interface{}{
			"State": installation.State,
		}).
		Where("ID = ?", installation.ID),
	)
	if err != nil {
		return errors.Wrap(err, "failed to update installation state")
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

// LockInstallation marks the installation as locked for exclusive use by the caller.
func (sqlStore *SQLStore) LockInstallation(installationID, lockerID string) (bool, error) {
	return sqlStore.lockRows("Installation", []string{installationID}, lockerID)
}

// UnlockInstallation releases a lock previously acquired against a caller.
func (sqlStore *SQLStore) UnlockInstallation(installationID, lockerID string, force bool) (bool, error) {
	return sqlStore.unlockRows("Installation", []string{installationID}, lockerID, force)
}

// LockInstallationAPI locks updates to the installation from the API.
func (sqlStore *SQLStore) LockInstallationAPI(installationID string) error {
	return sqlStore.setInstallationAPILock(installationID, true)
}

// UnlockInstallationAPI unlocks updates to the installation from the API.
func (sqlStore *SQLStore) UnlockInstallationAPI(installationID string) error {
	return sqlStore.setInstallationAPILock(installationID, false)
}

func (sqlStore *SQLStore) setInstallationAPILock(installationID string, lock bool) error {
	_, err := sqlStore.execBuilder(sqlStore.db, sq.
		Update("Installation").
		Set("APISecurityLock", lock).
		Where("ID = ?", installationID),
	)
	if err != nil {
		return errors.Wrap(err, "failed to store installation API lock")
	}

	return nil
}
