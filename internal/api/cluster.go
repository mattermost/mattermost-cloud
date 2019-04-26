package api

import (
	"encoding/json"
	"io"
	"net/http"
	"strconv"

	"github.com/gorilla/mux"
	"github.com/mattermost/mattermost-cloud/internal/model"
)

// outputJSON is a helper method to write the given data as JSON to the given writer.
//
// It only logs an error if one occurs, rather than returning, since there is no point in trying
// to send a new status code back to the client once the body has started sending.
func outputJSON(c *Context, w io.Writer, data interface{}) {
	encoder := json.NewEncoder(w)
	err := encoder.Encode(data)
	if err != nil {
		c.Logger.WithField("error", err).Error("failed to encode result")
	}
}

// handleGetClusters responds to GET /api/clusters.
func handleGetClusters(c *Context, w http.ResponseWriter, r *http.Request) {
	pageStr := r.URL.Query().Get("page")
	page, err := strconv.Atoi(pageStr)
	if err != nil {
		c.Logger.WithField("error", err).Error("failed to parse page")
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	perPageStr := r.URL.Query().Get("per_page")
	perPage := 100
	if perPageStr != "" {
		perPage, err = strconv.Atoi(perPageStr)
		if err != nil {
			c.Logger.WithField("error", err).Error("failed to parse perPage")
			w.WriteHeader(http.StatusBadRequest)
			return
		}
	}

	includeDeletedStr := r.URL.Query().Get("include_deleted")
	includeDeleted := includeDeletedStr == "true"

	clusters, err := c.SQLStore.GetClusters(page, perPage, includeDeleted)
	if err != nil {
		c.Logger.WithField("error", err).Error("failed to query clusters")
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	if clusters == nil {
		clusters = []*model.Cluster{}
	}

	outputJSON(c, w, clusters)
}

// handleCreateCluster responds to POST /api/clusters
func handleCreateCluster(c *Context, w http.ResponseWriter, r *http.Request) {
	createClusterRequest, err := newCreateClusterRequestFromReader(r.Body)
	if err != nil {
		c.Logger.WithField("error", err).Error("failed to decode request")
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	cluster, err := c.Provisioner.CreateCluster(createClusterRequest)
	if err != nil {
		c.Logger.WithField("error", err).Error("failed to create cluster")
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	outputJSON(c, w, cluster)
}

// handleGetCluster responds to GET /api/clusters/{cluster}.
func handleGetCluster(c *Context, w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	clusterID := vars["cluster"]

	cluster, err := c.SQLStore.GetCluster(clusterID)
	if err != nil {
		c.Logger.WithField("error", err).Error("failed to query cluster")
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	if cluster == nil {
		w.WriteHeader(http.StatusNotFound)
		return
	}

	outputJSON(c, w, cluster)
}

// handleUpgradeCluster responds to PUT /api/clusters/{cluster}/kubernetes/{version}.
func handleUpgradeCluster(c *Context, w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	clusterID := vars["cluster"]
	version := vars["version"]

	cluster, err := c.SQLStore.GetCluster(clusterID)
	if err != nil {
		c.Logger.WithField("error", err).Error("failed to query cluster")
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	if cluster == nil {
		w.WriteHeader(http.StatusNotFound)
		return
	}

	err = c.Provisioner.UpgradeCluster(
		clusterID,
		version,
	)
	if err != nil {
		c.Logger.WithField("error", err).Error("failed to upgrade cluster")
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
}

// handleDeleteCluster responds to DELETE /api/clusters/{cluster}.
func handleDeleteCluster(c *Context, w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	clusterID := vars["cluster"]

	cluster, err := c.SQLStore.GetCluster(clusterID)
	if err != nil {
		c.Logger.WithField("error", err).Error("failed to query cluster")
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	if cluster == nil {
		w.WriteHeader(http.StatusNotFound)
		return
	}

	err = c.Provisioner.DeleteCluster(clusterID)
	if err != nil {
		c.Logger.WithField("error", err).Error("failed to delete cluster")
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
}
