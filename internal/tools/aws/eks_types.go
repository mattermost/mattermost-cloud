// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.
//

package aws

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/service/eks"
)

// EKSAPI represents the series of calls we require from the AWS SDK v2 EKS Client
type EKSAPI interface {
	CreateAddon(ctx context.Context, params *eks.CreateAddonInput, optFns ...func(*eks.Options)) (*eks.CreateAddonOutput, error)
	DeleteAddon(ctx context.Context, params *eks.DeleteAddonInput, optFns ...func(*eks.Options)) (*eks.DeleteAddonOutput, error)

	CreateCluster(ctx context.Context, params *eks.CreateClusterInput, optFns ...func(*eks.Options)) (*eks.CreateClusterOutput, error)
	DeleteCluster(ctx context.Context, params *eks.DeleteClusterInput, optFns ...func(*eks.Options)) (*eks.DeleteClusterOutput, error)
	DescribeCluster(ctx context.Context, params *eks.DescribeClusterInput, optFns ...func(*eks.Options)) (*eks.DescribeClusterOutput, error)

	CreateNodegroup(ctx context.Context, params *eks.CreateNodegroupInput, optFns ...func(*eks.Options)) (*eks.CreateNodegroupOutput, error)
	DeleteNodegroup(ctx context.Context, params *eks.DeleteNodegroupInput, optFns ...func(*eks.Options)) (*eks.DeleteNodegroupOutput, error)
	DescribeNodegroup(ctx context.Context, params *eks.DescribeNodegroupInput, optFns ...func(*eks.Options)) (*eks.DescribeNodegroupOutput, error)
	ListNodegroups(ctx context.Context, params *eks.ListNodegroupsInput, optFns ...func(*eks.Options)) (*eks.ListNodegroupsOutput, error)
}
