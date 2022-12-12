// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.
//

package aws

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	ec2Types "github.com/aws/aws-sdk-go-v2/service/ec2/types"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
)

// TagResource tags an AWS EC2 resource.
func (a *Client) TagResource(resourceID, key, value string, logger log.FieldLogger) error {
	ctx := context.TODO()

	if resourceID == "" {
		return errors.New("Missing resource ID")
	}

	output, err := a.Service().ec2.CreateTags(ctx, &ec2.CreateTagsInput{
		Resources: []string{resourceID},
		Tags: []ec2Types.Tag{
			{
				Key:   aws.String(key),
				Value: aws.String(value),
			},
		},
	})
	if err != nil {
		return errors.Wrapf(err, "unable to tag resource id: %s", resourceID)
	}

	logger.WithFields(log.Fields{
		"tag-key":   key,
		"tag-value": value,
	}).Debugf("AWS EC2 create tag response for %s: %s", resourceID, prettyCreateTagsResponse(output))

	return nil
}

// UntagResource deletes tags from an AWS EC2 resource.
func (a *Client) UntagResource(resourceID, key, value string, logger log.FieldLogger) error {
	ctx := context.TODO()

	if resourceID == "" {
		return errors.New("unable to remove AWS tag from resource: missing resource ID")
	}

	output, err := a.Service().ec2.DeleteTags(ctx, &ec2.DeleteTagsInput{
		Resources: []string{
			resourceID,
		},
		Tags: []ec2Types.Tag{
			{
				Key:   aws.String(key),
				Value: aws.String(value),
			},
		},
	})
	if err != nil {
		return errors.Wrap(err, "unable to remove AWS tag from resource")
	}

	logger.WithFields(log.Fields{
		"tag-key":   key,
		"tag-value": value,
	}).Debugf("AWS EC2 delete tag response for %s: %s", resourceID, prettyDeleteTagsResponse(output))

	return nil
}

// IsValidAMI check if the provided AMI exists
func (a *Client) IsValidAMI(AMIImage string, logger log.FieldLogger) (bool, error) {
	ctx := context.TODO()

	// if AMI image is blank it will use the default KOPS image
	if AMIImage == "" {
		return true, nil
	}

	output, err := a.Service().ec2.DescribeImages(ctx, &ec2.DescribeImagesInput{
		Filters: []ec2Types.Filter{
			{
				Name:   aws.String("image-id"),
				Values: []string{AMIImage},
			},
		},
	})
	if err != nil {
		return false, err
	}
	if len(output.Images) == 0 {
		return false, nil
	}

	return true, nil
}

// GetVpcsWithFilters returns VPCs matching a given filter.
func (a *Client) GetVpcsWithFilters(filters []ec2Types.Filter) ([]ec2Types.Vpc, error) {
	ctx := context.TODO()

	output, err := a.Service().ec2.DescribeVpcs(ctx, &ec2.DescribeVpcsInput{
		Filters: filters,
	})
	if err != nil {
		return nil, err
	}

	return output.Vpcs, nil
}

// GetSubnetsWithFilters returns subnets matching a given filter.
func (a *Client) GetSubnetsWithFilters(filters []ec2Types.Filter) ([]ec2Types.Subnet, error) {
	ctx := context.TODO()

	output, err := a.Service().ec2.DescribeSubnets(ctx, &ec2.DescribeSubnetsInput{
		Filters: filters,
	})
	if err != nil {
		return nil, err
	}

	return output.Subnets, nil
}

// GetSecurityGroupsWithFilters returns SGs matching a given filter.
func (a *Client) GetSecurityGroupsWithFilters(filters []ec2Types.Filter) ([]ec2Types.SecurityGroup, error) {
	ctx := context.TODO()

	output, err := a.Service().ec2.DescribeSecurityGroups(ctx, &ec2.DescribeSecurityGroupsInput{
		Filters: filters,
	})
	if err != nil {
		return nil, err
	}

	return output.SecurityGroups, nil
}

func prettyCreateTagsResponse(output *ec2.CreateTagsOutput) string {
	prettyResp, err := json.Marshal(output)
	if err != nil {
		return fmt.Sprintf("%v", output)
	}

	return string(prettyResp)
}

func prettyDeleteTagsResponse(output *ec2.DeleteTagsOutput) string {
	prettyResp, err := json.Marshal(output)
	if err != nil {
		return fmt.Sprintf("%v", output)
	}

	return string(prettyResp)
}
