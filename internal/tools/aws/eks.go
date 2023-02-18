// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.
//

package aws

import (
	"context"
	"fmt"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	ec2Types "github.com/aws/aws-sdk-go-v2/service/ec2/types"
	"github.com/aws/aws-sdk-go-v2/service/eks"
	eksTypes "github.com/aws/aws-sdk-go-v2/service/eks/types"
	"github.com/mattermost/mattermost-cloud/model"
	"github.com/pkg/errors"
)

// CreateEKSCluster creates EKS cluster.
func (a *Client) CreateEKSCluster(cluster *model.Cluster, eksMetadata model.EKSMetadata) (*eksTypes.Cluster, error) {
	ctx := context.TODO()

	resources := eksMetadata.ClusterResource
	vpcConfig := eksTypes.VpcConfigRequest{
		EndpointPrivateAccess: aws.Bool(true),
		EndpointPublicAccess:  aws.Bool(true),
		SubnetIds:             append(resources.PrivateSubnetIDs, resources.PublicSubnetsIDs...),
	}

	// TODO: we can allow further parametrization in the future
	input := eks.CreateClusterInput{
		Name:               aws.String(cluster.ID),
		ResourcesVpcConfig: &vpcConfig,
		RoleArn:            eksMetadata.ClusterRoleARN,
		Version:            eksMetadata.KubernetesVersion,
	}

	out, err := a.Service().eks.CreateCluster(ctx, &input)
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
	_, err := a.Service().eks.CreateAddon(context.TODO(), &input)
	if err != nil {
		// In case addon already configured we do not want to fail.
		if IsErrorResourceInUseException(err) {
			return nil
		}
		return errors.Wrap(err, "failed to create ebs-csi addon")
	}

	input = eks.CreateAddonInput{
		AddonName:   aws.String("vpc-cni"),
		ClusterName: aws.String(cluster.ID),
	}
	_, err = a.Service().eks.CreateAddon(context.TODO(), &input)
	if err != nil {
		// In case addon already configured we do not want to fail.
		if IsErrorResourceInUseException(err) {
			return nil
		}
		return errors.Wrap(err, "failed to create cni addon")
	}

	return nil
}

// EnsureEKSCluster ensures EKS cluster is created.
func (a *Client) EnsureEKSCluster(cluster *model.Cluster, eksMetadata model.EKSMetadata) (*eksTypes.Cluster, error) {
	input := eks.DescribeClusterInput{
		Name: aws.String(cluster.ID),
	}

	out, err := a.Service().eks.DescribeCluster(context.TODO(), &input)
	if err != nil {
		if IsErrorResourceNotFound(err) {
			return a.CreateEKSCluster(cluster, eksMetadata)
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
	out, err := a.Service().eks.DescribeCluster(context.TODO(), &input)
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

	ctx := context.TODO()

	_, err = a.Service().ec2.AuthorizeSecurityGroupIngress(ctx, &ec2.AuthorizeSecurityGroupIngressInput{
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
	ctx := context.TODO()
	postgresSG, err := a.getPostgresSecurityGroup(eksMetadata.VPC)
	if err != nil {
		return errors.Wrap(err, "failed to get Postgres security group for VPC")
	}

	input := eks.DescribeClusterInput{
		Name: aws.String(cluster.ID),
	}
	out, err := a.Service().eks.DescribeCluster(ctx, &input)
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

	_, err = a.Service().ec2.RevokeSecurityGroupIngress(ctx, &ec2.RevokeSecurityGroupIngressInput{
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

func (a *Client) getPostgresSecurityGroup(vpcID string) (ec2Types.SecurityGroup, error) {
	ctx := context.TODO()
	var postgresSG ec2Types.SecurityGroup
	securityGroupsResp, err := a.Service().ec2.DescribeSecurityGroups(ctx, &ec2.DescribeSecurityGroupsInput{
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

func (a *Client) getEKSPostgresIPPermissions(cluster *eksTypes.Cluster) ([]ec2Types.IpPermission, error) {
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

// EnsureEKSClusterNodeGroups ensures EKS cluster node groups are created.
func (a *Client) EnsureEKSClusterNodeGroups(cluster *model.Cluster, eksMetadata model.EKSMetadata) ([]*eksTypes.Nodegroup, error) {
	return a.CreateNodeGroups(cluster.ID, eksMetadata)
}

// CreateNodeGroups creates node groups for EKS cluster.
func (a *Client) CreateNodeGroups(clusterName string, eksMetadata model.EKSMetadata) ([]*eksTypes.Nodegroup, error) {
	ctx := context.TODO()

	// If more node groups exist than we expect, this function will not
	// delete them, nor return them.
	existingNgs, err := a.listNodeGroups(ctx, clusterName)
	if err != nil {
		return nil, errors.Wrap(err, "failed to list existing node groups")
	}

	for _, existingNg := range existingNgs {
		if *existingNg.NodegroupName == "worker" {
			return nil, nil
		}
	}

	ngCfg := eksMetadata.NodeGroup

	nodeGroupReq := eks.CreateNodegroupInput{
		ClusterName:   aws.String(clusterName),
		InstanceTypes: ngCfg.InstanceTypes,
		LaunchTemplate: &eksTypes.LaunchTemplateSpecification{
			Name:    aws.String(eksMetadata.LaunchTemplateName),
			Version: aws.String("$Latest"),
		},
		AmiType:       eksTypes.AMITypesCustom,
		NodeRole:      ngCfg.RoleARN,
		NodegroupName: aws.String("worker"),
		ScalingConfig: &eksTypes.NodegroupScalingConfig{
			DesiredSize: ngCfg.DesiredSize,
			MaxSize:     ngCfg.MaxSize,
			MinSize:     ngCfg.MinSize,
		},
		Subnets: append(eksMetadata.ClusterResource.PrivateSubnetIDs),
		Tags: map[string]string{
			fmt.Sprintf("kubernetes.io/cluster/%s", clusterName): "owned",
		},
	}

	out, err := a.Service().eks.CreateNodegroup(ctx, &nodeGroupReq)
	if err != nil {
		return nil, errors.Wrap(err, "failed to create one of the node groups")
	}

	return []*eksTypes.Nodegroup{out.Nodegroup}, nil
}

// IsClusterReady checks if EKS cluster is ready.
func (a *Client) IsClusterReady(clusterName string) (bool, error) {
	cluster, err := a.GetEKSCluster(clusterName)
	if err != nil {
		return false, errors.Wrap(err, "failed to get EKS cluster")
	}

	if cluster.Status == eksTypes.ClusterStatusFailed {
		return false, errors.New("cluster creation failed")
	}
	if cluster.Status != eksTypes.ClusterStatusActive {
		return false, nil
	}

	return true, nil
}

// GetEKSCluster returns EKS cluster with given name.
func (a *Client) GetEKSCluster(clusterName string) (*eksTypes.Cluster, error) {
	input := eks.DescribeClusterInput{
		Name: aws.String(clusterName),
	}

	out, err := a.Service().eks.DescribeCluster(context.TODO(), &input)
	if err != nil {
		return nil, errors.Wrap(err, "failed to create EKS cluster")
	}

	return out.Cluster, nil
}

// EnsureNodeGroupsDeleted ensures EKS node groups are deleted.
func (a *Client) EnsureNodeGroupsDeleted(cluster *model.Cluster) (bool, error) {
	ctx := context.TODO()
	nodeGroups, err := a.listNodeGroups(ctx, cluster.ID)
	if err != nil {
		return false, errors.Wrap(err, "failed to list node groups for the cluster")
	}
	// Node groups deleted, we can return
	if len(nodeGroups) == 0 {
		return true, nil
	}

	for _, ng := range nodeGroups {
		if ng.Status == eksTypes.NodegroupStatusDeleting {
			continue
		}
		if ng.Status == eksTypes.NodegroupStatusDeleteFailed {
			return false, errors.Wrapf(err, "node group deletion failed %q", *ng.NodegroupName)
		}

		delNgReq := &eks.DeleteNodegroupInput{
			ClusterName:   aws.String(cluster.ID),
			NodegroupName: ng.NodegroupName,
		}

		_, err = a.Service().eks.DeleteNodegroup(ctx, delNgReq)
		if err != nil {
			return false, errors.Wrap(err, "failed to delete node group")
		}
	}

	// Node groups still exist therefore we return false
	return false, nil
}

// EnsureEKSClusterDeleted ensures EKS cluster is deleted.
func (a *Client) EnsureEKSClusterDeleted(cluster *model.Cluster) (bool, error) {
	ctx := context.TODO()

	input := eks.DescribeClusterInput{
		Name: aws.String(cluster.ID),
	}

	out, err := a.Service().eks.DescribeCluster(ctx, &input)
	if err != nil {
		// Cluster was deleted
		if IsErrorResourceNotFound(err) {
			return true, nil
		}
		return false, errors.Wrap(err, "failed to check if EKS cluster exists")
	}

	// Still deleting
	if out.Cluster.Status == eksTypes.ClusterStatusDeleting {
		return false, nil
	}

	delInput := &eks.DeleteClusterInput{Name: aws.String(cluster.ID)}
	_, err = a.Service().eks.DeleteCluster(ctx, delInput)
	if err != nil {
		return false, errors.Wrap(err, "failed to trigger EKS cluster deletion")
	}

	// Cluster just started deletion
	return false, nil
}

func (a *Client) listNodeGroups(ctx context.Context, clusterName string) ([]*eksTypes.Nodegroup, error) {
	listNgInput := eks.ListNodegroupsInput{
		ClusterName: aws.String(clusterName),
	}

	nodeGroups := make([]*eksTypes.Nodegroup, 0)
	for {
		ngListOut, err := a.Service().eks.ListNodegroups(ctx, &listNgInput)
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
				NodegroupName: aws.String(ng),
			}
			out, err := a.Service().eks.DescribeNodegroup(ctx, &ngInput)
			if err != nil {
				return nil, errors.Wrapf(err, "failed to describe node group %q", ng)
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
