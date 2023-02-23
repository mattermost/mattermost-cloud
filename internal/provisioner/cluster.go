package provisioner

import "github.com/mattermost/mattermost-cloud/internal/supervisor"

func (p Provisioner) GetClusterProvisioner(provisioner string) supervisor.ClusterProvisioner {
	if provisioner == "eks" {
		return p.eksProvisioner
	}

	return p.kopsProvisioner
}
