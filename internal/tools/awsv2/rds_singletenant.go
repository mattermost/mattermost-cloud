// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.
//

package awsv2

import (
	"context"
	"fmt"

	"emperror.dev/errors"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/kms"
	"github.com/mattermost/mattermost-cloud/model"
	log "github.com/sirupsen/logrus"
	corev1 "k8s.io/api/core/v1"
)

type RDSSingleTenantDatabase struct {
	databaseType   model.DatabaseType
	installationID string
	logger         log.FieldLogger

	client *Client
}

func (db *RDSSingleTenantDatabase) Provision(store model.InstallationDatabaseStoreInterface, logger log.FieldLogger) error {
	ctx := context.TODO()
	awsID := formatCloudResourceID(db.installationID)

	logger = logger.WithFields(log.Fields{
		"db-cluster-name": awsID,
		"database-type":   db.databaseType,
	})
	logger.Info("Provisioning AWS RDS Single Tenant database")

	if store == nil {
		return errors.New("store was not provided, can't provision as need access to cluster and installation information")
	}

	clusterInstallations, err := store.GetClusterInstallations(&model.ClusterInstallationFilter{
		Paging:         model.AllPagesNotDeleted(),
		InstallationID: db.installationID,
	})
	if err != nil {
		return errors.Wrapf(err, "failed to lookup cluster installations for installation %s", db.installationID)
	}

	clusterInstallationCount := len(clusterInstallations)
	if clusterInstallationCount == 0 {
		return fmt.Errorf("no cluster installations found for %s", db.installationID)
	}
	if clusterInstallationCount != 1 {
		return fmt.Errorf("RDS provisioning is not currently supported for multiple cluster installations (found %d)", clusterInstallationCount)
	}

	// TODO: verify that we're only supposed to have one cluster installation?
	clusterID := clusterInstallations[0].ClusterID
	vpcs, err := db.client.getVPCFromTags(ctx, []Tag{NewTag(VpcClusterIDTagKey, clusterID), NewTag(VpcAvailableTagKey, True)})
	if err != nil {
		return err
	}
	if len(vpcs) != 1 {
		return fmt.Errorf("expected 1 VPC for cluster %s (found %d)", clusterID, len(vpcs))
	}

	rdsSecret, err := db.client.secretsManagerEnsureRDSSecretCreated(awsID, logger)
	if err != nil {
		return err
	}

	kmsResourceNames, err := db.getKMSResourceNames(awsID)
	if err != nil {
		return err
	}

	var keyMetadata *kms.KeyMetadata
	if len(kmsResourceNames) > 0 {
		enabledKeys, err := db.getEnabledEncryptionKeys(kmsResourceNames)
		if err != nil {
			return errors.Wrapf(err, "failed to get encryption keys for db cluster %s", awsID)
		}

		if len(enabledKeys) != 1 {
			return errors.Errorf("db cluster %s should have exactly one enabled/active encryption key (found %d)", awsID, len(enabledKeys))
		}

		keyMetadata = enabledKeys[0]
	} else {
		keyMetadata, err = db.client.kmsCreateSymmetricKey(KMSKeyDescriptionRDS(awsID), []*kms.Tag{
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

	dbConfig, err := db.client.store.GetSingleTenantDatabaseConfigForInstallation(db.installationID)
	if err != nil {
		return errors.Wrap(err, "failed to get single tenant database config for installation")
	}
	if dbConfig == nil {
		return fmt.Errorf("single tenant database not found for installation")
	}

	dbEngine, err := dbEngineFromType(db.databaseType)
	if err != nil {
		return errors.Wrapf(err, "failed to convert database type to database engine")
	}

	err = db.client.rdsEnsureDBClusterCreated(awsID, *vpcs[0].VpcId, rdsSecret.MasterUsername, rdsSecret.MasterPassword, *keyMetadata.KeyId, db.databaseType, logger)
	if err != nil {
		return errors.Wrap(err, "failed to ensure DB cluster was created")
	}

	// Create primary
	err = db.client.rdsEnsureDBClusterInstanceCreated(awsID, fmt.Sprintf("%s-master", awsID), dbEngine, dbConfig.PrimaryInstanceType, logger)
	if err != nil {
		return errors.Wrap(err, "failed to ensure DB primary instance was created")
	}

	// Create replicas
	for i := 0; i < dbConfig.ReplicasCount; i++ {
		err = db.client.rdsEnsureDBClusterInstanceCreated(awsID, fmt.Sprintf("%s-replica-%d", awsID, i), dbEngine, dbConfig.ReplicaInstanceType, logger)
		if err != nil {
			return errors.Wrap(err, "failed to ensure DB replica instance was created")
		}
	}

	return nil
}

func (db *RDSSingleTenantDatabase) Teardown(store model.InstallationDatabaseStoreInterface, keepData bool, logger log.FieldLogger) error {
	return nil
}

func (db *RDSSingleTenantDatabase) Snapshot(store model.InstallationDatabaseStoreInterface, logger log.FieldLogger) error {
	return nil
}

func (db *RDSSingleTenantDatabase) GenerateDatabaseSecret(store model.InstallationDatabaseStoreInterface, logger log.FieldLogger) (*corev1.Secret, error) {
	return nil, nil
}

func (db *RDSSingleTenantDatabase) RefreshResourceMetadata(store model.InstallationDatabaseStoreInterface, logger log.FieldLogger) error {
	return nil
}

func (db *RDSSingleTenantDatabase) MigrateOut(store model.InstallationDatabaseStoreInterface, dbMigration *model.InstallationDBMigrationOperation, logger log.FieldLogger) error {
	return nil
}

func (db *RDSSingleTenantDatabase) MigrateTo(store model.InstallationDatabaseStoreInterface, dbMigration *model.InstallationDBMigrationOperation, logger log.FieldLogger) error {
	return nil
}

func (db *RDSSingleTenantDatabase) TeardownMigrated(store model.InstallationDatabaseStoreInterface, migrationOp *model.InstallationDBMigrationOperation, logger log.FieldLogger) error {
	return nil
}

func (db *RDSSingleTenantDatabase) RollbackMigration(store model.InstallationDatabaseStoreInterface, dbMigration *model.InstallationDBMigrationOperation, logger log.FieldLogger) error {
	return nil
}

func NewRDSSingleTenantDatabase(logger log.FieldLogger, client model.AWS, dbType model.DatabaseType, installationID string) *RDSSingleTenantDatabase {
	return &RDSSingleTenantDatabase{
		databaseType:   dbType,
		installationID: installationID,
		logger:         logger,
		client:         client,
	}
}
