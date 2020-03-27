package api

import (
	"net/http"
	"sync"

	"github.com/mattermost/mattermost-cloud/model"
)

// lockCluster synchronizes access to the given cluster across potentially
// multiple provisioning servers.
func lockCluster(c *Context, clusterID string) (*model.Cluster, int, func()) {
	cluster, err := c.Store.GetCluster(clusterID)
	if err != nil {
		c.Logger.WithError(err).Error("failed to query cluster")
		return nil, http.StatusInternalServerError, nil
	}
	if cluster == nil {
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

	return cluster, 0, func() {
		unlockOnce.Do(func() {
			unlocked, err := c.Store.UnlockCluster(cluster.ID, c.RequestID, false)
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
func lockInstallation(c *Context, installationID string) (*model.Installation, int, func()) {
	installation, err := c.Store.GetInstallation(installationID, false, false)
	if err != nil {
		c.Logger.WithError(err).Error("failed to query installation")
		return nil, http.StatusInternalServerError, nil
	}
	if installation == nil {
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

	return installation, 0, func() {
		unlockOnce.Do(func() {
			unlocked, err := c.Store.UnlockInstallation(installation.ID, c.RequestID, false)
			if err != nil {
				c.Logger.WithError(err).Errorf("failed to unlock installation")
			} else if unlocked != true {
				c.Logger.Warn("failed to release lock for installation")
			}
		})
	}
}
