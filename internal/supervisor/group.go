// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.
//

package supervisor

import (
	"math/rand"
	"time"

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

	GetWebhooks(filter *model.WebhookFilter) ([]*model.Webhook, error)
}

// GroupSupervisor finds installations belonging to groups that need to have
// their configuration reconciled to match a new group configuration setting.
//
// The degree of parallelism is controlled by a weighted semaphore, intended to
// be shared with other clients needing to coordinate background jobs.
type GroupSupervisor struct {
	store          groupStore
	eventsProducer eventProducer
	instanceID     string
	logger         log.FieldLogger
}

// NewGroupSupervisor creates a new GroupSupervisor.
func NewGroupSupervisor(store groupStore, eventsProducer eventProducer, instanceID string, logger log.FieldLogger) *GroupSupervisor {
	return &GroupSupervisor{
		store:          store,
		eventsProducer: eventsProducer,
		instanceID:     instanceID,
		logger:         logger,
	}
}

// Shutdown performs graceful shutdown tasks for the group supervisor.
func (s *GroupSupervisor) Shutdown() {
	s.logger.Debug("Shutting down group supervisor")
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
	rollingIsPaused := group.MaxRolling == 0
	logger := s.logger.WithFields(log.Fields{
		"groupID":         group.ID,
		"groupName":       group.Name,
		"maxRolling":      group.MaxRolling,
		"rollingIsPaused": rollingIsPaused,
	})

	groupLock := newGroupLock(group.ID, s.instanceID, s.store, logger)
	if !groupLock.TryLock() {
		return
	}
	defer groupLock.Unlock()

	logger.Debug("Supervising group")

	if rollingIsPaused {
		logger.Warn("Group rolling update is paused (MaxRolling=0); skipping...")
		return
	}

	groupMetadata, err := s.store.GetGroupRollingMetadata(group.ID)
	if err != nil {
		logger.WithError(err).Error("Unable to get installations in group")
		return
	}

	logger = logger.WithFields(log.Fields{
		"installations-total":   groupMetadata.InstallationsTotalCount,
		"installations-rolling": groupMetadata.InstallationsRolling,
	})

	if groupMetadata.InstallationsRolling >= group.MaxRolling {
		logger.Infof("Group already has %d rolling installations with a max of %d", groupMetadata.InstallationsRolling, group.MaxRolling)
		return
	}

	rand.Seed(time.Now().UnixNano())
	rand.Shuffle(len(groupMetadata.InstallationIDsToBeRolled), func(i, j int) {
		groupMetadata.InstallationIDsToBeRolled[i], groupMetadata.InstallationIDsToBeRolled[j] =
			groupMetadata.InstallationIDsToBeRolled[j], groupMetadata.InstallationIDsToBeRolled[i]
	})

	var moved int64
	for _, id := range groupMetadata.InstallationIDsToBeRolled {
		if groupMetadata.InstallationsRolling+moved >= group.MaxRolling {
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
			installationLock.Unlock()
			continue
		}

		oldState := installation.State
		installation.State = model.InstallationStateUpdateRequested
		err = s.store.UpdateInstallationState(installation)
		if err != nil {
			logger.WithError(err).Error("Unable to set new installation state")
		} else {
			moved++

			err = s.eventsProducer.ProduceInstallationStateChangeEvent(installation, oldState)
			if err != nil {
				logger.WithError(err).Error("Failed to create installation state change event")
			}
		}
		installationLock.Unlock()
	}

	logger.Infof("Moved %d installations to %s", moved, model.InstallationStateUpdateRequested)
}
