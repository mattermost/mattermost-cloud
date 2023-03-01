// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.
//

//go:build e2e
// +build e2e

package workflow

import (
	"context"

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
	state.ClusterID = cluster.ID

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

	if w.Params.CreateRequest.Provisioner == "eks" {
		events = append(events, eventstest.EventOccurrence{
			ResourceType: model.TypeCluster.String(),
			ResourceID:   w.Meta.ClusterID,
			OldState:     model.ClusterStateCreationRequested,
			NewState:     model.ClusterStateCreationInProgress,
		})

		events = append(events, eventstest.EventOccurrence{
			ResourceType: model.TypeCluster.String(),
			ResourceID:   w.Meta.ClusterID,
			OldState:     model.ClusterStateCreationInProgress,
			NewState:     model.ClusterStateWaitingForNodes,
		})

		events = append(events, eventstest.EventOccurrence{
			ResourceType: model.TypeCluster.String(),
			ResourceID:   w.Meta.ClusterID,
			OldState:     model.ClusterStateWaitingForNodes,
			NewState:     model.ClusterStateProvisionInProgress,
		})
	} else {
		events = append(events, eventstest.EventOccurrence{
			ResourceType: model.TypeCluster.String(),
			ResourceID:   w.Meta.ClusterID,
			OldState:     model.ClusterStateCreationRequested,
			NewState:     model.ClusterStateProvisionInProgress,
		})
	}

	events = append(events, eventstest.EventOccurrence{
		ResourceType: model.TypeCluster.String(),
		ResourceID:   w.Meta.ClusterID,
		OldState:     model.ClusterStateProvisionInProgress,
		NewState:     model.ClusterStateStable,
	})

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
