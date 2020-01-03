package aws

import (
	"github.com/aws/aws-sdk-go/service/acm"
	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/aws/aws-sdk-go/service/route53"
	"github.com/mattermost/mattermost-cloud/model"
	log "github.com/sirupsen/logrus"
)

// AWS interface for use by other packages.
type AWS interface {
	GetCertificateSummaryByTag(key, value string) (*acm.CertificateSummary, error)

	GetAndClaimVpcResources(clusterID string, logger log.FieldLogger) (ClusterResources, error)
	ReleaseVpc(clusterID string, logger log.FieldLogger) error

	CreateCNAME(hostedZoneID, dnsName string, dnsEndpoints []string, logger log.FieldLogger) error
	CreatePrivateCNAME(dnsName string, dnsEndpoints []string, logger log.FieldLogger) error
	CreatePublicCNAME(dnsName string, dnsEndpoints []string, logger log.FieldLogger) error

	DeleteCNAME(hostedZoneID, dnsName string, logger log.FieldLogger) error
	DeletePrivateCNAME(dnsName string, logger log.FieldLogger) error
	DeletePublicCNAME(dnsName string, logger log.FieldLogger) error

	TagResource(resourceID, key, value string, logger log.FieldLogger) error
	UntagResource(resourceID, key, value string, logger log.FieldLogger) error
}

// Client is a client for interacting with AWS resources.
type Client struct {
	api   api
	store model.InstallationDatabaseStoreInterface
}

// api mocks out the AWS API calls for testing.
type api interface {
	getRoute53Client() (*route53.Route53, error)
	changeResourceRecordSets(*route53.Route53, *route53.ChangeResourceRecordSetsInput) (*route53.ChangeResourceRecordSetsOutput, error)
	listResourceRecordSets(*route53.Route53, *route53.ListResourceRecordSetsInput) (*route53.ListResourceRecordSetsOutput, error)
	listHostedZones(*route53.Route53, *route53.ListHostedZonesInput) (*route53.ListHostedZonesOutput, error)
	listTagsForResource(*route53.Route53, *route53.ListTagsForResourceInput) (*route53.ListTagsForResourceOutput, error)

	getEC2Client() (*ec2.EC2, error)
	tagResource(*ec2.EC2, *ec2.CreateTagsInput) (*ec2.CreateTagsOutput, error)
	untagResource(*ec2.EC2, *ec2.DeleteTagsInput) (*ec2.DeleteTagsOutput, error)

	getACMClient() (*acm.ACM, error)
	listCertificates(*acm.ACM, *acm.ListCertificatesInput) (*acm.ListCertificatesOutput, error)
	listTagsForCertificate(*acm.ACM, *acm.ListTagsForCertificateInput) (*acm.ListTagsForCertificateOutput, error)
}

// New returns a new AWS client.
func New() *Client {
	return &Client{
		api: &apiInterface{},
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
