// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.
//

package store

import (
	"database/sql"
	"encoding/json"
	"fmt"

	sq "github.com/Masterminds/squirrel"
	"github.com/mattermost/mattermost-cloud/model"
	"github.com/pkg/errors"
)

var installationSelect sq.SelectBuilder

func init() {
	installationSelect = sq.
		Select(
			"Installation.ID", "OwnerID", "Version", "Image", "DNS", "Database", "Filestore", "Size",
			"Affinity", "GroupID", "GroupSequence", "State", "License",
			"MattermostEnvRaw", "SingleTenantDatabaseConfigRaw", "CreateAt", "DeleteAt",
			"APISecurityLock", "LockAcquiredBy", "LockAcquiredAt", "CRVersion",
		).
		From("Installation")
}

type rawInstallation struct {
	*model.Installation
	MattermostEnvRaw              []byte
	SingleTenantDatabaseConfigRaw []byte
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

	if r.SingleTenantDatabaseConfigRaw != nil {
		singleTenantDBConfig := &model.SingleTenantDatabaseConfig{}
		err = json.Unmarshal(r.SingleTenantDatabaseConfigRaw, singleTenantDBConfig)
		if err != nil {
			return nil, err
		}
		r.Installation.SingleTenantDatabaseConfig = singleTenantDBConfig
	}

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
	builder = sqlStore.applyInstallationFilter(builder, filter)

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

func (sqlStore *SQLStore) applyInstallationFilter(builder sq.SelectBuilder, filter *model.InstallationFilter) sq.SelectBuilder {
	builder = applyPagingFilter(builder, filter.Paging)

	if len(filter.InstallationIDs) != 0 {
		builder = builder.Where(sq.Eq{"Installation.ID": filter.InstallationIDs})
	}
	if filter.OwnerID != "" {
		builder = builder.Where("OwnerID = ?", filter.OwnerID)
	}
	if filter.GroupID != "" {
		builder = builder.Where("GroupID = ?", filter.GroupID)
	}
	if filter.State != "" {
		builder = builder.Where("State = ?", filter.State)
	}
	if filter.DNS != "" {
		builder = builder.Where("DNS = ?", filter.DNS)
	}

	return builder
}

// GetInstallationsCount returns the number of installations filtered by the
// deleteAt field.
func (sqlStore *SQLStore) GetInstallationsCount(includeDeleted bool) (int64, error) {
	stateCounts, err := sqlStore.getInstallationCountsByState(includeDeleted)
	if err != nil {
		return 0, errors.Wrap(err, "failed to query installation state counts")
	}
	var totalCount int64
	for _, count := range stateCounts {
		totalCount += count
	}

	return totalCount, nil
}

// GetInstallationsStatus returns status of all installations which aren't
// deleted.
func (sqlStore *SQLStore) GetInstallationsStatus() (*model.InstallationsStatus, error) {
	stateCounts, err := sqlStore.getInstallationCountsByState(false)
	if err != nil {
		return nil, errors.Wrap(err, "failed to query installation state counts")
	}

	var totalCount int64
	for _, count := range stateCounts {
		totalCount += count
	}
	stableCount := stateCounts[model.InstallationStateStable]
	hibernatingCount := stateCounts[model.InstallationStateHibernating]

	return &model.InstallationsStatus{
		InstallationsTotal:       totalCount,
		InstallationsStable:      stableCount,
		InstallationsHibernating: hibernatingCount,
		InstallationsUpdating:    totalCount - stableCount - hibernatingCount,
	}, nil
}

// getInstallationCountsByState returns the number of installations in each
// state.
func (sqlStore *SQLStore) getInstallationCountsByState(includeDeleted bool) (map[string]int64, error) {
	type Count struct {
		Count int64
		State string
	}
	var counts []Count

	installationBuilder := sq.
		Select("Count (*) as Count, State").
		From("Installation").
		GroupBy("State")
	if !includeDeleted {
		installationBuilder = installationBuilder.Where("DeleteAt = 0")
	}
	err := sqlStore.selectBuilder(sqlStore.db, &counts, installationBuilder)
	if err != nil {
		return nil, errors.Wrap(err, "failed to query for installations by state")
	}

	result := make(map[string]int64)
	for _, count := range counts {
		result[count.State] = count.Count
	}

	return result, nil
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

// GetSingleTenantDatabaseConfigForInstallation fetches single tenant database configuration
// for specified installation.
func (sqlStore *SQLStore) GetSingleTenantDatabaseConfigForInstallation(installationID string) (*model.SingleTenantDatabaseConfig, error) {
	builder := sq.Select("SingleTenantDatabaseConfigRaw").
		From("Installation").
		Where("ID = ?", installationID)

	dbConfig := struct {
		SingleTenantDatabaseConfigRaw []byte
	}{}

	err := sqlStore.getBuilder(sqlStore.db, &dbConfig, builder)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get single tenant database configuration by installationID")
	}

	if dbConfig.SingleTenantDatabaseConfigRaw == nil {
		return nil, fmt.Errorf("single tenant database configuration does not exist for installation")
	}

	singleTenantDBConfig := model.SingleTenantDatabaseConfig{}
	err = json.Unmarshal(dbConfig.SingleTenantDatabaseConfigRaw, &singleTenantDBConfig)
	if err != nil {
		return nil, errors.Wrap(err, "failed to unmarshall single tenant database configuration")
	}

	return &singleTenantDBConfig, nil
}

// CreateInstallation records the given installation to the database, assigning it a unique ID.
func (sqlStore *SQLStore) CreateInstallation(installation *model.Installation, annotations []*model.Annotation) error {
	tx, err := sqlStore.beginTransaction(sqlStore.db)
	if err != nil {
		return errors.Wrap(err, "failed to begin transaction")
	}
	defer tx.RollbackUnlessCommitted()

	err = sqlStore.createInstallation(tx, installation)
	if err != nil {
		return errors.Wrap(err, "failed to create installation")
	}

	if len(annotations) > 0 {
		annotations, err := sqlStore.getOrCreateAnnotations(tx, annotations)
		if err != nil {
			return errors.Wrap(err, "failed to get or create annotations")
		}

		_, err = sqlStore.createInstallationAnnotations(tx, installation.ID, annotations)
		if err != nil {
			return errors.Wrap(err, "failed to create annotations for installation")
		}
	}

	err = tx.Commit()
	if err != nil {
		return errors.Wrap(err, "failed to commit the transaction")
	}

	return nil
}

func (sqlStore *SQLStore) createInstallation(db execer, installation *model.Installation) error {
	installation.ID = model.NewID()
	installation.CreateAt = GetMillis()

	envJSON, err := json.Marshal(installation.MattermostEnv)
	if err != nil {
		return errors.Wrap(err, "unable to marshal MattermostEnv")
	}

	insertsMap := map[string]interface{}{
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
		"CRVersion":        installation.CRVersion,
	}

	singleTenantDBConfJSON, err := installation.SingleTenantDatabaseConfig.ToJSON()
	if err != nil {
		return errors.Wrap(err, "unable to marshal SingleTenantDatabaseConfig")
	}

	// For Postgres we cannot set typed nil as it is not mapped to NULL value.
	if singleTenantDBConfJSON != nil {
		insertsMap["SingleTenantDatabaseConfigRaw"] = singleTenantDBConfJSON
	}

	_, err = sqlStore.execBuilder(db, sq.
		Insert("Installation").
		SetMap(insertsMap),
	)
	if err != nil {
		return errors.Wrap(err, "failed to create installation")
	}

	return nil
}

// UpdateInstallation updates the given installation in the database.
func (sqlStore *SQLStore) UpdateInstallation(installation *model.Installation) error {
	return sqlStore.updateInstallation(sqlStore.db, installation)
}

func (sqlStore *SQLStore) updateInstallation(db execer, installation *model.Installation) error {
	if installation.ConfigMergedWithGroup() {
		return errors.New("unable to save installations that have merged group config")
	}
	envJSON, err := json.Marshal(installation.MattermostEnv)
	if err != nil {
		return errors.Wrap(err, "unable to marshal MattermostEnv")
	}

	_, err = sqlStore.execBuilder(db, sq.
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
			"CRVersion":        installation.CRVersion,
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

// UpdateInstallationCRVersion updates the given installation CRVersion.
func (sqlStore *SQLStore) UpdateInstallationCRVersion(installationID, crVersion string) error {
	_, err := sqlStore.execBuilder(sqlStore.db, sq.
		Update("Installation").
		SetMap(map[string]interface{}{
			"CRVersion": crVersion,
		}).
		Where("ID = ?", installationID),
	)
	if err != nil {
		return errors.Wrap(err, "failed to update installation")
	}

	return nil
}

// GetInstallationsTotalDatabaseWeight returns the total weight value of the
// provided installations.
func (sqlStore *SQLStore) GetInstallationsTotalDatabaseWeight(installationIDs []string) (float64, error) {
	installations, err := sqlStore.GetInstallations(&model.InstallationFilter{
		InstallationIDs: installationIDs,
		Paging:          model.AllPagesNotDeleted(),
	}, false, false)
	if err != nil {
		return 0, errors.Wrap(err, "failed to lookup installations in database")
	}

	var totalWeight float64
	for _, installation := range installations {
		totalWeight += installation.GetDatabaseWeight()
	}

	return totalWeight, nil
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
