// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.
//

package provisioner

import (
	"github.com/mattermost/mattermost-cloud/model"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
)

// TriggerBackup triggers backup job for specific installation on the cluster.
func (provisioner Provisioner) TriggerBackup(backup *model.InstallationBackup, cluster *model.Cluster, installation *model.Installation) (*model.S3DataResidence, error) {
	logger := provisioner.logger.WithFields(log.Fields{
		"cluster":      cluster.ID,
		"installation": installation.ID,
		"backup":       backup.ID,
	})
	logger.Info("Triggering backup for installation")

	k8sClient, err := provisioner.k8sClient(cluster)
	if err != nil {
		return nil, errors.Wrap(err, "failed to create k8s client")
	}

	filestoreCfg, filestoreSecret, err := provisioner.resourceUtil.GetFilestore(installation).
		GenerateFilestoreSpecAndSecret(provisioner.store, logger)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get files store configuration for installation")
	}
	// InstallationBackup is not supported for local MinIO storage, therefore this should not happen
	if filestoreCfg == nil || filestoreSecret == nil {
		return nil, errors.New("filestore secret and config cannot be empty for backup")
	}
	dbSecret, err := provisioner.resourceUtil.GetDatabaseForInstallation(installation).GenerateDatabaseSecret(provisioner.store, logger)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get database configuration")
	}
	// InstallationBackup is not supported for local MySQL, therefore this should not happen
	if dbSecret == nil {
		return nil, errors.New("database secret cannot be empty for backup")
	}

	jobsClient := k8sClient.Clientset.BatchV1().Jobs(installation.ID)

	return provisioner.backupOperator.TriggerBackup(jobsClient, backup, installation, filestoreCfg, dbSecret.Name, logger)
}

// CheckBackupStatus checks status of running backup job,
// returns job start time, when the job finished or -1 if it is still running.
func (provisioner Provisioner) CheckBackupStatus(backup *model.InstallationBackup, cluster *model.Cluster) (int64, error) {
	logger := provisioner.logger.WithFields(log.Fields{
		"cluster":      cluster.ID,
		"installation": backup.InstallationID,
		"backup":       backup.ID,
	})
	logger.Info("Checking backup status for installation")

	k8sClient, err := provisioner.k8sClient(cluster)
	if err != nil {
		return -1, errors.Wrap(err, "failed to create k8s client")
	}

	jobsClient := k8sClient.Clientset.BatchV1().Jobs(backup.InstallationID)

	return provisioner.backupOperator.CheckBackupStatus(jobsClient, backup, logger)
}

// CleanupBackupJob deletes backup job from the cluster if it exists.
func (provisioner Provisioner) CleanupBackupJob(backup *model.InstallationBackup, cluster *model.Cluster) error {
	logger := provisioner.logger.WithFields(log.Fields{
		"cluster":      cluster.ID,
		"installation": backup.InstallationID,
		"backup":       backup.ID,
	})
	logger.Info("Cleaning up backup job for installation")

	k8sClient, err := provisioner.k8sClient(cluster)
	if err != nil {
		return errors.Wrap(err, "failed to create k8s client")
	}

	jobsClient := k8sClient.Clientset.BatchV1().Jobs(backup.InstallationID)

	return provisioner.backupOperator.CleanupBackupJob(jobsClient, backup, logger)
}

// TriggerRestore triggers restoration job for specific installation on the cluster.

func (provisioner Provisioner) TriggerRestore(installation *model.Installation, backup *model.InstallationBackup, cluster *model.Cluster) error {
	logger := provisioner.logger.WithFields(log.Fields{
		"cluster":      cluster.ID,
		"installation": installation.ID,
		"backup":       backup.ID,
	})
	logger.Info("Triggering restoration for installation")

	k8sClient, err := provisioner.k8sClient(cluster)
	if err != nil {
		return errors.Wrap(err, "failed to create k8s client")
	}

	filestoreCfg, filestoreSecret, err := provisioner.resourceUtil.GetFilestore(installation).
		GenerateFilestoreSpecAndSecret(provisioner.store, logger)
	if err != nil {
		return errors.Wrap(err, "failed to get files store configuration for installation")
	}
	// Backup-restore is not supported for local MinIO storage, therefore this should not happen
	if filestoreCfg == nil || filestoreSecret == nil {
		return errors.New("filestore secret and config cannot be empty for database restoration")
	}
	dbSecret, err := provisioner.resourceUtil.GetDatabaseForInstallation(installation).GenerateDatabaseSecret(provisioner.store, logger)
	if err != nil {
		return errors.Wrap(err, "failed to get database configuration")
	}
	// Backup-restore is not supported for local MySQL, therefore this should not happen
	if dbSecret == nil {
		return errors.New("database secret cannot be empty for database restoration")
	}

	jobsClient := k8sClient.Clientset.BatchV1().Jobs(installation.ID)

	return provisioner.backupOperator.TriggerRestore(jobsClient, backup, installation, filestoreCfg, dbSecret.Name, logger)
}

// CheckRestoreStatus checks status of running backup job,
// returns job completion time, when the job finished or -1 if it is still running.
func (provisioner Provisioner) CheckRestoreStatus(backup *model.InstallationBackup, cluster *model.Cluster) (int64, error) {
	logger := provisioner.logger.WithFields(log.Fields{
		"cluster":      cluster.ID,
		"installation": backup.InstallationID,
		"backup":       backup.ID,
	})
	logger.Info("Checking restoration status for installation")

	k8sClient, err := provisioner.k8sClient(cluster)
	if err != nil {
		return -1, errors.Wrap(err, "failed to create k8s client")
	}

	jobsClient := k8sClient.Clientset.BatchV1().Jobs(backup.InstallationID)

	return provisioner.backupOperator.CheckRestoreStatus(jobsClient, backup, logger)
}

// CleanupRestoreJob deletes restore job from the cluster if it exists.
func (provisioner Provisioner) CleanupRestoreJob(backup *model.InstallationBackup, cluster *model.Cluster) error {
	logger := provisioner.logger.WithFields(log.Fields{
		"cluster":      cluster.ID,
		"installation": backup.InstallationID,
		"backup":       backup.ID,
	})
	logger.Info("Cleaning up restoration job for installation")

	k8sClient, err := provisioner.k8sClient(cluster)
	if err != nil {
		return errors.Wrap(err, "failed to create k8s client")
	}

	jobsClient := k8sClient.Clientset.BatchV1().Jobs(backup.InstallationID)

	return provisioner.backupOperator.CleanupRestoreJob(jobsClient, backup, logger)
}
