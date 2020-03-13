package aws

import (
	"fmt"
	"os"
	"strings"
	"sync"
	"testing"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/aws/aws-sdk-go/service/rds"
	"github.com/aws/aws-sdk-go/service/secretsmanager"
	"github.com/golang/mock/gomock"
	testlib "github.com/mattermost/mattermost-cloud/internal/testlib"
	"github.com/mattermost/mattermost-cloud/model"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/require"
)

func (a *AWSTestSuite) TestSnapshot() {
	database := RDSDatabase{
		installationID: a.InstallationA.ID,
		client:         a.Mocks.AWS,
	}

	gomock.InOrder(
		a.Mocks.API.RDS.EXPECT().CreateDBClusterSnapshot(gomock.Any()).
			Return(&rds.CreateDBClusterSnapshotOutput{}, nil).Do(func(input *rds.CreateDBClusterSnapshotInput) {
			a.Assert().Equal(*input.DBClusterIdentifier, CloudID(a.ClusterA.ID))
			a.Assert().True(strings.Contains(*input.DBClusterSnapshotIdentifier, fmt.Sprintf("%s-snapshot-", a.ClusterA.ID)))
			a.Assert().Greater(len(input.Tags), 0)
			a.Assert().Equal(*input.Tags[0].Key, DefaultClusterInstallationSnapshotTagKey)
			a.Assert().Equal(*input.Tags[0].Value, RDSSnapshotTagValue(CloudID(a.ClusterA.ID)))
		}).
			Times(1),

		a.Mocks.LOG.Logger.EXPECT().
			WithField("installation-id", a.InstallationA.ID).
			Return(testlib.NewLoggerEntry()).
			Times(1),
	)

	err := database.Snapshot(a.Mocks.LOG.Logger)
	a.Assert().NoError(err)
}

func (a *AWSTestSuite) TestSnapshotError() {
	database := RDSDatabase{
		installationID: a.InstallationA.ID,
		client:         a.Mocks.AWS,
	}

	gomock.InOrder(
		a.Mocks.API.RDS.EXPECT().
			CreateDBClusterSnapshot(gomock.Any()).
			Return(nil, errors.New("database is not stable")).
			Times(1),

		a.Mocks.LOG.Logger.EXPECT().
			WithField("installation-id", a.InstallationA.ID).
			Return(testlib.NewLoggerEntry()).
			Times(1),
	)

	err := database.Snapshot(a.Mocks.LOG.Logger)

	a.Assert().Error(err)
	a.Assert().Equal("failed to create a DB cluster snapshot: database is not stable", err.Error())
}

// TODO(gsagula): test other database use-cases.
// Acceptance test for provisioning a RDS database.
func (a *AWSTestSuite) TestProvisionRDS() {
	database := RDSDatabase{
		installationID: a.InstallationA.ID,
		client:         a.Mocks.AWS,
	}

	a.Mocks.LOG.Logger.EXPECT().Infof("Provisioning AWS RDS database with ID %s", CloudID(a.InstallationA.ID)).Return().Times(1)

	a.Mocks.Model.DatabaseInstallationStore.EXPECT().
		GetClusterInstallations(gomock.Any()).
		Do(func(input *model.ClusterInstallationFilter) {
			a.Assert().Equal(input.InstallationID, a.InstallationA.ID)
		}).
		Return([]*model.ClusterInstallation{
			&model.ClusterInstallation{ID: a.ClusterA.ID},
		}, nil).
		Times(1)

	a.Mocks.API.EC2.EXPECT().
		DescribeVpcs(gomock.Any()).
		Return(&ec2.DescribeVpcsOutput{Vpcs: []*ec2.Vpc{&ec2.Vpc{VpcId: &a.VPCa}}}, nil).
		Times(1)

	a.Mocks.LOG.Logger.EXPECT().
		WithField("secret-name", RDSSecretName(CloudID(a.InstallationA.ID))).
		Return(testlib.NewLoggerEntry()).
		Times(1).
		After(a.Mocks.API.SecretsManager.EXPECT().
			GetSecretValue(gomock.Any()).
			Return(&secretsmanager.GetSecretValueOutput{SecretString: &a.SecretString}, nil).
			Times(1))

	a.Mocks.API.RDS.EXPECT().DescribeDBClusters(gomock.Any()).Return(nil, errors.New("db cluster does not exist")).Times(1)

	a.Mocks.LOG.Logger.EXPECT().
		WithField("security-group-ids", []string{a.GroupID}).
		Return(testlib.NewLoggerEntry()).Times(1).
		After(a.Mocks.API.EC2.EXPECT().
			DescribeSecurityGroups(gomock.Any()).
			Return(&ec2.DescribeSecurityGroupsOutput{
				SecurityGroups: []*ec2.SecurityGroup{&ec2.SecurityGroup{GroupId: &a.GroupID}},
			}, nil))

	a.Mocks.LOG.Logger.EXPECT().
		WithField("db-subnet-group-name", DBSubnetGroupName(a.VPCa)).
		Return(testlib.NewLoggerEntry()).
		Times(1).
		After(a.Mocks.API.RDS.EXPECT().
			DescribeDBSubnetGroups(gomock.Any()).
			Return(&rds.DescribeDBSubnetGroupsOutput{
				DBSubnetGroups: []*rds.DBSubnetGroup{
					&rds.DBSubnetGroup{
						DBSubnetGroupName: aws.String(DBSubnetGroupName(a.VPCa)),
					},
				},
			}, nil))

	a.Mocks.LOG.Logger.EXPECT().WithField("db-cluster-name", CloudID(a.InstallationA.ID)).Return(testlib.NewLoggerEntry()).Times(1)

	a.Mocks.API.RDS.EXPECT().
		CreateDBCluster(gomock.Any()).
		Return(nil, nil).
		Do(func(input *rds.CreateDBClusterInput) {
			for _, zone := range input.AvailabilityZones {
				a.Assert().Contains(a.RDSAvailabilityZones, *zone)
			}
			a.Assert().Equal(*input.BackupRetentionPeriod, int64(7))
			a.Assert().Equal(*input.DBClusterIdentifier, CloudID(a.InstallationA.ID))
			a.Assert().Equal(*input.DatabaseName, a.DBName)
			a.Assert().Equal(*input.VpcSecurityGroupIds[0], a.GroupID)
		}).
		Times(1)

	a.Mocks.API.RDS.EXPECT().
		DescribeDBInstances(gomock.Any()).
		Return(nil, errors.New("db cluster instance does not exist")).
		Do(func(input *rds.DescribeDBInstancesInput) {
			a.Assert().Equal(*input.DBInstanceIdentifier, RDSMasterInstanceID(a.InstallationA.ID))
		})

	a.Mocks.LOG.Logger.EXPECT().WithField("db-instance-name", RDSMasterInstanceID(a.InstallationA.ID)).
		Return(testlib.NewLoggerEntry()).
		Times(1).
		After(a.Mocks.API.RDS.EXPECT().
			CreateDBInstance(gomock.Any()).Return(nil, nil).
			Do(func(input *rds.CreateDBInstanceInput) {
				a.Assert().Equal(*input.DBClusterIdentifier, CloudID(a.InstallationA.ID))
				a.Assert().Equal(*input.DBInstanceIdentifier, RDSMasterInstanceID(a.InstallationA.ID))
			}).
			Times(1))

	err := database.Provision(a.Mocks.Model.DatabaseInstallationStore, a.Mocks.LOG.Logger)
	a.Assert().NoError(err)
}

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
	database := NewRDSDatabase(id, &Client{
		mux: &sync.Mutex{},
	})

	err := database.Provision(nil, logger)
	require.NoError(t, err)
}

func TestDatabaseTeardown(t *testing.T) {
	id := os.Getenv("SUPER_AWS_DATABASE_TEST")
	if id == "" {
		return
	}

	logger := logrus.New()
	database := NewRDSDatabase(id, &Client{
		mux: &sync.Mutex{},
	})

	err := database.Teardown(false, logger)
	require.NoError(t, err)
}
