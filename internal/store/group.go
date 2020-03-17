package store

import (
	"database/sql"
	"reflect"

	sq "github.com/Masterminds/squirrel"
	"github.com/mattermost/mattermost-cloud/model"
	"github.com/pkg/errors"
)

var groupSelect sq.SelectBuilder

type rawGroup struct {
	*model.Group
	MattermostEnvRaw []byte
}

type rawGroups []*rawGroup

func init() {
	groupSelect = sq.
		Select("ID", "Name", "Description", "Version", "Sequence", "CreateAt",
			"DeleteAt", "MattermostEnvRaw", "MaxRolling", "LockAcquiredBy",
			"LockAcquiredAt").
		From(`"Group"`)
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
			Where("GroupSequence IS NULL OR GroupSequence != ?", rawGroup.Sequence).
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

	if filter.PerPage != model.AllPerPage {
		builder = builder.
			Limit(uint64(filter.PerPage)).
			Offset(uint64(filter.Page * filter.PerPage))
	}

	if !filter.IncludeDeleted {
		builder = builder.Where("DeleteAt = 0")
	}

	var rawGroups rawGroups
	err := sqlStore.selectBuilder(sqlStore.db, &rawGroups, builder)
	if err != nil {
		return nil, errors.Wrap(err, "failed to query for groups")
	}

	return rawGroups.toGroups()
}

// CreateGroup records the given group to the database, assigning it a unique ID.
func (sqlStore *SQLStore) CreateGroup(group *model.Group) error {
	group.ID = model.NewID()
	group.CreateAt = GetMillis()
	envVarMap, err := group.MattermostEnv.ToJSON()
	if err != nil {
		return err
	}

	_, err = sqlStore.execBuilder(sqlStore.db, sq.
		Insert(`"Group"`).
		SetMap(map[string]interface{}{
			"ID":               group.ID,
			"Sequence":         0,
			"Name":             group.Name,
			"Description":      group.Description,
			"Version":          group.Version,
			"MattermostEnvRaw": envVarMap,
			"MaxRolling":       group.MaxRolling,
			"CreateAt":         group.CreateAt,
			"DeleteAt":         0,
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
func (sqlStore *SQLStore) UpdateGroup(group *model.Group) error {
	originalGroup, err := sqlStore.GetGroup(group.ID)
	if err != nil {
		return err
	}
	if originalGroup.Version != group.Version || reflect.DeepEqual(originalGroup.MattermostEnv, group.MattermostEnv) {
		// Update the sequence number, but don't trust the group sequence number
		// that was passed in.
		group.Sequence = originalGroup.Sequence + 1
	}
	envVarMap, err := group.MattermostEnv.ToJSON()
	if err != nil {
		return err
	}
	_, err = sqlStore.execBuilder(sqlStore.db, sq.
		Update(`"Group"`).
		SetMap(map[string]interface{}{
			"Sequence":         group.Sequence,
			"Name":             group.Name,
			"Description":      group.Description,
			"Version":          group.Version,
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
		Update(`"Group"`).
		Set("DeleteAt", GetMillis()).
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
	return sqlStore.lockRows(`"Group"`, []string{groupID}, lockerID)
}

// UnlockGroup releases a lock previously acquired against a caller.
func (sqlStore *SQLStore) UnlockGroup(groupID, lockerID string, force bool) (bool, error) {
	return sqlStore.unlockRows(`"Group"`, []string{groupID}, lockerID, force)
}
