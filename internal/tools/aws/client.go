package aws

import (
	awsSession "github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/acm"
	"github.com/aws/aws-sdk-go/service/acm/acmiface"
	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/aws/aws-sdk-go/service/ec2/ec2iface"
	"github.com/aws/aws-sdk-go/service/iam"
	"github.com/aws/aws-sdk-go/service/iam/iamiface"
	"github.com/aws/aws-sdk-go/service/rds"
	"github.com/aws/aws-sdk-go/service/rds/rdsiface"
	"github.com/aws/aws-sdk-go/service/route53"
	"github.com/aws/aws-sdk-go/service/route53/route53iface"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/aws/aws-sdk-go/service/s3/s3iface"
	"github.com/aws/aws-sdk-go/service/secretsmanager"
	"github.com/aws/aws-sdk-go/service/secretsmanager/secretsmanageriface"
	"github.com/mattermost/mattermost-cloud/model"
	log "github.com/sirupsen/logrus"
)

// AWS interface for use by other packages.
type AWS interface {
	GetCertificateSummaryByTag(key, value string) (*acm.CertificateSummary, error)

	GetAndClaimVpcResources(clusterID, owner string, logger log.FieldLogger) (ClusterResources, error)
	ReleaseVpc(clusterID string, logger log.FieldLogger) error

	GetPrivateZoneDomainName(logger log.FieldLogger) (string, error)
	CreatePrivateCNAME(dnsName string, dnsEndpoints []string, logger log.FieldLogger) error
	CreatePublicCNAME(dnsName string, dnsEndpoints []string, logger log.FieldLogger) error

	DeletePrivateCNAME(dnsName string, logger log.FieldLogger) error
	DeletePublicCNAME(dnsName string, logger log.FieldLogger) error

	TagResource(resourceID, key, value string, logger log.FieldLogger) error
	UntagResource(resourceID, key, value string, logger log.FieldLogger) error
	IsValidAMI(AMIImage string) (bool, error)

	CreateDatabaseSnapshot(installationID string) error
}

// Client is a client for interacting with AWS resources.
type Client struct {
	api api

	store model.InstallationDatabaseStoreInterface

	acm            acmiface.ACMAPI
	ec2            ec2iface.EC2API
	iam            iamiface.IAMAPI
	rds            rdsiface.RDSAPI
	s3             s3iface.S3API
	route53        route53iface.Route53API
	secretsManager secretsmanageriface.SecretsManagerAPI
}

// api mocks out the AWS API calls for testing.
// TODO(gsagula): This should be deprecated in favour of the interfaces provided by AWS SDK.
type api interface {
	getRoute53Client() (*route53.Route53, error)
	changeResourceRecordSets(*route53.Route53, *route53.ChangeResourceRecordSetsInput) (*route53.ChangeResourceRecordSetsOutput, error)
	listResourceRecordSets(*route53.Route53, *route53.ListResourceRecordSetsInput) (*route53.ListResourceRecordSetsOutput, error)
	listHostedZones(*route53.Route53, *route53.ListHostedZonesInput) (*route53.ListHostedZonesOutput, error)
	listTagsForResource(*route53.Route53, *route53.ListTagsForResourceInput) (*route53.ListTagsForResourceOutput, error)

	getEC2Client() (*ec2.EC2, error)
	tagResource(*ec2.EC2, *ec2.CreateTagsInput) (*ec2.CreateTagsOutput, error)
	untagResource(*ec2.EC2, *ec2.DeleteTagsInput) (*ec2.DeleteTagsOutput, error)
	describeImages(svc *ec2.EC2, input *ec2.DescribeImagesInput) (*ec2.DescribeImagesOutput, error)

	getACMClient() (*acm.ACM, error)
	listCertificates(*acm.ACM, *acm.ListCertificatesInput) (*acm.ListCertificatesOutput, error)
	listTagsForCertificate(*acm.ACM, *acm.ListTagsForCertificateInput) (*acm.ListTagsForCertificateOutput, error)
}

// NewAWSClient returns a new AWS client.
func NewAWSClient(sess *awsSession.Session) *Client {
	return &Client{
		api: &apiInterface{},

		acm:            acm.New(sess),
		ec2:            ec2.New(sess),
		iam:            iam.New(sess),
		rds:            rds.New(sess),
		s3:             s3.New(sess),
		route53:        route53.New(sess),
		secretsManager: secretsmanager.New(sess),
	}
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
