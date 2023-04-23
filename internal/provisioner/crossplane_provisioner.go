// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.
//

package provisioner

import (
	"context"
	"fmt"
	"os"

	crossplaneV1Alpha1 "github.com/mattermost/mattermost-cloud-crossplane/apis/crossplane/v1alpha1"
	"github.com/mattermost/mattermost-cloud/internal/supervisor"
	"github.com/mattermost/mattermost-cloud/internal/tools/aws"
	"github.com/mattermost/mattermost-cloud/internal/tools/utils"
	"github.com/mattermost/mattermost-cloud/k8s"
	"github.com/mattermost/mattermost-cloud/model"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
	k8sErrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/tools/clientcmd"
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
	databaseStore     model.ClusterUtilityDatabaseStoreInterface
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
	databaseStore model.ClusterUtilityDatabaseStoreInterface,
	kube2IAMAccountID string,
	logger log.FieldLogger,
) *CrossplaneProvisioner {
	return &CrossplaneProvisioner{
		kubeClient:        kubeClient,
		awsClient:         awsClient,
		parameters:        parameters,
		databaseStore:     databaseStore,
		kube2IAMAccountID: kube2IAMAccountID,
		logger:            logger,
	}
}

// PrepareCluster prepares the cluster for provisioning by assigning it a name (if not manually
// provided) and claiming the VPC required for the cluster to be provisioned and checking and
// setting any creation request parameters.
func (provisioner *CrossplaneProvisioner) PrepareCluster(cluster *model.Cluster) bool {
	logger := provisioner.logger.WithField("cluster", cluster.ID)
	metadata := cluster.ProvisionerMetadataCrossplane

	// Validate the cluster request AMI if provided.
	if metadata.ChangeRequest.AMI != "" && metadata.ChangeRequest.AMI != "latest" {
		var isAMIValid bool
		isAMIValid, err := provisioner.awsClient.IsValidAMI(metadata.ChangeRequest.AMI, logger)
		if err != nil {
			logger.WithError(err).Errorf("error checking the AWS AMI image %s", metadata.ChangeRequest.AMI)
			return false
		}
		if !isAMIValid {
			logger.Errorf("invalid AWS AMI image %s", metadata.ChangeRequest.AMI)
			return false
		}
	}

	metadata.Name = fmt.Sprintf("%s-crossplane-k8s-local", cluster.ID)

	if err := metadata.ApplyChangeRequest(); err != nil {
		logger.WithError(err).Error("Failed to apply change request")
		return false
	}

	var (
		resources aws.ClusterResources
		err       error
	)
	if metadata.VPC == "" {
		resources, err = provisioner.awsClient.GetAndClaimVpcResources(cluster, provisioner.parameters.Owner, provisioner.logger)
	} else {
		resources, err = provisioner.awsClient.ClaimVPC(metadata.VPC, cluster, provisioner.parameters.Owner, provisioner.logger)
	}
	if err != nil {
		provisioner.logger.WithError(err).WithField("vpc", metadata.VPC).Error("Failed to claim VPC resources")
		return false
	}

	metadata.VPC = resources.VpcID
	for _, subnet := range resources.PublicSubnets {
		if utils.Contains[string](cluster.ProviderMetadataAWS.Zones, *subnet.AvailabilityZone) {
			metadata.Subnets = append(metadata.Subnets, *subnet.SubnetId)
		}
	}
	for _, subnet := range resources.PrivateSubnets {
		if utils.Contains[string](cluster.ProviderMetadataAWS.Zones, *subnet.AvailabilityZone) {
			metadata.Subnets = append(metadata.Subnets, *subnet.SubnetId)
			metadata.PrivateSubnets = append(metadata.PrivateSubnets, *subnet.SubnetId)
		}
	}
	metadata.AccountID = provisioner.kube2IAMAccountID

	return true
}

// CreateCluster creates the Crossplane cluster resource in the kubernetes CNC cluster.
func (provisioner *CrossplaneProvisioner) CreateCluster(cluster *model.Cluster) error {
	ctx := context.TODO()
	// logger := provisioner.logger.WithField("cluster", cluster.ID)

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
				SubnetIds:             cluster.ProvisionerMetadataCrossplane.Subnets,
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

	// bites, err := json.Marshal(obj)
	// provisioner.logger.Warnf("obj: %s", string(bites))

	_, err := provisioner.kubeClient.CrossplaneClient.CloudV1alpha1().MMK8Ss(crossplaneProvisionerNamespace).Create(ctx, obj, metav1.CreateOptions{})
	if err != nil {
		return errors.Wrap(err, "error creating object")
	}

	return nil
}

// CheckClusterCreated checks if cluster creation finished.
func (provisioner *CrossplaneProvisioner) CheckClusterCreated(cluster *model.Cluster) (bool, error) {
	resources, err := provisioner.kubeClient.CrossplaneClient.CloudV1alpha1().MMK8Ss(crossplaneProvisionerNamespace).List(context.TODO(), metav1.ListOptions{
		FieldSelector: fmt.Sprintf("metadata.name=%s", cluster.ProvisionerMetadataCrossplane.Name),
	})
	if err != nil && !k8sErrors.IsNotFound(err) {
		return false, errors.Wrap(err, "error getting crossplane resource information")
	}

	if len(resources.Items) == 0 {
		return false, fmt.Errorf("no crossplane resource found")
	}

	if len(resources.Items) > 1 {
		return false, fmt.Errorf("expected one eks cluster, found %d", len(resources.Items))
	}

	resource := resources.Items[0]
	ready, err := resource.Status.GetReadyCondition()
	if err != nil && !errors.Is(err, crossplaneV1Alpha1.ErrConditionNotFound) {
		return false, errors.Wrap(err, "error getting crossplane cluster ready status")
	}

	// TODO: Check if cluster has been pending creation for too long

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
	logger := provisioner.logger.WithField("cluster", cluster.ID)

	metadata := cluster.ProvisionerMetadataCrossplane
	if metadata == nil {
		return errors.New("expected metadata to be present")
	}

	kubeConfigPath, err := provisioner.getKubeConfigPath(cluster)
	if err != nil {
		return errors.Wrap(err, "failed to get kubeconfig file path")
	}

	return provisionCluster(cluster, kubeConfigPath, provisioner.awsClient, provisioner.parameters, provisioner.databaseStore, logger)
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

	err := provisioner.awsClient.ReleaseVpc(cluster, logger)
	if err != nil {
		return false, errors.Wrap(err, "failed to release cluster VPC")
	}

	err = provisioner.kubeClient.CrossplaneClient.CloudV1alpha1().MMK8Ss(crossplaneProvisionerNamespace).Delete(context.TODO(), cluster.ProvisionerMetadataCrossplane.Name, metav1.DeleteOptions{})
	if err != nil {
		return false, errors.Wrap(err, "failed to delete crossplane resource")
	}

	logger.Info("Successfully deleted Crossplane cluster")

	return true, nil
}

// RefreshClusterMetadata is no-op.
func (provisioner *CrossplaneProvisioner) RefreshClusterMetadata(cluster *model.Cluster) error {
	return nil
}

func (provisioner *CrossplaneProvisioner) getKubeConfigPath(cluster *model.Cluster) (string, error) {
	clusterName := "cluster-" + cluster.ID
	eksCluster, err := provisioner.awsClient.GetActiveEKSCluster(clusterName)
	if err != nil {
		return "", errors.Wrap(err, "failed to get EKS cluster")
	}
	if eksCluster == nil {
		return "", errors.New("EKS cluster not ready")
	}

	kubeconfig, err := newEKSKubeConfig(eksCluster, provisioner.awsClient)
	if err != nil {
		return "", errors.Wrap(err, "failed to create kubeconfig")
	}

	kubeconfigFile, err := os.CreateTemp("", clusterName)
	if err != nil {
		return "", errors.Wrap(err, "failed to create kubeconfig tempfile")
	}
	defer kubeconfigFile.Close()

	rawKubeconfig, err := clientcmd.Write(kubeconfig)
	if err != nil {
		return "", errors.Wrap(err, "failed to serialize kubeconfig")
	}
	_, err = kubeconfigFile.Write(rawKubeconfig)
	if err != nil {
		return "", errors.Wrap(err, "failed to write kubeconfig")
	}

	return kubeconfigFile.Name(), nil
}

func (provisioner *CrossplaneProvisioner) getKubeClient(cluster *model.Cluster) (*k8s.KubeClient, error) {
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
