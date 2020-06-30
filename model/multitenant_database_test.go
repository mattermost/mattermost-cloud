// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.
//

package model

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestGetAndSetInstallationIDs(t *testing.T) {
	t.Run("get empty", func(t *testing.T) {
		database := MultitenantDatabase{}
		err := database.SetInstallationIDs(MultitenantDatabaseInstallationIDs{})
		require.NoError(t, err)

		ids, err := database.GetInstallationIDs()
		require.NoError(t, err)
		require.Equal(t, 0, len(ids))
	})

	t.Run("get data", func(t *testing.T) {
		expectedIDs := MultitenantDatabaseInstallationIDs{"id1", "id2", "id3"}

		database := MultitenantDatabase{}
		err := database.SetInstallationIDs(expectedIDs)
		require.NoError(t, err)

		ids, err := database.GetInstallationIDs()
		require.NoError(t, err)
		require.Equal(t, 3, len(ids))
		require.Equal(t, ids, expectedIDs)
	})

	t.Run("get error", func(t *testing.T) {
		database := MultitenantDatabase{
			RawInstallationIDs: []byte{'a', 'b', 'c'},
		}
		ids, err := database.GetInstallationIDs()
		require.Error(t, err)
		require.Equal(t, "failed to unmarshal installation IDs: invalid character "+
			"'a' looking for beginning of value", err.Error())
		require.Nil(t, ids)
	})
}
func TestMultitenantDatabaseInstallationIDs(t *testing.T) {
	t.Run("add id", func(t *testing.T) {
		installationIDs := MultitenantDatabaseInstallationIDs{}
		require.Equal(t, 0, len(installationIDs))

		installationIDs.Add("id1")
		require.Equal(t, 1, len(installationIDs))
	})

	t.Run("remove id", func(t *testing.T) {
		installationIDs := MultitenantDatabaseInstallationIDs{}
		require.Equal(t, 0, len(installationIDs))

		installationIDs.Add("id1")
		installationIDs.Add("id2")
		installationIDs.Add("id3")
		require.Equal(t, 3, len(installationIDs))

		installationIDs.Remove("id3")
		require.Equal(t, 2, len(installationIDs))

		installationIDs.Remove("id")
		require.Equal(t, 2, len(installationIDs))

		installationIDs.Remove("id1")
		require.Equal(t, 1, len(installationIDs))

		installationIDs.Remove("id2")
		require.Equal(t, 0, len(installationIDs))
	})

	t.Run("contain id", func(t *testing.T) {
		installationIDs := MultitenantDatabaseInstallationIDs{}

		installationIDs.Add("id1")
		installationIDs.Add("id2")
		installationIDs.Add("id3")

		require.True(t, installationIDs.Contains("id2"))
		require.False(t, installationIDs.Contains("id"))
	})
}
