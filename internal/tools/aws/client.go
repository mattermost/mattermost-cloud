// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.
//

package aws

import (
	"strings"
	"sync"

	"github.com/aws/aws-sdk-go/service/applicationautoscaling"
	"github.com/aws/aws-sdk-go/service/applicationautoscaling/applicationautoscalingiface"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/acm"
	"github.com/aws/aws-sdk-go/service/acm/acmiface"
	"github.com/aws/aws-sdk-go/service/dynamodb"
	"github.com/aws/aws-sdk-go/service/dynamodb/dynamodbiface"
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
	"github.com/aws/aws-sdk-go/service/sts"
	"github.com/aws/aws-sdk-go/service/sts/stsiface"
	"github.com/mattermost/mattermost-cloud/model"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
	corev1 "k8s.io/api/core/v1"
)

// AWS interface for use by other packages.
type AWS interface {
	GetCertificateSummaryByTag(key, value string, logger log.FieldLogger) (*acm.CertificateSummary, error)

	GetAccountAliases() (*iam.ListAccountAliasesOutput, error)
	GetCloudEnvironmentName() (string, error)

	GetAndClaimVpcResources(clusterID, owner string, logger log.FieldLogger) (ClusterResources, error)
	GetVpcResources(clusterID string, logger log.FieldLogger) (ClusterResources, error)
	ReleaseVpc(clusterID string, logger log.FieldLogger) error
	AttachPolicyToRole(roleName, policyName string, logger log.FieldLogger) error
	DetachPolicyFromRole(roleName, policyName string, logger log.FieldLogger) error

	GetPrivateZoneDomainName(logger log.FieldLogger) (string, error)
	GetPrivateZoneIDForDefaultTag(logger log.FieldLogger) (string, error)
	GetTagByKeyAndZoneID(key string, id string, logger log.FieldLogger) (*Tag, error)

	CreatePrivateCNAME(dnsName string, dnsEndpoints []string, logger log.FieldLogger) error
	CreatePublicCNAME(dnsName string, dnsEndpoints []string, logger log.FieldLogger) error
	IsProvisionedPrivateCNAME(dnsName string, logger log.FieldLogger) bool
	DeletePrivateCNAME(dnsName string, logger log.FieldLogger) error
	DeletePublicCNAME(dnsName string, logger log.FieldLogger) error

	TagResource(resourceID, key, value string, logger log.FieldLogger) error
	UntagResource(resourceID, key, value string, logger log.FieldLogger) error
	IsValidAMI(AMIImage string, logger log.FieldLogger) (bool, error)

	DynamoDBEnsureTableDeleted(tableName string, logger log.FieldLogger) error
	S3EnsureBucketDeleted(bucketName string, logger log.FieldLogger) error

	GenerateBifrostUtilitySecret(clusterID string, logger log.FieldLogger) (*corev1.Secret, error)
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
	dynamodb              dynamodbiface.DynamoDBAPI
	sts                   stsiface.STSAPI
	appAutoscaling        applicationautoscalingiface.ApplicationAutoScalingAPI
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
		dynamodb:              dynamodb.New(sess),
		sts:                   sts.New(sess),
		appAutoscaling:        applicationautoscaling.New(sess),
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

// Helpers

// GetCloudEnvironmentName looks for a standard cloud account environment name
// and returns it.
func (c *Client) GetCloudEnvironmentName() (string, error) {
	accountAliases, err := c.GetAccountAliases()
	if err != nil {
		return "", errors.Wrap(err, "failed to get account aliases")
	}
	if len(accountAliases.AccountAliases) < 1 {
		return "", errors.New("account alias not defined")
	}

	for _, alias := range accountAliases.AccountAliases {
		if strings.HasPrefix(*alias, "mattermost-cloud") && len(strings.Split(*alias, "-")) == 3 {
			envName := strings.Split(*alias, "-")[2]
			if len(envName) == 0 {
				return "", errors.New("environment name value was empty")
			}

			return envName, nil
		}
	}

	return "", errors.New("account environment name could not be found from account aliases")
}
