// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.
//

package provisioner

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/mattermost/mattermost-cloud/internal/store"
	"github.com/mattermost/mattermost-cloud/internal/supervisor"
	"github.com/mattermost/mattermost-cloud/internal/tools/aws"
	"github.com/mattermost/mattermost-cloud/k8s"
	"github.com/mattermost/mattermost-cloud/model"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
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

// PrepareCluster is no-op for external clusters.
func (provisioner *ExternalProvisioner) PrepareCluster(cluster *model.Cluster) bool {
	return false
}

// CreateCluster is no-op for external clusters.
func (provisioner *ExternalProvisioner) CreateCluster(cluster *model.Cluster) error {
	provisioner.logger.WithField("cluster", cluster.ID).Info("Cluster is managed externally; skipping creation...")

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

// ProvisionCluster is no-op for external clusters.
func (provisioner *ExternalProvisioner) ProvisionCluster(cluster *model.Cluster) error {
	provisioner.logger.WithField("cluster", cluster.ID).Info("Cluster is managed externally; skipping provision...")

	return nil
}

// UpgradeCluster is no-op for external clusters.
func (provisioner *ExternalProvisioner) UpgradeCluster(cluster *model.Cluster) error {
	provisioner.logger.WithField("cluster", cluster.ID).Info("Cluster is managed externally; skipping upgrade...")

	return nil
}

// RotateClusterNodes is no-op for external clusters.
func (provisioner *ExternalProvisioner) RotateClusterNodes(cluster *model.Cluster) error {
	return nil
}

// ResizeCluster is no-op for external clusters.
func (provisioner *ExternalProvisioner) ResizeCluster(cluster *model.Cluster) error {
	provisioner.logger.WithField("cluster", cluster.ID).Info("Cluster is managed externally; skipping resize...")

	return nil
}

// DeleteCluster is no-op for external clusters.
func (provisioner *ExternalProvisioner) DeleteCluster(cluster *model.Cluster) (bool, error) {
	provisioner.logger.WithField("cluster", cluster.ID).Info("Cluster is managed externally; no deletion steps required...")

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
