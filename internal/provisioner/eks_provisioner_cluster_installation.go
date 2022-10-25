// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.
//

package provisioner

import (
	"github.com/mattermost/mattermost-cloud/internal/tools/aws"
	"github.com/mattermost/mattermost-cloud/model"
	mmv1beta1 "github.com/mattermost/mattermost-operator/apis/mattermost/v1beta1"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
)

// CreateClusterInstallation creates ClusterInstallation.
func (provisioner *EKSProvisioner) CreateClusterInstallation(cluster *model.Cluster, installation *model.Installation, installationDNS []*model.InstallationDNS, clusterInstallation *model.ClusterInstallation) error {
	logger := provisioner.logger.WithFields(log.Fields{
		"cluster":      clusterInstallation.ClusterID,
		"installation": clusterInstallation.InstallationID,
		"version":      "v1beta1",
	})
	logger.Info("Creating cluster installation")

	kubeconfigPath, err := provisioner.prepareClusterKubeconfig(cluster.ID)
	if err != nil {
		return errors.Wrap(err, "failed to prepare kubeconfig")
	}

	return provisioner.commonProvisioner.createClusterInstallation(clusterInstallation, installation, installationDNS, kubeconfigPath, logger)
}

// EnsureCRMigrated ensures CR is in correct version.
func (provisioner *EKSProvisioner) EnsureCRMigrated(cluster *model.Cluster, clusterInstallation *model.ClusterInstallation) (bool, error) {
	return true, nil
}

// HibernateClusterInstallation hibernates the cluster installation.
func (provisioner *EKSProvisioner) HibernateClusterInstallation(cluster *model.Cluster, installation *model.Installation, clusterInstallation *model.ClusterInstallation) error {
	logger := provisioner.logger.WithFields(log.Fields{
		"cluster":      clusterInstallation.ClusterID,
		"installation": clusterInstallation.InstallationID,
	})

	kubeconfigPath, err := provisioner.prepareClusterKubeconfig(cluster.ID)
	if err != nil {
		return errors.Wrap(err, "failed to prepare kubeconfig")
	}

	return hibernateInstallation(kubeconfigPath, logger, clusterInstallation, installation)
}

// UpdateClusterInstallation updates ClusterInstsallation.
func (provisioner *EKSProvisioner) UpdateClusterInstallation(cluster *model.Cluster, installation *model.Installation, installationDNS []*model.InstallationDNS, clusterInstallation *model.ClusterInstallation) error {
	logger := provisioner.logger.WithFields(log.Fields{
		"cluster":      clusterInstallation.ClusterID,
		"installation": clusterInstallation.InstallationID,
	})

	kubeconfigPath, err := provisioner.prepareClusterKubeconfig(cluster.ID)
	if err != nil {
		return errors.Wrap(err, "failed to prepare kubeconfig")
	}

	return provisioner.commonProvisioner.updateClusterInstallation(kubeconfigPath, installation, installationDNS, clusterInstallation, logger)
}

// VerifyClusterInstallationMatchesConfig verifies ClusterInstallation matches config - not implemented.
func (provisioner *EKSProvisioner) VerifyClusterInstallationMatchesConfig(cluster *model.Cluster, installation *model.Installation, clusterInstallation *model.ClusterInstallation) (bool, error) {
	//TODO implement me
	panic("implement me")
}

// DeleteOldClusterInstallationLicenseSecrets deletes old installation license secret.
func (provisioner *EKSProvisioner) DeleteOldClusterInstallationLicenseSecrets(cluster *model.Cluster, installation *model.Installation, clusterInstallation *model.ClusterInstallation) error {
	logger := provisioner.logger.WithFields(log.Fields{
		"cluster":      clusterInstallation.ClusterID,
		"installation": clusterInstallation.InstallationID,
	})

	configLocation, err := provisioner.prepareClusterKubeconfig(cluster.ID)
	if err != nil {
		return errors.Wrap(err, "failed to prepare kubeconfig")
	}

	return deleteOldClusterInstallationLicenseSecrets(configLocation, installation, clusterInstallation, logger)
}

// DeleteClusterInstallation deletes ClusterInstallation.
func (provisioner *EKSProvisioner) DeleteClusterInstallation(cluster *model.Cluster, installation *model.Installation, clusterInstallation *model.ClusterInstallation) error {
	logger := provisioner.logger.WithFields(log.Fields{
		"cluster":      clusterInstallation.ClusterID,
		"installation": clusterInstallation.InstallationID,
	})

	configLocation, err := provisioner.prepareClusterKubeconfig(cluster.ID)
	if err != nil {
		return errors.Wrap(err, "failed to prepare kubeconfig")
	}

	return deleteClusterInstallation(installation, clusterInstallation, configLocation, logger)
}

// IsResourceReadyAndStable determines if the resource is ready and stable.
func (provisioner *EKSProvisioner) IsResourceReadyAndStable(cluster *model.Cluster, clusterInstallation *model.ClusterInstallation) (bool, bool, error) {
	logger := provisioner.logger.WithFields(log.Fields{
		"cluster":      clusterInstallation.ClusterID,
		"installation": clusterInstallation.InstallationID,
	})

	cr, err := provisioner.getMattermostCustomResource(cluster, clusterInstallation, logger)
	if err != nil {
		return false, false, errors.Wrap(err, "failed to get ClusterInstallation Custom Resource")
	}
	return isMattermostReady(cr)
}

// RefreshSecrets refreshes Installation secrets.
func (provisioner *EKSProvisioner) RefreshSecrets(cluster *model.Cluster, installation *model.Installation, clusterInstallation *model.ClusterInstallation) error {
	logger := provisioner.logger.WithFields(log.Fields{
		"cluster":      clusterInstallation.ClusterID,
		"installation": clusterInstallation.InstallationID,
	})
	logger.Info("Refreshing secrets for cluster installation")

	configLocation, err := provisioner.prepareClusterKubeconfig(cluster.ID)
	if err != nil {
		return errors.Wrap(err, "failed to prepare kubeconfig")
	}

	return provisioner.commonProvisioner.refreshSecrets(installation, clusterInstallation, configLocation)
}

// PrepareClusterUtilities prepares cluster utilities.
func (provisioner *EKSProvisioner) PrepareClusterUtilities(cluster *model.Cluster, installation *model.Installation, store model.ClusterUtilityDatabaseStoreInterface, awsClient aws.AWS) error {
	logger := provisioner.logger.WithField("cluster", cluster.ID)
	logger.Info("Preparing cluster utilities")

	// TODO: move this logic to a database interface method.
	if installation.Database != model.InstallationDatabaseMultiTenantRDSPostgresPGBouncer {
		return nil
	}

	configLocation, err := provisioner.prepareClusterKubeconfig(cluster.ID)
	if err != nil {
		return errors.Wrap(err, "failed to prepare kubeconfig")
	}

	return prepareClusterUtilities(cluster, configLocation, store, awsClient, provisioner.params.PGBouncerConfig, logger)
}

// ExecClusterInstallationCLI executes command on ClusterInstallation.
func (provisioner *EKSProvisioner) ExecClusterInstallationCLI(cluster *model.Cluster, clusterInstallation *model.ClusterInstallation, args ...string) ([]byte, error, error) {
	//TODO implement me
	panic("implement me")
}

// ExecMMCTL executes mmctl command on ClusterInstallation.
func (provisioner *EKSProvisioner) ExecMMCTL(cluster *model.Cluster, clusterInstallation *model.ClusterInstallation, args ...string) ([]byte, error) {
	//TODO implement me
	panic("implement me")
}

// ExecMattermostCLI executes Mattermost cli on ClusterInstallation.
func (provisioner *EKSProvisioner) ExecMattermostCLI(cluster *model.Cluster, clusterInstallation *model.ClusterInstallation, args ...string) ([]byte, error) {
	//TODO implement me
	panic("implement me")
}

// ExecClusterInstallationJob executes ClusterInstallation command as a job.
func (provisioner *EKSProvisioner) ExecClusterInstallationJob(cluster *model.Cluster, clusterInstallation *model.ClusterInstallation, args ...string) error {
	//TODO implement me
	panic("implement me")
}

// getMattermostCustomResource gets the cluster installation resource from
// the kubernetes API.
func (provisioner *EKSProvisioner) getMattermostCustomResource(cluster *model.Cluster, clusterInstallation *model.ClusterInstallation, logger log.FieldLogger) (*mmv1beta1.Mattermost, error) {
	configLocation, err := provisioner.prepareClusterKubeconfig(cluster.ID)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get kops config from cache")
	}

	return getMattermostCustomResource(clusterInstallation, configLocation, logger)
}
