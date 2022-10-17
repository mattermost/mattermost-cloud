// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.
//

package supervisor

// Provisioner groups different provisioner interfaces.
type Provisioner interface {
	clusterProvisioner
	installationProvisioner
	clusterInstallationProvisioner
	BackupProvisioner
	restoreOperator
	importProvisioner
	dbMigrationCIProvisioner
}
