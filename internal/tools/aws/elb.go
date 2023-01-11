package aws

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/aws"
	elbv1 "github.com/aws/aws-sdk-go-v2/service/elasticloadbalancing"
	elbv1Type "github.com/aws/aws-sdk-go-v2/service/elasticloadbalancing/types"
	elbv2 "github.com/aws/aws-sdk-go-v2/service/elasticloadbalancingv2"
	elbv2Type "github.com/aws/aws-sdk-go-v2/service/elasticloadbalancingv2/types"
)

type elasticLoadbalancer struct {
	elasticLoadbalancerV1
	elasticLoadbalancerV2
}

func newElasticLoadbalancerFromConfig(cfg aws.Config) elasticLoadbalancer {
	return elasticLoadbalancer{
		elasticLoadbalancerV1: elasticLoadbalancerV1{
			elbv1.NewFromConfig(cfg),
		},
		elasticLoadbalancerV2: elasticLoadbalancerV2{
			elbv2.NewFromConfig(cfg),
		},
	}
}

// GetLoadBalancerAPIByType returns the correct ELB API based on elb type
func (c *Client) GetLoadBalancerAPIByType(elbType string) ELB {
	if elbType == "nlb" {
		return c.service.elb.elasticLoadbalancerV2
	}
	return c.service.elb.elasticLoadbalancerV1
}

type elasticLoadbalancerV1 struct {
	ELBV1
}

// GetLoadBalancerResource does nothing
func (e elasticLoadbalancerV1) GetLoadBalancerResource(name string) (string, error) {
	return name, nil
}

// TagLoadBalancer adds tags to the ELB
func (e elasticLoadbalancerV1) TagLoadBalancer(name string, tags map[string]string) error {
	var elbTags []elbv1Type.Tag
	for key, value := range tags {
		elbTags = append(elbTags, elbv1Type.Tag{
			Key:   aws.String(key),
			Value: aws.String(value),
		})
	}

	_, err := e.AddTags(context.Background(), &elbv1.AddTagsInput{
		LoadBalancerNames: []string{name},
		Tags:              elbTags,
	})
	return err
}

type elasticLoadbalancerV2 struct {
	ELBV2
}

// GetLoadBalancerResource returns the ARN of the ELB
func (e elasticLoadbalancerV2) GetLoadBalancerResource(name string) (string, error) {
	loadBalancer, err := e.DescribeLoadBalancers(context.Background(), &elbv2.DescribeLoadBalancersInput{
		Names: []string{name},
	})
	if err != nil {
		return "", err
	}

	if len(loadBalancer.LoadBalancers) == 0 || loadBalancer.LoadBalancers[0].LoadBalancerArn == nil {
		return "", nil
	}

	return *loadBalancer.LoadBalancers[0].LoadBalancerArn, nil
}

// TagLoadBalancer adds tags to the ELB
func (e elasticLoadbalancerV2) TagLoadBalancer(arn string, tags map[string]string) error {
	var elbTags []elbv2Type.Tag
	for key, value := range tags {
		elbTags = append(elbTags, elbv2Type.Tag{
			Key:   aws.String(key),
			Value: aws.String(value),
		})
	}

	_, err := e.AddTags(context.Background(), &elbv2.AddTagsInput{
		ResourceArns: []string{arn},
		Tags:         elbTags,
	})
	return err
}
