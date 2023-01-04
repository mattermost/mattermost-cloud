// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.
//

package aws

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	ec2Types "github.com/aws/aws-sdk-go-v2/service/ec2/types"
	"github.com/aws/aws-sdk-go-v2/service/rds"
	rdsTypes "github.com/aws/aws-sdk-go-v2/service/rds/types"
	"github.com/golang/mock/gomock"
	"github.com/mattermost/mattermost-cloud/internal/testlib"
	"github.com/pkg/errors"
)

func (a *AWSTestSuite) TestRDSEnsureDBClusterCreated() {
	tags, err := NewTags("tag1", "value1", "tag2", "value2")
	a.Assert().NoError(err)

	a.Mocks.API.RDS.EXPECT().DescribeDBClusters(gomock.Any(), gomock.Any()).Return(nil, errors.New("db cluster does not exist")).Times(1)

	a.Mocks.Log.Logger.EXPECT().
		WithField(gomock.Any(), gomock.Any()).
		Return(testlib.NewLoggerEntry()).
		Times(3)

	a.Mocks.API.EC2.EXPECT().
		DescribeSecurityGroups(context.TODO(), gomock.Any()).
		Return(&ec2.DescribeSecurityGroupsOutput{
			SecurityGroups: []ec2Types.SecurityGroup{
				{
					GroupId: &a.GroupID,
				},
			},
		}, nil)

	a.Mocks.API.RDS.EXPECT().
		DescribeDBSubnetGroups(gomock.Any(), gomock.Any()).
		Return(&rds.DescribeDBSubnetGroupsOutput{
			DBSubnetGroups: []rdsTypes.DBSubnetGroup{
				{
					DBSubnetGroupName: aws.String(DBSubnetGroupName(a.VPCa)),
				},
			},
		}, nil)

	a.Mocks.API.RDS.EXPECT().
		CreateDBCluster(gomock.Any(), gomock.Any()).
		Return(nil, nil).
		Do(func(ctx context.Context, input *rds.CreateDBClusterInput, optFns ...func(*rds.Options)) {
			for _, zone := range input.AvailabilityZones {
				a.Assert().Contains(a.RDSAvailabilityZones, zone)
			}
			a.Assert().Equal(*input.BackupRetentionPeriod, int32(7))
			a.Assert().Equal(*input.DBClusterIdentifier, CloudID(a.InstallationA.ID))
			a.Assert().Equal(*input.DatabaseName, a.DBName)
			a.Assert().Equal(input.VpcSecurityGroupIds[0], a.GroupID)
			a.Assert().Len(input.Tags, tags.Len())
		}).
		Times(1)

	// Retrive the Availability Zones.
	a.Mocks.API.EC2.EXPECT().DescribeAvailabilityZones(context.TODO(), gomock.Any()).
		Return(&ec2.DescribeAvailabilityZonesOutput{AvailabilityZones: []ec2Types.AvailabilityZone{{ZoneName: aws.String("us-honk-1a")}, {ZoneName: aws.String("us-honk-1b")}}}, nil).
		Times(1)

	err = a.Mocks.AWS.rdsEnsureDBClusterCreated(CloudID(a.InstallationA.ID), a.VPCa, a.DBUser, a.DBPassword, a.RDSEncryptionKeyID, a.RDSEngineType, tags, a.Mocks.Log.Logger)
	a.Assert().NoError(err)
}

func (a *AWSTestSuite) TestRDSEnsureDBClusterCreatedAlreadyCreated() {
	a.Mocks.Log.Logger.EXPECT().
		WithField("db-cluster-name", CloudID(a.InstallationA.ID)).
		Return(testlib.NewLoggerEntry()).
		Times(1).
		After(a.Mocks.API.RDS.EXPECT().
			DescribeDBClusters(gomock.Any(), gomock.Any()).
			Return(nil, nil).
			Times(1))

	err := a.Mocks.AWS.rdsEnsureDBClusterCreated(CloudID(a.InstallationA.ID), a.VPCa, a.DBUser, a.DBPassword, a.RDSEncryptionKeyID, a.RDSEngineType, &Tags{}, a.Mocks.Log.Logger)
	a.Assert().NoError(err)
}

func (a *AWSTestSuite) TestRDSEnsureDBClusterCreatedWithSGError() {
	a.Mocks.API.RDS.EXPECT().
		DescribeDBClusters(gomock.Any(), gomock.Any()).
		Return(nil, errors.New("db cluster does not exist")).
		Times(1)

	a.Mocks.API.EC2.EXPECT().
		DescribeSecurityGroups(context.TODO(), gomock.Any()).
		Return(nil, errors.New("invalid group id"))

	err := a.Mocks.AWS.rdsEnsureDBClusterCreated(CloudID(a.InstallationA.ID), a.VPCa, a.DBUser, a.DBPassword, a.RDSEncryptionKeyID, a.RDSEngineType, &Tags{}, a.Mocks.Log.Logger)
	a.Assert().Error(err)
	a.Assert().Equal(err.Error(), "invalid group id")
}

func (a *AWSTestSuite) TestRDSEnsureDBClusterCreatedSubnetError() {
	a.Mocks.API.RDS.EXPECT().DescribeDBClusters(gomock.Any(), gomock.Any()).Return(nil, errors.New("db cluster does not exist")).Times(1)

	a.Mocks.Log.Logger.EXPECT().
		WithField("security-group-ids", []string{a.GroupID}).
		Return(testlib.NewLoggerEntry()).Times(1).
		After(a.Mocks.API.EC2.EXPECT().
			DescribeSecurityGroups(context.TODO(), gomock.Any()).
			Return(&ec2.DescribeSecurityGroupsOutput{
				SecurityGroups: []ec2Types.SecurityGroup{{GroupId: &a.GroupID}},
			}, nil))

	a.Mocks.API.RDS.EXPECT().
		DescribeDBSubnetGroups(gomock.Any(), gomock.Any()).
		Return(&rds.DescribeDBSubnetGroupsOutput{
			DBSubnetGroups: []rdsTypes.DBSubnetGroup{},
		}, errors.New("invalid cluster id"))

	err := a.Mocks.AWS.rdsEnsureDBClusterCreated(CloudID(a.InstallationA.ID), a.VPCa, a.DBUser, a.DBPassword, a.RDSEncryptionKeyID, a.RDSEngineType, &Tags{}, a.Mocks.Log.Logger)

	a.Assert().Error(err)
	a.Assert().Equal(err.Error(), "invalid cluster id")
}

func (a *AWSTestSuite) TestRDSEnsureDBClusterCreatedError() {
	a.Mocks.API.RDS.EXPECT().DescribeDBClusters(gomock.Any(), gomock.Any()).Return(nil, errors.New("db cluster does not exist")).Times(1)

	a.Mocks.Log.Logger.EXPECT().
		WithField("security-group-ids", []string{a.GroupID}).
		Return(testlib.NewLoggerEntry()).Times(1).
		After(a.Mocks.API.EC2.EXPECT().
			DescribeSecurityGroups(context.TODO(), gomock.Any()).
			Return(&ec2.DescribeSecurityGroupsOutput{
				SecurityGroups: []ec2Types.SecurityGroup{{GroupId: &a.GroupID}},
			}, nil))

	a.Mocks.Log.Logger.EXPECT().
		WithField("db-subnet-group-name", DBSubnetGroupName(a.VPCa)).
		Return(testlib.NewLoggerEntry()).
		Times(1).
		After(a.Mocks.API.RDS.EXPECT().
			DescribeDBSubnetGroups(gomock.Any(), gomock.Any()).
			Return(&rds.DescribeDBSubnetGroupsOutput{
				DBSubnetGroups: []rdsTypes.DBSubnetGroup{
					{
						DBSubnetGroupName: aws.String(DBSubnetGroupName(a.VPCa)),
					},
				},
			}, nil))

	a.Mocks.API.RDS.EXPECT().
		CreateDBCluster(gomock.Any(), gomock.Any()).
		Return(nil, errors.New("invalid cluster name")).
		Times(1)

	// Retrive the Availability Zones.
	a.Mocks.API.EC2.EXPECT().DescribeAvailabilityZones(context.TODO(), gomock.Any()).
		Return(&ec2.DescribeAvailabilityZonesOutput{AvailabilityZones: []ec2Types.AvailabilityZone{{ZoneName: aws.String("us-honk-1a")}, {ZoneName: aws.String("us-honk-1b")}}}, nil).
		Times(1)

	err := a.Mocks.AWS.rdsEnsureDBClusterCreated(CloudID(a.InstallationA.ID), a.VPCa, a.DBUser, a.DBPassword, a.RDSEncryptionKeyID, a.RDSEngineType, &Tags{}, a.Mocks.Log.Logger)
	a.Assert().Error(err)
	a.Assert().Equal(err.Error(), "invalid cluster name")
}

func (a *AWSTestSuite) TestRDSEnsureDBClusterInstanceCreated() {
	a.Mocks.API.RDS.EXPECT().
		DescribeDBInstances(gomock.Any(), gomock.Any()).
		Return(nil, errors.New("db cluster instance does not exist")).
		Do(func(ctx context.Context, input *rds.DescribeDBInstancesInput, optFns ...func(*rds.Options)) {
			a.Assert().Equal(*input.DBInstanceIdentifier, RDSMasterInstanceID(a.InstallationA.ID))
		})

	a.Mocks.Log.Logger.EXPECT().WithField("db-instance-name", RDSMasterInstanceID(a.InstallationA.ID)).
		Return(testlib.NewLoggerEntry()).
		Times(1).
		After(a.Mocks.API.RDS.EXPECT().
			CreateDBInstance(gomock.Any(), gomock.Any()).Return(nil, nil).
			Do(func(ctx context.Context, input *rds.CreateDBInstanceInput, optFns ...func(*rds.Options)) {
				a.Assert().Equal(*input.DBClusterIdentifier, CloudID(a.InstallationA.ID))
				a.Assert().Equal(*input.DBInstanceIdentifier, RDSMasterInstanceID(a.InstallationA.ID))
			}).
			Times(1))

	err := a.Mocks.AWS.rdsEnsureDBClusterInstanceCreated(CloudID(a.InstallationA.ID), RDSMasterInstanceID(a.InstallationA.ID), a.RDSEngineType, "db.r5.large", &Tags{}, a.Mocks.Log.Logger)
	a.Assert().NoError(err)
}

func (a *AWSTestSuite) TestRDSEnsureDBClusterInstanceAlreadyExistError() {
	a.Mocks.API.RDS.EXPECT().
		DescribeDBInstances(gomock.Any(), gomock.Any()).
		Return(nil, nil).
		Do(func(ctx context.Context, input *rds.DescribeDBInstancesInput, optFns ...func(*rds.Options)) {
			a.Assert().Equal(*input.DBInstanceIdentifier, RDSMasterInstanceID(a.InstallationA.ID))
		})

	a.Mocks.Log.Logger.EXPECT().WithField("db-instance-name", RDSMasterInstanceID(a.InstallationA.ID)).
		Return(testlib.NewLoggerEntry()).
		Times(1)

	err := a.Mocks.AWS.rdsEnsureDBClusterInstanceCreated(CloudID(a.InstallationA.ID), RDSMasterInstanceID(a.InstallationA.ID), a.RDSEngineType, "db.r5.large", &Tags{}, a.Mocks.Log.Logger)
	a.Assert().NoError(err)
}

func (a *AWSTestSuite) TestRDSEnsureDBClusterInstanceCreateError() {
	a.Mocks.API.RDS.EXPECT().
		DescribeDBInstances(gomock.Any(), gomock.Any()).
		Return(nil, errors.New("db cluster instance does not exist")).
		Do(func(ctx context.Context, input *rds.DescribeDBInstancesInput, optFns ...func(*rds.Options)) {
			a.Assert().Equal(*input.DBInstanceIdentifier, RDSMasterInstanceID(a.InstallationA.ID))
		})

	a.Mocks.API.RDS.EXPECT().
		CreateDBInstance(gomock.Any(), gomock.Any()).Return(nil, errors.New("instance creation failure")).
		Do(func(ctx context.Context, input *rds.CreateDBInstanceInput, optFns ...func(*rds.Options)) {
			a.Assert().Equal(*input.DBClusterIdentifier, CloudID(a.InstallationA.ID))
			a.Assert().Equal(*input.DBInstanceIdentifier, RDSMasterInstanceID(a.InstallationA.ID))
		}).
		Times(1)

	err := a.Mocks.AWS.rdsEnsureDBClusterInstanceCreated(CloudID(a.InstallationA.ID), RDSMasterInstanceID(a.InstallationA.ID), a.RDSEngineType, "db.r5.large", &Tags{}, a.Mocks.Log.Logger)

	a.Assert().Error(err)
	a.Assert().Equal(err.Error(), "instance creation failure")
}
