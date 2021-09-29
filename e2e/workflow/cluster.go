// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.
//

//+build e2e

package workflow

import (
	"context"

	"github.com/mattermost/mattermost-cloud/e2e/pkg"
	"github.com/mattermost/mattermost-cloud/model"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

// NewClusterSuite creates new Cluster testing suite.
func NewClusterSuite(params ClusterSuiteParams, env string, client *model.Client, whChan <-chan *model.WebhookPayload, logger logrus.FieldLogger) *ClusterSuite {
	return &ClusterSuite{
		client: client,
		whChan: whChan,
		logger: logger.WithField("suite", "cluster"),
		env:    env,
		Params: params,
		Meta:   ClusterSuiteMeta{},
	}
}

// ClusterSuite is testing suite for Clusters.
type ClusterSuite struct {
	client *model.Client
	whChan <-chan *model.WebhookPayload
	logger logrus.FieldLogger
	env    string

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

	err := pkg.WaitForClusterToBeStable(ctx, w.Meta.ClusterID, w.whChan, w.logger)
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
