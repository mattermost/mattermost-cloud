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
		err2 := sqlStore.CreateAnnotation(&model.Annotation{Name: annotation1.Name})
		require.Error(t, err2)
		assert.Contains(t, strings.ToLower(err2.Error()), "unique constraint") // Make sure error comes from DB
	})

	t.Run("get annotation by name", func(t *testing.T) {
		annotation, err2 := sqlStore.GetAnnotationByName(annotation1.Name)
		require.NoError(t, err2)
		assert.Equal(t, &annotation1, annotation)
	})

	t.Run("get unknown annotation", func(t *testing.T) {
		annotation, err2 := sqlStore.GetAnnotationByName("unknown")
		require.NoError(t, err2)
		assert.Nil(t, annotation)
	})

	annotations := []*model.Annotation{&annotation1, &annotation2}

	cluster1 := model.Cluster{}
	err = sqlStore.createCluster(sqlStore.db, &cluster1)
	require.NoError(t, err)

	_, err = sqlStore.CreateClusterAnnotations(cluster1.ID, annotations)
	require.NoError(t, err)

	t.Run("get annotations for cluster", func(t *testing.T) {
		annotationsForCluster, err2 := sqlStore.GetAnnotationsForCluster(cluster1.ID)
		require.NoError(t, err2)
		assert.Equal(t, len(annotations), len(annotationsForCluster))
		assert.True(t, model.ContainsAnnotation(annotationsForCluster, &annotation1))
		assert.True(t, model.ContainsAnnotation(annotationsForCluster, &annotation2))
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
		annotationsForCluster, err2 := sqlStore.GetAnnotationsForCluster(cluster2.ID)
		require.NoError(t, err2)
		assert.Equal(t, len(annotations), len(annotationsForCluster))
		assert.True(t, model.ContainsAnnotation(annotationsForCluster, &annotation1))
		assert.True(t, model.ContainsAnnotation(annotationsForCluster, &annotation2))
	})

	t.Run("get annotations for clusters", func(t *testing.T) {
		annotationsForClusters, err2 := sqlStore.GetAnnotationsForClusters(&model.ClusterFilter{Paging: model.AllPagesNotDeleted()})
		require.NoError(t, err2)
		assert.Equal(t, 2, len(annotationsForClusters))
		assert.True(t, model.ContainsAnnotation(annotationsForClusters[cluster1.ID], &annotation1))
		assert.True(t, model.ContainsAnnotation(annotationsForClusters[cluster1.ID], &annotation2))
		assert.True(t, model.ContainsAnnotation(annotationsForClusters[cluster2.ID], &annotation1))
		assert.True(t, model.ContainsAnnotation(annotationsForClusters[cluster2.ID], &annotation2))
	})

	t.Run("delete cluster annotation", func(t *testing.T) {
		err2 := sqlStore.DeleteClusterAnnotation(cluster1.ID, annotation1.Name)
		require.NoError(t, err2)
		annotationsForCluster, err3 := sqlStore.GetAnnotationsForCluster(cluster1.ID)
		require.NoError(t, err3)
		assert.Equal(t, 1, len(annotationsForCluster))

		t.Run("do not fail when deleting cluster annotation twice", func(t *testing.T) {
			err4 := sqlStore.DeleteClusterAnnotation(cluster1.ID, annotation1.Name)
			require.NoError(t, err4)
		})
	})

	t.Run("delete unknown annotation", func(t *testing.T) {
		err = sqlStore.DeleteClusterAnnotation(cluster1.ID, "unknown-annotation")
		require.NoError(t, err)
	})

	t.Run("fail to delete annotation if present on installation scheduled on the cluster", func(t *testing.T) {
		installation1 := model.Installation{}
		err2 := sqlStore.createInstallation(sqlStore.db, &installation1)
		require.NoError(t, err2)

		clusterInstallation := &model.ClusterInstallation{
			ClusterID:      cluster1.ID,
			InstallationID: installation1.ID,
		}
		err3 := sqlStore.CreateClusterInstallation(clusterInstallation)
		require.NoError(t, err3)

		_, err4 := sqlStore.CreateInstallationAnnotations(installation1.ID, []*model.Annotation{&annotation2})
		require.NoError(t, err4)

		err5 := sqlStore.DeleteClusterAnnotation(cluster1.ID, annotation2.Name)
		require.Error(t, err5)
		assert.Equal(t, ErrClusterAnnotationUsedByInstallation, err5)
	})

	newAnnotations := []*model.Annotation{
		{Name: "new-annotation1"},
		{Name: "new-annotation2"},
	}

	t.Run("correctly create new cluster annotations", func(t *testing.T) {
		_, err2 := sqlStore.CreateClusterAnnotations(cluster2.ID, newAnnotations)
		require.NoError(t, err2)
		annotationsForInstallation, err3 := sqlStore.GetAnnotationsForCluster(cluster2.ID)
		require.NoError(t, err3)
		assert.Equal(t, len(annotations)+len(newAnnotations), len(annotationsForInstallation))
	})
}

func TestAnnotations_Installation(t *testing.T) {
	logger := testlib.MakeLogger(t)
	sqlStore := MakeTestSQLStore(t, logger)
	defer CloseConnection(t, sqlStore)

	annotation1 := model.Annotation{Name: "annotation1"}
	annotation2 := model.Annotation{Name: "annotation2"}

	errCreate1 := sqlStore.CreateAnnotation(&annotation1)
	require.NoError(t, errCreate1)
	errCreate2 := sqlStore.CreateAnnotation(&annotation2)
	require.NoError(t, errCreate2)

	annotations := []*model.Annotation{&annotation1, &annotation2}

	installation1 := model.Installation{Name: "test"}
	err4 := sqlStore.createInstallation(sqlStore.db, &installation1)
	require.NoError(t, err4)

	_, err5 := sqlStore.CreateInstallationAnnotations(installation1.ID, annotations)
	require.NoError(t, err5)

	t.Run("get annotations for installation", func(t *testing.T) {
		annotationsForInstallation, err2 := sqlStore.GetAnnotationsForInstallation(installation1.ID)
		require.NoError(t, err2)
		assert.Equal(t, len(annotations), len(annotationsForInstallation))
		assert.True(t, model.ContainsAnnotation(annotationsForInstallation, &annotation1))
		assert.True(t, model.ContainsAnnotation(annotationsForInstallation, &annotation2))
	})

	t.Run("fail to assign the same annotation to the installation twice", func(t *testing.T) {
		_, err2 := sqlStore.CreateInstallationAnnotations(installation1.ID, annotations)
		require.Error(t, err2)
		assert.Contains(t, strings.ToLower(err2.Error()), "unique constraint") // Make sure error comes from DB
	})

	installation2 := model.Installation{Name: "test2"}
	errCreate3 := sqlStore.CreateInstallation(&installation2, annotations, nil)
	require.NoError(t, errCreate3)

	t.Run("get annotations for installation2", func(t *testing.T) {
		annotationsForInstallation, err2 := sqlStore.GetAnnotationsForInstallation(installation2.ID)
		require.NoError(t, err2)
		assert.Equal(t, len(annotations), len(annotationsForInstallation))
		assert.True(t, model.ContainsAnnotation(annotationsForInstallation, &annotation1))
		assert.True(t, model.ContainsAnnotation(annotationsForInstallation, &annotation2))
	})

	t.Run("get annotations for installations", func(t *testing.T) {
		annotationsForInstallations, err2 := sqlStore.GetAnnotationsForInstallations(&model.InstallationFilter{Paging: model.AllPagesNotDeleted()})
		require.NoError(t, err2)
		assert.Equal(t, 2, len(annotationsForInstallations))
		assert.True(t, model.ContainsAnnotation(annotationsForInstallations[installation1.ID], &annotation1))
		assert.True(t, model.ContainsAnnotation(annotationsForInstallations[installation1.ID], &annotation2))
		assert.True(t, model.ContainsAnnotation(annotationsForInstallations[installation2.ID], &annotation1))
		assert.True(t, model.ContainsAnnotation(annotationsForInstallations[installation2.ID], &annotation2))
	})

	t.Run("delete installation annotation", func(t *testing.T) {
		errCreate3 = sqlStore.DeleteInstallationAnnotation(installation1.ID, annotation1.Name)
		require.NoError(t, errCreate3)
		annotationsForInstallation, err2 := sqlStore.GetAnnotationsForInstallation(installation1.ID)
		require.NoError(t, err2)
		assert.Equal(t, 1, len(annotationsForInstallation))

		t.Run("do not fail when deleting installation annotation twice", func(t *testing.T) {
			err3 := sqlStore.DeleteInstallationAnnotation(installation1.ID, annotation1.Name)
			require.NoError(t, err3)
		})
	})

	t.Run("delete unknown annotation", func(t *testing.T) {
		errCreate3 = sqlStore.DeleteClusterAnnotation(installation1.ID, "unknown-annotation")
		require.NoError(t, errCreate3)
	})

	t.Run("fail to create annotation if not present on cluster on which installation is scheduled", func(t *testing.T) {
		cluster := model.Cluster{}
		errCreate3 = sqlStore.CreateCluster(&cluster, []*model.Annotation{&annotation1})
		require.NoError(t, errCreate3)
		installation := model.Installation{}
		errCreate3 = sqlStore.CreateInstallation(&installation, nil, nil)
		require.NoError(t, errCreate3)

		clusterInstallation := &model.ClusterInstallation{
			ClusterID:      cluster.ID,
			InstallationID: installation.ID,
		}
		errCreate3 = sqlStore.CreateClusterInstallation(clusterInstallation)
		require.NoError(t, errCreate3)

		_, errCreate3 = sqlStore.CreateInstallationAnnotations(installation.ID, []*model.Annotation{&annotation1})
		require.NoError(t, errCreate3)

		_, errCreate3 = sqlStore.CreateInstallationAnnotations(installation.ID, []*model.Annotation{&annotation2})
		require.Error(t, errCreate3)
		assert.Equal(t, ErrInstallationAnnotationDoNotMatchClusters, errCreate3)
	})

	newAnnotations := []*model.Annotation{
		{Name: "new-annotation1"},
		{Name: "new-annotation2"},
	}

	t.Run("correctly create new installation annotations", func(t *testing.T) {
		_, errCreate3 = sqlStore.CreateInstallationAnnotations(installation2.ID, newAnnotations)
		require.NoError(t, errCreate3)
		annotationsForInstallation, err := sqlStore.GetAnnotationsForInstallation(installation2.ID)
		require.NoError(t, err)
		assert.Equal(t, len(annotations)+len(newAnnotations), len(annotationsForInstallation))
	})
}

func TestAnnotations_GetAnnotationsByName(t *testing.T) {
	logger := testlib.MakeLogger(t)
	sqlStore := MakeTestSQLStore(t, logger)
	defer CloseConnection(t, sqlStore)

	annotation1 := model.Annotation{Name: "annotation1"}
	annotation2 := model.Annotation{Name: "annotation2"}

	err := sqlStore.CreateAnnotation(&annotation1)
	require.NoError(t, err)
	err = sqlStore.CreateAnnotation(&annotation2)
	require.NoError(t, err)

	t.Run("get all existing annotations", func(t *testing.T) {
		annotations, err := sqlStore.GetAnnotationsByName([]string{"annotation1", "annotation2"})
		require.NoError(t, err)
		assert.ElementsMatch(t, []*model.Annotation{&annotation1, &annotation2}, annotations)
	})

	t.Run("try getting not existing annotations", func(t *testing.T) {
		annotations, err := sqlStore.GetAnnotationsByName([]string{"none1", "none2"})
		require.NoError(t, err)
		assert.Empty(t, annotations)
	})

	t.Run("try getting existing and not existing annotations", func(t *testing.T) {
		annotations, err := sqlStore.GetAnnotationsByName([]string{"annotation1", "annotation2", "none1", "none2"})
		require.NoError(t, err)
		assert.Equal(t, 2, len(annotations))
		assert.ElementsMatch(t, []*model.Annotation{&annotation1, &annotation2}, annotations)
	})

}

func TestAnnotations_Group(t *testing.T) {
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

	group1 := model.Group{Name: "test"}
	err = sqlStore.CreateGroup(&group1, nil)
	require.NoError(t, err)

	_, err = sqlStore.CreateGroupAnnotations(group1.ID, annotations)
	require.NoError(t, err)

	t.Run("get annotations for group", func(t *testing.T) {
		annotationsForGroup, err2 := sqlStore.getAnnotationsForGroup(sqlStore.db, group1.ID)
		require.NoError(t, err2)
		assert.Equal(t, len(annotations), len(annotationsForGroup))
		assert.True(t, model.ContainsAnnotation(annotationsForGroup, &annotation1))
		assert.True(t, model.ContainsAnnotation(annotationsForGroup, &annotation2))
	})

	t.Run("fail to assign the same annotation to the group twice", func(t *testing.T) {
		_, err = sqlStore.CreateGroupAnnotations(group1.ID, annotations)
		require.Error(t, err)
		assert.Contains(t, strings.ToLower(err.Error()), "unique constraint") // Make sure error comes from DB
	})

	group2 := model.Group{Name: "test2"}
	err = sqlStore.CreateGroup(&group2, annotations)
	require.NoError(t, err)

	t.Run("get annotations for group2", func(t *testing.T) {
		annotationsForGroup, err2 := sqlStore.getAnnotationsForGroup(sqlStore.db, group2.ID)
		require.NoError(t, err2)
		assert.Equal(t, len(annotations), len(annotationsForGroup))
		assert.True(t, model.ContainsAnnotation(annotationsForGroup, &annotation1))
		assert.True(t, model.ContainsAnnotation(annotationsForGroup, &annotation2))
	})

	t.Run("get annotations for groups", func(t *testing.T) {
		annotationsForGroups, err2 := sqlStore.getAnnotationsForGroups([]string{group1.ID, group2.ID})
		require.NoError(t, err2)
		assert.Equal(t, 2, len(annotationsForGroups))
		assert.True(t, model.ContainsAnnotation(annotationsForGroups[group1.ID], &annotation1))
		assert.True(t, model.ContainsAnnotation(annotationsForGroups[group1.ID], &annotation2))
		assert.True(t, model.ContainsAnnotation(annotationsForGroups[group2.ID], &annotation1))
		assert.True(t, model.ContainsAnnotation(annotationsForGroups[group2.ID], &annotation2))
	})

	t.Run("delete Group annotation", func(t *testing.T) {
		err = sqlStore.DeleteGroupAnnotation(group1.ID, annotation1.Name)
		require.NoError(t, err)
		annotationsForGroup, err2 := sqlStore.getAnnotationsForGroup(sqlStore.db, group1.ID)
		require.NoError(t, err2)
		assert.Equal(t, 1, len(annotationsForGroup))

		t.Run("do not fail when deleting Group annotation twice", func(t *testing.T) {
			err2 = sqlStore.DeleteGroupAnnotation(group1.ID, annotation1.Name)
			require.NoError(t, err2)
		})
	})

	t.Run("delete unknown annotation", func(t *testing.T) {
		err = sqlStore.DeleteGroupAnnotation(group1.ID, "unknown-annotation")
		require.NoError(t, err)
	})

	newAnnotations := []*model.Annotation{
		{Name: "new-annotation1"},
		{Name: "new-annotation2"},
	}

	t.Run("correctly create new group annotations", func(t *testing.T) {
		_, err = sqlStore.CreateGroupAnnotations(group2.ID, newAnnotations)
		require.NoError(t, err)
		annotationsForGroup, err := sqlStore.getAnnotationsForGroup(sqlStore.db, group2.ID)
		require.NoError(t, err)
		assert.Equal(t, len(annotations)+len(newAnnotations), len(annotationsForGroup))
	})
}
