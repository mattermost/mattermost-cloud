package grafana

import (
	"net/url"

	gapi "github.com/grafana/grafana-api-golang-client"
	"github.com/mattermost/mattermost-cloud/model"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
)

// Grafana is a wrapper for one or more Grafana clients which can be used to
// easily make API calls to a Grafana backend.
type Grafana struct {
	cloudEnvironmentName string
	clients              []*gapi.Client
}

// LogConfiguration logs client configuration.
func (gc *Grafana) LogConfiguration(logger log.FieldLogger) {
	logger.Infof("Grafana client configured for %s account with %d API token(s)", gc.cloudEnvironmentName, len(gc.clients))
}

// AddGrafanaClusterProvisionAnnotation adds a new cluster provision annotation.
func (gc *Grafana) AddGrafanaClusterProvisionAnnotation(clusterID string, logger log.FieldLogger) {
	gc.addAllAnnoationsAndLogErrors(
		"Cluster Provisioning",
		[]string{newKVTag("cluster", clusterID), clusterProvisionTag},
		logger,
	)
}

// UpdateGrafanaClusterProvisionAnnotation updates an existing cluster provision
// annotation.
func (gc *Grafana) UpdateGrafanaClusterProvisionAnnotation(clusterID string, logger log.FieldLogger) {
	gc.updateAllAnnotationsAndLogErrors(
		[]string{newKVTag("cluster", clusterID), clusterProvisionTag},
		logger,
	)
}

// AddGrafanaClusterUpgradeAnnotation adds a new cluster upgrade annotation.
func (gc *Grafana) AddGrafanaClusterUpgradeAnnotation(clusterID string, logger log.FieldLogger) {
	gc.addAllAnnoationsAndLogErrors(
		"Cluster Upgrade",
		[]string{newKVTag("cluster", clusterID), clusterUpgradeTag},
		logger,
	)
}

// UpdateGrafanaClusterUpgradeAnnotation updates an existing cluster upgrade
// annotation.
func (gc *Grafana) UpdateGrafanaClusterUpgradeAnnotation(clusterID string, logger log.FieldLogger) {
	gc.updateAllAnnotationsAndLogErrors(
		[]string{newKVTag("cluster", clusterID), clusterUpgradeTag},
		logger,
	)
}

// AddGrafanaClusterResizeAnnotation adds a new cluster resize annotation.
func (gc *Grafana) AddGrafanaClusterResizeAnnotation(clusterID string, logger log.FieldLogger) {
	gc.addAllAnnoationsAndLogErrors(
		"Cluster Resize",
		[]string{newKVTag("cluster", clusterID), clusterResizeTag},
		logger,
	)
}

// UpdateGrafanaClusterResizeAnnotation updates an existing cluster resize
// annotation.
func (gc *Grafana) UpdateGrafanaClusterResizeAnnotation(clusterID string, logger log.FieldLogger) {
	gc.updateAllAnnotationsAndLogErrors(
		[]string{newKVTag("cluster", clusterID), clusterResizeTag},
		logger,
	)
}

// addAllAnnoationsAndLogErrors attempts to create a given annotation for all
// configured grafana clients and logs any errors encountered.
func (gc *Grafana) addAllAnnoationsAndLogErrors(text string, extraTags []string, logger log.FieldLogger) {
	for i, client := range gc.clients {
		err := gc.addGrafanaAnnotation(client, text, extraTags, logger)
		if err != nil {
			logger.WithError(err).Errorf("Failed to create annotation for grafana token %d of %d", i+1, len(gc.clients))
		}
	}
}

// addGrafanaAnnotation adds an annotation via the Grafana API.
func (gc *Grafana) addGrafanaAnnotation(client *gapi.Client, text string, extraTags []string, logger log.FieldLogger) error {
	tags := append(extraTags, newKVTag("environment", gc.cloudEnvironmentName), provisionerTag)
	id, err := client.NewAnnotation(&gapi.Annotation{
		Text: text,
		Tags: tags,
		Time: model.GetMillis(),
	})
	if err != nil {
		return errors.Wrap(err, "failed to create Grafana annotation")
	}
	logger.Debugf("Annotation created successfully with ID %d", id)

	return nil
}

// updateAllAnnotationsAndLogErrors attempts to update a given annotation for all
// configured grafana clients and logs any errors encountered.
func (gc *Grafana) updateAllAnnotationsAndLogErrors(tagFilters []string, logger log.FieldLogger) {
	for i, client := range gc.clients {
		err := updateGrafanaAnnotation(client, tagFilters, logger)
		if err != nil {
			logger.WithError(err).Errorf("Failed to create annotation for grafana token %d of %d", i+1, len(gc.clients))
		}
	}
}

// updateGrafanaAnnotation updates the end time of an existing Grafana annotation.
func updateGrafanaAnnotation(client *gapi.Client, tagFilters []string, logger log.FieldLogger) error {
	values := url.Values{}
	for _, filter := range tagFilters {
		values.Add("tags", filter)
	}
	annotations, err := client.Annotations(values)
	if err != nil {
		return errors.Wrap(err, "failed to query grafana annotations")
	}
	if len(annotations) == 0 {
		return errors.New("failed to find grafana annotation to update")
	}

	annotation := annotations[0]
	annotation.TimeEnd = model.GetMillis()

	logger.Debugf("Updating grafana annotation %d end time", annotation.ID)
	_, err = client.UpdateAnnotation(annotation.ID, &annotation)
	if err != nil {
		return errors.Wrapf(err, "failed to update grafana annotation %d", annotation.ID)
	}

	return nil
}
