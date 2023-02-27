package provisioner

import (
	"github.com/mattermost/mattermost-cloud/internal/supervisor"
	"github.com/mattermost/mattermost-cloud/model"
)

func (provisioner Provisioner) GetClusterProvisioner(provisionerOption string) supervisor.ClusterProvisioner {
	if provisionerOption == model.ProvisionerEKS {
		return provisioner.eksProvisioner
	}

	return provisioner.kopsProvisioner
}
