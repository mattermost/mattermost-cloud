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
	gt "github.com/aws/aws-sdk-go-v2/service/resourcegroupstaggingapi"
	gtTypes "github.com/aws/aws-sdk-go-v2/service/resourcegroupstaggingapi/types"
	"github.com/aws/aws-sdk-go-v2/service/secretsmanager"
	"github.com/golang/mock/gomock"
	"github.com/mattermost/mattermost-cloud/internal/testlib"
	"github.com/mattermost/mattermost-cloud/model"
	log "github.com/sirupsen/logrus"
)

// Tests provisioning a multitenante database. Use this test for deriving other tests.
// If tests are broken, this should be the first test to get fixed. This test assumes
// data provisioner is running in multitenant database for the first time, so pretty much
// the entire code will run here.
func (a *AWSTestSuite) TestProvisioningMultitenantDatabase() {
	database := NewRDSMultitenantDatabase(
		model.DatabaseEngineTypeMySQL,
		a.InstanceID,
		a.InstallationA.ID,
		a.Mocks.AWS,
		0,
		false,
	)

	databaseType := database.DatabaseEngineTypeTagValue()

	var databaseID string

	gomock.InOrder(
		a.Mocks.Log.Logger.EXPECT().
			WithFields(log.Fields{
				"multitenant-rds-database": MattermostRDSDatabaseName(a.InstallationA.ID),
				"database-type":            database.databaseType,
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

		// Get multitenant databases from the datastore to check if any belongs to the installation ID.
		a.Mocks.Model.DatabaseInstallationStore.EXPECT().
			GetMultitenantDatabases(gomock.Any()).
			Do(func(input *model.MultitenantDatabaseFilter) {
				a.Assert().Equal(input.InstallationID, a.InstallationA.ID)
				a.Assert().Equal(model.NoInstallationsLimit, input.MaxInstallationsLimit)
				a.Assert().Equal(input.PerPage, model.AllPerPage)
			}).
			Return(make([]*model.MultitenantDatabase, 0), nil),

		// Get multitenant databases from the datastore to check if any belongs to the installation ID.
		a.Mocks.Model.DatabaseInstallationStore.EXPECT().
			GetMultitenantDatabases(gomock.Any()).
			Do(func(input *model.MultitenantDatabaseFilter) {
				a.Assert().Equal(DefaultRDSMultitenantDatabaseMySQLCountLimit, input.MaxInstallationsLimit)
				a.Assert().Equal(input.PerPage, model.AllPerPage)
			}).
			Return(make([]*model.MultitenantDatabase, 0), nil),

		// Get resources from AWS and try to find a RDS cluster that the database can be created.
		a.Mocks.API.ResourceGroupsTagging.EXPECT().
			GetResources(gomock.Any(), gomock.Any()).
			Do(func(ctx context.Context, input *gt.GetResourcesInput, optFns ...func(*gt.Options)) {
				a.Assert().Equal(input.ResourceTypeFilters, []string{DefaultResourceTypeClusterRDS})
				tagFilter := []gtTypes.TagFilter{
					{
						Key:    aws.String("DatabaseType"),
						Values: []string{"multitenant-rds"},
					},
					{
						Key:    aws.String(trimTagPrefix(CloudInstallationDatabaseTagKey)),
						Values: []string{databaseType},
					},
					{
						Key:    aws.String("VpcID"),
						Values: []string{a.VPCa},
					},
					{
						Key:    aws.String("Purpose"),
						Values: []string{"provisioning"},
					},
					{
						Key:    aws.String("Owner"),
						Values: []string{"cloud-team"},
					},
					{
						Key:    aws.String("Terraform"),
						Values: []string{"true"},
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
				ResourceTagMappingList: []gtTypes.ResourceTagMapping{
					{
						ResourceARN: aws.String(a.RDSResourceARN),

						// WARNING: If you ever find the need to change some of the hardcoded values such as
						// Owner, Terraform or any of the keys here, make sure that an E2e system's test still
						// passes.
						Tags: []gtTypes.Tag{
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
			DescribeDBClusterEndpoints(gomock.Any(), gomock.Any()).
			Return(&rds.DescribeDBClusterEndpointsOutput{
				DBClusterEndpoints: []rdsTypes.DBClusterEndpoint{
					{
						Status: aws.String("available"),
					},
				},
			}, nil).
			Times(1),

		a.Mocks.API.RDS.EXPECT().
			DescribeDBClusters(gomock.Any(), gomock.Any()).
			Do(func(ctx context.Context, input *rds.DescribeDBClustersInput, optFns ...func(*rds.Options)) {
				a.Assert().Equal(input.Filters, []rdsTypes.Filter{
					{
						Name:   aws.String("db-cluster-id"),
						Values: []string{a.RDSClusterID},
					},
				})
			}).
			Return(&rds.DescribeDBClustersOutput{
				DBClusters: []rdsTypes.DBCluster{
					{
						DBClusterIdentifier: &a.RDSClusterID,
						Endpoint:            aws.String("writer.rds.aws.com"),
						ReaderEndpoint:      aws.String("reader.rds.aws.com"),
						Status:              aws.String("available"),
					},
				},
			}, nil).
			Times(1),

		// Create the multitenant database.
		a.Mocks.Model.DatabaseInstallationStore.EXPECT().
			CreateMultitenantDatabase(gomock.Any()).
			Do(func(input *model.MultitenantDatabase) {
				a.Assert().Equal(input.RdsClusterID, a.RDSClusterID)
				databaseID = model.NewID()
			}).
			Return(nil).
			Times(1),

		a.Mocks.Model.DatabaseInstallationStore.EXPECT().
			LockMultitenantDatabase(databaseID, a.InstanceID).
			Return(true, nil).
			Times(1),

		a.Mocks.Model.DatabaseInstallationStore.EXPECT().
			GetMultitenantDatabase(databaseID).
			Return(&model.MultitenantDatabase{
				ID:           databaseID,
				RdsClusterID: a.RDSClusterID,
			}, nil).
			Times(1),

		a.Mocks.Model.DatabaseInstallationStore.EXPECT().
			UpdateMultitenantDatabase(&model.MultitenantDatabase{
				ID:           databaseID,
				RdsClusterID: a.RDSClusterID,
				Installations: model.MultitenantDatabaseInstallations{
					database.installationID,
				}}).
			Times(1),

		a.Mocks.API.RDS.EXPECT().
			DescribeDBClusters(gomock.Any(), gomock.Any()).
			Do(func(ctx context.Context, input *rds.DescribeDBClustersInput, optFns ...func(*rds.Options)) {
				a.Assert().Equal(input.Filters, []rdsTypes.Filter{
					{
						Name:   aws.String("db-cluster-id"),
						Values: []string{a.RDSClusterID},
					},
				})
			}).
			Return(&rds.DescribeDBClustersOutput{
				DBClusters: []rdsTypes.DBCluster{
					{
						DBClusterIdentifier: &a.RDSClusterID,
						Status:              aws.String("available"),
						Endpoint:            aws.String("aws.rds.com/mattermost"),
					},
				},
			}, nil).
			Times(1),

		a.Mocks.API.SecretsManager.EXPECT().
			GetSecretValue(gomock.Any(), gomock.Any()).
			Do(func(ctx context.Context, input *secretsmanager.GetSecretValueInput, opts ...func(*secretsmanager.Options)) {

			}).
			Return(&secretsmanager.GetSecretValueOutput{
				SecretString: aws.String("VR1pJl2KdNyQy0sxvJYdWbDeTQH1Pk71sXLbS4sw"),
			}, nil).
			Times(1),

		a.Mocks.Model.DatabaseInstallationStore.EXPECT().
			UnlockMultitenantDatabase(databaseID, a.InstanceID, true).
			Return(true, nil).
			Times(1),
	)

	err := database.Provision(a.Mocks.Model.DatabaseInstallationStore, a.Mocks.Log.Logger)
	a.Assert().Error(err)
	a.Assert().Equal("failed to run provisioning sql commands: failed to create schema in multitenant RDS cluster "+
		"rds-cluster-multitenant-09d44077df9934f96-97670d43: failed to run create database SQL command: dial tcp: "+
		"lookup aws.rds.com/mattermost: no such host", err.Error())
}
