// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.
//

package aws

import (
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/aws/aws-sdk-go/service/rds"
	gt "github.com/aws/aws-sdk-go/service/resourcegroupstaggingapi"
	"github.com/aws/aws-sdk-go/service/secretsmanager"

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
		a.Mocks.Log.Logger.EXPECT().
			WithField("multitenant-rds-database", MattermostRDSDatabaseName(a.InstallationA.ID)).
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
		a.Mocks.API.EC2.EXPECT().DescribeVpcs(gomock.Any()).
			Return(&ec2.DescribeVpcsOutput{Vpcs: []*ec2.Vpc{{VpcId: &a.VPCa}}}, nil).
			Times(1),

		// Get multitenant databases from the datastore to check if any belongs to the installation ID.
		a.Mocks.Model.DatabaseInstallationStore.EXPECT().
			GetMultitenantDatabases(gomock.Any()).
			Do(func(input *model.MultitenantDatabaseFilter) {
				a.Assert().Equal(input.InstallationID, a.InstallationA.ID)
				a.Assert().Equal(model.NoInstallationsLimit, input.NumOfInstallationsLimit)
				a.Assert().Equal(input.PerPage, model.AllPerPage)
			}).
			Return(make([]*model.MultitenantDatabase, 0), nil),

		// Get multitenant databases from the datastore to check if any belongs to the installation ID.
		a.Mocks.Model.DatabaseInstallationStore.EXPECT().
			GetMultitenantDatabases(gomock.Any()).
			Do(func(input *model.MultitenantDatabaseFilter) {
				a.Assert().Equal(DefaultRDSMultitenantDatabaseCountLimit, input.NumOfInstallationsLimit)
				a.Assert().Equal(input.PerPage, model.AllPerPage)
			}).
			Return(make([]*model.MultitenantDatabase, 0), nil),

		// Get resources from AWS and try to find a RDS cluster that the database can be created.
		a.Mocks.API.ResourceGroupsTagging.EXPECT().
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
						Values: []*string{aws.String(a.VPCa)},
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
				ResourceTagMappingList: []*gt.ResourceTagMapping{
					{
						ResourceARN: aws.String(a.RDSResourceARN),

						// WARNING: If you ever find the need to change some of the hardcoded values such as
						// Owner, Terraform or any of the keys here, make sure that an E2e system's test still
						// passes.
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
								Value: aws.String(a.VPCa),
							},
							{
								Key:   aws.String("Counter"),
								Value: aws.String("0"),
							},
							{
								Key:   aws.String("MultitenantDatabaseID"),
								Value: aws.String(a.RDSClusterID),
							},
						},
					},
				},
			}, nil),

		a.Mocks.API.RDS.EXPECT().
			DescribeDBClusterEndpoints(gomock.Any()).
			Return(&rds.DescribeDBClusterEndpointsOutput{
				DBClusterEndpoints: []*rds.DBClusterEndpoint{
					{
						Status: aws.String("available"),
					},
				},
			}, nil).
			Times(1),

		// Create the multitenant database.
		a.Mocks.Model.DatabaseInstallationStore.EXPECT().
			CreateMultitenantDatabase(gomock.Any()).
			Do(func(input *model.MultitenantDatabase) {
				a.Assert().Equal(input.ID, a.RDSClusterID)
			}).
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

		a.Mocks.API.RDS.EXPECT().
			DescribeDBClusters(gomock.Any()).
			Do(func(input *rds.DescribeDBClustersInput) {
				a.Assert().Equal(input.Filters, []*rds.Filter{
					{
						Name:   aws.String("db-cluster-id"),
						Values: []*string{&a.RDSClusterID},
					},
				})
			}).
			Return(&rds.DescribeDBClustersOutput{
				DBClusters: []*rds.DBCluster{
					{
						DBClusterIdentifier: &a.RDSClusterID,
						Status:              aws.String("available"),
						Endpoint:            aws.String("aws.rds.com/mattermost"),
					},
				},
			}, nil).
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
	a.Assert().Equal("failed to create schema in multitenant RDS cluster rds-cluster-multitenant-09d44077df9934f96-97670d43: "+
		"failed to run create database SQL command: dial tcp: lookup aws.rds.com/mattermost: no such host", err.Error())
}
