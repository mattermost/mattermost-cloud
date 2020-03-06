package aws

import (
	"fmt"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/aws/aws-sdk-go/service/rds"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/mock"
)

func (a *AWSTestSuite) TestRDSEnsureDBClusterCreated() {
	a.SetDescribeDBClustersNotFoundExpectation().Once()
	a.SetDescribeSecurityGroupsExpectation().Once()
	a.SetDescribeDBSubnetGroupsExpectation(a.VPCa).Once()
	a.SetCreateDBClusterExpectation(a.InstallationA.ID).Return(nil, nil).Once()
	a.Mocks.LOG.WithFieldArgs("security-group-ids", a.GroupID).Once()
	a.Mocks.LOG.WithFieldString("db-subnet-group-name", SubnetGroupName(a.VPCa)).Once()
	a.Mocks.LOG.WithFieldString("db-cluster-name", CloudID(a.InstallationA.ID)).Once()

	err := a.Mocks.AWS.rdsEnsureDBClusterCreated(CloudID(a.InstallationA.ID), a.VPCa, a.DBUser, a.DBPassword, a.Mocks.LOG.Logger)

	a.Assert().NoError(err)
	a.Mocks.API.EC2.AssertExpectations(a.T())
	a.Mocks.API.RDS.AssertExpectations(a.T())
}

func (a *AWSTestSuite) TestRDSEnsureDBClusterCreatedAlreadyCreated() {
	a.SetDescribeDBClustersFoundExpectation().Once()
	a.Mocks.LOG.WithFieldString("db-cluster-name", CloudID(a.InstallationA.ID)).Once()

	err := a.Mocks.AWS.rdsEnsureDBClusterCreated(CloudID(a.InstallationA.ID), a.VPCa, a.DBUser, a.DBPassword, a.Mocks.LOG.Logger)

	a.Assert().NoError(err)
	a.Mocks.API.EC2.AssertNotCalled(a.T(), "DescribeSecurityGroups")
	a.Mocks.API.RDS.AssertNotCalled(a.T(), "DescribeDBSubnetGroups")
	a.Mocks.API.RDS.AssertNotCalled(a.T(), "CreateDBCluster")
}

func (a *AWSTestSuite) TestRDSEnsureDBClusterCreatedWithSGError() {
	a.SetDescribeDBClustersNotFoundExpectation().Once()
	a.SetDescribeSecurityGroupsErrorExpectation().Once()

	err := a.Mocks.AWS.rdsEnsureDBClusterCreated(CloudID(a.InstallationA.ID), a.VPCa, a.DBUser, a.DBPassword, a.Mocks.LOG.Logger)

	a.Assert().Error(err)
	a.Assert().Equal(err.Error(), "bad request")
	a.Mocks.API.RDS.AssertNotCalled(a.T(), "DescribeDBSubnetGroups")
	a.Mocks.API.RDS.AssertNotCalled(a.T(), "CreateDBCluster")
}

func (a *AWSTestSuite) TestRDSEnsureDBClusterCreatedSubnetError() {
	a.SetDescribeDBClustersNotFoundExpectation().Once()
	a.SetDescribeSecurityGroupsExpectation().Once()
	a.SetDescribeDBSubnetGroupsErrorExpectation().Once()
	a.Mocks.LOG.WithFieldArgs("security-group-ids", a.GroupID).Once()

	err := a.Mocks.AWS.rdsEnsureDBClusterCreated(CloudID(a.InstallationA.ID), a.VPCa, a.DBUser, a.DBPassword, a.Mocks.LOG.Logger)

	a.Assert().Error(err)
	a.Assert().Equal(err.Error(), "bad request")
	a.Mocks.API.RDS.AssertNotCalled(a.T(), "CreateDBCluster")
}

func (a *AWSTestSuite) TestRDSEnsureDBClusterCreatedError() {
	a.SetDescribeDBClustersNotFoundExpectation().Once()
	a.SetDescribeSecurityGroupsExpectation().Once()
	a.SetDescribeDBSubnetGroupsExpectation(a.VPCa).Once()
	a.SetCreateDBClusterExpectation(a.InstallationA.ID).Return(nil, nil).Once().Return(nil, errors.New("cannot find parameter groups")).Once()
	a.Mocks.LOG.WithFieldArgs("security-group-ids", a.GroupID).Once()
	a.Mocks.LOG.WithFieldString("db-subnet-group-name", SubnetGroupName(a.VPCa)).Once()

	err := a.Mocks.AWS.rdsEnsureDBClusterCreated(CloudID(a.InstallationA.ID), a.VPCa, a.DBUser, a.DBPassword, a.Mocks.LOG.Logger)

	a.Assert().Error(err)
	a.Assert().Equal(err.Error(), "cannot find parameter groups")
	a.Mocks.API.EC2.AssertExpectations(a.T())
	a.Mocks.API.RDS.AssertExpectations(a.T())
}

func (a *AWSTestSuite) TestRDSEnsureDBClusterInstanceCreated() {
	a.SetDescribeDBInstancesNotFoundExpectation(a.InstallationA.ID).Once()
	a.SetCreateDBInstanceExpectation(a.InstallationA.ID).Return(nil, nil)
	a.Mocks.LOG.InfofString("Provisioning AWS RDS master instance with name %s", RDSMasterInstanceID(a.InstallationA.ID)).Once()
	a.Mocks.LOG.WithFieldString("db-instance-name", RDSMasterInstanceID(a.InstallationA.ID)).Once()

	err := a.Mocks.AWS.rdsEnsureDBClusterInstanceCreated(CloudID(a.InstallationA.ID), RDSMasterInstanceID(a.InstallationA.ID), a.Mocks.LOG.Logger)

	a.Assert().NoError(err)
	a.Mocks.API.RDS.AssertExpectations(a.T())
}

func (a *AWSTestSuite) TestRDSEnsureDBClusterInstanceAlreadyExistError() {
	a.SetDescribeDBInstancesFoundExpectation(a.InstallationA.ID).Once()
	a.Mocks.LOG.InfofString("Provisioning AWS RDS master instance with name %s", RDSMasterInstanceID(a.InstallationA.ID)).Once()
	a.Mocks.LOG.WithFieldString("db-instance-name", RDSMasterInstanceID(a.InstallationA.ID)).Once()

	err := a.Mocks.AWS.rdsEnsureDBClusterInstanceCreated(CloudID(a.InstallationA.ID), RDSMasterInstanceID(a.InstallationA.ID), a.Mocks.LOG.Logger)

	a.Assert().NoError(err)
	a.Mocks.API.RDS.AssertNotCalled(a.T(), "CreateDBInstance")
}

func (a *AWSTestSuite) TestRDSEnsureDBClusterInstanceCreateError() {
	a.SetDescribeDBInstancesNotFoundExpectation(a.InstallationA.ID).Once()
	a.SetCreateDBInstanceExpectation(a.InstallationA.ID).Return(nil, errors.New("bad request"))
	a.Mocks.LOG.InfofString("Provisioning AWS RDS master instance with name %s", RDSMasterInstanceID(a.InstallationA.ID)).Once()
	a.Mocks.LOG.WithFieldString("db-instance-name", RDSMasterInstanceID(a.InstallationA.ID)).Once()

	err := a.Mocks.AWS.rdsEnsureDBClusterInstanceCreated(CloudID(a.InstallationA.ID), RDSMasterInstanceID(a.InstallationA.ID), a.Mocks.LOG.Logger)

	a.Assert().Error(err)
	a.Assert().Equal(err.Error(), "bad request")
	a.Mocks.API.RDS.AssertNotCalled(a.T(), "CreateDBInstance")
}

// Helpers

func (a *AWSTestSuite) SetCreateDBClusterSnapshotExpectation(installationID string) *mock.Call {
	return a.Mocks.API.RDS.On("CreateDBClusterSnapshot", mock.MatchedBy(func(input *rds.CreateDBClusterSnapshotInput) bool {
		return *input.DBClusterIdentifier == CloudID(installationID) &&
			*input.Tags[0].Key == DefaultClusterInstallationSnapshotTagKey &&
			*input.Tags[0].Value == RDSSnapshotTagValue(CloudID(installationID))
	}))
}

func SubnetGroupName(vpcID string) string {
	return fmt.Sprintf("mattermost-provisioner-db-%s", vpcID)
}

func RDSMasterInstanceID(installationID string) string {
	return fmt.Sprintf("%s-master", CloudID(installationID))
}

func (a *AWSTestSuite) SetCreateDBInstanceExpectation(installationID string) *mock.Call {
	return a.Mocks.API.RDS.On("CreateDBInstance", mock.MatchedBy(func(input *rds.CreateDBInstanceInput) bool {
		return *input.DBClusterIdentifier == CloudID(installationID) &&
			*input.DBInstanceIdentifier == RDSMasterInstanceID(installationID)
	}))
}

func (a *AWSTestSuite) SetDescribeDBInstancesNotFoundExpectation(installationID string) *mock.Call {
	return a.Mocks.API.RDS.On("DescribeDBInstances", mock.MatchedBy(func(input *rds.DescribeDBInstancesInput) bool {
		return *input.DBInstanceIdentifier == RDSMasterInstanceID(installationID)
	})).Return(nil, errors.New("db cluster instance does not exist"))
}

func (a *AWSTestSuite) SetDescribeDBInstancesFoundExpectation(installationID string) *mock.Call {
	return a.Mocks.API.RDS.On("DescribeDBInstances", mock.MatchedBy(func(input *rds.DescribeDBInstancesInput) bool {
		return *input.DBInstanceIdentifier == RDSMasterInstanceID(installationID)
	})).Return(nil, nil)
}

func (a *AWSTestSuite) SetDescribeDBClustersNotFoundExpectation() *mock.Call {
	return a.Mocks.API.RDS.On("DescribeDBClusters", mock.Anything).Return(nil, errors.New("db cluster does not exist"))
}

func (a *AWSTestSuite) SetDescribeDBClustersFoundExpectation() *mock.Call {
	return a.Mocks.API.RDS.On("DescribeDBClusters", mock.Anything).Return(nil, nil)
}

func (a *AWSTestSuite) SetDescribeSecurityGroupsExpectation() *mock.Call {
	return a.Mocks.API.EC2.On("DescribeSecurityGroups", mock.AnythingOfType("*ec2.DescribeSecurityGroupsInput")).Return(&ec2.DescribeSecurityGroupsOutput{
		SecurityGroups: []*ec2.SecurityGroup{&ec2.SecurityGroup{GroupId: &a.GroupID}},
	}, nil)
}

func (a *AWSTestSuite) SetDescribeSecurityGroupsErrorExpectation() *mock.Call {
	return a.Mocks.API.EC2.On("DescribeSecurityGroups", mock.AnythingOfType("*ec2.DescribeSecurityGroupsInput")).Return(nil, errors.New("bad request"))
}

func (a *AWSTestSuite) SetDescribeDBSubnetGroupsExpectation(vpcID string) *mock.Call {
	return a.Mocks.API.RDS.On("DescribeDBSubnetGroups", mock.AnythingOfType("*rds.DescribeDBSubnetGroupsInput")).Return(&rds.DescribeDBSubnetGroupsOutput{
		DBSubnetGroups: []*rds.DBSubnetGroup{
			&rds.DBSubnetGroup{
				DBSubnetGroupName: aws.String(SubnetGroupName(vpcID)),
			},
		},
	}, nil)
}

func (a *AWSTestSuite) SetDescribeDBSubnetGroupsErrorExpectation() *mock.Call {
	return a.Mocks.API.RDS.On("DescribeDBSubnetGroups", mock.AnythingOfType("*rds.DescribeDBSubnetGroupsInput")).Return(nil, errors.New("bad request"))
}

func (a *AWSTestSuite) SetCreateDBClusterExpectation(installationID string) *mock.Call {
	return a.Mocks.API.RDS.On("CreateDBCluster", mock.MatchedBy(func(input *rds.CreateDBClusterInput) bool {
		for _, zone := range input.AvailabilityZones {
			if !a.Assert().Contains(a.RDSAvailabilityZones, *zone) {
				return false
			}
		}
		return *input.BackupRetentionPeriod == 7 &&
			*input.DBClusterIdentifier == CloudID(installationID) &&
			*input.DatabaseName == a.DBName &&
			*input.VpcSecurityGroupIds[0] == a.GroupID
	}))
}
