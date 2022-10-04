// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.
//

package aws

import (
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/ec2"
)

// GetCIDRByVPCTag fetches VPC CIDR block by 'Name' tag.
func (a *Client) GetCIDRByVPCTag(vpcTagName string, logger log.FieldLogger) (string, error) {
	vpcInput := ec2.DescribeVpcsInput{
		Filters: []*ec2.Filter{
			{
				Name:   aws.String(VpcNameTagKey),
				Values: []*string{aws.String(vpcTagName)},
			},
		},
	}

	vpcOut, err := a.Service().ec2.DescribeVpcs(&vpcInput)
	if err != nil {
		return "", errors.Wrapf(err, "failed to fetch the VPC information using tag %s", vpcTagName)
	}

	if len(vpcOut.Vpcs) != 1 {
		return "", errors.Errorf("expected exactly one VPC in the list, got %d", len(vpcOut.Vpcs))
	}

	return *vpcOut.Vpcs[0].CidrBlock, nil
}
