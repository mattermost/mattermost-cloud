// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.
//

package provisioner

import (
	"context"
	"encoding/base64"
	"fmt"
	"os"

	eksTypes "github.com/aws/aws-sdk-go-v2/service/eks/types"
	"github.com/aws/smithy-go/ptr"
	"github.com/mattermost/mattermost-cloud/internal/store"
	"github.com/mattermost/mattermost-cloud/internal/supervisor"
	"github.com/mattermost/mattermost-cloud/internal/tools/aws"
	"github.com/mattermost/mattermost-cloud/k8s"
	"github.com/mattermost/mattermost-cloud/model"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/tools/clientcmd"
	clientcmdapi "k8s.io/client-go/tools/clientcmd/api"
)

// EKSProvisionerType is provisioner type for EKS clusters.
const EKSProvisionerType = "eks"

type clusterUpdateStore interface {
	UpdateCluster(cluster *model.Cluster) error
}

// EKSProvisioner provisions clusters using AWS EKS.
type EKSProvisioner struct {
	params             ProvisioningParams
	awsClient          aws.AWS
	clusterUpdateStore clusterUpdateStore
	store              model.InstallationDatabaseStoreInterface
	logger             log.FieldLogger
}

var _ supervisor.ClusterProvisioner = (*KopsProvisioner)(nil)

// NewEKSProvisioner creates new EKSProvisioner.
func NewEKSProvisioner(
	params ProvisioningParams,
	awsClient aws.AWS,
	store *store.SQLStore,
	logger log.FieldLogger,
) *EKSProvisioner {
	return &EKSProvisioner{
		params:             params,
		awsClient:          awsClient,
		clusterUpdateStore: store,
		store:              store,
		logger:             logger,
	}
}

// ProvisionerType returns type of the provisioner.
func (provisioner *EKSProvisioner) ProvisionerType() string {
	return EKSProvisionerType
}

// PrepareCluster is noop for EKSProvisioner.
func (provisioner *EKSProvisioner) PrepareCluster(cluster *model.Cluster) bool {
	// Don't regenerate the name if already set.
	if cluster.ProvisionerMetadataEKS.Name != "" {
		return false
	}

	// Generate the kops name using the cluster id.
	cluster.ProvisionerMetadataEKS.Name = fmt.Sprintf("%s-eks-k8s-local", cluster.ID)

	return true
}

// CreateCluster creates the EKS cluster.
func (provisioner *EKSProvisioner) CreateCluster(cluster *model.Cluster, awsClient aws.AWS) error {
	logger := provisioner.logger.WithField("cluster", cluster.ID)

	eksMetadata := cluster.ProvisionerMetadataEKS
	if eksMetadata == nil {
		return errors.New("error: EKS metadata not set when creating EKS cluster")
	}

	err := eksMetadata.ValidateChangeRequest()
	if err != nil {
		return errors.Wrap(err, "EKS Metadata ChangeRequest failed validation")
	}

	if eksMetadata.ChangeRequest.AMI != "" && eksMetadata.ChangeRequest.AMI != "latest" {
		var isAMIValid bool
		isAMIValid, err = awsClient.IsValidAMI(eksMetadata.ChangeRequest.AMI, logger)
		if err != nil {
			return errors.Wrapf(err, "error checking the AWS AMI image %s", eksMetadata.ChangeRequest.AMI)
		}
		if !isAMIValid {
			return errors.Errorf("invalid AWS AMI image %s", eksMetadata.ChangeRequest.AMI)
		}
	}

	// TODO: Add cncVPCCIDR

	var clusterResources aws.ClusterResources
	if eksMetadata.ChangeRequest.VPC != "" && provisioner.params.UseExistingAWSResources {
		clusterResources, err = awsClient.ClaimVPC(eksMetadata.ChangeRequest.VPC, cluster, provisioner.params.Owner, logger)
		if err != nil {
			return errors.Wrap(err, "couldn't claim VPC")
		}
	} else if provisioner.params.UseExistingAWSResources {
		clusterResources, err = awsClient.GetAndClaimVpcResources(cluster, provisioner.params.Owner, logger)
		if err != nil {
			return err
		}
	}

	// Update cluster to set VPC ID that is needed later.
	cluster.ProvisionerMetadataEKS.ChangeRequest.VPC = clusterResources.VpcID

	_, err = awsClient.EnsureEKSCluster(cluster, clusterResources)
	if err != nil {
		releaseErr := awsClient.ReleaseVpc(cluster, logger)
		if releaseErr != nil {
			logger.WithError(releaseErr).Error("Unable to release VPC")
		}

		return errors.Wrap(err, "unable to create eks cluster")
	}

	err = provisioner.clusterUpdateStore.UpdateCluster(cluster)
	if err != nil {
		releaseErr := awsClient.ReleaseVpc(cluster, logger)
		if releaseErr != nil {
			logger.WithError(releaseErr).Error("Failed to release VPC after failed update")
		}
		return errors.Wrap(err, "failed to update EKS metadata with VPC ID")
	}

	return nil
}

// CheckClusterCreated checks if cluster creation finished.
func (provisioner *EKSProvisioner) CheckClusterCreated(cluster *model.Cluster, awsClient aws.AWS) (bool, error) {
	logger := provisioner.logger.WithField("cluster", cluster.ID)

	if cluster.ProvisionerMetadataEKS == nil {
		return false, errors.New("expected EKS metadata not to be nil")
	}

	eksMetadata := cluster.ProvisionerMetadataEKS

	eksCluster, err := awsClient.GetReadyCluster(eksMetadata.Name)
	if err != nil {
		return false, errors.Wrap(err, "failed to check if EKS cluster is ready")
	}
	if eksCluster == nil {
		logger.Info("EKS cluster not ready")
		return false, nil
	}

	if eksCluster.Version != nil {
		eksMetadata.ChangeRequest.Version = *eksCluster.Version
	}

	eksMetadata.ClusterRoleARN = *eksCluster.RoleArn
	eksMetadata.Networking = model.NetworkingCalico

	// When cluster is ready, we need to create LaunchTemplate for NodeGroup.
	launchTemplateVersion, err := awsClient.EnsureLaunchTemplate(eksMetadata.Name, cluster.ProvisionerMetadataEKS)
	if err != nil {
		return false, errors.Wrap(err, "failed to ensure launch template")
	}
	eksMetadata.ChangeRequest.LaunchTemplateVersion = launchTemplateVersion

	err = provisioner.clusterUpdateStore.UpdateCluster(cluster)
	if err != nil {
		return false, errors.Wrap(err, "failed to store cluster")
	}

	// To install Calico Networking, We need to delete VPC CNI plugin (aws-node)
	// and install Calico CNI plugin before creating any pods
	k8sClient, err := provisioner.getKubeClient(cluster)
	if err != nil {
		return false, errors.Wrap(err, "failed to initialize K8s client from kubeconfig")
	}

	// Delete aws-node daemonset to disable VPC CNI plugin
	_ = k8sClient.Clientset.AppsV1().DaemonSets("kube-system").Delete(context.Background(), "aws-node", metav1.DeleteOptions{})

	var files []k8s.ManifestFile
	files = append(files, k8s.ManifestFile{
		Path:            "manifests/eks/calico-eks.yaml",
		DeployNamespace: "kube-system",
	})

	err = k8sClient.CreateFromFiles(files)
	if err != nil {
		return false, err
	}

	return true, nil
}

// CreateNodes creates the EKS nodes.
func (provisioner *EKSProvisioner) CreateNodes(cluster *model.Cluster, awsClient aws.AWS) error {

	eksMetadata := cluster.ProvisionerMetadataEKS
	if eksMetadata == nil {
		return errors.New("error: EKS metadata not set when creating EKS cluster")
	}
	eksMetadata.ChangeRequest.WorkerName = "worker-1"

	nodeGroup, err := awsClient.EnsureEKSClusterNodeGroups(cluster)
	if err != nil || nodeGroup == nil {
		return errors.Wrap(err, "failed to ensure EKS node group")
	}

	err = provisioner.clusterUpdateStore.UpdateCluster(cluster)
	if err != nil {
		return errors.Wrap(err, "failed to store cluster")
	}

	return nil
}

// CheckNodesCreated provisions EKS cluster.
func (provisioner *EKSProvisioner) CheckNodesCreated(cluster *model.Cluster, awsClient aws.AWS) (bool, error) {
	logger := provisioner.logger.WithField("cluster", cluster.ID)

	eksMetadata := cluster.ProvisionerMetadataEKS
	clusterName := eksMetadata.Name
	workerName := eksMetadata.ChangeRequest.WorkerName

	nodeGroup, err := awsClient.GetActiveNode(clusterName, workerName)
	if err != nil {
		return false, errors.Wrap(err, "failed to ensure node groups created")
	}

	if nodeGroup == nil {
		logger.Info("EKS node group not ready")
		return false, nil
	}

	if eksMetadata.NodeInstanceGroups == nil {
		eksMetadata.NodeInstanceGroups = make(map[string]model.EKSInstanceGroupMetadata)
	}

	eksMetadata.WorkerName = *nodeGroup.NodegroupName
	eksMetadata.NodeInstanceGroups[*nodeGroup.NodegroupName] = model.EKSInstanceGroupMetadata{
		NodeInstanceType: nodeGroup.InstanceTypes[0],
		NodeMinCount:     int64(ptr.ToInt32(nodeGroup.ScalingConfig.MinSize)),
		NodeMaxCount:     int64(ptr.ToInt32(nodeGroup.ScalingConfig.MaxSize)),
	}

	eksMetadata.NodeInstanceType = nodeGroup.InstanceTypes[0]
	eksMetadata.NodeMinCount = int64(ptr.ToInt32(nodeGroup.ScalingConfig.MinSize))
	eksMetadata.NodeMaxCount = int64(ptr.ToInt32(nodeGroup.ScalingConfig.MaxSize))

	eksMetadata.NodeRoleARN = *nodeGroup.NodeRole

	err = provisioner.clusterUpdateStore.UpdateCluster(cluster)
	if err != nil {
		return false, errors.Wrap(err, "failed to store cluster")
	}

	return true, nil
}

// ProvisionCluster provisions EKS cluster.
func (provisioner *EKSProvisioner) ProvisionCluster(cluster *model.Cluster, awsClient aws.AWS) error {
	logger := provisioner.logger.WithField("cluster", cluster.ID)

	eksMetadata := cluster.ProvisionerMetadataEKS
	if eksMetadata == nil {
		return errors.New("expected EKS metadata not to be nil when using EKS Provisioner")
	}

	// TODO: ideally we would do it as part of cluster creation as this
	// also is async operation.
	err := awsClient.InstallEKSEBSAddon(cluster)
	if err != nil {
		return errors.Wrap(err, "failed to install EKS EBS Addon")
	}

	kubeConfigPath, err := provisioner.getKubeConfigPath(cluster)
	if err != nil {
		return errors.Wrap(err, "failed to get kubeconfig file path")
	}

	return provisionCluster(cluster, kubeConfigPath, awsClient, provisioner.params, provisioner.store, logger)
}

// UpgradeCluster upgrades EKS cluster - not implemented.
func (provisioner *EKSProvisioner) UpgradeCluster(cluster *model.Cluster, awsClient aws.AWS) error {
	logger := provisioner.logger.WithField("cluster", cluster.ID)

	eksMetadata := cluster.ProvisionerMetadataEKS
	changeRequest := eksMetadata.ChangeRequest

	err := eksMetadata.ValidateChangeRequest()
	if err != nil {
		return errors.Wrap(err, "eks Metadata ChangeRequest failed validation")
	}

	if changeRequest.AMI != "" && changeRequest.AMI != "latest" {
		var isAMIValid bool
		isAMIValid, err = awsClient.IsValidAMI(eksMetadata.ChangeRequest.AMI, logger)
		if err != nil {
			return errors.Wrapf(err, "error checking the AWS AMI image %s", eksMetadata.ChangeRequest.AMI)
		}
		if !isAMIValid {
			return errors.Errorf("invalid AWS AMI image %s", eksMetadata.ChangeRequest.AMI)
		}
	}

	if changeRequest.AMI != "" || changeRequest.MaxPodsPerNode > 0 {
		// When cluster is ready, we need to create LaunchTemplate for NodeGroup.
		var launchTemplateVersion *int64
		launchTemplateVersion, err = awsClient.UpdateLaunchTemplate(eksMetadata.Name, cluster.ProvisionerMetadataEKS)
		if err != nil {
			return errors.Wrap(err, "failed to update launch template")
		}
		changeRequest.LaunchTemplateVersion = launchTemplateVersion

		err = awsClient.EnsureEKSClusterNodeGroupUpdated(cluster)
		if err != nil {
			return errors.Wrap(err, "failed to update node group")
		}

		err = provisioner.clusterUpdateStore.UpdateCluster(cluster)
		if err != nil {
			return errors.Wrap(err, "failed to store cluster")
		}
	}

	if changeRequest.Version != "" {
		err = awsClient.EnsureEKSClusterUpdated(cluster)
		if err != nil {
			return errors.Wrap(err, "failed to update EKS cluster")
		}
	}

	wait := 3600 // seconds
	logger.Infof("Waiting up to %d seconds for k8s cluster to become ready...", wait)
	err = awsClient.WaitForClusterReadiness(eksMetadata.Name, wait)
	if err != nil {
		return err
	}

	return nil
}

// RotateClusterNodes rotates cluster nodes - not implemented.
func (provisioner *EKSProvisioner) RotateClusterNodes(cluster *model.Cluster) error {
	return nil
}

// ResizeCluster resizes cluster - not implemented.
func (provisioner *EKSProvisioner) ResizeCluster(cluster *model.Cluster, awsClient aws.AWS) error {
	return nil
}

// RefreshKopsMetadata is a noop for EKSProvisioner.
func (provisioner *EKSProvisioner) RefreshKopsMetadata(cluster *model.Cluster) error {
	return nil
}

func (provisioner *EKSProvisioner) getKubeConfigPath(cluster *model.Cluster) (string, error) {
	configLocation, err := provisioner.prepareClusterKubeConfig(cluster.ProvisionerMetadataEKS.Name)
	if err != nil {
		return "", errors.Wrap(err, "failed to prepare kube config")
	}

	return configLocation, nil
}

func (provisioner *EKSProvisioner) getKubeClient(cluster *model.Cluster) (*k8s.KubeClient, error) {
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

// DeleteCluster deletes EKS cluster.
func (provisioner *EKSProvisioner) DeleteCluster(cluster *model.Cluster, awsClient aws.AWS) (bool, error) {
	logger := provisioner.logger.WithField("cluster", cluster.ID)

	logger.Info("Deleting cluster")

	eksMetadata := cluster.ProvisionerMetadataEKS
	if eksMetadata == nil {
		return false, errors.New("expected EKS metadata not to be nil when using EKS Provisioner")
	}

	deleted, err := awsClient.EnsureNodeGroupsDeleted(eksMetadata.Name, eksMetadata.WorkerName)
	if err != nil {
		return false, errors.Wrap(err, "failed to delete node groups")
	}
	if !deleted {
		return false, nil
	}

	deleted, err = awsClient.EnsureLaunchTemplateDeleted(eksMetadata.Name)
	if err != nil {
		return false, errors.Wrap(err, "failed to delete launch template")
	}
	if !deleted {
		return false, nil
	}

	deleted, err = awsClient.EnsureEKSClusterDeleted(eksMetadata.Name)
	if err != nil {
		return false, errors.Wrap(err, "failed to delete EKS cluster")
	}
	if !deleted {
		return false, nil
	}

	err = awsClient.ReleaseVpc(cluster, logger)
	if err != nil {
		return false, errors.Wrap(err, "failed to release cluster VPC")
	}

	logger.Info("Successfully deleted EKS cluster")

	return true, nil
}

// GetClusterResources returns resources for EKS cluster.
func (provisioner *EKSProvisioner) GetClusterResources(cluster *model.Cluster, onlySchedulable bool, logger log.FieldLogger) (*k8s.ClusterResources, error) {
	logger = logger.WithField("cluster", cluster.ID)

	configLocation, err := provisioner.prepareClusterKubeConfig(cluster.ProvisionerMetadataEKS.Name)
	if err != nil {
		return nil, errors.Wrap(err, "failed to prepare kubeconfig")
	}

	return getClusterResources(configLocation, onlySchedulable, logger)
}

// GetPublicLoadBalancerEndpoint returns endpoint of public load balancer.
func (provisioner *EKSProvisioner) GetPublicLoadBalancerEndpoint(cluster *model.Cluster, namespace string) (string, error) {
	logger := provisioner.logger.WithFields(log.Fields{
		"cluster":         cluster.ID,
		"nginx-namespace": namespace,
	})

	configLocation, err := provisioner.prepareClusterKubeConfig(cluster.ProvisionerMetadataEKS.Name)
	if err != nil {
		return "", errors.Wrap(err, "failed to prepare kubeconfig")
	}

	return getPublicLoadBalancerEndpoint(configLocation, namespace, logger)
}

func (provisioner *EKSProvisioner) prepareClusterKubeConfig(clusterName string) (string, error) {
	eksCluster, err := provisioner.awsClient.GetReadyCluster(clusterName)
	if err != nil {
		return "", errors.Wrap(err, "failed to get eks cluster")
	}
	if eksCluster == nil {
		return "", errors.New("eks cluster not ready")
	}

	kubeConfig, err := newEKSKubeconfig(eksCluster, provisioner.awsClient)
	if err != nil {
		return "", errors.Wrap(err, "failed to create kubeconfig")
	}

	kubeConfigFile, err := os.CreateTemp("", clusterName)
	if err != nil {
		return "", errors.Wrap(err, "failed to create kubeconfig tempfile")
	}
	defer kubeConfigFile.Close()

	rawKubeconfig, err := clientcmd.Write(kubeConfig)
	if err != nil {
		return "", errors.Wrap(err, "failed to serialize kubeconfig")
	}
	_, err = kubeConfigFile.Write(rawKubeconfig)
	if err != nil {
		return "", errors.Wrap(err, "failed to write kubeconfig")
	}

	return kubeConfigFile.Name(), nil
}

// newEKSKubeconfig creates kubeconfig for EKS cluster.
func newEKSKubeconfig(cluster *eksTypes.Cluster, aws aws.AWS) (clientcmdapi.Config, error) {
	region := aws.GetRegion()
	accountID, err := aws.GetAccountID()
	if err != nil {
		return clientcmdapi.Config{}, errors.Wrap(err, "failed to get account ID")
	}
	clusterName := *cluster.Name

	fullClusterName := fmt.Sprintf("arn:aws:eks:%s:%s:cluster/%s", region, accountID, clusterName)
	userName := fmt.Sprintf("arn:aws:eks:%s:%s:cluster/%s", region, accountID, clusterName)

	cert, err := base64.StdEncoding.DecodeString(*cluster.CertificateAuthority.Data)
	if err != nil {
		return clientcmdapi.Config{}, errors.Wrap(err, "failed to base64 decode cert")
	}

	return clientcmdapi.Config{
		Clusters: map[string]*clientcmdapi.Cluster{
			fullClusterName: {
				Server:                   *cluster.Endpoint,
				CertificateAuthorityData: cert,
			},
		},
		Contexts: map[string]*clientcmdapi.Context{
			fullClusterName: {
				Cluster:  fullClusterName,
				AuthInfo: userName,
			},
		},
		CurrentContext: fullClusterName,
		AuthInfos: map[string]*clientcmdapi.AuthInfo{
			userName: {
				// TODO: we can explore other ways to authenticate,
				// for now we just use aws cli as this is the simplest
				// and best documented one.
				Exec: &clientcmdapi.ExecConfig{
					Command:    "aws",
					Args:       []string{"--region", region, "eks", "get-token", "--cluster-name", *cluster.Name},
					APIVersion: "client.authentication.k8s.io/v1beta1",
				},
			},
		},
	}, nil
}

func (provisioner *EKSProvisioner) RefreshClusterMetadata(cluster *model.Cluster) error {
	if cluster.ProvisionerMetadataEKS != nil {
		cluster.ProvisionerMetadataEKS.ApplyChangeRequest()
		cluster.ProvisionerMetadataEKS.ClearChangeRequest()
		cluster.ProvisionerMetadataEKS.ClearWarnings()
	}
	return nil
}
