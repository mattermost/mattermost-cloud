// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.
//

package provisioner

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/mattermost/mattermost-cloud/internal/provisioner/pgbouncer"
	"github.com/mattermost/mattermost-cloud/internal/provisioner/utility"
	"github.com/mattermost/mattermost-cloud/internal/store"
	"github.com/mattermost/mattermost-cloud/internal/supervisor"
	"github.com/mattermost/mattermost-cloud/internal/tools/aws"
	"github.com/mattermost/mattermost-cloud/k8s"
	"github.com/mattermost/mattermost-cloud/model"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// ExternalProvisionerType is provisioner type for external clusters.
const ExternalProvisionerType = "external"

// EKSProvisioner provisions clusters using AWS EKS.
type ExternalProvisioner struct {
	params             ProvisioningParams
	awsClient          aws.AWS
	clusterUpdateStore clusterUpdateStore
	logger             log.FieldLogger
}

var _ supervisor.ClusterProvisioner = (*ExternalProvisioner)(nil)

// NewExternalProvisioner creates new ExternalProvisioner.
func NewExternalProvisioner(
	params ProvisioningParams,
	awsClient aws.AWS,
	store *store.SQLStore,
	logger log.FieldLogger,
) *ExternalProvisioner {
	return &ExternalProvisioner{
		params:             params,
		awsClient:          awsClient,
		clusterUpdateStore: store,
		logger:             logger,
	}
}

// PrepareCluster prepares metadata for ExternalProvisioner.
func (provisioner *ExternalProvisioner) PrepareCluster(cluster *model.Cluster) bool {
	// Don't regenerate the name if already set.
	if cluster.ProvisionerMetadataExternal.Name != "" {
		return false
	}

	// Generate the external name using the cluster ID.
	cluster.ProvisionerMetadataExternal.Name = fmt.Sprintf("%s-external-k8s", cluster.ID)

	return true
}

// CreateCluster manages VPC claiming for external clusters.
func (provisioner *ExternalProvisioner) CreateCluster(cluster *model.Cluster) error {
	logger := provisioner.logger.WithField("cluster", cluster.ID)
	logger.Info("Performing cluster creation tasks")

	externalMetadata := cluster.ProvisionerMetadataExternal
	if externalMetadata == nil {
		return errors.New("external metadata not set when creating an external cluster")
	}

	if cluster.ProviderMetadataExternal.HasAWSInfrastructure {
		logger.Debugf("Claiming VPC %s for external cluster", externalMetadata.VPC)
		_, err := provisioner.awsClient.ClaimVPC(externalMetadata.VPC, cluster, provisioner.params.Owner, logger)
		if err != nil {
			return errors.Wrap(err, "couldn't claim VPC")
		}
	} else {
		logger.Debug("External cluster has no VPC ID; skipping VPC claim process...")
	}

	return nil
}

// CheckClusterCreated is no-op for external clusters.
func (provisioner *ExternalProvisioner) CheckClusterCreated(cluster *model.Cluster) (bool, error) {
	return true, nil
}

// CreateNodegroups is no-op for external clusters.
func (provisioner *ExternalProvisioner) CreateNodegroups(cluster *model.Cluster) error {
	return nil
}

// CheckNodegroupsCreated is no-op for external clusters.
func (provisioner *ExternalProvisioner) CheckNodegroupsCreated(cluster *model.Cluster) (bool, error) {
	return true, nil
}

// DeleteNodegroups is no-op for external clusters.
func (provisioner *ExternalProvisioner) DeleteNodegroups(cluster *model.Cluster) error {
	return nil
}

// ProvisionCluster runs provisioning tasks for external clusters.
func (provisioner *ExternalProvisioner) ProvisionCluster(cluster *model.Cluster) error {
	logger := provisioner.logger.WithField("cluster", cluster.ID)

	logger.Info("Provisioning cluster")

	k8sClient, err := provisioner.getKubeClient(cluster)
	if err != nil {
		return err
	}
	if cluster.HasAWSInfrastructure() {
		logger.Info("Provisioning resources for AWS infrastructure")

		logger.Info("Deploying PgBouncer manifests")
		err = utility.DeployPgbouncerManifests(k8sClient, logger)
		if err != nil {
			return errors.Wrap(err, "failed to deploy pgbouncer manifests")
		}

		vpc := cluster.VpcID()
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		logger.Info("Updating PgBouncer ConfigMap")

		err = pgbouncer.UpdatePGBouncerConfigMap(ctx, vpc, provisioner.clusterUpdateStore, cluster.PgBouncerConfig, k8sClient, logger)
		if err != nil {
			return errors.Wrap(err, "failed to update configmap for pgbouncer-configmap")
		}
		logger.Info("PgBouncer ConfigMap updated successfully")
	}

	return nil
}

// UpgradeCluster is no-op for external clusters.
func (provisioner *ExternalProvisioner) UpgradeCluster(cluster *model.Cluster) error {
	provisioner.logger.WithField("cluster", cluster.ID).Info("Cluster is managed externally; skipping upgrade...")

	return nil
}

// RotateClusterNodes is no-op for external clusters.
func (provisioner *ExternalProvisioner) RotateClusterNodes(cluster *model.Cluster) error {
	provisioner.logger.WithField("cluster", cluster.ID).Info("Cluster is managed externally; skipping node rotation...")

	return nil
}

// ResizeCluster is no-op for external clusters.
func (provisioner *ExternalProvisioner) ResizeCluster(cluster *model.Cluster) error {
	provisioner.logger.WithField("cluster", cluster.ID).Info("Cluster is managed externally; skipping resize...")

	return nil
}

// DeleteCluster manages VPC releasing for external clusters.
func (provisioner *ExternalProvisioner) DeleteCluster(cluster *model.Cluster) (bool, error) {
	logger := provisioner.logger.WithField("cluster", cluster.ID)
	logger.Info("Performing external cluster deletion tasks")

	if cluster.ProviderMetadataExternal.HasAWSInfrastructure {
		err := provisioner.awsClient.ReleaseVpc(cluster, logger)
		if err != nil {
			return false, errors.Wrap(err, "failed to release cluster VPC")
		}
	} else {
		logger.Debug("External cluster has no VPC ID; skipping VPC release process...")
	}

	return true, nil
}

func (provisioner *ExternalProvisioner) RefreshClusterMetadata(cluster *model.Cluster) error {
	logger := provisioner.logger.WithField("cluster", cluster.ID)

	logger.Info("Refreshing external cluster metadata")

	cluster.ProvisionerMetadataExternal.ClearWarnings()

	k8sClient, err := provisioner.getKubeClient(cluster)
	if err != nil {
		return err
	}
	versionInfo, err := k8sClient.Clientset.Discovery().ServerVersion()
	if err != nil {
		return errors.Wrap(err, "failed to get kubernetes version")
	}
	cluster.ProvisionerMetadataExternal.Version = strings.TrimLeft(versionInfo.GitVersion, "v")

	// Perform basic validation on the most important services for installations.
	err = basicDeploymentReview("mattermost-operator", k8sClient, logger)
	if err != nil {
		logger.WithError(err).Warn("Failed mattermost-oprerator health check")
		cluster.ProvisionerMetadataExternal.AddWarning(fmt.Sprintf("Health Check: %s", err.Error()))
	}
	err = basicDeploymentReview("pgbouncer", k8sClient, logger)
	if err != nil {
		logger.WithError(err).Warn("Failed pgbouncer health check")
		cluster.ProvisionerMetadataExternal.AddWarning(fmt.Sprintf("Health Check: %s", err.Error()))
	}
	err = basicDeploymentReview("bifrost", k8sClient, logger)
	if err != nil {
		logger.WithError(err).Warn("Failed bifrost health check")
		cluster.ProvisionerMetadataExternal.AddWarning(fmt.Sprintf("Health Check: %s", err.Error()))
	}

	return nil
}

func (provisioner *ExternalProvisioner) getKubeConfigPath(cluster *model.Cluster) (string, error) {
	kubeconfigBytes, err := provisioner.awsClient.SecretsManagerGetSecretBytes(cluster.ProvisionerMetadataExternal.SecretName)
	if err != nil {
		return "", errors.Wrap(err, "failed to get kubeconfig from aws secret manager")
	}

	tempDir, err := os.MkdirTemp("", "external-")
	if err != nil {
		return "", errors.Wrap(err, "failed to create temporary kubeconfig directory")
	}

	filename := filepath.Join(tempDir, fmt.Sprintf("%s.yaml", cluster.ID))
	err = os.WriteFile(filename, kubeconfigBytes, 0600)
	if err != nil {
		return "", errors.Wrap(err, "failed to write kubeconfig")
	}

	return filename, nil
}

func (provisioner *ExternalProvisioner) getKubeClient(cluster *model.Cluster) (*k8s.KubeClient, error) {
	configLocation, err := provisioner.getKubeConfigPath(cluster)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get kube config")
	}

	var k8sClient *k8s.KubeClient
	k8sClient, err = k8s.NewFromFile(configLocation, provisioner.logger)
	if err != nil {
		return nil, errors.Wrap(err, "failed to create k8s client from file")
	}

	return k8sClient, nil
}

func basicDeploymentReview(namespace string, k8sClient *k8s.KubeClient, logger log.FieldLogger) error {
	deployments, err := k8sClient.Clientset.AppsV1().Deployments(namespace).List(context.TODO(), v1.ListOptions{})
	if err != nil {
		return errors.Wrap(err, "encountered error trying to look for mattermost-operator deployment")
	}
	if len(deployments.Items) == 0 {
		return errors.Errorf("failed to find any deployments in namespace %s", namespace)
	} else {
		var deploymentNames []string
		for _, deployment := range deployments.Items {
			deploymentNames = append(deploymentNames, deployment.Name)
		}
		logger.Debugf("Found %d deployment(s) in namespace %s: %s", len(deploymentNames), namespace, strings.Join(deploymentNames, ", "))
	}

	return nil
}
