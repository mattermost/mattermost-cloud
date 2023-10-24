package grafana

import (
	log "github.com/sirupsen/logrus"
)

// noopClient is a Grafana Client that is not configured.
type noopClient struct{}

// LogConfiguration logs client configuration.
func (gc *noopClient) LogConfiguration(logger log.FieldLogger) {
	logger.Info("Grafana client is not configured")
}

func (gc *noopClient) AddGrafanaClusterProvisionAnnotation(clusterID string, logger log.FieldLogger) {
	logger.Debug("Grafana client is not configured; skipping cluster provision annotation creation")
}

func (gc *noopClient) UpdateGrafanaClusterProvisionAnnotation(clusterID string, logger log.FieldLogger) {
	logger.Debug("Grafana client is not configured; skipping cluster provision annotation update")
}

func (gc *noopClient) AddGrafanaClusterUpgradeAnnotation(clusterID string, logger log.FieldLogger) {
	logger.Debug("Grafana client is not configured; skipping cluster provision annotation creation")
}

func (gc *noopClient) UpdateGrafanaClusterUpgradeAnnotation(clusterID string, logger log.FieldLogger) {
	logger.Debug("Grafana client is not configured; skipping cluster provision annotation update")
}

func (gc *noopClient) AddGrafanaClusterResizeAnnotation(clusterID string, logger log.FieldLogger) {
	logger.Debug("Grafana client is not configured; skipping cluster provision annotation creation")
}

func (gc *noopClient) UpdateGrafanaClusterResizeAnnotation(clusterID string, logger log.FieldLogger) {
	logger.Debug("Grafana client is not configured; skipping cluster provision annotation update")
}
