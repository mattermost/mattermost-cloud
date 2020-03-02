package aws

import (
	"os"
	"testing"

	"github.com/aws/aws-sdk-go/service/rds"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
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
