// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.
//

package model_test

import (
	"testing"

	"github.com/mattermost/mattermost-cloud/model"
	"github.com/stretchr/testify/require"
)

func TestNewKopsMetadata(t *testing.T) {
	t.Run("nil payload", func(t *testing.T) {
		kopsMetadata, err := model.NewKopsMetadata(nil)
		require.NoError(t, err)
		require.Nil(t, kopsMetadata)
	})

	t.Run("invalid payload", func(t *testing.T) {
		_, err := model.NewKopsMetadata([]byte(`{`))
		require.Error(t, err)
	})

	t.Run("valid payload", func(t *testing.T) {
		kopsMetadata, err := model.NewKopsMetadata([]byte(`{"Name": "name"}`))
		require.NoError(t, err)
		require.Equal(t, "name", kopsMetadata.Name)
	})
}
