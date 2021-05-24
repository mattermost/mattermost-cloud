// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.
//

package api

import (
	"fmt"
	"net/http"

	"github.com/gorilla/mux"
	"github.com/mattermost/mattermost-cloud/model"
)

// initClusterInstallation registers cluster installation endpoints on the given router.
func initClusterInstallation(apiRouter *mux.Router, context *Context) {
	addContext := func(handler contextHandlerFunc) *contextHandler {
		return newContextHandler(context, handler)
	}

	clusterInstallationsRouter := apiRouter.PathPrefix("/cluster_installations").Subrouter()
	clusterInstallationsRouter.Handle("", addContext(handleGetClusterInstallations)).Methods("GET")
	clusterInstallationsRouter.Handle("/migrate", addContext(handleMigrateClusterInstallations)).Methods("POST")
	clusterInstallationsRouter.Handle("/migrate/dns", addContext(handleMigrateDNS)).Methods("POST")
	clusterInstallationsRouter.Handle("/migrate/delete_stale/{clusterID}", addContext(handleDeleteStaleClusterInstallationsByCluster)).Methods("DELETE")
	clusterInstallationsRouter.Handle("/migrate/delete_stale/cluster_installation/{ClusterInstallationID}", addContext(handleDeleteStaleClusterInstallationByID)).Methods("DELETE")

	clusterInstallationRouter := apiRouter.PathPrefix("/cluster_installation/{cluster_installation:[A-Za-z0-9]{26}}").Subrouter()
	clusterInstallationRouter.Handle("", addContext(handleGetClusterInstallation)).Methods("GET")
	clusterInstallationRouter.Handle("/config", addContext(handleGetClusterInstallationConfig)).Methods("GET")
	clusterInstallationRouter.Handle("/config", addContext(handleSetClusterInstallationConfig)).Methods("PUT")
	clusterInstallationRouter.Handle("/exec/{command}", addContext(handleRunClusterInstallationExecCommand)).Methods("POST")
	clusterInstallationRouter.Handle("/mattermost_cli", addContext(handleRunClusterInstallationMattermostCLI)).Methods("POST")
}

// handleGetClusterInstallations responds to GET /api/cluster_installations, returning the specified page of cluster installations.
func handleGetClusterInstallations(c *Context, w http.ResponseWriter, r *http.Request) {
	clusterID := r.URL.Query().Get("cluster")
	installationID := r.URL.Query().Get("installation")

	paging, err := parsePaging(r.URL)
	if err != nil {
		c.Logger.WithError(err).Error("failed to parse paging parameters")
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	filter := &model.ClusterInstallationFilter{
		ClusterID:      clusterID,
		InstallationID: installationID,
		Paging:         paging,
	}

	clusterInstallations, err := c.Store.GetClusterInstallations(filter)
	if err != nil {
		c.Logger.WithError(err).Error("failed to query cluster installations")
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	if clusterInstallations == nil {
		clusterInstallations = []*model.ClusterInstallation{}
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	outputJSON(c, w, clusterInstallations)
}

// handleGetClusterInstallation responds to GET /api/cluster_installation/{cluster_installation}, returning the cluster installation in question.
func handleGetClusterInstallation(c *Context, w http.ResponseWriter, r *http.Request) {
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

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	outputJSON(c, w, clusterInstallation)
}

// handleGetClusterInstallationConfig responds to GET /api/cluster_installation/{cluster_installation}/config, returning the config for the cluster installation in question.
func handleGetClusterInstallationConfig(c *Context, w http.ResponseWriter, r *http.Request) {
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
	if clusterInstallation.IsDeleted() {
		c.Logger.Error("cluster installation is deleted")
		w.WriteHeader(http.StatusGone)
		return
	}

	cluster, err := c.Store.GetCluster(clusterInstallation.ClusterID)
	if err != nil {
		c.Logger.WithError(err).Error("failed to query cluster")
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	if cluster == nil {
		c.Logger.Errorf("failed to find cluster %s associated with cluster installations", clusterInstallation.ClusterID)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	output, err := c.Provisioner.ExecMattermostCLI(cluster, clusterInstallation, "config", "show", "--json")
	if err != nil {
		c.Logger.WithError(err).Error("failed to execute mattermost cli")
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	w.Write(output)
}

// handleSetClusterInstallationConfig responds to PUT /api/cluster_installation/{cluster_installation}/config, merging the given config into the given cluster installation.
func handleSetClusterInstallationConfig(c *Context, w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	clusterInstallationID := vars["cluster_installation"]
	c.Logger = c.Logger.WithField("cluster_installation", clusterInstallationID)

	clusterInstallationConfigRequest, err := model.NewClusterInstallationConfigRequestFromReader(r.Body)
	if err != nil {
		c.Logger.WithError(err).Error("failed to decode request")
		w.WriteHeader(http.StatusBadRequest)
		return
	}

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
	if clusterInstallation.IsDeleted() {
		c.Logger.Error("cluster installation is deleted")
		w.WriteHeader(http.StatusGone)
		return
	}

	if clusterInstallation.APISecurityLock {
		logSecurityLockConflict("cluster-installation", c.Logger)
		w.WriteHeader(http.StatusForbidden)
		return
	}

	cluster, err := c.Store.GetCluster(clusterInstallation.ClusterID)
	if err != nil {
		c.Logger.WithError(err).Error("failed to query cluster")
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	if cluster == nil {
		w.WriteHeader(http.StatusNotFound)
		return
	}

	var applyConfig func(parentKey string, value map[string]interface{}) error

	// applyConfig takes the decomposed configuration, walks the resulting map, and invokes
	// something akin to:
	//	mattermost config set <ParentKey1.ParentKey2.LeafKey> <value>
	//
	// Ideally, this would be replaced by simply using the API and passing in the config struct
	// directly, but at the moment that requires authentication.
	applyConfig = func(parentKey string, parentValue map[string]interface{}) error {
		if parentKey != "" {
			parentKey = parentKey + "."
		}

		for key, value := range parentValue {
			fullKey := parentKey + key

			valueMap, ok := value.(map[string]interface{})
			if ok {
				err = applyConfig(fullKey, valueMap)
				if err != nil {
					return err
				}

				continue
			}

			valueStr, ok := value.(string)
			if ok {
				_, err := c.Provisioner.ExecMattermostCLI(cluster, clusterInstallation, "config", "set", fullKey, valueStr)
				if err != nil {
					c.Logger.WithError(err).Errorf("failed to set key %s to value %s", fullKey, valueStr)
					return err
				}

				c.Logger.Infof("Successfully set config key %s to value %s", fullKey, valueStr)
				continue
			}

			c.Logger.WithError(err).Errorf("unable to set key %s with value %t", fullKey, value)
			return err
		}

		return nil
	}

	err = applyConfig("", clusterInstallationConfigRequest)
	if err != nil {
		c.Logger.WithError(err).Error("failed to set the config")
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
}

// handleRunClusterInstallationExecCommand responds to POST /api/cluster_installation/{cluster_installation}/exec/{command},
// running a valid exec command and returning any output.
func handleRunClusterInstallationExecCommand(c *Context, w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	clusterInstallationID := vars["cluster_installation"]
	command := vars["command"]
	c.Logger = c.Logger.WithField("cluster_installation", clusterInstallationID)

	if !model.IsValidExecCommand(command) {
		c.Logger.Errorf("%s is not a permitted exec command", command)
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	clusterInstallationExecSubcommand, err := model.NewClusterInstallationExecSubcommandFromReader(r.Body)
	if err != nil {
		c.Logger.WithError(err).Error("failed to decode request")
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	clusterInstallation, err := c.Store.GetClusterInstallation(clusterInstallationID)
	if err != nil {
		c.Logger.WithError(err).Error("failed to query cluster installation")
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	if clusterInstallation == nil {
		c.Logger.Error("cluster installation not found")
		w.WriteHeader(http.StatusNotFound)
		return
	}
	if clusterInstallation.IsDeleted() {
		c.Logger.Error("cluster installation is deleted")
		w.WriteHeader(http.StatusGone)
		return
	}

	if clusterInstallation.APISecurityLock {
		logSecurityLockConflict("cluster-installation", c.Logger)
		w.WriteHeader(http.StatusForbidden)
		return
	}

	cluster, err := c.Store.GetCluster(clusterInstallation.ClusterID)
	if err != nil {
		c.Logger.WithError(err).Error("failed to query cluster")
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	if cluster == nil {
		c.Logger.Errorf("failed to find cluster %s associated with cluster installation", clusterInstallation.ClusterID)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	args := append([]string{fmt.Sprintf("./bin/%s", command)}, clusterInstallationExecSubcommand...)
	output, err := c.Provisioner.ExecClusterInstallationCLI(cluster, clusterInstallation, args...)
	if err != nil {
		c.Logger.WithError(err).Error("failed to execute command")
		w.WriteHeader(http.StatusInternalServerError)
		w.Write(output)
		return
	}

	w.WriteHeader(http.StatusOK)
	w.Write(output)
}

// handleRunClusterInstallationMattermostCLI responds to POST /api/cluster_installation/{cluster_installation}/mattermost_cli, running a Mattermost CLI command and returning any output.
// TODO: deprecate or refactor into /exec/command endpoint
func handleRunClusterInstallationMattermostCLI(c *Context, w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	clusterInstallationID := vars["cluster_installation"]
	c.Logger = c.Logger.WithField("cluster_installation", clusterInstallationID)

	clusterInstallationMattermostCLISubcommandRequest, err := model.NewClusterInstallationMattermostCLISubcommandFromReader(r.Body)
	if err != nil {
		c.Logger.WithError(err).Error("failed to decode request")
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	clusterInstallation, err := c.Store.GetClusterInstallation(clusterInstallationID)
	if err != nil {
		c.Logger.WithError(err).Error("failed to query cluster installation")
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	if clusterInstallation == nil {
		c.Logger.Error("cluster installation not found")
		w.WriteHeader(http.StatusNotFound)
		return
	}
	if clusterInstallation.IsDeleted() {
		c.Logger.Error("cluster installation is deleted")
		w.WriteHeader(http.StatusGone)
		return
	}

	if clusterInstallation.APISecurityLock {
		logSecurityLockConflict("cluster-installation", c.Logger)
		w.WriteHeader(http.StatusForbidden)
		return
	}

	cluster, err := c.Store.GetCluster(clusterInstallation.ClusterID)
	if err != nil {
		c.Logger.WithError(err).Error("failed to query cluster")
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	if cluster == nil {
		c.Logger.Errorf("failed to find cluster %s associated with cluster installations", clusterInstallation.ClusterID)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	output, err := c.Provisioner.ExecMattermostCLI(cluster, clusterInstallation, clusterInstallationMattermostCLISubcommandRequest...)
	if err != nil {
		c.Logger.WithError(err).Error("failed to execute mattermost cli")
		w.WriteHeader(http.StatusInternalServerError)
		w.Write(output)
		return
	}

	w.WriteHeader(http.StatusOK)
	w.Write(output)
}

// handleMigrateClusterInstallations responds to Post /api/cluster_installation/migrate.
func handleMigrateClusterInstallations(c *Context, w http.ResponseWriter, r *http.Request) {
	mcir, err := model.NewMigrateClusterInstallationRequestFromReader(r.Body)
	if err != nil {
		c.Logger.WithError(err).Error("failed to decode cluster migration request")
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	c.Logger = c.Logger.WithField("Migration request for cluster_installation", mcir)

	if len(mcir.ClusterID) == 0 {
		c.Logger.WithError(err).Error("Missing mandatory primary cluster in a migration request")
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	if len(mcir.TargetCluster) == 0 {
		c.Logger.WithError(err).Error("Missing mandatory secondary cluster in a migration request")
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	filter := &model.ClusterInstallationFilter{
		ClusterID:      mcir.ClusterID,
		InstallationID: mcir.InstallationID,
		Paging:         model.AllPagesNotDeleted(),
	}
	clusterInstallations, err := c.Store.GetClusterInstallations(filter)
	if err != nil {
		c.Logger.WithError(err).Error("failed to query cluster installations")
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	if clusterInstallations == nil {
		c.Logger.WithError(err).Error("No matching cluster installations found")
		w.WriteHeader(http.StatusNotFound)
		return
	}

	// Migrate the cluster installations to the target cluster
	c.Logger.Infof("Migrating installation(s) to the clusterID: %s", mcir.TargetCluster)
	err = c.Store.MigrateClusterInstallations(clusterInstallations, mcir.TargetCluster)
	if err != nil {
		c.Logger.WithError(err).Error("failed to migrate cluster installation(s)")
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	// Mark old cluster installation as stale
	err = c.Store.UpdateClusterInstallationsStaleStatus(mcir.ClusterID, true)
	if err != nil {
		c.Logger.WithError(err).Error("failed to disable old cluster installation(s)")
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	c.Logger.Infof("Cluster installations have been marked as stale for cluster: %s", mcir.ClusterID)

	// Reset the DNS configuration status for respective installations to update the CNAME with the new LB.
	//fetch non stale cluster installation to avoid duplicate update
	isStaleClusterInstallations := false
	filter = &model.ClusterInstallationFilter{
		ClusterID:      mcir.ClusterID,
		InstallationID: mcir.InstallationID,
		Paging:         model.AllPagesNotDeleted(),
		IsStale:        &isStaleClusterInstallations,
	}
	clusterInstallations, err = c.Store.GetClusterInstallations(filter)
	if err != nil {
		c.Logger.WithError(err).Error("failed to query active cluster installations")
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	if clusterInstallations == nil {
		c.Logger.WithError(err).Error("No matching active cluster installations found")
		w.WriteHeader(http.StatusNotFound)
		return
	}

	if mcir.DNSSwitch {
		var installations []*model.Installation
		if mcir.LockInstallation {
			c.Logger.Infof("Locking %s installation(s) ", len(clusterInstallations))
			for _, ci := range clusterInstallations {
				installationDTO, status, unlockOnce := lockInstallation(c, ci.InstallationID)
				if status != 0 {
					w.WriteHeader(status)
					return
				}
				defer unlockOnce()
				installations = append(installations, installationDTO.Installation)
			}
		} else {
			for _, ci := range clusterInstallations {
				installation, err := c.Store.GetInstallation(ci.InstallationID, false, false)
				if err != nil {
					c.Logger.WithError(err).Error("failed to retrieve installation")
					w.WriteHeader(http.StatusInternalServerError)
					return
				}
				installations = append(installations, installation)
			}
		}
		err = c.Store.MigrateInstallationsDNS(installations)
		if err != nil {
			c.Logger.WithError(err).Error("failed to migrate DNS records")
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		c.Logger.Infof("DNS Switch over has been completed for cluster %s: ", mcir.ClusterID)
	}

	w.WriteHeader(http.StatusOK)
}

// handleMigrateDns responds to Post /api/cluster_installation/migrate/dns.
func handleMigrateDNS(c *Context, w http.ResponseWriter, r *http.Request) {
	mcir, err := model.NewMigrateClusterInstallationRequestFromReader(r.Body)
	if err != nil {
		c.Logger.WithError(err).Error("failed to decode cluster migration request")
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	c.Logger = c.Logger.WithField("cluster_installation", mcir)

	if len(mcir.ClusterID) == 0 {
		c.Logger.WithError(err).Error("Missing mandatory primary cluster in a migration request")
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	if len(mcir.TargetCluster) == 0 {
		c.Logger.WithError(err).Error("Missing mandatory secondary cluster in a migration request")
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	// Reset the DNS configuration status for respective installations to update the CNAME with the new LB.
	isStaleClusterInstallations := false
	filter := &model.ClusterInstallationFilter{
		ClusterID:      mcir.ClusterID,
		InstallationID: mcir.InstallationID,
		Paging:         model.AllPagesNotDeleted(),
		IsStale:        &isStaleClusterInstallations,
	}
	clusterInstallations, err := c.Store.GetClusterInstallations(filter)
	if err != nil {
		c.Logger.WithError(err).Error("failed to query cluster installations")
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	if clusterInstallations == nil {
		c.Logger.WithError(err).Error("No matching active cluster installations found")
		w.WriteHeader(http.StatusNotFound)
		return
	}

	c.Logger.Infof("total DNS records to migrate: %s", len(clusterInstallations))
	var installations []*model.Installation
	if mcir.LockInstallation {
		c.Logger.Infof("Locking %s installation(s) ", len(clusterInstallations))
		for _, ci := range clusterInstallations {
			installationDTO, status, unlockOnce := lockInstallation(c, ci.InstallationID)
			if status != 0 {
				w.WriteHeader(status)
				return
			}
			defer unlockOnce()
			installations = append(installations, installationDTO.Installation)
		}
	} else {
		for _, ci := range clusterInstallations {
			installation, err := c.Store.GetInstallation(ci.InstallationID, false, false)
			if err != nil {
				c.Logger.WithError(err).Error("failed to retrieve installation")
				w.WriteHeader(http.StatusInternalServerError)
				return
			}
			installations = append(installations, installation)
		}
	}
	err = c.Store.MigrateInstallationsDNS(installations)
	if err != nil {
		c.Logger.WithError(err).Error("failed to migrate DNS records")
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	c.Logger.Infof("DNS Switch over has been completed for cluster %s: ", mcir.ClusterID)
	w.WriteHeader(http.StatusOK)
}

// handleDeleteStaleClusterInstallations responds to Delete /api/cluster_installation/migrate/delete_stale/clusterID.
func handleDeleteStaleClusterInstallationsByCluster(c *Context, w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	clusterID := vars["clusterID"]
	c.Logger = c.Logger.WithField("clusterID", clusterID)
	if len(clusterID) == 0 {
		c.Logger.Error("Missing mandatory primary cluster in a migration request")
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	// Deleting multiple stale cluster installations
	c.Logger.Infof("Deleting stale cluster installations for cluster ID %s", clusterID)
	err := c.Store.DeleteStaleClusterInstallationByClusterID(clusterID)
	if err != nil {
		c.Logger.WithError(err).Error("failed to delete stale cluster installations for cluster ID", clusterID)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
}

// handleDeleteStaleClusterInstallationByID responds to Post /api/cluster_installation/migrate/delete_stale/ID.
func handleDeleteStaleClusterInstallationByID(c *Context, w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	clusterInstallationID := vars["ClusterInstallationID"]
	c.Logger = c.Logger.WithField("ClusterInstallationID", clusterInstallationID)
	if len(clusterInstallationID) == 0 {
		c.Logger.Error("Missing mandatory cluster installation id in a migration request")
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	// Delete single stale cluster installation
	err := c.Store.DeleteClusterInstallation(clusterInstallationID)
	if err != nil {
		c.Logger.WithError(err).Errorf("failed to delete stale cluster installation %s", clusterInstallationID)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
}
