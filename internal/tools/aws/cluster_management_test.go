// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.
//

package aws

import (
	"context"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	ec2Types "github.com/aws/aws-sdk-go-v2/service/ec2/types"
	"github.com/golang/mock/gomock"
	"github.com/sirupsen/logrus"
)

func (a *AWSTestSuite) TestFixSubnetTagsForVPC() {
	vpcID := "mock-vpc"
	privateSubnetID := "private-id1"
	publicSubnetID := "public-id1"
	ctx := context.TODO()

	a.Run("correct vpc subnets does nothing", func() {
		a.SetupTest()
		gomock.InOrder(
			// Look into private subnets
			a.Mocks.API.EC2.EXPECT().
				DescribeSubnets(ctx, &ec2.DescribeSubnetsInput{
					Filters: []ec2Types.Filter{
						{
							Name:   aws.String("vpc-id"),
							Values: []string{vpcID},
						},
						{
							Name:   aws.String("tag:SubnetType"),
							Values: []string{"Private"},
						},
					},
				}).
				Return(&ec2.DescribeSubnetsOutput{
					Subnets: []ec2Types.Subnet{},
				}, nil).
				Times(1),
			// Look into public subnets
			a.Mocks.API.EC2.EXPECT().
				DescribeSubnets(ctx, &ec2.DescribeSubnetsInput{
					Filters: []ec2Types.Filter{
						{
							Name:   aws.String("vpc-id"),
							Values: []string{vpcID},
						},
						{
							Name:   aws.String("tag:SubnetType"),
							Values: []string{"Utility"},
						},
					},
				}).
				Return(&ec2.DescribeSubnetsOutput{
					Subnets: []ec2Types.Subnet{},
				}, nil).
				Times(1),
		)

		logger := logrus.New()

		err := a.Mocks.AWS.FixSubnetTagsForVPC(vpcID, logger)
		a.Assert().NoError(err)
	})

	a.Run("incorrect vpc subnets fixes tags", func() {
		a.SetupTest()
		gomock.InOrder(
			// Look into private subnets
			a.Mocks.API.EC2.EXPECT().
				DescribeSubnets(ctx, &ec2.DescribeSubnetsInput{
					Filters: []ec2Types.Filter{
						{
							Name:   aws.String("vpc-id"),
							Values: []string{vpcID},
						},
						{
							Name:   aws.String("tag:SubnetType"),
							Values: []string{"Private"},
						},
					},
				}).
				Return(&ec2.DescribeSubnetsOutput{
					Subnets: []ec2Types.Subnet{{
						SubnetId: aws.String(privateSubnetID),
						Tags: []ec2Types.Tag{
							{
								Key:   aws.String("SubnetType"),
								Value: aws.String("Private"),
							},
						},
					}},
				}, nil).
				Times(1),
			// Fix private tag
			a.Mocks.API.EC2.EXPECT().
				CreateTags(ctx, &ec2.CreateTagsInput{
					Resources: []string{privateSubnetID},
					Tags: []ec2Types.Tag{
						{
							Key:   aws.String("SubnetType"),
							Value: aws.String("private"),
						},
					},
				}).
				Return(&ec2.CreateTagsOutput{}, nil).
				Times(1),
			// Look into public subnets
			a.Mocks.API.EC2.EXPECT().
				DescribeSubnets(ctx, &ec2.DescribeSubnetsInput{
					Filters: []ec2Types.Filter{
						{
							Name:   aws.String("vpc-id"),
							Values: []string{vpcID},
						},
						{
							Name:   aws.String("tag:SubnetType"),
							Values: []string{"Utility"},
						},
					},
				}).
				Return(&ec2.DescribeSubnetsOutput{
					Subnets: []ec2Types.Subnet{{
						SubnetId: aws.String(publicSubnetID),
						Tags: []ec2Types.Tag{
							{
								Key:   aws.String("SubnetType"),
								Value: aws.String("Utility"),
							},
						},
					}},
				}, nil).
				Times(1),
			// Fix public tag
			a.Mocks.API.EC2.EXPECT().
				CreateTags(ctx, &ec2.CreateTagsInput{
					Resources: []string{publicSubnetID},
					Tags: []ec2Types.Tag{
						{
							Key:   aws.String("SubnetType"),
							Value: aws.String("public"),
						},
					},
				}).
				Return(&ec2.CreateTagsOutput{}, nil).
				Times(1),
		)

		logger := logrus.New()

		err := a.Mocks.AWS.FixSubnetTagsForVPC(vpcID, logger)
		a.Assert().NoError(err)
	})
}

func (a *AWSTestSuite) TestClaimVPCCreateCluster() {
	owner := "test"
	vpcID := "mock-vpc"
	privateSubnetID := "private-id1"
	publicSubnetID := "public-id1"
	masterSecurityGroupID := "master-sg-id"
	workerSecurityGroupID := "worker-sg-id"
	callsSecurityGroupID := "calls-sg-id"
	cidrBlock := "100.0.0.0/16"
	cluster := a.ClusterA
	ctx := context.TODO()

	a.Run("claim with specific vpc as primary cluster", func() {
		a.SetupTest()
		gomock.InOrder(
			// ClaimVPC
			a.Mocks.API.EC2.EXPECT().
				DescribeVpcs(ctx, &ec2.DescribeVpcsInput{
					VpcIds: []string{vpcID},
				}).
				Return(&ec2.DescribeVpcsOutput{
					Vpcs: []ec2Types.Vpc{{
						VpcId:     aws.String(vpcID),
						CidrBlock: aws.String(cidrBlock),
					}},
				}, nil).
				Times(1),
			// Look into private subnets
			a.Mocks.API.EC2.EXPECT().
				DescribeSubnets(ctx, &ec2.DescribeSubnetsInput{
					Filters: []ec2Types.Filter{
						{
							Name:   aws.String("vpc-id"),
							Values: []string{vpcID},
						},
						{
							Name:   aws.String("tag:SubnetType"),
							Values: []string{"Private"},
						},
					},
				}).
				Return(&ec2.DescribeSubnetsOutput{
					Subnets: []ec2Types.Subnet{},
				}, nil).
				Times(1),
			// Look into public subnets
			a.Mocks.API.EC2.EXPECT().
				DescribeSubnets(ctx, &ec2.DescribeSubnetsInput{
					Filters: []ec2Types.Filter{
						{
							Name:   aws.String("vpc-id"),
							Values: []string{vpcID},
						},
						{
							Name:   aws.String("tag:SubnetType"),
							Values: []string{"Utility"},
						},
					},
				}).
				Return(&ec2.DescribeSubnetsOutput{
					Subnets: []ec2Types.Subnet{},
				}, nil).
				Times(1),
			// getClusterResourcesForVPC
			a.Mocks.API.EC2.EXPECT().DescribeSubnets(ctx, &ec2.DescribeSubnetsInput{
				Filters: []ec2Types.Filter{
					{
						Name:   aws.String("vpc-id"),
						Values: []string{vpcID},
					},
					{
						Name:   aws.String("tag:SubnetType"),
						Values: []string{"private"},
					},
				},
			}).
				Return(&ec2.DescribeSubnetsOutput{
					Subnets: []ec2Types.Subnet{{
						SubnetId: aws.String(privateSubnetID),
					}},
				}, nil).
				Times(1),
			a.Mocks.API.EC2.EXPECT().DescribeSubnets(ctx, &ec2.DescribeSubnetsInput{
				Filters: []ec2Types.Filter{
					{
						Name:   aws.String("vpc-id"),
						Values: []string{vpcID},
					},
					{
						Name:   aws.String("tag:SubnetType"),
						Values: []string{"public"},
					},
				},
			}).
				Return(&ec2.DescribeSubnetsOutput{
					Subnets: []ec2Types.Subnet{{
						SubnetId: aws.String(publicSubnetID),
					}},
				}, nil).
				Times(1),
			a.Mocks.API.EC2.EXPECT().DescribeSecurityGroups(ctx, &ec2.DescribeSecurityGroupsInput{
				Filters: []ec2Types.Filter{
					{
						Name:   aws.String("vpc-id"),
						Values: []string{vpcID},
					},
					{
						Name:   aws.String("tag:NodeType"),
						Values: []string{"master"},
					},
				},
			}).
				Return(&ec2.DescribeSecurityGroupsOutput{
					SecurityGroups: []ec2Types.SecurityGroup{{
						GroupId: aws.String(masterSecurityGroupID),
					}},
				}, nil).
				Times(1),
			a.Mocks.API.EC2.EXPECT().DescribeSecurityGroups(ctx, &ec2.DescribeSecurityGroupsInput{
				Filters: []ec2Types.Filter{
					{
						Name:   aws.String("vpc-id"),
						Values: []string{vpcID},
					},
					{
						Name:   aws.String("tag:NodeType"),
						Values: []string{"worker"},
					},
				},
			}).
				Return(&ec2.DescribeSecurityGroupsOutput{
					SecurityGroups: []ec2Types.SecurityGroup{{
						GroupId: aws.String(workerSecurityGroupID),
					}},
				}, nil).
				Times(1),
			a.Mocks.API.EC2.EXPECT().DescribeSecurityGroups(ctx, &ec2.DescribeSecurityGroupsInput{
				Filters: []ec2Types.Filter{
					{
						Name:   aws.String("vpc-id"),
						Values: []string{vpcID},
					},
					{
						Name:   aws.String("tag:NodeType"),
						Values: []string{"calls"},
					},
				},
			}).
				Return(&ec2.DescribeSecurityGroupsOutput{
					SecurityGroups: []ec2Types.SecurityGroup{{
						GroupId: aws.String(callsSecurityGroupID),
					}},
				}, nil).
				Times(1),
			// claimVpc
			a.Mocks.API.EC2.EXPECT().DescribeVpcs(ctx, &ec2.DescribeVpcsInput{
				Filters: []ec2Types.Filter{
					{
						Name:   aws.String("vpc-id"),
						Values: []string{vpcID},
					},
					{
						Name:   aws.String(VpcAvailableTagKey),
						Values: []string{VpcAvailableTagValueTrue},
					},
					{
						Name:   aws.String(VpcClusterIDTagKey),
						Values: []string{VpcClusterIDTagValueNone},
					},
				},
			}).
				Return(&ec2.DescribeVpcsOutput{
					Vpcs: []ec2Types.Vpc{{
						VpcId:     aws.String(vpcID),
						CidrBlock: aws.String(cidrBlock),
					}},
				}, nil).
				Times(1),
			a.Mocks.API.EC2.EXPECT().CreateTags(ctx, &ec2.CreateTagsInput{
				Resources: []string{
					vpcID,
				},
				Tags: []ec2Types.Tag{
					{
						Key:   aws.String(trimTagPrefix(VpcAvailableTagKey)),
						Value: aws.String(VpcAvailableTagValueFalse),
					},
				},
			}).
				Return(nil, nil).
				Times(1),
			a.Mocks.API.EC2.EXPECT().CreateTags(ctx, &ec2.CreateTagsInput{
				Resources: []string{
					vpcID,
				},
				Tags: []ec2Types.Tag{
					{
						Key:   aws.String(trimTagPrefix(VpcClusterIDTagKey)),
						Value: aws.String(cluster.ID),
					},
				},
			}).
				Return(nil, nil).
				Times(1),
			a.Mocks.API.EC2.EXPECT().CreateTags(ctx, &ec2.CreateTagsInput{
				Resources: []string{
					vpcID,
				},
				Tags: []ec2Types.Tag{
					{
						Key:   aws.String(trimTagPrefix(VpcClusterOwnerKey)),
						Value: aws.String(owner),
					},
				},
			}).
				Return(nil, nil).
				Times(1),
			a.Mocks.API.EC2.EXPECT().CreateTags(ctx, gomock.Any()).
				Return(nil, nil).
				Times(3),
		)

		logger := logrus.New()

		clusterResources, err := a.Mocks.AWS.ClaimVPC(vpcID, cluster, owner, logger)
		a.Assert().NoError(err)
		a.Assert().Equal(clusterResources.VpcID, vpcID)
		a.Assert().Contains(clusterResources.PrivateSubnetIDs, privateSubnetID)
		a.Assert().Contains(clusterResources.PublicSubnetsIDs, publicSubnetID)
		a.Assert().Contains(clusterResources.MasterSecurityGroupIDs, masterSecurityGroupID)
		a.Assert().Contains(clusterResources.WorkerSecurityGroupIDs, workerSecurityGroupID)
		a.Assert().Contains(clusterResources.CallsSecurityGroupIDs, callsSecurityGroupID)
	})

	a.Run("claim with specific vpc as primary cluster", func() {
		a.SetupTest()
		gomock.InOrder(
			// ClaimVPC
			a.Mocks.API.EC2.EXPECT().
				DescribeVpcs(gomock.Any(), &ec2.DescribeVpcsInput{
					VpcIds: []string{vpcID},
				}).
				Return(&ec2.DescribeVpcsOutput{
					Vpcs: []ec2Types.Vpc{{
						VpcId:     aws.String(vpcID),
						CidrBlock: aws.String(cidrBlock),
					}},
				}, nil).
				Times(1),
			// Look into private subnets
			a.Mocks.API.EC2.EXPECT().
				DescribeSubnets(ctx, &ec2.DescribeSubnetsInput{
					Filters: []ec2Types.Filter{
						{
							Name:   aws.String("vpc-id"),
							Values: []string{vpcID},
						},
						{
							Name:   aws.String("tag:SubnetType"),
							Values: []string{"Private"},
						},
					},
				}).
				Return(&ec2.DescribeSubnetsOutput{
					Subnets: []ec2Types.Subnet{},
				}, nil).
				Times(1),
			// Look into public subnets
			a.Mocks.API.EC2.EXPECT().
				DescribeSubnets(ctx, &ec2.DescribeSubnetsInput{
					Filters: []ec2Types.Filter{
						{
							Name:   aws.String("vpc-id"),
							Values: []string{vpcID},
						},
						{
							Name:   aws.String("tag:SubnetType"),
							Values: []string{"Utility"},
						},
					},
				}).
				Return(&ec2.DescribeSubnetsOutput{
					Subnets: []ec2Types.Subnet{},
				}, nil).
				Times(1),
			// getClusterResourcesForVPC
			a.Mocks.API.EC2.EXPECT().DescribeSubnets(gomock.Any(), &ec2.DescribeSubnetsInput{
				Filters: []ec2Types.Filter{
					{
						Name:   aws.String("vpc-id"),
						Values: []string{vpcID},
					},
					{
						Name:   aws.String("tag:SubnetType"),
						Values: []string{"private"},
					},
				},
			}).
				Return(&ec2.DescribeSubnetsOutput{
					Subnets: []ec2Types.Subnet{{
						SubnetId: aws.String(privateSubnetID),
					}},
				}, nil).
				Times(1),
			a.Mocks.API.EC2.EXPECT().DescribeSubnets(gomock.Any(), &ec2.DescribeSubnetsInput{
				Filters: []ec2Types.Filter{
					{
						Name:   aws.String("vpc-id"),
						Values: []string{vpcID},
					},
					{
						Name:   aws.String("tag:SubnetType"),
						Values: []string{"public"},
					},
				},
			}).
				Return(&ec2.DescribeSubnetsOutput{
					Subnets: []ec2Types.Subnet{{
						SubnetId: aws.String(publicSubnetID),
					}},
				}, nil).
				Times(1),
			a.Mocks.API.EC2.EXPECT().DescribeSecurityGroups(gomock.Any(), &ec2.DescribeSecurityGroupsInput{
				Filters: []ec2Types.Filter{
					{
						Name:   aws.String("vpc-id"),
						Values: []string{vpcID},
					},
					{
						Name:   aws.String("tag:NodeType"),
						Values: []string{"master"},
					},
				},
			}).
				Return(&ec2.DescribeSecurityGroupsOutput{
					SecurityGroups: []ec2Types.SecurityGroup{{
						GroupId: aws.String(masterSecurityGroupID),
					}},
				}, nil).
				Times(1),
			a.Mocks.API.EC2.EXPECT().DescribeSecurityGroups(gomock.Any(), &ec2.DescribeSecurityGroupsInput{
				Filters: []ec2Types.Filter{
					{
						Name:   aws.String("vpc-id"),
						Values: []string{vpcID},
					},
					{
						Name:   aws.String("tag:NodeType"),
						Values: []string{"worker"},
					},
				},
			}).
				Return(&ec2.DescribeSecurityGroupsOutput{
					SecurityGroups: []ec2Types.SecurityGroup{{
						GroupId: aws.String(workerSecurityGroupID),
					}},
				}, nil).
				Times(1),
			a.Mocks.API.EC2.EXPECT().DescribeSecurityGroups(gomock.Any(), &ec2.DescribeSecurityGroupsInput{
				Filters: []ec2Types.Filter{
					{
						Name:   aws.String("vpc-id"),
						Values: []string{vpcID},
					},
					{
						Name:   aws.String("tag:NodeType"),
						Values: []string{"calls"},
					},
				},
			}).
				Return(&ec2.DescribeSecurityGroupsOutput{
					SecurityGroups: []ec2Types.SecurityGroup{{
						GroupId: aws.String(callsSecurityGroupID),
					}},
				}, nil).
				Times(1),
			// claimVpc
			a.Mocks.API.EC2.EXPECT().DescribeVpcs(gomock.Any(), &ec2.DescribeVpcsInput{
				Filters: []ec2Types.Filter{
					{
						Name:   aws.String("vpc-id"),
						Values: []string{vpcID},
					},
					{
						Name:   aws.String(VpcAvailableTagKey),
						Values: []string{VpcAvailableTagValueTrue},
					},
					{
						Name:   aws.String(VpcClusterIDTagKey),
						Values: []string{VpcClusterIDTagValueNone},
					},
				},
			}).
				Return(&ec2.DescribeVpcsOutput{
					Vpcs: []ec2Types.Vpc{{
						VpcId:     aws.String(vpcID),
						CidrBlock: aws.String(cidrBlock),
					}},
				}, nil).
				Times(1),
			a.Mocks.API.EC2.EXPECT().CreateTags(gomock.Any(), &ec2.CreateTagsInput{
				Resources: []string{vpcID},
				Tags: []ec2Types.Tag{
					{
						Key:   aws.String(trimTagPrefix(VpcAvailableTagKey)),
						Value: aws.String(VpcAvailableTagValueFalse),
					},
				},
			}).
				Return(nil, nil).
				Times(1),
			a.Mocks.API.EC2.EXPECT().CreateTags(gomock.Any(), &ec2.CreateTagsInput{
				Resources: []string{
					vpcID,
				},
				Tags: []ec2Types.Tag{
					{
						Key:   aws.String(trimTagPrefix(VpcClusterIDTagKey)),
						Value: aws.String(cluster.ID),
					},
				},
			}).
				Return(nil, nil).
				Times(1),
			a.Mocks.API.EC2.EXPECT().CreateTags(gomock.Any(), &ec2.CreateTagsInput{
				Resources: []string{vpcID},
				Tags: []ec2Types.Tag{
					{
						Key:   aws.String(trimTagPrefix(VpcClusterOwnerKey)),
						Value: aws.String(owner),
					},
				},
			}).
				Return(nil, nil).
				Times(1),
			a.Mocks.API.EC2.EXPECT().CreateTags(gomock.Any(), gomock.Any()).
				Return(nil, nil).
				Times(3),
		)

		logger := logrus.New()

		clusterResources, err := a.Mocks.AWS.ClaimVPC(vpcID, cluster, owner, logger)
		a.Assert().NoError(err)
		a.Assert().Equal(clusterResources.VpcID, vpcID)
		a.Assert().Contains(clusterResources.PrivateSubnetIDs, privateSubnetID)
		a.Assert().Contains(clusterResources.PublicSubnetsIDs, publicSubnetID)
		a.Assert().Contains(clusterResources.MasterSecurityGroupIDs, masterSecurityGroupID)
		a.Assert().Contains(clusterResources.WorkerSecurityGroupIDs, workerSecurityGroupID)
		a.Assert().Contains(clusterResources.CallsSecurityGroupIDs, callsSecurityGroupID)
	})

	a.Run("claim with specific vpc as secondary cluster", func() {
		a.SetupTest()
		gomock.InOrder(
			// ClaimVPC
			a.Mocks.API.EC2.EXPECT().
				DescribeVpcs(gomock.Any(), &ec2.DescribeVpcsInput{
					VpcIds: []string{vpcID},
				}).
				Return(&ec2.DescribeVpcsOutput{
					Vpcs: []ec2Types.Vpc{{
						VpcId:     aws.String(vpcID),
						CidrBlock: aws.String(cidrBlock),
					}},
				}, nil).
				Times(1),
			// Look into private subnets
			a.Mocks.API.EC2.EXPECT().
				DescribeSubnets(ctx, &ec2.DescribeSubnetsInput{
					Filters: []ec2Types.Filter{
						{
							Name:   aws.String("vpc-id"),
							Values: []string{vpcID},
						},
						{
							Name:   aws.String("tag:SubnetType"),
							Values: []string{"Private"},
						},
					},
				}).
				Return(&ec2.DescribeSubnetsOutput{
					Subnets: []ec2Types.Subnet{},
				}, nil).
				Times(1),
			// Look into public subnets
			a.Mocks.API.EC2.EXPECT().
				DescribeSubnets(ctx, &ec2.DescribeSubnetsInput{
					Filters: []ec2Types.Filter{
						{
							Name:   aws.String("vpc-id"),
							Values: []string{vpcID},
						},
						{
							Name:   aws.String("tag:SubnetType"),
							Values: []string{"Utility"},
						},
					},
				}).
				Return(&ec2.DescribeSubnetsOutput{
					Subnets: []ec2Types.Subnet{},
				}, nil).
				Times(1),
			// getClusterResourcesForVPC
			a.Mocks.API.EC2.EXPECT().DescribeSubnets(gomock.Any(), &ec2.DescribeSubnetsInput{
				Filters: []ec2Types.Filter{
					{
						Name:   aws.String("vpc-id"),
						Values: []string{vpcID},
					},
					{
						Name:   aws.String("tag:SubnetType"),
						Values: []string{"private"},
					},
				},
			}).
				Return(&ec2.DescribeSubnetsOutput{
					Subnets: []ec2Types.Subnet{{
						SubnetId: aws.String(privateSubnetID),
					}},
				}, nil).
				Times(1),
			a.Mocks.API.EC2.EXPECT().DescribeSubnets(gomock.Any(), &ec2.DescribeSubnetsInput{
				Filters: []ec2Types.Filter{
					{
						Name:   aws.String("vpc-id"),
						Values: []string{vpcID},
					},
					{
						Name:   aws.String("tag:SubnetType"),
						Values: []string{"public"},
					},
				},
			}).
				Return(&ec2.DescribeSubnetsOutput{
					Subnets: []ec2Types.Subnet{{
						SubnetId: aws.String(publicSubnetID),
					}},
				}, nil).
				Times(1),
			a.Mocks.API.EC2.EXPECT().DescribeSecurityGroups(gomock.Any(), &ec2.DescribeSecurityGroupsInput{
				Filters: []ec2Types.Filter{
					{
						Name:   aws.String("vpc-id"),
						Values: []string{vpcID},
					},
					{
						Name:   aws.String("tag:NodeType"),
						Values: []string{"master"},
					},
				},
			}).
				Return(&ec2.DescribeSecurityGroupsOutput{
					SecurityGroups: []ec2Types.SecurityGroup{{
						GroupId: aws.String(masterSecurityGroupID),
					}},
				}, nil).
				Times(1),
			a.Mocks.API.EC2.EXPECT().DescribeSecurityGroups(gomock.Any(), &ec2.DescribeSecurityGroupsInput{
				Filters: []ec2Types.Filter{
					{
						Name:   aws.String("vpc-id"),
						Values: []string{vpcID},
					},
					{
						Name:   aws.String("tag:NodeType"),
						Values: []string{"worker"},
					},
				},
			}).
				Return(&ec2.DescribeSecurityGroupsOutput{
					SecurityGroups: []ec2Types.SecurityGroup{{
						GroupId: aws.String(workerSecurityGroupID),
					}},
				}, nil).
				Times(1),
			a.Mocks.API.EC2.EXPECT().DescribeSecurityGroups(gomock.Any(), &ec2.DescribeSecurityGroupsInput{
				Filters: []ec2Types.Filter{
					{
						Name:   aws.String("vpc-id"),
						Values: []string{vpcID},
					},
					{
						Name:   aws.String("tag:NodeType"),
						Values: []string{"calls"},
					},
				},
			}).
				Return(&ec2.DescribeSecurityGroupsOutput{
					SecurityGroups: []ec2Types.SecurityGroup{{
						GroupId: aws.String(callsSecurityGroupID),
					}},
				}, nil).
				Times(1),
			// claimVpc
			a.Mocks.API.EC2.EXPECT().DescribeVpcs(gomock.Any(), &ec2.DescribeVpcsInput{
				Filters: []ec2Types.Filter{
					{
						Name:   aws.String("vpc-id"),
						Values: []string{vpcID},
					},
					{
						Name:   aws.String(VpcAvailableTagKey),
						Values: []string{VpcAvailableTagValueTrue},
					},
					{
						Name:   aws.String(VpcClusterIDTagKey),
						Values: []string{VpcClusterIDTagValueNone},
					},
				},
			}).
				Return(&ec2.DescribeVpcsOutput{
					Vpcs: []ec2Types.Vpc{},
				}, nil).
				Times(1),
			a.Mocks.API.EC2.EXPECT().DescribeVpcs(gomock.Any(), &ec2.DescribeVpcsInput{
				Filters: []ec2Types.Filter{
					{
						Name:   aws.String("vpc-id"),
						Values: []string{vpcID},
					},
					{
						Name:   aws.String(VpcAvailableTagKey),
						Values: []string{VpcAvailableTagValueFalse},
					},
					{
						Name:   aws.String(VpcSecondaryClusterIDTagKey),
						Values: []string{VpcClusterIDTagValueNone},
					},
				},
			}).
				Return(&ec2.DescribeVpcsOutput{
					Vpcs: []ec2Types.Vpc{{
						VpcId:     aws.String(vpcID),
						CidrBlock: aws.String(cidrBlock),
					}},
				}, nil).
				Times(1),
			a.Mocks.API.EC2.EXPECT().CreateTags(gomock.Any(), &ec2.CreateTagsInput{
				Resources: []string{vpcID},
				Tags: []ec2Types.Tag{
					{
						Key:   aws.String(trimTagPrefix(VpcSecondaryClusterIDTagKey)),
						Value: aws.String(cluster.ID),
					},
				},
			}).
				Return(nil, nil).
				Times(1),
			a.Mocks.API.EC2.EXPECT().CreateTags(gomock.Any(), gomock.Any()).
				Return(nil, nil).
				Times(3),
		)

		logger := logrus.New()

		clusterResources, err := a.Mocks.AWS.ClaimVPC(vpcID, cluster, owner, logger)
		a.Assert().NoError(err)
		a.Assert().Equal(clusterResources.VpcID, vpcID)
		a.Assert().Contains(clusterResources.PrivateSubnetIDs, privateSubnetID)
		a.Assert().Contains(clusterResources.PublicSubnetsIDs, publicSubnetID)
		a.Assert().Contains(clusterResources.MasterSecurityGroupIDs, masterSecurityGroupID)
		a.Assert().Contains(clusterResources.WorkerSecurityGroupIDs, workerSecurityGroupID)
		a.Assert().Contains(clusterResources.CallsSecurityGroupIDs, callsSecurityGroupID)
	})

	a.Run("claim with specific unavailable vpc", func() {
		a.SetupTest()
		gomock.InOrder(
			// ClaimVPC
			a.Mocks.API.EC2.EXPECT().
				DescribeVpcs(gomock.Any(), &ec2.DescribeVpcsInput{
					VpcIds: []string{vpcID},
				}).
				Return(&ec2.DescribeVpcsOutput{
					Vpcs: []ec2Types.Vpc{{
						VpcId:     aws.String(vpcID),
						CidrBlock: aws.String(cidrBlock),
					}},
				}, nil).
				Times(1),
			// Look into private subnets
			a.Mocks.API.EC2.EXPECT().
				DescribeSubnets(ctx, &ec2.DescribeSubnetsInput{
					Filters: []ec2Types.Filter{
						{
							Name:   aws.String("vpc-id"),
							Values: []string{vpcID},
						},
						{
							Name:   aws.String("tag:SubnetType"),
							Values: []string{"Private"},
						},
					},
				}).
				Return(&ec2.DescribeSubnetsOutput{
					Subnets: []ec2Types.Subnet{},
				}, nil).
				Times(1),
			// Look into public subnets
			a.Mocks.API.EC2.EXPECT().
				DescribeSubnets(ctx, &ec2.DescribeSubnetsInput{
					Filters: []ec2Types.Filter{
						{
							Name:   aws.String("vpc-id"),
							Values: []string{vpcID},
						},
						{
							Name:   aws.String("tag:SubnetType"),
							Values: []string{"Utility"},
						},
					},
				}).
				Return(&ec2.DescribeSubnetsOutput{
					Subnets: []ec2Types.Subnet{},
				}, nil).
				Times(1),
			// getClusterResourcesForVPC
			a.Mocks.API.EC2.EXPECT().DescribeSubnets(gomock.Any(), &ec2.DescribeSubnetsInput{
				Filters: []ec2Types.Filter{
					{
						Name:   aws.String("vpc-id"),
						Values: []string{vpcID},
					},
					{
						Name:   aws.String("tag:SubnetType"),
						Values: []string{"private"},
					},
				},
			}).
				Return(&ec2.DescribeSubnetsOutput{
					Subnets: []ec2Types.Subnet{{
						SubnetId: aws.String(privateSubnetID),
					}},
				}, nil).
				Times(1),
			a.Mocks.API.EC2.EXPECT().DescribeSubnets(gomock.Any(), &ec2.DescribeSubnetsInput{
				Filters: []ec2Types.Filter{
					{
						Name:   aws.String("vpc-id"),
						Values: []string{vpcID},
					},
					{
						Name:   aws.String("tag:SubnetType"),
						Values: []string{"public"},
					},
				},
			}).
				Return(&ec2.DescribeSubnetsOutput{
					Subnets: []ec2Types.Subnet{{
						SubnetId: aws.String(publicSubnetID),
					}},
				}, nil).
				Times(1),
			a.Mocks.API.EC2.EXPECT().DescribeSecurityGroups(gomock.Any(), &ec2.DescribeSecurityGroupsInput{
				Filters: []ec2Types.Filter{
					{
						Name:   aws.String("vpc-id"),
						Values: []string{vpcID},
					},
					{
						Name:   aws.String("tag:NodeType"),
						Values: []string{"master"},
					},
				},
			}).
				Return(&ec2.DescribeSecurityGroupsOutput{
					SecurityGroups: []ec2Types.SecurityGroup{{
						GroupId: aws.String(masterSecurityGroupID),
					}},
				}, nil).
				Times(1),
			a.Mocks.API.EC2.EXPECT().DescribeSecurityGroups(gomock.Any(), &ec2.DescribeSecurityGroupsInput{
				Filters: []ec2Types.Filter{
					{
						Name:   aws.String("vpc-id"),
						Values: []string{vpcID},
					},
					{
						Name:   aws.String("tag:NodeType"),
						Values: []string{"worker"},
					},
				},
			}).
				Return(&ec2.DescribeSecurityGroupsOutput{
					SecurityGroups: []ec2Types.SecurityGroup{{
						GroupId: aws.String(workerSecurityGroupID),
					}},
				}, nil).
				Times(1),
			a.Mocks.API.EC2.EXPECT().DescribeSecurityGroups(gomock.Any(), &ec2.DescribeSecurityGroupsInput{
				Filters: []ec2Types.Filter{
					{
						Name:   aws.String("vpc-id"),
						Values: []string{vpcID},
					},
					{
						Name:   aws.String("tag:NodeType"),
						Values: []string{"calls"},
					},
				},
			}).
				Return(&ec2.DescribeSecurityGroupsOutput{
					SecurityGroups: []ec2Types.SecurityGroup{{
						GroupId: aws.String(callsSecurityGroupID),
					}},
				}, nil).
				Times(1),
			// claimVpc
			a.Mocks.API.EC2.EXPECT().DescribeVpcs(gomock.Any(), &ec2.DescribeVpcsInput{
				Filters: []ec2Types.Filter{
					{
						Name:   aws.String("vpc-id"),
						Values: []string{vpcID},
					},
					{
						Name:   aws.String(VpcAvailableTagKey),
						Values: []string{VpcAvailableTagValueTrue},
					},
					{
						Name:   aws.String(VpcClusterIDTagKey),
						Values: []string{VpcClusterIDTagValueNone},
					},
				},
			}).
				Return(&ec2.DescribeVpcsOutput{
					Vpcs: []ec2Types.Vpc{},
				}, nil).
				Times(1),
			a.Mocks.API.EC2.EXPECT().DescribeVpcs(gomock.Any(), &ec2.DescribeVpcsInput{
				Filters: []ec2Types.Filter{
					{
						Name:   aws.String("vpc-id"),
						Values: []string{vpcID},
					},
					{
						Name:   aws.String(VpcAvailableTagKey),
						Values: []string{VpcAvailableTagValueFalse},
					},
					{
						Name:   aws.String(VpcSecondaryClusterIDTagKey),
						Values: []string{VpcClusterIDTagValueNone},
					},
				},
			}).
				Return(&ec2.DescribeVpcsOutput{
					Vpcs: []ec2Types.Vpc{},
				}, nil).
				Times(1),
		)

		logger := logrus.New()

		_, err := a.Mocks.AWS.ClaimVPC(vpcID, cluster, owner, logger)
		a.Assert().Error(err)
	})
}

func (a *AWSTestSuite) TestReleaseVPC() {
	vpcID := "mock-vpc"
	publicSubnetID := "public-id1"
	callsSecurityGroupID := "calls-sg-id"
	cluster := a.ClusterA

	a.Run("release primary cluster", func() {
		a.SetupTest()
		gomock.InOrder(
			// GetVpcsWithFilters
			a.Mocks.API.EC2.EXPECT().
				DescribeVpcs(gomock.Any(), &ec2.DescribeVpcsInput{
					Filters: []ec2Types.Filter{{
						Name:   aws.String(VpcAvailableTagKey),
						Values: []string{VpcAvailableTagValueFalse},
					}, {
						Name:   aws.String(VpcSecondaryClusterIDTagKey),
						Values: []string{cluster.ID},
					}},
				}).
				Return(&ec2.DescribeVpcsOutput{
					Vpcs: []ec2Types.Vpc{}, // Faking no VPCs as secondary cluster
				}, nil).
				Times(1),
			// GetVpcsWithFilters
			a.Mocks.API.EC2.EXPECT().
				DescribeVpcs(gomock.Any(), &ec2.DescribeVpcsInput{
					Filters: []ec2Types.Filter{{
						Name:   aws.String(VpcAvailableTagKey),
						Values: []string{VpcAvailableTagValueFalse},
					}, {
						Name:   aws.String(VpcClusterIDTagKey),
						Values: []string{cluster.ID},
					}},
				}).
				Return(&ec2.DescribeVpcsOutput{
					Vpcs: []ec2Types.Vpc{{
						VpcId: &vpcID,
						Tags: []ec2Types.Tag{{ // VPC has no secondary clusters
							Key:   aws.String(trimTagPrefix(VpcSecondaryClusterIDTagKey)),
							Value: aws.String(VpcClusterIDTagValueNone),
						}},
					}},
				}, nil).
				Times(1),
			// getClusterResourcesForVPC
			a.Mocks.API.EC2.EXPECT().DescribeSubnets(gomock.Any(), &ec2.DescribeSubnetsInput{
				Filters: []ec2Types.Filter{
					{
						Name:   aws.String("vpc-id"),
						Values: []string{vpcID},
					},
					{
						Name:   aws.String("tag:SubnetType"),
						Values: []string{"public"},
					},
				},
			}).
				Return(&ec2.DescribeSubnetsOutput{
					Subnets: []ec2Types.Subnet{{
						SubnetId: aws.String(publicSubnetID),
					}},
				}, nil).
				Times(1),
			a.Mocks.API.EC2.EXPECT().DeleteTags(gomock.Any(), &ec2.DeleteTagsInput{
				Resources: []string{publicSubnetID},
				Tags: []ec2Types.Tag{{
					Key:   aws.String(fmt.Sprintf("kubernetes.io/cluster/%s", getClusterTag(cluster))),
					Value: aws.String("shared"),
				}},
			}).
				// Return(&ec2.DeleteTagsOutput{}, gomock.Any()).
				Times(1),
			a.Mocks.API.EC2.EXPECT().DescribeSecurityGroups(gomock.Any(), &ec2.DescribeSecurityGroupsInput{
				Filters: []ec2Types.Filter{
					{
						Name:   aws.String("vpc-id"),
						Values: []string{vpcID},
					},
					{
						Name:   aws.String("tag:NodeType"),
						Values: []string{"calls"},
					},
				},
			}).
				Return(&ec2.DescribeSecurityGroupsOutput{
					SecurityGroups: []ec2Types.SecurityGroup{{
						GroupId: aws.String(callsSecurityGroupID),
					}},
				}, nil).
				Times(1),
			a.Mocks.API.EC2.EXPECT().DeleteTags(gomock.Any(), &ec2.DeleteTagsInput{
				Resources: []string{callsSecurityGroupID},
				Tags: []ec2Types.Tag{{
					Key:   aws.String(fmt.Sprintf("kubernetes.io/cluster/%s", getClusterTag(cluster))),
					Value: aws.String("shared"),
				}},
			}).
				Times(1),
			a.Mocks.API.EC2.EXPECT().DeleteTags(gomock.Any(), &ec2.DeleteTagsInput{
				Resources: []string{callsSecurityGroupID},
				Tags: []ec2Types.Tag{{
					Key:   aws.String("KubernetesCluster"),
					Value: aws.String(getClusterTag(cluster)),
				}},
			}).
				Times(1),
			a.Mocks.API.EC2.EXPECT().CreateTags(gomock.Any(), &ec2.CreateTagsInput{
				Resources: []string{vpcID},
				Tags: []ec2Types.Tag{{
					Key:   aws.String(trimTagPrefix(VpcClusterIDTagKey)),
					Value: aws.String(VpcClusterIDTagValueNone),
				}},
			}).
				// Return(gomock.Any()).
				Times(1),
			a.Mocks.API.EC2.EXPECT().CreateTags(gomock.Any(), &ec2.CreateTagsInput{
				Resources: []string{vpcID},
				Tags: []ec2Types.Tag{{
					Key:   aws.String(trimTagPrefix(VpcAvailableTagKey)),
					Value: aws.String(VpcAvailableTagValueTrue),
				}},
			}).
				// Return(gomock.Any()).
				Times(1),
			a.Mocks.API.EC2.EXPECT().CreateTags(gomock.Any(), &ec2.CreateTagsInput{
				Resources: []string{vpcID},
				Tags: []ec2Types.Tag{{
					Key:   aws.String(trimTagPrefix(VpcClusterOwnerKey)),
					Value: aws.String(VpcClusterIDTagValueNone),
				}},
			}).
				// Return(gomock.Any()).
				Times(1),
		)

		logger := logrus.New()

		err := a.Mocks.AWS.ReleaseVpc(cluster, logger)
		a.Assert().NoError(err)
	})

	a.Run("release secondary cluster", func() {
		a.SetupTest()
		gomock.InOrder(
			// GetVpcsWithFilters
			a.Mocks.API.EC2.EXPECT().
				DescribeVpcs(gomock.Any(), &ec2.DescribeVpcsInput{
					Filters: []ec2Types.Filter{{
						Name:   aws.String(VpcAvailableTagKey),
						Values: []string{VpcAvailableTagValueFalse},
					}, {
						Name:   aws.String(VpcSecondaryClusterIDTagKey),
						Values: []string{cluster.ID},
					}},
				}).
				Return(&ec2.DescribeVpcsOutput{
					Vpcs: []ec2Types.Vpc{{
						VpcId: &vpcID,
						Tags: []ec2Types.Tag{{ // VPC has no secondary clusters
							Key:   aws.String(trimTagPrefix(VpcSecondaryClusterIDTagKey)),
							Value: aws.String(cluster.ID),
						}},
					}},
				}, nil).
				Times(1),
			// getClusterResourcesForVPC
			a.Mocks.API.EC2.EXPECT().DescribeSubnets(gomock.Any(), &ec2.DescribeSubnetsInput{
				Filters: []ec2Types.Filter{
					{
						Name:   aws.String("vpc-id"),
						Values: []string{vpcID},
					},
					{
						Name:   aws.String("tag:SubnetType"),
						Values: []string{"public"},
					},
				},
			}).
				Return(&ec2.DescribeSubnetsOutput{
					Subnets: []ec2Types.Subnet{{
						SubnetId: aws.String(publicSubnetID),
					}},
				}, nil).
				Times(1),
			a.Mocks.API.EC2.EXPECT().DeleteTags(gomock.Any(), &ec2.DeleteTagsInput{
				Resources: []string{publicSubnetID},
				Tags: []ec2Types.Tag{{
					Key:   aws.String(fmt.Sprintf("kubernetes.io/cluster/%s", getClusterTag(cluster))),
					Value: aws.String("shared"),
				}},
			}).
				// Return(&ec2.DeleteTagsOutput{}, gomock.Any()).
				Times(1),
			a.Mocks.API.EC2.EXPECT().DescribeSecurityGroups(gomock.Any(), &ec2.DescribeSecurityGroupsInput{
				Filters: []ec2Types.Filter{
					{
						Name:   aws.String("vpc-id"),
						Values: []string{vpcID},
					},
					{
						Name:   aws.String("tag:NodeType"),
						Values: []string{"calls"},
					},
				},
			}).
				Return(&ec2.DescribeSecurityGroupsOutput{
					SecurityGroups: []ec2Types.SecurityGroup{{
						GroupId: aws.String(callsSecurityGroupID),
					}},
				}, nil).
				Times(1),
			a.Mocks.API.EC2.EXPECT().DeleteTags(gomock.Any(), &ec2.DeleteTagsInput{
				Resources: []string{callsSecurityGroupID},
				Tags: []ec2Types.Tag{{
					Key:   aws.String(fmt.Sprintf("kubernetes.io/cluster/%s", getClusterTag(cluster))),
					Value: aws.String("shared"),
				}},
			}).
				Times(1),
			a.Mocks.API.EC2.EXPECT().DeleteTags(gomock.Any(), &ec2.DeleteTagsInput{
				Resources: []string{callsSecurityGroupID},
				Tags: []ec2Types.Tag{{
					Key:   aws.String("KubernetesCluster"),
					Value: aws.String(getClusterTag(cluster)),
				}},
			}).
				Times(1),
			a.Mocks.API.EC2.EXPECT().CreateTags(gomock.Any(), &ec2.CreateTagsInput{
				Resources: []string{vpcID},
				Tags: []ec2Types.Tag{{
					Key:   aws.String(trimTagPrefix(VpcSecondaryClusterIDTagKey)),
					Value: aws.String(VpcClusterIDTagValueNone),
				}},
			}).
				// Return(gomock.Any()).
				Times(1),
		)

		logger := logrus.New()

		err := a.Mocks.AWS.ReleaseVpc(cluster, logger)
		a.Assert().NoError(err)
	})

	a.Run("release primary cluster and promote secondary cluster", func() {
		gomock.InOrder(
			// GetVpcsWithFilters
			a.Mocks.API.EC2.EXPECT().
				DescribeVpcs(gomock.Any(), &ec2.DescribeVpcsInput{
					Filters: []ec2Types.Filter{{
						Name:   aws.String(VpcAvailableTagKey),
						Values: []string{VpcAvailableTagValueFalse},
					}, {
						Name:   aws.String(VpcSecondaryClusterIDTagKey),
						Values: []string{cluster.ID},
					}},
				}).
				Return(&ec2.DescribeVpcsOutput{
					Vpcs: []ec2Types.Vpc{}, // Faking no VPCs as secondary cluster
				}, nil).
				Times(1),
			// GetVpcsWithFilters
			a.Mocks.API.EC2.EXPECT().
				DescribeVpcs(gomock.Any(), &ec2.DescribeVpcsInput{
					Filters: []ec2Types.Filter{{
						Name:   aws.String(VpcAvailableTagKey),
						Values: []string{VpcAvailableTagValueFalse},
					}, {
						Name:   aws.String(VpcClusterIDTagKey),
						Values: []string{cluster.ID},
					}},
				}).
				Return(&ec2.DescribeVpcsOutput{
					Vpcs: []ec2Types.Vpc{{
						VpcId: &vpcID,
						Tags: []ec2Types.Tag{{ // VPC has no secondary clusters
							Key:   aws.String(trimTagPrefix(VpcSecondaryClusterIDTagKey)),
							Value: aws.String(a.ClusterB.ID),
						}},
					}},
				}, nil).
				Times(1),
			// getClusterResourcesForVPC
			a.Mocks.API.EC2.EXPECT().DescribeSubnets(gomock.Any(), &ec2.DescribeSubnetsInput{
				Filters: []ec2Types.Filter{
					{
						Name:   aws.String("vpc-id"),
						Values: []string{vpcID},
					},
					{
						Name:   aws.String("tag:SubnetType"),
						Values: []string{"public"},
					},
				},
			}).
				Return(&ec2.DescribeSubnetsOutput{
					Subnets: []ec2Types.Subnet{{
						SubnetId: aws.String(publicSubnetID),
					}},
				}, nil).
				Times(1),
			a.Mocks.API.EC2.EXPECT().DeleteTags(gomock.Any(), &ec2.DeleteTagsInput{
				Resources: []string{publicSubnetID},
				Tags: []ec2Types.Tag{{
					Key:   aws.String(fmt.Sprintf("kubernetes.io/cluster/%s", getClusterTag(cluster))),
					Value: aws.String("shared"),
				}},
			}).
				// Return(&ec2.DeleteTagsOutput{}, gomock.Any()).
				Times(1),
			a.Mocks.API.EC2.EXPECT().DescribeSecurityGroups(gomock.Any(), &ec2.DescribeSecurityGroupsInput{
				Filters: []ec2Types.Filter{
					{
						Name:   aws.String("vpc-id"),
						Values: []string{vpcID},
					},
					{
						Name:   aws.String("tag:NodeType"),
						Values: []string{"calls"},
					},
				},
			}).
				Return(&ec2.DescribeSecurityGroupsOutput{
					SecurityGroups: []ec2Types.SecurityGroup{{
						GroupId: aws.String(callsSecurityGroupID),
					}},
				}, nil).
				Times(1),
			a.Mocks.API.EC2.EXPECT().DeleteTags(gomock.Any(), &ec2.DeleteTagsInput{
				Resources: []string{callsSecurityGroupID},
				Tags: []ec2Types.Tag{{
					Key:   aws.String(fmt.Sprintf("kubernetes.io/cluster/%s", getClusterTag(cluster))),
					Value: aws.String("shared"),
				}},
			}).
				Times(1),
			a.Mocks.API.EC2.EXPECT().DeleteTags(gomock.Any(), &ec2.DeleteTagsInput{
				Resources: []string{callsSecurityGroupID},
				Tags: []ec2Types.Tag{{
					Key:   aws.String("KubernetesCluster"),
					Value: aws.String(getClusterTag(cluster)),
				}},
			}).
				Times(1),
			a.Mocks.API.EC2.EXPECT().CreateTags(gomock.Any(), &ec2.CreateTagsInput{
				Resources: []string{vpcID},
				Tags: []ec2Types.Tag{{
					Key:   aws.String(trimTagPrefix(VpcClusterIDTagKey)),
					Value: aws.String(a.ClusterB.ID),
				}},
			}).
				Times(1),
			a.Mocks.API.EC2.EXPECT().CreateTags(gomock.Any(), &ec2.CreateTagsInput{
				Resources: []string{vpcID},
				Tags: []ec2Types.Tag{{
					Key:   aws.String(trimTagPrefix(VpcSecondaryClusterIDTagKey)),
					Value: aws.String(VpcClusterIDTagValueNone),
				}},
			}).
				Times(1),
		)

		logger := logrus.New()

		err := a.Mocks.AWS.ReleaseVpc(cluster, logger)
		a.Assert().NoError(err)
	})
}
