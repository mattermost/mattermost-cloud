// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.
//

package supervisor

// Provisioner groups different provisioner interfaces.
type Provisioner interface {
	ClusterProvisioner
	InstallationProvisioner
	ClusterInstallationProvisioner
	BackupProvisioner
	RestoreOperator
	ImportProvisioner
	DBMigrationCIProvisioner
}

type ProvisionerOption interface {
	ClusterProvisionerOption
	InstallationProvisionerOption
	ClusterInstallationProvisionerOption
	backupProvisionerOption
	ImportProvisionerOption
	RestoreOperatorOption
	DBMigrationCIProvisionerOption
}

type provisionerOption struct {
	kopsProvisioner Provisioner
	eksProvisioner  Provisioner
}

func GetProvisionerOption(eks, kops Provisioner) ProvisionerOption {
	return provisionerOption{
		eksProvisioner:  eks,
		kopsProvisioner: kops,
	}
}

func (p provisionerOption) getProvisioner(provisioner string) Provisioner {
	if provisioner == "eks" {
		return p.eksProvisioner
	}
	return p.kopsProvisioner
}
