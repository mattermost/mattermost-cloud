// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.
//

package api

import (
	"net/http"
	"time"

	"github.com/mattermost/mattermost-cloud/internal/store"
	"github.com/pkg/errors"

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
	clusterRouter.Handle("/kubernetes", addContext(handleUpgradeKubernetes)).Methods("PUT")
	clusterRouter.Handle("/size", addContext(handleResizeCluster)).Methods("PUT")
	clusterRouter.Handle("/utilities", addContext(handleGetAllUtilityMetadata)).Methods("GET")
	clusterRouter.Handle("/annotations", addContext(handleAddClusterAnnotations)).Methods("POST")
	clusterRouter.Handle("/annotation/{annotation-name}", addContext(handleDeleteClusterAnnotation)).Methods("DELETE")

	clusterRouter.Handle("", addContext(handleDeleteCluster)).Methods("DELETE")
}

// handleGetCluster responds to GET /api/cluster/{cluster}, returning the cluster in question.
func handleGetCluster(c *Context, w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	clusterID := vars["cluster"]
	c.Logger = c.Logger.WithField("cluster", clusterID)

	cluster, err := c.Store.GetClusterDTO(clusterID)
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

	clusters, err := c.Store.GetClusterDTOs(filter)
	if err != nil {
		c.Logger.WithError(err).Error("failed to query clusters")
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	if clusters == nil {
		clusters = []*model.ClusterDTO{}
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
		Provider: createClusterRequest.Provider,
		ProviderMetadataAWS: &model.AWSMetadata{
			Zones: createClusterRequest.Zones,
		},
		Provisioner: "kops",
		ProvisionerMetadataKops: &model.KopsMetadata{
			ChangeRequest: &model.KopsMetadataRequestedState{
				Version:            createClusterRequest.Version,
				AMI:                createClusterRequest.KopsAMI,
				MasterInstanceType: createClusterRequest.MasterInstanceType,
				MasterCount:        createClusterRequest.MasterCount,
				NodeInstanceType:   createClusterRequest.NodeInstanceType,
				NodeMinCount:       createClusterRequest.NodeMinCount,
				NodeMaxCount:       createClusterRequest.NodeMaxCount,
			},
		},
		AllowInstallations: createClusterRequest.AllowInstallations,
		APISecurityLock:    createClusterRequest.APISecurityLock,
		State:              model.ClusterStateCreationRequested,
	}

	err = cluster.SetUtilityDesiredVersions(createClusterRequest.DesiredUtilityVersions)
	if err != nil {
		c.Logger.WithError(err).Error("provided utility metadata could not be applied without error")
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	annotations, err := model.AnnotationsFromStringSlice(createClusterRequest.Annotations)
	if err != nil {
		c.Logger.WithError(err).Error("failed to validate extra annotations")
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	err = c.Store.CreateCluster(&cluster, annotations)
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
	outputJSON(c, w, cluster.ToDTO(annotations))
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

	clusterDTO, status, unlockOnce := lockCluster(c, clusterID)
	if status != 0 {
		w.WriteHeader(status)
		return
	}
	defer unlockOnce()

	newState := model.ClusterStateCreationRequested

	if !clusterDTO.ValidTransitionState(newState) {
		c.Logger.Warnf("unable to retry cluster creation while in state %s", clusterDTO.State)
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	if clusterDTO.State != newState {
		webhookPayload := &model.WebhookPayload{
			Type:      model.TypeCluster,
			ID:        clusterDTO.ID,
			NewState:  newState,
			OldState:  clusterDTO.State,
			Timestamp: time.Now().UnixNano(),
		}
		clusterDTO.State = newState

		err := c.Store.UpdateCluster(clusterDTO.Cluster)
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
	outputJSON(c, w, clusterDTO)
}

// handleProvisionCluster responds to POST /api/cluster/{cluster}/provision,
// provisioning k8s resources on a previously-created cluster.
func handleProvisionCluster(c *Context, w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	clusterID := vars["cluster"]
	c.Logger = c.Logger.WithField("cluster", clusterID)

	clusterDTO, status, unlockOnce := lockCluster(c, clusterID)
	if status != 0 {
		w.WriteHeader(status)
		return
	}
	defer unlockOnce()

	if clusterDTO.APISecurityLock {
		logSecurityLockConflict("cluster", c.Logger)
		w.WriteHeader(http.StatusForbidden)
		return
	}

	provisionClusterRequest, err := model.NewProvisionClusterRequestFromReader(r.Body)
	if err != nil {
		c.Logger.WithError(err).Error("failed to deserialize cluster provision request body")
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	err = clusterDTO.SetUtilityDesiredVersions(provisionClusterRequest.DesiredUtilityVersions)
	if err != nil {
		c.Logger.WithError(err).Error("provided utility metadata could not be applied without error")
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	newState := model.ClusterStateProvisioningRequested

	if !clusterDTO.ValidTransitionState(newState) {
		c.Logger.Warnf("unable to provision cluster while in state %s", clusterDTO.State)
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	if clusterDTO.State != newState {
		webhookPayload := &model.WebhookPayload{
			Type:      model.TypeCluster,
			ID:        clusterDTO.ID,
			NewState:  newState,
			OldState:  clusterDTO.State,
			Timestamp: time.Now().UnixNano(),
		}
		clusterDTO.State = newState

		err := c.Store.UpdateCluster(clusterDTO.Cluster)
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
	outputJSON(c, w, clusterDTO)
}

// handleUpdateClusterConfiguration responds to PUT /api/cluster/{cluster}, updating a cluster's
// configuration.
func handleUpdateClusterConfiguration(c *Context, w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	clusterID := vars["cluster"]
	c.Logger = c.Logger.WithField("cluster", clusterID)

	clusterDTO, status, unlockOnce := lockCluster(c, clusterID)
	if status != 0 {
		w.WriteHeader(status)
		return
	}
	defer unlockOnce()

	if clusterDTO.APISecurityLock {
		logSecurityLockConflict("cluster", c.Logger)
		w.WriteHeader(http.StatusForbidden)
		return
	}

	updateClusterRequest, err := model.NewUpdateClusterRequestFromReader(r.Body)
	if err != nil {
		c.Logger.WithError(err).Error("failed to decode request")
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	if clusterDTO.AllowInstallations != updateClusterRequest.AllowInstallations {
		clusterDTO.AllowInstallations = updateClusterRequest.AllowInstallations
		err := c.Store.UpdateCluster(clusterDTO.Cluster)
		if err != nil {
			c.Logger.WithError(err).Error("failed to update cluster")
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
	}

	unlockOnce()

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusAccepted)
	outputJSON(c, w, clusterDTO)
}

// handleUpgradeKubernetes responds to PUT /api/cluster/{cluster}/kubernetes,
// upgrading the cluster to the given Kubernetes version and AMI image.
func handleUpgradeKubernetes(c *Context, w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	clusterID := vars["cluster"]
	c.Logger = c.Logger.WithField("cluster", clusterID)

	upgradeClusterRequest, err := model.NewUpgradeClusterRequestFromReader(r.Body)
	if err != nil {
		c.Logger.WithError(err).Error("failed to decode request")
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	clusterDTO, status, unlockOnce := lockCluster(c, clusterID)
	if status != 0 {
		w.WriteHeader(status)
		return
	}
	defer unlockOnce()

	if clusterDTO.APISecurityLock {
		logSecurityLockConflict("cluster", c.Logger)
		w.WriteHeader(http.StatusForbidden)
		return
	}

	oldState := clusterDTO.State
	newState := model.ClusterStateUpgradeRequested

	if !clusterDTO.ValidTransitionState(newState) {
		c.Logger.Warnf("unable to upgrade cluster while in state %s", clusterDTO.State)
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	if upgradeClusterRequest.Apply(clusterDTO.ProvisionerMetadataKops) {
		clusterDTO.State = newState
		err := c.Store.UpdateCluster(clusterDTO.Cluster)
		if err != nil {
			c.Logger.WithError(err).Error("failed to update cluster")
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		if oldState != newState {
			webhookPayload := &model.WebhookPayload{
				Type:      model.TypeCluster,
				ID:        clusterDTO.ID,
				NewState:  newState,
				OldState:  oldState,
				Timestamp: time.Now().UnixNano(),
			}

			err = webhook.SendToAllWebhooks(c.Store, webhookPayload, c.Logger.WithField("webhookEvent", webhookPayload.NewState))
			if err != nil {
				c.Logger.WithError(err).Error("Unable to process and send webhooks")
			}
		}
	}

	unlockOnce()
	c.Supervisor.Do()

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusAccepted)
	outputJSON(c, w, clusterDTO)
}

// handleResizeCluster responds to PUT /api/cluster/{cluster}/size,
// resizing the cluster.
func handleResizeCluster(c *Context, w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	clusterID := vars["cluster"]
	c.Logger = c.Logger.WithField("cluster", clusterID)

	resizeClusterRequest, err := model.NewResizeClusterRequestFromReader(r.Body)
	if err != nil {
		c.Logger.WithError(err).Error("failed to decode request")
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	clusterDTO, status, unlockOnce := lockCluster(c, clusterID)
	if status != 0 {
		w.WriteHeader(status)
		return
	}
	defer unlockOnce()

	if clusterDTO.APISecurityLock {
		logSecurityLockConflict("cluster", c.Logger)
		w.WriteHeader(http.StatusForbidden)
		return
	}

	// One more check that can't be done without both the request and the cluster.
	if resizeClusterRequest.NodeMinCount == nil &&
		resizeClusterRequest.NodeMaxCount != nil &&
		*resizeClusterRequest.NodeMaxCount < clusterDTO.ProvisionerMetadataKops.NodeMinCount {
		c.Logger.Error("resize patch would set max node count lower than min node count")
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	oldState := clusterDTO.State
	newState := model.ClusterStateResizeRequested

	if !clusterDTO.ValidTransitionState(newState) {
		c.Logger.Warnf("unable to resize cluster while in state %s", clusterDTO.State)
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	if resizeClusterRequest.Apply(clusterDTO.ProvisionerMetadataKops) {
		clusterDTO.State = newState
		err = c.Store.UpdateCluster(clusterDTO.Cluster)
		if err != nil {
			c.Logger.WithError(err).Error("failed to update cluster")
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		if oldState != newState {
			webhookPayload := &model.WebhookPayload{
				Type:      model.TypeCluster,
				ID:        clusterDTO.ID,
				NewState:  newState,
				OldState:  oldState,
				Timestamp: time.Now().UnixNano(),
			}

			err = webhook.SendToAllWebhooks(c.Store, webhookPayload, c.Logger.WithField("webhookEvent", webhookPayload.NewState))
			if err != nil {
				c.Logger.WithError(err).Error("Unable to process and send webhooks")
			}
		}
	}

	unlockOnce()
	c.Supervisor.Do()

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusAccepted)
	outputJSON(c, w, clusterDTO)
}

// handleDeleteCluster responds to DELETE /api/cluster/{cluster}, beginning the process of
// deleting the cluster.
func handleDeleteCluster(c *Context, w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	clusterID := vars["cluster"]
	c.Logger = c.Logger.WithField("cluster", clusterID)

	clusterDTO, status, unlockOnce := lockCluster(c, clusterID)
	if status != 0 {
		w.WriteHeader(status)
		return
	}
	defer unlockOnce()

	if clusterDTO.APISecurityLock {
		logSecurityLockConflict("cluster", c.Logger)
		w.WriteHeader(http.StatusForbidden)
		return
	}

	newState := model.ClusterInstallationStateDeletionRequested

	if !clusterDTO.ValidTransitionState(newState) {
		c.Logger.Warnf("unable to delete cluster while in state %s", clusterDTO.State)
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	clusterInstallations, err := c.Store.GetClusterInstallations(&model.ClusterInstallationFilter{
		ClusterID:      clusterDTO.ID,
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
		w.WriteHeader(http.StatusForbidden)
		return
	}

	if clusterDTO.State != newState {
		webhookPayload := &model.WebhookPayload{
			Type:      model.TypeCluster,
			ID:        clusterDTO.ID,
			NewState:  newState,
			OldState:  clusterDTO.State,
			Timestamp: time.Now().UnixNano(),
		}
		clusterDTO.State = newState

		err := c.Store.UpdateCluster(clusterDTO.Cluster)
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

func handleGetAllUtilityMetadata(c *Context, w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	clusterID := vars["cluster"]
	c.Logger = c.Logger.WithField("cluster", clusterID).WithField("action", "get-utilities")

	cluster, err := c.Store.GetCluster(clusterID)
	if err != nil {
		c.Logger.WithError(err).Errorf("failed to look up cluster %s", clusterID)
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	outputJSON(c, w, cluster.UtilityMetadata)
}

// handleAddClusterAnnotations responds to POST /api/cluster/{cluster}/annotations,
// adds the set of annotations to the Cluster.
func handleAddClusterAnnotations(c *Context, w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	clusterID := vars["cluster"]
	c.Logger = c.Logger.WithField("cluster", clusterID).WithField("action", "add-cluster-annotations")

	clusterDTO, status, unlockOnce := lockCluster(c, clusterID)
	if status != 0 {
		w.WriteHeader(status)
		return
	}
	defer unlockOnce()

	annotations, err := annotationsFromRequest(r)
	if err != nil {
		c.Logger.WithError(err).Error("failed to get annotations from request")
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	annotations, err = c.Store.CreateClusterAnnotations(clusterID, annotations)
	if err != nil {
		c.Logger.WithError(err).Error("failed to create cluster annotations")
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	clusterDTO.Annotations = append(clusterDTO.Annotations, annotations...)

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	outputJSON(c, w, clusterDTO)
}

// handleDeleteClusterAnnotation responds to DELETE /api/cluster/{cluster}/annotation/{annotation-name},
// removes annotation from the Cluster.
func handleDeleteClusterAnnotation(c *Context, w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	clusterID := vars["cluster"]
	annotationName := vars["annotation-name"]
	c.Logger = c.Logger.
		WithField("cluster", clusterID).
		WithField("action", "delete-cluster-annotation").
		WithField("annotation-name", annotationName)

	_, status, unlockOnce := lockCluster(c, clusterID)
	if status != 0 {
		w.WriteHeader(status)
		return
	}
	defer unlockOnce()

	err := c.Store.DeleteClusterAnnotation(clusterID, annotationName)
	if err != nil {
		c.Logger.WithError(err).Error("failed delete cluster annotation")
		if errors.Is(err, store.ErrClusterAnnotationUsedByInstallation) {
			w.WriteHeader(http.StatusForbidden)
			return
		}
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusNoContent)
}

func annotationsFromRequest(req *http.Request) ([]*model.Annotation, error) {
	annotationsRequest, err := model.NewAddAnnotationsRequestFromReader(req.Body)
	if err != nil {
		return nil, errors.Wrap(err, "failed to decode request")
	}
	defer req.Body.Close()

	annotations, err := model.AnnotationsFromStringSlice(annotationsRequest.Annotations)
	if err != nil {
		return nil, errors.Wrap(err, "failed to validate annotations")
	}

	return annotations, nil
}
