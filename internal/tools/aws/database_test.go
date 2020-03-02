package aws

import (
	"os"
	"testing"

	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/aws/aws-sdk-go/service/rds"
	"github.com/aws/aws-sdk-go/service/secretsmanager"
	"github.com/mattermost/mattermost-cloud/model"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

// WARNING:
// This test is meant to exercise the provisioning and teardown of an AWS RDS
// database in a real AWS account. Only set the test env vars below if you wish
// to test this process with real AWS resources.

func TestDatabaseProvision(t *testing.T) {
	id := os.Getenv("SUPER_AWS_DATABASE_TEST")
	if id == "" {
		return
	}

	logger := logrus.New()
	database := NewRDSDatabase(id)

	err := database.Provision(nil, logger)
	require.NoError(t, err)
}

func TestDatabaseTeardown(t *testing.T) {
	id := os.Getenv("SUPER_AWS_DATABASE_TEST")
	if id == "" {
		return
	}

	logger := logrus.New()
	database := NewRDSDatabase(id)

	err := database.Teardown(false, logger)
	require.NoError(t, err)
}

func (a *AWSTestSuite) TestSnapshot() {
	database := RDSDatabase{
		installationID: a.InstallationA.ID,
	}

	a.SetCreateDBClusterSnapshotExpectation(a.InstallationA.ID).Return(&rds.CreateDBClusterSnapshotOutput{}, nil).Once()
	a.Mocks.LOG.WithFieldString("installation-id", a.InstallationA.ID)

	err := database.Snapshot(a.Mocks.LOG.Logger)

	a.Assert().NoError(err)
	a.Mocks.API.RDS.AssertExpectations(a.T())
}

func (a *AWSTestSuite) TestSnapshotError() {
	database := RDSDatabase{
		installationID: a.InstallationA.ID,
	}

	a.SetCreateDBClusterSnapshotExpectation(a.InstallationA.ID).Return(nil, errors.New("database is not stable")).Once()
	a.Mocks.LOG.WithFieldString("installation-id", a.InstallationA.ID)

	err := database.Snapshot(a.Mocks.LOG.Logger)

	a.Assert().Error(err)
	a.Assert().Equal("failed to create a DB cluster snapshot for replication: database is not stable", err.Error())
	a.Mocks.API.RDS.AssertExpectations(a.T())
}

// TODO(gsagula): test other database use-cases.
// Acceptance test for provisioning a RDS database.
func (a *AWSTestSuite) TestProvisionRDS() {
	database := RDSDatabase{
		installationID: a.InstallationA.ID,
	}

	a.SetDescribeVpcsExpectations(a.ClusterA.ID).Return(&ec2.DescribeVpcsOutput{Vpcs: []*ec2.Vpc{&ec2.Vpc{VpcId: &a.VPCa}}}, nil).Once()
	a.SetGetSecretValueExpectations(a.InstallationA.ID).Return(&secretsmanager.GetSecretValueOutput{SecretString: &a.SecretString}, nil).Once()
	a.SetDescribeDBClustersNotFoundExpectation().Once()
	a.SetDescribeSecurityGroupsExpectation().Once()
	a.SetDescribeDBSubnetGroupsExpectation(a.VPCa).Once()
	a.SetCreateDBClusterExpectation(a.InstallationA.ID).Return(nil, nil).Once()

	a.Mocks.LOG.WithFieldArgs("security-group-ids", a.GroupID).Once()
	a.Mocks.LOG.WithFieldString("db-subnet-group-name", SubnetGroupName(a.VPCa)).Once()
	a.Mocks.LOG.WithFieldString("db-cluster-name", CloudID(a.InstallationA.ID)).Once()
	a.Mocks.LOG.WithFieldString("secret-name", RDSSecretName(CloudID(a.InstallationA.ID))).Once()
	a.Mocks.LOG.WithFieldString("db-cluster-name", CloudID(a.InstallationA.ID)).Once()

	a.SetDescribeDBInstancesNotFoundExpectation(a.InstallationA.ID).Once()
	a.SetCreateDBInstanceExpectation(a.InstallationA.ID).Return(nil, nil).Once()
	a.SetClusterInstallationFilter(a.InstallationA.ID).Return([]*model.ClusterInstallation{
		&model.ClusterInstallation{ID: a.ClusterA.ID},
	}, nil).Once()
	a.Mocks.LOG.InfofString("Provisioning AWS RDS database with ID %s", CloudID(a.InstallationA.ID)).Once()
	a.Mocks.LOG.WithFieldString("db-instance-name", RDSMasterInstanceID(a.InstallationA.ID)).Once()

	err := database.Provision(a.Mocks.Model.DatabaseInstallationStore, a.Mocks.LOG.Logger)

	a.Assert().NoError(err)
}

// Helpers

func (a *AWSTestSuite) SetClusterInstallationFilter(installationID string) *mock.Call {
	return a.Mocks.Model.DatabaseInstallationStore.On("GetClusterInstallations", mock.MatchedBy(
		func(input *model.ClusterInstallationFilter) bool {
			return input.InstallationID == installationID
		}))
}

func (a *AWSTestSuite) SetDescribeVpcsExpectations(clusterID string) *mock.Call {
	return a.Mocks.API.EC2.On("DescribeVpcs", mock.MatchedBy(
		func(input *ec2.DescribeVpcsInput) bool {
			return *input.Filters[0].Name == VpcClusterIDTagKey &&
				*input.Filters[1].Name == VpcAvailableTagKey &&
				*input.Filters[1].Values[0] == VpcAvailableTagValueFalse
		}))
}

func (a *AWSTestSuite) SetGetSecretValueExpectations(installationID string) *mock.Call {
	return a.Mocks.API.SecretsManager.On("GetSecretValue", mock.MatchedBy(
		func(input *secretsmanager.GetSecretValueInput) bool {
			return *input.SecretId == RDSSecretName(CloudID(installationID))
		}))
}
