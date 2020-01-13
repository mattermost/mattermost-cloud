package aws

import (
	"encoding/json"
	"errors"
	"strings"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/ec2"
	log "github.com/sirupsen/logrus"
)

// TagResource tags an AWS EC2 resource.
func (a *Client) TagResource(resourceID, key, value string, logger log.FieldLogger) error {
	if resourceID == "" {
		return errors.New("Missing resource ID")
	}

	svc := ec2.New(session.New(), &aws.Config{
		Region: aws.String(DefaultAWSRegion),
	})

	input := &ec2.CreateTagsInput{
		Resources: []*string{
			aws.String(resourceID),
		},
		Tags: []*ec2.Tag{
			&ec2.Tag{
				Key:   aws.String(key),
				Value: aws.String(value),
			},
		},
	}

	resp, err := a.api.tagResource(svc, input)
	if err != nil {
		return err
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
		return errors.New("Missing resource ID")
	}

	svc, err := a.api.getEC2Client()
	if err != nil {
		return err
	}

	input := &ec2.DeleteTagsInput{
		Resources: []*string{
			aws.String(resourceID),
		},
		Tags: []*ec2.Tag{
			&ec2.Tag{
				Key:   aws.String(key),
				Value: aws.String(value),
			},
		},
	}

	resp, err := a.api.untagResource(svc, input)
	if err != nil {
		return err
	}

	logger.WithFields(log.Fields{
		"tag-key":   key,
		"tag-value": value,
	}).Debugf("AWS EC2 delete tag response for %s: %s", resourceID, prettyDeleteTagsResponse(resp))

	return nil
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

func (api *apiInterface) getEC2Client() (*ec2.EC2, error) {
	svc := ec2.New(session.New(), &aws.Config{
		Region: aws.String(DefaultAWSRegion),
	})

	return svc, nil
}

func (api *apiInterface) tagResource(svc *ec2.EC2, input *ec2.CreateTagsInput) (*ec2.CreateTagsOutput, error) {
	return svc.CreateTags(input)
}

func (api *apiInterface) untagResource(svc *ec2.EC2, input *ec2.DeleteTagsInput) (*ec2.DeleteTagsOutput, error) {
	return svc.DeleteTags(input)
}

func (api *apiInterface) describeImages(svc *ec2.EC2, input *ec2.DescribeImagesInput) (*ec2.DescribeImagesOutput, error) {
	return svc.DescribeImages(input)
}

// GetVpcsWithFilters returns VPCs matching a given filter.
func GetVpcsWithFilters(filters []*ec2.Filter) ([]*ec2.Vpc, error) {
	svc := ec2.New(session.New(), &aws.Config{
		Region: aws.String(DefaultAWSRegion),
	})

	vpcOutput, err := svc.DescribeVpcs(&ec2.DescribeVpcsInput{
		Filters: filters,
	})
	if err != nil {
		return nil, err
	}

	return vpcOutput.Vpcs, nil
}

// GetSubnetsWithFilters returns subnets matching a given filter.
func GetSubnetsWithFilters(filters []*ec2.Filter) ([]*ec2.Subnet, error) {
	svc := ec2.New(session.New(), &aws.Config{
		Region: aws.String(DefaultAWSRegion),
	})

	subnetOutput, err := svc.DescribeSubnets(&ec2.DescribeSubnetsInput{
		Filters: filters,
	})
	if err != nil {
		return nil, err
	}

	return subnetOutput.Subnets, nil
}

// GetSecurityGroupsWithFilters returns SGs matching a given filter.
func GetSecurityGroupsWithFilters(filters []*ec2.Filter) ([]*ec2.SecurityGroup, error) {
	svc := ec2.New(session.New(), &aws.Config{
		Region: aws.String(DefaultAWSRegion),
	})

	sgOutput, err := svc.DescribeSecurityGroups(&ec2.DescribeSecurityGroupsInput{
		Filters: filters,
	})
	if err != nil {
		return nil, err
	}

	return sgOutput.SecurityGroups, nil
}

// IsValidAMI check if the provided AMI exists
func (a *Client) IsValidAMI(AMIImage string) bool {
	if AMIImage == "" {
		return false
	}

	svc := ec2.New(session.New(), &aws.Config{
		Region: aws.String(DefaultAWSRegion),
	})

	describeImageInput := &ec2.DescribeImagesInput{
		Filters: []*ec2.Filter{
			{
				Name:   aws.String("image-id"),
				Values: []*string{aws.String(AMIImage)},
			},
		},
	}

	out, err := a.api.describeImages(svc, describeImageInput)
	if err != nil {
		log.Errorf("failed to find ami %s. err=%s", AMIImage, err.Error())
		return false
	}
	if len(out.Images) == 0 {
		log.Errorf("found no AMIs with the ID %s. err=%s", AMIImage, err.Error())
		return false
	}

	return true
}
