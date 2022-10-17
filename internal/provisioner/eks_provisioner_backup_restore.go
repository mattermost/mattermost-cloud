// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.
//

package provisioner

import "github.com/mattermost/mattermost-cloud/model"

// TriggerBackup triggers backup.
func (provisioner *EKSProvisioner) TriggerBackup(backupMeta *model.InstallationBackup, cluster *model.Cluster, installation *model.Installation) (*model.S3DataResidence, error) {
	//TODO implement me
	panic("implement me")
}

// CheckBackupStatus checks backup state.
func (provisioner *EKSProvisioner) CheckBackupStatus(backupMeta *model.InstallationBackup, cluster *model.Cluster) (int64, error) {
	//TODO implement me
	panic("implement me")
}

// CleanupBackupJob cleans up backup job.
func (provisioner *EKSProvisioner) CleanupBackupJob(backup *model.InstallationBackup, cluster *model.Cluster) error {
	//TODO implement me
	panic("implement me")
}

// TriggerRestore triggers restore.
func (provisioner *EKSProvisioner) TriggerRestore(installation *model.Installation, backup *model.InstallationBackup, cluster *model.Cluster) error {
	//TODO implement me
	panic("implement me")
}

// CheckRestoreStatus checks restore status.
func (provisioner *EKSProvisioner) CheckRestoreStatus(backupMeta *model.InstallationBackup, cluster *model.Cluster) (int64, error) {
	//TODO implement me
	panic("implement me")
}

// CleanupRestoreJob cleans up restore job.
func (provisioner *EKSProvisioner) CleanupRestoreJob(backup *model.InstallationBackup, cluster *model.Cluster) error {
	//TODO implement me
	panic("implement me")
}
