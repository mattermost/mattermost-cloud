package aws

import (
	"testing"

	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/acm"
	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/aws/aws-sdk-go/service/iam"
	"github.com/aws/aws-sdk-go/service/rds"
	"github.com/aws/aws-sdk-go/service/route53"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/aws/aws-sdk-go/service/secretsmanager"

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
	LOG   *testlib.MockedFieldLogger
	Model *testlib.ModelMockedAPI
}

// Holds mocks, clients and fixtures for testing any AWS service. Any new fixtures
// should be added here.
type AWSTestSuite struct {
	suite.Suite

	// Mocked client and services.
	Mocks *Mocks

	// General AWS fixtures.
	InstallationA *model.Installation
	InstallationB *model.Installation

	ClusterA *model.Cluster
	ClusterB *model.Cluster

	ClusterInstallationA *model.ClusterInstallation
	ClusterInstallationB *model.ClusterInstallation

	VPCa string
	VPCb string

	// RDS database fixtures.
	RDSSecretID          string
	SecretString         string
	SecretStringUserErr  string
	SecretStringPassErr  string
	DBUser               string
	DBPassword           string
	GroupID              string
	RDSParamGroupCluster string
	RDSParamGroup        string
	DBName               string
	ResourceID           string
	RDSAvailabilityZones []string

	//Route53 fixtures
	EndpointsA []string
	EndpointsB []string
	DNSNameA   string
	DNSNameB   string
}

// NewAWSTestSuite gives a new instance of the entire AWS testing suite.
func NewAWSTestSuite() *AWSTestSuite {
	return &AWSTestSuite{
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
		RDSAvailabilityZones: []string{"us-east-1a", "us-east-1b", "us-east-1c"},
		GroupID:              "id-0000000000000000",
		SecretString:         `{"MasterUsername":"mmcloud","MasterPassword":"oX5rWueZt6ynsijE9PHpUO0VUWSwWSxqXCaZw1dC"}`,
		SecretStringUserErr:  `{"username":"mmcloud","MasterPassword":"oX5rWueZt6ynsijE9PHpUO0VUWSwWSxqXCaZw1dC"}`,
		SecretStringPassErr:  `{"MasterUsername":"mmcloud","password":"oX5rWueZt6ynsijE9PHpUO0VUWSwWSxqXCaZw1dC"}`,
		ResourceID:           "WSxqXCaZw1dC",

		EndpointsA: []string{"example1.mattermost.com", "example2.mattermost.com"},
		EndpointsB: []string{"example1.mattermost.com"},
		DNSNameA:   "mattermost.com",
		DNSNameB:   "mattermost-cloud.com",
	}
}

// This will take care of reseting the mocks on every run. Any new mocked library should be added here.
func (a *AWSTestSuite) SetupTest() {
	api := testlib.NewAWSMockedAPI()

	aws := &Client{
		rds:            api.RDS,
		ec2:            api.EC2,
		iam:            api.IAM,
		acm:            api.ACM,
		s3:             api.S3,
		route53:        api.Route53,
		secretsManager: api.SecretsManager,
	}

	// Point the singleton AWS client instance to a mocked AWS client instance.
	AWSClient = aws

	a.Mocks = &Mocks{
		API:   api,
		AWS:   aws,
		LOG:   testlib.NewMockedFieldLogger(),
		Model: testlib.NewModelMockedAPI(),
	}
}

func (a *AWSTestSuite) TestNewClient() {
	session, err := session.NewSession()
	a.Assert().NoError(err)

	client := &Client{
		acm:            acm.New(session),
		ec2:            ec2.New(session),
		iam:            iam.New(session),
		rds:            rds.New(session),
		s3:             s3.New(session),
		route53:        route53.New(session),
		secretsManager: secretsmanager.New(session),
	}

	a.Assert().NotNil(client)
	a.Assert().NotNil(client.acm)
	a.Assert().NotNil(client.iam)
	a.Assert().NotNil(client.rds)
	a.Assert().NotNil(client.s3)
	a.Assert().NotNil(client.route53)
	a.Assert().NotNil(client.secretsManager)
}

func TestAWSSuite(t *testing.T) {
	suite.Run(t, NewAWSTestSuite())
}
