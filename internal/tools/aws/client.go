// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.
//

package aws

import (
	"context"
	"strings"
	"sync"

	"github.com/aws/aws-sdk-go-v2/service/acm"
	"github.com/aws/aws-sdk-go/service/eks"
	"github.com/aws/aws-sdk-go/service/eks/eksiface"

	"github.com/aws/aws-sdk-go/service/applicationautoscaling"
	"github.com/aws/aws-sdk-go/service/applicationautoscaling/applicationautoscalingiface"

	awsv2 "github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
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
	GetCertificateSummaryByTag(key, value string, logger log.FieldLogger) (*model.Certificate, error)

	GetCloudEnvironmentName() string

	GetAndClaimVpcResources(cluster *model.Cluster, owner string, logger log.FieldLogger) (ClusterResources, error)
	ClaimVPC(vpcID string, cluster *model.Cluster, owner string, logger log.FieldLogger) (ClusterResources, error)
	GetVpcResources(clusterID string, logger log.FieldLogger) (ClusterResources, error)
	ReleaseVpc(cluster *model.Cluster, logger log.FieldLogger) error
	AttachPolicyToRole(roleName, policyName string, logger log.FieldLogger) error
	DetachPolicyFromRole(roleName, policyName string, logger log.FieldLogger) error

	GetPrivateZoneDomainName(logger log.FieldLogger) (string, error)
	GetPrivateHostedZoneID() string
	GetPublicHostedZoneNames() []string
	GetTagByKeyAndZoneID(key string, id string, logger log.FieldLogger) (*Tag, error)

	CreatePrivateCNAME(dnsName string, dnsEndpoints []string, logger log.FieldLogger) error
	CreatePublicCNAME(dnsName string, dnsEndpoints []string, dnsIdentifier string, logger log.FieldLogger) error
	UpdatePublicRecordIDForCNAME(dnsName, newID string, logger log.FieldLogger) error
	IsProvisionedPrivateCNAME(dnsName string, logger log.FieldLogger) bool
	DeletePrivateCNAME(dnsName string, logger log.FieldLogger) error
	DeletePublicCNAME(dnsName string, logger log.FieldLogger) error
	DeletePublicCNAMEs(dnsName []string, logger log.FieldLogger) error
	UpsertPublicCNAMEs(dnsNames []string, endpoints []string, logger log.FieldLogger) error

	TagResource(resourceID, key, value string, logger log.FieldLogger) error
	UntagResource(resourceID, key, value string, logger log.FieldLogger) error
	IsValidAMI(AMIImage string, logger log.FieldLogger) (bool, error)

	DynamoDBEnsureTableDeleted(tableName string, logger log.FieldLogger) error
	S3EnsureBucketDeleted(bucketName string, logger log.FieldLogger) error
	S3EnsureObjectDeleted(bucketName, path string) error
	S3LargeCopy(srcBucketName, srcKey, destBucketName, destKey *string) error
	GetMultitenantBucketNameForInstallation(installationID string, store model.InstallationDatabaseStoreInterface) (string, error)

	GenerateBifrostUtilitySecret(clusterID string, logger log.FieldLogger) (*corev1.Secret, error)
	GetCIDRByVPCTag(vpcTagName string, logger log.FieldLogger) (string, error)

	GetVpcResourcesByVpcID(vpcID string, logger log.FieldLogger) (ClusterResources, error)
	TagResourcesByCluster(clusterResources ClusterResources, cluster *model.Cluster, owner string, logger log.FieldLogger) error

	SecretsManagerGetPGBouncerAuthUserPassword(vpcID string) (string, error)
	SecretsManagerValidateExternalDatabaseSecret(name string) error
	SwitchClusterTags(clusterID string, targetClusterID string, logger log.FieldLogger) error

	EnsureEKSCluster(cluster *model.Cluster, resources ClusterResources, eksMetadata model.EKSMetadata) (*eks.Cluster, error)
	EnsureEKSClusterNodeGroups(cluster *model.Cluster, resources ClusterResources, eksMetadata model.EKSMetadata) ([]*eks.Nodegroup, error)
	GetEKSCluster(clusterName string) (*eks.Cluster, error)
	IsClusterReady(clusterName string) (bool, error)
	EnsureNodeGroupsDeleted(cluster *model.Cluster) (bool, error)
	EnsureEKSClusterDeleted(cluster *model.Cluster) (bool, error)
	InstallEKSEBSAddon(cluster *model.Cluster) error

	AllowEKSPostgresTraffic(cluster *model.Cluster, eksMetadata model.EKSMetadata) error
	RevokeEKSPostgresTraffic(cluster *model.Cluster, eksMetadata model.EKSMetadata) error

	GetRegion() string
	GetAccountID() (string, error)
}

// Client is a client for interacting with AWS resources in a single AWS account.
type Client struct {
	store   model.InstallationDatabaseStoreInterface
	logger  log.FieldLogger
	cache   *cache
	service *Service
	config  *aws.Config
	mux     *sync.Mutex
}

// NewAWSClientWithConfig returns a new instance of Client with a custom configuration.
func NewAWSClientWithConfig(config *aws.Config, logger log.FieldLogger) (*Client, error) {
	client := &Client{
		logger: logger,
		config: config,
		mux:    &sync.Mutex{},
		cache:  &cache{},
	}
	err := client.buildCache()
	if err != nil {
		return nil, errors.Wrap(err, "failed to build AWS client cache")
	}
	err = client.validateCache()
	if err != nil {
		return nil, errors.Wrap(err, "invalid client cache")
	}

	return client, nil
}

type cache struct {
	environment string
	route53     *route53Cache
}

// Service hold AWS clients for each service.
type Service struct {
	acm ACMAPI
	ec2 EC2API
	// ec2                   ec2iface.EC2API
	iam                   iamiface.IAMAPI
	rds                   rdsiface.RDSAPI
	s3                    s3iface.S3API
	route53               route53iface.Route53API
	secretsManager        secretsmanageriface.SecretsManagerAPI
	resourceGroupsTagging resourcegroupstaggingapiiface.ResourceGroupsTaggingAPIAPI
	kms                   kmsiface.KMSAPI
	dynamodb              DynamoDBAPI
	sts                   stsiface.STSAPI
	appAutoscaling        applicationautoscalingiface.ApplicationAutoScalingAPI
	eks                   eksiface.EKSAPI
}

// NewService creates a new instance of Service.
func NewService(sess *session.Session, cfg awsv2.Config) *Service {
	return &Service{
		acm:                   acm.NewFromConfig(cfg), // v2
		iam:                   iam.New(sess),
		rds:                   rds.New(sess),
		s3:                    s3.New(sess),
		route53:               route53.New(sess),
		secretsManager:        secretsmanager.New(sess),
		resourceGroupsTagging: resourcegroupstaggingapi.New(sess),
		ec2:                   ec2.NewFromConfig(cfg), // v2
		kms:                   kms.New(sess),
		dynamodb:              dynamodb.NewFromConfig(cfg), // v2
		sts:                   sts.New(sess),
		appAutoscaling:        applicationautoscaling.New(sess),
		eks:                   eks.New(sess),
	}
}

// GetRegion returns current AWS region.
func (c *Client) GetRegion() string {
	return *c.config.Region
}

// Service constructs an AWS session and configuration if not yet successfully done and returns AWS
// clients set up.
func (c *Client) Service() *Service {
	ctx := context.TODO()

	if c.service == nil {
		// Load configuration for the V2 SDK
		cfg, err := NewAWSConfig(ctx)
		if err != nil {
			c.logger.WithError(err).Error("Can't load AWS Configuration")
		}

		// Load session for the V1 SDK
		sess, err := NewAWSSessionWithLogger(c.config, c.logger.WithField("tools-aws", "client"))
		if err != nil {
			c.logger.WithError(err).Error("failed to initialize AWS session")
			// Calls to AWS will fail until a healthy session is acquired.
			return NewService(&session.Session{}, cfg)
		}

		c.mux.Lock()
		c.service = NewService(sess, cfg)
		c.mux.Unlock()
	}

	return c.service
}

func (c *Client) buildCache() error {
	err := c.buildCloudEnvironmentNameCache()
	if err != nil {
		return errors.Wrap(err, "failed to lookup AWS environment value")
	}

	err = c.buildRoute53Cache()
	if err != nil {
		return errors.Wrap(err, "failed to build route53 cache")
	}

	c.logger.WithFields(log.Fields{
		"environment":            c.cache.environment,
		"private-hosted-zone-id": c.cache.route53.privateHostedZoneID,
		"public-hosted-zone-ids": c.cache.route53.publicHostedZones,
	}).Info("AWS client cache initialized")

	return nil
}

func (c *Client) validateCache() error {
	if c.cache == nil || c.cache.route53 == nil {
		return errors.New("cache has not been properly initialized")
	}
	if len(c.cache.environment) == 0 {
		return errors.New("environment cache value is empty")
	}
	if len(c.cache.route53.privateHostedZoneID) == 0 {
		return errors.New("private hosted zone ID cache value is empty")
	}
	if len(c.cache.route53.publicHostedZones) == 0 {
		return errors.New("public hosted zone IDs cache is empty")
	}

	return nil
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
func (c *Client) GetCloudEnvironmentName() string {
	return c.cache.environment
}

// buildCloudEnvironmentNameCache looks for a standard cloud account environment
// name and chacnes it in the AWS client.
func (c *Client) buildCloudEnvironmentNameCache() error {
	accountAliases, err := c.GetAccountAliases()
	if err != nil {
		return errors.Wrap(err, "failed to get account aliases")
	}
	if len(accountAliases.AccountAliases) < 1 {
		return errors.New("account alias not defined")
	}

	for _, alias := range accountAliases.AccountAliases {
		if strings.HasPrefix(*alias, "mattermost-cloud") && len(strings.Split(*alias, "-")) == 3 {
			envName := strings.Split(*alias, "-")[2]
			if len(envName) == 0 {
				return errors.New("environment name value was empty")
			}

			c.cache.environment = envName
			return nil
		}
	}

	return errors.New("account environment name could not be found from account aliases")
}
