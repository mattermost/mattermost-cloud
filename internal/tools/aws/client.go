package aws

import (
	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/aws/aws-sdk-go/service/route53"
	"github.com/mattermost/mattermost-cloud/model"
	log "github.com/sirupsen/logrus"
)

// AWS interface for use by other packages.
type AWS interface {
	GetAndClaimVpcResources(clusterID string, logger log.FieldLogger) (ClusterResources, error)
	ReleaseVpc(clusterID string, logger log.FieldLogger) error

	CreateCNAME(dnsName string, dnsEndpoints []string, logger log.FieldLogger) error
	DeleteCNAME(dnsName string, logger log.FieldLogger) error

	TagResource(resourceID, key, value string, logger log.FieldLogger) error
	UntagResource(resourceID, key, value string, logger log.FieldLogger) error
	IsValidAMI(AMIImage string) (bool, error)
}

// Client is a client for interacting with AWS resources.
type Client struct {
	hostedZoneID string
	api          api
	store        model.InstallationDatabaseStoreInterface
}

// api mocks out the AWS API calls for testing.
type api interface {
	getRoute53Client() (*route53.Route53, error)
	changeResourceRecordSets(*route53.Route53, *route53.ChangeResourceRecordSetsInput) (*route53.ChangeResourceRecordSetsOutput, error)
	listResourceRecordSets(*route53.Route53, *route53.ListResourceRecordSetsInput) (*route53.ListResourceRecordSetsOutput, error)

	getEC2Client() (*ec2.EC2, error)
	tagResource(*ec2.EC2, *ec2.CreateTagsInput) (*ec2.CreateTagsOutput, error)
	untagResource(*ec2.EC2, *ec2.DeleteTagsInput) (*ec2.DeleteTagsOutput, error)
	describeImages(svc *ec2.EC2, input *ec2.DescribeImagesInput) (*ec2.DescribeImagesOutput, error)
}

// New returns a new AWS client.
func New(hostedZoneID string) *Client {
	return &Client{
		hostedZoneID: hostedZoneID,
		api:          &apiInterface{},
	}
}

// AddSQLStore adds SQLStore functionality to the AWS client.
func (c *Client) AddSQLStore(store model.InstallationDatabaseStoreInterface) {
	c.store = store
}

// HasSQLStore returns whether the AWS client has a SQL store or not.
func (c *Client) HasSQLStore() bool {
	return c.store != nil
}
