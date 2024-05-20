// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.
//

package supervisor

import (
	"time"

	log "github.com/sirupsen/logrus"

	"github.com/mattermost/mattermost-cloud/model"
)

// installationDeletionStore abstracts the database operations required to query
// installations for deletion operation.
type installationDeletionStore interface {
	GetInstallation(installationID string, includeGroupConfig, includeGroupConfigOverrides bool) (*model.Installation, error)
	GetUnlockedInstallationsPendingDeletion() ([]*model.Installation, error)
	GetInstallationsStatus() (*model.InstallationsStatus, error)
	UpdateInstallationState(*model.Installation) error
	installationLockStore

	GetStateChangeEvents(filter *model.StateChangeEventFilter) ([]*model.StateChangeEventData, error)

	GetWebhooks(filter *model.WebhookFilter) ([]*model.Webhook, error)
}

// InstallationDeletionSupervisor finds installations pending deletion and effects the required changes.
//
// The degree of parallelism is controlled by a weighted semaphore, intended to be shared with
// other clients needing to coordinate background jobs.
type InstallationDeletionSupervisor struct {
	instanceID               string
	deletionPendingTime      time.Duration
	currentlyUpdatingLimit   int64
	currentlyUpdatingCounter int64
	store                    installationDeletionStore
	logger                   log.FieldLogger
	eventsProducer           eventProducer
}

// NewInstallationDeletionSupervisor creates a new InstallationDeletionSupervisor.
func NewInstallationDeletionSupervisor(
	instanceID string,
	deletionPendingTime time.Duration,
	currentlyUpdatingLimit int64,
	store installationDeletionStore,
	eventsProducer eventProducer,
	logger log.FieldLogger) *InstallationDeletionSupervisor {
	return &InstallationDeletionSupervisor{
		instanceID:               instanceID,
		deletionPendingTime:      deletionPendingTime,
		currentlyUpdatingLimit:   currentlyUpdatingLimit,
		currentlyUpdatingCounter: 0,
		store:                    store,
		eventsProducer:           eventsProducer,
		logger:                   logger,
	}
}

// Shutdown performs graceful shutdown tasks for the installation deletion supervisor.
func (s *InstallationDeletionSupervisor) Shutdown() {
	s.logger.Debug("Shutting down installation-deletion supervisor")
}

// Do looks for work to be done on any pending installations and attempts to schedule the required work.
func (s *InstallationDeletionSupervisor) Do() error {
	installations, err := s.store.GetUnlockedInstallationsPendingDeletion()
	if err != nil {
		s.logger.WithError(err).Warn("Failed to query for installation pending deletion")
		return nil
	}

	// Limit the number of installations that can be deleted at one time.
	// NOTE: this is a bit of a soft limit. Multiple provisioners running at the
	// same time could lead to a situation where the limit is exceeded. This
	// was done initially to balance complexity while still having control over
	// deletion spikes. A hard limit could be added in the future if required.
	status, err := s.store.GetInstallationsStatus()
	if err != nil {
		s.logger.WithError(err).Warn("Failed to query for installation status")
		return nil
	}
	s.currentlyUpdatingCounter = status.InstallationsUpdating

	for _, installation := range installations {
		if s.currentlyUpdatingCounter >= s.currentlyUpdatingLimit {
			s.logger.Infof("Max installation updating counter (%d) reached for pending deletions", s.currentlyUpdatingLimit)
			return nil
		}
		s.Supervise(installation)
	}

	return nil
}

// Supervise schedules the required work on the given installation.
func (s *InstallationDeletionSupervisor) Supervise(installation *model.Installation) {
	logger := s.logger.WithFields(log.Fields{
		"installation": installation.ID,
	})

	lock := newInstallationLock(installation.ID, s.instanceID, s.store, logger)
	if !lock.TryLock() {
		return
	}
	defer lock.Unlock()

	// Before working on the installation, it is crucial that we ensure that it
	// was not updated to a new state by another provisioning server.
	originalState := installation.State
	installation, err := s.store.GetInstallation(installation.ID, true, false)
	if err != nil {
		logger.WithError(err).Errorf("Failed to get refreshed installation")
		return
	}
	if installation.State != originalState {
		logger.WithField("oldInstallationState", originalState).
			WithField("newInstallationState", installation.State).
			Warn("Another provisioner has worked on this installation; skipping...")
		return
	}

	logger.Debugf("Supervising installation in state %s", installation.State)

	newState := s.transitionInstallation(installation, logger)

	installation, err = s.store.GetInstallation(installation.ID, true, false)
	if err != nil {
		logger.WithError(err).Errorf("Failed to get installation and thus persist state %s", newState)
		return
	}

	if installation.State == newState {
		return
	}

	oldState := installation.State
	installation.State = newState

	err = s.store.UpdateInstallationState(installation)
	if err != nil {
		logger.WithError(err).Errorf("Failed to set installation state to %s", newState)
		return
	}

	err = s.eventsProducer.ProduceInstallationStateChangeEvent(installation, oldState)
	if err != nil {
		logger.WithError(err).Error("Failed to create installation state change event")
	}

	logger.Debugf("Transitioned installation from %s to %s", oldState, installation.State)
}

// transitionInstallation works with the given installation to transition it to a final state.
func (s *InstallationDeletionSupervisor) transitionInstallation(installation *model.Installation, logger log.FieldLogger) string {
	switch installation.State {
	case model.InstallationStateDeletionPending:
		return s.checkIfInstallationShouldBeDeleted(installation, logger)
	default:
		logger.Warnf("Found installation pending deletion in unexpected state %s", installation.State)
		return installation.State
	}
}

func (s *InstallationDeletionSupervisor) checkIfInstallationShouldBeDeleted(installation *model.Installation, logger log.FieldLogger) string {
	if installation.DeletionPendingExpiry != 0 {
		// Primary deletion pending check.
		if model.GetMillis() < installation.DeletionPendingExpiry {
			timeUntilDeletion := time.Until(model.TimeFromMillis(installation.DeletionPendingExpiry))
			logger.WithField("time-until-deletion", timeUntilDeletion.Round(time.Second).String()).Debug("Installation is not ready for deletion")
			return model.InstallationStateDeletionPending
		}
	} else {
		// Backup deletion pending check. Grab the latest event matching the
		// deletion pending state change.
		events, err := s.store.GetStateChangeEvents(&model.StateChangeEventFilter{
			ResourceID: installation.ID,
			NewStates:  []string{model.InstallationStateDeletionPending},
			Paging: model.Paging{
				Page:           0,
				PerPage:        1,
				IncludeDeleted: false,
			},
		})
		if err != nil {
			logger.WithError(err).Warn("Failed to query installation events")
			return model.InstallationStateDeletionPending
		}
		if len(events) != 1 {
			logger.WithError(err).Warnf("Expected 1 installation event, but got %d", len(events))
			return model.InstallationStateDeletionPending
		}
		deletionQueuedEvent := events[0]

		// Check to see if enough time has passed that the installation should be
		// deleted.
		timeSincePending := time.Since(model.TimeFromMillis(deletionQueuedEvent.Event.Timestamp))
		logger = logger.WithField("time-spent-pending-deletion", timeSincePending.String())
		if timeSincePending < s.deletionPendingTime {
			timeUntilDeletion := s.deletionPendingTime - timeSincePending
			logger.WithField("time-until-deletion", timeUntilDeletion.String()).Debug("Installation is not ready for deletion")
			return model.InstallationStateDeletionPending
		}
	}

	logger.Info("Installation is ready for deletion")
	s.currentlyUpdatingCounter++

	return model.InstallationStateDeletionRequested
}
