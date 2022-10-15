// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.
//

package aws

import (
	"strings"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/aws/aws-sdk-go/service/eks"
	"github.com/mattermost/mattermost-cloud/model"
	"github.com/pkg/errors"
)

// CreateEKSCluster creates EKS cluster.
func (a *Client) CreateEKSCluster(cluster *model.Cluster, resources ClusterResources, eksMetadata model.EKSMetadata) (*eks.Cluster, error) {
	// TODO: we do not expect to query that many subnets but for safety
	// we can check the NextToken.
	subnetsOut, err := a.Service().ec2.DescribeSubnets(&ec2.DescribeSubnetsInput{
		SubnetIds: stringsToPtr(resources.PublicSubnetsIDs),
	})
	if err != nil {
		return nil, errors.Wrap(err, "failed to describe subnets")
	}

	subnetsIDs := []*string{}
	for _, sub := range subnetsOut.Subnets {
		// TODO: for some reason it is not possible to creates EKS in this zone
		if *sub.AvailabilityZone == "us-east-1e" {
			continue
		}
		subnetsIDs = append(subnetsIDs, sub.SubnetId)
	}

	vpcConfig := eks.VpcConfigRequest{
		EndpointPrivateAccess: nil,
		EndpointPublicAccess:  nil,
		SecurityGroupIds:      stringsToPtr(resources.MasterSecurityGroupIDs),
		SubnetIds:             subnetsIDs,
	}

	// TODO: we can allow further parametrization in the future
	input := eks.CreateClusterInput{
		Name:               aws.String(cluster.ID),
		ResourcesVpcConfig: &vpcConfig,
		RoleArn:            eksMetadata.ClusterRoleARN,
		Version:            eksMetadata.KubernetesVersion,
	}

	out, err := a.Service().eks.CreateCluster(&input)
	if err != nil {
		return nil, errors.Wrap(err, "failed to create EKS cluster")
	}

	return out.Cluster, nil
}

// InstallEKSEBSAddon installs EKS EBS addon to the existing cluster.
func (a *Client) InstallEKSEBSAddon(cluster *model.Cluster) error {
	input := eks.CreateAddonInput{
		AddonName:   aws.String("aws-ebs-csi-driver"),
		ClusterName: aws.String(cluster.ID),
	}
	_, err := a.Service().eks.CreateAddon(&input)
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
func (a *Client) EnsureEKSCluster(cluster *model.Cluster, resources ClusterResources, eksMetadata model.EKSMetadata) (*eks.Cluster, error) {
	input := eks.DescribeClusterInput{
		Name: aws.String(cluster.ID),
	}

	out, err := a.Service().eks.DescribeCluster(&input)
	if err != nil {
		if IsErrorResourceNotFound(err) {
			return a.CreateEKSCluster(cluster, resources, eksMetadata)
		}
		return nil, errors.Wrap(err, "failed to check if EKS cluster exists")
	}

	return out.Cluster, nil
}

// AllowEKSPostgresTraffic allows traffic to Postgres from EKS Security
// Group.
func (a *Client) AllowEKSPostgresTraffic(cluster *model.Cluster, eksMetadata model.EKSMetadata) error {
	input := eks.DescribeClusterInput{
		Name: aws.String(cluster.ID),
	}
	out, err := a.Service().eks.DescribeCluster(&input)
	if err != nil {
		return errors.Wrap(err, "failed to describe EKS cluster")
	}

	postgresSG, err := a.getPostgresSecurityGroup(eksMetadata.VPC)
	if err != nil {
		return errors.Wrap(err, "failed to get Postgres security group for VPC")
	}

	ipPermissions, err := a.getEKSPostgresIPPermissions(out.Cluster)
	if err != nil {
		return errors.Wrap(err, "failed to get EKS Postgres IP permissions")
	}

	_, err = a.Service().ec2.AuthorizeSecurityGroupIngress(&ec2.AuthorizeSecurityGroupIngressInput{
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
func (a *Client) RevokeEKSPostgresTraffic(cluster *model.Cluster, eksMetadata model.EKSMetadata) error {
	postgresSG, err := a.getPostgresSecurityGroup(eksMetadata.VPC)
	if err != nil {
		return errors.Wrap(err, "failed to get Postgres security group for VPC")
	}

	input := eks.DescribeClusterInput{
		Name: aws.String(cluster.ID),
	}
	out, err := a.Service().eks.DescribeCluster(&input)
	if err != nil {
		if IsErrorResourceNotFound(err) {
			return nil
		}
		return errors.Wrap(err, "failed to describe EKS cluster")
	}

	ipPermissions, err := a.getEKSPostgresIPPermissions(out.Cluster)
	if err != nil {
		return errors.Wrap(err, "failed to get EKS Postgres IP permissions")
	}

	_, err = a.Service().ec2.RevokeSecurityGroupIngress(&ec2.RevokeSecurityGroupIngressInput{
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

func (a *Client) getPostgresSecurityGroup(vpcID string) (*ec2.SecurityGroup, error) {
	securityGroupsResp, err := a.Service().ec2.DescribeSecurityGroups(&ec2.DescribeSecurityGroupsInput{
		DryRun: nil,
		Filters: []*ec2.Filter{
			{Name: aws.String("vpc-id"), Values: []*string{&vpcID}},
		},
		// TODO: make sure to list all
	})
	if err != nil {
		return nil, errors.Wrap(err, "failed to describe security groups for VPC")
	}

	var postgresSG *ec2.SecurityGroup
	for _, sg := range securityGroupsResp.SecurityGroups {
		if strings.HasSuffix(*sg.GroupName, "-db-postgresql-sg") {
			postgresSG = sg
			break
		}
	}
	if postgresSG == nil {
		return nil, errors.New("postgres db security group not found")
	}

	return postgresSG, nil
}

func (a *Client) getEKSPostgresIPPermissions(cluster *eks.Cluster) ([]*ec2.IpPermission, error) {
	eksSecurityGroup := cluster.ResourcesVpcConfig.ClusterSecurityGroupId

	return []*ec2.IpPermission{{
		FromPort:   aws.Int64(5432),
		IpProtocol: aws.String("tcp"),
		ToPort:     aws.Int64(5432),
		UserIdGroupPairs: []*ec2.UserIdGroupPair{
			{GroupId: eksSecurityGroup, Description: aws.String("EKS permission")},
		},
	}}, nil
}

// EnsureEKSClusterNodeGroups ensures EKS cluster node groups are created.
func (a *Client) EnsureEKSClusterNodeGroups(cluster *model.Cluster, resources ClusterResources, eksMetadata model.EKSMetadata) ([]*eks.Nodegroup, error) {
	return a.CreateNodeGroups(cluster.ID, resources, eksMetadata)
}

// CreateNodeGroups creates node groups for EKS cluster.
func (a *Client) CreateNodeGroups(clusterName string, resources ClusterResources, eksMetadata model.EKSMetadata) ([]*eks.Nodegroup, error) {
	// If more node groups exist than we expect, this function will not
	// delete them, nor return them.
	existingNgs, err := a.listNodeGroups(clusterName)
	if err != nil {
		return nil, errors.Wrap(err, "failed to list existing node groups")
	}

	allNodeGroups := make([]*eks.Nodegroup, 0, len(eksMetadata.EKSNodeGroups))

	for ngName, ngCfg := range eksMetadata.EKSNodeGroups {
		// If given node group already exist, just return it
		foundExisting := false
		for _, existingNg := range existingNgs {
			if *existingNg.NodegroupName == ngName {
				// TODO: theoretically the node group config could change
				// so we probably should update here.
				allNodeGroups = append(allNodeGroups, existingNg)
				foundExisting = true
				break
			}
		}
		if foundExisting {
			continue
		}

		nodeGroupReq := eks.CreateNodegroupInput{
			ClusterName:    aws.String(clusterName),
			InstanceTypes:  ngCfg.InstanceTypes,
			NodeRole:       ngCfg.RoleARN,
			NodegroupName:  aws.String(ngName),
			ReleaseVersion: ngCfg.AMIVersion,
			ScalingConfig: &eks.NodegroupScalingConfig{
				DesiredSize: ngCfg.DesiredSize,
				MaxSize:     ngCfg.MaxSize,
				MinSize:     ngCfg.MinSize,
			},
			Subnets: stringsToPtr(resources.PrivateSubnetIDs),
			Version: eksMetadata.KubernetesVersion,
		}

		out, err := a.Service().eks.CreateNodegroup(&nodeGroupReq)
		if err != nil {

			return nil, errors.Wrap(err, "failed to create one of the node groups")
		}

		allNodeGroups = append(allNodeGroups, out.Nodegroup)
	}
	return allNodeGroups, nil
}

// IsClusterReady checks if EKS cluster is ready.
func (a *Client) IsClusterReady(clusterName string) (bool, error) {
	cluster, err := a.GetEKSCluster(clusterName)
	if err != nil {
		return false, errors.Wrap(err, "failed to get EKS cluster")
	}

	if cluster.Status == nil {
		return false, nil
	}
	if *cluster.Status == eks.ClusterStatusFailed {
		return false, errors.New("cluster creation failed")
	}
	if *cluster.Status != eks.ClusterStatusActive {
		return false, nil
	}

	return true, nil
}

// GetEKSCluster returns EKS cluster with given name.
func (a *Client) GetEKSCluster(clusterName string) (*eks.Cluster, error) {
	input := eks.DescribeClusterInput{
		Name: aws.String(clusterName),
	}

	out, err := a.Service().eks.DescribeCluster(&input)
	if err != nil {
		return nil, errors.Wrap(err, "failed to create EKS cluster")
	}

	return out.Cluster, nil
}

// EnsureNodeGroupsDeleted ensures EKS node groups are deleted.
func (a *Client) EnsureNodeGroupsDeleted(cluster *model.Cluster) (bool, error) {
	nodeGroups, err := a.listNodeGroups(cluster.ID)
	if err != nil {
		return false, errors.Wrap(err, "failed to list node groups for the cluster")
	}
	// Node groups deleted, we can return
	if len(nodeGroups) == 0 {
		return true, nil
	}

	for _, ng := range nodeGroups {
		if ng.Status != nil {
			if *ng.Status == eks.NodegroupStatusDeleting {
				continue
			}
			if *ng.Status == eks.NodegroupStatusDeleteFailed {
				return false, errors.Wrapf(err, "node group deletion failed %q", *ng.NodegroupName)
			}
		}

		delNgReq := &eks.DeleteNodegroupInput{
			ClusterName:   aws.String(cluster.ID),
			NodegroupName: ng.NodegroupName,
		}

		_, err = a.Service().eks.DeleteNodegroup(delNgReq)
		if err != nil {
			return false, errors.Wrap(err, "failed to delete node group")
		}
	}

	// Node groups still exist therefore we return false
	return false, nil
}

// EnsureEKSClusterDeleted ensures EKS cluster is deleted.
func (a *Client) EnsureEKSClusterDeleted(cluster *model.Cluster) (bool, error) {
	input := eks.DescribeClusterInput{
		Name: aws.String(cluster.ID),
	}

	out, err := a.Service().eks.DescribeCluster(&input)
	if err != nil {
		// Cluster was deleted
		if IsErrorResourceNotFound(err) {
			return true, nil
		}
		return false, errors.Wrap(err, "failed to check if EKS cluster exists")
	}

	// Still deleting
	if out.Cluster.Status != nil && *out.Cluster.Status == eks.ClusterStatusDeleting {
		return false, nil
	}

	delInput := &eks.DeleteClusterInput{Name: aws.String(cluster.ID)}
	_, err = a.Service().eks.DeleteCluster(delInput)
	if err != nil {
		return false, errors.Wrap(err, "failed to trigger EKS cluster deletion")
	}

	// Cluster just started deletion
	return false, nil
}

func (a *Client) listNodeGroups(clusterName string) ([]*eks.Nodegroup, error) {
	listNgInput := eks.ListNodegroupsInput{
		ClusterName: aws.String(clusterName),
	}

	nodeGroups := make([]*eks.Nodegroup, 0)
	for {
		ngListOut, err := a.Service().eks.ListNodegroups(&listNgInput)
		if err != nil {
			// If cluster does not exist anymore, listing node groups will
			// fail with ResourceNotFoundException.
			if IsErrorResourceNotFound(err) {
				return nodeGroups, nil
			}
			return nil, errors.Wrap(err, "failed to list node groups for the cluster")
		}

		for _, ng := range ngListOut.Nodegroups {
			ngInput := eks.DescribeNodegroupInput{
				ClusterName:   aws.String(clusterName),
				NodegroupName: ng,
			}
			out, err := a.Service().eks.DescribeNodegroup(&ngInput)
			if err != nil {
				return nil, errors.Wrapf(err, "failed to describe node group %q", *ng)
			}
			nodeGroups = append(nodeGroups, out.Nodegroup)
		}

		if ngListOut.NextToken == nil {
			break
		}
		listNgInput.NextToken = ngListOut.NextToken
	}

	return nodeGroups, nil
}

func stringsToPtr(list []string) []*string {
	out := make([]*string, 0, len(list))

	for i := range list {
		out = append(out, &list[i])
	}
	return out
}
