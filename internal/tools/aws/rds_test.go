// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.
//

package aws

import (
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/aws/aws-sdk-go/service/kms"
	"github.com/aws/aws-sdk-go/service/rds"
	"github.com/golang/mock/gomock"
	testlib "github.com/mattermost/mattermost-cloud/internal/testlib"
	"github.com/pkg/errors"
)

func (a *AWSTestSuite) TestRDSEnsureDBClusterCreated() {
	a.Mocks.API.RDS.EXPECT().DescribeDBClusters(gomock.Any()).Return(nil, errors.New("db cluster does not exist")).Times(1)

	a.Mocks.Log.Logger.EXPECT().
		WithField(gomock.Any(), gomock.Any()).
		Return(testlib.NewLoggerEntry()).
		Times(3)

	a.Mocks.API.EC2.EXPECT().
		DescribeSecurityGroups(gomock.Any()).
		Return(&ec2.DescribeSecurityGroupsOutput{
			SecurityGroups: []*ec2.SecurityGroup{
				{
					GroupId: &a.GroupID,
				},
			},
		}, nil)

	a.Mocks.API.RDS.EXPECT().
		DescribeDBSubnetGroups(gomock.Any()).
		Return(&rds.DescribeDBSubnetGroupsOutput{
			DBSubnetGroups: []*rds.DBSubnetGroup{
				{
					DBSubnetGroupName: aws.String(DBSubnetGroupName(a.VPCa)),
				},
			},
		}, nil)

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

	a.Mocks.API.KMS.EXPECT().
		CreateKey(gomock.Any()).
		Return(&kms.CreateKeyOutput{
			KeyMetadata: &kms.KeyMetadata{
				KeyId: aws.String(a.RDSEncryptionKeyID),
			},
		}, nil).
		Do(func(input *kms.CreateKeyInput) {
			a.Assert().Equal(*input.Description, "Key used for encrypting RDS database")
		}).
		Times(1)

	a.Mocks.API.KMS.EXPECT().
		CreateAlias(gomock.Any()).
		Return(nil, nil).
		Do(func(input *kms.CreateAliasInput) {
			a.Assert().Equal(*input.AliasName, KMSAliasNameRDS(CloudID(a.InstallationA.ID)))
		}).
		Times(1)

	// Retrive the Availability Zones.
	a.Mocks.API.EC2.EXPECT().DescribeAvailabilityZones(gomock.Any()).
		Return(&ec2.DescribeAvailabilityZonesOutput{AvailabilityZones: []*ec2.AvailabilityZone{{ZoneName: aws.String("us-honk-1a")}, {ZoneName: aws.String("us-honk-1b")}}}, nil).
		Times(1)

	err := a.Mocks.AWS.rdsEnsureDBClusterCreated(CloudID(a.InstallationA.ID), a.VPCa, a.DBUser, a.DBPassword, a.RDSEncryptionKeyID, a.RDSEngineType, a.Mocks.Log.Logger)
	a.Assert().NoError(err)
}

func (a *AWSTestSuite) TestRDSEnsureDBClusterCreatedAlreadyCreated() {
	a.Mocks.Log.Logger.EXPECT().
		WithField("db-cluster-name", CloudID(a.InstallationA.ID)).
		Return(testlib.NewLoggerEntry()).
		Times(1).
		After(a.Mocks.API.RDS.EXPECT().
			DescribeDBClusters(gomock.Any()).
			Return(nil, nil).
			Times(1))

	err := a.Mocks.AWS.rdsEnsureDBClusterCreated(CloudID(a.InstallationA.ID), a.VPCa, a.DBUser, a.DBPassword, a.RDSEncryptionKeyID, a.RDSEngineType, a.Mocks.Log.Logger)
	a.Assert().NoError(err)
}

func (a *AWSTestSuite) TestRDSEnsureDBClusterCreatedWithSGError() {
	a.Mocks.API.RDS.EXPECT().
		DescribeDBClusters(gomock.Any()).
		Return(nil, errors.New("db cluster does not exist")).
		Times(1)

	a.Mocks.Log.Logger.EXPECT().
		WithField("security-group-ids", []string{a.GroupID}).
		Return(testlib.NewLoggerEntry()).
		Times(1).
		After(a.Mocks.API.EC2.EXPECT().
			DescribeSecurityGroups(gomock.Any()).
			Return(nil, errors.New("invalid group id")))

	err := a.Mocks.AWS.rdsEnsureDBClusterCreated(CloudID(a.InstallationA.ID), a.VPCa, a.DBUser, a.DBPassword, a.RDSEncryptionKeyID, a.RDSEngineType, a.Mocks.Log.Logger)
	a.Assert().Error(err)
	a.Assert().Equal(err.Error(), "invalid group id")
}

func (a *AWSTestSuite) TestRDSEnsureDBClusterCreatedSubnetError() {
	a.Mocks.API.RDS.EXPECT().DescribeDBClusters(gomock.Any()).Return(nil, errors.New("db cluster does not exist")).Times(1)

	a.Mocks.Log.Logger.EXPECT().
		WithField("security-group-ids", []string{a.GroupID}).
		Return(testlib.NewLoggerEntry()).Times(1).
		After(a.Mocks.API.EC2.EXPECT().
			DescribeSecurityGroups(gomock.Any()).
			Return(&ec2.DescribeSecurityGroupsOutput{
				SecurityGroups: []*ec2.SecurityGroup{{GroupId: &a.GroupID}},
			}, nil))

	a.Mocks.Log.Logger.EXPECT().
		WithField("db-subnet-group-name", DBSubnetGroupName(a.VPCa)).
		Return(testlib.NewLoggerEntry()).
		Times(1).
		After(a.Mocks.API.RDS.EXPECT().
			DescribeDBSubnetGroups(gomock.Any()).
			Return(&rds.DescribeDBSubnetGroupsOutput{
				DBSubnetGroups: []*rds.DBSubnetGroup{},
			}, errors.New("invalid cluster id")))

	err := a.Mocks.AWS.rdsEnsureDBClusterCreated(CloudID(a.InstallationA.ID), a.VPCa, a.DBUser, a.DBPassword, a.RDSEncryptionKeyID, a.RDSEngineType, a.Mocks.Log.Logger)

	a.Assert().Error(err)
	a.Assert().Equal(err.Error(), "invalid cluster id")
}

func (a *AWSTestSuite) TestRDSEnsureDBClusterCreatedError() {
	a.Mocks.API.RDS.EXPECT().DescribeDBClusters(gomock.Any()).Return(nil, errors.New("db cluster does not exist")).Times(1)

	a.Mocks.Log.Logger.EXPECT().
		WithField("security-group-ids", []string{a.GroupID}).
		Return(testlib.NewLoggerEntry()).Times(1).
		After(a.Mocks.API.EC2.EXPECT().
			DescribeSecurityGroups(gomock.Any()).
			Return(&ec2.DescribeSecurityGroupsOutput{
				SecurityGroups: []*ec2.SecurityGroup{{GroupId: &a.GroupID}},
			}, nil))

	a.Mocks.Log.Logger.EXPECT().
		WithField("db-subnet-group-name", DBSubnetGroupName(a.VPCa)).
		Return(testlib.NewLoggerEntry()).
		Times(1).
		After(a.Mocks.API.RDS.EXPECT().
			DescribeDBSubnetGroups(gomock.Any()).
			Return(&rds.DescribeDBSubnetGroupsOutput{
				DBSubnetGroups: []*rds.DBSubnetGroup{
					{
						DBSubnetGroupName: aws.String(DBSubnetGroupName(a.VPCa)),
					},
				},
			}, nil))

	a.Mocks.API.RDS.EXPECT().
		CreateDBCluster(gomock.Any()).
		Return(nil, errors.New("invalid cluster name")).
		Times(1)

	// Retrive the Availability Zones.
	a.Mocks.API.EC2.EXPECT().DescribeAvailabilityZones(gomock.Any()).
		Return(&ec2.DescribeAvailabilityZonesOutput{AvailabilityZones: []*ec2.AvailabilityZone{{ZoneName: aws.String("us-honk-1a")}, {ZoneName: aws.String("us-honk-1b")}}}, nil).
		Times(1)

	err := a.Mocks.AWS.rdsEnsureDBClusterCreated(CloudID(a.InstallationA.ID), a.VPCa, a.DBUser, a.DBPassword, a.RDSEncryptionKeyID, a.RDSEngineType, a.Mocks.Log.Logger)
	a.Assert().Error(err)
	a.Assert().Equal(err.Error(), "invalid cluster name")
}

func (a *AWSTestSuite) TestRDSEnsureDBClusterInstanceCreated() {
	a.Mocks.Log.Logger.EXPECT().
		Infof("Provisioning AWS RDS database with ID %s", CloudID(a.InstallationA.ID)).
		Return().
		Times(1)

	a.Mocks.API.RDS.EXPECT().
		DescribeDBInstances(gomock.Any()).
		Return(nil, errors.New("db cluster instance does not exist")).
		Do(func(input *rds.DescribeDBInstancesInput) {
			a.Assert().Equal(*input.DBInstanceIdentifier, RDSMasterInstanceID(a.InstallationA.ID))
		})

	a.Mocks.Log.Logger.EXPECT().WithField("db-instance-name", RDSMasterInstanceID(a.InstallationA.ID)).
		Return(testlib.NewLoggerEntry()).
		Times(1).
		After(a.Mocks.API.RDS.EXPECT().
			CreateDBInstance(gomock.Any()).Return(nil, nil).
			Do(func(input *rds.CreateDBInstanceInput) {
				a.Assert().Equal(*input.DBClusterIdentifier, CloudID(a.InstallationA.ID))
				a.Assert().Equal(*input.DBInstanceIdentifier, RDSMasterInstanceID(a.InstallationA.ID))
			}).
			Times(1))

	err := a.Mocks.AWS.rdsEnsureDBClusterInstanceCreated(CloudID(a.InstallationA.ID), RDSMasterInstanceID(a.InstallationA.ID), a.RDSEngineType, a.Mocks.Log.Logger)
	a.Assert().NoError(err)
}

func (a *AWSTestSuite) TestRDSEnsureDBClusterInstanceAlreadyExistError() {
	a.Mocks.Log.Logger.EXPECT().
		Infof("Provisioning AWS RDS database with ID %s", CloudID(a.InstallationA.ID)).
		Return().
		Times(1)

	a.Mocks.API.RDS.EXPECT().
		DescribeDBInstances(gomock.Any()).
		Return(nil, nil).
		Do(func(input *rds.DescribeDBInstancesInput) {
			a.Assert().Equal(*input.DBInstanceIdentifier, RDSMasterInstanceID(a.InstallationA.ID))
		})

	a.Mocks.Log.Logger.EXPECT().WithField("db-instance-name", RDSMasterInstanceID(a.InstallationA.ID)).
		Return(testlib.NewLoggerEntry()).
		Times(1)

	err := a.Mocks.AWS.rdsEnsureDBClusterInstanceCreated(CloudID(a.InstallationA.ID), RDSMasterInstanceID(a.InstallationA.ID), a.RDSEngineType, a.Mocks.Log.Logger)
	a.Assert().NoError(err)
}

func (a *AWSTestSuite) TestRDSEnsureDBClusterInstanceCreateError() {
	a.Mocks.Log.Logger.EXPECT().
		Infof("Provisioning AWS RDS database with ID %s", CloudID(a.InstallationA.ID)).
		Return().
		Times(1)

	a.Mocks.API.RDS.EXPECT().
		DescribeDBInstances(gomock.Any()).
		Return(nil, errors.New("db cluster instance does not exist")).
		Do(func(input *rds.DescribeDBInstancesInput) {
			a.Assert().Equal(*input.DBInstanceIdentifier, RDSMasterInstanceID(a.InstallationA.ID))
		})

	a.Mocks.Log.Logger.EXPECT().WithField("db-instance-name", RDSMasterInstanceID(a.InstallationA.ID)).
		Return(testlib.NewLoggerEntry()).
		Times(1).
		After(a.Mocks.API.RDS.EXPECT().
			CreateDBInstance(gomock.Any()).Return(nil, errors.New("instance creation failure")).
			Do(func(input *rds.CreateDBInstanceInput) {
				a.Assert().Equal(*input.DBClusterIdentifier, CloudID(a.InstallationA.ID))
				a.Assert().Equal(*input.DBInstanceIdentifier, RDSMasterInstanceID(a.InstallationA.ID))
			}).
			Times(1))

	err := a.Mocks.AWS.rdsEnsureDBClusterInstanceCreated(CloudID(a.InstallationA.ID), RDSMasterInstanceID(a.InstallationA.ID), a.RDSEngineType, a.Mocks.Log.Logger)

	a.Assert().Error(err)
	a.Assert().Equal(err.Error(), "instance creation failure")
}
