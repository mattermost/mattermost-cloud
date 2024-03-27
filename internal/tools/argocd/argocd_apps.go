package argocd

import (
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
	"gopkg.in/yaml.v3"
)

type Application struct {
	Name            string   `yaml:"name"`
	Namespace       string   `yaml:"namespace"`
	Repo            string   `yaml:"repo"`
	Path            string   `yaml:"path"`
	Revision        string   `yaml:"revision"`
	Helm            Helm     `yaml:"helm"`
	ServerSideApply bool     `yaml:"serverSideApply,omitempty"`
	Replace         bool     `yaml:"replace,omitempty"`
	ClusterLabels   []Labels `yaml:"cluster_labels"`
}

type Helm struct {
	Enabled             bool `yaml:"enabled"`
	AdditionalManifests bool `yaml:"additionalManifests"`
}

type Labels struct {
	ClusterType string `yaml:"cluster-type,omitempty"`
	ClusterID   string `yaml:"cluster-id,omitempty"`
}

// ReadArgoApplicationFile take a argocd application file and load it into Application struct
func ReadArgoApplicationFile(appsFile []byte) (map[string][]Application, error) {
	var apps map[string][]Application

	if err := yaml.Unmarshal(appsFile, &apps); err != nil {
		return nil, errors.Wrap(err, "Error unmarshaling Argo k8s cluster YAML definition")
	}
	return apps, nil
}

func AddClusterIDLabel(data map[string][]Application, appName, clusterID string, logger log.FieldLogger) {
	if appList, ok := data["applications"]; ok {
		for app := range appList {
			if appList[app].Name == appName {
				if labelExists(appList[app].ClusterLabels, clusterID) {
					logger.Debugf("Application %s already has label for cluster %s\n", appName, clusterID)
					return
				}

				appList[app].ClusterLabels = append(appList[app].ClusterLabels, Labels{
					ClusterID: clusterID,
				})
				logger.Debug("Updated applications.yaml file with a new label")
				return
			}
		}
	}
}

func labelExists(labels []Labels, clusterID string) bool {
	for _, label := range labels {
		if label.ClusterID == clusterID {
			return true
		}
	}
	return false
}

func RemoveClusterIDLabel(data map[string][]Application, appName, clusterID string, logger log.FieldLogger) {
	if appList, ok := data["applications"]; ok {
		for app := range appList {
			if appList[app].Name == appName {
				for i, label := range appList[app].ClusterLabels {
					if label.ClusterID == clusterID {
						appList[app].ClusterLabels = append(appList[app].ClusterLabels[:i], appList[app].ClusterLabels[i+1:]...)
						logger.Debug("Updated applications.yaml file with a new label")
						return
					}
				}
			}
		}
	}
}
