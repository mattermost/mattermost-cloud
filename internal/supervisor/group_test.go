package supervisor_test

import (
	"testing"

	"github.com/mattermost/mattermost-cloud/internal/store"
	"github.com/mattermost/mattermost-cloud/internal/supervisor"
	"github.com/mattermost/mattermost-cloud/internal/testlib"
	"github.com/mattermost/mattermost-cloud/model"
	"github.com/stretchr/testify/require"
)

type mockGroupStore struct {
	Group                     *model.Group
	UnlockedGroupsPendingWork []*model.Group
	GroupRollingMetadata      *store.GroupRollingMetadata

	Installation *model.Installation

	UnlockChan              chan interface{}
	UpdateInstallationCalls int
}

func (s *mockGroupStore) GetUnlockedGroupsPendingWork() ([]*model.Group, error) {
	return s.UnlockedGroupsPendingWork, nil
}

func (s *mockGroupStore) GetGroupRollingMetadata(groupID string) (*store.GroupRollingMetadata, error) {
	return s.GroupRollingMetadata, nil
}

func (s *mockGroupStore) LockGroup(groupID, lockerID string) (bool, error) {
	return true, nil
}

func (s *mockGroupStore) UnlockGroup(groupID, lockerID string, force bool) (bool, error) {
	if s.UnlockChan != nil {
		close(s.UnlockChan)
	}
	return true, nil
}

func (s *mockGroupStore) GetInstallation(installationID string, includeGroupConfig, includeGroupConfigOverrides bool) (*model.Installation, error) {
	return s.Installation, nil
}

func (s *mockGroupStore) UpdateInstallation(installation *model.Installation) error {
	s.UpdateInstallationCalls++
	return nil
}

func (s *mockGroupStore) UpdateInstallationState(id, state string) error {
	s.UpdateInstallationCalls++
	return nil
}

func (s *mockGroupStore) LockInstallation(installationID, lockerID string) (bool, error) {
	return true, nil
}

func (s *mockGroupStore) UnlockInstallation(installationID, lockerID string, force bool) (bool, error) {
	if s.UnlockChan != nil {
		close(s.UnlockChan)
	}
	return true, nil
}

func TestGroupSupervisorDo(t *testing.T) {
	t.Run("no groups pending work", func(t *testing.T) {
		logger := testlib.MakeLogger(t)
		mockStore := &mockGroupStore{}

		supervisor := supervisor.NewGroupSupervisor(mockStore, "instanceID", logger)
		err := supervisor.Do()
		require.NoError(t, err)

		require.Equal(t, 0, mockStore.UpdateInstallationCalls)
	})

	t.Run("mock group and installation creation", func(t *testing.T) {
		logger := testlib.MakeLogger(t)
		mockStore := &mockGroupStore{}

		mockStore.UnlockedGroupsPendingWork = []*model.Group{&model.Group{
			ID: model.NewID(),
		}}
		mockStore.Group = mockStore.UnlockedGroupsPendingWork[0]

		mockStore.Installation = &model.Installation{
			ID:      model.NewID(),
			GroupID: &mockStore.Group.ID,
		}

		mockStore.GroupRollingMetadata = &store.GroupRollingMetadata{
			InstallationIDsToBeRolled: []string{mockStore.Installation.ID},
		}
		mockStore.UnlockChan = make(chan interface{})

		supervisor := supervisor.NewGroupSupervisor(mockStore, "instanceID", logger)
		err := supervisor.Do()
		require.NoError(t, err)

		<-mockStore.UnlockChan
		require.Equal(t, 0, mockStore.UpdateInstallationCalls)
	})
}
