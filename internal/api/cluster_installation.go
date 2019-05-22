package api

import (
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

	clusterInstallationRouter := apiRouter.PathPrefix("/cluster_installation/{cluster_installation:[A-Za-z0-9]{26}}").Subrouter()
	clusterInstallationRouter.Handle("", addContext(handleGetClusterInstallation)).Methods("GET")
	clusterInstallationRouter.Handle("/config", addContext(handleGetClusterInstallationConfig)).Methods("GET")
	clusterInstallationRouter.Handle("/config", addContext(handleSetClusterInstallationConfig)).Methods("PUT")
}

// handleGetClusterInstallations responds to GET /api/cluster_installations, returning the specified page of cluster installations.
func handleGetClusterInstallations(c *Context, w http.ResponseWriter, r *http.Request) {
	clusterID := r.URL.Query().Get("cluster")
	installationID := r.URL.Query().Get("installation")

	page, perPage, includeDeleted, err := parsePaging(r.URL)
	if err != nil {
		c.Logger.WithError(err).Error("failed to parse paging parameters")
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	filter := &model.ClusterInstallationFilter{
		ClusterID:      clusterID,
		InstallationID: installationID,
		Page:           page,
		PerPage:        perPage,
		IncludeDeleted: includeDeleted,
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
