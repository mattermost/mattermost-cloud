// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.
//

package aws

import (
	"sync"
	"testing"

	ec2Types "github.com/aws/aws-sdk-go-v2/service/ec2/types"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/golang/mock/gomock"

	testlib "github.com/mattermost/mattermost-cloud/internal/testlib"
	"github.com/mattermost/mattermost-cloud/model"

	"github.com/stretchr/testify/suite"
)

// ClientTestSuite supplies tests for aws package Client.
type ClientTestSuite struct {
	suite.Suite

	session *session.Session
}

func (d *ClientTestSuite) SetupTest() {
	d.session = session.Must(session.NewSession())
}

func TestClientSuite(t *testing.T) {
	suite.Run(t, new(ClientTestSuite))
}

type Mocks struct {
	AWS   *Client
	API   *testlib.AWSMockedAPI
	Log   *testlib.MockedFieldLogger
	Model *testlib.ModelMockedAPI
}

// Holds mocks, clients and fixtures for testing any AWS service. Any new fixtures
// should be added here.
type AWSTestSuite struct {
	suite.Suite

	// GOMock controller.
	ctrl *gomock.Controller

	// Mocked client and services.
	Mocks *Mocks

	// General AWS fixtures.
	InstallationA *model.Installation
	InstallationB *model.Installation

	InstanceID string

	ClusterA *model.Cluster
	ClusterB *model.Cluster

	ClusterInstallationA *model.ClusterInstallation
	ClusterInstallationB *model.ClusterInstallation

	VPCa string
	VPCb string

	// RDS database fixtures.
	RDSEngineType        string
	RDSSecretID          string
	RDSClusterID         string
	SecretString         string
	SecretStringUserErr  string
	SecretStringPassErr  string
	DBUser               string
	DBPassword           string
	GroupID              string
	RDSParamGroupCluster string
	RDSParamGroup        string
	RDSEncryptionKeyID   string
	DBName               string
	ResourceID           string
	HostedZoneID         string
	CertifcateARN        string
	ResourceARN          string
	RDSResourceARN       string
	RDSAvailabilityZones []string

	// Route53 fixtures
	EndpointsA []string
	EndpointsB []string
	DNSNameA   string
	DNSNameB   string

	// AWS Tags
	DefaultRDSTag ec2Types.Tag
}

// NewAWSTestSuite gives a new instance of the entire AWS testing suite.
func NewAWSTestSuite(t *testing.T) *AWSTestSuite {
	return &AWSTestSuite{
		ctrl: gomock.NewController(t),

		VPCa: "vpc-000000000000000a",
		VPCb: "vpc-000000000000000b",

		InstallationA: &model.Installation{
			ID: "id000000000000000000000000a",
		},
		InstallationB: &model.Installation{
			ID: "id000000000000000000000000b",
		},

		ClusterA: &model.Cluster{
			ID: "id000000000000000000000000a",
		},
		ClusterB: &model.Cluster{
			ID: "id000000000000000000000000b",
		},

		ClusterInstallationA: &model.ClusterInstallation{
			ID:             "id000000000000000000000000a",
			InstallationID: "id000000000000000000000000a",
			ClusterID:      "id000000000000000000000000a",
		},
		ClusterInstallationB: &model.ClusterInstallation{
			ID:             "id000000000000000000000000b",
			InstallationID: "id000000000000000000000000b",
			ClusterID:      "id000000000000000000000000b",
		},

		DBName:               "mattermost",
		DBUser:               "admin",
		DBPassword:           "secret",
		RDSParamGroupCluster: "mattermost-provisioner-rds-cluster-pg",
		RDSParamGroup:        "mattermost-provisioner-rds-pg",
		RDSClusterID:         "rds-cluster-multitenant-09d44077df9934f96-97670d43",
		RDSAvailabilityZones: []string{"us-honk-1a", "us-honk-1b"},
		RDSEngineType:        model.DatabaseEngineTypeMySQL,
		GroupID:              "id-0000000000000000",
		SecretString:         `{"MasterUsername":"mmcloud","MasterPassword":"oX5rWueZt6ynsijE9PHpUO0VUWSwWSxqXCaZw1dC"}`,
		SecretStringUserErr:  `{"username":"mmcloud","MasterPassword":"oX5rWueZt6ynsijE9PHpUO0VUWSwWSxqXCaZw1dC"}`,
		SecretStringPassErr:  `{"MasterUsername":"mmcloud","password":"oX5rWueZt6ynsijE9PHpUO0VUWSwWSxqXCaZw1dC"}`,
		RDSEncryptionKeyID:   "rds-encryption-key-id-123",
		ResourceID:           "WSxqXCaZw1dC",
		CertifcateARN:        "arn:aws:certificate::123456789012",
		ResourceARN:          "arn:aws:kms:us-east-1:526412419611:key/10cbe864-7411-4cda-bd28-3355218d0995",
		RDSResourceARN:       "arn:aws:rds:us-east-1:926412419614:cluster:rds-cluster-multitenant-09d44077df9934f96-97670d43",
		InstanceID:           "WSxaaqafXCgeaZers2qergasdgsw1dC",

		EndpointsA: []string{"example1.mattermost.com", "example2.mattermost.com"},
		EndpointsB: []string{"example1.mattermost.com"},
		DNSNameA:   "mattermost.com",
		DNSNameB:   "mattermost-cloud.com",

		DefaultRDSTag: ec2Types.Tag{
			Key:   aws.String(trimTagPrefix(DefaultDBSecurityGroupTagKey)),
			Value: aws.String(DefaultDBSecurityGroupTagMySQLValue),
		},
	}
}

// This will take care of resetting the mocks on every run. Any new mocked library should be added here.
func (a *AWSTestSuite) SetupTest() {
	api := testlib.NewAWSMockedAPI(a.ctrl)
	a.Mocks = &Mocks{
		API: api,
		AWS: &Client{
			service: &Service{
				rds:                   api.RDS,
				ec2:                   api.EC2,
				iam:                   api.IAM,
				acm:                   api.ACM,
				s3:                    api.S3,
				route53:               api.Route53,
				secretsManager:        api.SecretsManager,
				resourceGroupsTagging: api.ResourceGroupsTagging,
				kms:                   api.KMS,
				sts:                   api.STS,
			},
			cache:  newClientDummyCache(),
			config: &aws.Config{},
			mux:    &sync.Mutex{},
		},
		Log:   testlib.NewMockedFieldLogger(a.ctrl),
		Model: testlib.NewModelMockedAPI(a.ctrl),
	}
}

func (a *AWSTestSuite) TearDown() {
	a.ctrl.Finish()
}

func TestAWSSuite(t *testing.T) {
	testSuite := NewAWSTestSuite(t)
	defer testSuite.TearDown()

	suite.Run(t, testSuite)
}

func newClientDummyCache() *cache {
	return &cache{
		environment: "dev",
		route53: &route53Cache{
			privateHostedZoneID: "HZONE1",
			publicHostedZones: map[string]awsHostedZone{
				"mattermost.com": {ID: "HZONE2"},
			},
		},
	}
}
