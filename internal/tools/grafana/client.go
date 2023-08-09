package grafana

import (
	gapi "github.com/grafana/grafana-api-golang-client"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
)

// Client is an interface for interacting with a Grafana instance.
type Client interface {
	LogConfiguration(logger log.FieldLogger)

	AddGrafanaClusterProvisionAnnotation(clusterID string, logger log.FieldLogger)
	UpdateGrafanaClusterProvisionAnnotation(clusterID string, logger log.FieldLogger)

	AddGrafanaClusterUpgradeAnnotation(clusterID string, logger log.FieldLogger)
	UpdateGrafanaClusterUpgradeAnnotation(clusterID string, logger log.FieldLogger)

	AddGrafanaClusterResizeAnnotation(clusterID string, logger log.FieldLogger)
	UpdateGrafanaClusterResizeAnnotation(clusterID string, logger log.FieldLogger)
}

// NewGrafanaClient returns a GrafanaClient interface. If any errors are
// encountered a noop client is returned instead.
func NewGrafanaClient(cloudEnvironmentName, url string, tokens []string) (Client, error) {
	if len(cloudEnvironmentName) == 0 {
		return &noopClient{}, errors.New("cloudEnvironmentName is empty")
	}
	if len(url) == 0 {
		return &noopClient{}, errors.New("grafana URL is empty")
	}
	if len(tokens) == 0 {
		return &noopClient{}, errors.New("no grafana API tokens provided")
	}

	grafana := &Grafana{cloudEnvironmentName: cloudEnvironmentName}
	for i, token := range tokens {
		c, err := gapi.New(url, gapi.Config{APIKey: token})
		if err != nil {
			return &noopClient{}, errors.Wrapf(err, "failed to initiate Grafana client %d", i)
		}
		grafana.clients = append(grafana.clients, c)
	}

	return grafana, nil
}
