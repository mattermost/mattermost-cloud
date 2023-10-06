// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.
//

package api

import (
	"net/http"

	"github.com/gorilla/mux"
	"github.com/mattermost/mattermost-cloud/internal/store"
	"github.com/mattermost/mattermost-cloud/model"
	"github.com/pkg/errors"
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
	clusterRouter.Handle("/nodegroups", addContext(handleCreateNodegroups)).Methods("POST")
	clusterRouter.Handle("/nodegroup/{nodegroup}", addContext(handleDeleteNodegroup)).Methods("DELETE")
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
//
//	{
//			"provider": "aws",
//			"version": "1.15.0",
//			"kops-ami": "ami-xoxoxo",
//			"size": "SizeAlef1000",
//			"zones": "",
//			"allow-installations": true
//	}
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
		Provisioner:        createClusterRequest.Provisioner,
		AllowInstallations: createClusterRequest.AllowInstallations,
		APISecurityLock:    createClusterRequest.APISecurityLock,
		State:              model.ClusterStateCreationRequested,
	}

	if createClusterRequest.Provisioner == model.ProvisionerEKS {
		cluster.ProvisionerMetadataEKS = &model.EKSMetadata{}
		cluster.ProvisionerMetadataEKS.ApplyClusterCreateRequest(createClusterRequest)
	} else {
		cluster.ProvisionerMetadataKops = &model.KopsMetadata{}
		cluster.ProvisionerMetadataKops.ApplyClusterCreateRequest(createClusterRequest)
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
	c.Logger.Info("I am inside kubernetes handler")
	c.Logger.Info(*upgradeClusterRequest)
	var isUpgradeApplied bool
	if clusterDTO.Provisioner == model.ProvisionerKops {
		isUpgradeApplied = clusterDTO.ProvisionerMetadataKops.ApplyUpgradePatch(upgradeClusterRequest)
	} else if clusterDTO.Provisioner == model.ProvisionerEKS {
		isUpgradeApplied = clusterDTO.ProvisionerMetadataEKS.ApplyUpgradePatch(upgradeClusterRequest)
	}

	if isUpgradeApplied {
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

	if clusterDTO.Provisioner == model.ProvisionerEKS {
		err = clusterDTO.ProvisionerMetadataEKS.ValidateClusterSizePatch(resizeClusterRequest)
		if err != nil {
			c.Logger.WithError(err).Error("failed to validate cluster size patch")
			w.WriteHeader(http.StatusBadRequest)
			return
		}
	} else if clusterDTO.Provisioner == model.ProvisionerKops {
		err = clusterDTO.ProvisionerMetadataKops.ValidateClusterSizePatch(resizeClusterRequest)
		if err != nil {
			c.Logger.WithError(err).Error("failed to validate cluster size patch")
			w.WriteHeader(http.StatusBadRequest)
			return
		}
	}

	oldState := clusterDTO.State

	var isResizeApplied bool
	if clusterDTO.Provisioner == model.ProvisionerKops {
		isResizeApplied = clusterDTO.ProvisionerMetadataKops.ApplyClusterSizePatch(resizeClusterRequest)
	} else if clusterDTO.Provisioner == model.ProvisionerEKS {
		isResizeApplied = clusterDTO.ProvisionerMetadataEKS.ApplyClusterSizePatch(resizeClusterRequest)
	}

	if isResizeApplied {
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

// handleCreateNodegroups responds to POST /api/cluster/{cluster}/nodegroups, creating
// the requested nodegroups.
func handleCreateNodegroups(c *Context, w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	clusterID := vars["cluster"]
	c.Logger = c.Logger.WithField("cluster", clusterID)

	createNodegroupsRequest, err := model.NewCreateNodegroupsRequestFromReader(r.Body)
	if err != nil {
		c.Logger.WithError(err).Error("failed to decode request")
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	newState := model.ClusterStateNodegroupsCreationRequested

	clusterDTO, status, unlockOnce := getClusterForTransition(c, clusterID, newState)
	if status != 0 {
		c.Logger.Debug("Cluster is not in a valid state for nodegroup creation")
		w.WriteHeader(status)
		return
	}
	defer unlockOnce()

	if clusterDTO.Provisioner == model.ProvisionerKops {
		c.Logger.Debug("Creating nodegroups for Kops cluster is not supported")
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	if clusterDTO.Provisioner == model.ProvisionerEKS {
		err = clusterDTO.ProvisionerMetadataEKS.ValidateNodegroupsCreateRequest(createNodegroupsRequest.Nodegroups)
		if err != nil {
			c.Logger.WithError(err).Error("failed to validate nodegroups create request")
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		// Set Nodegroups in change request
		clusterDTO.ProvisionerMetadataEKS.ApplyNodegroupsCreateRequest(createNodegroupsRequest)
	}

	oldState := clusterDTO.State

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
			c.Logger.WithError(err).Error("failed to create cluster state change event")
		}
	}

	unlockOnce()
	_ = c.Supervisor.Do()

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusAccepted)
	outputJSON(c, w, clusterDTO)
}

// handleDeleteNodegroup responds to DELETE /api/cluster/{cluster}/nodegroup/{nodegroup}, deleting
// the requested nodegroup.
func handleDeleteNodegroup(c *Context, w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	clusterID := vars["cluster"]
	nodegroup := vars["nodegroup"]
	c.Logger = c.Logger.WithField("cluster", clusterID)

	newState := model.ClusterStateNodegroupsDeletionRequested

	clusterDTO, status, unlockOnce := getClusterForTransition(c, clusterID, newState)
	if status != 0 {
		c.Logger.Debug("Cluster is not in a valid state for nodegroup deletion")
		w.WriteHeader(status)
		return
	}
	defer unlockOnce()

	if clusterDTO.Provisioner == model.ProvisionerKops {
		c.Logger.Debug("Deleting nodegroup for Kops cluster is not supported")
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	if clusterDTO.Provisioner == model.ProvisionerEKS {
		err := clusterDTO.ProvisionerMetadataEKS.ValidateNodegroupDeleteRequest(nodegroup)
		if err != nil {
			c.Logger.WithError(err).Error("failed to validate nodegroup delete request")
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		// Set Nodegroup in change request
		clusterDTO.ProvisionerMetadataEKS.ApplyNodegroupDeleteRequest(nodegroup)
	}

	oldState := clusterDTO.State

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
			c.Logger.WithError(err).Error("failed to create cluster state change event")
		}
	}

	unlockOnce()
	_ = c.Supervisor.Do()

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

	installations, err := c.Store.GetInstallations(&model.InstallationFilter{
		Paging: model.AllPagesNotDeleted(),
		State:  model.InstallationStateDeletionInProgress,
	}, false, false)
	if err != nil {
		c.Logger.WithError(err).Error("failed to get installations")
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	for _, installation := range installations {
		clusterInstallations, err := c.Store.GetClusterInstallations(&model.ClusterInstallationFilter{
			InstallationID: installation.ID,
			ClusterID:      clusterID,
			Paging:         model.AllPagesWithDeleted(),
		})
		if err != nil {
			c.Logger.WithError(err).Error("failed to get cluster installations")
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		if len(clusterInstallations) != 0 {
			c.Logger.Errorf("unable to delete cluster while it still has at least one installation")
			w.WriteHeader(http.StatusForbidden)
			return
		}
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
