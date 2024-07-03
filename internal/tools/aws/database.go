// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.
//

package aws

import (
	"context"
	"fmt"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	ec2Types "github.com/aws/aws-sdk-go-v2/service/ec2/types"
	kmsTypes "github.com/aws/aws-sdk-go-v2/service/kms/types"
	"github.com/aws/aws-sdk-go-v2/service/rds"
	"github.com/aws/aws-sdk-go-v2/service/rds/types"
	"github.com/aws/aws-sdk-go-v2/service/resourcegroupstaggingapi"
	tgTypes "github.com/aws/aws-sdk-go-v2/service/resourcegroupstaggingapi/types"
	"github.com/mattermost/mattermost-cloud/model"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
	corev1 "k8s.io/api/core/v1"
)

// RDSDatabase is a database backed by AWS RDS.
type RDSDatabase struct {
	databaseType   string
	installationID string
	client         *Client
	disableDBCheck bool
}

// NewRDSDatabase returns a new RDSDatabase interface.
func NewRDSDatabase(databaseType, installationID string, client *Client, disableDBCheck bool) *RDSDatabase {
	return &RDSDatabase{
		databaseType:   databaseType,
		installationID: installationID,
		client:         client,
		disableDBCheck: disableDBCheck,
	}
}

// RefreshResourceMetadata ensures various database resource's metadata are correct.
func (d *RDSDatabase) RefreshResourceMetadata(store model.InstallationDatabaseStoreInterface, logger log.FieldLogger) error {
	return nil
}

// Provision completes all the steps necessary to provision a RDS database.
func (d *RDSDatabase) Provision(store model.InstallationDatabaseStoreInterface, logger log.FieldLogger) error {
	d.client.AddSQLStore(store)

	err := d.rdsDatabaseProvision(d.installationID, logger)
	if err != nil {
		return errors.Wrap(err, "unable to provision RDS database")
	}

	return nil
}

// Teardown removes all AWS resources related to a RDS database.
func (d *RDSDatabase) Teardown(store model.InstallationDatabaseStoreInterface, keepData bool, logger log.FieldLogger) error {
	awsID := CloudID(d.installationID)

	logger = logger.WithFields(log.Fields{
		"db-cluster-name": awsID,
		"database-type":   d.databaseType,
	})
	logger.Info("Tearing down RDS DB cluster")

	err := d.client.secretsManagerEnsureRDSSecretDeleted(awsID, logger)
	if err != nil {
		return errors.Wrap(err, "unable to delete RDS secret")
	}

	if keepData {
		logger.Info("AWS RDS DB cluster was left intact due to the keep-data setting of this server")
		return nil
	}

	err = d.client.rdsEnsureDBClusterDeleted(awsID, logger)
	if err != nil {
		return errors.Wrap(err, "unable to delete RDS DB cluster")
	}

	resourceNames, err := d.getKMSResourceNames(awsID)
	if err != nil {
		return errors.Wrapf(err, "unabled to get KMS resources associated with db cluster %s", awsID)
	}

	if len(resourceNames) > 0 {
		enabledKeys, err := d.getEnabledEncryptionKeys(resourceNames)
		if err != nil {
			return errors.Wrapf(err, "unabled to get encryption key associated with db cluster %s", awsID)
		}

		for _, keyMetadata := range enabledKeys {
			err = d.client.kmsScheduleKeyDeletion(*keyMetadata.KeyId, KMSMaxTimeEncryptionKeyDeletion)
			if err != nil {
				return errors.Wrapf(err, "encryption key associated with db cluster %s could not be scheduled for deletion", awsID)
			}
			logger.Infof("Encryption key %s scheduled for deletion in %d days", *keyMetadata.Arn, KMSMaxTimeEncryptionKeyDeletion)
		}
	} else {
		logger.Warn("Could not find any encryption key. It has been already deleted or never created.")
	}

	logger.Debug("AWS RDS database cluster teardown completed")

	return nil
}

// Snapshot creates a snapshot of the RDS database.
func (d *RDSDatabase) Snapshot(store model.InstallationDatabaseStoreInterface, logger log.FieldLogger) error {
	awsID := CloudID(d.installationID)

	logger = logger.WithFields(log.Fields{
		"db-cluster-name": awsID,
		"database-type":   d.databaseType,
	})

	_, err := d.client.Service().rds.CreateDBClusterSnapshot(
		context.TODO(),
		&rds.CreateDBClusterSnapshotInput{
			DBClusterIdentifier:         aws.String(awsID),
			DBClusterSnapshotIdentifier: aws.String(fmt.Sprintf("%s-snapshot-%v", awsID, time.Now().Nanosecond())),
			Tags: []types.Tag{
				{
					Key:   aws.String(DefaultClusterInstallationSnapshotTagKey),
					Value: aws.String(RDSSnapshotTagValue(awsID)),
				},
			},
		})
	if err != nil {
		return errors.Wrap(err, "failed to create a DB cluster snapshot")
	}

	logger.Info("RDS database snapshot in progress")

	return nil
}

// GenerateDatabaseSecret creates the k8s database spec and secret for
// accessing the RDS database.
func (d *RDSDatabase) GenerateDatabaseSecret(store model.InstallationDatabaseStoreInterface, logger log.FieldLogger) (*corev1.Secret, error) {
	awsID := CloudID(d.installationID)

	logger = logger.WithFields(log.Fields{
		"db-cluster-name": awsID,
		"database-type":   d.databaseType,
	})

	installationSecret, err := d.client.secretsManagerGetRDSSecret(RDSSecretName(awsID))
	if err != nil {
		return nil, err
	}

	dbClusters, err := d.client.Service().rds.DescribeDBClusters(
		context.TODO(),
		&rds.DescribeDBClustersInput{
			DBClusterIdentifier: aws.String(awsID),
		})
	if err != nil {
		return nil, err
	}

	if len(dbClusters.DBClusters) != 1 {
		return nil, fmt.Errorf("expected 1 DB cluster, but got %d", len(dbClusters.DBClusters))
	}
	rdsCluster := dbClusters.DBClusters[0]

	var databaseConnectionString, databaseReadReplicasString, databaseConnectionCheck string
	switch d.databaseType {
	case model.DatabaseEngineTypeMySQL:
		databaseConnectionString, databaseReadReplicasString =
			MattermostMySQLConnStrings(
				"mattermost",
				installationSecret.MasterUsername,
				installationSecret.MasterPassword,
				&rdsCluster,
			)
		databaseConnectionCheck = fmt.Sprintf("http://%s:3306", *rdsCluster.Endpoint)
	case model.DatabaseEngineTypePostgres:
		databaseConnectionString, databaseReadReplicasString =
			MattermostPostgresConnStrings(
				"mattermost",
				installationSecret.MasterUsername,
				installationSecret.MasterPassword,
				&rdsCluster,
			)
		databaseConnectionCheck = databaseConnectionString
	default:
		return nil, errors.Errorf("%s is an invalid database engine type", d.databaseType)
	}

	databaseSecretName := fmt.Sprintf("%s-rds", d.installationID)

	secret := InstallationDBSecret{
		InstallationSecretName: databaseSecretName,
		ConnectionString:       databaseConnectionString,
		DBCheckURL:             databaseConnectionCheck,
		ReadReplicasURL:        databaseReadReplicasString,
	}

	logger.Debug("AWS multitenant database configuration generated for cluster installation")

	return secret.ToK8sSecret(d.disableDBCheck), nil
}

// MigrateOut migration is not supported for single tenant RDS.
func (d *RDSDatabase) MigrateOut(store model.InstallationDatabaseStoreInterface, dbMigration *model.InstallationDBMigrationOperation, logger log.FieldLogger) error {
	return errors.New("database migration is not supported for single tenant RDS")
}

// MigrateTo migration is not supported for single tenant RDS.
func (d *RDSDatabase) MigrateTo(store model.InstallationDatabaseStoreInterface, dbMigration *model.InstallationDBMigrationOperation, logger log.FieldLogger) error {
	return errors.New("database migration is not supported for single tenant RDS")
}

// TeardownMigrated tearing down migrated databases is not supported for single tenant RDS.
func (d *RDSDatabase) TeardownMigrated(store model.InstallationDatabaseStoreInterface, migrationOp *model.InstallationDBMigrationOperation, logger log.FieldLogger) error {
	return errors.New("tearing down migrated installations is not supported for single tenant RDS")
}

// RollbackMigration rolling back migration is not supported for single tenant RDS.
func (d *RDSDatabase) RollbackMigration(store model.InstallationDatabaseStoreInterface, dbMigration *model.InstallationDBMigrationOperation, logger log.FieldLogger) error {
	return errors.New("rolling back db migration is not supported for single tenant RDS")
}

func (d *RDSDatabase) rdsDatabaseProvision(installationID string, logger log.FieldLogger) error {
	awsID := CloudID(installationID)

	logger = logger.WithFields(log.Fields{
		"db-cluster-name": awsID,
		"database-type":   d.databaseType,
	})
	logger.Info("Provisioning AWS RDS database")

	// To properly provision the database we need a SQL client to lookup which
	// cluster(s) the installation is running on.
	if !d.client.HasSQLStore() {
		return errors.New("the provided AWS client does not have SQL store access")
	}

	clusterInstallations, err := d.client.store.GetClusterInstallations(&model.ClusterInstallationFilter{
		Paging:         model.AllPagesNotDeleted(),
		InstallationID: installationID,
	})
	if err != nil {
		return errors.Wrapf(err, "failed to lookup cluster installations for installation %s", installationID)
	}

	clusterInstallationCount := len(clusterInstallations)
	if clusterInstallationCount == 0 {
		return fmt.Errorf("no cluster installations found for %s", installationID)
	}
	if clusterInstallationCount != 1 {
		return fmt.Errorf("RDS provisioning is not currently supported for multiple cluster installations (found %d)", clusterInstallationCount)
	}

	clusterID := clusterInstallations[0].ClusterID
	vpcFilters := []ec2Types.Filter{
		{
			Name:   aws.String(VpcClusterIDTagKey),
			Values: []string{clusterID},
		},
		{
			Name:   aws.String(VpcAvailableTagKey),
			Values: []string{VpcAvailableTagValueFalse},
		},
	}
	vpcs, err := d.client.GetVpcsWithFilters(vpcFilters)
	if err != nil {
		return err
	}
	if len(vpcs) != 1 {
		return fmt.Errorf("expected 1 VPC for cluster %s (found %d)", clusterID, len(vpcs))
	}

	rdsSecret, err := d.client.secretsManagerEnsureRDSSecretCreated(awsID, logger)
	if err != nil {
		return err
	}

	kmsResourceNames, err := d.getKMSResourceNames(awsID)
	if err != nil {
		return err
	}

	var keyMetadata *kmsTypes.KeyMetadata
	if len(kmsResourceNames) > 0 {
		var enabledKeys []*kmsTypes.KeyMetadata
		enabledKeys, err = d.getEnabledEncryptionKeys(kmsResourceNames)
		if err != nil {
			return errors.Wrapf(err, "failed to get encryption keys for db cluster %s", awsID)
		}

		if len(enabledKeys) != 1 {
			return errors.Errorf("db cluster %s should have exactly one enabled/active encryption key (found %d)", awsID, len(enabledKeys))
		}

		keyMetadata = enabledKeys[0]
	} else {
		keyMetadata, err = d.client.kmsCreateSymmetricKey(KMSKeyDescriptionRDS(awsID), []kmsTypes.Tag{
			{
				TagKey:   aws.String(DefaultRDSEncryptionTagKey),
				TagValue: aws.String(awsID),
			},
		})
		if err != nil {
			return errors.Wrapf(err, "failed to create an encryption key for db cluster %s", awsID)
		}
	}

	logger.Infof("Encrypting RDS database with key %s", *keyMetadata.Arn)

	dbConfig, err := d.client.store.GetSingleTenantDatabaseConfigForInstallation(installationID)
	if err != nil {
		return errors.Wrap(err, "failed to get single tenant database config for installation")
	}
	if dbConfig == nil {
		return fmt.Errorf("single tenant database not found for installation")
	}

	dbEngine, err := dbEngineFromType(d.databaseType)
	if err != nil {
		return errors.Wrapf(err, "failed to convert database type to database engine")
	}

	tags, err := NewTags(ClusterIDTagKey, clusterID)
	if err != nil {
		return errors.Wrap(err, "failed to generate AWS Tags")
	}

	err = d.client.rdsEnsureDBClusterCreated(awsID, *vpcs[0].VpcId, rdsSecret.MasterUsername, rdsSecret.MasterPassword, *keyMetadata.KeyId, d.databaseType, tags, logger)
	if err != nil {
		return errors.Wrap(err, "failed to ensure DB cluster was created")
	}

	// Create primary
	err = d.client.rdsEnsureDBClusterInstanceCreated(awsID, fmt.Sprintf("%s-master", awsID), dbEngine, dbConfig.PrimaryInstanceType, tags, logger)
	if err != nil {
		return errors.Wrap(err, "failed to ensure DB primary instance was created")
	}

	// Create replicas
	for i := 0; i < dbConfig.ReplicasCount; i++ {
		err = d.client.rdsEnsureDBClusterInstanceCreated(awsID, fmt.Sprintf("%s-replica-%d", awsID, i), dbEngine, dbConfig.ReplicaInstanceType, tags, logger)
		if err != nil {
			return errors.Wrap(err, "failed to ensure DB replica instance was created")
		}
	}

	return nil
}

func (d *RDSDatabase) getKMSResourceNames(awsID string) ([]*string, error) {
	kmsResources, err := d.client.resourceTaggingGetAllResources(resourcegroupstaggingapi.GetResourcesInput{
		TagFilters: []tgTypes.TagFilter{
			{
				Key:    aws.String(DefaultRDSEncryptionTagKey),
				Values: []string{awsID},
			},
		},
	})
	if err != nil {
		return nil, errors.Wrapf(err, "failed to get KMS resources with tag %s:%s", DefaultRDSEncryptionTagKey, awsID)
	}

	resourceNameList := make([]*string, len(kmsResources))
	for i, resource := range kmsResources {
		resourceNameList[i] = resource.ResourceARN
	}

	return resourceNameList, nil
}

func (d *RDSDatabase) getEnabledEncryptionKeys(resourceNameList []*string) ([]*kmsTypes.KeyMetadata, error) {
	var keys []*kmsTypes.KeyMetadata

	for _, name := range resourceNameList {
		keyMetadata, err := d.client.kmsGetSymmetricKey(*name)
		if err != nil {
			return nil, err
		}
		if keyMetadata != nil && keyMetadata.KeyState == kmsTypes.KeyStateEnabled {
			keys = append(keys, keyMetadata)
		}
	}

	return keys, nil
}

func dbEngineFromType(dbType string) (string, error) {
	switch dbType {
	case model.DatabaseEngineTypeMySQL:
		return "aurora-mysql", nil
	case model.DatabaseEngineTypePostgres:
		return "aurora-postgresql", nil
	default:
		return "", errors.Errorf("%s is an invalid database engine type", dbType)
	}
}
