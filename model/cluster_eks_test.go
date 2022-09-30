// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.
//

package model

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestNewKopsMetadata(t *testing.T) {
	t.Run("nil payload", func(t *testing.T) {
		eksMetadata, err := NewEKSMetadata(nil)
		require.NoError(t, err)
		require.Nil(t, eksMetadata)
	})

	t.Run("invalid payload", func(t *testing.T) {
		_, err := NewEKSMetadata([]byte(`{`))
		require.Error(t, err)
	})

	t.Run("valid payload", func(t *testing.T) {
		eksMetadata, err := NewEKSMetadata([]byte(`{"ClusterRoleARN": "test"}`))
		require.NoError(t, err)
		require.Equal(t, "test", *eksMetadata.ClusterRoleARN)
	})
}
