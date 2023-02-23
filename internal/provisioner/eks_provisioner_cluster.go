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
	"github.com/mattermost/mattermost-cloud/internal/store"
	"github.com/mattermost/mattermost-cloud/internal/tools/aws"
	"github.com/mattermost/mattermost-cloud/k8s"
	"github.com/mattermost/mattermost-cloud/model"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/tools/clientcmd"
	clientcmdapi "k8s.io/client-go/tools/clientcmd/api"
)

// EKSProvisionerType is Provisioner type for EKS clusters.
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

// ProvisionerType returns type of the Provisioner.
func (provisioner *EKSProvisioner) ProvisionerType() string {
	return EKSProvisionerType
}

// PrepareCluster is noop for EKSProvisioner.
func (provisioner *EKSProvisioner) PrepareCluster(cluster *model.Cluster) bool {
	return false
}

// CreateCluster creates the EKS cluster.
func (provisioner *EKSProvisioner) CreateCluster(cluster *model.Cluster, awsClient aws.AWS) error {
	logger := provisioner.logger.WithField("cluster", cluster.ID)

	eksMetadata := cluster.ProvisionerMetadataEKS
	if eksMetadata == nil {
		return errors.New("error: EKS metadata not set when creating EKS cluster")
	}

	var clusterResources aws.ClusterResources
	var err error
	if eksMetadata.VPC != "" {
		clusterResources, err = awsClient.ClaimVPC(eksMetadata.VPC, cluster, provisioner.params.Owner, logger)
		if err != nil {
			return err
		}
	} else {
		clusterResources, err = awsClient.GetAndClaimVpcResources(cluster, provisioner.params.Owner, logger)
		if err != nil {
			return err
		}
	}

	// Update cluster to set VPC ID that is needed later.
	cluster.ProvisionerMetadataEKS.VPC = clusterResources.VpcID
	err = provisioner.clusterUpdateStore.UpdateCluster(cluster)
	if err != nil {
		releaseErr := awsClient.ReleaseVpc(cluster, logger)
		if releaseErr != nil {
			logger.WithError(releaseErr).Error("Failed to release VPC after failed update")
		}
		return errors.Wrap(err, "failed to update EKS metadata with VPC ID")
	}

	_, err = awsClient.EnsureEKSCluster(cluster, clusterResources, *eksMetadata)
	if err != nil {
		return errors.Wrap(err, "failed to ensure EKS cluster exists")
	}

	return nil
}

// CheckClusterCreated checks if cluster creation finished.
func (provisioner *EKSProvisioner) CheckClusterCreated(cluster *model.Cluster, awsClient aws.AWS) (bool, error) {
	logger := provisioner.logger.WithField("cluster", cluster.ID)

	if cluster.ProvisionerMetadataEKS == nil {
		return false, errors.New("expected EKS metadata not to be nil")
	}

	ready, err := awsClient.IsClusterReady(cluster.ID)
	if err != nil {
		return false, errors.Wrap(err, "failed to check if EKS cluster is ready")
	}
	if !ready {
		logger.Info("EKS cluster not ready")
		return false, nil
	}

	// When cluster is ready, we need to create LaunchTemplate for NodeGroup.
	_, err = awsClient.EnsureLaunchTemplate(cluster.ID, *cluster.ProvisionerMetadataEKS)
	if err != nil {
		return false, errors.Wrap(err, "failed to ensure launch template")
	}

	// To install Calico Networking, We need to delete VPC CNI plugin (aws-node)
	// and install Calico CNI plugin before creating any pods
	kubeConfigFile, err := provisioner.prepareClusterKubeconfig(cluster.ID)
	if err != nil {
		return false, errors.Wrap(err, "failed to prepare kubeconfig file")
	}

	k8sClient, err := k8s.NewFromFile(kubeConfigFile, logger)
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

// CheckNodesCreated provisions EKS cluster.
func (provisioner *EKSProvisioner) CheckNodesCreated(cluster *model.Cluster, awsClient aws.AWS) (bool, error) {
	logger := provisioner.logger.WithField("cluster", cluster.ID)

	clusterResources, err := awsClient.GetVpcResourcesByVpcID(cluster.ProvisionerMetadataEKS.VPC, logger)
	if err != nil {
		return false, errors.Wrap(err, "failed to get VPC resources")
	}

	nodeGroups, err := awsClient.EnsureEKSClusterNodeGroups(cluster, clusterResources, *cluster.ProvisionerMetadataEKS)
	if err != nil {
		return false, errors.Wrap(err, "failed to ensure node groups created")
	}

	for _, ng := range nodeGroups {
		if ng.Status == eksTypes.NodegroupStatusCreateFailed {
			return false, errors.Errorf("failed to check node group ready %q", *ng.NodegroupName)
		}

		if ng.Status != eksTypes.NodegroupStatusActive {
			logger.WithField("node=group", *ng.NodegroupName).Info("Node group not yet ready")
			return false, nil
		}
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

	err = awsClient.AllowEKSPostgresTraffic(cluster, *eksMetadata)
	if err != nil {
		return errors.Wrap(err, "failed to create ingress rule to allow Postgres traffic")
	}

	kubeconfigFile, err := provisioner.prepareClusterKubeconfig(cluster.ID)
	if err != nil {
		return errors.Wrap(err, "failed to prepare kubeconfig file")
	}

	return provisionCluster(cluster, kubeconfigFile, awsClient, provisioner.params, provisioner.store, logger)
}

// UpgradeCluster upgrades EKS cluster - not implemented.
func (provisioner *EKSProvisioner) UpgradeCluster(cluster *model.Cluster, awsClient aws.AWS) error {
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

func (provisioner *EKSProvisioner) GetKubeConfigPath(cluster *model.Cluster) (string, error) {
	configLocation, err := provisioner.prepareClusterKubeconfig(cluster.ID)
	if err != nil {
		return "", errors.Wrap(err, "failed to prepare kubeconfig")
	}

	return configLocation, nil
}

func (provisioner *EKSProvisioner) GetKubeClient(cluster *model.Cluster) (*k8s.KubeClient, error) {
	configLocation, err := provisioner.GetKubeConfigPath(cluster)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get kops config from cache")
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

	err := awsClient.RevokeEKSPostgresTraffic(cluster, *eksMetadata)
	if err != nil {
		return false, errors.Wrap(err, "failed to delete ingress rule to allow Postgres traffic")
	}

	deleted, err := awsClient.EnsureNodeGroupsDeleted(cluster)
	if err != nil {
		return false, errors.Wrap(err, "failed to delete node groups")
	}
	if !deleted {
		return false, nil
	}

	deleted, err = awsClient.EnsureLaunchTemplateDeleted(cluster.ID)
	if err != nil {
		return false, errors.Wrap(err, "failed to delete launch template")
	}
	if !deleted {
		return false, nil
	}

	deleted, err = awsClient.EnsureEKSClusterDeleted(cluster)
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

	configLocation, err := provisioner.prepareClusterKubeconfig(cluster.ID)
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

	configLocation, err := provisioner.prepareClusterKubeconfig(cluster.ID)
	if err != nil {
		return "", errors.Wrap(err, "failed to prepare kubeconfig")
	}

	return getPublicLoadBalancerEndpoint(configLocation, namespace, logger)
}

func (provisioner *EKSProvisioner) prepareClusterKubeconfig(clusterID string) (string, error) {
	eksCluster, err := provisioner.awsClient.GetEKSCluster(clusterID)
	if err != nil {
		return "", errors.Wrap(err, "failed to get eks cluster")
	}

	kubeconfig, err := NewEKSKubeconfig(eksCluster, provisioner.awsClient)
	if err != nil {
		return "", errors.Wrap(err, "failed to create kubeconfig")
	}

	kubconfFile, err := os.CreateTemp("", clusterID)
	if err != nil {
		return "", errors.Wrap(err, "failed to create kubeconfig tempfile")
	}
	defer kubconfFile.Close()

	rawKubeconfig, err := clientcmd.Write(kubeconfig)
	if err != nil {
		return "", errors.Wrap(err, "failed to serialize kubeconfig")
	}
	_, err = kubconfFile.Write(rawKubeconfig)
	if err != nil {
		return "", errors.Wrap(err, "failed to write kubeconfig")
	}

	return kubconfFile.Name(), nil
}

// NewEKSKubeconfig creates kubeconfig for EKS cluster.
func NewEKSKubeconfig(cluster *eksTypes.Cluster, aws aws.AWS) (clientcmdapi.Config, error) {
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
