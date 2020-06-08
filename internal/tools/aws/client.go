package aws

import (
	"sync"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/acm"
	"github.com/aws/aws-sdk-go/service/acm/acmiface"
	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/aws/aws-sdk-go/service/ec2/ec2iface"
	"github.com/aws/aws-sdk-go/service/iam"
	"github.com/aws/aws-sdk-go/service/iam/iamiface"
	"github.com/aws/aws-sdk-go/service/kms"
	"github.com/aws/aws-sdk-go/service/kms/kmsiface"
	"github.com/aws/aws-sdk-go/service/rds"
	"github.com/aws/aws-sdk-go/service/rds/rdsiface"
	"github.com/aws/aws-sdk-go/service/resourcegroupstaggingapi"
	"github.com/aws/aws-sdk-go/service/resourcegroupstaggingapi/resourcegroupstaggingapiiface"
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
	GetCertificateSummaryByTag(key, value string, logger log.FieldLogger) (*acm.CertificateSummary, error)

	GetAndClaimVpcResources(clusterID, owner string, logger log.FieldLogger) (ClusterResources, error)
	ReleaseVpc(clusterID string, logger log.FieldLogger) error

	GetPrivateZoneDomainName(logger log.FieldLogger) (string, error)
	CreatePrivateCNAME(dnsName string, dnsEndpoints []string, logger log.FieldLogger) error
	CreatePublicCNAME(dnsName string, dnsEndpoints []string, logger log.FieldLogger) error

	IsProvisionedPrivateCNAME(dnsName string, logger log.FieldLogger) bool
	DeletePrivateCNAME(dnsName string, logger log.FieldLogger) error
	DeletePublicCNAME(dnsName string, logger log.FieldLogger) error

	TagResource(resourceID, key, value string, logger log.FieldLogger) error
	UntagResource(resourceID, key, value string, logger log.FieldLogger) error
	IsValidAMI(AMIImage string, logger log.FieldLogger) (bool, error)

	GetAccountAliases() (*iam.ListAccountAliasesOutput, error)
}

// NewAWSClientWithConfig returns a new instance of Client with a custom configuration.
func NewAWSClientWithConfig(config *aws.Config, logger log.FieldLogger) *Client {
	return &Client{
		logger: logger,
		config: config,
		mux:    &sync.Mutex{},
	}
}

// Service hold AWS clients for each service.
type Service struct {
	acm                   acmiface.ACMAPI
	ec2                   ec2iface.EC2API
	iam                   iamiface.IAMAPI
	rds                   rdsiface.RDSAPI
	s3                    s3iface.S3API
	route53               route53iface.Route53API
	secretsManager        secretsmanageriface.SecretsManagerAPI
	resourceGroupsTagging resourcegroupstaggingapiiface.ResourceGroupsTaggingAPIAPI
	kms                   kmsiface.KMSAPI
}

// NewService creates a new instance of Service.
func NewService(sess *session.Session) *Service {
	return &Service{
		acm:                   acm.New(sess),
		iam:                   iam.New(sess),
		rds:                   rds.New(sess),
		s3:                    s3.New(sess),
		route53:               route53.New(sess),
		secretsManager:        secretsmanager.New(sess),
		resourceGroupsTagging: resourcegroupstaggingapi.New(sess),
		ec2:                   ec2.New(sess),
		kms:                   kms.New(sess),
	}
}

// Client is a client for interacting with AWS resources.
type Client struct {
	store   model.InstallationDatabaseStoreInterface
	logger  log.FieldLogger
	service *Service
	config  *aws.Config
	mux     *sync.Mutex
}

// Service contructs an AWS session if not yet successfully done and returns AWS clients.
func (c *Client) Service() *Service {
	if c.service == nil {
		sess, err := NewAWSSessionWithLogger(c.config, c.logger.WithField("tools-aws", "client"))
		if err != nil {
			c.logger.WithError(err).Error("failed to initialize AWS session")
			// Calls to AWS will fail until a healthy session is acquired.
			return NewService(&session.Session{})
		}

		c.mux.Lock()
		c.service = NewService(sess)
		c.mux.Unlock()
	}

	return c.service
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
