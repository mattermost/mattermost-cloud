package model

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestGroupClone(t *testing.T) {
	group := &Group{
		ID:          "id",
		Name:        "name",
		Description: "description",
		Version:     "version",
	}

	clone := group.Clone()
	require.Equal(t, group, clone)

	// Verify changing fields in the clone doesn't affect the original.
	clone.Version = "new version"
	require.NotEqual(t, group, clone)
}

func TestGroupFromReader(t *testing.T) {
	t.Run("empty request", func(t *testing.T) {
		group, err := GroupFromReader(bytes.NewReader([]byte(
			``,
		)))
		require.NoError(t, err)
		require.Equal(t, &Group{}, group)
	})

	t.Run("invalid request", func(t *testing.T) {
		group, err := GroupFromReader(bytes.NewReader([]byte(
			`{test`,
		)))
		require.Error(t, err)
		require.Nil(t, group)
	})

	t.Run("request", func(t *testing.T) {
		group, err := GroupFromReader(bytes.NewReader([]byte(`{
			"ID":"id",
			"Name":"name",
			"Description":"description",
			"Version":"version",
			"CreateAt":10,
			"DeleteAt":20
		}`)))
		require.NoError(t, err)
		require.Equal(t, &Group{
			ID:          "id",
			Name:        "name",
			Description: "description",
			Version:     "version",
			CreateAt:    10,
			DeleteAt:    20,
		}, group)
	})
}

func TestGroupsFromReader(t *testing.T) {
	t.Run("empty request", func(t *testing.T) {
		groups, err := GroupsFromReader(bytes.NewReader([]byte(
			``,
		)))
		require.NoError(t, err)
		require.Equal(t, []*Group{}, groups)
	})

	t.Run("invalid request", func(t *testing.T) {
		groups, err := GroupsFromReader(bytes.NewReader([]byte(
			`{test`,
		)))
		require.Error(t, err)
		require.Nil(t, groups)
	})

	t.Run("request", func(t *testing.T) {
		group, err := GroupsFromReader(bytes.NewReader([]byte(`[
			{
				"ID":"id1",
				"Name":"name1",
				"Description":"description1",
				"Version":"version1",
				"CreateAt":10,
				"DeleteAt":20
			},
			{
				"ID":"id2",
				"Name":"name2",
				"Description":"description2",
				"Version":"version2",
				"CreateAt":30,
				"DeleteAt":40
			}
		]`)))
		require.NoError(t, err)
		require.Equal(t, []*Group{
			&Group{
				ID:          "id1",
				Name:        "name1",
				Description: "description1",
				Version:     "version1",
				CreateAt:    10,
				DeleteAt:    20,
			},
			&Group{
				ID:          "id2",
				Name:        "name2",
				Description: "description2",
				Version:     "version2",
				CreateAt:    30,
				DeleteAt:    40,
			},
		}, group)
	})
}
