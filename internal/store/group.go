package store

import (
	"database/sql"

	sq "github.com/Masterminds/squirrel"
	"github.com/mattermost/mattermost-cloud/model"
	"github.com/pkg/errors"
)

var groupSelect sq.SelectBuilder

func init() {
	groupSelect = sq.
		Select("ID", "Name", "Description", "Version", "CreateAt", "DeleteAt").
		From(`"Group"`)
}

// GetGroup fetches the given group by id.
func (sqlStore *SQLStore) GetGroup(id string) (*model.Group, error) {
	var group model.Group
	err := sqlStore.getBuilder(sqlStore.db, &group,
		groupSelect.Where("ID = ?", id),
	)
	if err == sql.ErrNoRows {
		return nil, nil
	} else if err != nil {
		return nil, errors.Wrap(err, "failed to get group by id")
	}

	return &group, nil
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

	var groups []*model.Group
	err := sqlStore.selectBuilder(sqlStore.db, &groups, builder)
	if err != nil {
		return nil, errors.Wrap(err, "failed to query for groups")
	}

	return groups, nil
}

// CreateGroup records the given group to the database, assigning it a unique ID.
func (sqlStore *SQLStore) CreateGroup(group *model.Group) error {
	group.ID = model.NewID()
	group.CreateAt = GetMillis()

	_, err := sqlStore.execBuilder(sqlStore.db, sq.
		Insert(`"Group"`).
		SetMap(map[string]interface{}{
			"ID":          group.ID,
			"Name":        group.Name,
			"Description": group.Description,
			"Version":     group.Version,
			"CreateAt":    group.CreateAt,
			"DeleteAt":    0,
		}),
	)
	if err != nil {
		return errors.Wrap(err, "failed to create group")
	}

	return nil
}

// UpdateGroup updates the given group in the database.
func (sqlStore *SQLStore) UpdateGroup(group *model.Group) error {
	_, err := sqlStore.execBuilder(sqlStore.db, sq.
		Update(`"Group"`).
		SetMap(map[string]interface{}{
			"Name":        group.Name,
			"Description": group.Description,
			"Version":     group.Version,
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
