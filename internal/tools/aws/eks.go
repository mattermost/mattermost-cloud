// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.
//

package aws

import (
	"context"
	"fmt"
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
		SubnetIds: resources.PrivateSubnetIDs,
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
			return nil, errors.Wrap(err, "failed to describe EKS cluster")
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

	var addons = []eks.CreateAddonInput{
		{
			AddonName:        aws.String("coredns"),
			AddonVersion:     aws.String("v1.8.7-eksbuild.1"),
			ClusterName:      aws.String(cluster.ProvisionerMetadataEKS.Name),
			ResolveConflicts: eksTypes.ResolveConflictsOverwrite,
		},
		{
			AddonName:        aws.String("kube-proxy"),
			AddonVersion:     aws.String("v1.22.6-eksbuild.1"),
			ClusterName:      aws.String(cluster.ProvisionerMetadataEKS.Name),
			ResolveConflicts: eksTypes.ResolveConflictsOverwrite,
		},
		{
			AddonName:           aws.String("vpc-cni"),
			AddonVersion:        aws.String("v1.11.0-eksbuild.1"),
			ClusterName:         aws.String(cluster.ProvisionerMetadataEKS.Name),
			ConfigurationValues: aws.String("{\"env\":{\"ENABLE_PREFIX_DELEGATION\":\"true\"}}"),
			ResolveConflicts:    eksTypes.ResolveConflictsOverwrite,
		},
		{
			AddonName:        aws.String("aws-ebs-csi-driver"),
			AddonVersion:     aws.String("v1.11.2-eksbuild.1"),
			ClusterName:      aws.String(cluster.ProvisionerMetadataEKS.Name),
			ResolveConflicts: eksTypes.ResolveConflictsOverwrite,
		},
	}

	for i, addon := range addons {
		_, err := a.Service().eks.CreateAddon(context.TODO(), &addons[i])
		if err != nil {
			// In case addon already configured we do not want to fail.
			if IsErrorResourceInUseException(err) {
				return nil
			}
			return errors.Wrapf(err, "failed to create %s addon", *addon.AddonName)
		}
	}

	return nil
}

func (c *Client) EnsureEKSClusterUpdated(cluster *model.Cluster) (*eksTypes.Update, error) {
	clusterName := cluster.ProvisionerMetadataEKS.Name
	eksCluster, err := c.getEKSCluster(clusterName)
	if err != nil {
		return nil, err
	}

	if eksCluster == nil {
		return nil, errors.Errorf("requested EKS cluster %s does not exist", clusterName)
	}

	if eksCluster.Status != eksTypes.ClusterStatusActive {
		return nil, errors.Errorf("requested EKS cluster %s is not active", clusterName)
	}

	eksMetadata := cluster.ProvisionerMetadataEKS
	if eksMetadata.ChangeRequest.Version == "" {
		return nil, nil
	}

	output, err := c.Service().eks.UpdateClusterVersion(context.TODO(), &eks.UpdateClusterVersionInput{
		Name:    aws.String(clusterName),
		Version: aws.String(eksMetadata.ChangeRequest.Version),
	})

	if err != nil {
		return nil, errors.Wrap(err, "failed to update EKS cluster version")
	}

	return output.Update, nil
}

func (a *Client) createEKSNodeGroup(cluster *model.Cluster, ngPrefix string) (*eksTypes.Nodegroup, error) {

	eksMetadata := cluster.ProvisionerMetadataEKS
	changeRequest := eksMetadata.ChangeRequest
	if changeRequest == nil {
		return nil, errors.New("metadata ChangeRequest is nil")
	}

	clusterResource, err := a.GetVpcResourcesByVpcID(changeRequest.VPC, a.logger)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get VPC resources")
	}

	clusterName := eksMetadata.Name
	launchTemplate := fmt.Sprintf("%s-%s", clusterName, ngPrefix)

	ngChangeRequest := changeRequest.NodeGroups[ngPrefix]

	var subnets []string
	if ngChangeRequest.WithPublicSubnet {
		subnets = clusterResource.PublicSubnetsIDs
	} else {
		subnets = clusterResource.PrivateSubnetIDs
	}

	subnetsOut, err := a.Service().ec2.DescribeSubnets(context.TODO(), &ec2.DescribeSubnetsInput{
		SubnetIds: subnets,
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
		NodeRole:      &changeRequest.NodeRoleARN,
		NodegroupName: aws.String(ngChangeRequest.Name),
		AmiType:       eksTypes.AMITypesCustom,
		ScalingConfig: &eksTypes.NodegroupScalingConfig{
			DesiredSize: ptr.Int32(int32(ngChangeRequest.MinCount)),
			MaxSize:     ptr.Int32(int32(ngChangeRequest.MaxCount)),
			MinSize:     ptr.Int32(int32(ngChangeRequest.MinCount)),
		},
		Subnets: subnetsIDs,
		LaunchTemplate: &eksTypes.LaunchTemplateSpecification{
			Name:    aws.String(launchTemplate),
			Version: aws.String("$Latest"),
		},
		Tags: map[string]string{
			fmt.Sprintf("kubernetes.io/cluster/%s", clusterName): "owned",
		},
		Labels: map[string]string{
			"type": ngPrefix,
		},
	}

	if ngPrefix != model.NodeGroupWorker {
		nodeGroupReq.Taints = []eksTypes.Taint{
			{
				Effect: eksTypes.TaintEffectNoSchedule,
				Key:    aws.String(ngPrefix),
				Value:  aws.String("true"),
			},
		}
	}

	out, err := a.Service().eks.CreateNodegroup(context.TODO(), &nodeGroupReq)
	if err != nil {
		return nil, errors.Wrap(err, "failed to create EKS nodegroup")
	}

	return out.Nodegroup, nil
}

func (c *Client) getEKSNodeGroup(clusterName, nodegroupName string) (*eksTypes.Nodegroup, error) {

	output, err := c.Service().eks.DescribeNodegroup(context.TODO(), &eks.DescribeNodegroupInput{
		ClusterName:   aws.String(clusterName),
		NodegroupName: aws.String(nodegroupName),
	})
	if err != nil {
		if !IsErrorResourceNotFound(err) {
			return nil, errors.Wrap(err, "failed to describe EKS nodegroup")
		}
	}

	if output != nil && output.Nodegroup != nil {
		return output.Nodegroup, nil
	}

	return nil, nil
}

// EnsureEKSNodegroup ensures EKS nodegroup is created.
func (c *Client) EnsureEKSNodegroup(cluster *model.Cluster, ngPrefix string) (*eksTypes.Nodegroup, error) {

	clusterName := cluster.ProvisionerMetadataEKS.Name

	eksMetadata := cluster.ProvisionerMetadataEKS
	changeRequest := eksMetadata.ChangeRequest

	ngChangeRequest, found := changeRequest.NodeGroups[ngPrefix]
	if !found {
		return nil, errors.Errorf("nodegroup metadata for %s not found in ChangeRequest", ngPrefix)
	}

	nodeGroup, err := c.getEKSNodeGroup(clusterName, ngChangeRequest.Name)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to get an EKS nodegroup %s", ngChangeRequest.Name)
	}

	if nodeGroup != nil {
		return nodeGroup, nil
	}

	return c.createEKSNodeGroup(cluster, ngPrefix)
}

// EnsureEKSNodegroupMigrated updates EKS nodegroup.
func (c *Client) EnsureEKSNodegroupMigrated(cluster *model.Cluster, ngPrefix string) error {
	logger := c.logger.WithField("cluster", cluster.ID)

	eksMetadata := cluster.ProvisionerMetadataEKS
	changeRequest := eksMetadata.ChangeRequest

	ngChangeRequest, found := changeRequest.NodeGroups[ngPrefix]
	if !found {
		return errors.Errorf("nodegroup metadata for %s not found in ChangeRequest", ngPrefix)
	}

	clusterName := eksMetadata.Name

	oldNodeGroupMeta := eksMetadata.NodeGroups[ngPrefix]
	oldNodeGroupName := oldNodeGroupMeta.Name

	changeRequest.VPC = eksMetadata.VPC
	changeRequest.NodeRoleARN = eksMetadata.NodeRoleARN

	_, err := c.createEKSNodeGroup(cluster, ngPrefix)
	if err != nil {
		return errors.Wrapf(err, "failed to create a new EKS nodegroup %s", ngChangeRequest.Name)
	}

	wait := 600 // seconds
	logger.Infof("Waiting up to %d seconds for EKS nodegroup %s to become active...", wait, ngChangeRequest.Name)

	_, err = c.WaitForActiveEKSNodegroup(eksMetadata.Name, ngChangeRequest.Name, wait)
	if err != nil {
		return err
	}

	logger.Debugf("Deleting the old EKS nodegroup %s", oldNodeGroupName)

	err = c.EnsureEKSNodegroupDeleted(clusterName, oldNodeGroupName)
	if err != nil {
		return errors.Wrapf(err, "failed to delete the old EKS nodegroup %s", oldNodeGroupName)
	}

	logger.Infof("Waiting up to %d seconds for EKS nodegroup %s to be deleted...", wait, oldNodeGroupName)
	err = c.WaitForEKSNodegroupToBeDeleted(eksMetadata.Name, oldNodeGroupName, wait)
	if err != nil {
		return err
	}

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
		return errors.New("requested EKS cluster is in failed state")
	}

	delInput := &eks.DeleteClusterInput{Name: aws.String(clusterName)}
	_, err = a.Service().eks.DeleteCluster(ctx, delInput)
	if err != nil {
		return errors.Wrap(err, "failed to trigger EKS cluster deletion")
	}

	// Cluster just started deletion
	return nil
}

// EnsureEKSNodegroupDeleted ensures EKS nodegroup is deleted.
func (a *Client) EnsureEKSNodegroupDeleted(clusterName, nodegroupName string) error {
	if nodegroupName == "" {
		return nil
	}

	nodeGroup, err := a.getEKSNodeGroup(clusterName, nodegroupName)
	if err != nil {
		return errors.Wrap(err, "failed to get EKS nodegroup")
	}

	if nodeGroup == nil {
		return nil
	}

	if nodeGroup.Status == eksTypes.NodegroupStatusDeleting {
		return nil
	}

	if nodeGroup.Status == eksTypes.NodegroupStatusDeleteFailed {
		return errors.Wrapf(err, "failed to delete EKS nodegroup %s", nodegroupName)
	}

	_, err = a.Service().eks.DeleteNodegroup(context.TODO(), &eks.DeleteNodegroupInput{
		ClusterName:   aws.String(clusterName),
		NodegroupName: nodeGroup.NodegroupName,
	})
	if err != nil {
		return errors.Wrap(err, "failed to delete EKS nodegroup")
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
		return nil, errors.New("requested EKS cluster is in failed state")
	}

	if cluster.Status == eksTypes.ClusterStatusActive {
		return cluster, nil
	}

	return nil, nil
}

// GetActiveEKSNodegroup returns the EKS nodegroup if active.
func (c *Client) GetActiveEKSNodegroup(clusterName, workerName string) (*eksTypes.Nodegroup, error) {
	nodeGroup, err := c.getEKSNodeGroup(clusterName, workerName)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get EKS nodegroup")
	}

	if nodeGroup == nil {
		return nil, nil
	}

	if nodeGroup.Status == eksTypes.NodegroupStatusCreateFailed {
		return nil, errors.New("failed to create EKS nodegroup")
	}

	if nodeGroup.Status == eksTypes.NodegroupStatusActive {
		return nodeGroup, nil
	}

	return nil, nil
}

// WaitForActiveEKSCluster waits for EKS cluster to be ready.
func (c *Client) WaitForActiveEKSCluster(clusterName string, timeout int) (*eksTypes.Cluster, error) {
	timeoutTimer := time.NewTimer(time.Duration(timeout) * time.Second)
	defer timeoutTimer.Stop()
	tick := time.NewTicker(5 * time.Second)
	defer tick.Stop()

	for {
		select {
		case <-timeoutTimer.C:
			return nil, errors.New("timed out waiting for EKS cluster to become active")
		case <-tick.C:
			eksCluster, err := c.GetActiveEKSCluster(clusterName)
			if err != nil {
				return nil, errors.Wrap(err, "failed to check if EKS cluster is active")
			}
			if eksCluster != nil {
				return eksCluster, nil
			}
		}
	}
}

func (c *Client) WaitForActiveEKSNodegroup(clusterName, nodeGroupName string, timeout int) (*eksTypes.Nodegroup, error) {
	timeoutTimer := time.NewTimer(time.Duration(timeout) * time.Second)
	defer timeoutTimer.Stop()
	tick := time.NewTicker(5 * time.Second)
	defer tick.Stop()

	for {
		select {
		case <-timeoutTimer.C:
			return nil, errors.New("timed out waiting for EKS nodegroup to become active")
		case <-tick.C:
			nodeGroup, err := c.GetActiveEKSNodegroup(clusterName, nodeGroupName)
			if err != nil {
				return nil, errors.Wrap(err, "failed to check if EKS nodegroup is active")
			}
			if nodeGroup != nil {
				return nodeGroup, nil
			}
		}
	}
}

func (c *Client) WaitForEKSNodegroupToBeDeleted(clusterName, workerName string, timeout int) error {
	timeoutTimer := time.NewTimer(time.Duration(timeout) * time.Second)
	defer timeoutTimer.Stop()
	tick := time.NewTicker(5 * time.Second)
	defer tick.Stop()

	for {
		select {
		case <-timeoutTimer.C:
			return errors.New("timed out waiting for EKS nodegroup to be deleted")
		case <-tick.C:
			nodeGroup, err := c.getEKSNodeGroup(clusterName, workerName)
			if err != nil {
				return errors.Wrap(err, "failed to describe EKS nodegroup")
			}
			if nodeGroup == nil {
				return nil
			}
		}
	}
}

func (c *Client) WaitForEKSClusterToBeDeleted(clusterName string, timeout int) error {
	timeoutTimer := time.NewTimer(time.Duration(timeout) * time.Second)
	defer timeoutTimer.Stop()
	tick := time.NewTicker(5 * time.Second)
	defer tick.Stop()

	for {
		select {
		case <-timeoutTimer.C:
			return errors.New("timed out waiting for EKS cluster to become ready")
		case <-tick.C:
			eksCluster, err := c.getEKSCluster(clusterName)
			if err != nil {
				return errors.Wrap(err, "failed to describe EKS cluster")
			}
			if eksCluster == nil {
				return nil
			}
		}
	}
}

func (c *Client) WaitForEKSClusterUpdateToBeCompleted(clusterName, updateID string, timeout int) error {
	timeoutTimer := time.NewTimer(time.Duration(timeout) * time.Second)
	defer timeoutTimer.Stop()
	tick := time.NewTicker(5 * time.Second)
	defer tick.Stop()

	for {
		select {
		case <-timeoutTimer.C:
			return errors.New("timed out waiting for EKS cluster update to be completed")
		case <-tick.C:
			updateStatus, err := c.getEKSClusterUpdateStatus(clusterName, updateID)
			if err != nil {
				return errors.Wrap(err, "failed to describe EKS cluster")
			}

			if updateStatus == eksTypes.UpdateStatusFailed {
				return errors.New("failed to update EKS cluster")
			}

			if updateStatus == eksTypes.UpdateStatusSuccessful {
				return nil
			}
		}
	}
}

func (c *Client) getEKSClusterUpdateStatus(clusterName, updateID string) (eksTypes.UpdateStatus, error) {
	output, err := c.Service().eks.DescribeUpdate(context.TODO(), &eks.DescribeUpdateInput{
		Name:     ptr.String(clusterName),
		UpdateId: ptr.String(updateID),
	})
	if err != nil {
		return "", errors.Wrap(err, "failed to describe update for EKS cluster")
	}

	return output.Update.Status, nil
}
