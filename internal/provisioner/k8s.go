package provisioner

import (
	"github.com/mattermost/mattermost-cloud/k8s"
	"github.com/mattermost/mattermost-cloud/model"
)

type kube interface {
	getKubeClient(cluster *model.Cluster) (*k8s.KubeClient, error)
	getKubeConfigPath(cluster *model.Cluster) (string, error)
}

func (provisioner Provisioner) getKubeOption(provisionerOption string) kube {
	if provisionerOption == "eks" {
		return provisioner.eksProvisioner
	}

	return provisioner.kopsProvisioner
}
