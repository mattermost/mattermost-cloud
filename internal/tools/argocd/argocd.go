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
	Name          string              `yaml:"name"`
	Type          string              `yaml:"type"`
	Labels        ArgocdClusterLabels `yaml:"labels"`
	APIServer     string              `yaml:"api_server"`
	ArgoCDRoleARN string              `yaml:"argoCDRoleARN,omitempty"`
	ClusterName   string              `yaml:"clusterName,omitempty"`
	CertData      string              `yaml:"certData,omitempty"`
	CaData        string              `yaml:"caData,omitempty"`
	KeyData       string              `yaml:"keyData,omitempty"`
}

type Argock8sRegister struct {
	Clusters []ArgocdClusterRegisterParameters `yaml:"clusters"`
}

// ReadArgoK8sRegistrationFile take a argocd cluster file and load it into Argock8sRegister struct
func ReadArgoK8sRegistrationFile(clusterFile []byte) (*Argock8sRegister, error) {
	a := &Argock8sRegister{}

	if err := yaml.Unmarshal(clusterFile, a); err != nil {
		return nil, errors.Wrap(err, "Error unmarshaling Argo k8s cluster YAML definition")
	}
	return a, nil
}

// UpdateK8sClusterRegistrationFile take a argocd cluster file and Add new Cluster spec
func UpdateK8sClusterRegistrationFile(cluster *Argock8sRegister, newCluster ArgocdClusterRegisterParameters, filePath string) error {
	index := -1
	for i, clusterValue := range cluster.Clusters {
		if clusterValue.Name == newCluster.Name {
			index = i
			break
		}
	}

	if index != -1 {
		cluster.Clusters[index] = newCluster
	} else {
		cluster.Clusters = append(cluster.Clusters, newCluster)
	}

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

// DeleteK8sClusterFromRegistrationFile take a argocd cluster file and delete Cluster from spec
func DeleteK8sClusterFromRegistrationFile(cluster *Argock8sRegister, clusterName string, filePath string) error {
	for k, v := range cluster.Clusters {
		if v.Name == clusterName {
			cluster.Clusters = append(cluster.Clusters[:k], cluster.Clusters[k+1:]...)
		}
	}

	updatedYAML, err := yaml.Marshal(&cluster)
	if err != nil {
		return errors.Wrapf(err, "Error marshalling YAML: %v:", updatedYAML)
	}

	if err = os.WriteFile(filePath, updatedYAML, 0644); err != nil {
		return errors.Wrapf(err, "Error writing YAML file: %v", filePath)
	}
	return nil
}
