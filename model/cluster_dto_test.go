// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.
//

package model

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestClusterDTOFromReader(t *testing.T) {
	t.Run("empty request", func(t *testing.T) {
		clusterDTO, err := DTOFromReader[ClusterDTO](bytes.NewReader([]byte(
			``,
		)))
		require.NoError(t, err)
		require.Equal(t, &ClusterDTO{}, clusterDTO)
	})

	t.Run("invalid request", func(t *testing.T) {
		clusterDTO, err := DTOFromReader[ClusterDTO](bytes.NewReader([]byte(
			`{test`,
		)))
		require.Error(t, err)
		require.Nil(t, clusterDTO)
	})

	t.Run("request", func(t *testing.T) {
		clusterDTO, err := DTOFromReader[ClusterDTO](bytes.NewReader([]byte(
			`{"ID":"id","Provider":"aws","Annotations":[{"ID":"abc","Name":"efg"}]}`,
		)))
		require.NoError(t, err)
		require.Equal(t, &ClusterDTO{Cluster: &Cluster{ID: "id", Provider: "aws"}, Annotations: []*Annotation{
			{ID: "abc", Name: "efg"},
		}}, clusterDTO)
	})
}

func TestClustersDTOFromReader(t *testing.T) {
	t.Run("empty request", func(t *testing.T) {
		clusterDTOs, err := DTOsFromReader[ClusterDTO](bytes.NewReader([]byte(
			``,
		)))
		require.NoError(t, err)
		require.Equal(t, []*ClusterDTO{}, clusterDTOs)
	})

	t.Run("invalid request", func(t *testing.T) {
		clusterDTOs, err := DTOsFromReader[ClusterDTO](bytes.NewReader([]byte(
			`{test`,
		)))
		require.Error(t, err)
		require.Nil(t, clusterDTOs)
	})

	t.Run("request", func(t *testing.T) {
		clusterDTOs, err := DTOsFromReader[ClusterDTO](bytes.NewReader([]byte(
			`[{"ID":"id1","Provider":"aws","Annotations":[{"ID":"abc","Name":"efg"}]},{"ID":"id2","Provider":"aws"}]`,
		)))
		require.NoError(t, err)
		require.Equal(t, []*ClusterDTO{
			{
				Cluster:     &Cluster{ID: "id1", Provider: "aws"},
				Annotations: []*Annotation{{ID: "abc", Name: "efg"}},
			},
			{
				Cluster: &Cluster{ID: "id2", Provider: "aws"},
			},
		}, clusterDTOs)
	})
}
