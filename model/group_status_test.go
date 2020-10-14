// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.
//

package model

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestGroupStatusFromReader(t *testing.T) {
	t.Run("empty json", func(t *testing.T) {
		groupStatus, err := GroupStatusFromReader(bytes.NewReader([]byte(
			``,
		)))
		require.NoError(t, err)
		require.Equal(t, &GroupStatus{}, groupStatus)
	})

	t.Run("invalid json", func(t *testing.T) {
		groupStatus, err := GroupStatusFromReader(bytes.NewReader([]byte(
			`{test`,
		)))
		require.Error(t, err)
		require.Nil(t, groupStatus)
	})

	t.Run("valid group status", func(t *testing.T) {
		groupStatus, err := GroupStatusFromReader(bytes.NewReader([]byte(`{
			"InstallationsTotal": 4,
			"InstallationsUpdated": 2,
			"InstallationsBeingUpdated": 1,
			"InstallationsAwaitingUpdate": 1
		}`)))
		require.NoError(t, err)
		require.Equal(t, &GroupStatus{
			InstallationsTotal:          4,
			InstallationsUpdated:        2,
			InstallationsBeingUpdated:   1,
			InstallationsAwaitingUpdate: 1,
		}, groupStatus)
	})
}
