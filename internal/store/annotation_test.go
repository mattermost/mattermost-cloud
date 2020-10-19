// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.
//

package store

import (
	"strings"
	"testing"

	"github.com/mattermost/mattermost-cloud/internal/testlib"
	"github.com/mattermost/mattermost-cloud/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAnnotations_Cluster(t *testing.T) {
	logger := testlib.MakeLogger(t)
	sqlStore := MakeTestSQLStore(t, logger)
	defer CloseConnection(t, sqlStore)

	annotation1 := model.Annotation{Name: "annotation1"}
	annotation2 := model.Annotation{Name: "annotation2"}

	err := sqlStore.CreateAnnotation(&annotation1)
	require.NoError(t, err)

	err = sqlStore.CreateAnnotation(&annotation2)
	require.NoError(t, err)

	t.Run("fail to create annotations with same name", func(t *testing.T) {
		err := sqlStore.CreateAnnotation(&model.Annotation{Name: annotation1.Name})
		require.Error(t, err)
		assert.Contains(t, strings.ToLower(err.Error()), "unique constraint") // Make sure error comes from DB
	})

	t.Run("get annotation by name", func(t *testing.T) {
		annotation, err := sqlStore.GetAnnotationByName(annotation1.Name)
		require.NoError(t, err)
		assert.Equal(t, &annotation1, annotation)
	})

	t.Run("get unknown annotation", func(t *testing.T) {
		annotation, err := sqlStore.GetAnnotationByName("unknown")
		require.NoError(t, err)
		assert.Nil(t, annotation)
	})

	annotations := []*model.Annotation{&annotation1, &annotation2}

	cluster1 := model.Cluster{}
	err = sqlStore.createCluster(sqlStore.db, &cluster1)
	require.NoError(t, err)

	_, err = sqlStore.CreateClusterAnnotations(cluster1.ID, annotations)
	require.NoError(t, err)

	t.Run("get annotations for cluster", func(t *testing.T) {
		annotationsForCluster, err := sqlStore.GetAnnotationsForCluster(cluster1.ID)
		require.NoError(t, err)
		assert.Equal(t, len(annotations), len(annotationsForCluster))
		assert.True(t, containsAnnotation(annotation1, annotationsForCluster))
		assert.True(t, containsAnnotation(annotation2, annotationsForCluster))
	})

	t.Run("fail to assign the same annotation to the cluster twice", func(t *testing.T) {
		_, err = sqlStore.CreateClusterAnnotations(cluster1.ID, annotations)
		require.Error(t, err)
		assert.Contains(t, strings.ToLower(err.Error()), "unique constraint") // Make sure error comes from DB
	})

	cluster2 := model.Cluster{}
	err = sqlStore.CreateCluster(&cluster2, annotations)
	require.NoError(t, err)

	t.Run("get annotations for cluster2", func(t *testing.T) {
		annotationsForCluster, err := sqlStore.GetAnnotationsForCluster(cluster2.ID)
		require.NoError(t, err)
		assert.Equal(t, len(annotations), len(annotationsForCluster))
		assert.True(t, containsAnnotation(annotation1, annotationsForCluster))
		assert.True(t, containsAnnotation(annotation2, annotationsForCluster))
	})

	t.Run("get annotations for clusters", func(t *testing.T) {
		annotationsForClusters, err := sqlStore.GetAnnotationsForClusters(&model.ClusterFilter{PerPage: model.AllPerPage})
		require.NoError(t, err)
		assert.Equal(t, 2, len(annotationsForClusters))
		assert.True(t, containsAnnotation(annotation1, annotationsForClusters[cluster1.ID]))
		assert.True(t, containsAnnotation(annotation2, annotationsForClusters[cluster1.ID]))
		assert.True(t, containsAnnotation(annotation1, annotationsForClusters[cluster2.ID]))
		assert.True(t, containsAnnotation(annotation2, annotationsForClusters[cluster2.ID]))
	})
}

func TestAnnotations_Installation(t *testing.T) {
	logger := testlib.MakeLogger(t)
	sqlStore := MakeTestSQLStore(t, logger)
	defer CloseConnection(t, sqlStore)

	annotation1 := model.Annotation{Name: "annotation1"}
	annotation2 := model.Annotation{Name: "annotation2"}

	err := sqlStore.CreateAnnotation(&annotation1)
	require.NoError(t, err)
	err = sqlStore.CreateAnnotation(&annotation2)
	require.NoError(t, err)

	annotations := []*model.Annotation{&annotation1, &annotation2}

	installation1 := model.Installation{}
	err = sqlStore.createInstallation(sqlStore.db, &installation1)
	require.NoError(t, err)

	_, err = sqlStore.CreateInstallationAnnotations(installation1.ID, annotations)
	require.NoError(t, err)

	t.Run("get annotations for installation", func(t *testing.T) {
		annotationsForInstallation, err := sqlStore.GetAnnotationsForInstallation(installation1.ID)
		require.NoError(t, err)
		assert.Equal(t, len(annotations), len(annotationsForInstallation))
		assert.True(t, containsAnnotation(annotation1, annotationsForInstallation))
		assert.True(t, containsAnnotation(annotation2, annotationsForInstallation))
	})

	t.Run("fail to assign the same annotation to the installation twice", func(t *testing.T) {
		_, err = sqlStore.CreateInstallationAnnotations(installation1.ID, annotations)
		require.Error(t, err)
		assert.Contains(t, strings.ToLower(err.Error()), "unique constraint") // Make sure error comes from DB
	})

	installation2 := model.Installation{DNS: "dns2.com"}
	err = sqlStore.CreateInstallation(&installation2, annotations)
	require.NoError(t, err)

	t.Run("get annotations for installation2", func(t *testing.T) {
		annotationsForInstallation, err := sqlStore.GetAnnotationsForInstallation(installation2.ID)
		require.NoError(t, err)
		assert.Equal(t, len(annotations), len(annotationsForInstallation))
		assert.True(t, containsAnnotation(annotation1, annotationsForInstallation))
		assert.True(t, containsAnnotation(annotation2, annotationsForInstallation))
	})

	t.Run("get annotations for installations", func(t *testing.T) {
		annotationsForInstallations, err := sqlStore.GetAnnotationsForInstallations(&model.InstallationFilter{PerPage: model.AllPerPage})
		require.NoError(t, err)
		assert.Equal(t, 2, len(annotationsForInstallations))
		assert.True(t, containsAnnotation(annotation1, annotationsForInstallations[installation1.ID]))
		assert.True(t, containsAnnotation(annotation2, annotationsForInstallations[installation1.ID]))
		assert.True(t, containsAnnotation(annotation1, annotationsForInstallations[installation2.ID]))
		assert.True(t, containsAnnotation(annotation2, annotationsForInstallations[installation2.ID]))
	})
}

func containsAnnotation(annotation model.Annotation, annotations []*model.Annotation) bool {
	for _, a := range annotations {
		if *a == annotation {
			return true
		}
	}
	return false
}
