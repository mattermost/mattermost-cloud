// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.
//

package aws

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	"github.com/aws/aws-sdk-go-v2/service/eks"
	eksTypes "github.com/aws/aws-sdk-go-v2/service/eks/types"
	"github.com/aws/smithy-go/ptr"
	"github.com/mattermost/mattermost-cloud/internal/common"
	"github.com/mattermost/mattermost-cloud/model"
	"github.com/pkg/errors"
)

// createEKSCluster creates EKS cluster.
func (c *Client) createEKSCluster(cluster *model.Cluster, resources ClusterResources) (*eksTypes.Cluster, error) {
	ctx := context.TODO()

	// TODO: we do not expect to query that many subnets but for safety
	// we can check the NextToken.
	subnetsOut, err := c.Service().ec2.DescribeSubnets(ctx, &ec2.DescribeSubnetsInput{
		// TODO: is it public/private
		SubnetIds: resources.PublicSubnetsIDs,
	})
	if err != nil {
		return nil, errors.Wrap(err, "failed to describe subnets")
	}

	var zones []string
	if cluster.ProviderMetadataAWS != nil {
		zones = cluster.ProviderMetadataAWS.Zones
	}
	var subnetsIDs []string
	for _, sub := range subnetsOut.Subnets {
		// us-east-1e does not currently have sufficient capacity to support the cluster
		if *sub.AvailabilityZone == "us-east-1e" {
			continue
		}
		if len(zones) > 0 {
			if common.Contains(zones, *sub.AvailabilityZone) {
				subnetsIDs = append(subnetsIDs, *sub.SubnetId)
			}
		} else {
			subnetsIDs = append(subnetsIDs, *sub.SubnetId)
		}
	}

	vpcConfig := eksTypes.VpcConfigRequest{
		EndpointPrivateAccess: ptr.Bool(true),
		EndpointPublicAccess:  ptr.Bool(true),
		SecurityGroupIds:      resources.MasterSecurityGroupIDs,
		SubnetIds:             subnetsIDs,
	}

	eksMetadata := cluster.ProvisionerMetadataEKS
	// TODO: we can allow further parametrization in the future
	input := eks.CreateClusterInput{
		Name:               aws.String(eksMetadata.Name),
		ResourcesVpcConfig: &vpcConfig,
		RoleArn:            &eksMetadata.ChangeRequest.ClusterRoleARN,
	}
	if eksMetadata.ChangeRequest.Version != "" {
		input.Version = &eksMetadata.ChangeRequest.Version
	}

	out, err := c.Service().eks.CreateCluster(ctx, &input)
	if err != nil {
		return nil, errors.Wrap(err, "failed to create EKS cluster")
	}

	return out.Cluster, nil
}

func (a *Client) getEKSCluster(clusterName string) (*eksTypes.Cluster, error) {

	output, err := a.Service().eks.DescribeCluster(context.TODO(), &eks.DescribeClusterInput{
		Name: aws.String(clusterName),
	})
	if err != nil {
		if !IsErrorResourceNotFound(err) {
			return nil, errors.Wrap(err, "failed to describe cluster")
		}
	}

	if output != nil && output.Cluster != nil {
		return output.Cluster, nil
	}

	return nil, nil
}

// EnsureEKSCluster ensures EKS cluster is created.
func (c *Client) EnsureEKSCluster(cluster *model.Cluster, resources ClusterResources) (*eksTypes.Cluster, error) {
	clusterName := cluster.ProvisionerMetadataEKS.Name
	eksCluster, err := c.getEKSCluster(clusterName)
	if err != nil {
		return nil, err
	}

	if eksCluster != nil {
		return eksCluster, nil
	}

	return c.createEKSCluster(cluster, resources)
}

// InstallEKSAddons installs EKS EBS addon to the existing cluster.
func (a *Client) InstallEKSAddons(cluster *model.Cluster) error {
	input := eks.CreateAddonInput{
		AddonName:   aws.String("aws-ebs-csi-driver"),
		ClusterName: aws.String(cluster.ProvisionerMetadataEKS.Name),
	}
	_, err := a.Service().eks.CreateAddon(context.TODO(), &input)
	if err != nil {
		// In case addon already configured we do not want to fail.
		if IsErrorResourceInUseException(err) {
			return nil
		}
		return errors.Wrap(err, "failed to create ebs-csi addon")
	}

	return nil
}

func (c *Client) EnsureEKSClusterUpdated(cluster *model.Cluster) error {
	clusterName := cluster.ProvisionerMetadataEKS.Name
	eksCluster, err := c.getEKSCluster(clusterName)
	if err != nil {
		return err
	}

	if eksCluster == nil {
		return errors.Errorf("cluster %s does not exist", clusterName)
	}

	if eksCluster.Status != eksTypes.ClusterStatusActive {
		return errors.Errorf("cluster %s is not active", clusterName)
	}

	eksMetadata := cluster.ProvisionerMetadataEKS
	if eksMetadata.ChangeRequest.Version == "" {
		return nil
	}

	_, err = c.Service().eks.UpdateClusterVersion(context.TODO(), &eks.UpdateClusterVersionInput{
		Name:    aws.String(clusterName),
		Version: aws.String(eksMetadata.ChangeRequest.Version),
	})

	if err != nil {
		return errors.Wrap(err, "failed to update EKS cluster version")
	}

	return nil
}

func (a *Client) createEKSNodeGroup(cluster *model.Cluster) (*eksTypes.Nodegroup, error) {

	clusterName := cluster.ProvisionerMetadataEKS.Name
	eksMetadata := cluster.ProvisionerMetadataEKS
	changeRequest := eksMetadata.ChangeRequest
	if changeRequest == nil {
		return nil, errors.New("change request is nil")
	}

	workerName := cluster.ProvisionerMetadataEKS.ChangeRequest.WorkerName

	clusterResource, err := a.GetVpcResourcesByVpcID(changeRequest.VPC, a.logger)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get VPC resources")
	}

	launchTemplate := getLaunchTemplateName(clusterName)

	subnetsOut, err := a.Service().ec2.DescribeSubnets(context.TODO(), &ec2.DescribeSubnetsInput{
		SubnetIds: clusterResource.PrivateSubnetIDs,
	})
	if err != nil {
		return nil, errors.Wrap(err, "failed to describe subnets")
	}

	var zones []string
	if cluster.ProviderMetadataAWS != nil {
		zones = cluster.ProviderMetadataAWS.Zones
	}
	var subnetsIDs []string
	for _, sub := range subnetsOut.Subnets {
		// us-east-1e does not currently have sufficient capacity to support the cluster
		if *sub.AvailabilityZone == "us-east-1e" {
			continue
		}
		if len(zones) > 0 {
			if common.Contains(zones, *sub.AvailabilityZone) {
				subnetsIDs = append(subnetsIDs, *sub.SubnetId)
			}
		} else {
			subnetsIDs = append(subnetsIDs, *sub.SubnetId)
		}
	}

	nodeGroupReq := eks.CreateNodegroupInput{
		ClusterName:   aws.String(clusterName),
		InstanceTypes: []string{changeRequest.NodeInstanceType},
		NodeRole:      &changeRequest.NodeRoleARN,
		NodegroupName: aws.String(workerName),
		AmiType:       eksTypes.AMITypesCustom,
		ScalingConfig: &eksTypes.NodegroupScalingConfig{
			DesiredSize: ptr.Int32(int32(changeRequest.NodeMinCount)),
			MaxSize:     ptr.Int32(int32(changeRequest.NodeMaxCount)),
			MinSize:     ptr.Int32(int32(changeRequest.NodeMinCount)),
		},
		Subnets: subnetsIDs,
		LaunchTemplate: &eksTypes.LaunchTemplateSpecification{
			Name:    aws.String(launchTemplate),
			Version: aws.String(fmt.Sprintf("%d", *changeRequest.LaunchTemplateVersion)),
		},
		Tags: map[string]string{
			fmt.Sprintf("kubernetes.io/cluster/%s", clusterName): "owned",
		},
	}

	out, err := a.Service().eks.CreateNodegroup(context.TODO(), &nodeGroupReq)
	if err != nil {
		return nil, errors.Wrap(err, "failed to create EKS NodeGroup")
	}

	return out.Nodegroup, nil
}

func (c *Client) getEKSNodeGroup(clusterName, workerName string) (*eksTypes.Nodegroup, error) {

	output, err := c.Service().eks.DescribeNodegroup(context.TODO(), &eks.DescribeNodegroupInput{
		ClusterName:   aws.String(clusterName),
		NodegroupName: aws.String(workerName),
	})
	if err != nil {
		if !IsErrorResourceNotFound(err) {
			return nil, errors.Wrap(err, "failed to describe EKS NodeGroup")
		}
	}

	if output != nil && output.Nodegroup != nil {
		return output.Nodegroup, nil
	}

	return nil, nil
}

// EnsureEKSNodeGroup ensures EKS cluster node groups are created.
func (c *Client) EnsureEKSNodeGroup(cluster *model.Cluster) (*eksTypes.Nodegroup, error) {

	clusterName := cluster.ProvisionerMetadataEKS.Name
	workerName := cluster.ProvisionerMetadataEKS.ChangeRequest.WorkerName
	existingNodeGroup, err := c.getEKSNodeGroup(clusterName, workerName)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get existing EKS NodeGroup")
	}

	if existingNodeGroup != nil {
		return existingNodeGroup, nil
	}

	return c.createEKSNodeGroup(cluster)
}

// EnsureEKSNodeGroupMigrated updates EKS cluster node group.
func (c *Client) EnsureEKSNodeGroupMigrated(cluster *model.Cluster) error {
	logger := c.logger.WithField("cluster", cluster.ID)

	eksMetadata := cluster.ProvisionerMetadataEKS
	changeRequest := eksMetadata.ChangeRequest
	clusterName := eksMetadata.Name

	if eksMetadata.ChangeRequest == nil {
		return errors.New("change request is nil")
	}

	workerName := eksMetadata.WorkerName

	if workerName == "" {
		return errors.New("nodegroup name is missing")
	}

	nodeGroup, err := c.getEKSNodeGroup(clusterName, workerName)
	if err != nil {
		return errors.Wrap(err, "failed to describe EKS NodeGroup")
	}

	if nodeGroup == nil {
		return errors.New("EKS NodeGroup does not exist")
	}

	if nodeGroup.Status != eksTypes.NodegroupStatusActive {
		return errors.Errorf("EKS NodeGroup %s is not active", workerName)
	}

	var isUpdateRequired bool
	if changeRequest.LaunchTemplateVersion != nil {
		if nodeGroup.LaunchTemplate != nil && nodeGroup.LaunchTemplate.Version != nil &&
			*nodeGroup.LaunchTemplate.Version != fmt.Sprintf("%d", *changeRequest.LaunchTemplateVersion) {
			isUpdateRequired = true
		}
	}

	if !isUpdateRequired {
		return nil
	}

	logger.Info("creating an EKS NodeGroup")

	workerSeq := strings.TrimPrefix(workerName, "worker-")
	workerSeqInt, err := strconv.Atoi(workerSeq)
	if err != nil {
		return errors.Wrap(err, "failed to convert worker sequence to int")
	}

	eksMetadata.ChangeRequest.WorkerName = fmt.Sprintf("worker-%d", workerSeqInt+1)

	eksMetadata.ChangeRequest.VPC = eksMetadata.VPC
	eksMetadata.ChangeRequest.NodeInstanceType = eksMetadata.NodeInstanceType
	eksMetadata.ChangeRequest.NodeMinCount = eksMetadata.NodeMinCount
	eksMetadata.ChangeRequest.NodeMaxCount = eksMetadata.NodeMaxCount
	eksMetadata.ChangeRequest.NodeRoleARN = eksMetadata.NodeRoleARN

	nodeGroup, err = c.createEKSNodeGroup(cluster)
	if err != nil {
		return errors.Wrap(err, "failed to create a new NodeGroup")
	}

	eksMetadata.NodeInstanceGroups[*nodeGroup.NodegroupName] = model.EKSInstanceGroupMetadata{
		NodeInstanceType: nodeGroup.InstanceTypes[0],
		NodeMinCount:     int64(ptr.ToInt32(nodeGroup.ScalingConfig.MinSize)),
		NodeMaxCount:     int64(ptr.ToInt32(nodeGroup.ScalingConfig.MaxSize)),
	}

	wait := 600 // seconds
	logger.Infof("Waiting up to %d seconds for EKS NodeGroup to become active...", wait)
	_, err = c.WaitForActiveEKSNodeGroup(eksMetadata.Name, eksMetadata.ChangeRequest.WorkerName, wait)
	if err != nil {
		return err
	}

	logger.Info("deleting the old NodeGroup")

	err = c.EnsureEKSNodeGroupDeleted(clusterName, workerName)
	if err != nil {
		return errors.Wrap(err, "failed to delete the old NodeGroup")
	}

	delete(eksMetadata.NodeInstanceGroups, workerName)

	logger.Infof("Waiting up to %d seconds for NodeGroup to be deleted...", wait)
	err = c.WaitForEKSNodeGroupToBeDeleted(eksMetadata.Name, eksMetadata.WorkerName, wait)
	if err != nil {
		return err
	}

	eksMetadata.WorkerName = eksMetadata.ChangeRequest.WorkerName

	return nil
}

// EnsureEKSClusterDeleted ensures EKS cluster is deleted.
func (a *Client) EnsureEKSClusterDeleted(clusterName string) error {
	ctx := context.TODO()

	eksCluster, err := a.getEKSCluster(clusterName)
	if err != nil {
		return errors.Wrap(err, "failed to describe EKS cluster")
	}

	if eksCluster == nil {
		return nil
	}

	// Still deleting
	if eksCluster.Status == eksTypes.ClusterStatusDeleting {
		return nil
	}

	if eksCluster.Status == eksTypes.ClusterStatusFailed {
		return errors.New("cluster is in failed state")
	}

	delInput := &eks.DeleteClusterInput{Name: aws.String(clusterName)}
	_, err = a.Service().eks.DeleteCluster(ctx, delInput)
	if err != nil {
		return errors.Wrap(err, "failed to trigger EKS cluster deletion")
	}

	// Cluster just started deletion
	return nil
}

// EnsureEKSNodeGroupDeleted ensures EKS node groups are deleted.
func (a *Client) EnsureEKSNodeGroupDeleted(clusterName, workerName string) error {
	if workerName == "" {
		return nil
	}

	nodeGroups, err := a.getEKSNodeGroup(clusterName, workerName)
	if err != nil {
		return errors.Wrap(err, "failed to get NodeGroup")
	}
	// Node groups deleted, we can return
	if nodeGroups == nil {
		return nil
	}

	if nodeGroups.Status == eksTypes.NodegroupStatusDeleting {
		return nil
	}

	if nodeGroups.Status == eksTypes.NodegroupStatusDeleteFailed {
		return errors.Wrapf(err, "node group deletion failed %q", *nodeGroups.NodegroupName)
	}

	_, err = a.Service().eks.DeleteNodegroup(context.TODO(), &eks.DeleteNodegroupInput{
		ClusterName:   aws.String(clusterName),
		NodegroupName: nodeGroups.NodegroupName,
	})
	if err != nil {
		return errors.Wrap(err, "failed to delete NodeGroup")
	}

	return nil
}

// GetActiveEKSCluster returns the EKS cluster if ready.
func (c *Client) GetActiveEKSCluster(clusterName string) (*eksTypes.Cluster, error) {
	cluster, err := c.getEKSCluster(clusterName)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get EKS cluster")
	}

	if cluster == nil {
		return nil, nil
	}

	if cluster.Status == eksTypes.ClusterStatusFailed {
		return nil, errors.New("cluster creation failed")
	}

	if cluster.Status == eksTypes.ClusterStatusActive {
		return cluster, nil
	}

	return nil, nil
}

// GetActiveEKSNodeGroup returns the EKS node group if active.
func (c *Client) GetActiveEKSNodeGroup(clusterName, workerName string) (*eksTypes.Nodegroup, error) {
	nodeGroup, err := c.getEKSNodeGroup(clusterName, workerName)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get NodeGroup")
	}

	if nodeGroup == nil {
		return nil, nil
	}

	if nodeGroup.Status == eksTypes.NodegroupStatusCreateFailed {
		return nil, errors.New("EKS NodeGroup creation failed")
	}

	if nodeGroup.Status == eksTypes.NodegroupStatusActive {
		return nodeGroup, nil
	}

	return nil, nil
}

// WaitForActiveEKSCluster waits for EKS cluster to be ready.
func (c *Client) WaitForActiveEKSCluster(clusterName string, timeout int) (*eksTypes.Cluster, error) {
	timer := time.NewTimer(time.Duration(timeout) * time.Second)
	defer timer.Stop()

	for {
		select {
		case <-timer.C:
			return nil, errors.New("timed out waiting for EKS cluster to become active")
		default:
			eksCluster, err := c.GetActiveEKSCluster(clusterName)
			if err != nil {
				return nil, errors.Wrap(err, "failed to check if EKS cluster is active")
			}
			if eksCluster != nil {
				return eksCluster, nil
			}

			time.Sleep(5 * time.Second)
		}
	}
}

func (c *Client) WaitForActiveEKSNodeGroup(clusterName, workerName string, timeout int) (*eksTypes.Nodegroup, error) {
	timer := time.NewTimer(time.Duration(timeout) * time.Second)
	defer timer.Stop()

	for {
		select {
		case <-timer.C:
			return nil, errors.New("timed out waiting for EKS NodeGroup to become active")
		default:
			nodeGroup, err := c.GetActiveEKSNodeGroup(clusterName, workerName)
			if err != nil {
				return nil, errors.Wrap(err, "failed to check if EKS NodeGroup is active")
			}
			if nodeGroup != nil {
				return nodeGroup, nil
			}

			time.Sleep(5 * time.Second)
		}
	}
}

func (c *Client) WaitForEKSNodeGroupToBeDeleted(clusterName, workerName string, timeout int) error {
	timer := time.NewTimer(time.Duration(timeout) * time.Second)
	defer timer.Stop()

	for {
		select {
		case <-timer.C:
			return errors.New("timed out waiting for EKS NodeGroup to become ready")
		default:
			nodeGroup, err := c.getEKSNodeGroup(clusterName, workerName)
			if err != nil {
				return errors.Wrap(err, "failed to describe NodeGroup")
			}
			if nodeGroup == nil {
				return nil
			}

			time.Sleep(5 * time.Second)
		}
	}
}

func (c *Client) WaitForEKSClusterToBeDeleted(clusterName string, timeout int) error {
	timer := time.NewTimer(time.Duration(timeout) * time.Second)
	defer timer.Stop()

	for {
		select {
		case <-timer.C:
			return errors.New("timed out waiting for EKS cluster to become ready")
		default:
			eksCluster, err := c.getEKSCluster(clusterName)
			if err != nil {
				return errors.Wrap(err, "failed to describe EKS cluster")
			}
			if eksCluster == nil {
				return nil
			}

			time.Sleep(5 * time.Second)
		}
	}
}
