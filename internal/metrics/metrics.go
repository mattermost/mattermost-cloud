// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.
//

package metrics

import (
	"strconv"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

const (
	provisionerNamespace    = "provisioner"
	provisionerSubsystemAPI = "api"
	provisionerSubsystemApp = "app"
)

// CloudMetrics holds all of the metrics needed to properly instrument
// the Provisioning server
type CloudMetrics struct {
	// API
	APIRequestsCounter prometheus.Counter
	APITimesHistograms *prometheus.HistogramVec

	// Installation
	InstallationCreationDurationHist    *prometheus.HistogramVec
	InstallationUpdateDurationHist      *prometheus.HistogramVec
	InstallationHibernationDurationHist *prometheus.HistogramVec
	InstallationWakeUpDurationHist      *prometheus.HistogramVec
	InstallationDeletionDurationHist    *prometheus.HistogramVec

	// ClusterInstallation
	ClusterInstallationReconcilingDurationHist *prometheus.HistogramVec
	ClusterInstallationDeletionDurationHist    *prometheus.HistogramVec

	// Cluster
	ClusterCreationDurationHist           *prometheus.HistogramVec
	ClusterUpgradeDurationHist            *prometheus.HistogramVec
	ClusterProvisioningDurationHist       *prometheus.HistogramVec
	ClusterResizeDurationHist             *prometheus.HistogramVec
	ClusterDeletionDurationHist           *prometheus.HistogramVec
	ClusterNodegroupsCreationDurationHist *prometheus.HistogramVec
	ClusterNodegroupsDeletionDurationHist *prometheus.HistogramVec
}

// New creates a new Prometheus-based Metrics object to be used
// throughout the Provisioner in order to record various performance
// metrics
func New() *CloudMetrics {
	return &CloudMetrics{
		APIRequestsCounter: promauto.NewCounter(prometheus.CounterOpts{
			Namespace: provisionerNamespace,
			Subsystem: provisionerSubsystemAPI,
			Name:      "requests_total",
			Help:      "The total number of http API requests",
		}),

		APITimesHistograms: promauto.NewHistogramVec(
			prometheus.HistogramOpts{
				Namespace: provisionerNamespace,
				Subsystem: provisionerSubsystemAPI,
				Name:      "requests_duration",
				Help:      "The duration of http API requests",
			},
			[]string{"handler", "method", "status_code"},
		),

		InstallationCreationDurationHist: promauto.NewHistogramVec(
			prometheus.HistogramOpts{
				Namespace: provisionerNamespace,
				Subsystem: provisionerSubsystemApp,
				Name:      "installation_creation_duration_seconds",
				Help:      "The duration of installation creation tasks",
				Buckets:   standardDurationBuckets(),
			},
			[]string{"group"},
		),

		InstallationUpdateDurationHist: promauto.NewHistogramVec(
			prometheus.HistogramOpts{
				Namespace: provisionerNamespace,
				Subsystem: provisionerSubsystemApp,
				Name:      "installation_update_duration_seconds",
				Help:      "The duration of installation update tasks",
				Buckets:   standardDurationBuckets(),
			},
			[]string{"group"},
		),

		InstallationHibernationDurationHist: promauto.NewHistogramVec(
			prometheus.HistogramOpts{
				Namespace: provisionerNamespace,
				Subsystem: provisionerSubsystemApp,
				Name:      "installation_hibernation_duration_seconds",
				Help:      "The duration of installation hibernation tasks",
				Buckets:   standardDurationBuckets(),
			},
			[]string{"group"},
		),

		InstallationWakeUpDurationHist: promauto.NewHistogramVec(
			prometheus.HistogramOpts{
				Namespace: provisionerNamespace,
				Subsystem: provisionerSubsystemApp,
				Name:      "installation_wakeup_duration_seconds",
				Help:      "The duration of installation wake up tasks",
				Buckets:   standardDurationBuckets(),
			},
			[]string{"group"},
		),

		InstallationDeletionDurationHist: promauto.NewHistogramVec(
			prometheus.HistogramOpts{
				Namespace: provisionerNamespace,
				Subsystem: provisionerSubsystemApp,
				Name:      "installation_deletion_duration_seconds",
				Help:      "The duration of installation deletion tasks",
				Buckets:   standardDurationBuckets(),
			},
			[]string{"group"},
		),

		ClusterInstallationReconcilingDurationHist: promauto.NewHistogramVec(
			prometheus.HistogramOpts{
				Namespace: provisionerNamespace,
				Subsystem: provisionerSubsystemApp,
				Name:      "cluster_installation_reconciling_duration_seconds",
				Help:      "The duration of cluster installation reconciliation tasks",
				Buckets:   standardDurationBuckets(),
			},
			[]string{"cluster"},
		),

		ClusterInstallationDeletionDurationHist: promauto.NewHistogramVec(
			prometheus.HistogramOpts{
				Namespace: provisionerNamespace,
				Subsystem: provisionerSubsystemApp,
				Name:      "cluster_installation_deletion_duration_seconds",
				Help:      "The duration of cluster installation deletion tasks",
				Buckets:   standardDurationBuckets(),
			},
			[]string{"cluster"},
		),
		ClusterCreationDurationHist: promauto.NewHistogramVec(
			prometheus.HistogramOpts{
				Namespace: provisionerNamespace,
				Subsystem: provisionerSubsystemApp,
				Name:      "cluster_creation_duration_seconds",
				Help:      "The duration of cluster creation tasks",
				Buckets:   standardDurationBuckets(),
			},
			[]string{},
		),
		ClusterUpgradeDurationHist: promauto.NewHistogramVec(
			prometheus.HistogramOpts{
				Namespace: provisionerNamespace,
				Subsystem: provisionerSubsystemApp,
				Name:      "cluster_upgrade_duration_seconds",
				Help:      "The duration of cluster upgrade tasks",
				Buckets:   standardDurationBuckets(),
			},
			[]string{},
		),
		ClusterProvisioningDurationHist: promauto.NewHistogramVec(
			prometheus.HistogramOpts{
				Namespace: provisionerNamespace,
				Subsystem: provisionerSubsystemApp,
				Name:      "cluster_provisioning_duration_seconds",
				Help:      "The duration of cluster provisioning tasks",
				Buckets:   standardDurationBuckets(),
			},
			[]string{},
		),
		ClusterResizeDurationHist: promauto.NewHistogramVec(
			prometheus.HistogramOpts{
				Namespace: provisionerNamespace,
				Subsystem: provisionerSubsystemApp,
				Name:      "cluster_resize_duration_seconds",
				Help:      "The duration of cluster resize tasks",
				Buckets:   standardDurationBuckets(),
			},
			[]string{},
		),
		ClusterDeletionDurationHist: promauto.NewHistogramVec(
			prometheus.HistogramOpts{
				Namespace: provisionerNamespace,
				Subsystem: provisionerSubsystemApp,
				Name:      "cluster_deletion_duration_seconds",
				Help:      "The duration of cluster deletion tasks",
				Buckets:   standardDurationBuckets(),
			},
			[]string{},
		),
		ClusterNodegroupsCreationDurationHist: promauto.NewHistogramVec(
			prometheus.HistogramOpts{
				Namespace: provisionerNamespace,
				Subsystem: provisionerSubsystemApp,
				Name:      "cluster_nodegroups_creation_duration_seconds",
				Help:      "The duration of cluster nodegroups creation tasks",
				Buckets:   standardDurationBuckets(),
			},
			[]string{},
		),
		ClusterNodegroupsDeletionDurationHist: promauto.NewHistogramVec(
			prometheus.HistogramOpts{
				Namespace: provisionerNamespace,
				Subsystem: provisionerSubsystemApp,
				Name:      "cluster_nodegroups_deletion_duration_seconds",
				Help:      "The duration of cluster nodegroups deletion tasks",
				Buckets:   standardDurationBuckets(),
			},
			[]string{},
		),
	}
}

// IncrementAPIRequest increases APIRequestsCounter by one.
func (cm *CloudMetrics) IncrementAPIRequest() {
	cm.APIRequestsCounter.Inc()
}

// ObserveAPIEndpointDuration observes the duration of an API request.
func (cm *CloudMetrics) ObserveAPIEndpointDuration(handler, method string, statusCode int, elapsed float64) {
	cm.APITimesHistograms.With(prometheus.Labels{"handler": handler, "method": method, "status_code": strconv.Itoa(statusCode)}).Observe(elapsed)
}

// 15 second buckets up to 5 minutes.
func standardDurationBuckets() []float64 {
	return prometheus.LinearBuckets(0, 15, 20)
}
