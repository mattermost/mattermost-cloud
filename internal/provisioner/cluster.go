package provisioner

import (
	"github.com/mattermost/mattermost-cloud/internal/supervisor"
	"github.com/mattermost/mattermost-cloud/k8s"
	"github.com/mattermost/mattermost-cloud/model"
)

func (provisioner Provisioner) GetClusterProvisioner(provisionerOption string) supervisor.ClusterProvisioner {
	if provisionerOption == "eks" {
		return provisioner.eksProvisioner
	}

	return provisioner.kopsProvisioner
}

func (provisioner Provisioner) k8sClient(cluster *model.Cluster) (*k8s.KubeClient, error) {
	return provisioner.getKubeOption(cluster.Provisioner).getKubeClient(cluster)
}

func (provisioner Provisioner) getClusterKubecfg(cluster *model.Cluster) (string, error) {
	return provisioner.getKubeOption(cluster.Provisioner).getKubeConfigPath(cluster)
}
