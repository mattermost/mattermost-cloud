// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.
//

package api_test

import (
	"bytes"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gorilla/mux"
	"github.com/mattermost/mattermost-cloud/internal/api"
	"github.com/mattermost/mattermost-cloud/internal/store"
	"github.com/mattermost/mattermost-cloud/internal/testlib"
	"github.com/mattermost/mattermost-cloud/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGetClusterInstallations(t *testing.T) {
	logger := testlib.MakeLogger(t)
	sqlStore := store.MakeTestSQLStore(t, logger)

	router := mux.NewRouter()
	api.Register(router, &api.Context{
		Store:      sqlStore,
		Supervisor: &mockSupervisor{},
		Logger:     logger,
	})
	ts := httptest.NewServer(router)
	defer ts.Close()

	client := model.NewClient(ts.URL)

	t.Run("unknown cluster installation", func(t *testing.T) {
		clusterInstallation, err := client.GetClusterInstallation(model.NewID())
		require.NoError(t, err)
		require.Nil(t, clusterInstallation)
	})

	t.Run("no cluster installations", func(t *testing.T) {
		clusterInstallations, err := client.GetClusterInstallations(&model.GetClusterInstallationsRequest{
			Paging: model.AllPagesWithDeleted(),
		})
		require.NoError(t, err)
		require.Empty(t, clusterInstallations)
	})

	t.Run("parameter handling", func(t *testing.T) {
		t.Run("invalid page", func(t *testing.T) {
			resp, err := http.Get(fmt.Sprintf("%s/api/cluster_installations?page=invalid&per_page=100", ts.URL))
			require.NoError(t, err)
			require.Equal(t, http.StatusBadRequest, resp.StatusCode)
		})

		t.Run("invalid perPage", func(t *testing.T) {
			resp, err := http.Get(fmt.Sprintf("%s/api/cluster_installations?page=0&per_page=invalid", ts.URL))
			require.NoError(t, err)
			require.Equal(t, http.StatusBadRequest, resp.StatusCode)
		})

		t.Run("no paging parameters", func(t *testing.T) {
			resp, err := http.Get(fmt.Sprintf("%s/api/cluster_installations", ts.URL))
			require.NoError(t, err)
			require.Equal(t, http.StatusOK, resp.StatusCode)
		})

		t.Run("missing page", func(t *testing.T) {
			resp, err := http.Get(fmt.Sprintf("%s/api/cluster_installations?per_page=100", ts.URL))
			require.NoError(t, err)
			require.Equal(t, http.StatusOK, resp.StatusCode)
		})

		t.Run("missing perPage", func(t *testing.T) {
			resp, err := http.Get(fmt.Sprintf("%s/api/cluster_installations?page=1", ts.URL))
			require.NoError(t, err)
			require.Equal(t, http.StatusOK, resp.StatusCode)
		})
	})

	t.Run("results", func(t *testing.T) {
		clusterID1 := model.NewID()
		clusterID2 := model.NewID()
		installationID1 := model.NewID()
		installationID2 := model.NewID()

		clusterInstallation1 := &model.ClusterInstallation{
			ClusterID:      clusterID1,
			InstallationID: installationID1,
			Namespace:      "cluster installation 1",
			State:          model.ClusterInstallationStateCreationRequested,
		}
		err := sqlStore.CreateClusterInstallation(clusterInstallation1)
		require.NoError(t, err)

		time.Sleep(1 * time.Millisecond)

		clusterInstallation2 := &model.ClusterInstallation{
			ClusterID:      clusterID1,
			InstallationID: installationID2,
			Namespace:      "cluster installation 2",
			State:          model.ClusterInstallationStateStable,
		}
		err = sqlStore.CreateClusterInstallation(clusterInstallation2)
		require.NoError(t, err)

		time.Sleep(1 * time.Millisecond)

		clusterInstallation3 := &model.ClusterInstallation{
			ClusterID:      clusterID2,
			InstallationID: installationID1,
			Namespace:      "cluster installation 3",
			State:          model.ClusterInstallationStateDeletionRequested,
		}
		err = sqlStore.CreateClusterInstallation(clusterInstallation3)
		require.NoError(t, err)

		time.Sleep(1 * time.Millisecond)

		clusterInstallation4 := &model.ClusterInstallation{
			ClusterID:      clusterID2,
			InstallationID: installationID2,
			Namespace:      "cluster installation 4",
			State:          model.ClusterInstallationStateDeleted,
		}
		err = sqlStore.CreateClusterInstallation(clusterInstallation4)
		require.NoError(t, err)
		err = sqlStore.DeleteClusterInstallation(clusterInstallation4.ID)
		require.NoError(t, err)
		clusterInstallation4, err = client.GetClusterInstallation(clusterInstallation4.ID)
		require.NoError(t, err)

		t.Run("get cluster installation", func(t *testing.T) {
			t.Run("cluster installation 1", func(t *testing.T) {
				clusterInstallation, err := client.GetClusterInstallation(clusterInstallation1.ID)
				require.NoError(t, err)
				require.Equal(t, clusterInstallation1, clusterInstallation)
			})

			t.Run("get deleted cluster installation", func(t *testing.T) {
				clusterInstallation, err := client.GetClusterInstallation(clusterInstallation4.ID)
				require.NoError(t, err)
				require.Equal(t, clusterInstallation4, clusterInstallation)
			})
		})

		t.Run("get cluster installations", func(t *testing.T) {
			testCases := []struct {
				Description                    string
				GetClusterInstallationsRequest *model.GetClusterInstallationsRequest
				Expected                       []*model.ClusterInstallation
			}{
				{
					"page 0, perPage 2, exclude deleted",
					&model.GetClusterInstallationsRequest{
						Paging: model.Paging{
							Page:           0,
							PerPage:        2,
							IncludeDeleted: false,
						},
					},
					[]*model.ClusterInstallation{clusterInstallation1, clusterInstallation2},
				},

				{
					"page 1, perPage 2, exclude deleted",
					&model.GetClusterInstallationsRequest{
						Paging: model.Paging{
							Page:           1,
							PerPage:        2,
							IncludeDeleted: false,
						},
					},
					[]*model.ClusterInstallation{clusterInstallation3},
				},

				{
					"page 0, perPage 2, include deleted",
					&model.GetClusterInstallationsRequest{
						Paging: model.Paging{
							Page:           0,
							PerPage:        2,
							IncludeDeleted: true,
						},
					},
					[]*model.ClusterInstallation{clusterInstallation1, clusterInstallation2},
				},

				{
					"page 1, perPage 2, include deleted",
					&model.GetClusterInstallationsRequest{
						Paging: model.Paging{
							Page:           1,
							PerPage:        2,
							IncludeDeleted: true,
						},
					},
					[]*model.ClusterInstallation{clusterInstallation3, clusterInstallation4},
				},

				{
					"filter by cluster",
					&model.GetClusterInstallationsRequest{
						ClusterID: clusterID1,
						Paging:    model.AllPagesNotDeleted(),
					},
					[]*model.ClusterInstallation{clusterInstallation1, clusterInstallation2},
				},

				{
					"filter by installation",
					&model.GetClusterInstallationsRequest{
						InstallationID: installationID1,
						Paging:         model.AllPagesNotDeleted(),
					},
					[]*model.ClusterInstallation{clusterInstallation1, clusterInstallation3},
				},

				{
					"filter by cluster + installation",
					&model.GetClusterInstallationsRequest{
						ClusterID:      clusterID2,
						InstallationID: installationID2,
						Paging:         model.AllPagesNotDeleted(),
					},
					[]*model.ClusterInstallation{},
				},

				{
					"filter by cluster + installation, include deleted",
					&model.GetClusterInstallationsRequest{
						ClusterID:      clusterID2,
						InstallationID: installationID2,
						Paging:         model.AllPagesWithDeleted(),
					},
					[]*model.ClusterInstallation{clusterInstallation4},
				},
			}

			for _, testCase := range testCases {
				t.Run(testCase.Description, func(t *testing.T) {
					clusterInstallations, err := client.GetClusterInstallations(testCase.GetClusterInstallationsRequest)
					require.NoError(t, err)
					require.Equal(t, testCase.Expected, clusterInstallations)
				})
			}
		})
	})
}

func TestGetClusterInstallationConfig(t *testing.T) {
	logger := testlib.MakeLogger(t)
	sqlStore := store.MakeTestSQLStore(t, logger)

	router := mux.NewRouter()
	api.Register(router, &api.Context{
		Store:       sqlStore,
		Supervisor:  &mockSupervisor{},
		Provisioner: &mockProvisioner{},
		Logger:      logger,
	})
	ts := httptest.NewServer(router)
	defer ts.Close()

	client := model.NewClient(ts.URL)

	cluster := &model.Cluster{}
	err := sqlStore.CreateCluster(cluster, nil)
	require.NoError(t, err)

	clusterInstallation1 := &model.ClusterInstallation{
		ClusterID:      cluster.ID,
		InstallationID: model.NewID(),
	}
	err = sqlStore.CreateClusterInstallation(clusterInstallation1)
	require.NoError(t, err)

	t.Run("unknown cluster installation", func(t *testing.T) {
		config, err := client.GetClusterInstallationConfig(model.NewID())
		require.NoError(t, err)
		require.Nil(t, config)
	})

	t.Run("success", func(t *testing.T) {
		config, err := client.GetClusterInstallationConfig(clusterInstallation1.ID)
		require.NoError(t, err)
		require.Contains(t, config, "ServiceSettings")
	})

	t.Run("cluster installation deleted", func(t *testing.T) {
		err = sqlStore.DeleteClusterInstallation(clusterInstallation1.ID)
		require.NoError(t, err)

		config, err := client.GetClusterInstallationConfig(clusterInstallation1.ID)
		require.Error(t, err)
		require.Nil(t, config)
	})
}

func TestSetClusterInstallationConfig(t *testing.T) {
	logger := testlib.MakeLogger(t)
	sqlStore := store.MakeTestSQLStore(t, logger)

	router := mux.NewRouter()
	api.Register(router, &api.Context{
		Store:       sqlStore,
		Supervisor:  &mockSupervisor{},
		Provisioner: &mockProvisioner{},
		Logger:      logger,
	})
	ts := httptest.NewServer(router)
	defer ts.Close()

	client := model.NewClient(ts.URL)

	cluster := &model.Cluster{}
	err := sqlStore.CreateCluster(cluster, nil)
	require.NoError(t, err)

	clusterInstallation1 := &model.ClusterInstallation{
		ClusterID:      cluster.ID,
		InstallationID: model.NewID(),
	}
	err = sqlStore.CreateClusterInstallation(clusterInstallation1)
	require.NoError(t, err)

	config := map[string]interface{}{"ServiceSettings": map[string]interface{}{"SiteURL": "test.example.com"}}

	t.Run("unknown cluster installation", func(t *testing.T) {
		err := client.SetClusterInstallationConfig(model.NewID(), config)
		require.EqualError(t, err, "failed with status code 404")
	})

	t.Run("invalid payload", func(t *testing.T) {
		httpRequest, err := http.NewRequest(http.MethodPut, fmt.Sprintf("%s/api/cluster_installation/%s/config", ts.URL, clusterInstallation1.ID), bytes.NewReader([]byte("invalid")))
		require.NoError(t, err)

		resp, err := http.DefaultClient.Do(httpRequest)
		require.NoError(t, err)
		require.Equal(t, http.StatusBadRequest, resp.StatusCode)
	})

	t.Run("empty payload", func(t *testing.T) {
		httpRequest, err := http.NewRequest(http.MethodPut, fmt.Sprintf("%s/api/cluster_installation/%s/config", ts.URL, clusterInstallation1.ID), bytes.NewReader([]byte("")))
		require.NoError(t, err)

		resp, err := http.DefaultClient.Do(httpRequest)
		require.NoError(t, err)
		require.Equal(t, http.StatusOK, resp.StatusCode)
	})

	t.Run("while api-security-locked", func(t *testing.T) {
		err = sqlStore.LockClusterInstallationAPI(clusterInstallation1.ID)
		require.NoError(t, err)

		err := client.SetClusterInstallationConfig(clusterInstallation1.ID, config)
		require.EqualError(t, err, "failed with status code 403")

		err = sqlStore.UnlockClusterInstallationAPI(clusterInstallation1.ID)
		require.NoError(t, err)
	})

	t.Run("success", func(t *testing.T) {
		err := client.SetClusterInstallationConfig(clusterInstallation1.ID, config)
		require.NoError(t, err)
	})

	t.Run("cluster installation deleted", func(t *testing.T) {
		err = sqlStore.DeleteClusterInstallation(clusterInstallation1.ID)
		require.NoError(t, err)

		err := client.SetClusterInstallationConfig(clusterInstallation1.ID, config)
		require.Error(t, err)
	})
}

func TestRunClusterInstallationExecCommand(t *testing.T) {
	logger := testlib.MakeLogger(t)
	sqlStore := store.MakeTestSQLStore(t, logger)

	mProvisioner := &mockProvisioner{}

	router := mux.NewRouter()
	api.Register(router, &api.Context{
		Store:       sqlStore,
		Supervisor:  &mockSupervisor{},
		Provisioner: mProvisioner,
		Logger:      logger,
	})
	ts := httptest.NewServer(router)
	defer ts.Close()

	client := model.NewClient(ts.URL)

	cluster := &model.Cluster{}
	err := sqlStore.CreateCluster(cluster, nil)
	require.NoError(t, err)

	clusterInstallation1 := &model.ClusterInstallation{
		ClusterID:      cluster.ID,
		InstallationID: model.NewID(),
	}
	err = sqlStore.CreateClusterInstallation(clusterInstallation1)
	require.NoError(t, err)

	command := "mmctl"
	subcommand := model.ClusterInstallationMattermostCLISubcommand{"get", "version"}

	t.Run("unknown cluster installation", func(t *testing.T) {
		bytes, err := client.ExecClusterInstallationCLI(model.NewID(), command, subcommand)
		require.EqualError(t, err, "failed with status code 404")
		require.Empty(t, bytes)
	})

	t.Run("success", func(t *testing.T) {
		bytes, err := client.ExecClusterInstallationCLI(clusterInstallation1.ID, command, subcommand)
		require.NoError(t, err)
		require.NotEmpty(t, bytes)
	})

	t.Run("invalid command", func(t *testing.T) {
		bytes, err := client.ExecClusterInstallationCLI(clusterInstallation1.ID, "invalid-command", subcommand)
		require.Error(t, err)
		require.Empty(t, bytes)
	})

	t.Run("invalid payload", func(t *testing.T) {
		httpRequest, err := http.NewRequest(http.MethodPost, fmt.Sprintf("%s/api/cluster_installation/%s/exec/mmctl", ts.URL, clusterInstallation1.ID), bytes.NewReader([]byte("invalid")))
		require.NoError(t, err)

		resp, err := http.DefaultClient.Do(httpRequest)
		require.NoError(t, err)
		require.Equal(t, http.StatusBadRequest, resp.StatusCode)
	})

	t.Run("while api-security-locked", func(t *testing.T) {
		err = sqlStore.LockClusterInstallationAPI(clusterInstallation1.ID)
		require.NoError(t, err)

		bytes, err := client.ExecClusterInstallationCLI(clusterInstallation1.ID, command, subcommand)
		require.EqualError(t, err, "failed with status code 403")
		require.Empty(t, bytes)

		err = sqlStore.UnlockClusterInstallationAPI(clusterInstallation1.ID)
		require.NoError(t, err)
	})

	t.Run("non-zero exit command", func(t *testing.T) {
		mProvisioner.CommandError = errors.New("encountered a command error")

		httpRequest, err := http.NewRequest(http.MethodPost, fmt.Sprintf("%s/api/cluster_installation/%s/exec/mmctl", ts.URL, clusterInstallation1.ID), bytes.NewReader([]byte("[]")))
		require.NoError(t, err)

		resp, err := http.DefaultClient.Do(httpRequest)
		require.NoError(t, err)
		require.Equal(t, http.StatusInternalServerError, resp.StatusCode)
	})

	t.Run("cluster installation deleted", func(t *testing.T) {
		err = sqlStore.DeleteClusterInstallation(clusterInstallation1.ID)
		require.NoError(t, err)

		bytes, err := client.ExecClusterInstallationCLI(clusterInstallation1.ID, command, subcommand)
		require.Error(t, err)
		require.Empty(t, bytes)
	})
}

func TestRunClusterInstallationMattermostCLI(t *testing.T) {
	logger := testlib.MakeLogger(t)
	sqlStore := store.MakeTestSQLStore(t, logger)

	mProvisioner := &mockProvisioner{}

	router := mux.NewRouter()
	api.Register(router, &api.Context{
		Store:       sqlStore,
		Supervisor:  &mockSupervisor{},
		Provisioner: mProvisioner,
		Logger:      logger,
	})
	ts := httptest.NewServer(router)
	defer ts.Close()

	client := model.NewClient(ts.URL)

	cluster := &model.Cluster{}
	err := sqlStore.CreateCluster(cluster, nil)
	require.NoError(t, err)

	clusterInstallation1 := &model.ClusterInstallation{
		ClusterID:      cluster.ID,
		InstallationID: model.NewID(),
	}
	err = sqlStore.CreateClusterInstallation(clusterInstallation1)
	require.NoError(t, err)

	subcommand := model.ClusterInstallationMattermostCLISubcommand{"get", "version"}

	t.Run("unknown cluster installation", func(t *testing.T) {
		bytes, err := client.RunMattermostCLICommandOnClusterInstallation(model.NewID(), subcommand)
		require.EqualError(t, err, "failed with status code 404")
		require.Empty(t, bytes)
	})

	t.Run("success", func(t *testing.T) {
		bytes, err := client.RunMattermostCLICommandOnClusterInstallation(clusterInstallation1.ID, subcommand)
		require.NoError(t, err)
		require.NotEmpty(t, bytes)
	})

	t.Run("invalid payload", func(t *testing.T) {
		httpRequest, err := http.NewRequest(http.MethodPost, fmt.Sprintf("%s/api/cluster_installation/%s/mattermost_cli", ts.URL, clusterInstallation1.ID), bytes.NewReader([]byte("invalid")))
		require.NoError(t, err)

		resp, err := http.DefaultClient.Do(httpRequest)
		require.NoError(t, err)
		require.Equal(t, http.StatusBadRequest, resp.StatusCode)
	})

	t.Run("while api-security-locked", func(t *testing.T) {
		err = sqlStore.LockClusterInstallationAPI(clusterInstallation1.ID)
		require.NoError(t, err)

		bytes, err := client.RunMattermostCLICommandOnClusterInstallation(clusterInstallation1.ID, subcommand)
		require.EqualError(t, err, "failed with status code 403")
		require.Empty(t, bytes)

		err = sqlStore.UnlockClusterInstallationAPI(clusterInstallation1.ID)
		require.NoError(t, err)
	})

	t.Run("non-zero exit command", func(t *testing.T) {
		mProvisioner.CommandError = errors.New("encountered a command error")

		httpRequest, err := http.NewRequest(http.MethodPost, fmt.Sprintf("%s/api/cluster_installation/%s/mattermost_cli", ts.URL, clusterInstallation1.ID), bytes.NewReader([]byte("[]")))
		require.NoError(t, err)

		resp, err := http.DefaultClient.Do(httpRequest)
		require.NoError(t, err)
		require.Equal(t, http.StatusInternalServerError, resp.StatusCode)
	})

	t.Run("cluster installation deleted", func(t *testing.T) {
		err = sqlStore.DeleteClusterInstallation(clusterInstallation1.ID)
		require.NoError(t, err)

		bytes, err := client.RunMattermostCLICommandOnClusterInstallation(clusterInstallation1.ID, subcommand)
		require.Error(t, err)
		require.Empty(t, bytes)
	})
}

func TestMigrateClusterInstallations(t *testing.T) {
	logger := testlib.MakeLogger(t)
	sqlStore := store.MakeTestSQLStore(t, logger)

	router := mux.NewRouter()
	context := &api.Context{
		Store:      sqlStore,
		Supervisor: &mockSupervisor{},
		Logger:     logger,
	}
	api.Register(router, context)
	ts := httptest.NewServer(router)
	defer ts.Close()

	model.SetDeployOperators(true, true)

	t.Run("invalid payload", func(t *testing.T) {
		resp, err := http.Post(fmt.Sprintf("%s/api/cluster_installations/migrate", ts.URL), "application/json", bytes.NewReader([]byte("invalid")))
		require.NoError(t, err)
		require.Equal(t, http.StatusBadRequest, resp.StatusCode)
	})

	t.Run("empty payload", func(t *testing.T) {
		resp, err := http.Post(fmt.Sprintf("%s/api/cluster_installations/migrate", ts.URL), "application/json", bytes.NewReader([]byte("")))
		require.NoError(t, err)
		require.Equal(t, http.StatusBadRequest, resp.StatusCode)
	})

	client := model.NewClient(ts.URL)
	t.Run("missing source cluster", func(t *testing.T) {
		err := client.MigrateClusterInstallation(&model.MigrateClusterInstallationRequest{SourceClusterID: "", TargetClusterID: "4567"})
		require.EqualError(t, err, "failed with status code 400")
	})

	t.Run("missing target cluster", func(t *testing.T) {
		err := client.MigrateClusterInstallation(&model.MigrateClusterInstallationRequest{SourceClusterID: "12345", TargetClusterID: ""})
		require.EqualError(t, err, "failed with status code 400")
	})

	t.Run("no cluster installation found to migrate", func(t *testing.T) {
		err := client.MigrateClusterInstallation(&model.MigrateClusterInstallationRequest{SourceClusterID: "12345", TargetClusterID: "67899"})
		require.EqualError(t, err, "failed with status code 404")
	})

	// Valid migration test
	installation1, err := client.CreateInstallation(&model.CreateInstallationRequest{
		OwnerID:  "owner1",
		Version:  "version",
		Image:    "custom-image",
		DNS:      "dns1.example.com",
		Affinity: model.InstallationAffinityIsolated,
	})
	require.NoError(t, err)

	time.Sleep(1 * time.Millisecond)

	installation2, err := client.CreateInstallation(&model.CreateInstallationRequest{
		OwnerID:  "owner2",
		Version:  "version",
		Image:    "custom-image",
		DNS:      "dns2.example.com",
		Affinity: model.InstallationAffinityIsolated,
	})
	require.NoError(t, err)
	sourceCluster, err := client.CreateCluster(&model.CreateClusterRequest{
		Provider:    model.ProviderAWS,
		Zones:       []string{"zone"},
		Annotations: []string{"my-annotation"},
	})
	require.NoError(t, err)
	require.NotNil(t, sourceCluster.ID)

	targetCluster, err := client.CreateCluster(&model.CreateClusterRequest{
		Provider:    model.ProviderAWS,
		Zones:       []string{"zone"},
		Annotations: []string{"my-annotation"},
	})
	require.NoError(t, err)
	require.NotNil(t, targetCluster.ID)
	clusterInstallation1 := &model.ClusterInstallation{
		ClusterID:      sourceCluster.ID,
		InstallationID: installation1.ID,
		Namespace:      "namespace_10",
		State:          model.ClusterInstallationStateCreationRequested,
		IsActive:       true,
	}

	time.Sleep(1 * time.Millisecond)

	clusterInstallation2 := &model.ClusterInstallation{
		ClusterID:      sourceCluster.ID,
		InstallationID: installation2.ID,
		Namespace:      "namespace_11",
		State:          model.ClusterInstallationStateCreationRequested,
		IsActive:       true,
	}

	err = sqlStore.CreateClusterInstallation(clusterInstallation1)
	require.NoError(t, err)

	err = sqlStore.CreateClusterInstallation(clusterInstallation2)
	require.NoError(t, err)

	// New Installation to test already migrated CIs
	installation3, err := client.CreateInstallation(&model.CreateInstallationRequest{
		OwnerID:  "owner1",
		Version:  "version",
		Image:    "custom-image",
		DNS:      "dns3.example.com",
		Affinity: model.InstallationAffinityIsolated,
	})
	require.NoError(t, err)

	time.Sleep(1 * time.Millisecond)

	clusterInstallation3 := &model.ClusterInstallation{
		ClusterID:      sourceCluster.ID,
		InstallationID: installation3.ID,
		Namespace:      "namespace_12",
		State:          model.ClusterInstallationStateCreationRequested,
		IsActive:       true,
	}

	t.Run("valid migration test", func(t *testing.T) {
		mcir := &model.MigrateClusterInstallationRequest{SourceClusterID: sourceCluster.ID, TargetClusterID: targetCluster.ID, InstallationID: "", DNSSwitch: true, LockInstallation: true}
		t.Log(mcir)
		err := client.MigrateClusterInstallation(mcir)
		require.NoError(t, err)
		newClusterInstallations, err := sqlStore.GetClusterInstallations(&model.ClusterInstallationFilter{ClusterID: targetCluster.ID, Paging: model.AllPagesNotDeleted()})
		require.NoError(t, err)
		assert.Len(t, newClusterInstallations, 2)
		for _, ci := range newClusterInstallations {
			assert.False(t, ci.IsActive)
			assert.Equal(t, model.ClusterInstallationStateCreationRequested, ci.State)
		}

		// Should only return 1 valid installation for migration
		err = sqlStore.CreateClusterInstallation(clusterInstallation3)
		require.NoError(t, err)
		cis, status := api.GetClusterInstallationsForMigration(context, *mcir)
		assert.Equal(t, status, 0)
		assert.Len(t, cis, 1)

	})
}

func TestMigrateDNS(t *testing.T) {
	logger := testlib.MakeLogger(t)
	sqlStore := store.MakeTestSQLStore(t, logger)

	router := mux.NewRouter()
	api.Register(router, &api.Context{
		Store:      sqlStore,
		Supervisor: &mockSupervisor{},
		Logger:     logger,
	})
	ts := httptest.NewServer(router)
	defer ts.Close()

	model.SetDeployOperators(true, true)

	t.Run("invalid payload", func(t *testing.T) {
		resp, err := http.Post(fmt.Sprintf("%s/api/cluster_installations/migrate/dns", ts.URL), "application/json", bytes.NewReader([]byte("invalid")))
		require.NoError(t, err)
		require.Equal(t, http.StatusBadRequest, resp.StatusCode)
	})

	t.Run("empty payload", func(t *testing.T) {
		resp, err := http.Post(fmt.Sprintf("%s/api/cluster_installations/migrate/dns", ts.URL), "application/json", bytes.NewReader([]byte("")))
		require.NoError(t, err)
		require.Equal(t, http.StatusBadRequest, resp.StatusCode)
	})

	client := model.NewClient(ts.URL)
	t.Run("missing source cluster", func(t *testing.T) {
		err := client.MigrateClusterInstallation(&model.MigrateClusterInstallationRequest{SourceClusterID: "", TargetClusterID: "4567"})
		require.EqualError(t, err, "failed with status code 400")
	})

	t.Run("missing target cluster", func(t *testing.T) {
		err := client.MigrateClusterInstallation(&model.MigrateClusterInstallationRequest{SourceClusterID: "12345", TargetClusterID: ""})
		require.EqualError(t, err, "failed with status code 400")
	})

	t.Run("No cluster installation found to migrate", func(t *testing.T) {
		err := client.MigrateClusterInstallation(&model.MigrateClusterInstallationRequest{SourceClusterID: "12345", TargetClusterID: "67899"})
		require.EqualError(t, err, "failed with status code 404")
	})

	// Valid migration test
	sourceCluster, err := client.CreateCluster(&model.CreateClusterRequest{
		Provider:    model.ProviderAWS,
		Zones:       []string{"zone"},
		Annotations: []string{"my-annotation"},
	})
	require.NoError(t, err)
	require.NotNil(t, sourceCluster.ID)

	installation1, err := client.CreateInstallation(&model.CreateInstallationRequest{
		OwnerID:  "owner1",
		Version:  "version",
		Image:    "custom-image",
		DNS:      "dns1.example.com",
		Affinity: model.InstallationAffinityIsolated,
	})
	require.NoError(t, err)

	time.Sleep(1 * time.Millisecond)

	installation2, err := client.CreateInstallation(&model.CreateInstallationRequest{
		OwnerID:  "owner2",
		Version:  "version",
		Image:    "custom-image",
		DNS:      "dns2.example.com",
		Affinity: model.InstallationAffinityIsolated,
	})
	require.NoError(t, err)

	installation3 := &model.Installation{
		OwnerID:   "owner3",
		Version:   "version3",
		Image:     "custom-image",
		DNS:       "dns3.example.com",
		Database:  model.InstallationDatabaseMysqlOperator,
		Filestore: model.InstallationFilestoreMinioOperator,
		Affinity:  model.InstallationAffinityIsolated,
		State:     model.InstallationStateHibernating,
	}

	err = sqlStore.CreateInstallation(installation3, nil)
	require.NoError(t, err)

	time.Sleep(1 * time.Millisecond)
	clusterInstallation1 := &model.ClusterInstallation{
		ClusterID:      sourceCluster.ID,
		InstallationID: installation1.ID,
		Namespace:      "namespace_10",
		State:          model.ClusterInstallationStateCreationRequested,
		IsActive:       true,
	}
	err = sqlStore.CreateClusterInstallation(clusterInstallation1)
	require.NoError(t, err)

	clusterInstallation2 := &model.ClusterInstallation{
		ClusterID:      sourceCluster.ID,
		InstallationID: installation2.ID,
		Namespace:      "namespace_11",
		State:          model.ClusterInstallationStateCreationRequested,
		IsActive:       true,
	}
	err = sqlStore.CreateClusterInstallation(clusterInstallation2)
	require.NoError(t, err)

	clusterInstallation3 := &model.ClusterInstallation{
		ClusterID:      sourceCluster.ID,
		InstallationID: installation3.ID,
		Namespace:      "namespace_11",
		State:          model.ClusterInstallationStateCreationRequested,
		IsActive:       true,
	}
	err = sqlStore.CreateClusterInstallation(clusterInstallation3)
	require.NoError(t, err)

	targetCluster, err := client.CreateCluster(&model.CreateClusterRequest{
		Provider:    model.ProviderAWS,
		Zones:       []string{"zone"},
		Annotations: []string{"my-annotation"},
	})
	require.NoError(t, err)
	require.NotNil(t, targetCluster)

	t.Run("valid migration test", func(t *testing.T) {
		err := client.MigrateClusterInstallation(&model.MigrateClusterInstallationRequest{InstallationID: "", SourceClusterID: sourceCluster.ID, TargetClusterID: targetCluster.ID, DNSSwitch: false, LockInstallation: false})
		require.NoError(t, err)

		err = client.MigrateDNS(&model.MigrateClusterInstallationRequest{InstallationID: "", SourceClusterID: sourceCluster.ID, TargetClusterID: targetCluster.ID, DNSSwitch: true, LockInstallation: true})
		require.NoError(t, err)

		// varifying the outcomes
		var isActiveClusterInstallations = false
		filter := &model.ClusterInstallationFilter{
			ClusterID:      sourceCluster.ID,
			InstallationID: "",
			Paging:         model.AllPagesNotDeleted(),
			IsActive:       &isActiveClusterInstallations,
		}
		cis, err := sqlStore.GetClusterInstallations(filter)
		require.NoError(t, err)
		require.NotEmpty(t, cis)

		isActiveClusterInstallations = true
		filter = &model.ClusterInstallationFilter{
			ClusterID:      targetCluster.ID,
			InstallationID: "",
			Paging:         model.AllPagesNotDeleted(),
			IsActive:       &isActiveClusterInstallations,
		}
		cis, err = sqlStore.GetClusterInstallations(filter)
		require.NoError(t, err)
		require.NotEmpty(t, cis)
		assert.Len(t, cis, 3)
	})

}
func TestMigrateDNSForHibernatingInstallation(t *testing.T) {
	logger := testlib.MakeLogger(t)
	sqlStore := store.MakeTestSQLStore(t, logger)

	router := mux.NewRouter()
	api.Register(router, &api.Context{
		Store:      sqlStore,
		Supervisor: &mockSupervisor{},
		Logger:     logger,
	})
	ts := httptest.NewServer(router)
	defer ts.Close()

	model.SetDeployOperators(true, true)

	t.Run("invalid payload", func(t *testing.T) {
		resp, err := http.Post(fmt.Sprintf("%s/api/cluster_installations/migrate/dns", ts.URL), "application/json", bytes.NewReader([]byte("invalid")))
		require.NoError(t, err)
		require.Equal(t, http.StatusBadRequest, resp.StatusCode)
	})

	t.Run("empty payload", func(t *testing.T) {
		resp, err := http.Post(fmt.Sprintf("%s/api/cluster_installations/migrate/dns", ts.URL), "application/json", bytes.NewReader([]byte("")))
		require.NoError(t, err)
		require.Equal(t, http.StatusBadRequest, resp.StatusCode)
	})

	client := model.NewClient(ts.URL)
	t.Run("missing source cluster", func(t *testing.T) {
		err := client.MigrateClusterInstallation(&model.MigrateClusterInstallationRequest{SourceClusterID: "", TargetClusterID: "4567"})
		require.EqualError(t, err, "failed with status code 400")
	})

	t.Run("missing target cluster", func(t *testing.T) {
		err := client.MigrateClusterInstallation(&model.MigrateClusterInstallationRequest{SourceClusterID: "12345", TargetClusterID: ""})
		require.EqualError(t, err, "failed with status code 400")
	})

	t.Run("No cluster installation found to migrate", func(t *testing.T) {
		err := client.MigrateClusterInstallation(&model.MigrateClusterInstallationRequest{SourceClusterID: "12345", TargetClusterID: "67899"})
		require.EqualError(t, err, "failed with status code 404")
	})

	// Valid migration test
	sourceCluster, err := client.CreateCluster(&model.CreateClusterRequest{
		Provider:    model.ProviderAWS,
		Zones:       []string{"zone"},
		Annotations: []string{"my-annotation"},
	})
	require.NoError(t, err)
	require.NotNil(t, sourceCluster.ID)

	installation1 := &model.Installation{
		OwnerID:   "owner1",
		Version:   "version1",
		Image:     "custom-image",
		DNS:       "dns1.example.com",
		Database:  model.InstallationDatabaseMysqlOperator,
		Filestore: model.InstallationFilestoreMinioOperator,
		Affinity:  model.InstallationAffinityIsolated,
		State:     model.InstallationStateHibernating,
	}

	err = sqlStore.CreateInstallation(installation1, nil)
	require.NoError(t, err)

	installation2 := &model.Installation{
		OwnerID:   "owner2",
		Version:   "version2",
		Image:     "custom-image",
		DNS:       "dns3.example.com",
		Database:  model.InstallationDatabaseMysqlOperator,
		Filestore: model.InstallationFilestoreMinioOperator,
		Affinity:  model.InstallationAffinityIsolated,
		State:     model.InstallationStateHibernating,
	}

	err = sqlStore.CreateInstallation(installation2, nil)
	require.NoError(t, err)

	time.Sleep(1 * time.Millisecond)
	clusterInstallation1 := &model.ClusterInstallation{
		ClusterID:      sourceCluster.ID,
		InstallationID: installation1.ID,
		Namespace:      "namespace_10",
		State:          model.ClusterInstallationStateCreationRequested,
		IsActive:       true,
	}
	err = sqlStore.CreateClusterInstallation(clusterInstallation1)
	require.NoError(t, err)

	clusterInstallation2 := &model.ClusterInstallation{
		ClusterID:      sourceCluster.ID,
		InstallationID: installation2.ID,
		Namespace:      "namespace_11",
		State:          model.ClusterInstallationStateCreationRequested,
		IsActive:       true,
	}
	err = sqlStore.CreateClusterInstallation(clusterInstallation2)
	require.NoError(t, err)

	targetCluster, err := client.CreateCluster(&model.CreateClusterRequest{
		Provider:    model.ProviderAWS,
		Zones:       []string{"zone"},
		Annotations: []string{"my-annotation"},
	})
	require.NoError(t, err)
	require.NotNil(t, targetCluster)

	t.Run("valid migration test", func(t *testing.T) {
		err := client.MigrateClusterInstallation(&model.MigrateClusterInstallationRequest{InstallationID: "", SourceClusterID: sourceCluster.ID, TargetClusterID: targetCluster.ID, DNSSwitch: false, LockInstallation: false})
		require.NoError(t, err)

		err = client.MigrateDNS(&model.MigrateClusterInstallationRequest{InstallationID: "", SourceClusterID: sourceCluster.ID, TargetClusterID: targetCluster.ID, DNSSwitch: true, LockInstallation: true})
		require.NoError(t, err)

		// varifying the outcomes
		var isActiveClusterInstallations = false
		filter := &model.ClusterInstallationFilter{
			ClusterID:      sourceCluster.ID,
			InstallationID: "",
			Paging:         model.AllPagesNotDeleted(),
			IsActive:       &isActiveClusterInstallations,
		}
		cis, err := sqlStore.GetClusterInstallations(filter)
		require.NoError(t, err)
		require.NotEmpty(t, cis)

		isActiveClusterInstallations = true
		filter = &model.ClusterInstallationFilter{
			ClusterID:      targetCluster.ID,
			InstallationID: "",
			Paging:         model.AllPagesNotDeleted(),
			IsActive:       &isActiveClusterInstallations,
		}
		cis, err = sqlStore.GetClusterInstallations(filter)
		require.NoError(t, err)
		require.NotEmpty(t, cis)
		assert.Len(t, cis, 2)

		// Verifying if migration has already processed
		err = client.MigrateClusterInstallation(&model.MigrateClusterInstallationRequest{InstallationID: "", SourceClusterID: sourceCluster.ID, TargetClusterID: targetCluster.ID, DNSSwitch: false, LockInstallation: false})
		require.EqualError(t, err, "failed with status code 404")

	})

}
func TestMigrateDNSForNonHibernatingInstallation(t *testing.T) {
	logger := testlib.MakeLogger(t)
	sqlStore := store.MakeTestSQLStore(t, logger)

	router := mux.NewRouter()
	api.Register(router, &api.Context{
		Store:      sqlStore,
		Supervisor: &mockSupervisor{},
		Logger:     logger,
	})
	ts := httptest.NewServer(router)
	defer ts.Close()

	model.SetDeployOperators(true, true)

	t.Run("invalid payload", func(t *testing.T) {
		resp, err := http.Post(fmt.Sprintf("%s/api/cluster_installations/migrate/dns", ts.URL), "application/json", bytes.NewReader([]byte("invalid")))
		require.NoError(t, err)
		require.Equal(t, http.StatusBadRequest, resp.StatusCode)
	})

	t.Run("empty payload", func(t *testing.T) {
		resp, err := http.Post(fmt.Sprintf("%s/api/cluster_installations/migrate/dns", ts.URL), "application/json", bytes.NewReader([]byte("")))
		require.NoError(t, err)
		require.Equal(t, http.StatusBadRequest, resp.StatusCode)
	})

	client := model.NewClient(ts.URL)
	t.Run("missing source cluster", func(t *testing.T) {
		err := client.MigrateClusterInstallation(&model.MigrateClusterInstallationRequest{SourceClusterID: "", TargetClusterID: "4567"})
		require.EqualError(t, err, "failed with status code 400")
	})

	t.Run("missing target cluster", func(t *testing.T) {
		err := client.MigrateClusterInstallation(&model.MigrateClusterInstallationRequest{SourceClusterID: "12345", TargetClusterID: ""})
		require.EqualError(t, err, "failed with status code 400")
	})

	t.Run("No cluster installation found to migrate", func(t *testing.T) {
		err := client.MigrateClusterInstallation(&model.MigrateClusterInstallationRequest{SourceClusterID: "12345", TargetClusterID: "67899"})
		require.EqualError(t, err, "failed with status code 404")
	})

	// Valid migration test
	sourceCluster, err := client.CreateCluster(&model.CreateClusterRequest{
		Provider:    model.ProviderAWS,
		Zones:       []string{"zone"},
		Annotations: []string{"my-annotation"},
	})
	require.NoError(t, err)
	require.NotNil(t, sourceCluster.ID)

	installation1 := &model.Installation{
		OwnerID:   "owner1",
		Version:   "version1",
		Image:     "custom-image",
		DNS:       "dns1.example.com",
		Database:  model.InstallationDatabaseMysqlOperator,
		Filestore: model.InstallationFilestoreMinioOperator,
		Affinity:  model.InstallationAffinityIsolated,
		State:     model.InstallationStateCreationRequested,
	}

	err = sqlStore.CreateInstallation(installation1, nil)
	require.NoError(t, err)

	installation2 := &model.Installation{
		OwnerID:   "owner2",
		Version:   "version2",
		Image:     "custom-image",
		DNS:       "dns3.example.com",
		Database:  model.InstallationDatabaseMysqlOperator,
		Filestore: model.InstallationFilestoreMinioOperator,
		Affinity:  model.InstallationAffinityIsolated,
		State:     model.InstallationStateCreationRequested,
	}

	err = sqlStore.CreateInstallation(installation2, nil)
	require.NoError(t, err)

	time.Sleep(1 * time.Millisecond)
	clusterInstallation1 := &model.ClusterInstallation{
		ClusterID:      sourceCluster.ID,
		InstallationID: installation1.ID,
		Namespace:      "namespace_10",
		State:          model.ClusterInstallationStateCreationRequested,
		IsActive:       true,
	}
	err = sqlStore.CreateClusterInstallation(clusterInstallation1)
	require.NoError(t, err)

	clusterInstallation2 := &model.ClusterInstallation{
		ClusterID:      sourceCluster.ID,
		InstallationID: installation2.ID,
		Namespace:      "namespace_11",
		State:          model.ClusterInstallationStateCreationRequested,
		IsActive:       true,
	}
	err = sqlStore.CreateClusterInstallation(clusterInstallation2)
	require.NoError(t, err)

	targetCluster, err := client.CreateCluster(&model.CreateClusterRequest{
		Provider:    model.ProviderAWS,
		Zones:       []string{"zone"},
		Annotations: []string{"my-annotation"},
	})
	require.NoError(t, err)
	require.NotNil(t, targetCluster)

	t.Run("valid migration test", func(t *testing.T) {
		err := client.MigrateClusterInstallation(&model.MigrateClusterInstallationRequest{InstallationID: "", SourceClusterID: sourceCluster.ID, TargetClusterID: targetCluster.ID, DNSSwitch: false, LockInstallation: false})
		require.NoError(t, err)

		err = client.MigrateDNS(&model.MigrateClusterInstallationRequest{InstallationID: "", SourceClusterID: sourceCluster.ID, TargetClusterID: targetCluster.ID, DNSSwitch: true, LockInstallation: true})
		require.NoError(t, err)

		// varifying the outcomes
		var isActiveClusterInstallations = false
		filter := &model.ClusterInstallationFilter{
			ClusterID:      sourceCluster.ID,
			InstallationID: "",
			Paging:         model.AllPagesNotDeleted(),
			IsActive:       &isActiveClusterInstallations,
		}
		cis, err := sqlStore.GetClusterInstallations(filter)
		require.NoError(t, err)
		require.NotEmpty(t, cis)

		isActiveClusterInstallations = true
		filter = &model.ClusterInstallationFilter{
			ClusterID:      targetCluster.ID,
			InstallationID: "",
			Paging:         model.AllPagesNotDeleted(),
			IsActive:       &isActiveClusterInstallations,
		}
		cis, err = sqlStore.GetClusterInstallations(filter)
		require.NoError(t, err)
		require.NotEmpty(t, cis)
		assert.Len(t, cis, 2)

		// Verifying if DNS Switch has already processed
		err = client.MigrateDNS(&model.MigrateClusterInstallationRequest{InstallationID: "", SourceClusterID: sourceCluster.ID, TargetClusterID: targetCluster.ID, DNSSwitch: true, LockInstallation: true})
		require.EqualError(t, err, "failed with status code 404")
	})

}
func TestDeleteInActiveClusterInstallationsByCluster(t *testing.T) {
	logger := testlib.MakeLogger(t)
	sqlStore := store.MakeTestSQLStore(t, logger)

	router := mux.NewRouter()
	api.Register(router, &api.Context{
		Store:      sqlStore,
		Supervisor: &mockSupervisor{},
		Logger:     logger,
	})
	ts := httptest.NewServer(router)
	defer ts.Close()
	client := model.NewClient(ts.URL)
	sourceClusterID := model.NewID()
	installationID1 := model.NewID()
	clusterInstallation1 := &model.ClusterInstallation{
		ClusterID:      sourceClusterID,
		InstallationID: installationID1,
		Namespace:      "namespace_10",
		State:          model.ClusterInstallationStateCreationRequested,
		IsActive:       false,
	}

	time.Sleep(1 * time.Millisecond)

	installationID2 := model.NewID()
	clusterInstallation2 := &model.ClusterInstallation{
		ClusterID:      sourceClusterID,
		InstallationID: installationID2,
		Namespace:      "namespace_11",
		State:          model.ClusterInstallationStateCreationRequested,
		IsActive:       false,
	}

	err := sqlStore.CreateClusterInstallation(clusterInstallation1)
	require.NoError(t, err)

	err = sqlStore.CreateClusterInstallation(clusterInstallation2)
	require.NoError(t, err)

	isActiveClusterInstallations := false
	filter := &model.ClusterInstallationFilter{
		ClusterID:      sourceClusterID,
		InstallationID: "",
		Paging:         model.AllPagesNotDeleted(),
		IsActive:       &isActiveClusterInstallations,
	}
	ci, err := sqlStore.GetClusterInstallations(filter)
	require.NoError(t, err)
	require.NotEmpty(t, ci)

	t.Run("delete inActive cluster installation in a given cluster", func(t *testing.T) {
		err := client.DeleteInActiveClusterInstallationsByCluster(sourceClusterID)
		require.NoError(t, err)
	})

	cis, err := sqlStore.GetClusterInstallations(filter)
	require.NoError(t, err)
	for _, ci := range cis {
		require.Equal(t, ci.State, model.ClusterInstallationStateDeletionRequested)
	}

}

func TestDeleteInActiveClusterInstallationsByID(t *testing.T) {
	logger := testlib.MakeLogger(t)
	sqlStore := store.MakeTestSQLStore(t, logger)

	router := mux.NewRouter()
	api.Register(router, &api.Context{
		Store:      sqlStore,
		Supervisor: &mockSupervisor{},
		Logger:     logger,
	})
	ts := httptest.NewServer(router)
	defer ts.Close()
	client := model.NewClient(ts.URL)

	// Valid migration test for single installation
	sourceClusterID := model.NewID()
	installationID1 := model.NewID()
	clusterInstallation1 := &model.ClusterInstallation{
		ClusterID:      sourceClusterID,
		InstallationID: installationID1,
		Namespace:      "namespace_12",
		State:          model.ClusterInstallationStateCreationRequested,
		IsActive:       false,
	}

	time.Sleep(1 * time.Millisecond)

	err := sqlStore.CreateClusterInstallation(clusterInstallation1)
	require.NoError(t, err)

	isActiveClusterInstallations := false
	filter := &model.ClusterInstallationFilter{
		ClusterID:      sourceClusterID,
		InstallationID: clusterInstallation1.InstallationID,
		Paging:         model.AllPagesNotDeleted(),
		IsActive:       &isActiveClusterInstallations,
	}
	ci, err := sqlStore.GetClusterInstallations(filter)
	require.NoError(t, err)
	require.NotEmpty(t, ci)

	t.Run("delete inActive cluster installation by ID", func(t *testing.T) {
		err := client.DeleteInActiveClusterInstallationByID(clusterInstallation1.ID)
		require.NoError(t, err)
	})

	ci, err = sqlStore.GetClusterInstallations(filter)
	require.NoError(t, err)
	require.Equal(t, ci[0].State, model.ClusterInstallationStateDeletionRequested)

}
