// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.
//

package store

import (
	"testing"

	"github.com/mattermost/mattermost-cloud/internal/testlib"
	"github.com/mattermost/mattermost-cloud/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func Test_GroupDTO(t *testing.T) {
	logger := testlib.MakeLogger(t)
	sqlStore := MakeTestSQLStore(t, logger)

	group1 := &model.Group{
		Name: "group1",
	}
	annotations1 := []*model.Annotation{
		{Name: "ann-group1"},
		{Name: "ann-group2"},
	}
	createAnnotations(t, sqlStore, annotations1)
	err := sqlStore.CreateGroup(group1, annotations1)
	require.NoError(t, err)

	group2 := &model.Group{
		Name: "group2",
	}
	annotations2 := []*model.Annotation{
		{Name: "ann-group3"},
	}
	createAnnotations(t, sqlStore, annotations2)
	err = sqlStore.CreateGroup(group2, annotations2)
	require.NoError(t, err)

	group3 := &model.Group{
		Name: "group3",
	}
	err = sqlStore.CreateGroup(group3, nil)
	require.NoError(t, err)

	group4 := &model.Group{
		Name: "group4",
	}
	err = sqlStore.CreateGroup(group4, annotations1[:1])
	require.NoError(t, err)

	t.Run("get group1 DTO", func(t *testing.T) {
		group1DTO, err := sqlStore.GetGroupDTO(group1.ID)
		require.NoError(t, err)
		assert.Equal(t, group1, group1DTO.Group)
		assert.Equal(t, 2, len(group1DTO.Annotations))
		model.SortAnnotations(group1DTO.Annotations)
		assert.Equal(t, group1.ToDTO(annotations1), group1DTO)
	})

	t.Run("get group3 DTO", func(t *testing.T) {
		group3DTO, err := sqlStore.GetGroupDTO(group3.ID)
		require.NoError(t, err)
		assert.Equal(t, group3, group3DTO.Group)
		assert.Equal(t, 0, len(group3DTO.Annotations))
		assert.Equal(t, group3.ToDTO(nil), group3DTO)
	})

	t.Run("get group DTOs with status", func(t *testing.T) {
		groups, err := sqlStore.GetGroupDTOs(&model.GroupFilter{
			Paging:     model.AllPagesNotDeleted(),
			WithStatus: true,
		})
		assert.NoError(t, err)
		assert.NotZero(t, len(groups))
		for _, g := range groups {
			assert.NotNil(t, g.Status)
		}
	})

	t.Run("get group DTOs", func(t *testing.T) {
		testCases := []struct {
			Description string
			Filter      *model.GroupFilter
			Expected    []*model.GroupDTO
		}{
			{
				"page 0, perPage 0",
				&model.GroupFilter{
					Paging: model.Paging{
						Page:           0,
						PerPage:        0,
						IncludeDeleted: false,
					},
				},
				[]*model.GroupDTO{},
			},
			{
				"page 0, perPage 1",
				&model.GroupFilter{
					Paging: model.Paging{
						Page:           0,
						PerPage:        1,
						IncludeDeleted: false,
					},
				},
				[]*model.GroupDTO{group1.ToDTO(annotations1)},
			},
			{
				"with multiple annotations",
				&model.GroupFilter{
					Paging: model.AllPagesNotDeleted(),
					Annotations: &model.AnnotationsFilter{
						MatchAllIDs: []string{annotations1[0].ID, annotations1[1].ID},
					},
				},
				[]*model.GroupDTO{group1.ToDTO(annotations1)},
			},
			{
				"with single annotation",
				&model.GroupFilter{
					Paging: model.AllPagesNotDeleted(),
					Annotations: &model.AnnotationsFilter{
						MatchAllIDs: []string{annotations1[0].ID},
					},
				},
				[]*model.GroupDTO{group1.ToDTO(annotations1), group4.ToDTO(annotations1[:1])},
			},
		}

		for _, testCase := range testCases {
			t.Run(testCase.Description, func(t *testing.T) {
				actual, err := sqlStore.GetGroupDTOs(testCase.Filter)
				require.NoError(t, err)
				for _, g := range actual {
					model.SortAnnotations(g.Annotations)
				}
				assert.ElementsMatch(t, testCase.Expected, actual)
			})
		}
	})
}

func createAnnotations(t *testing.T, sqlStore *SQLStore, anns []*model.Annotation) {
	for _, ann := range anns {
		err := sqlStore.CreateAnnotation(ann)
		require.NoError(t, err)
	}
}
