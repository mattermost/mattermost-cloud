// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.
//

//go:build e2e
// +build e2e

package workflow

import (
	"context"
	"os"

	"github.com/mattermost/mattermost-cloud/e2e/pkg"
	"github.com/mattermost/mattermost-cloud/e2e/pkg/eventstest"
	"github.com/mattermost/mattermost-cloud/e2e/tests/state"
	"github.com/mattermost/mattermost-cloud/model"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

// NewClusterSuite creates new Cluster testing suite.
func NewClusterSuite(params ClusterSuiteParams, meta ClusterSuiteMeta, client *model.Client, whChan <-chan *model.WebhookPayload, logger logrus.FieldLogger) *ClusterSuite {
	return &ClusterSuite{
		client: client,
		whChan: whChan,
		logger: logger.WithField("suite", "cluster"),
		Params: params,
		Meta:   meta,
	}
}

// ClusterSuite is testing suite for Clusters.
type ClusterSuite struct {
	client *model.Client
	whChan <-chan *model.WebhookPayload
	logger logrus.FieldLogger

	Params ClusterSuiteParams
	Meta   ClusterSuiteMeta
}

// ClusterSuiteParams contains parameters for ClusterSuite.
type ClusterSuiteParams struct {
	UseExistingCluster   bool
	CreateRequest        model.CreateClusterRequest
	ReprovisionUtilities map[string]*model.HelmUtilityVersion
}

// ClusterSuiteMeta contains metadata for ClusterSuite.
type ClusterSuiteMeta struct {
	ClusterID string
}

// CreateCluster creates new Cluster and waits for it to reach stable state.
func (w *ClusterSuite) CreateCluster(ctx context.Context) error {
	if w.Meta.ClusterID == "" {
		cluster, err := w.client.CreateCluster(&w.Params.CreateRequest)
		if err != nil {
			return errors.Wrap(err, "while creating cluster")
		}
		w.logger.Infof("Cluster creation requested: %s", cluster.ID)
		w.Meta.ClusterID = cluster.ID
		state.ClusterID = cluster.ID
	}

	// Make sure cluster not ready or failed - otherwise we will hang on webhook
	cluster, err := w.client.GetCluster(w.Meta.ClusterID)
	if err != nil {
		return errors.Wrap(err, "failed to get cluster")
	}
	if cluster.State == model.ClusterStateStable {
		return nil
	}
	if cluster.State == model.ClusterStateCreationFailed || cluster.State == model.ClusterStateProvisioningFailed {
		return errors.New("cluster creation failed")
	}

	err = pkg.WaitForClusterToBeStable(ctx, w.Meta.ClusterID, w.whChan, w.logger)
	if err != nil {
		return errors.Wrap(err, "while waiting for cluster creation")
	}

	return nil
}

// ProvisionCluster reprovisions the Cluster and waits for it to reach stable state.
func (w *ClusterSuite) ProvisionCluster(ctx context.Context) error {
	cluster, err := w.client.ProvisionCluster(w.Meta.ClusterID, &model.ProvisionClusterRequest{DesiredUtilityVersions: w.Params.ReprovisionUtilities})
	if err != nil {
		return errors.Wrap(err, "while provisioning cluster")
	}
	w.logger.Infof("Cluster provisioning requested: %s", cluster.ID)
	err = pkg.WaitForClusterToBeStable(ctx, w.Meta.ClusterID, w.whChan, w.logger)
	if err != nil {
		return errors.Wrap(err, "while waiting for cluster provisioning")
	}

	return nil
}

func (w *ClusterSuite) ResizeCluster(ctx context.Context) error {
	if resize, ok := os.LookupEnv("RUN_CLUSTER_RESIZE"); !ok || resize != "true" {
		return nil
	}
	masterInstanceTypeSize := "t3.medium"
	cluster, err := w.client.ResizeCluster(w.Meta.ClusterID, &model.PatchClusterSizeRequest{
		MasterInstanceType: &masterInstanceTypeSize,
	})
	if err != nil {
		return errors.Wrap(err, "while resizing cluster")
	}
	w.logger.Infof("Cluster resize requested: %s", cluster.ID)
	err = pkg.WaitForClusterToBeStable(ctx, w.Meta.ClusterID, w.whChan, w.logger)
	if err != nil {
		return errors.Wrap(err, "while waiting for cluster resize")
	}
	return nil
}

func (w *ClusterSuite) CheckClusterResize(ctx context.Context) error {
	cluster, err := w.client.GetCluster(w.Meta.ClusterID)
	if err != nil {
		return errors.Wrap(err, "while getting cluster")
	}

	// Check cluster resize success based on provisioner type
	switch cluster.Provisioner {
	case model.ProvisionerKops:
		if cluster.ProvisionerMetadataKops.MasterInstanceType != "t3.medium" {
			return errors.Errorf("Expected master instance type to be t3.medium, got %s", cluster.ProvisionerMetadataKops.MasterInstanceType)
		}
	case model.ProvisionerEKS:
		// For EKS, check that cluster is stable (resize operations complete when stable)
		if cluster.State != model.ClusterStateStable {
			return errors.Errorf("Expected cluster to be stable after resize, got state %s", cluster.State)
		}
		w.logger.Info("EKS cluster resize verification: cluster is stable")
	default:
		// For other provisioners, just verify cluster is stable
		if cluster.State != model.ClusterStateStable {
			return errors.Errorf("Expected cluster to be stable after resize, got state %s", cluster.State)
		}
		w.logger.Infof("Cluster resize verification for provisioner %s: cluster is stable", cluster.Provisioner)
	}

	return nil
}

// DeleteCluster deletes the Cluster and waits for its deletion.
func (w *ClusterSuite) DeleteCluster(ctx context.Context) error {
	err := w.client.DeleteCluster(w.Meta.ClusterID)
	if err != nil {
		return errors.Wrap(err, "while delete cluster")
	}
	w.logger.Infof("Cluster deletion requested: %s", w.Meta.ClusterID)
	err = pkg.WaitForClusterDeletion(ctx, w.Meta.ClusterID, w.whChan, w.logger)
	if err != nil {
		return errors.Wrap(err, "while waiting for cluster deletion")
	}

	return nil
}

// Cleanup cleans up installation saved in suite metadata.
func (w *ClusterSuite) Cleanup(ctx context.Context) error {
	cluster, err := w.client.GetCluster(w.Meta.ClusterID)
	if err != nil {
		return errors.Wrap(err, "error getting cluster")
	}
	if cluster == nil {
		return nil
	}
	if cluster.State == model.ClusterStateDeleted {
		w.logger.Info("cluster already deleted")
		return nil
	}
	if cluster.State == model.ClusterStateDeletionRequested ||
		cluster.State == model.ClusterStateDeletionFailed {
		w.logger.Info("cluster already marked for deletion")
		return nil
	}

	err = w.client.DeleteCluster(w.Meta.ClusterID)
	if err != nil {
		return errors.Wrap(err, "while requesting cluster removal")
	}

	err = pkg.WaitForClusterDeletion(context.TODO(), cluster.ID, w.whChan, w.logger)
	if err != nil {
		return errors.Wrap(err, "while waiting for cluster deletion")
	}

	return nil
}

// ClusterCreationEvents returns expected events that should occur while creating the cluster.
// This method should be called only after executing the workflow so that IDs are not empty.
func (w *ClusterSuite) ClusterCreationEvents() []eventstest.EventOccurrence {
	events := []eventstest.EventOccurrence{
		{
			ResourceType: model.TypeCluster.String(),
			ResourceID:   w.Meta.ClusterID,
			OldState:     "n/a",
			NewState:     model.ClusterStateCreationRequested,
		},
	}

	events = append(events, eventstest.EventOccurrence{
		ResourceType: model.TypeCluster.String(),
		ResourceID:   w.Meta.ClusterID,
		OldState:     model.ClusterStateCreationRequested,
		NewState:     model.ClusterStateProvisionInProgress,
	})

	events = append(events, eventstest.EventOccurrence{
		ResourceType: model.TypeCluster.String(),
		ResourceID:   w.Meta.ClusterID,
		OldState:     model.ClusterStateProvisionInProgress,
		NewState:     model.ClusterStateStable,
	})

	return events
}

func (w *ClusterSuite) ClusterResizeEvents() []eventstest.EventOccurrence {
	if resize, ok := os.LookupEnv("RUN_CLUSTER_RESIZE"); !ok || resize != "true" {
		return nil
	}

	events := []eventstest.EventOccurrence{
		{
			ResourceType: model.TypeCluster.String(),
			ResourceID:   w.Meta.ClusterID,
			OldState:     model.ClusterStateStable,
			NewState:     model.ClusterStateResizeRequested,
		},
		{
			ResourceType: model.TypeCluster.String(),
			ResourceID:   w.Meta.ClusterID,
			OldState:     model.ClusterStateResizeRequested,
			NewState:     model.ClusterStateStable,
		},
	}

	return events
}

// ClusterReprovisionEvents returns expected events that should occur while reprovisioning the cluster.
// This method should be called only after executing the workflow so that IDs are not empty.
func (w *ClusterSuite) ClusterReprovisionEvents() []eventstest.EventOccurrence {
	return []eventstest.EventOccurrence{
		{
			ResourceType: model.TypeCluster.String(),
			ResourceID:   w.Meta.ClusterID,
			OldState:     model.ClusterStateStable,
			NewState:     model.ClusterStateProvisioningRequested,
		},
		{
			ResourceType: model.TypeCluster.String(),
			ResourceID:   w.Meta.ClusterID,
			OldState:     model.ClusterStateProvisioningRequested,
			NewState:     model.ClusterStateStable,
		},
	}
}

// ClusterDeletionEvents returns expected events that should occur while deleting the cluster.
// This method should be called only after executing the workflow so that IDs are not empty.
func (w *ClusterSuite) ClusterDeletionEvents() []eventstest.EventOccurrence {
	return []eventstest.EventOccurrence{
		{
			ResourceType: model.TypeCluster.String(),
			ResourceID:   w.Meta.ClusterID,
			OldState:     model.ClusterStateStable,
			NewState:     model.ClusterStateDeletionRequested,
		},
		{
			ResourceType: model.TypeCluster.String(),
			ResourceID:   w.Meta.ClusterID,
			OldState:     model.ClusterStateDeletionRequested,
			NewState:     model.ClusterStateDeleted,
		},
	}
}
