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

func TestGetGroups(t *testing.T) {
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

	t.Run("unknown group", func(t *testing.T) {
		group, err := client.GetGroup(model.NewID())
		require.NoError(t, err)
		require.Nil(t, group)

	})

	t.Run("no groups", func(t *testing.T) {
		groups, err := client.GetGroups(&model.GetGroupsRequest{
			Page:           0,
			PerPage:        10,
			IncludeDeleted: true,
		})
		require.NoError(t, err)
		require.Empty(t, groups)
	})

	t.Run("parameter handling", func(t *testing.T) {
		t.Run("invalid page", func(t *testing.T) {
			resp, err := http.Get(fmt.Sprintf("%s/api/groups?page=invalid&per_page=100", ts.URL))
			require.NoError(t, err)
			require.Equal(t, http.StatusBadRequest, resp.StatusCode)
		})

		t.Run("invalid perPage", func(t *testing.T) {
			resp, err := http.Get(fmt.Sprintf("%s/api/groups?page=0&per_page=invalid", ts.URL))
			require.NoError(t, err)
			require.Equal(t, http.StatusBadRequest, resp.StatusCode)
		})

		t.Run("no paging parameters", func(t *testing.T) {
			resp, err := http.Get(fmt.Sprintf("%s/api/groups", ts.URL))
			require.NoError(t, err)
			require.Equal(t, http.StatusOK, resp.StatusCode)
		})

		t.Run("missing page", func(t *testing.T) {
			resp, err := http.Get(fmt.Sprintf("%s/api/groups?per_page=100", ts.URL))
			require.NoError(t, err)
			require.Equal(t, http.StatusOK, resp.StatusCode)
		})

		t.Run("missing perPage", func(t *testing.T) {
			resp, err := http.Get(fmt.Sprintf("%s/api/groups?page=1", ts.URL))
			require.NoError(t, err)
			require.Equal(t, http.StatusOK, resp.StatusCode)
		})
	})

	t.Run("results", func(t *testing.T) {
		group1 := &model.Group{
			Name:        "group1",
			Description: "This is group 1",
			Version:     "version1",
		}
		err := sqlStore.CreateGroup(group1)
		require.NoError(t, err)

		time.Sleep(1 * time.Millisecond)

		group2 := &model.Group{
			Name:        "group2",
			Description: "This is group 2",
			Version:     "version2",
		}
		err = sqlStore.CreateGroup(group2)
		require.NoError(t, err)

		time.Sleep(1 * time.Millisecond)

		group3 := &model.Group{
			Name:        "group3",
			Description: "This is group 3",
			Version:     "version3",
		}
		err = sqlStore.CreateGroup(group3)
		require.NoError(t, err)

		time.Sleep(1 * time.Millisecond)

		group4 := &model.Group{
			Name:        "group4",
			Description: "This is group 4",
			Version:     "version4",
		}
		err = sqlStore.CreateGroup(group4)
		require.NoError(t, err)
		err = sqlStore.DeleteGroup(group4.ID)
		require.NoError(t, err)
		group4, err = client.GetGroup(group4.ID)
		require.NoError(t, err)

		t.Run("get group", func(t *testing.T) {
			t.Run("group 1", func(t *testing.T) {
				group, err := client.GetGroup(group1.ID)
				require.NoError(t, err)
				require.Equal(t, group1, group)
			})

			t.Run("get deleted group", func(t *testing.T) {
				group, err := client.GetGroup(group4.ID)
				require.NoError(t, err)
				require.Equal(t, group4, group)
			})
		})

		t.Run("get groups", func(t *testing.T) {
			testCases := []struct {
				Description      string
				GetGroupsRequest *model.GetGroupsRequest
				Expected         []*model.Group
			}{
				{
					"page 0, perPage 2, exclude deleted",
					&model.GetGroupsRequest{
						Page:           0,
						PerPage:        2,
						IncludeDeleted: false,
					},
					[]*model.Group{group1, group2},
				},

				{
					"page 1, perPage 2, exclude deleted",
					&model.GetGroupsRequest{
						Page:           1,
						PerPage:        2,
						IncludeDeleted: false,
					},
					[]*model.Group{group3},
				},

				{
					"page 0, perPage 2, include deleted",
					&model.GetGroupsRequest{
						Page:           0,
						PerPage:        2,
						IncludeDeleted: true,
					},
					[]*model.Group{group1, group2},
				},

				{
					"page 1, perPage 2, include deleted",
					&model.GetGroupsRequest{
						Page:           1,
						PerPage:        2,
						IncludeDeleted: true,
					},
					[]*model.Group{group3, group4},
				},
			}

			for _, testCase := range testCases {
				t.Run(testCase.Description, func(t *testing.T) {
					groups, err := client.GetGroups(testCase.GetGroupsRequest)
					require.NoError(t, err)
					require.Equal(t, testCase.Expected, groups)
				})
			}
		})
	})
}

func TestCreateGroup(t *testing.T) {
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

	t.Run("invalid payload", func(t *testing.T) {
		resp, err := http.Post(fmt.Sprintf("%s/api/groups", ts.URL), "application/json", bytes.NewReader([]byte("invalid")))
		require.NoError(t, err)
		require.Equal(t, http.StatusBadRequest, resp.StatusCode)
	})

	t.Run("empty payload", func(t *testing.T) {
		resp, err := http.Post(fmt.Sprintf("%s/api/groups", ts.URL), "application/json", bytes.NewReader([]byte("")))
		require.NoError(t, err)
		require.Equal(t, http.StatusBadRequest, resp.StatusCode)
	})

	t.Run("missing name", func(t *testing.T) {
		_, err := client.CreateGroup(&model.CreateGroupRequest{
			Description: "description",
			Version:     "version",
		})
		require.EqualError(t, err, "failed with status code 400")
	})

	mattermostEnvFooBar := model.EnvVarMap{"foo": model.EnvVar{Value: "bar"}}
	t.Run("valid", func(t *testing.T) {
		group, err := client.CreateGroup(&model.CreateGroupRequest{
			Name:          "name",
			Description:   "description",
			Version:       "version",
			Image:         "sample/image",
			MaxRolling:    2,
			MattermostEnv: mattermostEnvFooBar,
		})
		require.NoError(t, err)
		require.Equal(t, "name", group.Name)
		require.Equal(t, "description", group.Description)
		require.Equal(t, "version", group.Version)
		require.Equal(t, "sample/image", group.Image)
		require.Equal(t, int64(2), group.MaxRolling)
		require.EqualValues(t, group.MattermostEnv, mattermostEnvFooBar)
		require.NotEqual(t, 0, group.CreateAt)
		require.EqualValues(t, 0, group.DeleteAt)
	})
}

func TestUpdateGroup(t *testing.T) {
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

	mattermostEnvFooBar := model.EnvVarMap{"foo": model.EnvVar{Value: "bar"}}

	group1, err := client.CreateGroup(&model.CreateGroupRequest{
		Name:          "name",
		Description:   "description",
		Version:       "version",
		Image:         "sample/image",
		MattermostEnv: mattermostEnvFooBar,
	})
	require.NoError(t, err)

	t.Run("invalid payload", func(t *testing.T) {
		httpRequest, err := http.NewRequest(http.MethodPut, fmt.Sprintf("%s/api/group/%s", ts.URL, group1.ID), bytes.NewReader([]byte("invalid")))
		require.NoError(t, err)

		resp, err := http.DefaultClient.Do(httpRequest)
		require.NoError(t, err)
		require.Equal(t, http.StatusBadRequest, resp.StatusCode)
	})

	t.Run("empty payload", func(t *testing.T) {
		httpRequest, err := http.NewRequest(http.MethodPut, fmt.Sprintf("%s/api/group/%s", ts.URL, group1.ID), bytes.NewReader([]byte("")))
		require.NoError(t, err)

		resp, err := http.DefaultClient.Do(httpRequest)
		require.NoError(t, err)
		require.Equal(t, http.StatusOK, resp.StatusCode)
	})

	t.Run("unknown group", func(t *testing.T) {
		group, err := client.UpdateGroup(&model.PatchGroupRequest{ID: model.NewID()})
		require.EqualError(t, err, "failed with status code 404")
		require.Nil(t, group)
	})

	t.Run("while api-security-locked", func(t *testing.T) {
		err = sqlStore.LockGroupAPI(group1.ID)
		require.NoError(t, err)

		groupResp, err := client.UpdateGroup(&model.PatchGroupRequest{ID: group1.ID})
		require.EqualError(t, err, "failed with status code 403")
		assert.Nil(t, groupResp)

		err = sqlStore.UnlockGroupAPI(group1.ID)
		require.NoError(t, err)
	})

	t.Run("only sequence updated", func(t *testing.T) {
		group1, err = client.GetGroup(group1.ID)
		require.NoError(t, err)
		oldSequence := group1.Sequence

		updateResponseGroup, err := client.UpdateGroup(&model.PatchGroupRequest{
			ID:                  group1.ID,
			ForceSequenceUpdate: true,
		})
		require.NoError(t, err)

		group1, err = client.GetGroup(group1.ID)
		require.NoError(t, err)
		require.Equal(t, "name", group1.Name)
		require.Equal(t, "description", group1.Description)
		require.Equal(t, "version", group1.Version)
		require.EqualValues(t, group1.MattermostEnv, mattermostEnvFooBar)
		require.Equal(t, updateResponseGroup, group1)
		require.Equal(t, oldSequence+1, group1.Sequence)
	})

	t.Run("partial update", func(t *testing.T) {
		updateResponseGroup, err := client.UpdateGroup(&model.PatchGroupRequest{
			ID:      group1.ID,
			Version: sToP("version2"),
		})
		require.NoError(t, err)

		group1, err = client.GetGroup(group1.ID)
		require.NoError(t, err)
		require.Equal(t, "name", group1.Name)
		require.Equal(t, "description", group1.Description)
		require.Equal(t, "version2", group1.Version)
		require.EqualValues(t, group1.MattermostEnv, mattermostEnvFooBar)
		require.Equal(t, updateResponseGroup, group1)
	})

	mattermostEnvBarBaz := model.EnvVarMap{"bar": model.EnvVar{Value: "baz"}}
	t.Run("full update", func(t *testing.T) {
		updateResponseGroup, err := client.UpdateGroup(&model.PatchGroupRequest{
			ID:            group1.ID,
			Name:          sToP("name2"),
			Description:   sToP("description2"),
			Version:       sToP("version2"),
			MattermostEnv: mattermostEnvBarBaz,
		})
		require.NoError(t, err)

		group1, err = client.GetGroup(group1.ID)
		require.NoError(t, err)
		require.Equal(t, "name2", group1.Name)
		require.Equal(t, "description2", group1.Description)
		require.Equal(t, "version2", group1.Version)
		require.True(t, mattermostEnvFooBar.ClearOrPatch(&mattermostEnvBarBaz))
		require.Equal(t, group1.MattermostEnv, mattermostEnvFooBar)
		require.Equal(t, updateResponseGroup, group1)
	})
}

func TestDeleteGroup(t *testing.T) {
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

	group1, err := client.CreateGroup(&model.CreateGroupRequest{
		Name:        "name",
		Description: "description",
		Version:     "version",
		Image:       "sample/image",
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

	t.Run("join group", func(t *testing.T) {
		err = client.JoinGroup(group1.ID, installation1.ID)
		require.NoError(t, err)

		installation1, err = client.GetInstallation(installation1.ID, nil)
		require.NoError(t, err)
		require.NotNil(t, installation1.GroupID)
		require.Equal(t, group1.ID, *installation1.GroupID)
	})

	t.Run("unknown group", func(t *testing.T) {
		err = client.DeleteGroup(model.NewID())
		require.EqualError(t, err, "failed with status code 404")
	})

	t.Run("while api-security-locked", func(t *testing.T) {
		err = sqlStore.LockGroupAPI(group1.ID)
		require.NoError(t, err)

		err := client.DeleteGroup(group1.ID)
		require.EqualError(t, err, "failed with status code 403")

		err = sqlStore.UnlockGroupAPI(group1.ID)
		require.NoError(t, err)
	})

	t.Run("installations in group", func(t *testing.T) {
		err = client.DeleteGroup(group1.ID)
		require.Error(t, err)

		group1, err = client.GetGroup(group1.ID)
		require.NoError(t, err)
		require.Equal(t, int64(0), group1.DeleteAt)
	})

	t.Run("success", func(t *testing.T) {
		err = client.LeaveGroup(installation1.ID, &model.LeaveGroupRequest{RetainConfig: true})
		require.NoError(t, err)

		err = client.DeleteGroup(group1.ID)
		require.NoError(t, err)

		group1, err = client.GetGroup(group1.ID)
		require.NoError(t, err)
		require.NotEqual(t, 0, group1.DeleteAt)
	})

	t.Run("delete again", func(t *testing.T) {
		err = client.DeleteGroup(group1.ID)
		require.NoError(t, err)
		require.NotEqual(t, 0, group1.DeleteAt)
	})
}

func TestGroupStatus(t *testing.T) {
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
	installationsCreated := 0

	// helper function for creating installations
	newInstallation := func(groupId *string, sequence *int64, state string) *model.Installation {
		installationsCreated += 1 // Increment to generate unique DNS for each installation

		createRequest := &model.CreateInstallationRequest{
			OwnerID:  "owner",
			Version:  "version",
			DNS:      fmt.Sprintf("dns%d.example.com", installationsCreated),
			Affinity: model.InstallationAffinityIsolated,
		}
		if groupId != nil {
			createRequest.GroupID = *groupId
		}

		installation, err := client.CreateInstallation(createRequest)
		require.NoError(t, err)

		installation.State = state
		installation.GroupSequence = sequence
		err = sqlStore.UpdateInstallation(installation.Installation)
		require.NoError(t, err)

		return installation.Installation
	}

	group, err := client.CreateGroup(&model.CreateGroupRequest{
		Name:        "group1",
		Description: "description",
		Version:     "version",
		Image:       "sample/image",
	})
	require.NoError(t, err)

	t.Run("empty group", func(t *testing.T) {
		expectedStatus := &model.GroupStatus{
			InstallationsTotal:          0,
			InstallationsUpdated:        0,
			InstallationsUpdating:       0,
			InstallationsAwaitingUpdate: 0,
		}
		groupStatus, err := client.GetGroupStatus(group.ID)
		require.NoError(t, err)
		assert.Equal(t, expectedStatus, groupStatus)
	})

	t.Run("ignore different groups", func(t *testing.T) {
		expectedStatus := &model.GroupStatus{
			InstallationsTotal:          0,
			InstallationsUpdated:        0,
			InstallationsUpdating:       0,
			InstallationsAwaitingUpdate: 0,
		}
		ignoredGroup, err := client.CreateGroup(&model.CreateGroupRequest{
			Name:        "group2",
			Description: "description",
			Version:     "version",
			Image:       "sample/image",
		})
		require.NoError(t, err)

		newInstallation(nil, nil, model.InstallationStateStable)
		newInstallation(&ignoredGroup.ID, nil, model.InstallationStateStable)

		groupStatus, err := client.GetGroupStatus(group.ID)
		require.NoError(t, err)
		assert.Equal(t, expectedStatus, groupStatus)
	})

	t.Run("count installations", func(t *testing.T) {
		expectedStatus := &model.GroupStatus{
			InstallationsTotal:          6,
			InstallationsUpdated:        2,
			InstallationsUpdating:       3,
			InstallationsAwaitingUpdate: 1,
		}
		var differentSequence int64 = -1

		// rolled out stable
		newInstallation(&group.ID, &group.Sequence, model.InstallationStateStable)
		newInstallation(&group.ID, &group.Sequence, model.InstallationStateStable)
		// rolled out not stable
		newInstallation(&group.ID, &group.Sequence, model.InstallationStateUpdateInProgress)
		newInstallation(&group.ID, &group.Sequence, model.InstallationStateCreationDNS)
		// not rolled out stable
		newInstallation(&group.ID, &differentSequence, model.InstallationStateStable)
		// not rolled out unstable
		newInstallation(&group.ID, &differentSequence, model.InstallationStateUpdateInProgress)

		groupStatus, err := client.GetGroupStatus(group.ID)
		require.NoError(t, err)
		assert.Equal(t, expectedStatus, groupStatus)
	})

	t.Run("unknown group", func(t *testing.T) {
		groupStatus, err := client.GetGroupStatus(model.NewID())
		require.Nil(t, groupStatus)
		require.Nil(t, err)
	})
}

func TestGroupsStatus(t *testing.T) {
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
	installationsCreated := 0

	// helper function for creating installations
	newInstallation := func(groupId *string, sequence *int64, state string) *model.Installation {
		installationsCreated += 1 // Increment to generate unique DNS for each installation

		createRequest := &model.CreateInstallationRequest{
			OwnerID:  "owner",
			Version:  "version",
			DNS:      fmt.Sprintf("dns%d.example.com", installationsCreated),
			Affinity: model.InstallationAffinityIsolated,
		}
		if groupId != nil {
			createRequest.GroupID = *groupId
		}

		installation, err := client.CreateInstallation(createRequest)
		require.NoError(t, err)

		installation.State = state
		installation.GroupSequence = sequence
		err = sqlStore.UpdateInstallation(installation.Installation)
		require.NoError(t, err)

		return installation.Installation
	}

	group, err := client.CreateGroup(&model.CreateGroupRequest{
		Name:        "group1",
		Description: "description",
		Version:     "version",
		Image:       "sample/image",
	})
	require.NoError(t, err)

	group2, err := client.CreateGroup(&model.CreateGroupRequest{
		Name:        "group2",
		Description: "description",
		Version:     "version",
		Image:       "sample/image",
	})
	require.NoError(t, err)

	t.Run("empty groups", func(t *testing.T) {
		expectedStatus := &[]model.GroupsStatus{
			{
				ID: group.ID,
				Status: model.GroupStatus{
					InstallationsTotal:          0,
					InstallationsUpdated:        0,
					InstallationsUpdating:       0,
					InstallationsAwaitingUpdate: 0,
				},
			},
			{
				ID: group2.ID,
				Status: model.GroupStatus{
					InstallationsTotal:          0,
					InstallationsUpdated:        0,
					InstallationsUpdating:       0,
					InstallationsAwaitingUpdate: 0,
				},
			},
		}
		groupsStatus, err := client.GetGroupsStatus()
		require.NoError(t, err)
		assert.Equal(t, expectedStatus, groupsStatus)
	})

	t.Run("count installations", func(t *testing.T) {
		expectedStatus := &[]model.GroupsStatus{
			{
				ID: group.ID,
				Status: model.GroupStatus{
					InstallationsTotal:          6,
					InstallationsUpdated:        2,
					InstallationsUpdating:       3,
					InstallationsAwaitingUpdate: 1,
				},
			},
			{
				ID: group2.ID,
				Status: model.GroupStatus{
					InstallationsTotal:          0,
					InstallationsUpdated:        0,
					InstallationsUpdating:       0,
					InstallationsAwaitingUpdate: 0,
				},
			},
		}
		var differentSequence int64 = -1

		// rolled out stable
		newInstallation(&group.ID, &group.Sequence, model.InstallationStateStable)
		newInstallation(&group.ID, &group.Sequence, model.InstallationStateStable)
		// rolled out not stable
		newInstallation(&group.ID, &group.Sequence, model.InstallationStateUpdateInProgress)
		newInstallation(&group.ID, &group.Sequence, model.InstallationStateCreationDNS)
		// not rolled out stable
		newInstallation(&group.ID, &differentSequence, model.InstallationStateStable)
		// not rolled out unstable
		newInstallation(&group.ID, &differentSequence, model.InstallationStateUpdateInProgress)

		groupsStatus, err := client.GetGroupsStatus()
		require.NoError(t, err)
		assert.Equal(t, expectedStatus, groupsStatus)
	})

}
