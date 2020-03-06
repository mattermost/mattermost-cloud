package aws

import (
	"sync"
	"testing"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/acm"

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

// This will take care of resetting the mocks on every run. Any new mocked library should be added here.
func (a *AWSTestSuite) SetupTest() {
	api := testlib.NewAWSMockedAPI()
	a.Mocks = &Mocks{
		API: api,
		AWS: &Client{
			service: &Service{
				rds:            api.RDS,
				ec2:            api.EC2,
				iam:            api.IAM,
				acm:            api.ACM,
				s3:             api.S3,
				route53:        api.Route53,
				secretsManager: api.SecretsManager,
			},
			config: &aws.Config{},
			mux:    &sync.Mutex{},
		},
		LOG:   testlib.NewMockedFieldLogger(),
		Model: testlib.NewModelMockedAPI(),
	}
}

func (a *AWSTestSuite) TestNewClient() {
	client := &Client{
		mux: &sync.Mutex{},
	}

	a.Assert().NotNil(client)
	a.Assert().NotNil(client.Service().acm)
	a.Assert().NotNil(client.Service().iam)
	a.Assert().NotNil(client.Service().s3)
	a.Assert().NotNil(client.Service().route53)
	a.Assert().NotNil(client.Service().secretsManager)
	a.Assert().NotNil(client.Service().ec2)

	_, err := client.Service().acm.ListCertificates(&acm.ListCertificatesInput{})
	a.Assert().Error(err)
}

func TestAWSSuite(t *testing.T) {
	suite.Run(t, NewAWSTestSuite())
}
