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
	"github.com/aws/smithy-go"
	"github.com/golang/mock/gomock"
	"github.com/mattermost/mattermost-cloud/internal/testlib"
	"github.com/mattermost/mattermost-cloud/model"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

func (a *AWSTestSuite) TestDatabaseRDSMigrationSetup() {
	database := NewRDSDatabaseMigration(a.InstallationA.ID, a.ClusterInstallationB.InstallationID, a.Mocks.AWS)

	gomock.InOrder(
		a.SetDescribeDBInstancesExpectation("sg-123-id-master").Times(1),
		a.SetDescribeSecurityGroupsExpectation("vpc-db-sg-123-master", "vpc-db-sg-123-id-master", a.DefaultRDSTag).Times(1),
		a.SetDescribeDBInstancesExpectation("sg-123-id-slave").Times(1),
		a.SetDescribeSecurityGroupsExpectation("vpc-db-sg-123-slave", "vpc-db-sg-123-id-slave", a.DefaultRDSTag).Times(1),
		a.SetAuthorizeSecurityGroupIngress("Ingress Traffic from other RDS instance", "vpc-db-sg-123-master", "vpc-db-sg-123-slave").
			Return(&ec2.AuthorizeSecurityGroupIngressOutput{}, nil).
			Times(1),
		a.Mocks.Log.Logger.EXPECT().
			WithFields(logrus.Fields{"master-installation-id": a.InstallationA.ID, "slave-installation-id": a.ClusterInstallationB.InstallationID}).
			Return(testlib.NewLoggerEntry()).
			Times(1),
	)

	status, err := database.Setup(a.Mocks.Log.Logger)
	a.Assert().NoError(err)
	a.Assert().Equal(model.DatabaseMigrationStatusSetupComplete, status)
}

func (a *AWSTestSuite) TestDatabaseRDSMigrationSetupAlreadyExist() {
	database := NewRDSDatabaseMigration(a.InstallationA.ID, a.ClusterInstallationB.InstallationID, a.Mocks.AWS)

	gomock.InOrder(
		a.SetDescribeDBInstancesExpectation("sg-123-id-master").Times(1),
		a.SetDescribeSecurityGroupsExpectation("vpc-db-sg-123-master", "vpc-db-sg-123-id-master", a.DefaultRDSTag).Times(1),
		a.SetDescribeDBInstancesExpectation("sg-123-id-slave").Times(1),
		a.SetDescribeSecurityGroupsExpectation("vpc-db-sg-123-slave", "vpc-db-sg-123-id-slave", a.DefaultRDSTag).Times(1),
		a.SetAuthorizeSecurityGroupIngress("Ingress Traffic from other RDS instance", "vpc-db-sg-123-master", "vpc-db-sg-123-slave").
			Return(nil, errors.Wrap(&smithy.GenericAPIError{
				Code:    "InvalidPermission.Duplicate",
				Message: "rule already exists",
			}, "test")).
			Times(1),
		a.Mocks.Log.Logger.EXPECT().
			WithFields(logrus.Fields{"master-installation-id": a.InstallationA.ID, "slave-installation-id": a.ClusterInstallationB.InstallationID}).
			Return(testlib.NewLoggerEntry()).
			Times(1),
	)

	status, err := database.Setup(a.Mocks.Log.Logger)
	a.Assert().NoError(err)
	a.Assert().Equal(model.DatabaseMigrationStatusSetupComplete, status)
}

func (a *AWSTestSuite) TestDatabaseRDSMigrationSetupError() {
	database := NewRDSDatabaseMigration(a.InstallationA.ID, a.ClusterInstallationB.InstallationID, a.Mocks.AWS)

	gomock.InOrder(
		a.SetDescribeDBInstancesExpectation("sg-123-id-master").Times(1),
		a.SetDescribeSecurityGroupsExpectation("vpc-db-sg-123-master", "vpc-db-sg-123-id-master", a.DefaultRDSTag).Times(1),
		a.SetDescribeDBInstancesExpectation("sg-123-id-slave").Times(1),
		a.SetDescribeSecurityGroupsExpectation("vpc-db-sg-123-slave", "vpc-db-sg-123-id-slave", a.DefaultRDSTag).Times(1),
		a.SetAuthorizeSecurityGroupIngress("Ingress Traffic from other RDS instance", "vpc-db-sg-123-master", "vpc-db-sg-123-slave").
			Return(nil, errors.New("invalid group id")).
			Times(1),
	)

	status, err := database.Setup(a.Mocks.Log.Logger)
	a.Assert().Error(err)
	a.Assert().Equal("unable to setup database migration for master installation id: "+
		"id000000000000000000000000a and to slave installation id: id000000000000000000000000a: invalid group id", err.Error())
	a.Assert().Equal("", status)
}

func (a *AWSTestSuite) TestDatabaseRDSMigrationSetupSGNotFoundError() {
	database := NewRDSDatabaseMigration(a.InstallationA.ID, a.ClusterInstallationB.InstallationID, a.Mocks.AWS)

	gomock.InOrder(
		a.SetDescribeDBInstancesExpectation("sg-123-id-master").Times(1),
		a.SetDescribeSecurityGroupsExpectation("vpc-123-master", "vpc-sg-123-id-master", ec2Types.Tag{
			Key:   aws.String("dummy_tag)"),
			Value: aws.String(DefaultDBSecurityGroupTagMySQLValue),
		}).Times(1),
	)

	status, err := database.Setup(a.Mocks.Log.Logger)
	a.Assert().Error(err)
	a.Assert().Equal("unable to setup database migration for master installation id: id000000000000000000000000a and "+
		"to slave installation id: id000000000000000000000000a: security group for RDS DB instance cloud-id000000000000000000000000a-master not found", err.Error())
	a.Assert().Equal("", status)
}

func (a *AWSTestSuite) TestDatabaseRDSMigrationTeardown() {
	database := NewRDSDatabaseMigration(a.InstallationA.ID, a.ClusterInstallationB.InstallationID, a.Mocks.AWS)

	gomock.InOrder(
		a.SetDescribeDBInstancesExpectation("sg-123-id-master").Times(1),
		a.SetDescribeSecurityGroupsExpectation("vpc-db-sg-123-master", "vpc-db-sg-123-id-master", a.DefaultRDSTag).Times(1),
		a.SetDescribeDBInstancesExpectation("sg-123-id-slave").Times(1),
		a.SetDescribeSecurityGroupsExpectation("vpc-db-sg-123-slave", "vpc-db-sg-123-id-slave", a.DefaultRDSTag).Times(1),
		a.SetRevokeSecurityGroupIngress("Ingress Traffic from other RDS instance", "vpc-db-sg-123-master", "vpc-db-sg-123-slave").
			Return(&ec2.RevokeSecurityGroupIngressOutput{}, nil).
			Times(1),
		a.Mocks.Log.Logger.EXPECT().
			WithFields(logrus.Fields{"master-installation-id": a.InstallationA.ID, "slave-installation-id": a.ClusterInstallationB.InstallationID}).
			Return(testlib.NewLoggerEntry()).
			Times(1),
	)

	status, err := database.Teardown(a.Mocks.Log.Logger)
	a.Assert().NoError(err)
	a.Assert().Equal(model.DatabaseMigrationStatusTeardownComplete, status)
}

func (a *AWSTestSuite) TestDatabaseRDSMigrationTeardownRuleError() {
	database := NewRDSDatabaseMigration(a.InstallationA.ID, a.ClusterInstallationB.InstallationID, a.Mocks.AWS)

	gomock.InOrder(
		a.SetDescribeDBInstancesExpectation("sg-123-id-master").Times(1),
		a.SetDescribeSecurityGroupsExpectation("vpc-db-sg-123-master", "vpc-db-sg-123-id-master", a.DefaultRDSTag).Times(1),
		a.SetDescribeDBInstancesExpectation("sg-123-id-slave").Times(1),
		a.SetDescribeSecurityGroupsExpectation("vpc-db-sg-123-slave", "vpc-db-sg-123-id-slave", a.DefaultRDSTag).Times(1),
		a.SetRevokeSecurityGroupIngress("Ingress Traffic from other RDS instance", "vpc-db-sg-123-master", "vpc-db-sg-123-slave").
			Return(&ec2.RevokeSecurityGroupIngressOutput{}, errors.Wrap(&smithy.GenericAPIError{
				Code:    "InvalidPermission.NotFound",
				Message: "rule already exists",
			}, "test")).
			Times(1),
		a.Mocks.Log.Logger.EXPECT().
			WithFields(logrus.Fields{"master-installation-id": a.InstallationA.ID, "slave-installation-id": a.ClusterInstallationB.InstallationID}).
			Return(testlib.NewLoggerEntry()).
			Times(1),
	)

	status, err := database.Teardown(a.Mocks.Log.Logger)
	a.Assert().NoError(err)
	a.Assert().Equal(model.DatabaseMigrationStatusTeardownComplete, status)
}

func (a *AWSTestSuite) TestDatabaseRDSMigrationTeardownError() {
	database := NewRDSDatabaseMigration(a.InstallationA.ID, a.ClusterInstallationB.InstallationID, a.Mocks.AWS)

	gomock.InOrder(
		a.SetDescribeDBInstancesExpectation("sg-123-id-master").Times(1),
		a.SetDescribeSecurityGroupsExpectation("vpc-db-sg-123-master", "vpc-db-sg-123-id-master", a.DefaultRDSTag).Times(1),
		a.SetDescribeDBInstancesExpectation("sg-123-id-slave").Times(1),
		a.SetDescribeSecurityGroupsExpectation("vpc-db-sg-123-slave", "vpc-db-sg-123-id-slave", a.DefaultRDSTag).Times(1),
		a.SetRevokeSecurityGroupIngress("Ingress Traffic from other RDS instance", "vpc-db-sg-123-master", "vpc-db-sg-123-slave").
			Return(&ec2.RevokeSecurityGroupIngressOutput{}, errors.New("not enough permissions to revoke ingress rule")).
			Times(1),
	)

	status, err := database.Teardown(a.Mocks.Log.Logger)
	a.Assert().Error(err)
	a.Assert().Equal("unable to setup database migration for master installation id: id000000000000000000000000a and "+
		"to slave installation id: id000000000000000000000000a: not enough permissions to revoke ingress rule", err.Error())
	a.Assert().Equal("", status)
}

func (a *AWSTestSuite) TestDatabaseRDSMigrationTeardownSGNotFoundError() {
	database := NewRDSDatabaseMigration(a.InstallationA.ID, a.ClusterInstallationB.InstallationID, a.Mocks.AWS)

	gomock.InOrder(
		a.SetDescribeDBInstancesExpectation("123-id-master").Times(1),
		a.SetDescribeSecurityGroupsExpectation("vpc-sg-123-master", "vpc-sg-123-id-master", ec2Types.Tag{
			Key:   aws.String("dummy_tag)"),
			Value: aws.String(DefaultDBSecurityGroupTagMySQLValue),
		}).Times(1),
	)

	status, err := database.Teardown(a.Mocks.Log.Logger)
	a.Assert().Error(err)
	a.Assert().Equal("unable to setup database migration for master installation id: id000000000000000000000000a and to "+
		"slave installation id: id000000000000000000000000a: security group for RDS DB instance cloud-id000000000000000000000000a-master not found", err.Error())
	a.Assert().Equal("", status)
}

// Helpers

func (a *AWSTestSuite) SetDescribeDBInstancesExpectation(vpcSecurityGroupID string) *gomock.Call {
	return a.Mocks.API.RDS.EXPECT().DescribeDBInstances(gomock.Any(), gomock.Any()).
		Return(&rds.DescribeDBInstancesOutput{
			DBInstances: []rdsTypes.DBInstance{{
				VpcSecurityGroups: []rdsTypes.VpcSecurityGroupMembership{{
					VpcSecurityGroupId: aws.String(vpcSecurityGroupID),
				}},
			}},
		}, nil)
}

func (a *AWSTestSuite) SetDescribeSecurityGroupsExpectation(groupID, groupName string, tag ec2Types.Tag) *gomock.Call {
	return a.Mocks.API.EC2.EXPECT().DescribeSecurityGroups(gomock.Any(), gomock.Any()).
		Return(&ec2.DescribeSecurityGroupsOutput{
			SecurityGroups: []ec2Types.SecurityGroup{{
				GroupId:   aws.String(groupID),
				GroupName: aws.String(groupName),
				Tags:      []ec2Types.Tag{tag},
			}},
		}, nil)
}

func (a *AWSTestSuite) SetAuthorizeSecurityGroupIngress(description, groupIDMaster, groupIDSlave string) *gomock.Call {
	return a.Mocks.API.EC2.EXPECT().AuthorizeSecurityGroupIngress(gomock.Any(), gomock.Any()).
		Do(func(ctx context.Context, input *ec2.AuthorizeSecurityGroupIngressInput, optFns ...func(*ec2.Options)) {
			a.Assert().Equal(description, *input.IpPermissions[0].UserIdGroupPairs[0].Description)
			a.Assert().Equal(groupIDSlave, *input.IpPermissions[0].UserIdGroupPairs[0].GroupId)
			a.Assert().Equal(groupIDMaster, *input.GroupId)
			a.Assert().Equal("tcp", *input.IpPermissions[0].IpProtocol)
			a.Assert().Equal(int32(3306), *input.IpPermissions[0].ToPort)
			a.Assert().Equal(int32(3306), *input.IpPermissions[0].FromPort)
		})
}

func (a *AWSTestSuite) SetRevokeSecurityGroupIngress(description, groupIDMaster, groupIDSlave string) *gomock.Call {
	return a.Mocks.API.EC2.EXPECT().RevokeSecurityGroupIngress(gomock.Any(), gomock.Any()).
		Do(func(ctx context.Context, input *ec2.RevokeSecurityGroupIngressInput, optFns ...func(*ec2.Options)) {
			a.Assert().Equal(groupIDSlave, *input.IpPermissions[0].UserIdGroupPairs[0].GroupId)
			a.Assert().Equal(groupIDMaster, *input.GroupId)
			a.Assert().Equal("tcp", *input.IpPermissions[0].IpProtocol)
			a.Assert().Equal(int32(3306), *input.IpPermissions[0].ToPort)
			a.Assert().Equal(int32(3306), *input.IpPermissions[0].FromPort)
		})
}
