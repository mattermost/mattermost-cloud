// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.
//

package api

import (
	"net/http"

	"github.com/mattermost/mattermost-cloud/internal/store"
	"github.com/pkg/errors"

	"github.com/gorilla/mux"
	"github.com/mattermost/mattermost-cloud/model"
)

// initCluster registers cluster endpoints on the given router.
func initCluster(apiRouter *mux.Router, context *Context) {
	addContext := func(handler contextHandlerFunc, name string) *contextHandler {
		return newContextHandler(context, handler, name)
	}

	clustersRouter := apiRouter.PathPrefix("/clusters").Subrouter()
	clustersRouter.Handle("", addContext(handleGetClusters, "handleGetClusters")).Methods("GET")
	clustersRouter.Handle("", addContext(handleCreateCluster, "handleCreateCluster")).Methods("POST")

	clusterRouter := apiRouter.PathPrefix("/cluster/{cluster:[A-Za-z0-9]{26}}").Subrouter()
	clusterRouter.Handle("", addContext(handleGetCluster, "handleGetCluster")).Methods("GET")
	clusterRouter.Handle("", addContext(handleRetryCreateCluster, "handleRetryCreateCluster")).Methods("POST")
	clusterRouter.Handle("", addContext(handleUpdateClusterConfiguration, "handleUpdateClusterConfiguration")).Methods("PUT")
	clusterRouter.Handle("/provision", addContext(handleProvisionCluster, "handleProvisionCluster")).Methods("POST")
	clusterRouter.Handle("/kubernetes", addContext(handleUpgradeKubernetes, "handleUpgradeKubernetes")).Methods("PUT")
	clusterRouter.Handle("/size", addContext(handleResizeCluster, "handleResizeCluster")).Methods("PUT")
	clusterRouter.Handle("/utilities", addContext(handleGetAllUtilityMetadata, "handleGetAllUtilityMetadata")).Methods("GET")
	clusterRouter.Handle("/annotations", addContext(handleAddClusterAnnotations, "handleAddClusterAnnotations")).Methods("POST")
	clusterRouter.Handle("/annotation/{annotation-name}", addContext(handleDeleteClusterAnnotation, "handleDeleteClusterAnnotation")).Methods("DELETE")
	clusterRouter.Handle("", addContext(handleDeleteCluster, "handleDeleteCluster")).Methods("DELETE")
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
	paging, err := parsePaging(r.URL)
	if err != nil {
		c.Logger.WithError(err).Error("failed to parse paging parameters")
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	filter := &model.ClusterFilter{
		Paging: paging,
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
				MaxPodsPerNode:     createClusterRequest.MaxPodsPerNode,
				Networking:         createClusterRequest.Networking,
				VPC:                createClusterRequest.VPC,
			},
		},
		AllowInstallations: createClusterRequest.AllowInstallations,
		APISecurityLock:    createClusterRequest.APISecurityLock,
		State:              model.ClusterStateCreationRequested,
	}

	cluster.SetUtilityDesiredVersions(createClusterRequest.DesiredUtilityVersions)

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

	err = c.EventProducer.ProduceClusterStateChangeEvent(&cluster, model.NonApplicableState)
	if err != nil {
		c.Logger.WithError(err).Error("Failed to create cluster state change event")
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

	newState := model.ClusterStateCreationRequested

	clusterDTO, status, unlockOnce := getClusterForTransition(c, clusterID, newState)
	if status != 0 {
		w.WriteHeader(status)
		return
	}
	defer unlockOnce()

	if clusterDTO.State != newState {
		oldState := clusterDTO.State
		clusterDTO.State = newState

		err := c.Store.UpdateCluster(clusterDTO.Cluster)
		if err != nil {
			c.Logger.WithError(err).Errorf("failed to retry cluster creation")
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		err = c.EventProducer.ProduceClusterStateChangeEvent(clusterDTO.Cluster, oldState)
		if err != nil {
			c.Logger.WithError(err).Error("Failed to create cluster state change event")
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

	provisionClusterRequest, err := model.NewProvisionClusterRequestFromReader(r.Body)
	if err != nil {
		c.Logger.WithError(err).Error("failed to deserialize cluster provision request body")
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	newState := model.ClusterStateProvisioningRequested

	clusterDTO, status, unlockOnce := getClusterForTransition(c, clusterID, newState)
	if status != 0 {
		w.WriteHeader(status)
		return
	}
	defer unlockOnce()

	if provisionClusterRequest.Force {
		// set default values for utility versions
		provisionClusterRequest = supplyDefaultDesiredUtilityVersions(provisionClusterRequest, clusterDTO)
	}

	clusterDTO.SetUtilityDesiredVersions(provisionClusterRequest.DesiredUtilityVersions)

	if clusterDTO.State != newState {
		oldState := clusterDTO.State
		clusterDTO.State = newState

		err := c.Store.UpdateCluster(clusterDTO.Cluster)
		if err != nil {
			c.Logger.WithError(err).Errorf("failed to mark cluster provisioning state")
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		err = c.EventProducer.ProduceClusterStateChangeEvent(clusterDTO.Cluster, oldState)
		if err != nil {
			c.Logger.WithError(err).Error("Failed to create cluster state change event")
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

	newState := model.ClusterStateUpgradeRequested

	clusterDTO, status, unlockOnce := getClusterForTransition(c, clusterID, newState)
	if status != 0 {
		w.WriteHeader(status)
		return
	}
	defer unlockOnce()

	oldState := clusterDTO.State

	if upgradeClusterRequest.Apply(clusterDTO.ProvisionerMetadataKops) {
		clusterDTO.State = newState
		err := c.Store.UpdateCluster(clusterDTO.Cluster)
		if err != nil {
			c.Logger.WithError(err).Error("failed to update cluster")
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		if oldState != newState {
			err = c.EventProducer.ProduceClusterStateChangeEvent(clusterDTO.Cluster, oldState)
			if err != nil {
				c.Logger.WithError(err).Error("Failed to create cluster state change event")
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

	newState := model.ClusterStateResizeRequested

	clusterDTO, status, unlockOnce := getClusterForTransition(c, clusterID, newState)
	if status != 0 {
		w.WriteHeader(status)
		return
	}
	defer unlockOnce()

	// One more check that can't be done without both the request and the cluster.
	if resizeClusterRequest.NodeMinCount == nil &&
		resizeClusterRequest.NodeMaxCount != nil &&
		*resizeClusterRequest.NodeMaxCount < clusterDTO.ProvisionerMetadataKops.NodeMinCount {
		c.Logger.Error("resize patch would set max node count lower than min node count")
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	oldState := clusterDTO.State

	if resizeClusterRequest.Apply(clusterDTO.ProvisionerMetadataKops) {
		clusterDTO.State = newState
		err = c.Store.UpdateCluster(clusterDTO.Cluster)
		if err != nil {
			c.Logger.WithError(err).Error("failed to update cluster")
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		if oldState != newState {
			err = c.EventProducer.ProduceClusterStateChangeEvent(clusterDTO.Cluster, oldState)
			if err != nil {
				c.Logger.WithError(err).Error("Failed to create cluster state change event")
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

	newState := model.ClusterInstallationStateDeletionRequested

	clusterDTO, status, unlockOnce := getClusterForTransition(c, clusterID, newState)
	if status != 0 {
		w.WriteHeader(status)
		return
	}
	defer unlockOnce()

	clusterInstallations, err := c.Store.GetClusterInstallations(&model.ClusterInstallationFilter{
		ClusterID: clusterDTO.ID,
		Paging:    model.AllPagesNotDeleted(),
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
		oldState := clusterDTO.State
		clusterDTO.State = newState

		err := c.Store.UpdateCluster(clusterDTO.Cluster)
		if err != nil {
			c.Logger.WithError(err).Error("failed to mark cluster for deletion")
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		err = c.EventProducer.ProduceClusterStateChangeEvent(clusterDTO.Cluster, oldState)
		if err != nil {
			c.Logger.WithError(err).Error("Failed to create cluster state change event")
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

	if clusterDTO.APISecurityLock {
		logSecurityLockConflict("cluster", c.Logger)
		w.WriteHeader(http.StatusForbidden)
		return
	}

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

// getClusterForTransition locks the cluster and validates if it can be transitioned to desired state.
func getClusterForTransition(c *Context, clusterID, newState string) (*model.ClusterDTO, int, func()) {
	clusterDTO, status, unlockOnce := lockCluster(c, clusterID)
	if status != 0 {
		return nil, status, unlockOnce
	}

	if clusterDTO.APISecurityLock {
		unlockOnce()
		logSecurityLockConflict("cluster", c.Logger)
		return nil, http.StatusForbidden, unlockOnce
	}

	if !clusterDTO.ValidTransitionState(newState) {
		unlockOnce()
		c.Logger.Warnf("unable to transition cluster to %q while in state %q", newState, clusterDTO.State)
		return nil, http.StatusBadRequest, unlockOnce
	}

	return clusterDTO, 0, unlockOnce
}

// supplyDefaultDesiredUtilityVersions fills in the DesiredVersions
// map with the current versions of utilities so that they will be
// reprovisioned without changing the version specified
func supplyDefaultDesiredUtilityVersions(pcr *model.ProvisionClusterRequest, cluster *model.ClusterDTO) *model.ProvisionClusterRequest {
	for utilityName, actualVersion := range cluster.UtilityMetadata.ActualVersions.AsMap() {
		version, found := pcr.DesiredUtilityVersions[utilityName]
		if !found || version == nil {
			pcr.DesiredUtilityVersions[utilityName] = actualVersion
			continue
		}
		if actualVersion == nil {
			continue
		}
		if version.ValuesPath == "" {
			pcr.DesiredUtilityVersions[utilityName].ValuesPath = actualVersion.ValuesPath
		}
		if version.Chart == "" {
			pcr.DesiredUtilityVersions[utilityName].Chart = actualVersion.Chart
		}
	}
	return pcr
}
