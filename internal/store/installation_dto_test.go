// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.
//

package store

import (
	"testing"
	"time"

	"github.com/mattermost/mattermost-cloud/internal/testlib"
	"github.com/mattermost/mattermost-cloud/model"
	mmv1alpha1 "github.com/mattermost/mattermost-operator/apis/mattermost/v1alpha1"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestInstallationDTOs(t *testing.T) {
	logger := testlib.MakeLogger(t)
	sqlStore := MakeTestSQLStore(t, logger)
	defer CloseConnection(t, sqlStore)

	t.Run("get unknown installation DTO", func(t *testing.T) {
		cluster, err := sqlStore.GetInstallationDTO("unknown", false, false)
		require.NoError(t, err)
		require.Nil(t, cluster)
	})

	annotation1 := model.Annotation{Name: "annotation1"}
	annotation2 := model.Annotation{Name: "annotation2"}

	// Create only one annotation beforehand to test both get by name and create.
	err := sqlStore.CreateAnnotation(&annotation1)
	require.NoError(t, err)
	annotations := []*model.Annotation{&annotation1, &annotation2}

	singleTenantDBConfig := &model.SingleTenantDatabaseConfig{
		PrimaryInstanceType: "db.r5.large",
		ReplicaInstanceType: "db.r5.xlarge",
		ReplicasCount:       11,
	}

	groupID1 := model.NewID()

	installation1 := &model.Installation{
		Name:                       "test",
		OwnerID:                    "owner1",
		Version:                    "version",
		Database:                   model.InstallationDatabaseMysqlOperator,
		Filestore:                  model.InstallationFilestoreMinioOperator,
		Size:                       mmv1alpha1.Size100String,
		Affinity:                   model.InstallationAffinityIsolated,
		GroupID:                    &groupID1,
		State:                      model.InstallationStateCreationRequested,
		SingleTenantDatabaseConfig: singleTenantDBConfig,
	}

	installation2 := &model.Installation{
		Name:      "test2",
		OwnerID:   "owner1",
		Version:   "version2",
		Image:     "custom-image",
		Database:  model.InstallationDatabaseMysqlOperator,
		Filestore: model.InstallationFilestoreMinioOperator,
		Size:      mmv1alpha1.Size100String,
		Affinity:  model.InstallationAffinityIsolated,
		GroupID:   &groupID1,
		State:     model.InstallationStateStable,
	}

	dnsRecords1 := fixDNSRecords(1)
	err = sqlStore.CreateInstallation(installation1, annotations, dnsRecords1)
	require.NoError(t, err)

	// Wait 1ms to guarantee the order.
	time.Sleep(1 * time.Millisecond)

	dnsRecords2 := fixDNSRecords(2)
	err = sqlStore.CreateInstallation(installation2, nil, dnsRecords2)
	require.NoError(t, err)

	clusterInstallation1 := &model.ClusterInstallation{
		ClusterID:      model.NewID(),
		InstallationID: installation1.ID,
	}

	clusterInstallation2 := &model.ClusterInstallation{
		ClusterID:      model.NewID(),
		InstallationID: installation1.ID,
	}

	clusterInstallation3 := &model.ClusterInstallation{
		ClusterID:      model.NewID(),
		InstallationID: installation2.ID,
	}

	err = sqlStore.CreateClusterInstallation(clusterInstallation1)
	require.NoError(t, err)
	err = sqlStore.CreateClusterInstallation(clusterInstallation2)
	require.NoError(t, err)
	err = sqlStore.CreateClusterInstallation(clusterInstallation3)
	require.NoError(t, err)

	t.Run("get installation DTO", func(t *testing.T) {
		installationDTO, err := sqlStore.GetInstallationDTO(installation1.ID, false, false)
		require.NoError(t, err)
		assert.Equal(t, installation1, installationDTO.Installation)
		assert.Equal(t, len(annotations), len(installationDTO.Annotations))
		assert.Equal(t, annotations, model.SortAnnotations(installationDTO.Annotations))
		assert.ElementsMatch(t, []string{clusterInstallation1.ClusterID}, installationDTO.ClusterIDs)
	})

	t.Run("get installation DTOs", func(t *testing.T) {
		installationDTOs, err := sqlStore.GetInstallationDTOs(
			&model.InstallationFilter{Paging: model.AllPagesWithDeleted()},
			false,
			false,
		)
		require.NoError(t, err)
		assert.Equal(t, 2, len(installationDTOs))
		for _, i := range installationDTOs {
			model.SortAnnotations(i.Annotations)
		}
		assert.ElementsMatch(t, []*model.InstallationDTO{
			installation1.ToDTO(annotations, dnsRecords1, []*string{&clusterInstallation1.ClusterID, &clusterInstallation2.ClusterID}),
			installation2.ToDTO(nil, dnsRecords2, []*string{&clusterInstallation3.ClusterID}),
		}, installationDTOs)
	})
}
