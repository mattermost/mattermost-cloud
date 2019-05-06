package api

import (
	"encoding/json"
	"io"
	"net/http"
	"strconv"
	"sync"

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
		c.Logger.WithError(err).Error("failed to encode result")
	}
}

// lockCluster synchronizes access to the given cluster across potentially multiple provisioning
// servers.
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
				c.Logger.Warn("failed to release lock for cluster")
			}
			c.Logger.Infof("unlocked %v", clusterID)
		})
	}
}

// handleGetClusters responds to GET /api/clusters, returning the specified page of clusters.
func handleGetClusters(c *Context, w http.ResponseWriter, r *http.Request) {
	var err error
	pageStr := r.URL.Query().Get("page")
	page := 0
	if pageStr != "" {
		page, err = strconv.Atoi(pageStr)
		if err != nil {
			c.Logger.WithError(err).Error("failed to parse page")
			w.WriteHeader(http.StatusBadRequest)
			return
		}
	}

	perPageStr := r.URL.Query().Get("per_page")
	perPage := 100
	if perPageStr != "" {
		perPage, err = strconv.Atoi(perPageStr)
		if err != nil {
			c.Logger.WithError(err).Error("failed to parse perPage")
			w.WriteHeader(http.StatusBadRequest)
			return
		}
	}

	includeDeletedStr := r.URL.Query().Get("include_deleted")
	includeDeleted := includeDeletedStr == "true"

	clusters, err := c.Store.GetClusters(page, perPage, includeDeleted)
	if err != nil {
		c.Logger.WithError(err).Error("failed to query clusters")
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	if clusters == nil {
		clusters = []*model.Cluster{}
	}

	outputJSON(c, w, clusters)
}

// handleCreateCluster responds to POST /api/clusters, beginning the process of creating a new
// cluster.
func handleCreateCluster(c *Context, w http.ResponseWriter, r *http.Request) {
	createClusterRequest, err := newCreateClusterRequestFromReader(r.Body)
	if err != nil {
		c.Logger.WithError(err).Error("failed to decode request")
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	cluster := model.Cluster{
		Provider:    createClusterRequest.Provider,
		Provisioner: "kops",
		Size:        createClusterRequest.Size,
		State:       model.ClusterStateCreationRequested,
	}
	cluster.SetProviderMetadata(model.AWSMetadata{
		Zones: createClusterRequest.Zones,
	})

	err = c.Store.CreateCluster(&cluster)
	if err != nil {
		c.Logger.WithError(err).Error("failed to create cluster")
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	c.Supervisor.Do()

	w.WriteHeader(http.StatusAccepted)
	outputJSON(c, w, cluster)
}

// handleRetryCreateCluster responds to POST /api/cluster/{cluster}, retrying a previously
// failed creation.
//
// Note that other operations on a cluster may be retried by simply repeating the same request,
// but repeating handleCreateCluster would create a second cluster.
func handleRetryCreateCluster(c *Context, w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	clusterID := vars["cluster"]
	c.Logger = c.Logger.WithField("cluster", clusterID)

	cluster, status, unlockOnce := lockCluster(c, clusterID)
	if status != 0 {
		w.WriteHeader(status)
		return
	}
	defer unlockOnce()

	switch cluster.State {
	case model.ClusterStateCreationRequested:
	case model.ClusterStateCreationFailed:
	default:
		c.Logger.Warnf("unable to retry cluster creation while in state %s", cluster.State)
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	if cluster.State != model.ClusterStateCreationRequested {
		cluster.State = model.ClusterStateCreationRequested

		err := c.Store.UpdateCluster(cluster)
		if err != nil {
			c.Logger.WithError(err).Errorf("failed to retry cluster creation")
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
	}

	// Notify even if we didn't make changes, to expedite even the no-op operations above.
	unlockOnce()
	c.Supervisor.Do()

	w.WriteHeader(http.StatusAccepted)
	outputJSON(c, w, cluster)
}

// handleGetCluster responds to GET /api/clusters/{cluster}, returning the cluster in question.
func handleGetCluster(c *Context, w http.ResponseWriter, r *http.Request) {
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

	outputJSON(c, w, cluster)
}

// handleUpgradeCluster responds to PUT /api/clusters/{cluster}/kubernetes/{version}, upgrading
// the cluster to the given Kubernetes version.
func handleUpgradeCluster(c *Context, w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	clusterID := vars["cluster"]
	version := vars["version"]
	c.Logger = c.Logger.WithField("cluster", clusterID)

	// TODO: Support something other than "latest".
	if version != "latest" {
		c.Logger.Warnf("unsupported kubernetes version %s", version)
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	cluster, status, unlockOnce := lockCluster(c, clusterID)
	if status != 0 {
		w.WriteHeader(status)
		return
	}
	defer unlockOnce()

	switch cluster.State {
	case model.ClusterStateStable:
	case model.ClusterStateUpgradeRequested:
	case model.ClusterStateUpgradeFailed:
	default:
		c.Logger.Warnf("unable to upgrade cluster while in state %s", cluster.State)
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	if cluster.State != model.ClusterStateUpgradeRequested {
		cluster.State = model.ClusterStateUpgradeRequested

		err := c.Store.UpdateCluster(cluster)
		if err != nil {
			c.Logger.WithError(err).Error("failed to mark cluster for upgrade")
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
	}

	unlockOnce()
	c.Supervisor.Do()

	w.WriteHeader(http.StatusAccepted)
}

// handleDeleteCluster responds to DELETE /api/clusters/{cluster}, beginning the process of
// deleting the cluster.
func handleDeleteCluster(c *Context, w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	clusterID := vars["cluster"]
	c.Logger = c.Logger.WithField("cluster", clusterID)

	cluster, status, unlockOnce := lockCluster(c, clusterID)
	if status != 0 {
		w.WriteHeader(status)
		return
	}
	defer unlockOnce()

	switch cluster.State {
	case model.ClusterStateStable:
	case model.ClusterStateDeletionRequested:
	case model.ClusterStateDeletionFailed:
	default:
		c.Logger.Warnf("unable to delete cluster while in state %s", cluster.State)
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	if cluster.State != model.ClusterStateDeletionRequested {
		cluster.State = model.ClusterStateDeletionRequested

		err := c.Store.UpdateCluster(cluster)
		if err != nil {
			c.Logger.WithError(err).Error("failed to mark cluster for deletion")
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
	}

	unlockOnce()
	c.Supervisor.Do()

	w.WriteHeader(http.StatusAccepted)
}
