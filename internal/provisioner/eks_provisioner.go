// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.
//

package provisioner

import (
	"encoding/base64"
	"fmt"
	"os"
	"sync"

	eksTypes "github.com/aws/aws-sdk-go-v2/service/eks/types"
	"github.com/aws/smithy-go/ptr"
	"github.com/mattermost/mattermost-cloud/internal/provisioner/utility"
	"github.com/mattermost/mattermost-cloud/internal/store"
	"github.com/mattermost/mattermost-cloud/internal/supervisor"
	"github.com/mattermost/mattermost-cloud/internal/tools/aws"
	"github.com/mattermost/mattermost-cloud/k8s"
	"github.com/mattermost/mattermost-cloud/model"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
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
	tempDir            string
}

var _ supervisor.ClusterProvisioner = (*EKSProvisioner)(nil)

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

// PrepareCluster prepares metadata for EKSProvisioner.
func (provisioner *EKSProvisioner) PrepareCluster(cluster *model.Cluster) bool {
	// Don't regenerate the name if already set.
	if cluster.ProvisionerMetadataEKS.Name != "" {
		return false
	}

	// Generate the EKS name using the cluster id.
	cluster.ProvisionerMetadataEKS.Name = fmt.Sprintf("%s-eks-k8s-local", cluster.ID)

	return true
}

// CreateCluster creates the EKS cluster.
func (provisioner *EKSProvisioner) CreateCluster(cluster *model.Cluster) error {
	logger := provisioner.logger.WithField("cluster", cluster.ID)

	eksMetadata := cluster.ProvisionerMetadataEKS
	if eksMetadata == nil {
		return errors.New("EKS Metadata not set when creating EKS cluster")
	}

	err := eksMetadata.ValidateChangeRequest()
	if err != nil {
		return errors.Wrap(err, "EKS Metadata ChangeRequest failed validation")
	}

	if eksMetadata.ChangeRequest.AMI != "" && eksMetadata.ChangeRequest.AMI != "latest" {
		var isAMIValid bool
		isAMIValid, err = provisioner.awsClient.IsValidAMI(eksMetadata.ChangeRequest.AMI, logger)
		if err != nil {
			return errors.Wrapf(err, "error checking the AWS AMI image %s", eksMetadata.ChangeRequest.AMI)
		}
		if !isAMIValid {
			return errors.Errorf("invalid AWS AMI image %s", eksMetadata.ChangeRequest.AMI)
		}
	}

	// TODO: Add cncVPCCIDR like KOPS

	var clusterResources aws.ClusterResources
	if eksMetadata.ChangeRequest.VPC != "" && provisioner.params.UseExistingAWSResources {
		clusterResources, err = provisioner.awsClient.ClaimVPC(eksMetadata.ChangeRequest.VPC, cluster, provisioner.params.Owner, logger)
		if err != nil {
			return errors.Wrap(err, "couldn't claim VPC")
		}
	} else if provisioner.params.UseExistingAWSResources {
		clusterResources, err = provisioner.awsClient.GetAndClaimVpcResources(cluster, provisioner.params.Owner, logger)
		if err != nil {
			return err
		}
	}

	// Update cluster to set VPC ID that is needed later.
	cluster.ProvisionerMetadataEKS.ChangeRequest.VPC = clusterResources.VpcID

	_, err = provisioner.awsClient.EnsureEKSCluster(cluster, clusterResources)
	if err != nil {
		releaseErr := provisioner.awsClient.ReleaseVpc(cluster, logger)
		if releaseErr != nil {
			logger.WithError(releaseErr).Error("Unable to release VPC")
		}

		return errors.Wrap(err, "unable to create EKS cluster")
	}

	err = provisioner.clusterUpdateStore.UpdateCluster(cluster)
	if err != nil {
		releaseErr := provisioner.awsClient.ReleaseVpc(cluster, logger)
		if releaseErr != nil {
			logger.WithError(releaseErr).Error("Failed to release VPC after failed update")
		}
		return errors.Wrap(err, "failed to update EKS metadata with VPC ID")
	}

	return nil
}

// CheckClusterCreated checks if cluster creation finished.
func (provisioner *EKSProvisioner) CheckClusterCreated(cluster *model.Cluster) (bool, error) {
	logger := provisioner.logger.WithField("cluster", cluster.ID)

	if cluster.ProvisionerMetadataEKS == nil {
		return false, errors.New("expected EKS metadata not to be nil")
	}

	eksMetadata := cluster.ProvisionerMetadataEKS
	changeRequest := eksMetadata.ChangeRequest

	wait := 1200
	logger.Infof("Waiting up to %d seconds for EKS cluster to become active...", wait)
	eksCluster, err := provisioner.awsClient.WaitForActiveEKSCluster(eksMetadata.Name, wait)
	if err != nil {
		return false, err
	}

	if eksCluster.Version != nil {
		changeRequest.Version = *eksCluster.Version
	}

	eksMetadata.ClusterRoleARN = *eksCluster.RoleArn
	eksMetadata.VPC = changeRequest.VPC
	eksMetadata.Version = changeRequest.Version

	err = provisioner.clusterUpdateStore.UpdateCluster(cluster)
	if err != nil {
		return false, errors.Wrap(err, "failed to store cluster")
	}

	return true, nil
}

func (provisioner *EKSProvisioner) prepareLaunchTemplate(cluster *model.Cluster, ngPrefix string, withSG bool, logger log.FieldLogger) error {
	eksMetadata := cluster.ProvisionerMetadataEKS
	changeRequest := eksMetadata.ChangeRequest

	var err error

	var securityGroups []string
	if withSG {
		vpc := changeRequest.VPC
		if vpc == "" {
			vpc = eksMetadata.VPC
		}
		securityGroups, err = provisioner.awsClient.ClaimSecurityGroups(cluster, ngPrefix, vpc, logger)
		if err != nil {
			return errors.Wrap(err, "failed to get security groups")
		}
	}

	launchTemplateData := &model.LaunchTemplateData{
		Name:           fmt.Sprintf("%s-%s", eksMetadata.Name, ngPrefix),
		ClusterName:    eksMetadata.Name,
		SecurityGroups: securityGroups,
		AMI:            changeRequest.AMI,
		MaxPodsPerNode: changeRequest.MaxPodsPerNode,
	}

	if launchTemplateData.AMI == "" {
		launchTemplateData.AMI = eksMetadata.AMI
	}
	if launchTemplateData.MaxPodsPerNode == 0 {
		launchTemplateData.MaxPodsPerNode = eksMetadata.MaxPodsPerNode
	}

	launchTemplateName := fmt.Sprintf("%s-%s", eksMetadata.Name, ngPrefix)
	isAvailable, err := provisioner.awsClient.IsLaunchTemplateAvailable(launchTemplateName)
	if err != nil {
		return errors.Wrap(err, "failed to check launch template availability")
	}

	if isAvailable {
		err = provisioner.awsClient.UpdateLaunchTemplate(launchTemplateData)
		if err != nil {
			return errors.Wrap(err, "failed to update launch template")
		}
	} else {
		err = provisioner.awsClient.CreateLaunchTemplate(launchTemplateData)
		if err != nil {
			return errors.Wrap(err, "failed to create launch template")
		}
	}

	return nil
}

// CreateNodegroups creates the EKS nodegroups.
func (provisioner *EKSProvisioner) CreateNodegroups(cluster *model.Cluster) error {
	logger := provisioner.logger.WithField("cluster", cluster.ID)

	eksMetadata := cluster.ProvisionerMetadataEKS
	if eksMetadata == nil {
		return errors.New("EKS Metadata not set when creating EKS NodeGroup")
	}

	eksMetadata = provisioner.setMissingChangeRequest(eksMetadata)

	changeRequest := eksMetadata.ChangeRequest

	var wg sync.WaitGroup
	var errOccurred bool

	for ng, meta := range changeRequest.NodeGroups {
		wg.Add(1)
		go func(ngPrefix string, ngMetadata model.NodeGroupMetadata) {
			defer wg.Done()
			logger.Debugf("Creating EKS NodeGroup %s", ngMetadata.Name)

			err := provisioner.prepareLaunchTemplate(cluster, ngPrefix, ngMetadata.WithSecurityGroup, logger)
			if err != nil {
				logger.WithError(err).Error("failed to ensure launch template")
				errOccurred = true
				return
			}

			_, err = provisioner.awsClient.EnsureEKSNodeGroup(cluster, ngPrefix)
			if err != nil {
				logger.WithError(err).Errorf("failed to create EKS NodeGroup %s", ngMetadata.Name)
				errOccurred = true
				return
			}
		}(ng, meta)
	}

	wg.Wait()

	if errOccurred {
		return errors.New("failed to create one of the EKS NodeGroups")
	}

	err := provisioner.clusterUpdateStore.UpdateCluster(cluster)
	if err != nil {
		return errors.Wrap(err, "failed to store cluster")
	}

	return nil
}

// CheckNodegroupsCreated checks if the EKS nodegroups are created.
func (provisioner *EKSProvisioner) CheckNodegroupsCreated(cluster *model.Cluster) (bool, error) {
	logger := provisioner.logger.WithField("cluster", cluster.ID)

	eksMetadata := cluster.ProvisionerMetadataEKS
	changeRequest := eksMetadata.ChangeRequest

	if eksMetadata.NodeGroups == nil {
		eksMetadata.NodeGroups = make(map[string]model.NodeGroupMetadata)
	}

	var wg sync.WaitGroup
	var errOccurred bool

	for ng, meta := range changeRequest.NodeGroups {
		wg.Add(1)
		go func(ngPrefix string, ngMetadata model.NodeGroupMetadata) {
			defer wg.Done()

			wait := 300
			logger.Infof("Waiting up to %d seconds for EKS NodeGroup %s to become active...", wait, ngMetadata.Name)

			nodeGroup, err := provisioner.awsClient.WaitForActiveEKSNodeGroup(eksMetadata.Name, ngMetadata.Name, wait)
			if err != nil {
				logger.WithError(err).Errorf("failed to wait for EKS NodeGroup %s to become active", ngMetadata.Name)
				errOccurred = true
				return
			}

			eksMetadata.NodeGroups[ngPrefix] = model.NodeGroupMetadata{
				Name:              ngMetadata.Name,
				Type:              ngMetadata.Type,
				InstanceType:      nodeGroup.InstanceTypes[0],
				MinCount:          int64(ptr.ToInt32(nodeGroup.ScalingConfig.MinSize)),
				MaxCount:          int64(ptr.ToInt32(nodeGroup.ScalingConfig.MaxSize)),
				WithPublicSubnet:  ngMetadata.WithPublicSubnet,
				WithSecurityGroup: ngMetadata.WithSecurityGroup,
			}
		}(ng, meta)
	}

	wg.Wait()

	logger.Debugf("All EKS NodeGroups are active")

	if errOccurred {
		return false, errors.New("one of the EKS NodeGroups failed to become active")
	}

	err := provisioner.awsClient.InstallEKSAddons(cluster)
	if err != nil {
		return false, errors.Wrap(err, "failed to install EKS EBS Addon")
	}

	eksMetadata.NodeRoleARN = changeRequest.NodeRoleARN
	eksMetadata.AMI = changeRequest.AMI
	eksMetadata.MaxPodsPerNode = changeRequest.MaxPodsPerNode
	eksMetadata.Networking = model.NetworkingVpcCni

	err = provisioner.clusterUpdateStore.UpdateCluster(cluster)
	if err != nil {
		return false, errors.Wrap(err, "failed to store cluster")
	}

	return true, nil
}

// DeleteNodegroups deletes the EKS nodegroup.
func (provisioner *EKSProvisioner) DeleteNodegroups(cluster *model.Cluster) error {
	logger := provisioner.logger.WithField("cluster", cluster.ID)

	eksMetadata := cluster.ProvisionerMetadataEKS
	if eksMetadata == nil {
		return errors.New("EKS Metadata not set when deleting EKS NodeGroup")
	}

	changeRequest := eksMetadata.ChangeRequest
	if changeRequest == nil || changeRequest.NodeGroups == nil {
		return errors.New("nodegroup change request not set when deleting EKS NodeGroup")
	}

	nodeGroups := changeRequest.NodeGroups

	var wg sync.WaitGroup
	var errOccurred bool

	for ng, meta := range nodeGroups {
		wg.Add(1)
		go func(ngPrefix string, ngMetadata model.NodeGroupMetadata) {
			defer wg.Done()

			err := provisioner.awsClient.EnsureEKSNodeGroupDeleted(eksMetadata.Name, ngMetadata.Name)
			if err != nil {
				logger.WithError(err).Errorf("failed to delete EKS NodeGroup %s", ngMetadata.Name)
				errOccurred = true
				return
			}

			wait := 600
			logger.Infof("Waiting up to %d seconds for NodeGroup %s to be deleted...", wait, ngMetadata.Name)
			err = provisioner.awsClient.WaitForEKSNodeGroupToBeDeleted(eksMetadata.Name, ngMetadata.Name, wait)
			if err != nil {
				logger.WithError(err).Errorf("failed to delete EKS NodeGroup %s", ngMetadata.Name)
				errOccurred = true
				return
			}

			launchTemplateName := fmt.Sprintf("%s-%s", eksMetadata.Name, ngPrefix)
			err = provisioner.awsClient.DeleteLaunchTemplate(launchTemplateName)
			if err != nil {
				logger.WithError(err).Errorf("failed to delete EKS LaunchTemplate %s", launchTemplateName)
				errOccurred = true
				return
			}

			logger.Debugf("Successfully deleted EKS NodeGroup %s", ngMetadata.Name)
		}(ng, meta)
	}

	wg.Wait()

	if errOccurred {
		return errors.New("failed to delete one of the nodegroups")
	}

	for ng := range nodeGroups {
		delete(eksMetadata.NodeGroups, ng)
	}

	return nil
}

// ProvisionCluster provisions EKS cluster.
func (provisioner *EKSProvisioner) ProvisionCluster(cluster *model.Cluster) error {
	logger := provisioner.logger.WithField("cluster", cluster.ID)

	eksMetadata := cluster.ProvisionerMetadataEKS
	if eksMetadata == nil {
		return errors.New("expected EKS metadata not to be nil when using EKS Provisioner")
	}

	kubeConfigPath, err := provisioner.getKubeConfigPath(cluster)
	if err != nil {
		return errors.Wrap(err, "failed to get kubeconfig file path")
	}

	//nil argocdClient and gitClient because EKSProvisioner isn't in use.
	return provisionCluster(cluster, kubeConfigPath, provisioner.tempDir, provisioner.awsClient, nil, nil, provisioner.params, provisioner.store, logger)
}

// UpgradeCluster upgrades EKS cluster - not implemented.
func (provisioner *EKSProvisioner) UpgradeCluster(cluster *model.Cluster) error {
	logger := provisioner.logger.WithField("cluster", cluster.ID)

	eksMetadata := cluster.ProvisionerMetadataEKS
	changeRequest := eksMetadata.ChangeRequest

	err := eksMetadata.ValidateChangeRequest()
	if err != nil {
		return errors.Wrap(err, "eks Metadata ChangeRequest failed validation")
	}

	if changeRequest.AMI != "" && changeRequest.AMI != "latest" {
		var isAMIValid bool
		isAMIValid, err = provisioner.awsClient.IsValidAMI(eksMetadata.ChangeRequest.AMI, logger)
		if err != nil {
			return errors.Wrapf(err, "error checking the AWS ami image %s", eksMetadata.ChangeRequest.AMI)
		}
		if !isAMIValid {
			return errors.Errorf("invalid AWS ami image %s", eksMetadata.ChangeRequest.AMI)
		}
	}

	if changeRequest.AMI != "" || changeRequest.MaxPodsPerNode > 0 {

		var wg sync.WaitGroup
		var errOccurred bool

		for ng, meta := range changeRequest.NodeGroups {
			wg.Add(1)
			go func(ngPrefix string, ngMetadata model.NodeGroupMetadata) {
				defer wg.Done()
				logger.Debugf("Migrating EKS NodeGroup for %s", ngPrefix)

				oldMetadata := eksMetadata.NodeGroups[ngPrefix]
				ngMetadata.CopyMissingFieldsFrom(oldMetadata)
				changeRequest.NodeGroups[ngPrefix] = ngMetadata

				err = provisioner.prepareLaunchTemplate(cluster, ngPrefix, ngMetadata.WithSecurityGroup, logger)
				if err != nil {
					logger.WithError(err).Error("failed to ensure launch template")
					errOccurred = true
					return
				}

				err = provisioner.awsClient.EnsureEKSNodeGroupMigrated(cluster, ngPrefix)
				if err != nil {
					logger.WithError(err).Errorf("failed to migrate EKS NodeGroup for %s", ngPrefix)
					errOccurred = true
					return
				}
				oldMetadata = eksMetadata.NodeGroups[ngPrefix]
				oldMetadata.Name = ngMetadata.Name
				eksMetadata.NodeGroups[ngPrefix] = oldMetadata

				logger.Debugf("Successfully migrated EKS NodeGroup for %s", ngPrefix)
			}(ng, meta)
		}

		wg.Wait()

		if errOccurred {
			return errors.New("failed to migrate one of the EKS NodeGroups")
		}

		if changeRequest.AMI != "" {
			eksMetadata.AMI = changeRequest.AMI
		}
		if changeRequest.MaxPodsPerNode > 0 {
			eksMetadata.MaxPodsPerNode = changeRequest.MaxPodsPerNode
		}

		err = provisioner.clusterUpdateStore.UpdateCluster(cluster)
		if err != nil {
			return errors.Wrap(err, "failed to store cluster")
		}
	}

	if changeRequest.Version != "" {
		clusterUpdateRequest, err := provisioner.awsClient.EnsureEKSClusterUpdated(cluster)
		if err != nil {
			return errors.Wrap(err, "failed to update EKS cluster")
		}

		if clusterUpdateRequest != nil && clusterUpdateRequest.Id != nil {
			wait := 3600 // seconds
			logger.Infof("Waiting up to %d seconds for EKS cluster to be updated...", wait)
			err = provisioner.awsClient.WaitForEKSClusterUpdateToBeCompleted(eksMetadata.Name, *clusterUpdateRequest.Id, wait)
			if err != nil {
				return errors.Wrap(err, "failed to update EKS cluster")
			}
		}

		eksMetadata.Version = changeRequest.Version

		err = provisioner.clusterUpdateStore.UpdateCluster(cluster)
		if err != nil {
			return errors.Wrap(err, "failed to store cluster")
		}
	}

	return nil
}

// RotateClusterNodes rotates cluster nodes - not implemented.
func (provisioner *EKSProvisioner) RotateClusterNodes(cluster *model.Cluster) error {
	return nil
}

func (provisioner *EKSProvisioner) isMigrationRequired(oldNodeGroup *eksTypes.Nodegroup, eksMetadata *model.EKSMetadata, ngPrefix string, logger log.FieldLogger) bool {
	changeRequest := eksMetadata.ChangeRequest

	if oldNodeGroup == nil {
		logger.Debug("Old EKS NodeGroup is nil")
		return false
	}

	ngChangeRequest, found := changeRequest.NodeGroups[ngPrefix]
	if !found {
		logger.Debugf("NodeGroup %s not found in ChangeRequest", ngPrefix)
		return false
	}

	if changeRequest.MaxPodsPerNode > 0 || changeRequest.AMI != "" {
		return true
	}

	if len(oldNodeGroup.InstanceTypes) > 0 && oldNodeGroup.InstanceTypes[0] != ngChangeRequest.InstanceType {
		return true
	}

	scalingInfo := oldNodeGroup.ScalingConfig
	if *scalingInfo.MinSize != int32(ngChangeRequest.MinCount) {
		return true
	}
	if *scalingInfo.MaxSize != int32(ngChangeRequest.MaxCount) {
		return true
	}

	return false
}

// ResizeCluster resizes cluster - not implemented.
func (provisioner *EKSProvisioner) ResizeCluster(cluster *model.Cluster) error {
	logger := provisioner.logger.WithField("cluster", cluster.ID)

	eksMetadata := cluster.ProvisionerMetadataEKS
	changeRequest := eksMetadata.ChangeRequest

	err := eksMetadata.ValidateChangeRequest()
	if err != nil {
		return errors.Wrap(err, "eks Metadata ChangeRequest failed validation")
	}

	var wg sync.WaitGroup
	var errOccurred bool

	for ng, meta := range changeRequest.NodeGroups {
		wg.Add(1)
		go func(ngPrefix string, ngMetadata model.NodeGroupMetadata) {
			defer wg.Done()
			logger.Debugf("Migrating EKS NodeGroup for %s", ngPrefix)

			oldMetadata := eksMetadata.NodeGroups[ngPrefix]
			ngMetadata.CopyMissingFieldsFrom(oldMetadata)
			changeRequest.NodeGroups[ngPrefix] = ngMetadata

			oldEKSNodeGroup, err2 := provisioner.awsClient.GetActiveEKSNodeGroup(eksMetadata.Name, oldMetadata.Name)
			if err2 != nil {
				logger.WithError(err2).Errorf("failed to get EKS NodeGroup for %s", ngPrefix)
				errOccurred = true
				return
			}

			if oldEKSNodeGroup == nil || oldEKSNodeGroup.Status != eksTypes.NodegroupStatusActive {
				logger.Debugf("No active EKS NodeGroup found for %s", ngPrefix)
				return
			}

			if !provisioner.isMigrationRequired(oldEKSNodeGroup, eksMetadata, ngPrefix, logger) {
				logger.Debugf("EKS NodeGroup migration not required for %s", ngPrefix)
				return
			}

			err = provisioner.awsClient.EnsureEKSNodeGroupMigrated(cluster, ngPrefix)
			if err != nil {
				logger.WithError(err).Errorf("failed to migrate EKS NodeGroup for %s", ngPrefix)
				errOccurred = true
				return
			}

			nodeGroup, err2 := provisioner.awsClient.GetActiveEKSNodeGroup(eksMetadata.Name, ngMetadata.Name)
			if err2 != nil {
				logger.WithError(err2).Errorf("failed to get EKS NodeGroup %s", ngMetadata.Name)
				errOccurred = true
				return
			}

			if nodeGroup == nil {
				logger.WithError(err2).Errorf("EKS NodeGroup %s not found", ngMetadata.Name)
				errOccurred = true
				return
			}

			oldNodeGroup := eksMetadata.NodeGroups[ngPrefix]
			oldNodeGroup.Name = ngMetadata.Name
			oldNodeGroup.InstanceType = nodeGroup.InstanceTypes[0]
			oldNodeGroup.MinCount = int64(ptr.ToInt32(nodeGroup.ScalingConfig.MinSize))
			oldNodeGroup.MaxCount = int64(ptr.ToInt32(nodeGroup.ScalingConfig.MaxSize))
			eksMetadata.NodeGroups[ngPrefix] = oldNodeGroup

			logger.Debugf("Successfully migrated EKS NodeGroup for %s", ngPrefix)
		}(ng, meta)
	}

	wg.Wait()

	if errOccurred {
		return errors.New("failed to migrate one of the nodegroups")
	}

	err = provisioner.clusterUpdateStore.UpdateCluster(cluster)
	if err != nil {
		return errors.Wrap(err, "failed to store cluster")
	}

	return nil
}

func (provisioner *EKSProvisioner) cleanupCluster(cluster *model.Cluster) error {
	logger := provisioner.logger.WithField("cluster", cluster.ID)

	kubeConfigPath, err := provisioner.getKubeConfigPath(cluster)
	if err != nil {
		return errors.Wrap(err, "failed to get kubeconfig file path")
	}

	//nil argocdClient and gitClient because EKSProvisioner isn't in use.
	ugh, err := utility.NewUtilityGroupHandle(provisioner.params.AllowCIDRRangeList, kubeConfigPath, provisioner.tempDir, cluster, provisioner.awsClient, nil, nil, logger)
	if err != nil {
		return errors.Wrap(err, "couldn't create new utility group handle while deleting the cluster")
	}

	err = ugh.DestroyUtilityGroup()
	if err != nil {
		return errors.Wrap(err, "failed to destroy all services in the utility group")
	}

	eksMetadata := cluster.ProvisionerMetadataEKS

	var wg sync.WaitGroup
	var errOccurred bool

	nodeGroups := eksMetadata.NodeGroups
	if len(nodeGroups) == 0 && eksMetadata.ChangeRequest != nil {
		nodeGroups = make(map[string]model.NodeGroupMetadata)
		for ng, meta := range eksMetadata.ChangeRequest.NodeGroups {
			nodeGroups[ng] = meta
		}
	}

	for ng, meta := range nodeGroups {
		wg.Add(1)
		go func(ngPrefix string, ngMetadata model.NodeGroupMetadata) {
			defer wg.Done()

			err = provisioner.awsClient.EnsureEKSNodeGroupDeleted(eksMetadata.Name, ngMetadata.Name)
			if err != nil {
				logger.WithError(err).Errorf("failed to delete EKS NodeGroup %s", ngMetadata.Name)
				errOccurred = true
				return
			}

			wait := 600
			logger.Infof("Waiting up to %d seconds for NodeGroup %s to be deleted...", wait, ngMetadata.Name)
			err = provisioner.awsClient.WaitForEKSNodeGroupToBeDeleted(eksMetadata.Name, ngMetadata.Name, wait)
			if err != nil {
				logger.WithError(err).Errorf("failed to delete EKS NodeGroup %s", ngMetadata.Name)
				errOccurred = true
				return
			}

			launchTemplateName := fmt.Sprintf("%s-%s", eksMetadata.Name, ngPrefix)
			err = provisioner.awsClient.DeleteLaunchTemplate(launchTemplateName)
			if err != nil {
				logger.WithError(err).Errorf("failed to delete EKS LaunchTemplate %s", launchTemplateName)
				errOccurred = true
				return
			}

			logger.Debugf("Successfully deleted EKS NodeGroup %s", ngMetadata.Name)
		}(ng, meta)
	}

	wg.Wait()

	if errOccurred {
		return errors.New("failed to delete one of the nodegroups")
	}

	err = provisioner.awsClient.EnsureEKSClusterDeleted(eksMetadata.Name)
	if err != nil {
		return errors.Wrap(err, "failed to delete EKS cluster")
	}

	wait := 1200
	logger.Infof("Waiting up to %d seconds for EKS cluster to be deleted...", wait)
	err = provisioner.awsClient.WaitForEKSClusterToBeDeleted(eksMetadata.Name, wait)
	if err != nil {
		return err
	}

	return nil

}

func (provisioner *EKSProvisioner) setMissingChangeRequest(eksMetadata *model.EKSMetadata) *model.EKSMetadata {
	changeRequest := eksMetadata.ChangeRequest

	if changeRequest.AMI == "" {
		changeRequest.AMI = eksMetadata.AMI
	}

	if changeRequest.MaxPodsPerNode == 0 {
		changeRequest.MaxPodsPerNode = eksMetadata.MaxPodsPerNode
	}

	if changeRequest.VPC == "" {
		changeRequest.VPC = eksMetadata.VPC
	}

	if changeRequest.NodeRoleARN == "" {
		changeRequest.NodeRoleARN = eksMetadata.NodeRoleARN
	}

	return eksMetadata
}

// DeleteCluster deletes EKS cluster.
func (provisioner *EKSProvisioner) DeleteCluster(cluster *model.Cluster) (bool, error) {
	logger := provisioner.logger.WithField("cluster", cluster.ID)

	logger.Info("Deleting cluster")

	eksMetadata := cluster.ProvisionerMetadataEKS
	if eksMetadata == nil {
		return false, errors.New("expected EKS metadata not to be nil when using EKS Provisioner")
	}

	eksCluster, err := provisioner.awsClient.GetActiveEKSCluster(eksMetadata.Name)
	if err != nil {
		return false, errors.Wrap(err, "failed to get EKS cluster")
	}

	if eksCluster == nil {
		logger.Infof("EKS cluster %s does not exist, assuming already deleted", eksMetadata.Name)
	} else {
		err = provisioner.cleanupCluster(cluster)
		if err != nil {
			return false, errors.Wrap(err, "failed to cleanup EKS cluster")
		}
	}

	err = provisioner.awsClient.ReleaseVpc(cluster, logger)
	if err != nil {
		return false, errors.Wrap(err, "failed to release cluster VPC")
	}

	logger.Info("Successfully deleted EKS cluster")

	return true, nil
}

func (provisioner *EKSProvisioner) RefreshClusterMetadata(cluster *model.Cluster) error {
	if cluster.ProvisionerMetadataEKS != nil {
		cluster.ProvisionerMetadataEKS.ApplyChangeRequest()
		cluster.ProvisionerMetadataEKS.ClearChangeRequest()
		cluster.ProvisionerMetadataEKS.ClearWarnings()
	}
	return nil
}

func (provisioner *EKSProvisioner) getKubeConfigPath(cluster *model.Cluster) (string, error) {
	clusterName := cluster.ProvisionerMetadataEKS.Name
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

// newEKSKubeConfig creates kubeconfig for EKS cluster.
func newEKSKubeConfig(cluster *eksTypes.Cluster, aws aws.AWS) (clientcmdapi.Config, error) {
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
