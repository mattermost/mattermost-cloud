package api

import (
	"net/http"

	"github.com/gorilla/mux"
	"github.com/mattermost/mattermost-cloud/internal/model"
)

// initCluster registers cluster endpoints on the given router.
func initCluster(apiRouter *mux.Router, context *Context) {
	addContext := func(handler contextHandlerFunc) *contextHandler {
		return newContextHandler(context, handler)
	}

	clustersRouter := apiRouter.PathPrefix("/clusters").Subrouter()
	clustersRouter.Handle("", addContext(handleGetClusters)).Methods("GET")
	clustersRouter.Handle("", addContext(handleCreateCluster)).Methods("POST")

	clusterRouter := apiRouter.PathPrefix("/cluster/{cluster:[A-Za-z0-9]{26}}").Subrouter()
	clusterRouter.Handle("", addContext(handleGetCluster)).Methods("GET")
	clusterRouter.Handle("", addContext(handleRetryCreateCluster)).Methods("POST")
	clusterRouter.Handle("/provision", addContext(handleProvisionCluster)).Methods("POST")
	clusterRouter.Handle("/kubernetes/{version}", addContext(handleUpgradeCluster)).Methods("PUT")
	clusterRouter.Handle("", addContext(handleDeleteCluster)).Methods("DELETE")
}

// handleGetClusters responds to GET /api/clusters, returning the specified page of clusters.
func handleGetClusters(c *Context, w http.ResponseWriter, r *http.Request) {
	page, perPage, includeDeleted, err := parsePaging(r.URL)
	if err != nil {
		c.Logger.WithError(err).Error("failed to parse paging parameters")
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	filter := &model.ClusterFilter{
		Page:           page,
		PerPage:        perPage,
		IncludeDeleted: includeDeleted,
	}

	clusters, err := c.Store.GetClusters(filter)
	if err != nil {
		c.Logger.WithError(err).Error("failed to query clusters")
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	if clusters == nil {
		clusters = []*model.Cluster{}
	}

	w.Header().Set("Content-Type", "application/json")
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

	w.Header().Set("Content-Type", "application/json")
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

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusAccepted)
	outputJSON(c, w, cluster)
}

// handleProvisionCluster responds to POST /api/cluster/{cluster}/provision,
// provisioning k8s resources on a previously-created cluster.
func handleProvisionCluster(c *Context, w http.ResponseWriter, r *http.Request) {
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
	case model.ClusterStateProvisioningFailed:
	case model.ClusterStateProvisioningRequested:
	default:
		c.Logger.Warnf("unable to provision cluster while in state %s", cluster.State)
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	if cluster.State != model.ClusterStateProvisioningRequested {
		cluster.State = model.ClusterStateProvisioningRequested

		err := c.Store.UpdateCluster(cluster)
		if err != nil {
			c.Logger.WithError(err).Errorf("failed to mark cluster provisioning state")
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
	}

	// Notify even if we didn't make changes, to expedite even the no-op operations above.
	unlockOnce()
	c.Supervisor.Do()

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusAccepted)
	outputJSON(c, w, cluster)
}

// handleGetCluster responds to GET /api/cluster/{cluster}, returning the cluster in question.
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

	w.Header().Set("Content-Type", "application/json")
	outputJSON(c, w, cluster)
}

// handleUpgradeCluster responds to PUT /api/cluster/{cluster}/kubernetes/{version}, upgrading
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

// handleDeleteCluster responds to DELETE /api/cluster/{cluster}, beginning the process of
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
	case model.ClusterStateCreationRequested:
	case model.ClusterStateCreationFailed:
	case model.ClusterStateProvisioningFailed:
	case model.ClusterStateUpgradeRequested:
	case model.ClusterStateUpgradeFailed:
	case model.ClusterStateDeletionRequested:
	case model.ClusterStateDeletionFailed:
	default:
		c.Logger.Warnf("unable to delete cluster while in state %s", cluster.State)
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	clusterInstallations, err := c.Store.GetClusterInstallations(&model.ClusterInstallationFilter{
		ClusterID:      cluster.ID,
		IncludeDeleted: false,
		PerPage:        model.AllPerPage,
	})
	if err != nil {
		c.Logger.WithError(err).Error("failed to get cluster installations")
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	if len(clusterInstallations) != 0 {
		c.Logger.Errorf("unable to delete cluster while it still has %d cluster installations", len(clusterInstallations))
		w.WriteHeader(http.StatusInternalServerError)
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
