// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.
//

package supervisor

import (
	"github.com/mattermost/mattermost-cloud/internal/metrics"
	"github.com/mattermost/mattermost-cloud/k8s"
	"github.com/mattermost/mattermost-cloud/model"
	log "github.com/sirupsen/logrus"
)

// CrossplaneSupervisor finds clusters and syncs their states with the provisioner.
type CrossplaneSupervisor struct {
	store          clusterStore
	k8sClient      k8s.KubeClient
	provisioner    ClusterProvisioner
	eventsProducer eventProducer
	instanceID     string
	metrics        *metrics.CloudMetrics
	logger         log.FieldLogger
}

func NewCrossplaneSupervisor(
	store clusterStore,
	k8sClient k8s.KubeClient,
	provisioner ClusterProvisioner,
	eventProducer eventProducer,
	instanceID string,
	logger log.FieldLogger,
	metrics *metrics.CloudMetrics,
) *CrossplaneSupervisor {
	return &CrossplaneSupervisor{
		store:          store,
		k8sClient:      k8sClient,
		provisioner:    provisioner,
		eventsProducer: eventProducer,
		instanceID:     instanceID,
		metrics:        metrics,
		logger:         logger,
	}
}

// Shutdown performs graceful shutdown tasks for the supervisor.
func (s *CrossplaneSupervisor) Shutdown() {
	s.logger.Debug("Shutting down crossplane supervisor")
}

// Do looks gets the list of clusters and syncs them.
func (s *CrossplaneSupervisor) Do() error {
	return nil
}

// Supervise schedules the required work on the given cluster.
func (s *CrossplaneSupervisor) Supervise(cluster *model.Cluster) {}
