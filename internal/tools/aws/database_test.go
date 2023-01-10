// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.
//

package aws

import (
	"context"
	"fmt"
	"os"
	"strings"
	"sync"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	ec2Types "github.com/aws/aws-sdk-go-v2/service/ec2/types"
	"github.com/aws/aws-sdk-go-v2/service/kms"
	kmsTypes "github.com/aws/aws-sdk-go-v2/service/kms/types"
	"github.com/aws/aws-sdk-go-v2/service/rds"
	rdsTypes "github.com/aws/aws-sdk-go-v2/service/rds/types"
	gt "github.com/aws/aws-sdk-go-v2/service/resourcegroupstaggingapi"
	gtTypes "github.com/aws/aws-sdk-go-v2/service/resourcegroupstaggingapi/types"
	"github.com/aws/aws-sdk-go-v2/service/secretsmanager"
	"github.com/golang/mock/gomock"
	"github.com/mattermost/mattermost-cloud/internal/testlib"
	"github.com/mattermost/mattermost-cloud/model"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
	"github.com/stretchr/testify/require"
)

// Tests provisioning database acceptance path. Use this test for deriving other tests.
// If tests are broken, this should be the first test to get fixed.
func (a *AWSTestSuite) TestProvisioningRDSAcceptance() {
	database := RDSDatabase{
		databaseType:   model.DatabaseEngineTypeMySQL,
		installationID: a.InstallationA.ID,
		client:         a.Mocks.AWS,
	}

	gomock.InOrder(
		a.Mocks.Log.Logger.EXPECT().
			WithFields(log.Fields{
				"db-cluster-name": CloudID(a.InstallationA.ID),
				"database-type":   database.databaseType,
			}).
			Return(testlib.NewLoggerEntry()).
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
		a.Mocks.API.EC2.EXPECT().DescribeVpcs(context.TODO(), gomock.Any()).
			Return(&ec2.DescribeVpcsOutput{Vpcs: []ec2Types.Vpc{{VpcId: &a.VPCa}}}, nil).
			Times(1),

		// Create a database secret.
		a.Mocks.API.SecretsManager.EXPECT().
			GetSecretValue(gomock.Any(), gomock.Any()).
			Return(&secretsmanager.GetSecretValueOutput{SecretString: &a.SecretString}, nil).
			Times(1),

		// Create encryption key since none has been created yet.
		a.Mocks.API.ResourceGroupsTagging.EXPECT().
			GetResources(gomock.Any(), gomock.Any()).
			Return(&gt.GetResourcesOutput{}, nil).
			Do(func(ctx context.Context, input *gt.GetResourcesInput, optFns ...func(*gt.Options)) {
				a.Assert().Equal(DefaultRDSEncryptionTagKey, *input.TagFilters[0].Key)
				a.Assert().Equal(CloudID(a.InstallationA.ID), input.TagFilters[0].Values[0])
				a.Assert().Nil(input.PaginationToken)
			}).
			Times(1),

		a.Mocks.API.KMS.EXPECT().
			CreateKey(gomock.Any(), gomock.Any()).
			Do(func(ctx context.Context, input *kms.CreateKeyInput, optFns ...func(*kms.Options)) {
				a.Assert().Equal(*input.Tags[0].TagKey, DefaultRDSEncryptionTagKey)
				a.Assert().Equal(*input.Tags[0].TagValue, CloudID(a.InstallationA.ID))
			}).
			Return(&kms.CreateKeyOutput{
				KeyMetadata: &kmsTypes.KeyMetadata{
					Arn:      aws.String(a.ResourceARN),
					KeyId:    aws.String(a.RDSEncryptionKeyID),
					KeyState: kmsTypes.KeyStateEnabled,
				},
			}, nil).
			Times(1),

		// Get single tenant database configuration.
		a.Mocks.Model.DatabaseInstallationStore.EXPECT().GetSingleTenantDatabaseConfigForInstallation(a.InstallationA.ID).
			Return(&model.SingleTenantDatabaseConfig{PrimaryInstanceType: "db.r5.large", ReplicaInstanceType: "db.r5.small", ReplicasCount: 1}, nil).
			Times(1),

		// Retrive the Availability Zones.
		a.Mocks.API.EC2.EXPECT().DescribeAvailabilityZones(context.TODO(), gomock.Any()).
			Return(&ec2.DescribeAvailabilityZonesOutput{AvailabilityZones: []ec2Types.AvailabilityZone{{ZoneName: aws.String("us-honk-1a")}, {ZoneName: aws.String("us-honk-1b")}}}, nil).
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
		databaseType:   model.DatabaseEngineTypeMySQL,
		installationID: a.InstallationA.ID,
		client:         a.Mocks.AWS,
	}

	gomock.InOrder(
		a.Mocks.Log.Logger.EXPECT().
			WithFields(log.Fields{
				"db-cluster-name": CloudID(a.InstallationA.ID),
				"database-type":   database.databaseType,
			}).
			Return(testlib.NewLoggerEntry()).
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
		a.Mocks.API.EC2.EXPECT().DescribeVpcs(context.TODO(), gomock.Any()).
			Return(&ec2.DescribeVpcsOutput{Vpcs: []ec2Types.Vpc{{VpcId: &a.VPCa}}}, nil).
			Times(1),

		// Create a database secret.
		a.Mocks.API.SecretsManager.EXPECT().
			GetSecretValue(gomock.Any(), gomock.Any()).
			Return(&secretsmanager.GetSecretValueOutput{SecretString: &a.SecretString}, nil).
			Times(1),

		// Get encryption key associated with this installation. This step assumes that
		// the key already exists.
		a.Mocks.API.ResourceGroupsTagging.EXPECT().
			GetResources(gomock.Any(), gomock.Any()).
			Do(func(ctx context.Context, input *gt.GetResourcesInput, optFns ...func(*gt.Options)) {
				a.Assert().Equal(DefaultRDSEncryptionTagKey, *input.TagFilters[0].Key)
				a.Assert().Equal(CloudID(a.InstallationA.ID), input.TagFilters[0].Values[0])
				a.Assert().Nil(input.PaginationToken)
			}).
			Return(&gt.GetResourcesOutput{
				ResourceTagMappingList: []gtTypes.ResourceTagMapping{
					{
						ResourceARN: aws.String(a.ResourceARN),
					},
				},
			}, nil).
			Times(1),

		a.Mocks.API.KMS.EXPECT().
			DescribeKey(gomock.Any(), gomock.Any()).
			Return(&kms.DescribeKeyOutput{
				KeyMetadata: &kmsTypes.KeyMetadata{
					Arn:      aws.String(a.ResourceARN),
					KeyId:    aws.String(a.RDSEncryptionKeyID),
					KeyState: kmsTypes.KeyStateEnabled,
				},
			}, nil).
			Do(func(ctx context.Context, input *kms.DescribeKeyInput, optFns ...func(*kms.Options)) {
				a.Assert().Equal(*input.KeyId, a.ResourceARN)
			}).
			Times(1),

		// Get single tenant database configuration.
		a.Mocks.Model.DatabaseInstallationStore.EXPECT().GetSingleTenantDatabaseConfigForInstallation(a.InstallationA.ID).
			Return(&model.SingleTenantDatabaseConfig{PrimaryInstanceType: "db.r5.large", ReplicaInstanceType: "db.r5.small", ReplicasCount: 1}, nil).
			Times(1),

		// Retrive the Availability Zones.
		a.Mocks.API.EC2.EXPECT().DescribeAvailabilityZones(context.TODO(), gomock.Any()).
			Return(&ec2.DescribeAvailabilityZonesOutput{AvailabilityZones: []ec2Types.AvailabilityZone{{ZoneName: aws.String("us-honk-1a")}, {ZoneName: aws.String("us-honk-1b")}}}, nil).
			Times(1),
	)

	a.SetExpectCreateDBCluster()
	a.SetExpectCreateDBInstance()

	err := database.Provision(a.Mocks.Model.DatabaseInstallationStore, a.Mocks.Log.Logger)
	a.Assert().NoError(err)
}

func (a *AWSTestSuite) TestSnapshot() {
	database := RDSDatabase{
		databaseType:   model.DatabaseEngineTypeMySQL,
		installationID: a.InstallationA.ID,
		client:         a.Mocks.AWS,
	}

	gomock.InOrder(
		a.Mocks.Log.Logger.EXPECT().
			WithFields(log.Fields{
				"db-cluster-name": CloudID(a.InstallationA.ID),
				"database-type":   database.databaseType,
			}).
			Return(testlib.NewLoggerEntry()).
			Times(1),

		a.Mocks.API.RDS.EXPECT().CreateDBClusterSnapshot(gomock.Any(), gomock.Any()).
			Return(&rds.CreateDBClusterSnapshotOutput{}, nil).
			Do(func(ctx context.Context, input *rds.CreateDBClusterSnapshotInput, optFns ...func(*rds.Options)) {
				a.Assert().Equal(*input.DBClusterIdentifier, CloudID(a.ClusterA.ID))
				a.Assert().True(strings.Contains(*input.DBClusterSnapshotIdentifier, fmt.Sprintf("%s-snapshot-", a.ClusterA.ID)))
				a.Assert().Greater(len(input.Tags), 0)
				a.Assert().Equal(*input.Tags[0].Key, DefaultClusterInstallationSnapshotTagKey)
				a.Assert().Equal(*input.Tags[0].Value, RDSSnapshotTagValue(CloudID(a.ClusterA.ID)))
			}).Times(1),
	)

	err := database.Snapshot(a.Mocks.AWS.store, a.Mocks.Log.Logger)
	a.Assert().NoError(err)
}

func (a *AWSTestSuite) TestSnapshotError() {
	database := RDSDatabase{
		databaseType:   model.DatabaseEngineTypeMySQL,
		installationID: a.InstallationA.ID,
		client:         a.Mocks.AWS,
	}

	gomock.InOrder(
		a.Mocks.Log.Logger.EXPECT().
			WithFields(log.Fields{
				"db-cluster-name": CloudID(a.InstallationA.ID),
				"database-type":   database.databaseType,
			}).
			Return(testlib.NewLoggerEntry()).
			Times(1),

		a.Mocks.API.RDS.EXPECT().
			CreateDBClusterSnapshot(gomock.Any(), gomock.Any()).
			Return(nil, errors.New("database is not stable")).
			Times(1),
	)

	err := database.Snapshot(a.Mocks.AWS.store, a.Mocks.Log.Logger)

	a.Assert().Error(err)
	a.Assert().Equal("failed to create a DB cluster snapshot: database is not stable", err.Error())
}

// Helpers

// This whole block deals with RDS DB Cluster creation.
func (a *AWSTestSuite) SetExpectCreateDBCluster() {
	gomock.InOrder(
		a.Mocks.API.RDS.EXPECT().
			DescribeDBClusters(gomock.Any(), gomock.Any()).
			Return(nil, errors.New("db cluster does not exist")).
			Times(1),

		a.Mocks.API.EC2.EXPECT().
			DescribeSecurityGroups(context.TODO(), gomock.Any()).
			Return(&ec2.DescribeSecurityGroupsOutput{
				SecurityGroups: []ec2Types.SecurityGroup{{GroupId: &a.GroupID}},
			}, nil).
			Times(1),

		a.Mocks.API.RDS.EXPECT().
			DescribeDBSubnetGroups(gomock.Any(), gomock.Any()).
			Return(&rds.DescribeDBSubnetGroupsOutput{
				DBSubnetGroups: []rdsTypes.DBSubnetGroup{
					{
						DBSubnetGroupName: aws.String(DBSubnetGroupName(a.VPCa)),
					},
				},
			}, nil).
			Times(1),

		a.Mocks.API.RDS.EXPECT().
			CreateDBCluster(gomock.Any(), gomock.Any()).
			Do(func(ctx context.Context, input *rds.CreateDBClusterInput, optFns ...func(*rds.Options)) {
				for _, zone := range input.AvailabilityZones {
					a.Assert().Contains(a.RDSAvailabilityZones, zone)
				}
				a.Assert().Equal(*input.BackupRetentionPeriod, int32(7))
				a.Assert().Equal(*input.DBClusterIdentifier, CloudID(a.InstallationA.ID))
				a.Assert().Equal(*input.DatabaseName, a.DBName)
				a.Assert().Equal(input.VpcSecurityGroupIds[0], a.GroupID)
			}).
			Times(1),
	)
}

// This whole block deals with RDS Instance creation.
func (a *AWSTestSuite) SetExpectCreateDBInstance() {
	gomock.InOrder(
		a.Mocks.API.RDS.EXPECT().
			DescribeDBInstances(gomock.Any(), gomock.Any()).
			Return(nil, errors.New("db cluster instance does not exist")).
			Do(func(ctx context.Context, input *rds.DescribeDBInstancesInput, optFns ...func(*rds.Options)) {
				a.Assert().Equal(*input.DBInstanceIdentifier, RDSMasterInstanceID(a.InstallationA.ID))
			}),

		a.Mocks.API.RDS.EXPECT().
			CreateDBInstance(gomock.Any(), gomock.Any()).Return(nil, nil).
			Do(func(ctx context.Context, input *rds.CreateDBInstanceInput, optFns ...func(*rds.Options)) {
				a.Assert().Equal(*input.DBClusterIdentifier, CloudID(a.InstallationA.ID))
				a.Assert().Equal(*input.DBInstanceIdentifier, RDSMasterInstanceID(a.InstallationA.ID))
			}).
			Times(1),

		a.Mocks.API.RDS.EXPECT().
			DescribeDBInstances(gomock.Any(), gomock.Any()).
			Return(nil, errors.New("db cluster instance does not exist")).
			Do(func(ctx context.Context, input *rds.DescribeDBInstancesInput, optFns ...func(*rds.Options)) {
				a.Assert().Equal(*input.DBInstanceIdentifier, RDSReplicaInstanceID(a.InstallationA.ID, 0))
			}),

		a.Mocks.API.RDS.EXPECT().
			CreateDBInstance(gomock.Any(), gomock.Any()).Return(nil, nil).
			Do(func(ctx context.Context, input *rds.CreateDBInstanceInput, optFns ...func(*rds.Options)) {
				a.Assert().Equal(*input.DBClusterIdentifier, CloudID(a.InstallationA.ID))
				a.Assert().Equal(*input.DBInstanceIdentifier, RDSReplicaInstanceID(a.InstallationA.ID, 0))
			}),
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

	logger := log.New()
	database := NewRDSDatabase(model.DatabaseEngineTypeMySQL, id, &Client{
		mux: &sync.Mutex{},
	}, false)

	err := database.Provision(nil, logger)
	require.NoError(t, err)
}

func TestDatabaseTeardown(t *testing.T) {
	id := os.Getenv("SUPER_AWS_DATABASE_TEST")
	if id == "" {
		return
	}

	logger := log.New()
	database := NewRDSDatabase(model.DatabaseEngineTypeMySQL, id, &Client{
		mux: &sync.Mutex{},
	}, false)

	err := database.Teardown(nil, false, logger)
	require.NoError(t, err)
}
