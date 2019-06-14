package aws

import (
	"errors"

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

	svc, err := a.api.getEC2Client()
	if err != nil {
		return err
	}

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

	logger.Debugf("AWS ec2 response: %s", resp)

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

	logger.Debugf("AWS ec2 response: %s", resp)

	return nil
}

func (api *apiInterface) getEC2Client() (*ec2.EC2, error) {
	sess, err := session.NewSession()
	if err != nil {
		return nil, err
	}

	return ec2.New(sess), nil
}

func (api *apiInterface) tagResource(svc *ec2.EC2, input *ec2.CreateTagsInput) (*ec2.CreateTagsOutput, error) {
	return svc.CreateTags(input)
}

func (api *apiInterface) untagResource(svc *ec2.EC2, input *ec2.DeleteTagsInput) (*ec2.DeleteTagsOutput, error) {
	return svc.DeleteTags(input)
}
