// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.
//

package api_test

import (
	"bytes"
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

func TestGetInstallations(t *testing.T) {
	logger := testlib.MakeLogger(t)
	sqlStore := store.MakeTestSQLStore(t, logger)
	defer store.CloseConnection(t, sqlStore)

	router := mux.NewRouter()
	api.Register(router, &api.Context{
		Store:      sqlStore,
		Supervisor: &mockSupervisor{},
		Logger:     logger,
	})
	ts := httptest.NewServer(router)
	defer ts.Close()

	client := model.NewClient(ts.URL)

	t.Run("unknown installation", func(t *testing.T) {
		installation, err := client.GetInstallation(model.NewID(), nil)
		require.NoError(t, err)
		require.Nil(t, installation)
	})

	t.Run("no installations", func(t *testing.T) {
		installations, err := client.GetInstallations(&model.GetInstallationsRequest{
			Page:           0,
			PerPage:        10,
			IncludeDeleted: true,
		})
		require.NoError(t, err)
		require.Empty(t, installations)
	})

	t.Run("group parameter handling", func(t *testing.T) {
		t.Run("invalid include group config", func(t *testing.T) {
			resp, err := http.Get(fmt.Sprintf("%s/api/installation/%s?include_group_config=invalid&include_group_config_overrides=true", ts.URL, model.NewID()))
			require.NoError(t, err)
			require.Equal(t, http.StatusBadRequest, resp.StatusCode)
		})

		t.Run("invalid include group config override", func(t *testing.T) {
			resp, err := http.Get(fmt.Sprintf("%s/api/installation/%s?include_group_config=true&include_group_config_overrides=invalid", ts.URL, model.NewID()))
			require.NoError(t, err)
			require.Equal(t, http.StatusBadRequest, resp.StatusCode)
		})

		t.Run("no group parameters", func(t *testing.T) {
			resp, err := http.Get(fmt.Sprintf("%s/api/installation/%s", ts.URL, model.NewID()))
			require.NoError(t, err)
			require.Equal(t, http.StatusNotFound, resp.StatusCode)
		})

		t.Run("missing include group config", func(t *testing.T) {
			resp, err := http.Get(fmt.Sprintf("%s/api/installation/%s?include_group_config_overrides=true", ts.URL, model.NewID()))
			require.NoError(t, err)
			require.Equal(t, http.StatusNotFound, resp.StatusCode)
		})

		t.Run("missing include group config overrides", func(t *testing.T) {
			resp, err := http.Get(fmt.Sprintf("%s/api/installation/%s?include_group_config=true", ts.URL, model.NewID()))
			require.NoError(t, err)
			require.Equal(t, http.StatusNotFound, resp.StatusCode)
		})
	})

	t.Run("page parameter handling", func(t *testing.T) {
		t.Run("invalid page", func(t *testing.T) {
			resp, err := http.Get(fmt.Sprintf("%s/api/installations?page=invalid&per_page=100", ts.URL))
			require.NoError(t, err)
			require.Equal(t, http.StatusBadRequest, resp.StatusCode)
		})

		t.Run("invalid perPage", func(t *testing.T) {
			resp, err := http.Get(fmt.Sprintf("%s/api/installations?page=0&per_page=invalid", ts.URL))
			require.NoError(t, err)
			require.Equal(t, http.StatusBadRequest, resp.StatusCode)
		})

		t.Run("no paging parameters", func(t *testing.T) {
			resp, err := http.Get(fmt.Sprintf("%s/api/installations", ts.URL))
			require.NoError(t, err)
			require.Equal(t, http.StatusOK, resp.StatusCode)
		})

		t.Run("missing page", func(t *testing.T) {
			resp, err := http.Get(fmt.Sprintf("%s/api/installations?per_page=100", ts.URL))
			require.NoError(t, err)
			require.Equal(t, http.StatusOK, resp.StatusCode)
		})

		t.Run("missing perPage", func(t *testing.T) {
			resp, err := http.Get(fmt.Sprintf("%s/api/installations?page=1", ts.URL))
			require.NoError(t, err)
			require.Equal(t, http.StatusOK, resp.StatusCode)
		})
	})

	t.Run("results", func(t *testing.T) {
		ownerID1 := model.NewID()
		ownerID2 := model.NewID()
		annotations := []*model.Annotation{
			{ID: "", Name: "multi-tenant"},
			{ID: "", Name: "super-awesome"},
		}

		for _, ann := range annotations {
			err := sqlStore.CreateAnnotation(ann)
			require.NoError(t, err)
		}

		installation1 := &model.Installation{
			OwnerID:  ownerID1,
			Version:  "version",
			DNS:      "dns.example.com",
			Size:     "1000users",
			Affinity: model.InstallationAffinityIsolated,
			State:    model.InstallationStateCreationRequested,
		}
		err := sqlStore.CreateInstallation(installation1, annotations)
		require.NoError(t, err)

		time.Sleep(1 * time.Millisecond)

		installation2 := &model.Installation{
			OwnerID:  ownerID2,
			Version:  "version",
			DNS:      "dns2.example.com",
			Affinity: model.InstallationAffinityIsolated,
			State:    model.InstallationStateCreationRequested,
		}
		err = sqlStore.CreateInstallation(installation2, nil)
		require.NoError(t, err)

		time.Sleep(1 * time.Millisecond)

		installation3 := &model.Installation{
			OwnerID:  ownerID1,
			Version:  "version",
			DNS:      "dns3.example.com",
			Affinity: model.InstallationAffinityIsolated,
			State:    model.InstallationStateCreationRequested,
		}
		err = sqlStore.CreateInstallation(installation3, nil)
		require.NoError(t, err)

		time.Sleep(1 * time.Millisecond)

		installation4 := &model.Installation{
			OwnerID:  ownerID2,
			Version:  "version",
			DNS:      "dns4.example.com",
			Affinity: model.InstallationAffinityIsolated,
			State:    model.InstallationStateCreationRequested,
		}
		err = sqlStore.CreateInstallation(installation4, nil)
		require.NoError(t, err)
		err = sqlStore.DeleteInstallation(installation4.ID)
		require.NoError(t, err)
		installation4DTO, err := client.GetInstallation(installation4.ID, nil)
		require.NoError(t, err)
		installation4 = installation4DTO.Installation

		t.Run("get installation", func(t *testing.T) {
			t.Run("installation 1", func(t *testing.T) {
				installationDTO, err := client.GetInstallation(installation1.ID, nil)
				require.NoError(t, err)
				require.Equal(t, installation1, installationDTO.Installation)
				require.Equal(t, 2, len(installationDTO.Annotations))
				require.Equal(t, annotations, model.SortAnnotations(installationDTO.Annotations))
				require.Nil(t, installationDTO.SingleTenantDatabaseConfig)
			})

			t.Run("get deleted installation", func(t *testing.T) {
				installationDTO, err := client.GetInstallation(installation4.ID, nil)
				require.NoError(t, err)
				require.Equal(t, installation4, installationDTO.Installation)
			})

			t.Run("get installation by name", func(t *testing.T) {
				installation, err := client.GetInstallationByDNS(installation1.DNS, nil)
				assert.NoError(t, err)
				require.NotNil(t, installation)
				assert.Equal(t, installation1.ID, installation.ID)
				assert.Equal(t, installation1.DNS, installation.DNS)

				noInstallation, err := client.GetInstallationByDNS("notreal", nil)
				assert.Nil(t, noInstallation)
				assert.NoError(t, err)
			})

		})

		t.Run("get installations", func(t *testing.T) {
			testCases := []struct {
				Description             string
				GetInstallationsRequest *model.GetInstallationsRequest
				Expected                []*model.Installation
			}{
				{
					"page 0, perPage 2, exclude deleted",
					&model.GetInstallationsRequest{
						Page:           0,
						PerPage:        2,
						IncludeDeleted: false,
					},
					[]*model.Installation{installation1, installation2},
				},

				{
					"page 1, perPage 2, exclude deleted",
					&model.GetInstallationsRequest{
						Page:           1,
						PerPage:        2,
						IncludeDeleted: false,
					},
					[]*model.Installation{installation3},
				},

				{
					"page 0, perPage 2, include deleted",
					&model.GetInstallationsRequest{
						Page:           0,
						PerPage:        2,
						IncludeDeleted: true,
					},
					[]*model.Installation{installation1, installation2},
				},

				{
					"page 1, perPage 2, include deleted",
					&model.GetInstallationsRequest{
						Page:           1,
						PerPage:        2,
						IncludeDeleted: true,
					},
					[]*model.Installation{installation3, installation4},
				},
				{
					"filter by owner",
					&model.GetInstallationsRequest{
						OwnerID:        ownerID1,
						Page:           0,
						PerPage:        100,
						IncludeDeleted: false,
					},
					[]*model.Installation{installation1, installation3},
				},
				{
					"filter by dns",
					&model.GetInstallationsRequest{
						DNS:            installation1.DNS,
						Page:           0,
						PerPage:        100,
						IncludeDeleted: false,
					},
					[]*model.Installation{installation1},
				},
				{
					"filter by state creation-requested",
					&model.GetInstallationsRequest{
						State:          model.InstallationStateCreationRequested,
						Page:           0,
						PerPage:        100,
						IncludeDeleted: false,
					},
					[]*model.Installation{installation1, installation2, installation3},
				},
				{
					"filter by state stable",
					&model.GetInstallationsRequest{
						State:          model.InstallationStateStable,
						Page:           0,
						PerPage:        100,
						IncludeDeleted: false,
					},
					[]*model.Installation{},
				},
			}

			for _, testCase := range testCases {
				t.Run(testCase.Description, func(t *testing.T) {
					installationDTOs, err := client.GetInstallations(testCase.GetInstallationsRequest)
					require.NoError(t, err)
					require.Equal(t, testCase.Expected, dtosToInstallations(installationDTOs))
				})
			}
		})

		t.Run("get installations count", func(t *testing.T) {
			testCases := []struct {
				Description    string
				IncludeDeleted bool
				Expected       int64
			}{
				{
					"count without deleted",
					false,
					3,
				},
				{
					"count with deleted",
					true,
					4,
				},
			}
			for _, testCase := range testCases {
				t.Run(testCase.Description, func(t *testing.T) {
					installations, err := client.GetInstallationsCount(testCase.IncludeDeleted)
					require.NoError(t, err)
					require.Equal(t, testCase.Expected, installations)
				})
			}
		})

		t.Run("check installations status", func(t *testing.T) {
			status, err := client.GetInstallationsStatus()
			require.NoError(t, err)
			assert.Equal(t, int64(3), status.InstallationsTotal)
			assert.Equal(t, int64(0), status.InstallationsStable)
			assert.Equal(t, int64(0), status.InstallationsHibernating)
			assert.Equal(t, int64(3), status.InstallationsUpdating)
		})
	})
}

func TestCreateInstallation(t *testing.T) {
	logger := testlib.MakeLogger(t)
	sqlStore := store.MakeTestSQLStore(t, logger)
	defer store.CloseConnection(t, sqlStore)

	router := mux.NewRouter()
	api.Register(router, &api.Context{
		Store:      sqlStore,
		Supervisor: &mockSupervisor{},
		Logger:     logger,
	})
	ts := httptest.NewServer(router)
	defer ts.Close()

	client := model.NewClient(ts.URL)

	t.Run("invalid payload", func(t *testing.T) {
		resp, err := http.Post(fmt.Sprintf("%s/api/installations", ts.URL), "application/json", bytes.NewReader([]byte("invalid")))
		require.NoError(t, err)
		require.Equal(t, http.StatusBadRequest, resp.StatusCode)
	})

	t.Run("empty payload", func(t *testing.T) {
		resp, err := http.Post(fmt.Sprintf("%s/api/installations", ts.URL), "application/json", bytes.NewReader([]byte("")))
		require.NoError(t, err)
		require.Equal(t, http.StatusBadRequest, resp.StatusCode)
	})

	t.Run("missing owner", func(t *testing.T) {
		_, err := client.CreateInstallation(&model.CreateInstallationRequest{
			Version:  "version",
			DNS:      "dns.example.com",
			Affinity: model.InstallationAffinityIsolated,
		})
		require.EqualError(t, err, "failed with status code 400")
	})

	t.Run("missing dns", func(t *testing.T) {
		_, err := client.CreateInstallation(&model.CreateInstallationRequest{
			OwnerID:  "owner",
			Version:  "version",
			Affinity: model.InstallationAffinityIsolated,
		})
		require.EqualError(t, err, "failed with status code 400")
	})

	t.Run("invalid size", func(t *testing.T) {
		_, err := client.CreateInstallation(&model.CreateInstallationRequest{
			OwnerID:  "owner",
			Version:  "version",
			DNS:      "dns.example.com",
			Size:     "junk",
			Affinity: model.InstallationAffinityIsolated,
		})
		require.EqualError(t, err, "failed with status code 400")
	})

	t.Run("dns too long", func(t *testing.T) {
		_, err := client.CreateInstallation(&model.CreateInstallationRequest{
			OwnerID:  "owner",
			Version:  "version",
			DNS:      "xoxo-serverwithverylongnametoexposeissuesrelatedtolengthofkeystha",
			Affinity: model.InstallationAffinityIsolated,
		})
		require.EqualError(t, err, "failed with status code 400")
	})

	t.Run("invalid dns", func(t *testing.T) {
		_, err := client.CreateInstallation(&model.CreateInstallationRequest{
			OwnerID:  "owner",
			Version:  "version",
			DNS:      string([]byte{0x7f}),
			Affinity: model.InstallationAffinityIsolated,
		})
		require.EqualError(t, err, "failed with status code 400")
	})

	t.Run("invalid affinity", func(t *testing.T) {
		_, err := client.CreateInstallation(&model.CreateInstallationRequest{
			OwnerID:  "owner",
			Version:  "version",
			DNS:      "dns.example.com",
			Affinity: "invalid",
		})
		require.EqualError(t, err, "failed with status code 400")
	})

	t.Run("invalid annotations", func(t *testing.T) {
		_, err := client.CreateInstallation(&model.CreateInstallationRequest{
			OwnerID:     "owner",
			Version:     "version",
			DNS:         "dns.example.com",
			Affinity:    model.InstallationAffinityIsolated,
			Annotations: []string{"my invalid annotation"},
		})
		require.EqualError(t, err, "failed with status code 400")
	})

	t.Run("valid", func(t *testing.T) {
		installation, err := client.CreateInstallation(&model.CreateInstallationRequest{
			OwnerID:     "owner",
			Version:     "version",
			DNS:         "dns.example.com",
			Affinity:    model.InstallationAffinityIsolated,
			Annotations: []string{"my-annotation"},
		})
		require.NoError(t, err)
		require.Equal(t, "owner", installation.OwnerID)
		require.Equal(t, "version", installation.Version)
		require.Equal(t, "mattermost/mattermost-enterprise-edition", installation.Image)
		require.Equal(t, "dns.example.com", installation.DNS)
		require.Equal(t, model.InstallationAffinityIsolated, installation.Affinity)
		require.Equal(t, model.InstallationStateCreationRequested, installation.State)
		require.Equal(t, model.DefaultCRVersion, installation.CRVersion)
		require.Empty(t, installation.LockAcquiredBy)
		require.EqualValues(t, 0, installation.LockAcquiredAt)
		require.NotEqual(t, 0, installation.CreateAt)
		require.EqualValues(t, 0, installation.DeleteAt)
		assert.True(t, containsAnnotation("my-annotation", installation.Annotations))
	})

	t.Run("valid with custom image", func(t *testing.T) {
		installation, err := client.CreateInstallation(&model.CreateInstallationRequest{
			OwnerID:  "owner1",
			Version:  "version",
			Image:    "custom-image",
			DNS:      "dns1.example.com",
			Affinity: model.InstallationAffinityIsolated,
		})
		require.NoError(t, err)
		require.Equal(t, "owner1", installation.OwnerID)
		require.Equal(t, "version", installation.Version)
		require.Equal(t, "custom-image", installation.Image)
		require.Equal(t, "dns1.example.com", installation.DNS)
		require.Equal(t, model.InstallationAffinityIsolated, installation.Affinity)
		require.Equal(t, model.InstallationStateCreationRequested, installation.State)
		require.Equal(t, model.DefaultCRVersion, installation.CRVersion)
		require.Empty(t, installation.LockAcquiredBy)
		require.EqualValues(t, 0, installation.LockAcquiredAt)
		require.NotEqual(t, 0, installation.CreateAt)
		require.EqualValues(t, 0, installation.DeleteAt)
	})

	t.Run("groups", func(t *testing.T) {
		t.Run("create with group", func(t *testing.T) {
			group, err := client.CreateGroup(&model.CreateGroupRequest{
				Name:    "name1",
				Version: "version1",
				Image:   "sample/image1",
			})
			require.NoError(t, err)

			installation, err := client.CreateInstallation(&model.CreateInstallationRequest{
				OwnerID:  "owner",
				GroupID:  group.ID,
				Version:  "version",
				Image:    "custom-image",
				DNS:      "dns2.example.com",
				Affinity: model.InstallationAffinityIsolated,
			})
			require.NoError(t, err)
			require.Equal(t, group.ID, *installation.GroupID)
		})

		t.Run("create with deleted group", func(t *testing.T) {
			group, err := client.CreateGroup(&model.CreateGroupRequest{
				Name:    "name2",
				Version: "version2",
				Image:   "sample/image2",
			})
			require.NoError(t, err)

			err = client.DeleteGroup(group.ID)
			require.NoError(t, err)

			_, err = client.CreateInstallation(&model.CreateInstallationRequest{
				OwnerID:  "owner",
				GroupID:  group.ID,
				Version:  "version",
				Image:    "custom-image",
				DNS:      "dns3.example.com",
				Affinity: model.InstallationAffinityIsolated,
			})
			require.EqualError(t, err, "failed with status code 400")
		})
	})

	t.Run("handle annotations", func(t *testing.T) {
		annotations := []*model.Annotation{
			{ID: "", Name: "multi-tenant"},
			{ID: "", Name: "super-awesome"},
		}

		for _, ann := range annotations {
			err := sqlStore.CreateAnnotation(ann)
			require.NoError(t, err)
		}

		for i, testCase := range []struct {
			description string
			annotations []string
			expected    []*model.Annotation
		}{
			{"nil annotations", nil, nil},
			{"empty annotations", []string{}, nil},
			{"with annotations", []string{"multi-tenant", "super-awesome"}, annotations},
		} {
			t.Run(testCase.description, func(t *testing.T) {
				installation, err := client.CreateInstallation(&model.CreateInstallationRequest{
					OwnerID:     "owner1",
					Version:     "version",
					DNS:         fmt.Sprintf("dns-annotation%d.example.com", i),
					Annotations: testCase.annotations,
				})
				require.NoError(t, err)

				assert.Equal(t, testCase.expected, installation.Annotations)
			})
		}
	})

	dbConfigRequest := model.SingleTenantDatabaseRequest{
		PrimaryInstanceType: "db.r5.xlarge",
		ReplicaInstanceType: "db.r5.large",
		ReplicasCount:       5,
	}

	t.Run("handle single tenant database configuration", func(t *testing.T) {
		installation, err := client.CreateInstallation(&model.CreateInstallationRequest{
			OwnerID:                    "owner1",
			Version:                    "version",
			DNS:                        "dns-db-config.example.com",
			SingleTenantDatabaseConfig: dbConfigRequest,
			Database:                   model.InstallationDatabaseSingleTenantRDSPostgres,
		})
		require.NoError(t, err)
		assert.Equal(t, installation.SingleTenantDatabaseConfig.PrimaryInstanceType, dbConfigRequest.PrimaryInstanceType)
		assert.Equal(t, installation.SingleTenantDatabaseConfig.ReplicaInstanceType, dbConfigRequest.ReplicaInstanceType)
		assert.Equal(t, installation.SingleTenantDatabaseConfig.ReplicasCount, dbConfigRequest.ReplicasCount)

		installation, err = client.GetInstallation(installation.ID, nil)
		require.NoError(t, err)
		assert.Equal(t, installation.SingleTenantDatabaseConfig.PrimaryInstanceType, dbConfigRequest.PrimaryInstanceType)
		assert.Equal(t, installation.SingleTenantDatabaseConfig.ReplicaInstanceType, dbConfigRequest.ReplicaInstanceType)
		assert.Equal(t, installation.SingleTenantDatabaseConfig.ReplicasCount, dbConfigRequest.ReplicasCount)
	})

	t.Run("ignore single tenant database configuration if db not single tenant", func(t *testing.T) {
		installation, err := client.CreateInstallation(&model.CreateInstallationRequest{
			OwnerID:                    "owner1",
			Version:                    "version",
			DNS:                        "dns-db-config2.example.com",
			SingleTenantDatabaseConfig: dbConfigRequest,
			Database:                   model.InstallationDatabaseMultiTenantRDSPostgres,
		})
		require.NoError(t, err)
		assert.Nil(t, installation.SingleTenantDatabaseConfig)
	})

	t.Run("set default values for single tenant database configuration", func(t *testing.T) {
		installation, err := client.CreateInstallation(&model.CreateInstallationRequest{
			OwnerID:                    "owner1",
			Version:                    "version",
			DNS:                        "dns-db-config3.example.com",
			SingleTenantDatabaseConfig: dbConfigRequest,
			Database:                   model.InstallationDatabaseSingleTenantRDSMySQL,
		})
		require.NoError(t, err)
		assert.NotNil(t, installation.SingleTenantDatabaseConfig)
		assert.NotEmpty(t, installation.SingleTenantDatabaseConfig.PrimaryInstanceType)
		assert.NotEmpty(t, installation.SingleTenantDatabaseConfig.ReplicaInstanceType)
	})
}

func TestRetryCreateInstallation(t *testing.T) {
	logger := testlib.MakeLogger(t)
	sqlStore := store.MakeTestSQLStore(t, logger)
	defer store.CloseConnection(t, sqlStore)

	router := mux.NewRouter()
	api.Register(router, &api.Context{
		Store:      sqlStore,
		Supervisor: &mockSupervisor{},
		Logger:     logger,
	})
	ts := httptest.NewServer(router)
	defer ts.Close()

	client := model.NewClient(ts.URL)

	installation1, err := client.CreateInstallation(&model.CreateInstallationRequest{
		OwnerID:     "owner",
		Version:     "version",
		DNS:         "dns.example.com",
		Affinity:    model.InstallationAffinityIsolated,
		Annotations: []string{"my-annotation"},
	})
	require.NoError(t, err)

	t.Run("unknown installation", func(t *testing.T) {
		err := client.RetryCreateInstallation(model.NewID())
		require.EqualError(t, err, "failed with status code 404")
	})

	t.Run("while locked", func(t *testing.T) {
		installation1.State = model.InstallationStateStable
		err = sqlStore.UpdateInstallation(installation1.Installation)
		require.NoError(t, err)

		lockerID := model.NewID()

		locked, err := sqlStore.LockInstallation(installation1.ID, lockerID)
		require.NoError(t, err)
		require.True(t, locked)
		defer func() {
			unlocked, err := sqlStore.UnlockInstallation(installation1.ID, lockerID, false)
			require.NoError(t, err)
			require.True(t, unlocked)
		}()

		err = client.RetryCreateInstallation(installation1.ID)
		require.EqualError(t, err, "failed with status code 409")
	})

	t.Run("while creating", func(t *testing.T) {
		installation1.State = model.InstallationStateCreationRequested
		err = sqlStore.UpdateInstallation(installation1.Installation)
		require.NoError(t, err)

		err = client.RetryCreateInstallation(installation1.ID)
		require.NoError(t, err)

		installation1, err = client.GetInstallation(installation1.ID, nil)
		require.NoError(t, err)
		require.Equal(t, model.InstallationStateCreationRequested, installation1.State)
		assert.True(t, containsAnnotation("my-annotation", installation1.Annotations))
	})

	t.Run("while stable", func(t *testing.T) {
		installation1.State = model.InstallationStateStable
		err = sqlStore.UpdateInstallation(installation1.Installation)
		require.NoError(t, err)

		err = client.RetryCreateInstallation(installation1.ID)
		require.EqualError(t, err, "failed with status code 400")
	})

	t.Run("while creation failed", func(t *testing.T) {
		installation1.State = model.InstallationStateCreationFailed
		err = sqlStore.UpdateInstallation(installation1.Installation)
		require.NoError(t, err)

		err = client.RetryCreateInstallation(installation1.ID)
		require.NoError(t, err)

		installation1, err = client.GetInstallation(installation1.ID, nil)
		require.NoError(t, err)
		require.Equal(t, model.InstallationStateCreationRequested, installation1.State)
	})
}

func TestUpdateInstallation(t *testing.T) {
	logger := testlib.MakeLogger(t)
	sqlStore := store.MakeTestSQLStore(t, logger)
	defer store.CloseConnection(t, sqlStore)

	router := mux.NewRouter()
	api.Register(router, &api.Context{
		Store:      sqlStore,
		Supervisor: &mockSupervisor{},
		Logger:     logger,
	})
	ts := httptest.NewServer(router)
	defer ts.Close()

	client := model.NewClient(ts.URL)

	installation1, err := client.CreateInstallation(&model.CreateInstallationRequest{
		OwnerID:     "owner",
		Version:     "version",
		DNS:         "dns.example.com",
		Affinity:    model.InstallationAffinityIsolated,
		Annotations: []string{"my-annotation"},
	})
	require.NoError(t, err)

	ensureInstallationMatchesRequest := func(t *testing.T, installation *model.Installation, request *model.PatchInstallationRequest) {
		if request.Version != nil {
			require.Equal(t, *request.Version, installation.Version)
		}
		if request.Version != nil {
			require.Equal(t, *request.License, installation.License)
		}
		if request.Size != nil {
			require.Equal(t, *request.Size, installation.Size)
		}
		if request.MattermostEnv != nil {
			require.Equal(t, request.MattermostEnv, installation.MattermostEnv)
		}
	}

	t.Run("unknown installation", func(t *testing.T) {
		upgradeRequest := &model.PatchInstallationRequest{
			Version: sToP(model.NewID()),
			License: sToP(model.NewID()),
		}
		installationReponse, err := client.UpdateInstallation(model.NewID(), upgradeRequest)
		require.EqualError(t, err, "failed with status code 404")
		require.Nil(t, installationReponse)
	})

	t.Run("while locked", func(t *testing.T) {
		installation1.State = model.InstallationStateStable
		err = sqlStore.UpdateInstallation(installation1.Installation)
		require.NoError(t, err)

		lockerID := model.NewID()

		locked, err := sqlStore.LockInstallation(installation1.ID, lockerID)
		require.NoError(t, err)
		require.True(t, locked)
		defer func() {
			unlocked, err := sqlStore.UnlockInstallation(installation1.ID, lockerID, false)
			require.NoError(t, err)
			require.True(t, unlocked)
		}()

		upgradeRequest := &model.PatchInstallationRequest{
			Version: sToP(model.NewID()),
			License: sToP(model.NewID()),
		}
		installationReponse, err := client.UpdateInstallation(installation1.ID, upgradeRequest)
		require.EqualError(t, err, "failed with status code 409")
		require.Nil(t, installationReponse)
	})

	t.Run("while api-security-locked", func(t *testing.T) {
		err = sqlStore.LockInstallationAPI(installation1.ID)
		require.NoError(t, err)

		upgradeRequest := &model.PatchInstallationRequest{
			Version: sToP(model.NewID()),
			License: sToP(model.NewID()),
		}
		installationResponse, err := client.UpdateInstallation(installation1.ID, upgradeRequest)
		require.EqualError(t, err, "failed with status code 403")
		require.Nil(t, installationResponse)

		err = sqlStore.UnlockInstallationAPI(installation1.ID)
		require.NoError(t, err)
	})

	t.Run("while upgrading", func(t *testing.T) {
		installation1.State = model.InstallationStateUpdateRequested
		err = sqlStore.UpdateInstallation(installation1.Installation)
		require.NoError(t, err)

		upgradeRequest := &model.PatchInstallationRequest{
			Version: sToP(model.NewID()),
			License: sToP(model.NewID()),
		}
		installationResponse, err := client.UpdateInstallation(installation1.ID, upgradeRequest)
		require.NoError(t, err)

		installation1, err = client.GetInstallation(installation1.ID, nil)
		require.NoError(t, err)
		require.Equal(t, model.InstallationStateUpdateRequested, installation1.State)
		ensureInstallationMatchesRequest(t, installation1.Installation, upgradeRequest)
		require.Equal(t, "mattermost/mattermost-enterprise-edition", installation1.Image)
		require.Equal(t, installationResponse, installation1)
		assert.True(t, containsAnnotation("my-annotation", installation1.Annotations))
	})

	t.Run("after upgrade failed", func(t *testing.T) {
		installation1.State = model.InstallationStateUpdateFailed
		err = sqlStore.UpdateInstallation(installation1.Installation)
		require.NoError(t, err)

		upgradeRequest := &model.PatchInstallationRequest{
			Version: sToP(model.NewID()),
			License: sToP(model.NewID()),
		}
		installationResponse, err := client.UpdateInstallation(installation1.ID, upgradeRequest)
		require.NoError(t, err)

		installation1, err = client.GetInstallation(installation1.ID, nil)
		require.NoError(t, err)
		require.Equal(t, model.InstallationStateUpdateRequested, installation1.State)
		ensureInstallationMatchesRequest(t, installation1.Installation, upgradeRequest)
		require.Equal(t, installationResponse, installation1)
	})

	t.Run("while stable", func(t *testing.T) {
		installation1.State = model.InstallationStateStable
		err = sqlStore.UpdateInstallation(installation1.Installation)
		require.NoError(t, err)

		upgradeRequest := &model.PatchInstallationRequest{
			Version: sToP(model.NewID()),
			License: sToP(model.NewID()),
		}
		installationResponse, err := client.UpdateInstallation(installation1.ID, upgradeRequest)
		require.NoError(t, err)

		installation1, err = client.GetInstallation(installation1.ID, nil)
		require.NoError(t, err)
		require.Equal(t, model.InstallationStateUpdateRequested, installation1.State)
		ensureInstallationMatchesRequest(t, installation1.Installation, upgradeRequest)
		require.Equal(t, installationResponse, installation1)
	})

	t.Run("after deletion failed", func(t *testing.T) {
		installation1.State = model.InstallationStateDeletionFailed
		err = sqlStore.UpdateInstallation(installation1.Installation)
		require.NoError(t, err)

		upgradeRequest := &model.PatchInstallationRequest{
			Version: sToP(model.NewID()),
			License: sToP(model.NewID()),
		}
		installationResponse, err := client.UpdateInstallation(installation1.ID, upgradeRequest)
		require.EqualError(t, err, "failed with status code 400")
		require.Nil(t, installationResponse)
	})

	t.Run("while deleting", func(t *testing.T) {
		installation1.State = model.InstallationStateDeletionRequested
		err = sqlStore.UpdateInstallation(installation1.Installation)
		require.NoError(t, err)

		upgradeRequest := &model.PatchInstallationRequest{
			Version: sToP(model.NewID()),
			License: sToP(model.NewID()),
		}
		installationResponse, err := client.UpdateInstallation(installation1.ID, upgradeRequest)
		require.EqualError(t, err, "failed with status code 400")
		require.Nil(t, installationResponse)
	})

	t.Run("to version with embedded slash", func(t *testing.T) {
		installation1.State = model.InstallationStateStable
		err = sqlStore.UpdateInstallation(installation1.Installation)
		require.NoError(t, err)

		upgradeRequest := &model.PatchInstallationRequest{
			Version: sToP("mattermost/mattermost-enterprise:v5.12"),
			License: sToP(model.NewID()),
		}
		installationResponse, err := client.UpdateInstallation(installation1.ID, upgradeRequest)
		require.NoError(t, err)

		installation1, err = client.GetInstallation(installation1.ID, nil)
		require.NoError(t, err)
		require.Equal(t, model.InstallationStateUpdateRequested, installation1.State)
		ensureInstallationMatchesRequest(t, installation1.Installation, upgradeRequest)
		require.Equal(t, installationResponse, installation1)
	})

	t.Run("to invalid size", func(t *testing.T) {
		installation1.State = model.InstallationStateStable
		err = sqlStore.UpdateInstallation(installation1.Installation)
		require.NoError(t, err)

		upgradeRequest := &model.PatchInstallationRequest{
			Size: sToP(model.NewID()),
		}
		installationResponse, err := client.UpdateInstallation(installation1.ID, upgradeRequest)
		require.EqualError(t, err, "failed with status code 400")
		require.Nil(t, installationResponse)
	})

	t.Run("installation record updated", func(t *testing.T) {
		installation1.State = model.InstallationStateStable
		err = sqlStore.UpdateInstallation(installation1.Installation)
		require.NoError(t, err)

		upgradeRequest := &model.PatchInstallationRequest{
			Version: sToP(model.NewID()),
			License: sToP(model.NewID()),
			Size:    sToP("miniSingleton"),
		}
		installationResponse, err := client.UpdateInstallation(installation1.ID, upgradeRequest)
		require.NoError(t, err)

		installation1, err = client.GetInstallation(installation1.ID, nil)
		require.NoError(t, err)
		require.Equal(t, model.InstallationStateUpdateRequested, installation1.State)
		ensureInstallationMatchesRequest(t, installation1.Installation, upgradeRequest)
		require.Equal(t, installationResponse, installation1)
	})

	t.Run("empty update request", func(t *testing.T) {
		installation1.State = model.InstallationStateStable
		err = sqlStore.UpdateInstallation(installation1.Installation)
		require.NoError(t, err)

		updateRequest := &model.PatchInstallationRequest{}
		installationResponse, err := client.UpdateInstallation(installation1.ID, updateRequest)
		require.NoError(t, err)

		installation1, err = client.GetInstallation(installation1.ID, nil)
		require.NoError(t, err)
		require.Equal(t, model.InstallationStateStable, installation1.State)
		ensureInstallationMatchesRequest(t, installation1.Installation, updateRequest)
		require.Equal(t, installationResponse, installation1)
	})
}

func TestJoinGroup(t *testing.T) {
	logger := testlib.MakeLogger(t)
	sqlStore := store.MakeTestSQLStore(t, logger)
	defer store.CloseConnection(t, sqlStore)

	router := mux.NewRouter()
	api.Register(router, &api.Context{
		Store:      sqlStore,
		Supervisor: &mockSupervisor{},
		Logger:     logger,
	})
	ts := httptest.NewServer(router)
	defer ts.Close()

	client := model.NewClient(ts.URL)

	group1, err := client.CreateGroup(&model.CreateGroupRequest{
		Name:    "name1",
		Version: "version1",
		Image:   "sample/image1",
	})
	require.NoError(t, err)

	group2, err := client.CreateGroup(&model.CreateGroupRequest{
		Name:    "name2",
		Version: "version2",
		Image:   "sample/image2",
	})
	require.NoError(t, err)

	installation1, err := client.CreateInstallation(&model.CreateInstallationRequest{
		OwnerID:  "owner",
		Version:  "version",
		DNS:      "dns.example.com",
		Affinity: model.InstallationAffinityIsolated,
	})
	require.NoError(t, err)

	t.Run("unknown installation", func(t *testing.T) {
		err := client.JoinGroup(group1.ID, model.NewID())
		require.EqualError(t, err, "failed with status code 404")
	})

	t.Run("unknown group", func(t *testing.T) {
		err := client.JoinGroup(model.NewID(), installation1.ID)
		require.EqualError(t, err, "failed with status code 404")
	})

	t.Run("while locked", func(t *testing.T) {
		lockerID := model.NewID()

		locked, err := sqlStore.LockInstallation(installation1.ID, lockerID)
		require.NoError(t, err)
		require.True(t, locked)
		defer func() {
			unlocked, err := sqlStore.UnlockInstallation(installation1.ID, lockerID, false)
			require.NoError(t, err)
			require.True(t, unlocked)
		}()

		err = client.JoinGroup(group1.ID, installation1.ID)
		require.EqualError(t, err, "failed with status code 409")
	})

	t.Run("while api-security-locked", func(t *testing.T) {
		err = sqlStore.LockInstallationAPI(installation1.ID)
		require.NoError(t, err)

		err = client.JoinGroup(group1.ID, installation1.ID)
		require.EqualError(t, err, "failed with status code 403")

		err = sqlStore.UnlockInstallationAPI(installation1.ID)
		require.NoError(t, err)
	})

	t.Run("to group 1", func(t *testing.T) {
		err = client.JoinGroup(group1.ID, installation1.ID)
		require.NoError(t, err)

		installation1, err = client.GetInstallation(installation1.ID, nil)
		require.NoError(t, err)
		require.NotNil(t, installation1.GroupID)
		require.Equal(t, group1.ID, *installation1.GroupID)
	})

	t.Run("to same group 1", func(t *testing.T) {
		err = client.JoinGroup(group1.ID, installation1.ID)
		require.NoError(t, err)

		installation1, err = client.GetInstallation(installation1.ID, nil)
		require.NoError(t, err)
		require.NotNil(t, installation1.GroupID)
		require.Equal(t, group1.ID, *installation1.GroupID)
	})

	t.Run("to group 2", func(t *testing.T) {
		err = client.JoinGroup(group2.ID, installation1.ID)
		require.NoError(t, err)

		installation1, err = client.GetInstallation(installation1.ID, nil)
		require.NoError(t, err)
		require.NotNil(t, installation1.GroupID)
		require.Equal(t, group2.ID, *installation1.GroupID)
	})

	t.Run("to deleted group", func(t *testing.T) {
		group3, err := client.CreateGroup(&model.CreateGroupRequest{
			Name:    "name3",
			Version: "version3",
			Image:   "sample/image3",
		})
		require.NoError(t, err)

		err = client.DeleteGroup(group3.ID)
		require.NoError(t, err)

		err = client.JoinGroup(group3.ID, installation1.ID)
		require.EqualError(t, err, "failed with status code 400")
	})
}

func TestLeaveGroup(t *testing.T) {
	logger := testlib.MakeLogger(t)
	sqlStore := store.MakeTestSQLStore(t, logger)
	defer store.CloseConnection(t, sqlStore)

	router := mux.NewRouter()
	api.Register(router, &api.Context{
		Store:      sqlStore,
		Supervisor: &mockSupervisor{},
		Logger:     logger,
	})
	ts := httptest.NewServer(router)
	defer ts.Close()

	client := model.NewClient(ts.URL)

	group1, err := client.CreateGroup(&model.CreateGroupRequest{
		Name:    "group-name",
		Version: "group-version",
		Image:   "sample/group-image",
	})
	require.NoError(t, err)

	installation1, err := client.CreateInstallation(&model.CreateInstallationRequest{
		OwnerID:  "owner",
		Version:  "version",
		DNS:      "dns.example.com",
		Affinity: model.InstallationAffinityIsolated,
	})
	require.NoError(t, err)

	installation1.State = model.InstallationStateStable
	err = sqlStore.UpdateInstallation(installation1.Installation)
	require.NoError(t, err)

	err = client.JoinGroup(group1.ID, installation1.ID)
	require.NoError(t, err)

	t.Run("unknown installation", func(t *testing.T) {
		err := client.LeaveGroup(model.NewID(), &model.LeaveGroupRequest{RetainConfig: false})
		require.EqualError(t, err, "failed with status code 404")
	})

	t.Run("while locked", func(t *testing.T) {
		lockerID := model.NewID()

		locked, err := sqlStore.LockInstallation(installation1.ID, lockerID)
		require.NoError(t, err)
		require.True(t, locked)
		defer func() {
			unlocked, err := sqlStore.UnlockInstallation(installation1.ID, lockerID, false)
			require.NoError(t, err)
			require.True(t, unlocked)
		}()

		err = client.LeaveGroup(installation1.ID, &model.LeaveGroupRequest{RetainConfig: true})
		require.EqualError(t, err, "failed with status code 409")
	})

	t.Run("while api-security-locked", func(t *testing.T) {
		err = sqlStore.LockInstallationAPI(installation1.ID)
		require.NoError(t, err)

		err = client.LeaveGroup(installation1.ID, &model.LeaveGroupRequest{RetainConfig: true})
		require.EqualError(t, err, "failed with status code 403")

		err = sqlStore.UnlockInstallationAPI(installation1.ID)
		require.NoError(t, err)
	})

	t.Run("while in group 1", func(t *testing.T) {
		t.Run("don't retain group config", func(t *testing.T) {
			oldVersion := installation1.Version
			oldImage := installation1.Image
			oldEnv := installation1.MattermostEnv

			err = client.LeaveGroup(installation1.ID, &model.LeaveGroupRequest{RetainConfig: false})
			require.NoError(t, err)

			installation1, err = client.GetInstallation(installation1.ID, nil)
			require.NoError(t, err)
			require.Nil(t, installation1.GroupID)
			require.Nil(t, installation1.GroupSequence)
			require.Equal(t, model.InstallationStateUpdateRequested, installation1.State)
			require.Equal(t, oldVersion, installation1.Version)
			require.Equal(t, oldImage, installation1.Image)
			require.Equal(t, oldEnv, installation1.MattermostEnv)
		})

		err = client.JoinGroup(group1.ID, installation1.ID)
		require.NoError(t, err)

		t.Run("retain group config", func(t *testing.T) {
			err = client.LeaveGroup(installation1.ID, &model.LeaveGroupRequest{RetainConfig: true})
			require.NoError(t, err)

			installation1, err = client.GetInstallation(installation1.ID, nil)
			require.NoError(t, err)
			require.Nil(t, installation1.GroupID)
			require.Nil(t, installation1.GroupSequence)
			require.Equal(t, model.InstallationStateUpdateRequested, installation1.State)
			require.Equal(t, group1.Version, installation1.Version)
			require.Equal(t, group1.Image, installation1.Image)
			require.Equal(t, group1.MattermostEnv, installation1.MattermostEnv)
		})
	})

	t.Run("while in no group", func(t *testing.T) {
		err = client.LeaveGroup(installation1.ID, &model.LeaveGroupRequest{RetainConfig: true})
		require.NoError(t, err)

		installation1, err = client.GetInstallation(installation1.ID, nil)
		require.NoError(t, err)
		require.Nil(t, installation1.GroupID)
	})
}

func TestDeleteInstallation(t *testing.T) {
	logger := testlib.MakeLogger(t)
	sqlStore := store.MakeTestSQLStore(t, logger)
	defer store.CloseConnection(t, sqlStore)

	router := mux.NewRouter()
	api.Register(router, &api.Context{
		Store:      sqlStore,
		Supervisor: &mockSupervisor{},
		Logger:     logger,
	})
	ts := httptest.NewServer(router)
	defer ts.Close()

	client := model.NewClient(ts.URL)

	installation1, err := client.CreateInstallation(&model.CreateInstallationRequest{
		OwnerID:  "owner",
		Version:  "version",
		DNS:      "dns.example.com",
		Affinity: model.InstallationAffinityIsolated,
	})
	require.NoError(t, err)

	t.Run("unknown installation", func(t *testing.T) {
		err := client.DeleteInstallation(model.NewID())
		require.EqualError(t, err, "failed with status code 404")
	})

	t.Run("while locked", func(t *testing.T) {
		installation1.State = model.InstallationStateStable
		err = sqlStore.UpdateInstallation(installation1.Installation)
		require.NoError(t, err)

		lockerID := model.NewID()

		locked, err := sqlStore.LockInstallation(installation1.ID, lockerID)
		require.NoError(t, err)
		require.True(t, locked)
		defer func() {
			unlocked, err := sqlStore.UnlockInstallation(installation1.ID, lockerID, false)
			require.NoError(t, err)
			require.True(t, unlocked)

			installation1, err = client.GetInstallation(installation1.ID, nil)
			require.NoError(t, err)
			require.Equal(t, int64(0), installation1.LockAcquiredAt)
		}()

		err = client.DeleteInstallation(installation1.ID)
		require.EqualError(t, err, "failed with status code 409")
	})

	t.Run("while api-security-locked", func(t *testing.T) {
		err = sqlStore.LockInstallationAPI(installation1.ID)
		require.NoError(t, err)

		err = client.DeleteInstallation(installation1.ID)
		require.EqualError(t, err, "failed with status code 403")

		err = sqlStore.UnlockInstallationAPI(installation1.ID)
		require.NoError(t, err)
	})

	t.Run("while backup is running", func(t *testing.T) {
		backup1 := &model.InstallationBackup{InstallationID: installation1.ID, State: model.InstallationBackupStateBackupRequested}
		backup2 := &model.InstallationBackup{InstallationID: installation1.ID, State: model.InstallationBackupStateBackupSucceeded}
		err = sqlStore.CreateInstallationBackup(backup1)
		require.NoError(t, err)
		err = sqlStore.CreateInstallationBackup(backup2)
		require.NoError(t, err)

		err = client.DeleteInstallation(installation1.ID)
		require.EqualError(t, err, "failed with status code 400")

		err = sqlStore.DeleteInstallationBackup(backup1.ID)
		require.NoError(t, err)
	})

	t.Run("while", func(t *testing.T) {
		validDeletingStates := []string{
			model.InstallationStateStable,
			model.InstallationStateCreationRequested,
			model.InstallationStateCreationPreProvisioning,
			model.InstallationStateCreationInProgress,
			model.InstallationStateCreationDNS,
			model.InstallationStateCreationNoCompatibleClusters,
			model.InstallationStateCreationFailed,
			model.InstallationStateUpdateRequested,
			model.InstallationStateUpdateInProgress,
			model.InstallationStateUpdateFailed,
			model.InstallationStateDeletionRequested,
			model.InstallationStateDeletionInProgress,
			model.InstallationStateDeletionFinalCleanup,
			model.InstallationStateDeletionFailed,
		}

		for _, validDeletingState := range validDeletingStates {
			t.Run(validDeletingState, func(t *testing.T) {
				installation1.State = validDeletingState
				err = sqlStore.UpdateInstallation(installation1.Installation)
				require.NoError(t, err)

				err := client.DeleteInstallation(installation1.ID)
				require.NoError(t, err)

				installation1, err = client.GetInstallation(installation1.ID, nil)
				require.NoError(t, err)
				require.Equal(t, model.InstallationStateDeletionRequested, installation1.State)
			})
		}
	})
}

func TestInstallationAnnotations(t *testing.T) {
	logger := testlib.MakeLogger(t)
	sqlStore := store.MakeTestSQLStore(t, logger)
	defer store.CloseConnection(t, sqlStore)

	router := mux.NewRouter()
	api.Register(router, &api.Context{
		Store:      sqlStore,
		Supervisor: &mockSupervisor{},
		Logger:     logger,
	})

	ts := httptest.NewServer(router)
	client := model.NewClient(ts.URL)
	installation, err := client.CreateInstallation(
		&model.CreateInstallationRequest{
			OwnerID:  "owner",
			Version:  "version",
			DNS:      "dns.example.com",
			Affinity: model.InstallationAffinityMultiTenant,
		})
	require.NoError(t, err)

	annotationsRequest := &model.AddAnnotationsRequest{
		Annotations: []string{"my-annotation", "super-awesome123"},
	}

	installation, err = client.AddInstallationAnnotations(installation.ID, annotationsRequest)
	require.NoError(t, err)
	assert.Equal(t, 2, len(installation.Annotations))
	assert.True(t, containsAnnotation("my-annotation", installation.Annotations))
	assert.True(t, containsAnnotation("super-awesome123", installation.Annotations))

	annotationsRequest = &model.AddAnnotationsRequest{
		Annotations: []string{"my-annotation2"},
	}
	installation, err = client.AddInstallationAnnotations(installation.ID, annotationsRequest)
	require.NoError(t, err)
	assert.Equal(t, 3, len(installation.Annotations))

	installation, err = client.GetInstallation(installation.ID, nil)
	require.NoError(t, err)
	assert.Equal(t, 3, len(installation.Annotations))
	assert.True(t, containsAnnotation("my-annotation", installation.Annotations))
	assert.True(t, containsAnnotation("my-annotation2", installation.Annotations))
	assert.True(t, containsAnnotation("super-awesome123", installation.Annotations))

	t.Run("fail to add duplicated annotation", func(t *testing.T) {
		annotationsRequest = &model.AddAnnotationsRequest{
			Annotations: []string{"my-annotation"},
		}
		_, err = client.AddInstallationAnnotations(installation.ID, annotationsRequest)
		require.Error(t, err)
	})

	t.Run("fail to add invalid annotation", func(t *testing.T) {
		annotationsRequest = &model.AddAnnotationsRequest{
			Annotations: []string{"_my-annotation"},
		}
		_, err = client.AddInstallationAnnotations(installation.ID, annotationsRequest)
		require.Error(t, err)
	})

	t.Run("fail to add or delete while api-security-locked", func(t *testing.T) {
		annotationsRequest = &model.AddAnnotationsRequest{
			Annotations: []string{"is-locked"},
		}
		err = sqlStore.LockInstallationAPI(installation.ID)
		require.NoError(t, err)

		_, err = client.AddInstallationAnnotations(installation.ID, annotationsRequest)
		require.Error(t, err)
		err = client.DeleteInstallationAnnotation(installation.ID, "my-annotation2")
		require.Error(t, err)

		err = sqlStore.UnlockInstallationAPI(installation.ID)
		require.NoError(t, err)
	})

	err = client.DeleteInstallationAnnotation(installation.ID, "my-annotation2")
	require.NoError(t, err)

	installation, err = client.GetInstallation(installation.ID, nil)
	require.NoError(t, err)
	assert.Equal(t, 2, len(installation.Annotations))

	t.Run("delete unknown annotation", func(t *testing.T) {
		err = client.DeleteInstallationAnnotation(installation.ID, "unknown")
		require.NoError(t, err)

		installation, err = client.GetInstallation(installation.ID, nil)
		require.NoError(t, err)
		assert.Equal(t, 2, len(installation.Annotations))
	})

	t.Run("fail with 403 if adding annotation when cluster on which it is scheduled does not contain it", func(t *testing.T) {
		annotations := []*model.Annotation{
			{Name: "my-annotation"},
			{Name: "super-awesome123"},
		}

		cluster := &model.Cluster{}
		err = sqlStore.CreateCluster(cluster, annotations)

		clusterInstallation := &model.ClusterInstallation{
			InstallationID: installation.ID,
			ClusterID:      cluster.ID,
		}
		err = sqlStore.CreateClusterInstallation(clusterInstallation)
		require.NoError(t, err)

		_, err = client.AddInstallationAnnotations(installation.ID, &model.AddAnnotationsRequest{
			Annotations: []string{"not-on-a-cluster"},
		})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "403")
	})
}

func dtosToInstallations(dtos []*model.InstallationDTO) []*model.Installation {
	installations := make([]*model.Installation, 0, len(dtos))
	for _, dto := range dtos {
		installations = append(installations, dto.Installation)
	}
	return installations
}
