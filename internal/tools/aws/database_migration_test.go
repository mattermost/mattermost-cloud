package aws

import (
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/aws/aws-sdk-go/service/rds"
	"github.com/golang/mock/gomock"
	testlib "github.com/mattermost/mattermost-cloud/internal/testlib"
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
			Return(nil, awserr.New("InvalidPermission.Duplicate", "rule already exists", nil)).
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
		a.SetDescribeSecurityGroupsExpectation("vpc-123-master", "vpc-sg-123-id-master", &ec2.Tag{
			Key:   aws.String("dummy_tag)"),
			Value: aws.String(DefaultDBSecurityGroupTagValue),
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
			Return(&ec2.RevokeSecurityGroupIngressOutput{}, awserr.New("InvalidPermission.NotFound", "rule not found", nil)).
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
		a.SetDescribeSecurityGroupsExpectation("vpc-sg-123-master", "vpc-sg-123-id-master", &ec2.Tag{
			Key:   aws.String("dummy_tag)"),
			Value: aws.String(DefaultDBSecurityGroupTagValue),
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
	return a.Mocks.API.RDS.EXPECT().DescribeDBInstances(gomock.Any()).
		Return(&rds.DescribeDBInstancesOutput{
			DBInstances: []*rds.DBInstance{{
				VpcSecurityGroups: []*rds.VpcSecurityGroupMembership{{
					VpcSecurityGroupId: aws.String(vpcSecurityGroupID),
				}},
			}},
		}, nil)
}

func (a *AWSTestSuite) SetDescribeSecurityGroupsExpectation(groupID, groupName string, tag *ec2.Tag) *gomock.Call {
	return a.Mocks.API.EC2.EXPECT().DescribeSecurityGroups(gomock.Any()).
		Return(&ec2.DescribeSecurityGroupsOutput{
			SecurityGroups: []*ec2.SecurityGroup{{
				GroupId:   aws.String(groupID),
				GroupName: aws.String(groupName),
				Tags:      []*ec2.Tag{tag},
			}},
		}, nil)
}

func (a *AWSTestSuite) SetAuthorizeSecurityGroupIngress(description, groupIDMaster, groupIDSlave string) *gomock.Call {
	return a.Mocks.API.EC2.EXPECT().AuthorizeSecurityGroupIngress(gomock.Any()).
		Do(func(input *ec2.AuthorizeSecurityGroupIngressInput) {
			a.Assert().Equal(description, *input.IpPermissions[0].UserIdGroupPairs[0].Description)
			a.Assert().Equal(groupIDSlave, *input.IpPermissions[0].UserIdGroupPairs[0].GroupId)
			a.Assert().Equal(groupIDMaster, *input.GroupId)
			a.Assert().Equal("tcp", *input.IpPermissions[0].IpProtocol)
			a.Assert().Equal(int64(3306), *input.IpPermissions[0].ToPort)
			a.Assert().Equal(int64(3306), *input.IpPermissions[0].FromPort)
		})
}

func (a *AWSTestSuite) SetRevokeSecurityGroupIngress(description, groupIDMaster, groupIDSlave string) *gomock.Call {
	return a.Mocks.API.EC2.EXPECT().RevokeSecurityGroupIngress(gomock.Any()).
		Do(func(input *ec2.RevokeSecurityGroupIngressInput) {
			a.Assert().Equal(groupIDSlave, *input.IpPermissions[0].UserIdGroupPairs[0].GroupId)
			a.Assert().Equal(groupIDMaster, *input.GroupId)
			a.Assert().Equal("tcp", *input.IpPermissions[0].IpProtocol)
			a.Assert().Equal(int64(3306), *input.IpPermissions[0].ToPort)
			a.Assert().Equal(int64(3306), *input.IpPermissions[0].FromPort)
		})
}
