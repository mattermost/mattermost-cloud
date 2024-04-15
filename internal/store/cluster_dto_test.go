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

func TestClusterDTOs(t *testing.T) {
	logger := testlib.MakeLogger(t)
	sqlStore := MakeTestSQLStore(t, logger)
	defer CloseConnection(t, sqlStore)

	t.Run("get unknown cluster DTO", func(t *testing.T) {
		cluster, err := sqlStore.GetClusterDTO("unknown")
		require.NoError(t, err)
		require.Nil(t, cluster)
	})

	annotation1 := model.Annotation{Name: "annotation1"}
	annotation2 := model.Annotation{Name: "annotation2"}

	// Create only one annotation beforehand to test both get by name and create.
	err := sqlStore.CreateAnnotation(&annotation1)
	require.NoError(t, err)

	annotations := []*model.Annotation{&annotation1, &annotation2}

	cluster1 := &model.Cluster{
		Provider:                "aws",
		Provisioner:             "kops",
		ProviderMetadataAWS:     &model.AWSMetadata{Zones: []string{"zone1"}},
		ProvisionerMetadataKops: &model.KopsMetadata{Version: "version1"},
		UtilityMetadata:         &model.UtilityMetadata{},
		PgBouncerConfig:         &model.PgBouncerConfig{},
		State:                   model.ClusterStateCreationRequested,
		AllowInstallations:      false,
	}

	cluster2 := &model.Cluster{
		Provider:               "azure",
		Provisioner:            model.ProvisionerEKS,
		ProviderMetadataAWS:    &model.AWSMetadata{Zones: []string{"zone1"}},
		ProvisionerMetadataEKS: &model.EKSMetadata{Version: "version1"},
		UtilityMetadata:        &model.UtilityMetadata{},
		PgBouncerConfig:        &model.PgBouncerConfig{},
		State:                  model.ClusterStateStable,
		AllowInstallations:     true,
	}

	err = sqlStore.CreateCluster(cluster1, annotations)
	require.NoError(t, err)

	err = sqlStore.CreateCluster(cluster2, nil)
	require.NoError(t, err)

	t.Run("get cluster DTO", func(t *testing.T) {
		clusterDTO, err := sqlStore.GetClusterDTO(cluster1.ID)
		require.NoError(t, err)
		assert.Equal(t, cluster1, clusterDTO.Cluster)
		assert.Equal(t, len(annotations), len(clusterDTO.Annotations))
		assert.Equal(t, annotations, model.SortAnnotations(clusterDTO.Annotations))
	})

	t.Run("get cluster DTOs", func(t *testing.T) {
		clusterDTOs, err := sqlStore.GetClusterDTOs(&model.ClusterFilter{Paging: model.AllPagesWithDeleted()})
		require.NoError(t, err)
		assert.Equal(t, 2, len(clusterDTOs))

		for _, c := range clusterDTOs {
			model.SortAnnotations(c.Annotations)
		}
		assert.Equal(t, []*model.ClusterDTO{cluster1.ToDTO(annotations), cluster2.ToDTO(nil)}, clusterDTOs)
	})
}
