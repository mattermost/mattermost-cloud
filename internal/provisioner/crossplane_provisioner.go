// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.
//

package provisioner

import (
	"context"
	"fmt"

	crossplaneV1Alpha1 "github.com/mattermost/mattermost-cloud-crossplane/apis/crossplane/v1alpha1"
	"github.com/mattermost/mattermost-cloud/internal/supervisor"
	"github.com/mattermost/mattermost-cloud/internal/tools/aws"
	"github.com/mattermost/mattermost-cloud/k8s"
	"github.com/mattermost/mattermost-cloud/model"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
	k8sErrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	// CrossplaneProvisionerType is provisioner type for EKS clusters.
	CrossplaneProvisionerType = "crossplane"

	// crossplaneProvisionerNamespace the namespace where the crossplane resources are created.
	// TODO: change to a proper namespace when tests are done.
	crossplaneProvisionerNamespace = "mm-xplane-eks-01"
)

// CrossplaneProvisioner provisions clusters using Crossplane
type CrossplaneProvisioner struct {
	awsClient         aws.AWS
	kubeClient        *k8s.KubeClient
	clusterStore      clusterUpdateStore
	parameters        ProvisioningParams
	kube2IAMAccountID string
	logger            log.FieldLogger
}

var _ supervisor.ClusterProvisioner = (*CrossplaneProvisioner)(nil)

// NewCrossplaneProvisioner creates a new Crossplane provisioner.
func NewCrossplaneProvisioner(
	kubeClient *k8s.KubeClient,
	awsClient aws.AWS,
	parameters ProvisioningParams,
	clusterStore clusterUpdateStore,
	kube2IAMAccountID string,
	logger log.FieldLogger,
) *CrossplaneProvisioner {
	return &CrossplaneProvisioner{
		kubeClient:        kubeClient,
		awsClient:         awsClient,
		parameters:        parameters,
		clusterStore:      clusterStore,
		kube2IAMAccountID: kube2IAMAccountID,
		logger:            logger,
	}
}

// PrepareCluster prepares the cluster for provisioning by assigning it a name (if not manually
// provided) and claiming the VPC required for the cluster to be provisioned.
func (provisioner *CrossplaneProvisioner) PrepareCluster(cluster *model.Cluster) bool {
	if cluster.ProvisionerMetadataCrossplane.Name == "" {
		cluster.ProvisionerMetadataCrossplane.Name = fmt.Sprintf("%s-crossplane-k8s-local", cluster.ID)
	}

	var (
		resources aws.ClusterResources
		err       error
	)
	if cluster.ProvisionerMetadataCrossplane.VPC == "" {
		resources, err = provisioner.awsClient.GetAndClaimVpcResources(cluster, provisioner.parameters.Owner, provisioner.logger)
	} else {
		resources, err = provisioner.awsClient.ClaimVPC(cluster.ProvisionerMetadataCrossplane.VPC, cluster, provisioner.parameters.Owner, provisioner.logger)
	}
	if err != nil {
		provisioner.logger.WithError(err).WithField("vpc", cluster.ProvisionerMetadataCrossplane.VPC).Error("Failed to claim VPC resources")
		return false
	}
	cluster.ProvisionerMetadataCrossplane.VPC = resources.VpcID
	cluster.ProvisionerMetadataCrossplane.PublicSubnets = resources.PublicSubnetsIDs
	cluster.ProvisionerMetadataCrossplane.PrivateSubnets = resources.PrivateSubnetIDs

	return true
}

// CreateCluster creates the Crossplane cluster resource in the kubernetes CNC cluster.
func (provisioner *CrossplaneProvisioner) CreateCluster(cluster *model.Cluster) error {
	ctx := context.TODO()
	obj := &crossplaneV1Alpha1.MMK8S{
		ObjectMeta: metav1.ObjectMeta{
			Name:      cluster.ProvisionerMetadataCrossplane.Name,
			Namespace: crossplaneProvisionerNamespace,
		},
		Spec: crossplaneV1Alpha1.EKSSpec{
			CompositionSelector: crossplaneV1Alpha1.EKSSpecCompositionSelector{
				MatchLabels: crossplaneV1Alpha1.EKSSpecCompositionSelectorMatchLabels{
					Provider: "aws",
					Service:  "eks",
				},
			},
			ID: cluster.ID,
			Parameters: crossplaneV1Alpha1.EKSSpecParameters{
				Version:               cluster.ProvisionerMetadataCrossplane.KubernetesVersion,
				AccountID:             provisioner.kube2IAMAccountID,
				Region:                cluster.ProvisionerMetadataCrossplane.Region,
				Environment:           "dev", // TODO
				ClusterShortName:      cluster.ID,
				EndpointPrivateAccess: true,  // TODO
				EndpointPublicAccess:  false, // TODO
				VpcID:                 cluster.ProvisionerMetadataCrossplane.VPC,
				SubnetIds:             cluster.ProvisionerMetadataCrossplane.PublicSubnets,
				PrivateSubnetIds:      cluster.ProvisionerMetadataCrossplane.PrivateSubnets,
				NodeCount:             int(cluster.ProvisionerMetadataCrossplane.NodeCount),
				InstanceType:          cluster.ProvisionerMetadataCrossplane.InstanceType,
				ImageID:               cluster.ProvisionerMetadataCrossplane.AMI,
				LaunchTemplateVersion: *cluster.ProvisionerMetadataCrossplane.LaunchTemplateVersion,
			},
			ResourceConfig: crossplaneV1Alpha1.EKSSpecResourceConfig{
				ProviderConfigName: "crossplane-provider-config",
			},
		},
	}

	_, err := provisioner.kubeClient.CrossplaneClient.CloudV1alpha1().MMK8Ss(crossplaneProvisionerNamespace).Create(ctx, obj, metav1.CreateOptions{})
	if err != nil {
		return errors.Wrap(err, "error creating object")
	}

	return nil
}

// CheckClusterCreated checks if cluster creation finished.
func (provisioner *CrossplaneProvisioner) CheckClusterCreated(cluster *model.Cluster) (bool, error) {
	resource, err := provisioner.kubeClient.CrossplaneClient.CloudV1alpha1().MMK8Ss(crossplaneProvisionerNamespace).Get(context.TODO(), cluster.ID, metav1.GetOptions{})
	if err != nil && !k8sErrors.IsNotFound(err) {
		return false, errors.Wrap(err, "error getting crossplane resource information")
	}

	ready, err := resource.Status.GetReadyCondition()
	if err != nil && !errors.Is(err, crossplaneV1Alpha1.ErrConditionNotFound) {
		return false, errors.Wrap(err, "error getting crossplane cluster ready status")
	}

	if ready == nil {
		return false, nil
	}

	return ready.Status == metav1.ConditionTrue, nil
}

// CreateNodes is no-op.
func (provisioner *CrossplaneProvisioner) CreateNodes(cluster *model.Cluster) error {
	return nil
}

// CheckNodesCreated is no-op.
func (provisioner *CrossplaneProvisioner) CheckNodesCreated(cluster *model.Cluster) (bool, error) {
	return true, nil
}

// ProvisionCluster
func (provisioner *CrossplaneProvisioner) ProvisionCluster(cluster *model.Cluster) error {
	return nil
}

// UpgradeCluster is no-op.
func (provisioner *CrossplaneProvisioner) UpgradeCluster(cluster *model.Cluster) error {
	return nil
}

// RotateClusterNodes is no-op.
func (provisioner *CrossplaneProvisioner) RotateClusterNodes(cluster *model.Cluster) error {
	return nil
}

// ResizeCluster is no-op.
func (provisioner *CrossplaneProvisioner) ResizeCluster(cluster *model.Cluster) error {
	return nil
}

// DeleteCluster deletes Crossplane cluster.
func (provisioner *CrossplaneProvisioner) DeleteCluster(cluster *model.Cluster) (bool, error) {
	logger := provisioner.logger.WithField("cluster", cluster.ID)

	err := provisioner.kubeClient.CrossplaneClient.CloudV1alpha1().MMK8Ss(crossplaneProvisionerNamespace).Delete(context.TODO(), cluster.ProvisionerMetadataCrossplane.Name, metav1.DeleteOptions{})
	if err != nil {
		provisioner.logger.WithError(err).Error("Failed to delete crossplane resource")
		// return false, errors.Wrap(err, "failed to delete crossplane resource")
	}

	err = provisioner.awsClient.ReleaseVpc(cluster, logger)
	if err != nil {
		return false, errors.Wrap(err, "failed to release cluster VPC")
	}

	logger.Info("Successfully deleted Crossplane cluster")

	return true, nil
}

// RefreshClusterMetadata is no-op.
func (provisioner *CrossplaneProvisioner) RefreshClusterMetadata(cluster *model.Cluster) error {
	return nil
}
