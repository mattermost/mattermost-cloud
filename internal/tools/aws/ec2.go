package aws

import (
	"encoding/json"
	"strings"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
)

// TagResource tags an AWS EC2 resource.
func (a *Client) TagResource(resourceID, key, value string, logger log.FieldLogger) error {
	if resourceID == "" {
		return errors.New("Missing resource ID")
	}

	resp, err := a.Service().ec2.CreateTags(&ec2.CreateTagsInput{
		Resources: []*string{
			aws.String(resourceID),
		},
		Tags: []*ec2.Tag{
			&ec2.Tag{
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
	}).Debugf("AWS EC2 create tag response for %s: %s", resourceID, prettyCreateTagsResponse(resp))

	return nil
}

// UntagResource deletes tags from an AWS EC2 resource.
func (a *Client) UntagResource(resourceID, key, value string, logger log.FieldLogger) error {
	if resourceID == "" {
		return errors.New("unable to remove AWS tag from resource: missing resource ID")
	}

	resp, err := a.Service().ec2.DeleteTags(&ec2.DeleteTagsInput{
		Resources: []*string{
			aws.String(resourceID),
		},
		Tags: []*ec2.Tag{
			&ec2.Tag{
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
	}).Debugf("AWS EC2 delete tag response for %s: %s", resourceID, prettyDeleteTagsResponse(resp))

	return nil
}

// IsValidAMI check if the provided AMI exists
func (a *Client) IsValidAMI(AMIImage string, logger log.FieldLogger) (bool, error) {
	// if AMI image is blank it will use the default KOPS image
	if AMIImage == "" {
		return true, nil
	}

	out, err := a.Service().ec2.DescribeImages(&ec2.DescribeImagesInput{
		Filters: []*ec2.Filter{
			{
				Name:   aws.String("image-id"),
				Values: []*string{aws.String(AMIImage)},
			},
		},
	})
	if err != nil {
		return false, err
	}
	if len(out.Images) == 0 {
		return false, nil
	}

	return true, nil
}

// GetVpcsWithFilters returns VPCs matching a given filter.
func (a *Client) GetVpcsWithFilters(filters []*ec2.Filter) ([]*ec2.Vpc, error) {
	vpcOutput, err := a.Service().ec2.DescribeVpcs(&ec2.DescribeVpcsInput{
		Filters: filters,
	})
	if err != nil {
		return nil, err
	}

	return vpcOutput.Vpcs, nil
}

// GetSubnetsWithFilters returns subnets matching a given filter.
func (a *Client) GetSubnetsWithFilters(filters []*ec2.Filter) ([]*ec2.Subnet, error) {
	subnetOutput, err := a.Service().ec2.DescribeSubnets(&ec2.DescribeSubnetsInput{
		Filters: filters,
	})
	if err != nil {
		return nil, err
	}

	return subnetOutput.Subnets, nil
}

// GetSecurityGroupsWithFilters returns SGs matching a given filter.
func (a *Client) GetSecurityGroupsWithFilters(filters []*ec2.Filter) ([]*ec2.SecurityGroup, error) {
	sgOutput, err := a.Service().ec2.DescribeSecurityGroups(&ec2.DescribeSecurityGroupsInput{
		Filters: filters,
	})
	if err != nil {
		return nil, err
	}

	return sgOutput.SecurityGroups, nil
}

func prettyCreateTagsResponse(resp *ec2.CreateTagsOutput) string {
	prettyResp, err := json.Marshal(resp)
	if err != nil {
		return strings.Replace(resp.String(), "\n", " ", -1)
	}

	return string(prettyResp)
}

func prettyDeleteTagsResponse(resp *ec2.DeleteTagsOutput) string {
	prettyResp, err := json.Marshal(resp)
	if err != nil {
		return strings.Replace(resp.String(), "\n", " ", -1)
	}

	return string(prettyResp)
}
