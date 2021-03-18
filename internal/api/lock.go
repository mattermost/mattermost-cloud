// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.
//

package api

import (
	"net/http"
	"sync"

	"github.com/mattermost/mattermost-cloud/model"
)

// lockCluster synchronizes access to the given cluster across potentially
// multiple provisioning servers.
func lockCluster(c *Context, clusterID string) (*model.ClusterDTO, int, func()) {
	clusterDTO, err := c.Store.GetClusterDTO(clusterID)
	if err != nil {
		c.Logger.WithError(err).Error("failed to query cluster")
		return nil, http.StatusInternalServerError, nil
	}
	if clusterDTO == nil {
		return nil, http.StatusNotFound, nil
	}

	locked, err := c.Store.LockCluster(clusterID, c.RequestID)
	if err != nil {
		c.Logger.WithError(err).Error("failed to lock cluster")
		return nil, http.StatusInternalServerError, nil
	} else if !locked {
		c.Logger.Error("failed to acquire lock for cluster")
		return nil, http.StatusConflict, nil
	}

	unlockOnce := sync.Once{}

	return clusterDTO, 0, func() {
		unlockOnce.Do(func() {
			unlocked, err := c.Store.UnlockCluster(clusterDTO.ID, c.RequestID, false)
			if err != nil {
				c.Logger.WithError(err).Errorf("failed to unlock cluster")
			} else if unlocked != true {
				c.Logger.Error("failed to release lock for cluster")
			}
		})
	}
}

// lockGroup synchronizes access to the given group across potentially multiple
// provisioning servers.
func lockGroup(c *Context, groupID string) (*model.Group, int, func()) {
	group, err := c.Store.GetGroup(groupID)
	if err != nil {
		c.Logger.WithError(err).Error("failed to query group")
		return nil, http.StatusInternalServerError, nil
	}
	if group == nil {
		return nil, http.StatusNotFound, nil
	}

	locked, err := c.Store.LockGroup(groupID, c.RequestID)
	if err != nil {
		c.Logger.WithError(err).Error("failed to lock group")
		return nil, http.StatusInternalServerError, nil
	} else if !locked {
		c.Logger.Error("failed to acquire lock for group")
		return nil, http.StatusConflict, nil
	}

	unlockOnce := sync.Once{}

	return group, 0, func() {
		unlockOnce.Do(func() {
			unlocked, err := c.Store.UnlockGroup(group.ID, c.RequestID, false)
			if err != nil {
				c.Logger.WithError(err).Errorf("failed to unlock group")
			} else if unlocked != true {
				c.Logger.Warn("failed to release lock for group")
			}
		})
	}
}

// lockInstallation synchronizes access to the given installation across
// potentially multiple provisioning servers.
func lockInstallation(c *Context, installationID string) (*model.InstallationDTO, int, func()) {
	installationDTO, err := c.Store.GetInstallationDTO(installationID, false, false)
	if err != nil {
		c.Logger.WithError(err).Error("failed to query installation")
		return nil, http.StatusInternalServerError, nil
	}
	if installationDTO == nil {
		return nil, http.StatusNotFound, nil
	}

	locked, err := c.Store.LockInstallation(installationID, c.RequestID)
	if err != nil {
		c.Logger.WithError(err).Error("failed to lock installation")
		return nil, http.StatusInternalServerError, nil
	} else if !locked {
		c.Logger.Error("failed to acquire lock for installation")
		return nil, http.StatusConflict, nil
	}

	unlockOnce := sync.Once{}

	return installationDTO, 0, func() {
		unlockOnce.Do(func() {
			unlocked, err := c.Store.UnlockInstallation(installationDTO.ID, c.RequestID, false)
			if err != nil {
				c.Logger.WithError(err).Errorf("failed to unlock installation")
			} else if unlocked != true {
				c.Logger.Warn("failed to release lock for installation")
			}
		})
	}
}

// lockInstallationBackup synchronizes access to the given installation backup across
// potentially multiple provisioning servers.
func lockInstallationBackup(c *Context, backupID string) (*model.InstallationBackup, int, func()) {
	backup, err := c.Store.GetInstallationBackup(backupID)
	if err != nil {
		c.Logger.WithError(err).Error("failed to query backup metadata")
		return nil, http.StatusInternalServerError, nil
	}
	if backup == nil {
		return nil, http.StatusNotFound, nil
	}

	locked, err := c.Store.LockInstallationBackup(backupID, c.RequestID)
	if err != nil {
		c.Logger.WithError(err).Error("failed to lock backup")
		return nil, http.StatusInternalServerError, nil
	} else if !locked {
		c.Logger.Error("failed to acquire lock for backup")
		return nil, http.StatusConflict, nil
	}

	unlockOnce := sync.Once{}

	return backup, 0, func() {
		unlockOnce.Do(func() {
			unlocked, err := c.Store.UnlockInstallationBackup(backup.ID, c.RequestID, false)
			if err != nil {
				c.Logger.WithError(err).Errorf("failed to unlock backup")
			} else if unlocked != true {
				c.Logger.Warn("failed to release lock for backup")
			}
		})
	}
}
