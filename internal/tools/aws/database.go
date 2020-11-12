// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.
//

package aws

import (
	"fmt"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/aws/aws-sdk-go/service/kms"
	"github.com/aws/aws-sdk-go/service/rds"
	"github.com/aws/aws-sdk-go/service/resourcegroupstaggingapi"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/mattermost/mattermost-cloud/model"
	mmv1alpha1 "github.com/mattermost/mattermost-operator/apis/mattermost/v1alpha1"
)

// RDSDatabase is a database backed by AWS RDS.
type RDSDatabase struct {
	databaseType   string
	installationID string
	client         *Client
}

// NewRDSDatabase returns a new RDSDatabase interface.
func NewRDSDatabase(databaseType, installationID string, client *Client) *RDSDatabase {
	return &RDSDatabase{
		databaseType:   databaseType,
		installationID: installationID,
		client:         client,
	}
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

	_, err := d.client.Service().rds.CreateDBClusterSnapshot(&rds.CreateDBClusterSnapshotInput{
		DBClusterIdentifier:         aws.String(awsID),
		DBClusterSnapshotIdentifier: aws.String(fmt.Sprintf("%s-snapshot-%v", awsID, time.Now().Nanosecond())),
		Tags: []*rds.Tag{
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

// GenerateDatabaseSpecAndSecret creates the k8s database spec and secret for
// accessing the RDS database.
func (d *RDSDatabase) GenerateDatabaseSpecAndSecret(store model.InstallationDatabaseStoreInterface, logger log.FieldLogger) (*mmv1alpha1.Database, *corev1.Secret, error) {
	awsID := CloudID(d.installationID)

	logger = logger.WithFields(log.Fields{
		"db-cluster-name": awsID,
		"database-type":   d.databaseType,
	})

	installationSecret, err := d.client.secretsManagerGetRDSSecret(awsID, logger)
	if err != nil {
		return nil, nil, err
	}

	dbClusters, err := d.client.Service().rds.DescribeDBClusters(&rds.DescribeDBClustersInput{
		DBClusterIdentifier: aws.String(awsID),
	})
	if err != nil {
		return nil, nil, err
	}

	if len(dbClusters.DBClusters) != 1 {
		return nil, nil, fmt.Errorf("expected 1 DB cluster, but got %d", len(dbClusters.DBClusters))
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
				rdsCluster,
			)
		databaseConnectionCheck = fmt.Sprintf("http://%s:3306", *rdsCluster.Endpoint)
	case model.DatabaseEngineTypePostgres:
		databaseConnectionString, databaseReadReplicasString =
			MattermostPostgresConnStrings(
				"mattermost",
				installationSecret.MasterUsername,
				installationSecret.MasterPassword,
				rdsCluster,
			)
	default:
		return nil, nil, errors.Errorf("%s is an invalid database engine type", d.databaseType)
	}

	databaseSecretName := fmt.Sprintf("%s-rds", d.installationID)
	secretStringData := map[string]string{
		"DB_CONNECTION_STRING":              databaseConnectionString,
		"MM_SQLSETTINGS_DATASOURCEREPLICAS": databaseReadReplicasString,
	}
	if len(databaseConnectionCheck) != 0 {
		secretStringData["DB_CONNECTION_CHECK_URL"] = databaseConnectionCheck
	}

	databaseSecret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name: databaseSecretName,
		},
		StringData: secretStringData,
	}

	databaseSpec := &mmv1alpha1.Database{
		Secret: databaseSecretName,
	}

	logger.Debug("AWS multitenant database configuration generated for cluster installation")

	return databaseSpec, databaseSecret, nil
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
		PerPage:        model.AllPerPage,
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
	vpcFilters := []*ec2.Filter{
		{
			Name:   aws.String(VpcClusterIDTagKey),
			Values: []*string{aws.String(clusterID)},
		},
		{
			Name:   aws.String(VpcAvailableTagKey),
			Values: []*string{aws.String(VpcAvailableTagValueFalse)},
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

	var keyMetadata *kms.KeyMetadata
	if len(kmsResourceNames) > 0 {
		enabledKeys, err := d.getEnabledEncryptionKeys(kmsResourceNames)
		if err != nil {
			return errors.Wrapf(err, "failed to get encryption keys for db cluster %s", awsID)
		}

		if len(enabledKeys) != 1 {
			return errors.Errorf("db cluster %s should have exactly one enabled/active encryption key (found %d)", awsID, len(enabledKeys))
		}

		keyMetadata = enabledKeys[0]
	} else {
		keyMetadata, err = d.client.kmsCreateSymmetricKey(KMSKeyDescriptionRDS(awsID), []*kms.Tag{
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

	err = d.client.rdsEnsureDBClusterCreated(awsID, *vpcs[0].VpcId, rdsSecret.MasterUsername, rdsSecret.MasterPassword, *keyMetadata.KeyId, d.databaseType, logger)
	if err != nil {
		return errors.Wrap(err, "failed to ensure DB cluster was created")
	}

	// Create primary
	err = d.client.rdsEnsureDBClusterInstanceCreated(awsID, fmt.Sprintf("%s-master", awsID), dbEngine, dbConfig.PrimaryInstanceType, logger)
	if err != nil {
		return errors.Wrap(err, "failed to ensure DB primary instance was created")
	}

	// Create replicas
	for i := 0; i < dbConfig.ReplicasCount; i++ {
		err = d.client.rdsEnsureDBClusterInstanceCreated(awsID, fmt.Sprintf("%s-replica-%d", awsID, i), dbEngine, dbConfig.ReplicaInstanceType, logger)
		if err != nil {
			return errors.Wrap(err, "failed to ensure DB replica instance was created")
		}
	}

	return nil
}

func (d *RDSDatabase) getKMSResourceNames(awsID string) ([]*string, error) {
	kmsResources, err := d.client.resourceTaggingGetAllResources(resourcegroupstaggingapi.GetResourcesInput{
		TagFilters: []*resourcegroupstaggingapi.TagFilter{
			{
				Key:    aws.String(DefaultRDSEncryptionTagKey),
				Values: []*string{aws.String(awsID)},
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

func (d *RDSDatabase) getEnabledEncryptionKeys(resourceNameList []*string) ([]*kms.KeyMetadata, error) {
	var keys []*kms.KeyMetadata

	for _, name := range resourceNameList {
		keyMetadata, err := d.client.kmsGetSymmetricKey(*name)
		if err != nil {
			return nil, err
		}
		if keyMetadata != nil && *keyMetadata.KeyState == kms.KeyStateEnabled {
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
