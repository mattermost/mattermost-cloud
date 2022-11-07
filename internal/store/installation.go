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

const (
	installationTable = "Installation"
)

func init() {
	installationSelect = sq.
		Select(
			"Installation.ID", "Installation.Name", "OwnerID", "Version", "Image", "Database", "Filestore", "Size",
			"Affinity", "GroupID", "GroupSequence", "Installation.State", "License",
			"MattermostEnvRaw", "PriorityEnvRaw", "SingleTenantDatabaseConfigRaw", "ExternalDatabaseConfigRaw",
			"Installation.CreateAt", "Installation.DeleteAt",
			"APISecurityLock", "LockAcquiredBy", "LockAcquiredAt", "CRVersion",
		).
		From(installationTable)
}

type rawInstallation struct {
	*model.Installation
	MattermostEnvRaw              []byte
	PriorityEnvRaw                []byte
	SingleTenantDatabaseConfigRaw []byte
	ExternalDatabaseConfigRaw     []byte
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

	priorityEnv := &model.EnvVarMap{}
	if r.PriorityEnvRaw != nil {
		priorityEnv, err = model.EnvVarFromJSON(r.PriorityEnvRaw)
		if err != nil {
			return nil, err
		}
	}
	r.Installation.PriorityEnv = *priorityEnv

	if r.SingleTenantDatabaseConfigRaw != nil {
		singleTenantDBConfig := &model.SingleTenantDatabaseConfig{}
		err = json.Unmarshal(r.SingleTenantDatabaseConfigRaw, singleTenantDBConfig)
		if err != nil {
			return nil, err
		}
		r.Installation.SingleTenantDatabaseConfig = singleTenantDBConfig
	}

	if r.ExternalDatabaseConfigRaw != nil {
		externalDBConfig := &model.ExternalDatabaseConfig{}
		err = json.Unmarshal(r.ExternalDatabaseConfigRaw, externalDBConfig)
		if err != nil {
			return nil, err
		}
		r.Installation.ExternalDatabaseConfig = externalDBConfig
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
	builder := installationSelect

	// We need to apply DeleteAt constraint at join level just to avoid some edge cases
	// where there is no InstallationDNS for Installation.
	if !filter.IncludeDeleted {
		builder = builder.LeftJoin("InstallationDNS ON Installation.ID=InstallationID AND InstallationDNS.DeleteAt = 0")
	} else {
		builder = builder.LeftJoin("InstallationDNS ON Installation.ID=InstallationID")
	}

	builder = builder.
		GroupBy("Installation.ID").
		OrderBy("Installation.CreateAt ASC")
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
	builder = applyPagingFilter(builder, filter.Paging, "Installation")

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
		builder = builder.Where("InstallationDNS.DomainName = ?", filter.DNS)
	}
	if filter.Name != "" {
		builder = builder.Where("Installation.Name = ?", filter.Name)
	}

	return builder
}

// GetInstallationsCount returns the number of installations filtered by the
// deleteAt field.
func (sqlStore *SQLStore) GetInstallationsCount(filter *model.InstallationFilter) (int64, error) {
	stateCounts, err := sqlStore.getInstallationCount(filter)
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
	stateCounts, err := sqlStore.getInstallationCount(&model.InstallationFilter{
		Paging: model.AllPagesNotDeleted(),
	})
	if err != nil {
		return nil, errors.Wrap(err, "failed to query installation state counts")
	}

	var totalCount int64
	for _, count := range stateCounts {
		totalCount += count
	}
	stableCount := stateCounts[model.InstallationStateStable]
	hibernatingCount := stateCounts[model.InstallationStateHibernating]
	pendingDeletionCount := stateCounts[model.InstallationStateDeletionPending]

	return &model.InstallationsStatus{
		InstallationsTotal:           totalCount,
		InstallationsStable:          stableCount,
		InstallationsHibernating:     hibernatingCount,
		InstallationsPendingDeletion: pendingDeletionCount,
		InstallationsUpdating:        totalCount - stableCount - hibernatingCount - pendingDeletionCount,
	}, nil
}

// getInstallationCount returns the number of installations in each
// state.
func (sqlStore *SQLStore) getInstallationCount(filter *model.InstallationFilter) (map[string]int64, error) {
	type Count struct {
		Count int64
		State string
	}
	var counts []Count

	installationBuilder := sq.
		Select("Count (*) as Count, State").
		From("Installation").
		GroupBy("State")

	installationBuilder = sqlStore.applyInstallationFilter(installationBuilder, filter)

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

// GetUnlockedInstallationsPendingWork returns unlocked installations in a
// pending work state.
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

// GetUnlockedInstallationsPendingDeletion returns unlocked installations in a
// pending deletion state.
func (sqlStore *SQLStore) GetUnlockedInstallationsPendingDeletion() ([]*model.Installation, error) {
	builder := installationSelect.
		Where(sq.Eq{
			"State": model.InstallationStateDeletionPending,
		}).
		Where("LockAcquiredAt = 0").
		OrderBy("CreateAt ASC")

	var rawInstallations rawInstallations
	err := sqlStore.selectBuilder(sqlStore.db, &rawInstallations, builder)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get installations pending deletion")
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
func (sqlStore *SQLStore) CreateInstallation(installation *model.Installation, annotations []*model.Annotation, dnsRecords []*model.InstallationDNS) error {
	tx, err := sqlStore.beginTransaction(sqlStore.db)
	if err != nil {
		return errors.Wrap(err, "failed to begin transaction")
	}
	defer tx.RollbackUnlessCommitted()

	err = sqlStore.createInstallation(tx, installation)
	if err != nil {
		return errors.Wrap(err, "failed to create installation")
	}

	// We can do bulk insert for better performance, but currently we do not expect more than 1 record.
	for _, record := range dnsRecords {
		err = sqlStore.createInstallationDNS(tx, installation.ID, record)
		if err != nil {
			return err
		}
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
	installation.CreateAt = model.GetMillis()

	envJSON, err := json.Marshal(installation.MattermostEnv)
	if err != nil {
		return errors.Wrap(err, "unable to marshal MattermostEnv")
	}

	priorityEnvJSON, err := json.Marshal(installation.PriorityEnv)
	if err != nil {
		return errors.Wrap(err, "unable to marshal PriorityEnv")
	}

	insertsMap := map[string]interface{}{
		"Name":             installation.Name,
		"ID":               installation.ID,
		"OwnerID":          installation.OwnerID,
		"GroupID":          installation.GroupID,
		"GroupSequence":    nil,
		"Version":          installation.Version,
		"Image":            installation.Image,
		"Database":         installation.Database,
		"Filestore":        installation.Filestore,
		"Size":             installation.Size,
		"Affinity":         installation.Affinity,
		"State":            installation.State,
		"License":          installation.License,
		"MattermostEnvRaw": []byte(envJSON),
		"PriorityEnvRaw":   []byte(priorityEnvJSON),
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

	externalDBConfJSON, err := installation.ExternalDatabaseConfig.ToJSON()
	if err != nil {
		return errors.Wrap(err, "unable to marshal ExternalDatabaseConfig")
	}

	// For Postgres we cannot set typed nil as it is not mapped to NULL value.
	if externalDBConfJSON != nil {
		insertsMap["ExternalDatabaseConfigRaw"] = externalDBConfJSON
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

	priorityEnvJSON, err := json.Marshal(installation.PriorityEnv)
	if err != nil {
		return errors.Wrap(err, "unable to marshal PriorityEnv")
	}

	_, err = sqlStore.execBuilder(db, sq.
		Update("Installation").
		SetMap(map[string]interface{}{
			"Name":             installation.Name,
			"OwnerID":          installation.OwnerID,
			"GroupID":          installation.GroupID,
			"GroupSequence":    installation.GroupSequence,
			"Version":          installation.Version,
			"Image":            installation.Image,
			"Database":         installation.Database,
			"Filestore":        installation.Filestore,
			"Size":             installation.Size,
			"Affinity":         installation.Affinity,
			"License":          installation.License,
			"MattermostEnvRaw": []byte(envJSON),
			"PriorityEnvRaw":   []byte(priorityEnvJSON),
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
	return sqlStore.updateInstallationState(sqlStore.db, installation)
}

func (sqlStore *SQLStore) updateInstallationState(execer execer, installation *model.Installation) error {
	_, err := sqlStore.execBuilder(execer, sq.
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
	if len(installationIDs) == 0 {
		return float64(0), nil
	}

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
		Set("DeleteAt", model.GetMillis()).
		Where("ID = ?", id).
		Where("DeleteAt = 0"),
	)
	if err != nil {
		return errors.Wrap(err, "failed to mark installation as deleted")
	}

	return nil
}

// RecoverInstallation recovers a given installation from the deleted state.
func (sqlStore *SQLStore) RecoverInstallation(installation *model.Installation) error {
	_, err := sqlStore.execBuilder(sqlStore.db, sq.
		Update("Installation").
		Set("State", installation.State).
		Set("DeleteAt", 0).
		Where("ID = ?", installation.ID).
		Where("State = ?", model.InstallationStateDeleted).
		Where("DeleteAt != 0"),
	)
	if err != nil {
		return errors.Wrap(err, "failed to update installation recovery values")
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

// LockInstallations marks the installation(s) as locked for exclusive use by the caller.
func (sqlStore *SQLStore) LockInstallations(installationIDs []string, lockerID string) (bool, error) {
	return sqlStore.lockRows("Installation", installationIDs, lockerID)
}

// UnlockInstallations releases a lock previously acquired against a caller for a set of installations.
func (sqlStore *SQLStore) UnlockInstallations(installationIDs []string, lockerID string, force bool) (bool, error) {
	return sqlStore.unlockRows("Installation", installationIDs, lockerID, force)
}

// UpdateInstallationsState updates the set of installations to a new state.
func (sqlStore *SQLStore) UpdateInstallationsState(db execer, installationIDs []string, state string) error {
	_, err := sqlStore.execBuilder(db, sq.
		Update("Installation").
		SetMap(map[string]interface{}{
			"State": state,
		}).
		Where(sq.Eq{
			"ID": installationIDs,
		}),
	)
	if err != nil {
		return errors.Wrap(err, "failed to update installation state")
	}

	return nil
}
