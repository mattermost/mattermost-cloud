package aws

import (
	"github.com/aws/aws-sdk-go/aws/session"
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

var client *Client

// Client is a client for interacting with AWS resources.
type Client struct {
	store model.InstallationDatabaseStoreInterface

	acm            acmiface.ACMAPI
	ec2            ec2iface.EC2API
	iam            iamiface.IAMAPI
	rds            rdsiface.RDSAPI
	s3             s3iface.S3API
	route53        route53iface.Route53API
	secretsManager secretsmanageriface.SecretsManagerAPI
}

// NewAWSClient returns a new AWS client.
func NewAWSClient(sess *session.Session) *Client {
	if client != nil {
		return client
	}

	return &Client{
		acm:            acm.New(sess),
		ec2:            ec2.New(sess),
		iam:            iam.New(sess),
		rds:            rds.New(sess),
		s3:             s3.New(sess),
		route53:        route53.New(sess),
		secretsManager: secretsmanager.New(sess),
	}
}

// AddSQLStore adds SQLStore functionality to the AWS client.
func (c *Client) AddSQLStore(store model.InstallationDatabaseStoreInterface) {
	if !c.HasSQLStore() {
		c.store = store
	}
}

// HasSQLStore returns whether the AWS client has a SQL store or not.
func (c *Client) HasSQLStore() bool {
	return c.store != nil
}
