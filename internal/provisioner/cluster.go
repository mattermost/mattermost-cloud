package provisioner

func (p provisioner) GetClusterProvisioner(provisioner string) ClusterProvisioner {
	if provisioner == "eks" {
		return p.eksProvisioner
	}

	return p.kopsProvisioner
}
