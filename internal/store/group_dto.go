// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.
//

package store

import (
	"github.com/mattermost/mattermost-cloud/model"
	"github.com/pkg/errors"
)

// GetGroupDTO fetches the given group by id with data from connected tables.
func (sqlStore *SQLStore) GetGroupDTO(id string) (*model.GroupDTO, error) {
	group, err := sqlStore.GetGroup(id)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get group")
	}
	if group == nil {
		return nil, nil
	}

	annotations, err := sqlStore.getAnnotationsForGroup(sqlStore.db, id)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get annotations for installation")
	}
	return group.ToDTO(annotations), nil
}

// GetGroupDTOs fetches the given page of groups with data from connected tables. The first page is 0.
func (sqlStore *SQLStore) GetGroupDTOs(filter *model.GroupFilter) ([]*model.GroupDTO, error) {
	groups, err := sqlStore.GetGroups(filter)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get installations")
	}

	ids := make([]string, 0, len(groups))
	for _, g := range groups {
		ids = append(ids, g.ID)
	}

	annotations, err := sqlStore.getAnnotationsForGroups(ids)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get annotations for groups")
	}

	dtos := make([]*model.GroupDTO, 0, len(groups))
	for _, g := range groups {
		dtos = append(dtos, g.ToDTO(annotations[g.ID]))
	}

	return dtos, nil
}
