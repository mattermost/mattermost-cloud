package aws

import (
	"fmt"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/rds"
	gt "github.com/aws/aws-sdk-go/service/resourcegroupstaggingapi"
	"github.com/aws/aws-sdk-go/service/secretsmanager"
	"github.com/pkg/errors"

	"github.com/golang/mock/gomock"

	testlib "github.com/mattermost/mattermost-cloud/internal/testlib"
	"github.com/mattermost/mattermost-cloud/model"
)

// Tests provisioning a multitenante database. Use this test for deriving other tests.
// If tests are broken, this should be the first test to get fixed. This test assumes
// data provisioner is running in multitenant database for the first time, so pretty much
// the entire code will run here.
func (a *AWSTestSuite) TestProvisioningMultitenantDatabase() {
	database := RDSMultitenantDatabase{
		installationID: a.InstallationA.ID,
		instanceID:     a.InstanceID,
		client:         a.Mocks.AWS,
	}

	gomock.InOrder(
		a.ExpectGetMultitenantDatabases(a.InstallationA.ID, a.VPCa, model.NoInstallationsLimit).
			Return(make([]*model.MultitenantDatabase, 0), nil),

		a.Mocks.Log.Logger.
			EXPECT().
			Infof("Installation %s is not yet assigned to a multitenant database; fetching available RDS clusters from datastore", a.InstallationA.ID).
			Times(1),

		a.ExpectGetMultitenantDatabases("", a.VPCa, DefaultRDSMultitenantDatabaseCountLimit).
			Return(make([]*model.MultitenantDatabase, 0), nil),

		a.Mocks.Log.Logger.
			EXPECT().
			Infof("No multitenant databases with less than %d installations found in the datastore; fetching all available resources from AWS.", DefaultRDSMultitenantDatabaseCountLimit).
			Times(1),

		a.ExpectGetRDSClusterResourcesFromTags(a.VPCa, []string{a.RDSClusterID}),

		a.ExpectRDSEndpointStatus(DefaultRDSStatusAvailable, nil).
			Times(1),

		a.ExpectCreateMultiTenantDatabase(a.RDSClusterID, a.VPCa).
			Return(nil).
			Times(1),

		a.Mocks.Model.DatabaseInstallationStore.EXPECT().
			LockMultitenantDatabase(a.RDSClusterID, a.InstanceID).
			Return(true, nil).
			Times(1),

		a.Mocks.Model.DatabaseInstallationStore.EXPECT().
			GetMultitenantDatabase(a.RDSClusterID).
			Return(&model.MultitenantDatabase{
				ID: a.RDSClusterID,
			}, nil).
			Times(1),

		a.ExpectDescribeRDSCluster(a.RDSClusterID).
			Return(&rds.DescribeDBClustersOutput{
				DBClusters: []*rds.DBCluster{
					{
						DBClusterIdentifier: &a.RDSClusterID,
						Status:              aws.String(DefaultRDSStatusAvailable),
						Endpoint:            aws.String("aws.rds.com/mattermost"),
					},
				},
			}, nil).
			Times(1),

		a.Mocks.Log.Logger.EXPECT().
			WithField("multitenant-rds-database", MattermostRDSDatabaseName(a.InstallationA.ID)).
			Return(testlib.NewLoggerEntry()).
			Times(1),

		a.Mocks.API.SecretsManager.EXPECT().
			GetSecretValue(gomock.Any()).
			Do(func(input *secretsmanager.GetSecretValueInput) {

			}).
			Return(&secretsmanager.GetSecretValueOutput{
				SecretString: aws.String("VR1pJl2KdNyQy0sxvJYdWbDeTQH1Pk71sXLbS4sw"),
			}, nil).
			Times(1),

		a.Mocks.Model.DatabaseInstallationStore.EXPECT().
			UnlockMultitenantDatabase(a.RDSClusterID, a.InstanceID, true).
			Return(true, nil).
			Times(1),
	)

	err := database.Provision(a.Mocks.Model.DatabaseInstallationStore, a.Mocks.Log.Logger)
	a.Assert().Error(err)
	a.Assert().Equal("failed to provision multitenant database: failed to create database "+
		"cloud_id000000000000000000000000a: dial tcp: lookup aws.rds.com/mattermost: no such host", err.Error())
}

func (a *AWSTestSuite) TestFindRDSClusterForInstallation() {
	database := RDSMultitenantDatabase{
		installationID: a.InstallationA.ID,
		instanceID:     a.InstanceID,
		client:         a.Mocks.AWS,
	}

	gomock.InOrder(
		a.ExpectGetMultitenantDatabases(a.InstallationA.ID, a.VPCa, model.NoInstallationsLimit).
			Return(make([]*model.MultitenantDatabase, 0), nil),

		a.Mocks.Log.Logger.
			EXPECT().
			Infof("Installation %s is not yet assigned to a multitenant database; fetching available RDS clusters from datastore", a.InstallationA.ID).
			Times(1),

		a.ExpectGetMultitenantDatabases("", a.VPCa, DefaultRDSMultitenantDatabaseCountLimit).
			Return(make([]*model.MultitenantDatabase, 0), nil),

		a.Mocks.Log.Logger.
			EXPECT().
			Infof("No multitenant databases with less than %d installations found in the datastore; fetching all available resources from AWS.", DefaultRDSMultitenantDatabaseCountLimit).
			Times(1),

		a.ExpectGetRDSClusterResourcesFromTags(a.VPCa, []string{a.RDSClusterID}),

		a.ExpectRDSEndpointStatus(DefaultRDSStatusAvailable, nil).
			Times(1),

		a.ExpectCreateMultiTenantDatabase(a.RDSClusterID, a.VPCa).
			Return(nil).
			Times(1),

		a.Mocks.Model.DatabaseInstallationStore.EXPECT().
			LockMultitenantDatabase(a.RDSClusterID, a.InstanceID).
			Return(true, nil).
			Times(1),

		a.Mocks.Model.DatabaseInstallationStore.EXPECT().
			GetMultitenantDatabase(a.RDSClusterID).
			Return(&model.MultitenantDatabase{
				ID: a.RDSClusterID,
			}, nil).
			Times(1),

		a.ExpectDescribeRDSCluster(a.RDSClusterID).
			Return(&rds.DescribeDBClustersOutput{
				DBClusters: []*rds.DBCluster{
					{
						DBClusterIdentifier: &a.RDSClusterID,
						Status:              aws.String(DefaultRDSStatusAvailable),
						Endpoint:            aws.String("aws.rds.com/mattermost"),
					},
				},
			}, nil).
			Times(1),
	)

	result, err := database.findRDSClusterForInstallation(a.VPCa, a.Mocks.Model.DatabaseInstallationStore, a.Mocks.Log.Logger)
	a.Assert().NoError(err)
	a.Assert().NotNil(result)
	a.Assert().Equal(*result.cluster.Status, DefaultRDSStatusAvailable)
	a.Assert().Equal(*result.cluster.DBClusterIdentifier, a.RDSClusterID)
}

func (a *AWSTestSuite) TestFindRDSClusterForInstallationRDSStatusError() {
	database := RDSMultitenantDatabase{
		installationID: a.InstallationA.ID,
		instanceID:     a.InstanceID,
		client:         a.Mocks.AWS,
	}

	gomock.InOrder(
		a.ExpectGetMultitenantDatabases(a.InstallationA.ID, a.VPCa, model.NoInstallationsLimit).
			Return(make([]*model.MultitenantDatabase, 0), nil),

		a.Mocks.Log.Logger.
			EXPECT().
			Infof("Installation %s is not yet assigned to a multitenant database; fetching available RDS clusters from datastore", a.InstallationA.ID).
			Times(1),

		a.ExpectGetMultitenantDatabases("", a.VPCa, DefaultRDSMultitenantDatabaseCountLimit).
			Return(make([]*model.MultitenantDatabase, 0), nil),

		a.Mocks.Log.Logger.
			EXPECT().
			Infof("No multitenant databases with less than %d installations found in the datastore; fetching all available resources from AWS.", DefaultRDSMultitenantDatabaseCountLimit).
			Times(1),

		a.ExpectGetRDSClusterResourcesFromTags(a.VPCa, []string{a.RDSClusterID}),

		a.ExpectRDSEndpointStatus(DefaultRDSStatusAvailable, nil).
			Times(1),

		a.ExpectCreateMultiTenantDatabase(a.RDSClusterID, a.VPCa).
			Return(nil).
			Times(1),

		a.Mocks.Model.DatabaseInstallationStore.EXPECT().
			LockMultitenantDatabase(a.RDSClusterID, a.InstanceID).
			Return(true, nil).
			Times(1),

		a.Mocks.Model.DatabaseInstallationStore.EXPECT().
			GetMultitenantDatabase(a.RDSClusterID).
			Return(&model.MultitenantDatabase{
				ID: a.RDSClusterID,
			}, nil).
			Times(1),

		a.ExpectDescribeRDSCluster(a.RDSClusterID).
			Return(&rds.DescribeDBClustersOutput{
				DBClusters: []*rds.DBCluster{
					{
						DBClusterIdentifier: &a.RDSClusterID,
						Status:              aws.String("Creating"),
						Endpoint:            aws.String("aws.rds.com/mattermost"),
					},
				},
			}, nil).
			Times(1),

		a.Mocks.Model.DatabaseInstallationStore.EXPECT().
			UnlockMultitenantDatabase(a.RDSClusterID, a.InstanceID, true).
			Return(true, nil).
			Times(1),

		a.ExpectRDSClusterStatusLogError(fmt.Sprintf("AWS RDS cluster ID %s is not available (status: Creating)", a.RDSClusterID)).
			Times(1),
	)

	result, err := database.findRDSClusterForInstallation(a.VPCa, a.Mocks.Model.DatabaseInstallationStore, a.Mocks.Log.Logger)
	a.Assert().Error(err)
	a.Assert().Equal("could not validate and lock RDS cluster: unable to find a AWS RDS cluster ready for receiving a multitenant database installation", err.Error())
	a.Assert().Nil(result)
}

func (a *AWSTestSuite) TestFindRDSClusterForInstallationRDSError() {
	database := RDSMultitenantDatabase{
		installationID: a.InstallationA.ID,
		instanceID:     a.InstanceID,
		client:         a.Mocks.AWS,
	}

	gomock.InOrder(
		a.ExpectGetMultitenantDatabases(a.InstallationA.ID, a.VPCa, model.NoInstallationsLimit).
			Return(make([]*model.MultitenantDatabase, 0), nil),

		a.Mocks.Log.Logger.
			EXPECT().
			Infof("Installation %s is not yet assigned to a multitenant database; fetching available RDS clusters from datastore", a.InstallationA.ID).
			Times(1),

		a.ExpectGetMultitenantDatabases("", a.VPCa, DefaultRDSMultitenantDatabaseCountLimit).
			Return(make([]*model.MultitenantDatabase, 0), nil),

		a.Mocks.Log.Logger.
			EXPECT().
			Infof("No multitenant databases with less than %d installations found in the datastore; fetching all available resources from AWS.", DefaultRDSMultitenantDatabaseCountLimit).
			Times(1),

		a.ExpectGetRDSClusterResourcesFromTags(a.VPCa, []string{a.RDSClusterID}),

		a.ExpectRDSEndpointStatus(DefaultRDSStatusAvailable, nil).
			Times(1),

		a.ExpectCreateMultiTenantDatabase(a.RDSClusterID, a.VPCa).
			Return(nil).
			Times(1),

		a.Mocks.Model.DatabaseInstallationStore.EXPECT().
			LockMultitenantDatabase(a.RDSClusterID, a.InstanceID).
			Return(true, nil).
			Times(1),

		a.Mocks.Model.DatabaseInstallationStore.EXPECT().
			GetMultitenantDatabase(a.RDSClusterID).
			Return(&model.MultitenantDatabase{
				ID: a.RDSClusterID,
			}, nil).
			Times(1),

		a.ExpectDescribeRDSCluster(a.RDSClusterID).
			Return(nil, errors.New("aws error")).
			Times(1),

		a.Mocks.Model.DatabaseInstallationStore.EXPECT().
			UnlockMultitenantDatabase(a.RDSClusterID, a.InstanceID, true).
			Return(true, nil).
			Times(1),

		a.ExpectRDSClusterStatusLogError(fmt.Sprintf("failed to get RDS DB cluster from AWS: failed to get DB cluster for AWS RDS Cluster ID %s: aws error", a.RDSClusterID)).
			Times(1),
	)

	result, err := database.findRDSClusterForInstallation(a.VPCa, a.Mocks.Model.DatabaseInstallationStore, a.Mocks.Log.Logger)
	a.Assert().Error(err)
	a.Assert().Equal("could not validate and lock RDS cluster: unable to find a AWS RDS cluster ready for receiving a multitenant database installation", err.Error())
	a.Assert().Nil(result)
}

func (a *AWSTestSuite) TestFindRDSClusterForInstallationGetDatabaseNil() {
	database := RDSMultitenantDatabase{
		installationID: a.InstallationA.ID,
		instanceID:     a.InstanceID,
		client:         a.Mocks.AWS,
	}

	gomock.InOrder(
		a.ExpectGetMultitenantDatabases(a.InstallationA.ID, a.VPCa, model.NoInstallationsLimit).
			Return(make([]*model.MultitenantDatabase, 0), nil),

		a.Mocks.Log.Logger.
			EXPECT().
			Infof("Installation %s is not yet assigned to a multitenant database; fetching available RDS clusters from datastore", a.InstallationA.ID).
			Times(1),

		a.ExpectGetMultitenantDatabases("", a.VPCa, DefaultRDSMultitenantDatabaseCountLimit).
			Return(make([]*model.MultitenantDatabase, 0), nil),

		a.Mocks.Log.Logger.
			EXPECT().
			Infof("No multitenant databases with less than %d installations found in the datastore; fetching all available resources from AWS.", DefaultRDSMultitenantDatabaseCountLimit).
			Times(1),

		a.ExpectGetRDSClusterResourcesFromTags(a.VPCa, []string{a.RDSClusterID}),

		a.ExpectRDSEndpointStatus(DefaultRDSStatusAvailable, nil).
			Times(1),

		a.ExpectCreateMultiTenantDatabase(a.RDSClusterID, a.VPCa).
			Return(nil).
			Times(1),

		a.Mocks.Model.DatabaseInstallationStore.EXPECT().
			LockMultitenantDatabase(a.RDSClusterID, a.InstanceID).
			Return(true, nil).
			Times(1),

		a.Mocks.Model.DatabaseInstallationStore.EXPECT().
			GetMultitenantDatabase(a.RDSClusterID).
			Return(nil, nil).
			Times(1),

		a.Mocks.Model.DatabaseInstallationStore.EXPECT().
			UnlockMultitenantDatabase(a.RDSClusterID, a.InstanceID, true).
			Return(true, nil).
			Times(1),

		a.ExpectRDSClusterStatusLogError(fmt.Sprintf("multitenant database validation failed: unable to find a multitenant database ID %s", a.RDSClusterID)).
			Times(1),
	)

	result, err := database.findRDSClusterForInstallation(a.VPCa, a.Mocks.Model.DatabaseInstallationStore, a.Mocks.Log.Logger)
	a.Assert().Error(err)
	a.Assert().Equal("could not validate and lock RDS cluster: unable to find a AWS RDS cluster ready for receiving a multitenant database installation", err.Error())
	a.Assert().Nil(result)
}

func (a *AWSTestSuite) TestFindRDSClusterForInstallationGetDatabaseError() {
	database := RDSMultitenantDatabase{
		installationID: a.InstallationA.ID,
		instanceID:     a.InstanceID,
		client:         a.Mocks.AWS,
	}

	gomock.InOrder(
		a.ExpectGetMultitenantDatabases(a.InstallationA.ID, a.VPCa, model.NoInstallationsLimit).
			Return(make([]*model.MultitenantDatabase, 0), nil),

		a.Mocks.Log.Logger.
			EXPECT().
			Infof("Installation %s is not yet assigned to a multitenant database; fetching available RDS clusters from datastore", a.InstallationA.ID).
			Times(1),

		a.ExpectGetMultitenantDatabases("", a.VPCa, DefaultRDSMultitenantDatabaseCountLimit).
			Return(make([]*model.MultitenantDatabase, 0), nil),

		a.Mocks.Log.Logger.
			EXPECT().
			Infof("No multitenant databases with less than %d installations found in the datastore; fetching all available resources from AWS.", DefaultRDSMultitenantDatabaseCountLimit).
			Times(1),

		a.ExpectGetRDSClusterResourcesFromTags(a.VPCa, []string{a.RDSClusterID}),

		a.ExpectRDSEndpointStatus(DefaultRDSStatusAvailable, nil).
			Times(1),

		a.ExpectCreateMultiTenantDatabase(a.RDSClusterID, a.VPCa).
			Return(nil).
			Times(1),

		a.Mocks.Model.DatabaseInstallationStore.EXPECT().
			LockMultitenantDatabase(a.RDSClusterID, a.InstanceID).
			Return(true, nil).
			Times(1),

		a.Mocks.Model.DatabaseInstallationStore.EXPECT().
			GetMultitenantDatabase(a.RDSClusterID).
			Return(nil, errors.New("data store error")).
			Times(1),

		a.Mocks.Model.DatabaseInstallationStore.EXPECT().
			UnlockMultitenantDatabase(a.RDSClusterID, a.InstanceID, true).
			Return(true, nil).
			Times(1),

		a.ExpectRDSClusterStatusLogError(fmt.Sprintf("multitenant database validation failed: failed to get multitenant database ID %s: data store error", a.RDSClusterID)).
			Times(1),
	)

	result, err := database.findRDSClusterForInstallation(a.VPCa, a.Mocks.Model.DatabaseInstallationStore, a.Mocks.Log.Logger)
	a.Assert().Error(err)
	a.Assert().Equal("could not validate and lock RDS cluster: unable to find a AWS RDS cluster ready for receiving a multitenant database installation", err.Error())
	a.Assert().Nil(result)
}

func (a *AWSTestSuite) TestFindRDSClusterForInstallationLockerError() {
	database := RDSMultitenantDatabase{
		installationID: a.InstallationA.ID,
		instanceID:     a.InstanceID,
		client:         a.Mocks.AWS,
	}

	gomock.InOrder(
		a.ExpectGetMultitenantDatabases(a.InstallationA.ID, a.VPCa, model.NoInstallationsLimit).
			Return(make([]*model.MultitenantDatabase, 0), nil),

		a.Mocks.Log.Logger.
			EXPECT().
			Infof("Installation %s is not yet assigned to a multitenant database; fetching available RDS clusters from datastore", a.InstallationA.ID).
			Times(1),

		a.ExpectGetMultitenantDatabases("", a.VPCa, DefaultRDSMultitenantDatabaseCountLimit).
			Return(make([]*model.MultitenantDatabase, 0), nil),

		a.Mocks.Log.Logger.
			EXPECT().
			Infof("No multitenant databases with less than %d installations found in the datastore; fetching all available resources from AWS.", DefaultRDSMultitenantDatabaseCountLimit).
			Times(1),

		a.ExpectGetRDSClusterResourcesFromTags(a.VPCa, []string{a.RDSClusterID}),

		a.ExpectRDSEndpointStatus(DefaultRDSStatusAvailable, nil).
			Times(1),

		a.ExpectCreateMultiTenantDatabase(a.RDSClusterID, a.VPCa).
			Return(nil).
			Times(1),

		a.Mocks.Model.DatabaseInstallationStore.EXPECT().
			LockMultitenantDatabase(a.RDSClusterID, a.InstanceID).
			Return(true, errors.New("locker error")).
			Times(1),

		a.ExpectRDSClusterStatusLogError(fmt.Sprintf("failed to lock multitenant database ID %s: failed to acquire a lock for multitenant database ID %s: locker error", a.RDSClusterID, a.RDSClusterID)).
			Times(1),
	)

	result, err := database.findRDSClusterForInstallation(a.VPCa, a.Mocks.Model.DatabaseInstallationStore, a.Mocks.Log.Logger)
	a.Assert().Error(err)
	a.Assert().Equal("could not validate and lock RDS cluster: unable to find a AWS RDS cluster ready for receiving a multitenant database installation", err.Error())
	a.Assert().Nil(result)
}

func (a *AWSTestSuite) TestFindRDSClusterForInstallationNotLocked() {
	database := RDSMultitenantDatabase{
		installationID: a.InstallationA.ID,
		instanceID:     a.InstanceID,
		client:         a.Mocks.AWS,
	}

	gomock.InOrder(
		a.ExpectGetMultitenantDatabases(a.InstallationA.ID, a.VPCa, model.NoInstallationsLimit).
			Return(make([]*model.MultitenantDatabase, 0), nil),

		a.Mocks.Log.Logger.
			EXPECT().
			Infof("Installation %s is not yet assigned to a multitenant database; fetching available RDS clusters from datastore", a.InstallationA.ID).
			Times(1),

		a.ExpectGetMultitenantDatabases("", a.VPCa, DefaultRDSMultitenantDatabaseCountLimit).
			Return(make([]*model.MultitenantDatabase, 0), nil),

		a.Mocks.Log.Logger.
			EXPECT().
			Infof("No multitenant databases with less than %d installations found in the datastore; fetching all available resources from AWS.", DefaultRDSMultitenantDatabaseCountLimit).
			Times(1),

		a.ExpectGetRDSClusterResourcesFromTags(a.VPCa, []string{a.RDSClusterID}),

		a.ExpectRDSEndpointStatus(DefaultRDSStatusAvailable, nil).
			Times(1),

		a.ExpectCreateMultiTenantDatabase(a.RDSClusterID, a.VPCa).
			Return(nil).
			Times(1),

		a.Mocks.Model.DatabaseInstallationStore.EXPECT().
			LockMultitenantDatabase(a.RDSClusterID, a.InstanceID).
			Return(false, nil).
			Times(1),

		a.ExpectRDSClusterStatusLogError(fmt.Sprintf("failed to lock multitenant database ID %s: unable to lock multitenant database ID %s", a.RDSClusterID, a.RDSClusterID)).
			Times(1),
	)

	result, err := database.findRDSClusterForInstallation(a.VPCa, a.Mocks.Model.DatabaseInstallationStore, a.Mocks.Log.Logger)
	a.Assert().Error(err)
	a.Assert().Equal("could not validate and lock RDS cluster: unable to find a AWS RDS cluster ready for receiving a multitenant database installation", err.Error())
	a.Assert().Nil(result)
}

func (a *AWSTestSuite) TestFindRDSClusterForInstallationCreatingError() {
	database := RDSMultitenantDatabase{
		installationID: a.InstallationA.ID,
		instanceID:     a.InstanceID,
		client:         a.Mocks.AWS,
	}

	gomock.InOrder(
		a.ExpectGetMultitenantDatabases(a.InstallationA.ID, a.VPCa, model.NoInstallationsLimit).
			Return(make([]*model.MultitenantDatabase, 0), nil),

		a.Mocks.Log.Logger.
			EXPECT().
			Infof("Installation %s is not yet assigned to a multitenant database; fetching available RDS clusters from datastore", a.InstallationA.ID).
			Times(1),

		a.ExpectGetMultitenantDatabases("", a.VPCa, DefaultRDSMultitenantDatabaseCountLimit).
			Return(make([]*model.MultitenantDatabase, 0), nil),

		a.Mocks.Log.Logger.
			EXPECT().
			Infof("No multitenant databases with less than %d installations found in the datastore; fetching all available resources from AWS.", DefaultRDSMultitenantDatabaseCountLimit).
			Times(1),

		a.ExpectGetRDSClusterResourcesFromTags(a.VPCa, []string{a.RDSClusterID}),

		a.ExpectRDSEndpointStatus(DefaultRDSStatusAvailable, nil).
			Times(1),

		a.ExpectCreateMultiTenantDatabase(a.RDSClusterID, a.VPCa).
			Return(errors.New("failed to create multitenant database")).
			Times(1),

		a.ExpectRDSClusterStatusLogError("failed to create multitenant database").
			Times(1),
	)

	result, err := database.findRDSClusterForInstallation(a.VPCa, a.Mocks.Model.DatabaseInstallationStore, a.Mocks.Log.Logger)
	a.Assert().Error(err)
	a.Assert().Equal("could not validate and lock RDS cluster: unable to find a AWS RDS cluster ready for receiving a multitenant database installation", err.Error())
	a.Assert().Nil(result)
}

// Helpers

func (a *AWSTestSuite) ExpectRDSClusterStatusLogError(msg string) *gomock.Call {
	return a.Mocks.Log.Logger.
		EXPECT().
		WithError(gomock.Any()).
		Do(func(input error) {
			a.Assert().Equal(msg, input.Error())
		}).
		Return(testlib.NewLoggerEntry())
}

func (a *AWSTestSuite) ExpectDescribeRDSCluster(rdsClusterID string) *gomock.Call {
	return a.Mocks.API.RDS.EXPECT().
		DescribeDBClusters(gomock.Any()).
		Do(func(input *rds.DescribeDBClustersInput) {
			a.Assert().Equal(input.Filters, []*rds.Filter{
				{
					Name:   aws.String("db-cluster-id"),
					Values: []*string{&rdsClusterID},
				},
			})
		})
}

func (a *AWSTestSuite) ExpectCreateMultiTenantDatabase(rdsClusterID, vpcID string) *gomock.Call {
	return a.Mocks.Model.DatabaseInstallationStore.EXPECT().
		CreateMultitenantDatabase(gomock.Any()).
		Do(func(input *model.MultitenantDatabase) {
			a.Assert().Equal(input.ID, rdsClusterID)
			a.Assert().Equal(input.VpcID, vpcID)
		})
}

func (a *AWSTestSuite) ExpectRDSEndpointStatus(status string, err error) *gomock.Call {
	if err != nil {
		return a.Mocks.API.RDS.EXPECT().
			DescribeDBClusterEndpoints(gomock.Any()).
			Return(nil, err)
	}

	return a.Mocks.API.RDS.EXPECT().
		DescribeDBClusterEndpoints(gomock.Any()).
		Return(&rds.DescribeDBClusterEndpointsOutput{
			DBClusterEndpoints: []*rds.DBClusterEndpoint{
				{
					Status: aws.String(status),
				},
			},
		}, nil)
}

func (a *AWSTestSuite) ExpectGetMultitenantDatabases(installationID, vpcID string, numOfInstallationsLimit int) *gomock.Call {
	return a.Mocks.Model.DatabaseInstallationStore.EXPECT().
		GetMultitenantDatabases(gomock.Any()).
		Do(func(input *model.MultitenantDatabaseFilter) {
			if installationID != "" {
				a.Assert().Equal(installationID, input.InstallationID)
			}
			if vpcID != "" {
				a.Assert().Equal(vpcID, input.VpcID)
			}
			a.Assert().Equal(numOfInstallationsLimit, input.NumOfInstallationsLimit)
			a.Assert().Equal(input.PerPage, model.AllPerPage)
		})
}

func (a *AWSTestSuite) ExpectGetRDSClusterResourcesFromTags(vpcID string, rdsClusterIDs []string) *gomock.Call {
	resourceTags := make([]*gt.ResourceTagMapping, len(rdsClusterIDs))
	for i, id := range rdsClusterIDs {
		resourceTags[i] = &gt.ResourceTagMapping{
			ResourceARN: aws.String(a.RDSResourceARN),
			// WARNING: If you ever find that you need to change some of the hardcoded values such as
			// Owner, Terraform or any of the keys here, make sure that an E2e tests still passes.
			Tags: []*gt.Tag{
				{
					Key:   aws.String("Purpose"),
					Value: aws.String("provisioning"),
				},
				{
					Key:   aws.String("Owner"),
					Value: aws.String("cloud-team"),
				},
				{
					Key:   aws.String("Terraform"),
					Value: aws.String("true"),
				},
				{
					Key:   aws.String("DatabaseType"),
					Value: aws.String("multitenant-rds"),
				},
				{
					Key:   aws.String("VpcID"),
					Value: aws.String(vpcID),
				},
				{
					Key:   aws.String("Counter"),
					Value: aws.String("0"),
				},
				{
					Key:   aws.String("MultitenantDatabaseID"),
					Value: aws.String(id),
				},
			},
		}
	}

	return a.Mocks.API.ResourceGroupsTagging.EXPECT().
		GetResources(gomock.Any()).
		Do(func(input *gt.GetResourcesInput) {
			a.Assert().Equal(input.ResourceTypeFilters, []*string{aws.String(DefaultResourceTypeClusterRDS)})
			tagFilter := []*gt.TagFilter{
				{
					Key:    aws.String("Purpose"),
					Values: []*string{aws.String("provisioning")},
				},
				{
					Key:    aws.String("Owner"),
					Values: []*string{aws.String("cloud-team")},
				},
				{
					Key:    aws.String("Terraform"),
					Values: []*string{aws.String("true")},
				},
				{
					Key:    aws.String("DatabaseType"),
					Values: []*string{aws.String("multitenant-rds")},
				},
				{
					Key:    aws.String("VpcID"),
					Values: []*string{aws.String(vpcID)},
				},
				{
					Key: aws.String("Counter"),
				},
				{
					Key: aws.String("MultitenantDatabaseID"),
				},
			}
			a.Assert().Equal(input.TagFilters, tagFilter)
		}).
		Return(&gt.GetResourcesOutput{
			ResourceTagMappingList: resourceTags,
		}, nil)
}
