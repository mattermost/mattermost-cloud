package api

import (
	"net/http"
	"time"

	"github.com/gorilla/mux"
	"github.com/mattermost/mattermost-cloud/internal/webhook"
	"github.com/mattermost/mattermost-cloud/model"
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
	clusterRouter.Handle("", addContext(handleUpdateClusterConfiguration)).Methods("PUT")
	clusterRouter.Handle("/provision", addContext(handleProvisionCluster)).Methods("POST")
	clusterRouter.Handle("/kubernetes/{version}", addContext(handleUpgradeKubernetes)).Methods("PUT")
	clusterRouter.Handle("", addContext(handleDeleteCluster)).Methods("DELETE")

	clusterRouter.Handle("/utilities", addContext(handleGetAllUtilityMetadata)).Methods("GET")
}

func handleGetAllUtilityMetadata(c *Context, w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	clusterID := vars["cluster"]
	c.Logger = c.Logger.WithField("cluster", clusterID).WithField("action", "get-utilities")

	cluster, err := c.Store.GetCluster(clusterID)
	if err != nil {
		c.Logger.WithError(err).Errorf("failed to look up cluster %s", clusterID)
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusAccepted)
	w.Write(cluster.UtilityMetadata)
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
	w.WriteHeader(http.StatusOK)
	outputJSON(c, w, clusters)
}

// handleCreateCluster responds to POST /api/clusters, beginning the process of creating a new
// cluster.
// sample body:
// {
//		"provider": "aws",
//		"version": "1.15.0",
//		"kops-ami": "ami-xoxoxo",
//		"size": "SizeAlef1000",
//		"zones": "",
//		"allow-installations": true
// }
func handleCreateCluster(c *Context, w http.ResponseWriter, r *http.Request) {
	createClusterRequest, err := model.NewCreateClusterRequestFromReader(r.Body)
	if err != nil {
		c.Logger.WithError(err).Error("failed to decode request")
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	cluster := model.Cluster{
		Provider:           createClusterRequest.Provider,
		Provisioner:        "kops",
		Version:            "0.0.0",
		Size:               createClusterRequest.Size,
		AllowInstallations: createClusterRequest.AllowInstallations,
		State:              model.ClusterStateCreationRequested,
	}

	err = cluster.SetProviderMetadata(model.AWSMetadata{
		Zones: createClusterRequest.Zones,
	})

	if err != nil {
		c.Logger.WithError(err).Error("failed to set provider metadata")
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	err = cluster.SetUtilityDesiredVersions(createClusterRequest.DesiredUtilityVersions)
	if err != nil {
		c.Logger.WithError(err).Errorf("provided utility metadata could not be applied without error")
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	err = cluster.SetProvisionerMetadata(model.KopsMetadata{
		Version: createClusterRequest.Version,
		AMI:     createClusterRequest.KopsAMI,
	})

	if err != nil {
		c.Logger.WithError(err).Error("failed to set provisioner metadata")
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	err = c.Store.CreateCluster(&cluster)
	if err != nil {
		c.Logger.WithError(err).Error("failed to create cluster")
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	webhookPayload := &model.WebhookPayload{
		Type:      model.TypeCluster,
		ID:        cluster.ID,
		NewState:  model.ClusterStateCreationRequested,
		OldState:  "n/a",
		Timestamp: time.Now().UnixNano(),
	}
	err = webhook.SendToAllWebhooks(c.Store, webhookPayload, c.Logger.WithField("webhookEvent", webhookPayload.NewState))
	if err != nil {
		c.Logger.WithError(err).Error("Unable to process and send webhooks")
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

	newState := model.ClusterStateCreationRequested

	if !cluster.ValidTransitionState(newState) {
		c.Logger.Warnf("unable to retry cluster creation while in state %s", cluster.State)
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	if cluster.State != newState {
		webhookPayload := &model.WebhookPayload{
			Type:      model.TypeCluster,
			ID:        cluster.ID,
			NewState:  newState,
			OldState:  cluster.State,
			Timestamp: time.Now().UnixNano(),
		}
		cluster.State = newState

		err := c.Store.UpdateCluster(cluster)
		if err != nil {
			c.Logger.WithError(err).Errorf("failed to retry cluster creation")
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		err = webhook.SendToAllWebhooks(c.Store, webhookPayload, c.Logger.WithField("webhookEvent", webhookPayload.NewState))
		if err != nil {
			c.Logger.WithError(err).Error("Unable to process and send webhooks")
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

	provisionClusterRequest, err := model.NewProvisionClusterRequestFromReader(r.Body)
	if err != nil {
		c.Logger.WithError(err).Errorf("failed to deserialize cluster provision request body")
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	err = cluster.SetUtilityDesiredVersions(provisionClusterRequest.DesiredUtilityVersions)
	if err != nil {
		c.Logger.WithError(err).Errorf("provided utility metadata could not be applied without error")
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	newState := model.ClusterStateProvisioningRequested

	if !cluster.ValidTransitionState(newState) {
		c.Logger.Warnf("unable to provision cluster while in state %s", cluster.State)
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	if cluster.State != newState {
		webhookPayload := &model.WebhookPayload{
			Type:      model.TypeCluster,
			ID:        cluster.ID,
			NewState:  newState,
			OldState:  cluster.State,
			Timestamp: time.Now().UnixNano(),
		}
		cluster.State = newState

		err := c.Store.UpdateCluster(cluster)
		if err != nil {
			c.Logger.WithError(err).Errorf("failed to mark cluster provisioning state")
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		err = webhook.SendToAllWebhooks(c.Store, webhookPayload, c.Logger.WithField("webhookEvent", webhookPayload.NewState))
		if err != nil {
			c.Logger.WithError(err).Error("Unable to process and send webhooks")
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
	w.WriteHeader(http.StatusOK)
	outputJSON(c, w, cluster)
}

// handleUpdateClusterConfiguration responds to PUT /api/cluster/{cluster}, updating a cluster's
// configuration.
func handleUpdateClusterConfiguration(c *Context, w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	clusterID := vars["cluster"]
	c.Logger = c.Logger.WithField("cluster", clusterID)

	cluster, status, unlockOnce := lockCluster(c, clusterID)
	if status != 0 {
		w.WriteHeader(status)
		return
	}
	defer unlockOnce()

	updateClusterRequest, err := model.NewUpdateClusterRequestFromReader(r.Body)
	if err != nil {
		c.Logger.WithError(err).Error("failed to decode request")
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	if cluster.AllowInstallations != updateClusterRequest.AllowInstallations {
		cluster.AllowInstallations = updateClusterRequest.AllowInstallations
		err := c.Store.UpdateCluster(cluster)
		if err != nil {
			c.Logger.WithError(err).Error("failed to update cluster")
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
	}

	unlockOnce()

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusAccepted)
	outputJSON(c, w, cluster)
}

// handleUpgradeKubernetes responds to PUT /api/cluster/{cluster}/kubernetes/{version},
// upgrading the cluster to the given Kubernetes version.
func handleUpgradeKubernetes(c *Context, w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	clusterID := vars["cluster"]
	version := vars["version"]
	c.Logger = c.Logger.WithField("cluster", clusterID)

	if !model.ValidClusterVersion(version) {
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

	newState := model.ClusterStateUpgradeRequested

	if !cluster.ValidTransitionState(newState) {
		c.Logger.Warnf("unable to upgrade cluster while in state %s", cluster.State)
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	kopsMetadata, err := model.NewKopsMetadata(cluster.ProvisionerMetadata)
	if err != nil {
		c.Logger.WithError(err).Error("failed to parse existing provisioner metadata")
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	if kopsMetadata.Version != version {
		kopsMetadata.Version = version
		err = cluster.SetProvisionerMetadata(kopsMetadata)
		if err != nil {
			c.Logger.WithError(err).Error("failed to set provisioner metadata")
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		err := c.Store.UpdateCluster(cluster)
		if err != nil {
			c.Logger.WithError(err).Error("failed to cluster version")
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
	}

	if cluster.State != newState {
		webhookPayload := &model.WebhookPayload{
			Type:      model.TypeCluster,
			ID:        cluster.ID,
			NewState:  newState,
			OldState:  cluster.State,
			Timestamp: time.Now().UnixNano(),
		}
		cluster.State = newState

		err := c.Store.UpdateCluster(cluster)
		if err != nil {
			c.Logger.WithError(err).Error("failed to mark cluster for upgrade")
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		err = webhook.SendToAllWebhooks(c.Store, webhookPayload, c.Logger.WithField("webhookEvent", webhookPayload.NewState))
		if err != nil {
			c.Logger.WithError(err).Error("Unable to process and send webhooks")
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

	newState := model.ClusterInstallationStateDeletionRequested

	if !cluster.ValidTransitionState(newState) {
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

	if cluster.State != newState {
		webhookPayload := &model.WebhookPayload{
			Type:      model.TypeCluster,
			ID:        cluster.ID,
			NewState:  newState,
			OldState:  cluster.State,
			Timestamp: time.Now().UnixNano(),
		}
		cluster.State = newState

		err := c.Store.UpdateCluster(cluster)
		if err != nil {
			c.Logger.WithError(err).Error("failed to mark cluster for deletion")
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		err = webhook.SendToAllWebhooks(c.Store, webhookPayload, c.Logger.WithField("webhookEvent", webhookPayload.NewState))
		if err != nil {
			c.Logger.WithError(err).Error("Unable to process and send webhooks")
		}
	}

	unlockOnce()
	c.Supervisor.Do()

	w.WriteHeader(http.StatusAccepted)
}
