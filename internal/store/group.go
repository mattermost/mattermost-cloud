// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.
//

package store

import (
	"database/sql"
	"fmt"

	sq "github.com/Masterminds/squirrel"
	"github.com/mattermost/mattermost-cloud/model"
	"github.com/pkg/errors"
)

const (
	groupTable = `"Group"`
)

var groupSelect sq.SelectBuilder

type rawGroup struct {
	*model.Group
	MattermostEnvRaw []byte
}

type rawGroups []*rawGroup

func init() {
	groupSelect = sq.
		Select(`"Group".ID`, `"Group".Name`, "Description", "Version", "Image", "Sequence",
			"CreateAt", "DeleteAt", "MattermostEnvRaw", "MaxRolling",
			"APISecurityLock", "LockAcquiredBy", "LockAcquiredAt").
		From(groupTable)
}

func (r *rawGroup) toGroup() (*model.Group, error) {
	// We only need to set values that are converted from a raw database format.
	var err error
	mattermostEnv := &model.EnvVarMap{}
	if r.MattermostEnvRaw != nil {
		mattermostEnv, err = model.EnvVarFromJSON(r.MattermostEnvRaw)
		if err != nil {
			return nil, err
		}
	}

	r.Group.MattermostEnv = *mattermostEnv
	return r.Group, nil
}

func (rs *rawGroups) toGroups() ([]*model.Group, error) {
	var groups []*model.Group
	for _, rawGroup := range *rs {
		group, err := rawGroup.toGroup()
		if err != nil {
			return nil, err
		}
		groups = append(groups, group)
	}

	return groups, nil
}

// GetUnlockedGroupsPendingWork returns unlocked groups that have installations
// that require configuration reconciliation.
func (sqlStore *SQLStore) GetUnlockedGroupsPendingWork() ([]*model.Group, error) {
	groupBuilder := groupSelect.
		Where("LockAcquiredAt = 0").
		Where("DeleteAt = 0")

	var allRawGroups rawGroups
	err := sqlStore.selectBuilder(sqlStore.db, &allRawGroups, groupBuilder)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get groups pending work")
	}

	var rawGroupsPendingWork rawGroups
	for _, rawGroup := range allRawGroups {
		var installations []*model.Installation

		// Look for a single non-deleted installation that is not on the group's
		// sequence number. If even a single value is returned we know this
		// group has work to be done.
		installationBuilder := sq.
			Select("ID").
			From("Installation").
			Where("GroupID = ?", rawGroup.ID).
			Where("State = ?", model.InstallationStateStable).
			Where("(GroupSequence IS NULL OR GroupSequence != ?)", rawGroup.Sequence).
			Where("DeleteAt = 0").
			Limit(1)
		err = sqlStore.selectBuilder(sqlStore.db, &installations, installationBuilder)
		if err != nil {
			return nil, err
		}
		if len(installations) > 0 {
			rawGroupsPendingWork = append(rawGroupsPendingWork, rawGroup)
		}
	}

	return rawGroupsPendingWork.toGroups()
}

// GroupRollingMetadata is a batch of information about a group where installatons
// are being rolled to match a new config.
type GroupRollingMetadata struct {
	InstallationIDsToBeRolled []string
	InstallationsTotalCount   int64
	InstallationsRolling      int64
}

// GetGroupRollingMetadata returns installation IDs and metadata related to
// installation configuration reconciliation from group updates.
//
// Note: custom SQL queries are used here instead of calling GetInstallations().
// This is done for performance as we don't need the actual installation objects
// in most cases.
func (sqlStore *SQLStore) GetGroupRollingMetadata(groupID string) (*GroupRollingMetadata, error) {
	group, err := sqlStore.GetGroup(groupID)
	if err != nil {
		return nil, err
	}

	metadata := &GroupRollingMetadata{InstallationIDsToBeRolled: []string{}}

	var installations []*model.Installation
	err = sqlStore.queryInstallationsToBeRolledOut(
		[]string{"ID"},
		group,
		&installations,
	)
	if err != nil {
		return nil, errors.Wrap(err, "failed to query for installations to be roll out")
	}
	for _, installation := range installations {
		metadata.InstallationIDsToBeRolled = append(metadata.InstallationIDsToBeRolled, installation.ID)
	}

	metadata.InstallationsTotalCount, err = sqlStore.countInstallationsInGroup(group)
	if err != nil {
		return nil, errors.Wrap(err, "failed to query for total installations count in a group")
	}

	var stableOrHibernateResult countResult
	installationBuilder := sq.
		Select("Count (*)").
		From("Installation").
		Where("GroupID = ?", group.ID).
		Where("(State = ? OR State = ? OR State = ?)", model.InstallationStateStable, model.InstallationStateHibernating, model.InstallationStateDeletionPending).
		Where("DeleteAt = 0")
	err = sqlStore.selectBuilder(sqlStore.db, &stableOrHibernateResult, installationBuilder)
	if err != nil {
		return nil, errors.Wrap(err, "failed to query for stable installations count in a group")
	}
	count, err := stableOrHibernateResult.value()
	if err != nil {
		return nil, errors.Wrap(err, "failed to get count result of stable installations")
	}
	metadata.InstallationsRolling = metadata.InstallationsTotalCount - count

	if metadata.InstallationsRolling < 0 {
		return nil, errors.Errorf("found more stable+hibernating installations (%d) than total installations (%d)", count, metadata.InstallationsTotalCount)
	}

	return metadata, nil
}

// GetGroupStatus returns total number of installations in the group as well as number or
// Installations already rolled out and awaiting rollout.
//
// Note: This function uses the same conditions as GetGroupRollingMetadata to be more accurate
// with the internal state seen by the Group Supervisor.
func (sqlStore *SQLStore) GetGroupStatus(groupID string) (*model.GroupStatus, error) {
	group, err := sqlStore.GetGroup(groupID)
	if err != nil {
		return nil, err
	}
	if group == nil {
		return nil, nil
	}

	var awaitingUpdateResult countResult
	err = sqlStore.queryInstallationsToBeRolledOut(
		[]string{"Count (*)"},
		group,
		&awaitingUpdateResult,
	)
	if err != nil {
		return nil, errors.Wrap(err, "failed to query for count of installation to be rolled out")
	}
	awaitingUpdateCount, err := awaitingUpdateResult.value()
	if err != nil {
		return nil, errors.Wrap(err, "failed to get count result of installations to be rolled out")
	}

	var updatedResult countResult
	installationBuilder := sq.
		Select("Count (*)").
		From("Installation").
		Where("GroupID = ?", group.ID).
		Where("GroupSequence = ?", group.Sequence).
		Where("State = ?", model.InstallationStateStable).
		Where("DeleteAt = 0")
	err = sqlStore.selectBuilder(sqlStore.db, &updatedResult, installationBuilder)
	if err != nil {
		return nil, errors.Wrap(err, "failed to query for rolled out installations")
	}
	updatedCount, err := updatedResult.value()
	if err != nil {
		return nil, errors.Wrap(err, "failed to get count result of rolled out installations")
	}

	var hibernatingResult countResult
	installationBuilder = sq.
		Select("Count (*)").
		From("Installation").
		Where("GroupID = ?", group.ID).
		Where("State = ?", model.InstallationStateHibernating).
		Where("DeleteAt = 0")
	err = sqlStore.selectBuilder(sqlStore.db, &hibernatingResult, installationBuilder)
	if err != nil {
		return nil, errors.Wrap(err, "failed to query for hibernating installations")
	}
	hibernatingCount, err := hibernatingResult.value()
	if err != nil {
		return nil, errors.Wrap(err, "failed to get count result of hibernating installations")
	}

	var pendingDeletionResult countResult
	installationBuilder = sq.
		Select("Count (*)").
		From("Installation").
		Where("GroupID = ?", group.ID).
		Where("State = ?", model.InstallationStateDeletionPending).
		Where("DeleteAt = 0")
	err = sqlStore.selectBuilder(sqlStore.db, &pendingDeletionResult, installationBuilder)
	if err != nil {
		return nil, errors.Wrap(err, "failed to query for deletion pending installations")
	}
	deletionPendingCount, err := pendingDeletionResult.value()
	if err != nil {
		return nil, errors.Wrap(err, "failed to get count result of deletion pending installations")
	}

	totalCount, err := sqlStore.countInstallationsInGroup(group)
	if err != nil {
		return nil, errors.Wrap(err, "failed to query for total installations count in a group")
	}

	updatingCount := totalCount - updatedCount - awaitingUpdateCount - hibernatingCount - deletionPendingCount

	return &model.GroupStatus{
		InstallationsTotal:           totalCount,
		InstallationsUpdated:         updatedCount,
		InstallationsUpdating:        updatingCount,
		InstallationsHibernating:     hibernatingCount,
		InstallationsPendingDeletion: deletionPendingCount,
		InstallationsAwaitingUpdate:  awaitingUpdateCount,
	}, nil
}

func (sqlStore *SQLStore) queryInstallationsToBeRolledOut(columns []string, group *model.Group, dest interface{}) error {
	builder := sq.
		Select(columns...).
		From("Installation").
		Where("GroupID = ?", group.ID).
		Where("(GroupSequence IS NULL OR GroupSequence != ?)", group.Sequence).
		Where("State = ?", model.InstallationStateStable).
		Where("DeleteAt = 0")

	return sqlStore.selectBuilder(sqlStore.db, dest, builder)
}

func (sqlStore *SQLStore) countInstallationsInGroup(group *model.Group) (int64, error) {
	var totalResult countResult
	builder := sq.
		Select("Count (*)").
		From("Installation").
		Where("GroupID = ?", group.ID).
		Where("DeleteAt = 0")
	err := sqlStore.selectBuilder(sqlStore.db, &totalResult, builder)
	if err != nil {
		return 0, err
	}
	return totalResult.value()
}

// GetGroup fetches the given group by id.
func (sqlStore *SQLStore) GetGroup(id string) (*model.Group, error) {
	var rawGroup rawGroup
	err := sqlStore.getBuilder(sqlStore.db, &rawGroup,
		groupSelect.Where("ID = ?", id),
	)
	if err == sql.ErrNoRows {
		return nil, nil
	} else if err != nil {
		return nil, errors.Wrap(err, "failed to get group by id")
	}

	return rawGroup.toGroup()
}

// GetGroups fetches the given page of created groups. The first page is 0.
func (sqlStore *SQLStore) GetGroups(filter *model.GroupFilter) ([]*model.Group, error) {
	builder := groupSelect.
		OrderBy("CreateAt ASC")

	builder = applyGroupsFilter(builder, filter)

	var rawGroups rawGroups
	err := sqlStore.selectBuilder(sqlStore.db, &rawGroups, builder)
	if err != nil {
		return nil, errors.Wrap(err, "failed to query for groups")
	}

	return rawGroups.toGroups()
}

func applyGroupsFilter(builder sq.SelectBuilder, filter *model.GroupFilter) sq.SelectBuilder {
	if filter == nil {
		return builder
	}

	builder = applyPagingFilter(builder, filter.Paging)

	if filter.Annotations != nil && len(filter.Annotations.MatchAllIDs) > 0 {
		builder = builder.Join(fmt.Sprintf(`%s ON "Group".ID=%s.GroupID`, groupAnnotationTable, groupAnnotationTable)).
			Where(sq.Eq{fmt.Sprintf("%s.AnnotationID", groupAnnotationTable): filter.Annotations.MatchAllIDs}).
			GroupBy(`"Group".ID`).
			Having(fmt.Sprintf("count(DISTINCT %s.AnnotationID) = ?", groupAnnotationTable), len(filter.Annotations.MatchAllIDs))
	}

	return builder
}

// CreateGroup records the given group to the database, assigning it a unique ID.
func (sqlStore *SQLStore) CreateGroup(group *model.Group, annotations []*model.Annotation) error {
	tx, err := sqlStore.beginTransaction(sqlStore.db)
	if err != nil {
		return errors.Wrap(err, "failed to start transaction")
	}
	defer tx.RollbackUnlessCommitted()

	err = sqlStore.createGroup(tx, group)
	if err != nil {
		return err
	}

	if len(annotations) > 0 {
		annotations, err := sqlStore.getOrCreateAnnotations(tx, annotations)
		if err != nil {
			return errors.Wrap(err, "failed to get or create annotations")
		}

		_, err = sqlStore.createGroupAnnotations(tx, group.ID, annotations)
		if err != nil {
			return errors.Wrap(err, "failed to create group annotations")
		}
	}

	err = tx.Commit()
	if err != nil {
		return errors.Wrap(err, "failed to commit transaction creating group")
	}

	return nil
}

func (sqlStore *SQLStore) createGroup(db execer, group *model.Group) error {
	group.ID = model.NewID()
	group.CreateAt = model.GetMillis()
	envVarMap, err := group.MattermostEnv.ToJSON()
	if err != nil {
		return err
	}

	_, err = sqlStore.execBuilder(db, sq.
		Insert(groupTable).
		SetMap(map[string]interface{}{
			"ID":               group.ID,
			"Sequence":         0,
			"Name":             group.Name,
			"Image":            group.Image,
			"Description":      group.Description,
			"Version":          group.Version,
			"MattermostEnvRaw": envVarMap,
			"MaxRolling":       group.MaxRolling,
			"CreateAt":         group.CreateAt,
			"DeleteAt":         0,
			"APISecurityLock":  group.APISecurityLock,
			"LockAcquiredBy":   nil,
			"LockAcquiredAt":   0,
		}),
	)
	if err != nil {
		return errors.Wrap(err, "failed to create group")
	}
	return nil
}

// UpdateGroup updates the given group in the database. If a value was updated
// that will possibly affect installation config then update the group sequence
// number.
func (sqlStore *SQLStore) UpdateGroup(group *model.Group, forceUpdateSequence bool) error {
	originalGroup, err := sqlStore.GetGroup(group.ID)
	if err != nil {
		return errors.Wrap(err, "failed to lookup original group")
	}

	originalEnvVarMap, err := originalGroup.MattermostEnv.ToJSON()
	if err != nil {
		return errors.Wrap(err, "failed to create original EnvVarMap JSON")
	}
	envVarMap, err := group.MattermostEnv.ToJSON()
	if err != nil {
		return errors.Wrap(err, "failed to create new EnvVarMap JSON")
	}

	// Values that don't bump the group sequence number:
	// - Name
	// - Description
	// - MaxRolling
	if forceUpdateSequence ||
		originalGroup.Version != group.Version ||
		originalGroup.Image != group.Image ||
		string(originalEnvVarMap) != string(envVarMap) {
		group.Sequence = originalGroup.Sequence + 1
	}

	_, err = sqlStore.execBuilder(sqlStore.db, sq.
		Update(groupTable).
		SetMap(map[string]interface{}{
			"Sequence":         group.Sequence,
			"Name":             group.Name,
			"Description":      group.Description,
			"Version":          group.Version,
			"Image":            group.Image,
			"MattermostEnvRaw": envVarMap,
			"MaxRolling":       group.MaxRolling,
		}).
		Where("ID = ?", group.ID),
	)
	if err != nil {
		return errors.Wrap(err, "failed to update group")
	}

	return nil
}

// DeleteGroup marks the given group as deleted, but does not remove the record from the
// database.
func (sqlStore *SQLStore) DeleteGroup(id string) error {
	_, err := sqlStore.execBuilder(sqlStore.db, sq.
		Update(groupTable).
		Set("DeleteAt", model.GetMillis()).
		Where("ID = ?", id).
		Where("DeleteAt = 0"),
	)
	if err != nil {
		return errors.Wrap(err, "failed to mark group as deleted")
	}

	return nil
}

// LockGroup marks the group as locked for exclusive use by the caller.
func (sqlStore *SQLStore) LockGroup(groupID, lockerID string) (bool, error) {
	return sqlStore.lockRows(groupTable, []string{groupID}, lockerID)
}

// UnlockGroup releases a lock previously acquired against a caller.
func (sqlStore *SQLStore) UnlockGroup(groupID, lockerID string, force bool) (bool, error) {
	return sqlStore.unlockRows(groupTable, []string{groupID}, lockerID, force)
}

// LockGroupAPI locks updates to the group from the API.
func (sqlStore *SQLStore) LockGroupAPI(id string) error {
	return sqlStore.setGroupAPILock(id, true)
}

// UnlockGroupAPI unlocks updates to the group from the API.
func (sqlStore *SQLStore) UnlockGroupAPI(id string) error {
	return sqlStore.setGroupAPILock(id, false)
}

func (sqlStore *SQLStore) setGroupAPILock(id string, lock bool) error {
	_, err := sqlStore.execBuilder(sqlStore.db, sq.
		Update(groupTable).
		Set("APISecurityLock", lock).
		Where("ID = ?", id),
	)
	if err != nil {
		return errors.Wrap(err, "failed to store group API lock")
	}

	return nil
}
