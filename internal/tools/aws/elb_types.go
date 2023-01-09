// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.
//

package aws

import (
	"context"

	elbv1 "github.com/aws/aws-sdk-go-v2/service/elasticloadbalancing"
	elbv2 "github.com/aws/aws-sdk-go-v2/service/elasticloadbalancingv2"
)

// LoadBalancerAPI holds a method to return right ELB API
type LoadBalancerAPI interface {
	GetLoadBalancerAPI(string) ELB
}

// ELB is an interface to access AWS resources
type ELB interface {
	GetLoadBalancerResource(name string) (string, error)
	TagLoadBalancer(arn string, tags map[string]string) error
}

// ELBV1 represents the series of calls we require from the AWS SDK v1 ELB Client
type ELBV1 interface {
	AddTags(ctx context.Context, params *elbv1.AddTagsInput, optFns ...func(*elbv1.Options)) (*elbv1.AddTagsOutput, error)
}

// ELBV2 represents the series of calls we require from the AWS SDK v2 ELB Client v2
type ELBV2 interface {
	DescribeLoadBalancers(ctx context.Context, params *elbv2.DescribeLoadBalancersInput, optFns ...func(*elbv2.Options)) (*elbv2.DescribeLoadBalancersOutput, error)
	AddTags(ctx context.Context, params *elbv2.AddTagsInput, optFns ...func(*elbv2.Options)) (*elbv2.AddTagsOutput, error)
}
