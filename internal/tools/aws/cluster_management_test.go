// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.
//

package aws

import (
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/golang/mock/gomock"
	"github.com/sirupsen/logrus"
)

func (a *AWSTestSuite) TestClaimVPC() {
	owner := "test"
	vpcID := "mock-vpc"
	privateSubnetID := "private-id1"
	publicSubnetID := "public-id1"
	masterSecurityGroupID := "master-sg-id"
	workerSecurityGroupID := "worker-sg-id"
	callsSecurityGroupID := "calls-sg-id"
	cidrBlock := "100.0.0.0/16"
	cluster := a.ClusterA

	gomock.InOrder(
		// ClaimVPC
		a.Mocks.API.EC2.EXPECT().
			DescribeVpcs(&ec2.DescribeVpcsInput{
				VpcIds: aws.StringSlice([]string{vpcID}),
			}).
			Return(&ec2.DescribeVpcsOutput{
				Vpcs: []*ec2.Vpc{{
					VpcId:     aws.String(vpcID),
					CidrBlock: aws.String(cidrBlock),
				}},
			}, nil).
			Times(1),
		// getClusterResourcesForVPC
		a.Mocks.API.EC2.EXPECT().DescribeSubnets(&ec2.DescribeSubnetsInput{
			Filters: []*ec2.Filter{
				{
					Name:   aws.String("vpc-id"),
					Values: aws.StringSlice([]string{vpcID}),
				},
				{
					Name:   aws.String("tag:SubnetType"),
					Values: aws.StringSlice([]string{"private"}),
				},
			},
		}).
			Return(&ec2.DescribeSubnetsOutput{
				Subnets: []*ec2.Subnet{{
					SubnetId: aws.String(privateSubnetID),
				}},
			}, nil).
			Times(1),
		a.Mocks.API.EC2.EXPECT().DescribeSubnets(&ec2.DescribeSubnetsInput{
			Filters: []*ec2.Filter{
				{
					Name:   aws.String("vpc-id"),
					Values: aws.StringSlice([]string{vpcID}),
				},
				{
					Name:   aws.String("tag:SubnetType"),
					Values: aws.StringSlice([]string{"public"}),
				},
			},
		}).
			Return(&ec2.DescribeSubnetsOutput{
				Subnets: []*ec2.Subnet{{
					SubnetId: aws.String(publicSubnetID),
				}},
			}, nil).
			Times(1),
		a.Mocks.API.EC2.EXPECT().DescribeSecurityGroups(&ec2.DescribeSecurityGroupsInput{
			Filters: []*ec2.Filter{
				{
					Name:   aws.String("vpc-id"),
					Values: aws.StringSlice([]string{vpcID}),
				},
				{
					Name:   aws.String("tag:NodeType"),
					Values: aws.StringSlice([]string{"master"}),
				},
			},
		}).
			Return(&ec2.DescribeSecurityGroupsOutput{
				SecurityGroups: []*ec2.SecurityGroup{{
					GroupId: aws.String(masterSecurityGroupID),
				}},
			}, nil).
			Times(1),
		a.Mocks.API.EC2.EXPECT().DescribeSecurityGroups(&ec2.DescribeSecurityGroupsInput{
			Filters: []*ec2.Filter{
				{
					Name:   aws.String("vpc-id"),
					Values: aws.StringSlice([]string{vpcID}),
				},
				{
					Name:   aws.String("tag:NodeType"),
					Values: aws.StringSlice([]string{"worker"}),
				},
			},
		}).
			Return(&ec2.DescribeSecurityGroupsOutput{
				SecurityGroups: []*ec2.SecurityGroup{{
					GroupId: aws.String(workerSecurityGroupID),
				}},
			}, nil).
			Times(1),
		a.Mocks.API.EC2.EXPECT().DescribeSecurityGroups(&ec2.DescribeSecurityGroupsInput{
			Filters: []*ec2.Filter{
				{
					Name:   aws.String("vpc-id"),
					Values: aws.StringSlice([]string{vpcID}),
				},
				{
					Name:   aws.String("tag:NodeType"),
					Values: aws.StringSlice([]string{"calls"}),
				},
			},
		}).
			Return(&ec2.DescribeSecurityGroupsOutput{
				SecurityGroups: []*ec2.SecurityGroup{{
					GroupId: aws.String(callsSecurityGroupID),
				}},
			}, nil).
			Times(1),
		// claimVpc
		a.Mocks.API.EC2.EXPECT().DescribeVpcs(&ec2.DescribeVpcsInput{
			Filters: []*ec2.Filter{
				{
					Name:   aws.String("vpc-id"),
					Values: aws.StringSlice([]string{vpcID}),
				},
				{
					Name:   aws.String(VpcAvailableTagKey),
					Values: aws.StringSlice([]string{VpcAvailableTagValueTrue}),
				},
				{
					Name:   aws.String(VpcClusterIDTagKey),
					Values: aws.StringSlice([]string{VpcClusterIDTagValueNone}),
				},
			},
		}).
			Return(&ec2.DescribeVpcsOutput{
				Vpcs: []*ec2.Vpc{{
					VpcId:     aws.String(vpcID),
					CidrBlock: aws.String(cidrBlock),
				}},
			}, nil).
			Times(1),
		a.Mocks.API.EC2.EXPECT().CreateTags(&ec2.CreateTagsInput{
			Resources: []*string{
				aws.String(vpcID),
			},
			Tags: []*ec2.Tag{
				{
					Key:   aws.String(trimTagPrefix(VpcAvailableTagKey)),
					Value: aws.String(VpcAvailableTagValueFalse),
				},
			},
		}).
			Return(nil, nil).
			Times(1),
		a.Mocks.API.EC2.EXPECT().CreateTags(&ec2.CreateTagsInput{
			Resources: []*string{
				aws.String(vpcID),
			},
			Tags: []*ec2.Tag{
				{
					Key:   aws.String(trimTagPrefix(VpcClusterIDTagKey)),
					Value: aws.String(cluster.ID),
				},
			},
		}).
			Return(nil, nil).
			Times(1),
		a.Mocks.API.EC2.EXPECT().CreateTags(&ec2.CreateTagsInput{
			Resources: []*string{
				aws.String(vpcID),
			},
			Tags: []*ec2.Tag{
				{
					Key:   aws.String(trimTagPrefix(VpcClusterOwnerKey)),
					Value: aws.String(owner),
				},
			},
		}).
			Return(nil, nil).
			Times(1),
		a.Mocks.API.EC2.EXPECT().CreateTags(gomock.Any()).
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
}
