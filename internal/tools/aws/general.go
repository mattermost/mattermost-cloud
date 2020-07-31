// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.
//

package aws

import (
	"github.com/pkg/errors"

	"github.com/aws/aws-sdk-go/service/ec2"
)

// getAvailabilityZones retrive the Availabitly zones for the AWS region set in the Client.
func (a *Client) getAvailabilityZones() ([]*string, error) {
	resp, err := a.Service().ec2.DescribeAvailabilityZones(&ec2.DescribeAvailabilityZonesInput{})
	if err != nil {
		return nil, errors.Wrapf(err, "unable to get the AWS availabity zones for region %s", *a.config.Region)
	}

	azs := []*string{}
	for _, az := range resp.AvailabilityZones {
		azs = append(azs, az.ZoneName)
	}

	return azs, nil
}
