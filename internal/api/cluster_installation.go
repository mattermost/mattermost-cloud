// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.
//

package api

import (
	"fmt"
	"net/http"

	"github.com/gorilla/mux"
	"github.com/mattermost/mattermost-cloud/model"
	log "github.com/sirupsen/logrus"
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
	clusterInstallationsRouter.Handle("/migrate/delete_inactive/{clusterID}", addContext(handleDeleteInActiveClusterInstallationsByCluster)).Methods("DELETE")
	clusterInstallationsRouter.Handle("/migrate/delete_inactive/cluster_installation/{ClusterInstallationID}", addContext(handleDeleteInActiveClusterInstallationByID)).Methods("DELETE")
	clusterInstallationsRouter.Handle("/migrate/switch_cluster_roles", addContext(handleSwitchClusterRoles)).Methods("POST")

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
		c.Logger.WithError(err).Error("failed to decode cluster migration request", mcir)
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	c.Logger = c.Logger.WithFields(log.Fields{
		"source-cluster-id": mcir.SourceClusterID,
		"target-cluster-id": mcir.TargetClusterID,
	})

	filter := &model.ClusterInstallationFilter{
		ClusterID:      mcir.SourceClusterID,
		InstallationID: mcir.InstallationID,
		Paging:         model.AllPagesNotDeleted(),
	}

	clusterInstallations, err := c.Store.GetClusterInstallations(filter)
	if err != nil {
		c.Logger.WithError(err).Error("failed to query cluster installations")
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	if len(clusterInstallations) == 0 {
		c.Logger.WithError(err).Error("no matching cluster installations found")
		w.WriteHeader(http.StatusNotFound)
		return
	}

	// verify that the allows installation is false for the source cluster before migration starts
	cluster, err := c.Store.GetCluster(mcir.SourceClusterID)
	if cluster == nil || err != nil {
		c.Logger.WithError(err).Error("failed to get cluster")
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	// Migrate the cluster installations to the target cluster
	if cluster.AllowInstallations {
		c.Logger.WithError(err).Error("allow installation must be set to false for the source cluster.")
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	c.Logger.Infof("migrating installation(s) to clusterID: %s", mcir.TargetClusterID)
	err = c.Store.MigrateClusterInstallations(clusterInstallations, mcir.TargetClusterID)
	if err != nil {
		c.Logger.WithError(err).Error("failed to migrate cluster installation(s)")
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
}

// handleMigrateDns responds to Post /api/cluster_installation/migrate/dns.
func handleMigrateDNS(c *Context, w http.ResponseWriter, r *http.Request) {
	mcir, err := model.NewMigrateClusterInstallationRequestFromReader(r.Body)
	if err != nil {
		c.Logger.WithError(err).Error("failed to decode cluster migration request", mcir)
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	c.Logger = c.Logger.WithFields(log.Fields{
		"source-cluster-id": mcir.SourceClusterID,
		"target-cluster-id": mcir.TargetClusterID,
	})

	// Reset the DNS configuration status for respective installations to update the CNAME with the new LB.
	filter := &model.ClusterInstallationFilter{
		ClusterID:      mcir.SourceClusterID,
		InstallationID: mcir.InstallationID,
		Paging:         model.AllPagesNotDeleted(),
	}
	clusterInstallations, err := c.Store.GetClusterInstallations(filter)
	if err != nil {
		c.Logger.WithError(err).Error("failed to query cluster installations")
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	if len(clusterInstallations) == 0 {
		c.Logger.WithError(err).Error("no matching cluster installations found")
		w.WriteHeader(http.StatusNotFound)
		return
	}

	filter = &model.ClusterInstallationFilter{
		ClusterID:      mcir.TargetClusterID,
		InstallationID: mcir.InstallationID,
		Paging:         model.AllPagesNotDeleted(),
	}
	newClusterInstallations, err := c.Store.GetClusterInstallations(filter)
	if err != nil {
		c.Logger.WithError(err).Error("failed to query cluster installations")
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	if len(newClusterInstallations) == 0 {
		c.Logger.WithError(err).Error("no matching cluster installations found")
		w.WriteHeader(http.StatusNotFound)
		return
	}

	// verify that the allows installation is false for the source cluster before migration starts
	cluster, err := c.Store.GetCluster(mcir.SourceClusterID)
	if cluster == nil || err != nil {
		c.Logger.WithError(err).Error("failed to get cluster")
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	// DNS Switch
	if cluster.AllowInstallations {
		c.Logger.WithError(err).Error("allow installation must be set to false for the source cluster.")
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	clusterInstallationIDs := getClusterInstallationIDs(clusterInstallations)
	newClusterInstallationIDs := getClusterInstallationIDs(newClusterInstallations)
	c.Logger.Infof("total DNS records to migrate: %s", len(clusterInstallations))
	var installationIDs []string
	for _, ci := range clusterInstallations {
		installationIDs = append(installationIDs, ci.InstallationID)
	}
	status := dnsMigration(c, mcir, clusterInstallationIDs, newClusterInstallationIDs, installationIDs)
	if status != 0 {
		c.Logger.Error("failed to migrate DNS records")
		w.WriteHeader(status)
		return
	}
	w.WriteHeader(http.StatusOK)
}

// handleDeleteInActiveClusterInstallationsByCluster responds to Delete /api/cluster_installation/migrate/delete_inactive/clusterID.
func handleDeleteInActiveClusterInstallationsByCluster(c *Context, w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	clusterID := vars["clusterID"]
	c.Logger = c.Logger.WithField("clusterID", clusterID)
	if len(clusterID) == 0 {
		c.Logger.Error("missing mandatory source cluster in a migration request")
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	// Deleting multiple inactive cluster installations
	c.Logger.Infof("deleting inactive cluster installations for cluster ID %s", clusterID)
	err := c.Store.DeleteInActiveClusterInstallationByClusterID(clusterID)
	if err != nil {
		c.Logger.WithError(err).Error("failed to delete inactive cluster installations for cluster ID", clusterID)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	c.Logger.Infof("successfully deleted inactive cluster installations for cluster ID %s", clusterID)
	w.WriteHeader(http.StatusOK)
}

// handleDeleteInActiveClusterInstallationByID responds to Post /api/cluster_installation/migrate/delete_inactive/ID.
func handleDeleteInActiveClusterInstallationByID(c *Context, w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	clusterInstallationID := vars["ClusterInstallationID"]
	c.Logger = c.Logger.WithField("ClusterInstallationID", clusterInstallationID)
	if len(clusterInstallationID) == 0 {
		c.Logger.Error("missing mandatory cluster installation id in a migration request")
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	// Delete single inactive cluster installation
	clusterInstallation, err := c.Store.GetClusterInstallation(clusterInstallationID)
	if err != nil {
		c.Logger.WithError(err).Errorf("unable to retrieve inactive cluster installation for deletion %s", clusterInstallationID)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	clusterInstallation.State = model.ClusterInstallationStateDeletionRequested
	err = c.Store.UpdateClusterInstallation(clusterInstallation)
	if err != nil {
		c.Logger.WithError(err).Errorf("failed to delete inactive cluster installation %s", clusterInstallationID)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	c.Logger.Infof("successfully deleted inactive cluster installations ID 5s", clusterInstallationID)
	w.WriteHeader(http.StatusOK)
}

func dnsMigration(c *Context, mcir model.MigrateClusterInstallationRequest, oldClusterInstallationIDs []string, newClusterInstallationIDs []string, installationIDs []string) int {

	if mcir.LockInstallation {
		c.Logger.Infof("Locking %d installation(s) ", len(installationIDs))
		locked, err := c.Store.LockInstallations(installationIDs, c.RequestID)
		if err != nil {
			c.Logger.WithError(err).Error("failed to lock installation")
			return http.StatusInternalServerError
		} else if !locked {
			c.Logger.Error("failed to acquire lock for installation")
			return http.StatusInternalServerError
		}
		defer func() {
			unlocked, err := c.Store.UnlockInstallations(installationIDs, c.RequestID, false)
			if err != nil {
				c.Logger.WithError(err).Errorf("failed to unlock installation")
			} else if !unlocked {
				c.Logger.Warn("failed to release lock for installation")
			}
		}()
	}
	err := c.Store.SwitchDNS(oldClusterInstallationIDs, newClusterInstallationIDs, installationIDs)
	if err != nil {
		c.Logger.WithError(err).Error("failed to migrate DNS records")
		return http.StatusInternalServerError
	}

	c.Logger.Infof("DNS Switch over has been processed for cluster %s: ", mcir.SourceClusterID)
	return 0
}

func getClusterInstallationIDs(clusterInstallations []*model.ClusterInstallation) []string {
	clusterInstallationIDs := make([]string, 0, len(clusterInstallations))
	for _, clusterInstallation := range clusterInstallations {
		clusterInstallationIDs = append(clusterInstallationIDs, clusterInstallation.ID)
	}

	return clusterInstallationIDs
}

// handleSwitchClusterRoles responds to Post /api/cluster_installations/migrate/switch_cluster_roles.
func handleSwitchClusterRoles(c *Context, w http.ResponseWriter, r *http.Request) {
	mcir, err := model.NewMigrateClusterInstallationRequestFromReader(r.Body)
	if err != nil {
		c.Logger.WithError(err).Error("failed to decode cluster migration request")
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	c.Logger = c.Logger.WithField("cluster_installation", mcir)

	err = c.AwsClient.SwithClusterTags(mcir.SourceClusterID, mcir.TargetClusterID, c.Logger)
	if err != nil {
		c.Logger.WithError(err).Error("failed to switch cluster tags")
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusOK)
}
