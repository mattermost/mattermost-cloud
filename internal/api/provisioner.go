package api

type ProvisionerOption interface {
	GetProvisioner(string) Provisioner
}

type provisionerOption struct {
	kopsProvisioner Provisioner
	eksProvisioner  Provisioner
}

func (p provisionerOption) GetProvisioner(provisioner string) Provisioner {
	if provisioner == "eks" {
		return p.eksProvisioner
	}
	return p.kopsProvisioner
}

func GetProvisionerOption(eks, kops Provisioner) ProvisionerOption {
	return provisionerOption{
		eksProvisioner:  eks,
		kopsProvisioner: kops,
	}
}
