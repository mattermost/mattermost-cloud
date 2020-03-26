package supervisor

import (
	log "github.com/sirupsen/logrus"

	"github.com/mattermost/mattermost-cloud/internal/store"
	"github.com/mattermost/mattermost-cloud/model"
)

// groupStore abstracts the database operations required to query groups.
type groupStore interface {
	GetUnlockedGroupsPendingWork() ([]*model.Group, error)
	GetGroupRollingMetadata(groupID string) (*store.GroupRollingMetadata, error)
	LockGroup(groupID, lockerID string) (bool, error)
	UnlockGroup(groupID, lockerID string, force bool) (bool, error)

	GetInstallation(installationID string, includeGroupConfig, includeGroupConfigOverrides bool) (*model.Installation, error)
	UpdateInstallationState(*model.Installation) error
	LockInstallation(installationID, lockerID string) (bool, error)
	UnlockInstallation(installationID, lockerID string, force bool) (bool, error)
}

// GroupSupervisor finds installations belonging to groups that need to have
// their configuration reconciled to match a new group configuration setting.
//
// The degree of parallelism is controlled by a weighted semaphore, intended to
// be shared with other clients needing to coordinate background jobs.
type GroupSupervisor struct {
	store      groupStore
	instanceID string
	logger     log.FieldLogger
}

// NewGroupSupervisor creates a new GroupSupervisor.
func NewGroupSupervisor(store groupStore, instanceID string, logger log.FieldLogger) *GroupSupervisor {
	return &GroupSupervisor{
		store:      store,
		instanceID: instanceID,
		logger:     logger,
	}
}

// Do looks for work to be done on any pending groups and attempts to schedule
// the required work.
func (s *GroupSupervisor) Do() error {
	groups, err := s.store.GetUnlockedGroupsPendingWork()
	if err != nil {
		s.logger.WithError(err).Warn("Failed to query for groups")
		return nil
	}

	for _, group := range groups {
		s.Supervise(group)
	}

	return nil
}

// Supervise schedules the required work on the given group.
func (s *GroupSupervisor) Supervise(group *model.Group) {
	logger := s.logger.WithFields(log.Fields{
		"group": group.ID,
	})

	groupLock := newGroupLock(group.ID, s.instanceID, s.store, logger)
	if !groupLock.TryLock() {
		return
	}
	defer groupLock.Unlock()

	logger.Debug("Supervising group")

	groupMetadata, err := s.store.GetGroupRollingMetadata(group.ID)
	if err != nil {
		logger.WithError(err).Error("Unable to get installations in group")
		return
	}

	logger = logger.WithFields(log.Fields{
		"maxRolling":            group.MaxRolling,
		"installations-total":   groupMetadata.InstallationTotalCount,
		"installations-rolling": groupMetadata.InstallationNonStableCount,
	})

	if int64(groupMetadata.InstallationNonStableCount) >= group.MaxRolling {
		logger.Infof("Group already has %d rolling installations with a max of %d", groupMetadata.InstallationNonStableCount, group.MaxRolling)
		return
	}

	var moved int64
	for _, id := range groupMetadata.InstallationIDsToBeRolled {
		if groupMetadata.InstallationNonStableCount+moved >= group.MaxRolling {
			// We have bumped up against the max rolling count with the new
			// installations added to the rolling pool.
			break
		}

		installationLock := newInstallationLock(id, s.instanceID, s.store, logger)
		if !installationLock.TryLock() {
			return
		}

		installation, err := s.store.GetInstallation(id, true, false)
		if err != nil {
			logger.WithError(err).Error("Unable to get installation to set new state")
			continue
		}

		installation.State = model.InstallationStateUpdateRequested
		err = s.store.UpdateInstallationState(installation)
		if err != nil {
			logger.WithError(err).Error("Unable to set new installation state")
		} else {
			moved++
		}
		installationLock.Unlock()
	}

	logger.Infof("Moved %d installations to %s", moved, model.InstallationStateUpdateRequested)
}
