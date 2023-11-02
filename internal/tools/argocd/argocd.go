package argocd

import (
	"os"

	"github.com/pkg/errors"
	"gopkg.in/yaml.v3"
)

type ArgocdClusterLabels struct {
	ClusterTypes string `yaml:"cluster-type"`
	ClusterID    string `yaml:"cluster-id"`
}

type ArgocdClusterRegisterParameters struct {
	Name      string              `yaml:"name"`
	Type      string              `yaml:"type"`
	Labels    ArgocdClusterLabels `yaml:"labels"`
	APIServer string              `yaml:"api_server"`
	CertData  string              `yaml:"certData"`
	CaData    string              `yaml:"caData"`
	KeyData   string              `yaml:"keyData"`
}

type Argock8sRegister struct {
	Clusters []ArgocdClusterRegisterParameters `yaml:"clusters"`
}

// ReadArgoK8sRegistrationFile take a argocd cluster file and load it into Argock8sRegister struct
func (a *Argock8sRegister) ReadArgoK8sRegistrationFile(clusterFile []byte) (*Argock8sRegister, error) {

	err := yaml.Unmarshal(clusterFile, a)
	if err != nil {
		return nil, errors.Wrap(err, "Error unmarshaling Argo k8s cluster YAML definition")
	}

	return a, nil

}

// UpdateK8sClusterRegistrationFile take a argocd cluster file and Add new Cluster spec
func (a *Argock8sRegister) UpdateK8sClusterRegistrationFile(cluster *Argock8sRegister, newCluster ArgocdClusterRegisterParameters, filePath string) error {
	cluster.Clusters = append(cluster.Clusters, newCluster)

	updatedYAML, err := yaml.Marshal(&cluster)
	if err != nil {
		return errors.Wrapf(err, "Error marshalling YAML: %v:", updatedYAML)
	}

	err = os.WriteFile(filePath, updatedYAML, 0644)
	if err != nil {
		return errors.Wrapf(err, "Error writing YAML file: %v", filePath)
	}
	return nil

}
