// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.
//

package api

import (
	"net/http"

	"github.com/gorilla/mux"
)

// initSecurity registers security endpoints on the given router.
func initSecurity(apiRouter *mux.Router, context *Context) {
	addContext := func(handler contextHandlerFunc) *contextHandler {
		return newContextHandler(context, handler)
	}

	securityRouter := apiRouter.PathPrefix("/security").Subrouter()

	securityClusterRouter := securityRouter.PathPrefix("/cluster/{cluster:[A-Za-z0-9]{26}}").Subrouter()
	securityClusterRouter.Handle("/api/lock", addContext(handleClusterLockAPI)).Methods("POST")
	securityClusterRouter.Handle("/api/unlock", addContext(handleClusterUnlockAPI)).Methods("POST")

	securityInstallationRouter := securityRouter.PathPrefix("/installation/{installation:[A-Za-z0-9]{26}}").Subrouter()
	securityInstallationRouter.Handle("/api/lock", addContext(handleInstallationLockAPI)).Methods("POST")
	securityInstallationRouter.Handle("/api/unlock", addContext(handleInstallationUnlockAPI)).Methods("POST")
	securityInstallationRouter.Handle("/deletion/lock", addContext(handleInstallationDeletionLockAPI)).Methods("POST")
	securityInstallationRouter.Handle("/deletion/unlock", addContext(handleInstallationDeletionUnlockAPI)).Methods("POST")

	securityClusterInstallationRouter := securityRouter.PathPrefix("/cluster_installation/{cluster_installation:[A-Za-z0-9]{26}}").Subrouter()
	securityClusterInstallationRouter.Handle("/api/lock", addContext(handleClusterInstallationLockAPI)).Methods("POST")
	securityClusterInstallationRouter.Handle("/api/unlock", addContext(handleClusterInstallationUnlockAPI)).Methods("POST")

	securityGroupRouter := securityRouter.PathPrefix("/group/{group:[A-Za-z0-9]{26}}").Subrouter()
	securityGroupRouter.Handle("/api/lock", addContext(handleGroupLockAPI)).Methods("POST")
	securityGroupRouter.Handle("/api/unlock", addContext(handleGroupUnlockAPI)).Methods("POST")

	securityBackupRouter := securityRouter.PathPrefix("/installation/backup/{backup:[A-Za-z0-9]{26}}").Subrouter()
	securityBackupRouter.Handle("/api/lock", addContext(handleBackupLockAPI)).Methods("POST")
	securityBackupRouter.Handle("/api/unlock", addContext(handleBackupUnlockAPI)).Methods("POST")
}

// handleClusterLockAPI responds to POST /api/security/cluster/{cluster}/api/lock,
// locking API changes for this cluster.
func handleClusterLockAPI(c *Context, w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	clusterID := vars["cluster"]
	c.Logger = c.Logger.WithField("cluster", clusterID)

	cluster, err := c.Store.GetCluster(clusterID)
	if err != nil {
		c.Logger.WithError(err).Error("failed to query cluster")
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	if cluster == nil {
		w.WriteHeader(http.StatusNotFound)
		return
	}

	if !cluster.APISecurityLock {
		err = c.Store.LockClusterAPI(cluster.ID)
		if err != nil {
			c.Logger.WithError(err).Error("failed to lock cluster API")
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
	}

	w.WriteHeader(http.StatusOK)
}

// handleClusterUnlockAPI responds to POST /api/security/cluster/{cluster}/api/unlock,
// unlocking API changes for this cluster.
func handleClusterUnlockAPI(c *Context, w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	clusterID := vars["cluster"]
	c.Logger = c.Logger.WithField("cluster", clusterID)

	cluster, err := c.Store.GetCluster(clusterID)
	if err != nil {
		c.Logger.WithError(err).Error("failed to query cluster")
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	if cluster == nil {
		w.WriteHeader(http.StatusNotFound)
		return
	}

	if cluster.APISecurityLock {
		err = c.Store.UnlockClusterAPI(cluster.ID)
		if err != nil {
			c.Logger.WithError(err).Error("failed to unlock cluster API")
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
	}

	w.WriteHeader(http.StatusOK)
}

// handleInstallationLockAPI responds to POST /api/security/installation/{installation}/api/lock,
// locking API changes for this installation.
func handleInstallationLockAPI(c *Context, w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	installationID := vars["installation"]
	c.Logger = c.Logger.WithField("installation", installationID)

	installation, err := c.Store.GetInstallation(installationID, false, false)
	if err != nil {
		c.Logger.WithError(err).Error("failed to query installation")
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	if installation == nil {
		w.WriteHeader(http.StatusNotFound)
		return
	}

	if !installation.APISecurityLock {
		err = c.Store.LockInstallationAPI(installation.ID)
		if err != nil {
			c.Logger.WithError(err).Error("failed to lock installation API")
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
	}

	w.WriteHeader(http.StatusOK)
}

// handleInstallationUnlockAPI responds to POST /api/security/installation/{installation}/api/unlock,
// unlocking API changes for this installation.
func handleInstallationUnlockAPI(c *Context, w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	installationID := vars["installation"]
	c.Logger = c.Logger.WithField("installation", installationID)

	installation, err := c.Store.GetInstallation(installationID, false, false)
	if err != nil {
		c.Logger.WithError(err).Error("failed to query installation")
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	if installation == nil {
		w.WriteHeader(http.StatusNotFound)
		return
	}

	if installation.APISecurityLock {
		err = c.Store.UnlockInstallationAPI(installation.ID)
		if err != nil {
			c.Logger.WithError(err).Error("failed to unlock installation API")
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
	}

	w.WriteHeader(http.StatusOK)
}

// handleInstallationDeletionLockAPI responds to POST /api/security/installation/{installation}/api/deletion_lock,
// setting a deletion lock for this installation, preventing it from being deleted until unlocked
func handleInstallationDeletionLockAPI(c *Context, w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	installationID := vars["installation"]
	c.Logger = c.Logger.WithField("installation", installationID)

	installation, err := c.Store.GetInstallation(installationID, false, false)
	if err != nil {
		c.Logger.WithError(err).Error("failed to query installation")
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	if installation == nil {
		w.WriteHeader(http.StatusNotFound)
		return
	}

	// TODO: Should this respect the APISecurityLock?
	err = c.Store.DeletionLockInstallation(installation.ID)
	if err != nil {
		c.Logger.WithError(err).Error("failed to lock deletion lock installation")
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
}

// handleInstallationDeletionUnlockAPI responds to POST /api/security/installation/{installation}/api/deletion_unlock,
// unlocking an installation that has been deletion locked, allowing it to be deleted
func handleInstallationDeletionUnlockAPI(c *Context, w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	installationID := vars["installation"]
	c.Logger = c.Logger.WithField("installation", installationID)

	installation, err := c.Store.GetInstallation(installationID, false, false)
	if err != nil {
		c.Logger.WithError(err).Error("failed to query installation")
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	if installation == nil {
		w.WriteHeader(http.StatusNotFound)
		return
	}

	// TODO: Should this respect the APISecurityLock?
	err = c.Store.DeletionUnlockInstallation(installation.ID)
	if err != nil {
		c.Logger.WithError(err).Error("failed to deletion unlock installation")
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
}

// handleClusterInstallationLockAPI responds to POST /api/security/cluster_installation/{cluster_installation}/api/lock,
// locking API changes for this cluster installation.
func handleClusterInstallationLockAPI(c *Context, w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	clusterInstallationID := vars["cluster_installation"]
	c.Logger = c.Logger.WithField("cluster_installation", clusterInstallationID)

	clusterInstallation, err := c.Store.GetClusterInstallation(clusterInstallationID)
	if err != nil {
		c.Logger.WithError(err).Error("failed to query cluster installation")
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	if clusterInstallation == nil {
		w.WriteHeader(http.StatusNotFound)
		return
	}

	if !clusterInstallation.APISecurityLock {
		err = c.Store.LockClusterInstallationAPI(clusterInstallation.ID)
		if err != nil {
			c.Logger.WithError(err).Error("failed to lock cluster installation API")
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
	}

	w.WriteHeader(http.StatusOK)
}

// handleClusterInstallationUnlockAPI responds to POST /api/security/cluster_installation/{cluster_installation}/api/unlock,
// unlocking API changes for this cluster installation.
func handleClusterInstallationUnlockAPI(c *Context, w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	clusterInstallationID := vars["cluster_installation"]
	c.Logger = c.Logger.WithField("cluster_installation", clusterInstallationID)

	clusterInstallation, err := c.Store.GetClusterInstallation(clusterInstallationID)
	if err != nil {
		c.Logger.WithError(err).Error("failed to query cluster installation")
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	if clusterInstallation == nil {
		w.WriteHeader(http.StatusNotFound)
		return
	}

	if clusterInstallation.APISecurityLock {
		err = c.Store.UnlockClusterInstallationAPI(clusterInstallation.ID)
		if err != nil {
			c.Logger.WithError(err).Error("failed to unlock cluster installation API")
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
	}

	w.WriteHeader(http.StatusOK)
}

// handleGroupLockAPI responds to POST /api/security/group/{group}/api/lock,
// locking API changes for this group.
func handleGroupLockAPI(c *Context, w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	groupID := vars["group"]
	c.Logger = c.Logger.WithField("group", groupID)

	group, err := c.Store.GetGroup(groupID)
	if err != nil {
		c.Logger.WithError(err).Error("failed to query group")
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	if group == nil {
		w.WriteHeader(http.StatusNotFound)
		return
	}

	if !group.APISecurityLock {
		err = c.Store.LockGroupAPI(group.ID)
		if err != nil {
			c.Logger.WithError(err).Error("failed to lock group API")
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
	}

	w.WriteHeader(http.StatusOK)
}

// handleGroupUnlockAPI responds to POST /api/security/group/{group}/api/unlock,
// unlocking API changes for this group.
func handleGroupUnlockAPI(c *Context, w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	groupID := vars["group"]
	c.Logger = c.Logger.WithField("group", groupID)

	group, err := c.Store.GetGroup(groupID)
	if err != nil {
		c.Logger.WithError(err).Error("failed to query group")
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	if group == nil {
		w.WriteHeader(http.StatusNotFound)
		return
	}

	if group.APISecurityLock {
		err = c.Store.UnlockGroupAPI(group.ID)
		if err != nil {
			c.Logger.WithError(err).Error("failed to unlock group API")
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
	}

	w.WriteHeader(http.StatusOK)
}

// handleBackupLockAPI responds to POST /api/security/installation/backup/{backup}/api/lock,
// locking API changes for this installation backup.
func handleBackupLockAPI(c *Context, w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	backupID := vars["backup"]
	c.Logger = c.Logger.WithField("backup", backupID)

	backupMetadata, err := c.Store.GetInstallationBackup(backupID)
	if err != nil {
		c.Logger.WithError(err).Error("failed to query backup")
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	if backupMetadata == nil {
		w.WriteHeader(http.StatusNotFound)
		return
	}

	if !backupMetadata.APISecurityLock {
		err = c.Store.LockInstallationBackupAPI(backupMetadata.ID)
		if err != nil {
			c.Logger.WithError(err).Error("failed to lock backup API")
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
	}

	w.WriteHeader(http.StatusOK)
}

// handleBackupUnlockAPI responds to POST /api/security/installation/backup/{backup}/api/unlock,
// unlocking API changes for this backup.
func handleBackupUnlockAPI(c *Context, w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	backupID := vars["backup"]
	c.Logger = c.Logger.WithField("backup", backupID)

	backupMetadata, err := c.Store.GetInstallationBackup(backupID)
	if err != nil {
		c.Logger.WithError(err).Error("failed to query backup")
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	if backupMetadata == nil {
		w.WriteHeader(http.StatusNotFound)
		return
	}

	if backupMetadata.APISecurityLock {
		err = c.Store.UnlockInstallationBackupAPI(backupMetadata.ID)
		if err != nil {
			c.Logger.WithError(err).Error("failed to unlock backup API")
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
	}

	w.WriteHeader(http.StatusOK)
}
