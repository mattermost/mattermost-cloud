package provisioner

import (
	"github.com/mattermost/mattermost-cloud/internal/supervisor"
)

func (provisioner Provisioner) GetClusterProvisioner(provisionerOption string) supervisor.ClusterProvisioner {
	if provisionerOption == "eks" {
		return provisioner.eksProvisioner
	}

	return provisioner.kopsProvisioner
}
