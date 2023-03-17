// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.
//

package model

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestGroupDTOFromReader(t *testing.T) {
	t.Run("empty request", func(t *testing.T) {
		groupDTO, err := DTOFromReader[GroupDTO](bytes.NewReader([]byte(
			``,
		)))
		require.NoError(t, err)
		require.Equal(t, &GroupDTO{}, groupDTO)
	})

	t.Run("invalid request", func(t *testing.T) {
		groupDTO, err := DTOFromReader[GroupDTO](bytes.NewReader([]byte(
			`{test`,
		)))
		require.Error(t, err)
		require.Nil(t, groupDTO)
	})

	t.Run("request", func(t *testing.T) {
		groupDTO, err := DTOFromReader[GroupDTO](bytes.NewReader([]byte(`{
			"ID":"id",
			"Sequence":10,
			"Name": "group1",
			"Version": "12",
			"Annotations": [
				{"ID": "abc", "Name": "efg"}
			],
			"installation_count": 2
		}`)))
		require.NoError(t, err)

		expectedInstallations := int64(2)
		require.Equal(t, groupDTO.GetInstallationCount(), expectedInstallations)
		require.Equal(t, &GroupDTO{
			Group: &Group{
				ID:              "id",
				Sequence:        10,
				Name:            "group1",
				Version:         "12",
				APISecurityLock: false,
				LockAcquiredBy:  nil,
				LockAcquiredAt:  int64(0),
			},
			Annotations:       []*Annotation{{ID: "abc", Name: "efg"}},
			InstallationCount: &expectedInstallations,
		}, groupDTO)
	})
}

func TestGroupDTOsFromReader(t *testing.T) {
	t.Run("empty request", func(t *testing.T) {
		groupDTOs, err := DTOsFromReader[GroupDTO](bytes.NewReader([]byte(
			``,
		)))
		require.NoError(t, err)
		require.Equal(t, []*GroupDTO{}, groupDTOs)
	})

	t.Run("invalid request", func(t *testing.T) {
		groupDTOs, err := DTOsFromReader[GroupDTO](bytes.NewReader([]byte(
			`{test`,
		)))
		require.Error(t, err)
		require.Nil(t, groupDTOs)
	})

	t.Run("request", func(t *testing.T) {
		groupDTOs, err := DTOsFromReader[GroupDTO](bytes.NewReader([]byte(`[
			{
				"ID":"id",
				"Sequence":10,
				"Name": "group1",
				"Version": "12",
				"Annotations": [
					{"ID": "abc", "Name": "efg"}
				]
			},
			{
				"ID":"id2",
				"Sequence":11,
				"Name": "group2",
				"Version": "13"
			}
		]`)))
		require.NoError(t, err)
		require.Equal(t, []*GroupDTO{
			{
				Group: &Group{
					ID:              "id",
					Sequence:        10,
					Name:            "group1",
					Version:         "12",
					APISecurityLock: false,
					LockAcquiredBy:  nil,
					LockAcquiredAt:  int64(0),
				},
				Annotations: []*Annotation{{ID: "abc", Name: "efg"}},
			},
			{
				Group: &Group{
					ID:              "id2",
					Sequence:        11,
					Name:            "group2",
					Version:         "13",
					APISecurityLock: false,
					LockAcquiredBy:  nil,
					LockAcquiredAt:  int64(0),
				},
			},
		}, groupDTOs)
	})
}
