// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.
//

package aws

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	ec2Types "github.com/aws/aws-sdk-go-v2/service/ec2/types"
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
		EndpointPrivateAccess: nil,
		EndpointPublicAccess:  nil,
		SecurityGroupIds:      resources.MasterSecurityGroupIDs,
		SubnetIds:             subnetsIDs,
	}

	eksMetadata := cluster.ProvisionerMetadataEKS
	// TODO: we can allow further parametrization in the future
	input := eks.CreateClusterInput{
		Name:               aws.String(cluster.ID),
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

// InstallEKSEBSAddon installs EKS EBS addon to the existing cluster.
func (c *Client) InstallEKSEBSAddon(cluster *model.Cluster) error {
	input := eks.CreateAddonInput{
		AddonName:   aws.String("aws-ebs-csi-driver"),
		ClusterName: aws.String(cluster.ID),
	}
	_, err := c.Service().eks.CreateAddon(context.TODO(), &input)
	if err != nil {
		// In case addon already configured we do not want to fail.
		if IsErrorResourceInUseException(err) {
			return nil
		}
		return errors.Wrap(err, "failed to create ebs-csi addon")
	}

	return nil
}

// EnsureEKSCluster ensures EKS cluster is created.
func (c *Client) EnsureEKSCluster(cluster *model.Cluster, resources ClusterResources) (*eksTypes.Cluster, error) {
	clusterName := cluster.ProvisionerMetadataEKS.Name
	eksCluster, err := c.describeCluster(clusterName)
	if err != nil {
		return nil, err
	}

	if eksCluster != nil {
		return eksCluster, nil
	}
	return c.createEKSCluster(cluster, resources)
}

// AllowEKSPostgresTraffic allows traffic to Postgres from EKS Security
// Group.
func (c *Client) AllowEKSPostgresTraffic(cluster *model.Cluster, eksMetadata model.EKSMetadata) error {
	input := eks.DescribeClusterInput{
		Name: aws.String(cluster.ID),
	}
	out, err := c.Service().eks.DescribeCluster(context.TODO(), &input)
	if err != nil {
		return errors.Wrap(err, "failed to describe EKS cluster")
	}

	postgresSG, err := c.getPostgresSecurityGroup(eksMetadata.ChangeRequest.VPC)
	if err != nil {
		return errors.Wrap(err, "failed to get Postgres security group for VPC")
	}

	ipPermissions, err := c.getEKSPostgresIPPermissions(out.Cluster)
	if err != nil {
		return errors.Wrap(err, "failed to get EKS Postgres IP permissions")
	}

	ctx := context.TODO()

	_, err = c.Service().ec2.AuthorizeSecurityGroupIngress(ctx, &ec2.AuthorizeSecurityGroupIngressInput{
		GroupId:       postgresSG.GroupId,
		IpPermissions: ipPermissions,
	})
	if err != nil {
		if IsErrorPermissionDuplicate(err) {
			return nil
		}
		return errors.Wrap(err, "failed to authorize rule")
	}

	return nil
}

// RevokeEKSPostgresTraffic revokes Postgres traffic permission from EKS
// Security Group.
func (c *Client) RevokeEKSPostgresTraffic(cluster *model.Cluster, eksMetadata model.EKSMetadata) error {
	ctx := context.TODO()
	postgresSG, err := c.getPostgresSecurityGroup(eksMetadata.ChangeRequest.VPC)
	if err != nil {
		return errors.Wrap(err, "failed to get Postgres security group for VPC")
	}

	input := eks.DescribeClusterInput{
		Name: aws.String(cluster.ID),
	}
	out, err := c.Service().eks.DescribeCluster(ctx, &input)
	if err != nil {
		if IsErrorResourceNotFound(err) {
			return nil
		}
		return errors.Wrap(err, "failed to describe EKS cluster")
	}

	ipPermissions, err := c.getEKSPostgresIPPermissions(out.Cluster)
	if err != nil {
		return errors.Wrap(err, "failed to get EKS Postgres IP permissions")
	}

	_, err = c.Service().ec2.RevokeSecurityGroupIngress(ctx, &ec2.RevokeSecurityGroupIngressInput{
		GroupId:       postgresSG.GroupId,
		IpPermissions: ipPermissions,
	})
	if err != nil {
		if IsErrorPermissionNotFound(err) {
			return nil
		}
		return errors.Wrap(err, "failed to revoke security group ingress")
	}
	return nil
}

func (c *Client) getPostgresSecurityGroup(vpcID string) (ec2Types.SecurityGroup, error) {
	ctx := context.TODO()
	var postgresSG ec2Types.SecurityGroup
	securityGroupsResp, err := c.Service().ec2.DescribeSecurityGroups(ctx, &ec2.DescribeSecurityGroupsInput{
		DryRun: nil,
		Filters: []ec2Types.Filter{
			{Name: aws.String("vpc-id"), Values: []string{vpcID}},
		},
		// TODO: make sure to list all
	})
	if err != nil {
		return postgresSG, errors.Wrap(err, "failed to describe security groups for VPC")
	}

	for _, sg := range securityGroupsResp.SecurityGroups {
		if strings.HasSuffix(*sg.GroupName, "-db-postgresql-sg") {
			postgresSG = sg
			break
		}
	}
	if postgresSG.GroupName == nil {
		return postgresSG, errors.New("postgres db security group not found")
	}

	return postgresSG, nil
}

func (c *Client) getEKSPostgresIPPermissions(cluster *eksTypes.Cluster) ([]ec2Types.IpPermission, error) {
	eksSecurityGroup := cluster.ResourcesVpcConfig.ClusterSecurityGroupId

	return []ec2Types.IpPermission{{
		FromPort:   aws.Int32(5432),
		IpProtocol: aws.String("tcp"),
		ToPort:     aws.Int32(5432),
		UserIdGroupPairs: []ec2Types.UserIdGroupPair{
			{GroupId: eksSecurityGroup, Description: aws.String("EKS permission")},
		},
	}}, nil
}

func (c *Client) createEKSClusterNodeGroup(clusterName string, eksMetadata *model.EKSMetadata) (*eksTypes.Nodegroup, error) {

	changeRequest := eksMetadata.ChangeRequest
	if changeRequest == nil {
		return nil, errors.New("change request is nil")
	}

	clusterResource, err := c.GetVpcResourcesByVpcID(eksMetadata.ChangeRequest.VPC, c.logger)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get VPC resources")
	}

	launchTemplate := getLaunchTemplateName(clusterName)

	nodeGroupReq := eks.CreateNodegroupInput{
		ClusterName:   aws.String(clusterName),
		InstanceTypes: []string{changeRequest.NodeInstanceType},
		NodeRole:      &changeRequest.NodeRoleARN,
		NodegroupName: aws.String("worker"),
		AmiType:       eksTypes.AMITypesCustom,
		ScalingConfig: &eksTypes.NodegroupScalingConfig{
			DesiredSize: ptr.Int32(int32(changeRequest.NodeMinCount)),
			MaxSize:     ptr.Int32(int32(changeRequest.NodeMaxCount)),
			MinSize:     ptr.Int32(int32(changeRequest.NodeMinCount)),
		},
		Subnets: clusterResource.PrivateSubnetIDs,
		LaunchTemplate: &eksTypes.LaunchTemplateSpecification{
			Name:    aws.String(launchTemplate),
			Version: aws.String(fmt.Sprintf("%d", *changeRequest.LaunchTemplateVersion)),
		},
		Tags: map[string]string{
			fmt.Sprintf("kubernetes.io/cluster/%s", clusterName): "owned",
		},
	}

	out, err := c.Service().eks.CreateNodegroup(context.TODO(), &nodeGroupReq)
	if err != nil {
		return nil, errors.Wrap(err, "failed to create one of the node groups")
	}

	return out.Nodegroup, nil
}

// EnsureEKSClusterNodeGroups ensures EKS cluster node groups are created.
func (c *Client) EnsureEKSClusterNodeGroups(cluster *model.Cluster) (*eksTypes.Nodegroup, error) {

	clusterName := cluster.ProvisionerMetadataEKS.Name
	existingNodeGroup, err := c.describeNodeGroup(clusterName)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get existing node group")
	}

	if existingNodeGroup != nil {
		return existingNodeGroup, nil
	}

	return c.createEKSClusterNodeGroup(clusterName, cluster.ProvisionerMetadataEKS)
}

// EnsureEKSClusterNodeGroupUpdated updates EKS cluster node group.
func (c *Client) EnsureEKSClusterNodeGroupUpdated(cluster *model.Cluster) (*eksTypes.Nodegroup, error) {
	eksMetadata := cluster.ProvisionerMetadataEKS
	changeRequest := eksMetadata.ChangeRequest
	clusterName := eksMetadata.Name

	if eksMetadata.ChangeRequest == nil {
		return nil, errors.New("change request is nil")
	}

	nodeGroup, err := c.describeNodeGroup(clusterName)
	if err != nil {
		return nil, errors.Wrap(err, "failed to describe node group")
	}

	if nodeGroup == nil {
		return c.createEKSClusterNodeGroup(clusterName, eksMetadata)
	}

	if nodeGroup.Status != eksTypes.NodegroupStatusActive {
		return nodeGroup, nil
	}

	updateRequest := &eks.UpdateNodegroupVersionInput{
		ClusterName:   aws.String(clusterName),
		NodegroupName: aws.String("worker"),
	}

	isUpdateRequired := false

	if changeRequest.Version != "" {
		if nodeGroup.Version != nil && *nodeGroup.Version != changeRequest.Version {
			isUpdateRequired = true
			updateRequest.Version = aws.String(changeRequest.Version)
		}
	}

	if changeRequest.LaunchTemplateVersion != nil {
		if nodeGroup.LaunchTemplate != nil && nodeGroup.LaunchTemplate.Version != nil &&
			*nodeGroup.LaunchTemplate.Version != fmt.Sprintf("%d", *changeRequest.LaunchTemplateVersion) {
			isUpdateRequired = true
			updateRequest.LaunchTemplate = &eksTypes.LaunchTemplateSpecification{
				Name:    aws.String(getLaunchTemplateName(clusterName)),
				Version: aws.String(fmt.Sprintf("%d", *changeRequest.LaunchTemplateVersion)),
			}
		}
	}

	if !isUpdateRequired {
		return nodeGroup, nil
	}

	_, err = c.Service().eks.UpdateNodegroupVersion(context.TODO(), updateRequest)
	if err != nil {
		return nil, errors.Wrap(err, "failed to update one of the node groups")
	}

	nodeGroup, err = c.describeNodeGroup(clusterName)
	if err != nil {
		return nil, errors.Wrap(err, "failed to describe node group")
	}

	return nodeGroup, nil
}

// GetReadyCluster returns the EKS cluster if ready.
func (c *Client) GetReadyCluster(clusterName string) (*eksTypes.Cluster, error) {
	cluster, err := c.getEKSCluster(clusterName)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get EKS cluster")
	}

	if cluster.Status == eksTypes.ClusterStatusFailed {
		return nil, errors.New("cluster creation failed")
	}
	if cluster.Status != eksTypes.ClusterStatusActive {
		return nil, nil
	}

	return cluster, nil
}

func (c *Client) WaitForNodeGroupReadiness(clusterName string, timeout int) error {
	timer := time.NewTimer(time.Duration(timeout) * time.Second)
	defer timer.Stop()

	for {
		select {
		case <-timer.C:
			return errors.New("timed out waiting for eks nodegroup to become ready")
		default:
			nodeGroup, err := c.describeNodeGroup(clusterName)
			if err != nil {
				return errors.Wrap(err, "failed to describe node group")
			}
			if nodeGroup == nil {
				return errors.New("node group not found")
			}

			if nodeGroup.Status == eksTypes.NodegroupStatusActive {
				return nil
			}

			time.Sleep(5 * time.Second)
		}
	}
}

// getEKSCluster returns EKS cluster with given name.
func (c *Client) getEKSCluster(clusterName string) (*eksTypes.Cluster, error) {
	input := eks.DescribeClusterInput{
		Name: aws.String(clusterName),
	}

	out, err := c.Service().eks.DescribeCluster(context.TODO(), &input)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get EKS cluster")
	}

	return out.Cluster, nil
}

// EnsureNodeGroupsDeleted ensures EKS node groups are deleted.
func (a *Client) EnsureNodeGroupsDeleted(cluster *model.Cluster) (bool, error) {

	nodeGroups, err := a.describeNodeGroup(cluster.ID)
	if err != nil {
		return false, errors.Wrap(err, "failed to get node group")
	}
	// Node groups deleted, we can return
	if nodeGroups == nil {
		return true, nil
	}

	if nodeGroups.Status == eksTypes.NodegroupStatusDeleting {
		return false, nil
	}
	if nodeGroups.Status == eksTypes.NodegroupStatusDeleteFailed {
		return false, errors.Wrapf(err, "node group deletion failed %q", *nodeGroups.NodegroupName)
	}

	_, err = a.Service().eks.DeleteNodegroup(context.TODO(), &eks.DeleteNodegroupInput{
		ClusterName:   aws.String(cluster.ID),
		NodegroupName: nodeGroups.NodegroupName,
	})
	if err != nil {
		return false, errors.Wrap(err, "failed to delete node group")
	}

	// Node groups still exist therefore we return false
	return false, nil
}

// EnsureEKSClusterDeleted ensures EKS cluster is deleted.
func (a *Client) EnsureEKSClusterDeleted(cluster *model.Cluster) (bool, error) {
	ctx := context.TODO()

	clusterName := cluster.ProvisionerMetadataEKS.Name

	eksCLuster, err := a.describeCluster(clusterName)
	if err != nil {
		return false, errors.Wrap(err, "failed to describe cluster")
	}

	if eksCLuster == nil {
		return true, nil
	}

	// Still deleting
	if eksCLuster.Status == eksTypes.ClusterStatusDeleting {
		return false, nil
	}

	delInput := &eks.DeleteClusterInput{Name: aws.String(clusterName)}
	_, err = a.Service().eks.DeleteCluster(ctx, delInput)
	if err != nil {
		return false, errors.Wrap(err, "failed to trigger EKS cluster deletion")
	}

	// Cluster just started deletion
	return false, nil
}

func (c *Client) describeCluster(clusterName string) (*eksTypes.Cluster, error) {

	cluster, err := c.Service().eks.DescribeCluster(context.TODO(), &eks.DescribeClusterInput{
		Name: aws.String(clusterName),
	})
	if err != nil {
		if !IsErrorResourceNotFound(err) {
			return nil, errors.Wrap(err, "failed to describe cluster")
		}
	}

	if cluster == nil || cluster.Cluster == nil {
		return nil, nil
	}

	return cluster.Cluster, nil

}

func (c *Client) describeNodeGroup(clusterName string) (*eksTypes.Nodegroup, error) {

	nodeGroup, err := c.Service().eks.DescribeNodegroup(context.TODO(), &eks.DescribeNodegroupInput{
		ClusterName:   aws.String(clusterName),
		NodegroupName: aws.String("worker"),
	})
	if err != nil {
		if !IsErrorResourceNotFound(err) {
			return nil, errors.Wrap(err, "failed to describe node group")
		}
	}

	if nodeGroup == nil || nodeGroup.Nodegroup == nil {
		return nil, nil
	}

	return nodeGroup.Nodegroup, nil
}
