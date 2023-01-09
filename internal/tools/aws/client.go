// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.
//

package aws

import (
	"context"
	"strings"
	"sync"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/acm"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	"github.com/aws/aws-sdk-go-v2/service/eks"
	eksTypes "github.com/aws/aws-sdk-go-v2/service/eks/types"
	"github.com/aws/aws-sdk-go-v2/service/iam"
	"github.com/aws/aws-sdk-go-v2/service/kms"
	"github.com/aws/aws-sdk-go-v2/service/rds"
	"github.com/aws/aws-sdk-go-v2/service/resourcegroupstaggingapi"
	"github.com/aws/aws-sdk-go-v2/service/route53"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/secretsmanager"
	"github.com/aws/aws-sdk-go-v2/service/sts"
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
	GetS3RegionURL() string

	GenerateBifrostUtilitySecret(clusterID string, logger log.FieldLogger) (*corev1.Secret, error)
	GetCIDRByVPCTag(vpcTagName string, logger log.FieldLogger) (string, error)

	GetVpcResourcesByVpcID(vpcID string, logger log.FieldLogger) (ClusterResources, error)
	TagResourcesByCluster(clusterResources ClusterResources, cluster *model.Cluster, owner string, logger log.FieldLogger) error

	SecretsManagerGetPGBouncerAuthUserPassword(vpcID string) (string, error)
	SecretsManagerValidateExternalDatabaseSecret(name string) error
	SwitchClusterTags(clusterID string, targetClusterID string, logger log.FieldLogger) error

	EnsureEKSCluster(cluster *model.Cluster, resources ClusterResources, eksMetadata model.EKSMetadata) (*eksTypes.Cluster, error)
	EnsureEKSClusterNodeGroups(cluster *model.Cluster, resources ClusterResources, eksMetadata model.EKSMetadata) ([]*eksTypes.Nodegroup, error)
	GetEKSCluster(clusterName string) (*eksTypes.Cluster, error)
	IsClusterReady(clusterName string) (bool, error)
	EnsureNodeGroupsDeleted(cluster *model.Cluster) (bool, error)
	EnsureEKSClusterDeleted(cluster *model.Cluster) (bool, error)
	InstallEKSEBSAddon(cluster *model.Cluster) error

	AllowEKSPostgresTraffic(cluster *model.Cluster, eksMetadata model.EKSMetadata) error
	RevokeEKSPostgresTraffic(cluster *model.Cluster, eksMetadata model.EKSMetadata) error

	GetRegion() string
	GetAccountID() (string, error)

	GetLoadBalancerAPIByType(string) ELB
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
	acm                   ACMAPI
	ec2                   EC2API
	rds                   RDSAPI
	iam                   IAMAPI
	s3                    S3API
	secretsManager        SecretsManagerAPI
	route53               Route53API
	resourceGroupsTagging ResourceGroupsTaggingAPIAPI
	kms                   KMSAPI
	dynamodb              DynamoDBAPI
	sts                   STSAPI
	eks                   EKSAPI
	elb                   elasticLoadbalancer
}

// NewService creates a new instance of Service.
func NewService(cfg aws.Config) *Service {
	return &Service{
		acm:                   acm.NewFromConfig(cfg),                      // v2
		rds:                   rds.NewFromConfig(cfg),                      // v2
		iam:                   iam.NewFromConfig(cfg),                      // v2
		s3:                    s3.NewFromConfig(cfg),                       // v2
		route53:               route53.NewFromConfig(cfg),                  // v2
		secretsManager:        secretsmanager.NewFromConfig(cfg),           // v2
		resourceGroupsTagging: resourcegroupstaggingapi.NewFromConfig(cfg), // v2
		ec2:                   ec2.NewFromConfig(cfg),                      // v2
		kms:                   kms.NewFromConfig(cfg),                      // v2
		dynamodb:              dynamodb.NewFromConfig(cfg),                 // v2
		sts:                   sts.NewFromConfig(cfg),                      // v2
		eks:                   eks.NewFromConfig(cfg),                      // v2
		elb:                   newElasticLoadbalancerFromConfig(cfg),
	}
}

// GetRegion returns current AWS region.
func (c *Client) GetRegion() string {
	return c.config.Region
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

		c.mux.Lock()
		c.service = NewService(cfg)
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
		envNameParts := strings.Split(alias, "-")
		if strings.HasPrefix(alias, "mattermost-cloud") && len(envNameParts) == 3 {
			envName := envNameParts[2]
			if len(envName) == 0 {
				return errors.New("environment name value was empty")
			}

			c.cache.environment = envName
			return nil
		}
	}

	return errors.New("account environment name could not be found from account aliases")
}
