// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.
//

package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

const (
	provisionerNamespace    = "provisioner"
	provisionerSubsystemApp = "app"
)

// CloudMetrics holds all of the metrics needed to properly instrument
// the Provisioning server
type CloudMetrics struct {
	InstallationCreationDurationHist    prometheus.Histogram
	InstallationUpdateDurationHist      prometheus.Histogram
	InstallationHibernationDurationHist prometheus.Histogram
	InstallationWakeUpDurationHist      prometheus.Histogram
	InstallationDeletionDurationHist    prometheus.Histogram
}

// New creates a new Prometheus-based Metrics object to be used
// throughout the Provisioner in order to record various performance
// metrics
func New() *CloudMetrics {
	return &CloudMetrics{
		InstallationCreationDurationHist: promauto.NewHistogram(
			prometheus.HistogramOpts{
				Namespace: provisionerNamespace,
				Subsystem: provisionerSubsystemApp,
				Name:      "installation_creation_duration_seconds",
				Help:      "The duration of installation creation tasks",
				Buckets:   standardDurationBuckets(),
			},
		),

		InstallationUpdateDurationHist: promauto.NewHistogram(
			prometheus.HistogramOpts{
				Namespace: provisionerNamespace,
				Subsystem: provisionerSubsystemApp,
				Name:      "installation_update_duration_seconds",
				Help:      "The duration of installation update tasks",
				Buckets:   standardDurationBuckets(),
			},
		),

		InstallationHibernationDurationHist: promauto.NewHistogram(
			prometheus.HistogramOpts{
				Namespace: provisionerNamespace,
				Subsystem: provisionerSubsystemApp,
				Name:      "installation_hibernation_duration_seconds",
				Help:      "The duration of installation hibernation tasks",
				Buckets:   standardDurationBuckets(),
			},
		),

		InstallationWakeUpDurationHist: promauto.NewHistogram(
			prometheus.HistogramOpts{
				Namespace: provisionerNamespace,
				Subsystem: provisionerSubsystemApp,
				Name:      "installation_wakeup_duration_seconds",
				Help:      "The duration of installation wake up tasks",
				Buckets:   standardDurationBuckets(),
			},
		),

		InstallationDeletionDurationHist: promauto.NewHistogram(
			prometheus.HistogramOpts{
				Namespace: provisionerNamespace,
				Subsystem: provisionerSubsystemApp,
				Name:      "installation_deletion_duration_seconds",
				Help:      "The duration of installation deletion tasks",
				Buckets:   standardDurationBuckets(),
			},
		),
	}
}

// 15 second buckets up to 5 minutes.
func standardDurationBuckets() []float64 {
	return prometheus.LinearBuckets(0, 15, 20)
}
