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
	"github.com/mattermost/mattermost-cloud/internal/model"
	"github.com/mattermost/mattermost-cloud/internal/store"
	"github.com/mattermost/mattermost-cloud/internal/testlib"
	"github.com/stretchr/testify/require"
)

func TestGetInstallations(t *testing.T) {
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

	client := api.NewClient(ts.URL)

	t.Run("unknown installation", func(t *testing.T) {
		installation, err := client.GetInstallation(model.NewID())
		require.NoError(t, err)
		require.Nil(t, installation)

	})

	t.Run("no installations", func(t *testing.T) {
		installations, err := client.GetInstallations(&api.GetInstallationsRequest{
			Page:           0,
			PerPage:        10,
			IncludeDeleted: true,
		})
		require.NoError(t, err)
		require.Empty(t, installations)
	})

	t.Run("parameter handling", func(t *testing.T) {
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

		installation1 := &model.Installation{
			OwnerID:  ownerID1,
			Version:  "version",
			DNS:      "dns.example.com",
			Affinity: model.InstallationAffinityIsolated,
		}
		err := sqlStore.CreateInstallation(installation1)
		require.NoError(t, err)

		time.Sleep(1 * time.Millisecond)

		installation2 := &model.Installation{
			OwnerID:  ownerID2,
			Version:  "version",
			DNS:      "dns2.example.com",
			Affinity: model.InstallationAffinityIsolated,
		}
		err = sqlStore.CreateInstallation(installation2)
		require.NoError(t, err)

		time.Sleep(1 * time.Millisecond)

		installation3 := &model.Installation{
			OwnerID:  ownerID1,
			Version:  "version",
			DNS:      "dns3.example.com",
			Affinity: model.InstallationAffinityIsolated,
		}
		err = sqlStore.CreateInstallation(installation3)
		require.NoError(t, err)

		time.Sleep(1 * time.Millisecond)

		installation4 := &model.Installation{
			OwnerID:  ownerID2,
			Version:  "version",
			DNS:      "dns4.example.com",
			Affinity: model.InstallationAffinityIsolated,
		}
		err = sqlStore.CreateInstallation(installation4)
		require.NoError(t, err)
		err = sqlStore.DeleteInstallation(installation4.ID)
		require.NoError(t, err)
		installation4, err = client.GetInstallation(installation4.ID)
		require.NoError(t, err)

		t.Run("get installation", func(t *testing.T) {
			t.Run("installation 1", func(t *testing.T) {
				installation, err := client.GetInstallation(installation1.ID)
				require.NoError(t, err)
				require.Equal(t, installation1, installation)
			})

			t.Run("get deleted installation", func(t *testing.T) {
				installation, err := client.GetInstallation(installation4.ID)
				require.NoError(t, err)
				require.Equal(t, installation4, installation)
			})
		})

		t.Run("get installations", func(t *testing.T) {
			testCases := []struct {
				Description             string
				GetInstallationsRequest *api.GetInstallationsRequest
				Expected                []*model.Installation
			}{
				{
					"page 0, perPage 2, exclude deleted",
					&api.GetInstallationsRequest{
						Page:           0,
						PerPage:        2,
						IncludeDeleted: false,
					},
					[]*model.Installation{installation1, installation2},
				},

				{
					"page 1, perPage 2, exclude deleted",
					&api.GetInstallationsRequest{
						Page:           1,
						PerPage:        2,
						IncludeDeleted: false,
					},
					[]*model.Installation{installation3},
				},

				{
					"page 0, perPage 2, include deleted",
					&api.GetInstallationsRequest{
						Page:           0,
						PerPage:        2,
						IncludeDeleted: true,
					},
					[]*model.Installation{installation1, installation2},
				},

				{
					"page 1, perPage 2, include deleted",
					&api.GetInstallationsRequest{
						Page:           1,
						PerPage:        2,
						IncludeDeleted: true,
					},
					[]*model.Installation{installation3, installation4},
				},
				{
					"filter by owner",
					&api.GetInstallationsRequest{
						Page:           0,
						PerPage:        100,
						OwnerID:        ownerID1,
						IncludeDeleted: false,
					},
					[]*model.Installation{installation1, installation3},
				},
			}

			for _, testCase := range testCases {
				t.Run(testCase.Description, func(t *testing.T) {
					installations, err := client.GetInstallations(testCase.GetInstallationsRequest)
					require.NoError(t, err)
					require.Equal(t, testCase.Expected, installations)
				})
			}
		})
	})
}

func TestCreateInstallation(t *testing.T) {
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

	client := api.NewClient(ts.URL)

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
		_, err := client.CreateInstallation(&api.CreateInstallationRequest{
			Version:  "version",
			DNS:      "dns.example.com",
			Affinity: model.InstallationAffinityIsolated,
		})
		require.EqualError(t, err, "failed with status code 400")
	})

	t.Run("missing dns", func(t *testing.T) {
		_, err := client.CreateInstallation(&api.CreateInstallationRequest{
			OwnerID:  "owner",
			Version:  "version",
			Affinity: model.InstallationAffinityIsolated,
		})
		require.EqualError(t, err, "failed with status code 400")
	})

	t.Run("invalid dns", func(t *testing.T) {
		_, err := client.CreateInstallation(&api.CreateInstallationRequest{
			OwnerID:  "owner",
			Version:  "version",
			DNS:      string([]byte{0x7f}),
			Affinity: model.InstallationAffinityIsolated,
		})
		require.EqualError(t, err, "failed with status code 400")
	})

	t.Run("invalid affinity", func(t *testing.T) {
		_, err := client.CreateInstallation(&api.CreateInstallationRequest{
			OwnerID:  "owner",
			Version:  "version",
			DNS:      "dns.example.com",
			Affinity: "invalid",
		})
		require.EqualError(t, err, "failed with status code 400")
	})

	t.Run("valid", func(t *testing.T) {
		installation, err := client.CreateInstallation(&api.CreateInstallationRequest{
			OwnerID:  "owner",
			Version:  "version",
			DNS:      "dns.example.com",
			Affinity: model.InstallationAffinityIsolated,
		})
		require.NoError(t, err)
		require.Equal(t, "owner", installation.OwnerID)
		require.Equal(t, "version", installation.Version)
		require.Equal(t, "dns.example.com", installation.DNS)
		require.Equal(t, model.InstallationAffinityIsolated, installation.Affinity)
		require.Equal(t, model.InstallationStateCreationRequested, installation.State)
		require.Empty(t, installation.LockAcquiredBy)
		require.EqualValues(t, 0, installation.LockAcquiredAt)
		require.NotEqual(t, 0, installation.CreateAt)
		require.EqualValues(t, 0, installation.DeleteAt)
	})
}

func TestRetryCreateInstallation(t *testing.T) {
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

	client := api.NewClient(ts.URL)

	installation1, err := client.CreateInstallation(&api.CreateInstallationRequest{
		OwnerID:  "owner",
		Version:  "version",
		DNS:      "dns.example.com",
		Affinity: model.InstallationAffinityIsolated,
	})
	require.NoError(t, err)

	t.Run("unknown installation", func(t *testing.T) {
		err := client.RetryCreateInstallation(model.NewID())
		require.EqualError(t, err, "failed with status code 404")
	})

	t.Run("while locked", func(t *testing.T) {
		installation1.State = model.InstallationStateStable
		err = sqlStore.UpdateInstallation(installation1)
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
		err = sqlStore.UpdateInstallation(installation1)
		require.NoError(t, err)

		err = client.RetryCreateInstallation(installation1.ID)
		require.NoError(t, err)

		installation1, err = client.GetInstallation(installation1.ID)
		require.NoError(t, err)
		require.Equal(t, model.InstallationStateCreationRequested, installation1.State)
	})

	t.Run("while stable", func(t *testing.T) {
		installation1.State = model.InstallationStateStable
		err = sqlStore.UpdateInstallation(installation1)
		require.NoError(t, err)

		err = client.RetryCreateInstallation(installation1.ID)
		require.EqualError(t, err, "failed with status code 400")
	})

	t.Run("while creation failed", func(t *testing.T) {
		installation1.State = model.InstallationStateCreationFailed
		err = sqlStore.UpdateInstallation(installation1)
		require.NoError(t, err)

		err = client.RetryCreateInstallation(installation1.ID)
		require.NoError(t, err)

		installation1, err = client.GetInstallation(installation1.ID)
		require.NoError(t, err)
		require.Equal(t, model.InstallationStateCreationRequested, installation1.State)
	})
}

func TestUpgradeInstallation(t *testing.T) {
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

	client := api.NewClient(ts.URL)

	installation1, err := client.CreateInstallation(&api.CreateInstallationRequest{
		OwnerID:  "owner",
		Version:  "version",
		DNS:      "dns.example.com",
		Affinity: model.InstallationAffinityIsolated,
	})
	require.NoError(t, err)

	t.Run("unknown installation", func(t *testing.T) {
		err := client.UpgradeInstallation(model.NewID(), "latest")
		require.EqualError(t, err, "failed with status code 404")
	})

	t.Run("while locked", func(t *testing.T) {
		installation1.State = model.InstallationStateStable
		err = sqlStore.UpdateInstallation(installation1)
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

		err = client.UpgradeInstallation(installation1.ID, "latest")
		require.EqualError(t, err, "failed with status code 409")
	})

	t.Run("while upgrading", func(t *testing.T) {
		installation1.State = model.InstallationStateUpgradeRequested
		err = sqlStore.UpdateInstallation(installation1)
		require.NoError(t, err)

		err = client.UpgradeInstallation(installation1.ID, "latest")
		require.NoError(t, err)

		installation1, err = client.GetInstallation(installation1.ID)
		require.NoError(t, err)
		require.Equal(t, model.InstallationStateUpgradeRequested, installation1.State)
	})

	t.Run("after upgrade failed", func(t *testing.T) {
		installation1.State = model.InstallationStateUpgradeFailed
		err = sqlStore.UpdateInstallation(installation1)
		require.NoError(t, err)

		err = client.UpgradeInstallation(installation1.ID, "latest")
		require.NoError(t, err)

		installation1, err = client.GetInstallation(installation1.ID)
		require.NoError(t, err)
		require.Equal(t, model.InstallationStateUpgradeRequested, installation1.State)
	})

	t.Run("while stable", func(t *testing.T) {
		installation1.State = model.InstallationStateStable
		err = sqlStore.UpdateInstallation(installation1)
		require.NoError(t, err)

		err = client.UpgradeInstallation(installation1.ID, "latest")
		require.NoError(t, err)

		installation1, err = client.GetInstallation(installation1.ID)
		require.NoError(t, err)
		require.Equal(t, model.InstallationStateUpgradeRequested, installation1.State)
	})

	t.Run("after deletion failed", func(t *testing.T) {
		installation1.State = model.InstallationStateDeletionFailed
		err = sqlStore.UpdateInstallation(installation1)
		require.NoError(t, err)

		err := client.DeleteInstallation(installation1.ID)
		require.NoError(t, err)

		installation1, err = client.GetInstallation(installation1.ID)
		require.NoError(t, err)
		require.Equal(t, model.InstallationStateDeletionRequested, installation1.State)
	})

	t.Run("while deleting", func(t *testing.T) {
		installation1.State = model.InstallationStateDeletionRequested
		err = sqlStore.UpdateInstallation(installation1)
		require.NoError(t, err)

		err = client.UpgradeInstallation(installation1.ID, "latest")
		require.EqualError(t, err, "failed with status code 400")
	})
}

func TestJoinGroup(t *testing.T) {
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

	client := api.NewClient(ts.URL)

	group1, err := client.CreateGroup(&api.CreateGroupRequest{
		Name:    "name1",
		Version: "version1",
	})
	require.NoError(t, err)

	group2, err := client.CreateGroup(&api.CreateGroupRequest{
		Name:    "name2",
		Version: "version2",
	})
	require.NoError(t, err)

	installation1, err := client.CreateInstallation(&api.CreateInstallationRequest{
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

	t.Run("to group 1", func(t *testing.T) {
		err = client.JoinGroup(group1.ID, installation1.ID)
		require.NoError(t, err)

		installation1, err = client.GetInstallation(installation1.ID)
		require.NoError(t, err)
		require.NotNil(t, installation1.GroupID)
		require.Equal(t, group1.ID, *installation1.GroupID)
	})

	t.Run("to same group 1", func(t *testing.T) {
		err = client.JoinGroup(group1.ID, installation1.ID)
		require.NoError(t, err)

		installation1, err = client.GetInstallation(installation1.ID)
		require.NoError(t, err)
		require.NotNil(t, installation1.GroupID)
		require.Equal(t, group1.ID, *installation1.GroupID)
	})

	t.Run("to group 2", func(t *testing.T) {
		err = client.JoinGroup(group2.ID, installation1.ID)
		require.NoError(t, err)

		installation1, err = client.GetInstallation(installation1.ID)
		require.NoError(t, err)
		require.NotNil(t, installation1.GroupID)
		require.Equal(t, group2.ID, *installation1.GroupID)
	})
}

func TestLeaveGroup(t *testing.T) {
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

	client := api.NewClient(ts.URL)

	group1, err := client.CreateGroup(&api.CreateGroupRequest{
		Name:    "name1",
		Version: "version1",
	})
	require.NoError(t, err)

	installation1, err := client.CreateInstallation(&api.CreateInstallationRequest{
		OwnerID:  "owner",
		Version:  "version",
		DNS:      "dns.example.com",
		Affinity: model.InstallationAffinityIsolated,
	})
	require.NoError(t, err)

	err = client.JoinGroup(group1.ID, installation1.ID)
	require.NoError(t, err)

	t.Run("unknown installation", func(t *testing.T) {
		err := client.LeaveGroup(model.NewID())
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

		err = client.LeaveGroup(installation1.ID)
		require.EqualError(t, err, "failed with status code 409")
	})

	t.Run("while in group 1", func(t *testing.T) {
		err = client.LeaveGroup(installation1.ID)
		require.NoError(t, err)

		installation1, err = client.GetInstallation(installation1.ID)
		require.NoError(t, err)
		require.Nil(t, installation1.GroupID)
	})

	t.Run("while in no group", func(t *testing.T) {
		err = client.LeaveGroup(installation1.ID)
		require.NoError(t, err)

		installation1, err = client.GetInstallation(installation1.ID)
		require.NoError(t, err)
		require.Nil(t, installation1.GroupID)
	})
}

func TestDeleteInstallation(t *testing.T) {
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

	client := api.NewClient(ts.URL)

	installation1, err := client.CreateInstallation(&api.CreateInstallationRequest{
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
		err = sqlStore.UpdateInstallation(installation1)
		require.NoError(t, err)

		lockerID := model.NewID()

		locked, err := sqlStore.LockInstallation(installation1.ID, lockerID)
		require.NoError(t, err)
		require.True(t, locked)
		defer func() {
			unlocked, err := sqlStore.UnlockInstallation(installation1.ID, lockerID, false)
			require.NoError(t, err)
			require.True(t, unlocked)

			installation1, err = client.GetInstallation(installation1.ID)
			require.NoError(t, err)
			require.Equal(t, int64(0), installation1.LockAcquiredAt)
		}()

		err = client.DeleteInstallation(installation1.ID)
		require.EqualError(t, err, "failed with status code 409")
	})

	t.Run("while", func(t *testing.T) {
		validDeletingStates := []string{
			model.InstallationStateStable,
			model.InstallationStateCreationRequested,
			model.InstallationStateCreationFailed,
			model.InstallationStateDeletionRequested,
			model.InstallationStateDeletionInProgress,
			model.InstallationStateDeletionFailed,
		}

		for _, validDeletingState := range validDeletingStates {
			t.Run(validDeletingState, func(t *testing.T) {
				installation1.State = validDeletingState
				err = sqlStore.UpdateInstallation(installation1)
				require.NoError(t, err)

				err := client.DeleteInstallation(installation1.ID)
				require.NoError(t, err)

				installation1, err = client.GetInstallation(installation1.ID)
				require.NoError(t, err)
				require.Equal(t, model.InstallationStateDeletionRequested, installation1.State)
			})
		}
	})
}
