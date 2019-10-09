package aws

import (
	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/aws/aws-sdk-go/service/route53"
	log "github.com/sirupsen/logrus"
)

// AWS interface for use by other packages.
type AWS interface {
	CreateCNAME(dnsName string, dnsEndpoints []string, logger log.FieldLogger) error
	DeleteCNAME(dnsName string, logger log.FieldLogger) error

	TagResource(resourceID, key, value string, logger log.FieldLogger) error
	UntagResource(resourceID, key, value string, logger log.FieldLogger) error
}

// Client is a client for interacting with AWS resources.
type Client struct {
	hostedZoneID string
	api          api
}

var _ AWS = &Client{}

// api mocks out the AWS API calls for testing.
type api interface {
	getRoute53Client() (*route53.Route53, error)
	changeResourceRecordSets(*route53.Route53, *route53.ChangeResourceRecordSetsInput) (*route53.ChangeResourceRecordSetsOutput, error)
	listResourceRecordSets(*route53.Route53, *route53.ListResourceRecordSetsInput) (*route53.ListResourceRecordSetsOutput, error)

	getEC2Client() (*ec2.EC2, error)
	tagResource(*ec2.EC2, *ec2.CreateTagsInput) (*ec2.CreateTagsOutput, error)
	untagResource(*ec2.EC2, *ec2.DeleteTagsInput) (*ec2.DeleteTagsOutput, error)
}

// New returns a new AWS client.
func New(hostedZoneID string) *Client {
	return &Client{
		hostedZoneID: hostedZoneID,
		api:          &apiInterface{},
	}
}
