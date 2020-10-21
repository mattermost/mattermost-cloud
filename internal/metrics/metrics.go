package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

// CloudMetrics holds all of the metrics needed to properly instrument
// the Provisioning server
type CloudMetrics struct {
	InstallationCreationDurationHist        prometheus.Histogram
	ClusterInstallationCreationDurationHist prometheus.Histogram
}

// New creates a new Prometheus-based Metrics object to be used
// throughout the Provisioner in order to record various performance
// metrics
func New() *CloudMetrics {
	return &CloudMetrics{
		InstallationCreationDurationHist: promauto.NewHistogram(
			prometheus.HistogramOpts{
				Name:    "mm_cloud_create_installation_duration_seconds",
				Help:    "The duration of Installation creation tasks",
				Buckets: prometheus.LinearBuckets(0, 30, 20),
			}),
		ClusterInstallationCreationDurationHist: promauto.NewHistogram(
			prometheus.HistogramOpts{
				Name:    "mm_cloud_create_cluster_installation_duration_seconds",
				Help:    "The duration of Cluster Installation creation tasks",
				Buckets: prometheus.LinearBuckets(0, 30, 20),
			}),
	}
}
