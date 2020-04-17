package aws

import (
	"fmt"
	"os"
	"strings"
	"sync"
	"testing"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/aws/aws-sdk-go/service/kms"
	"github.com/aws/aws-sdk-go/service/rds"
	gt "github.com/aws/aws-sdk-go/service/resourcegroupstaggingapi"
	"github.com/aws/aws-sdk-go/service/secretsmanager"
	"github.com/golang/mock/gomock"
	testlib "github.com/mattermost/mattermost-cloud/internal/testlib"
	"github.com/mattermost/mattermost-cloud/model"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/require"
)

// Tests provisioning database acceptance path. Use this test for deriving other tests.
// If tests are broken, this should be the first test to get fixed.
func (a *AWSTestSuite) TestProvisioningRDSAcceptance() {
	database := RDSDatabase{
		installationID: a.InstallationA.ID,
		client:         a.Mocks.AWS,
	}

	gomock.InOrder(
		a.Mocks.Log.Logger.EXPECT().
			Infof("Provisioning AWS RDS database with ID %s", CloudID(a.InstallationA.ID)).
			Return().
			Times(1),

		// Get cluster installations from data store.
		a.Mocks.Model.DatabaseInstallationStore.EXPECT().
			GetClusterInstallations(gomock.Any()).
			Do(func(input *model.ClusterInstallationFilter) {
				a.Assert().Equal(input.InstallationID, a.InstallationA.ID)
			}).
			Return([]*model.ClusterInstallation{{ID: a.ClusterA.ID}}, nil).
			Times(1),

		// Find the VPC which the installation belongs to.
		a.Mocks.API.EC2.EXPECT().DescribeVpcs(gomock.Any()).
			Return(&ec2.DescribeVpcsOutput{Vpcs: []*ec2.Vpc{{VpcId: &a.VPCa}}}, nil).
			Times(1),

		// Create a database secret.
		a.Mocks.API.SecretsManager.EXPECT().
			GetSecretValue(gomock.Any()).
			Return(&secretsmanager.GetSecretValueOutput{SecretString: &a.SecretString}, nil).
			Times(1),

		a.Mocks.Log.Logger.EXPECT().
			WithField("secret-name", RDSSecretName(CloudID(a.InstallationA.ID))).
			Return(testlib.NewLoggerEntry()).
			Times(1),

		// Create encryption key since none has been created yet.
		a.Mocks.API.ResourceGroupsTagging.EXPECT().
			GetResources(gomock.Any()).
			Return(&gt.GetResourcesOutput{}, nil).
			Do(func(input *gt.GetResourcesInput) {
				a.Assert().Equal(DefaultRDSEncryptionTagKey, *input.TagFilters[0].Key)
				a.Assert().Equal(CloudID(a.InstallationA.ID), *input.TagFilters[0].Values[0])
				a.Assert().Nil(input.PaginationToken)
			}).
			Times(1),

		a.Mocks.API.KMS.EXPECT().
			CreateKey(gomock.Any()).
			Do(func(input *kms.CreateKeyInput) {
				a.Assert().Equal(*input.Tags[0].TagKey, DefaultRDSEncryptionTagKey)
				a.Assert().Equal(*input.Tags[0].TagValue, CloudID(a.InstallationA.ID))
			}).
			Return(&kms.CreateKeyOutput{
				KeyMetadata: &kms.KeyMetadata{
					Arn:      aws.String(a.ResourceARN),
					KeyId:    aws.String(a.RDSEncryptionKeyID),
					KeyState: aws.String(kms.KeyStateEnabled),
				},
			}, nil).
			Times(1),

		a.Mocks.Log.Logger.EXPECT().
			Infof("Encrypting RDS database with key %s", a.ResourceARN).
			Times(1),
	)

	a.SetExpectCreateDBCluster()
	a.SetExpectCreateDBInstance()

	err := database.Provision(a.Mocks.Model.DatabaseInstallationStore, a.Mocks.Log.Logger)
	a.Assert().NoError(err)
}

// Tests provisioning database assuming that an encryption key already exists.
func (a *AWSTestSuite) TestProvisioningRDSWithExistentEncryptionKey() {
	database := RDSDatabase{
		installationID: a.InstallationA.ID,
		client:         a.Mocks.AWS,
	}

	gomock.InOrder(
		a.Mocks.Log.Logger.EXPECT().
			Infof("Provisioning AWS RDS database with ID %s", CloudID(a.InstallationA.ID)).
			Return().
			Times(1),

		// Get cluster installations from data store.
		a.Mocks.Model.DatabaseInstallationStore.EXPECT().
			GetClusterInstallations(gomock.Any()).
			Do(func(input *model.ClusterInstallationFilter) {
				a.Assert().Equal(input.InstallationID, a.InstallationA.ID)
			}).
			Return([]*model.ClusterInstallation{{ID: a.ClusterA.ID}}, nil).
			Times(1),

		// Find the VPC which the installation belongs to.
		a.Mocks.API.EC2.EXPECT().DescribeVpcs(gomock.Any()).
			Return(&ec2.DescribeVpcsOutput{Vpcs: []*ec2.Vpc{{VpcId: &a.VPCa}}}, nil).
			Times(1),

		// Create a database secret.
		a.Mocks.API.SecretsManager.EXPECT().
			GetSecretValue(gomock.Any()).
			Return(&secretsmanager.GetSecretValueOutput{SecretString: &a.SecretString}, nil).
			Times(1),

		a.Mocks.Log.Logger.EXPECT().
			WithField("secret-name", RDSSecretName(CloudID(a.InstallationA.ID))).
			Return(testlib.NewLoggerEntry()).
			Times(1),

		// Get encryption key associated with this installation. This step assumes that
		// the key already exists.
		a.Mocks.API.ResourceGroupsTagging.EXPECT().
			GetResources(gomock.Any()).
			Do(func(input *gt.GetResourcesInput) {
				a.Assert().Equal(DefaultRDSEncryptionTagKey, *input.TagFilters[0].Key)
				a.Assert().Equal(CloudID(a.InstallationA.ID), *input.TagFilters[0].Values[0])
				a.Assert().Nil(input.PaginationToken)
			}).
			Return(&gt.GetResourcesOutput{
				ResourceTagMappingList: []*gt.ResourceTagMapping{
					{
						ResourceARN: aws.String(a.ResourceARN),
					},
				},
			}, nil).
			Times(1),

		a.Mocks.API.KMS.EXPECT().
			DescribeKey(gomock.Any()).
			Return(&kms.DescribeKeyOutput{
				KeyMetadata: &kms.KeyMetadata{
					Arn:      aws.String(a.ResourceARN),
					KeyId:    aws.String(a.RDSEncryptionKeyID),
					KeyState: aws.String(kms.KeyStateEnabled),
				},
			}, nil).
			Do(func(input *kms.DescribeKeyInput) {
				a.Assert().Equal(*input.KeyId, a.ResourceARN)
			}).
			Times(1),

		a.Mocks.Log.Logger.EXPECT().
			Infof("Encrypting RDS database with key %s", a.ResourceARN).
			Times(1),
	)

	a.SetExpectCreateDBCluster()
	a.SetExpectCreateDBInstance()

	err := database.Provision(a.Mocks.Model.DatabaseInstallationStore, a.Mocks.Log.Logger)
	a.Assert().NoError(err)
}

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
		}).Times(1),

		a.Mocks.Log.Logger.EXPECT().
			WithField("installation-id", a.InstallationA.ID).
			Return(testlib.NewLoggerEntry()).
			Times(1),
	)

	err := database.Snapshot(a.Mocks.Log.Logger)
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

		a.Mocks.Log.Logger.EXPECT().
			WithField("installation-id", a.InstallationA.ID).
			Return(testlib.NewLoggerEntry()).
			Times(1),
	)

	err := database.Snapshot(a.Mocks.Log.Logger)

	a.Assert().Error(err)
	a.Assert().Equal("failed to create a DB cluster snapshot: database is not stable", err.Error())
}

// Helpers

// This whole block deals with RDS DB Cluster creation.
func (a *AWSTestSuite) SetExpectCreateDBCluster() {
	gomock.InOrder(
		a.Mocks.API.RDS.EXPECT().
			DescribeDBClusters(gomock.Any()).
			Return(nil, errors.New("db cluster does not exist")).
			Times(1),

		a.Mocks.API.EC2.EXPECT().
			DescribeSecurityGroups(gomock.Any()).
			Return(&ec2.DescribeSecurityGroupsOutput{
				SecurityGroups: []*ec2.SecurityGroup{{GroupId: &a.GroupID}},
			}, nil).
			Times(1),

		a.Mocks.Log.Logger.EXPECT().
			WithField("security-group-ids", []string{a.GroupID}).
			Return(testlib.NewLoggerEntry()).
			Times(1),

		a.Mocks.API.RDS.EXPECT().
			DescribeDBSubnetGroups(gomock.Any()).
			Return(&rds.DescribeDBSubnetGroupsOutput{
				DBSubnetGroups: []*rds.DBSubnetGroup{
					{
						DBSubnetGroupName: aws.String(DBSubnetGroupName(a.VPCa)),
					},
				},
			}, nil).
			Times(1),

		a.Mocks.Log.Logger.EXPECT().
			WithField("db-subnet-group-name", DBSubnetGroupName(a.VPCa)).
			Return(testlib.NewLoggerEntry()).
			Times(1),

		a.Mocks.API.RDS.EXPECT().
			CreateDBCluster(gomock.Any()).
			Do(func(input *rds.CreateDBClusterInput) {
				for _, zone := range input.AvailabilityZones {
					a.Assert().Contains(a.RDSAvailabilityZones, *zone)
				}
				a.Assert().Equal(*input.BackupRetentionPeriod, int64(7))
				a.Assert().Equal(*input.DBClusterIdentifier, CloudID(a.InstallationA.ID))
				a.Assert().Equal(*input.DatabaseName, a.DBName)
				a.Assert().Equal(*input.VpcSecurityGroupIds[0], a.GroupID)
			}).
			Times(1),

		a.Mocks.Log.Logger.EXPECT().
			WithField("db-cluster-name", CloudID(a.InstallationA.ID)).
			Return(testlib.NewLoggerEntry()).
			Times(1),
	)
}

// This whole block deals with RDS Instance creation.
func (a *AWSTestSuite) SetExpectCreateDBInstance() {
	gomock.InOrder(
		a.Mocks.API.RDS.EXPECT().
			DescribeDBInstances(gomock.Any()).
			Return(nil, errors.New("db cluster instance does not exist")).
			Do(func(input *rds.DescribeDBInstancesInput) {
				a.Assert().Equal(*input.DBInstanceIdentifier, RDSMasterInstanceID(a.InstallationA.ID))
			}),

		a.Mocks.API.RDS.EXPECT().
			CreateDBInstance(gomock.Any()).Return(nil, nil).
			Do(func(input *rds.CreateDBInstanceInput) {
				a.Assert().Equal(*input.DBClusterIdentifier, CloudID(a.InstallationA.ID))
				a.Assert().Equal(*input.DBInstanceIdentifier, RDSMasterInstanceID(a.InstallationA.ID))
			}).
			Times(1),

		a.Mocks.Log.Logger.EXPECT().WithField("db-instance-name", RDSMasterInstanceID(a.InstallationA.ID)).
			Return(testlib.NewLoggerEntry()).
			Times(1),
	)
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
