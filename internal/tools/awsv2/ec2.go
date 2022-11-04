// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.
//

package awsv2

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	ec2Types "github.com/aws/aws-sdk-go-v2/service/ec2/types"
)

func (c *Client) getVPCFromTags(ctx context.Context, tags []Tag) ([]ec2Types.Vpc, error) {
	var filters []ec2Types.Filter

	for _, tag := range tags {
		filters = append(filters, ec2Types.Filter{
			Name:   aws.String(formatAsTagFilter(tag.Key)),
			Values: []string{tag.Value},
		})
	}

	vpcOutput, err := c.aws.ec2.DescribeVpcs(
		ctx,
		&ec2.DescribeVpcsInput{
			Filters: filters,
		},
	)
	if err != nil {
		return nil, err
	}

	return vpcOutput.Vpcs, nil
}
