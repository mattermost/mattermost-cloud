package aws

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/aws/aws-sdk-go/service/kms"
	"github.com/aws/aws-sdk-go/service/rds"
	"github.com/aws/aws-sdk-go/service/resourcegroupstaggingapi"
	gt "github.com/aws/aws-sdk-go/service/resourcegroupstaggingapi"
	"github.com/aws/aws-sdk-go/service/secretsmanager"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	_ "github.com/go-sql-driver/mysql"
	"github.com/mattermost/mattermost-cloud/model"
	mmv1alpha1 "github.com/mattermost/mattermost-operator/pkg/apis/mattermost/v1alpha1"
)

// RDSMultitenantDatabase is a database backed by AWS RDS that supports multi tenancy.
type RDSMultitenantDatabase struct {
	client         *Client
	db             *sql.DB
	installationID string
}

// NewRDSMultitenantDatabase returns a new RDSDatabase interface.
func NewRDSMultitenantDatabase(installationID string, client *Client) *RDSMultitenantDatabase {
	return &RDSMultitenantDatabase{
		client:         client,
		installationID: installationID,
	}
}

// Teardown removes all AWS resources related to a multi-tenant RDS database.
func (d *RDSMultitenantDatabase) Teardown(store model.InstallationDatabaseStoreInterface, keepData bool, logger log.FieldLogger) error {
	return nil
}

// Snapshot creates a snapshot of the multi-tenant multi-tenant RDS database..
func (d *RDSMultitenantDatabase) Snapshot(store model.InstallationDatabaseStoreInterface, logger log.FieldLogger) error {
	return nil
}

// GenerateDatabaseSpecAndSecret creates the k8s database spec and secret for
// accessing the multi-tenant RDS database cluster.
func (d *RDSMultitenantDatabase) GenerateDatabaseSpecAndSecret(store model.InstallationDatabaseStoreInterface, logger log.FieldLogger) (*mmv1alpha1.Database, *corev1.Secret, error) {
	vpcID, err := d.getVPCID(store, logger)
	if err != nil {
		return nil, nil, err
	}

	rdsDBCluster, err := d.findAvailableDBCluster(vpcID)
	if err != nil {
		return nil, nil, err
	}

	if rdsDBCluster == nil {
		return nil, nil, errors.New("RDS DB Cluster not found")
	}

	databaseSecretName := RDSMultitenantSecretName(d.installationID)

	result, err := d.client.Service().secretsManager.GetSecretValue(&secretsmanager.GetSecretValueInput{
		SecretId: aws.String(databaseSecretName),
	})
	if err != nil {
		return nil, nil, err
	}

	installationSecret, err := unmarshalSecretPayload(*result.SecretString)
	if err != nil {
		return nil, nil, err
	}

	databaseName := MattermostRDSDatabaseName(d.installationID)

	databaseSecret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name: databaseSecretName,
		},
		StringData: map[string]string{
			"DB_CONNECTION_STRING": MattermostMySQLConnString(databaseName, *rdsDBCluster.Endpoint, installationSecret.MasterUsername, installationSecret.MasterPassword),
		},
	}

	databaseSpec := &mmv1alpha1.Database{
		Secret: databaseSecretName,
	}

	logger.Debugf("RDS DB cluster %s configured to use database %s", *rdsDBCluster.DBClusterIdentifier, databaseName)

	return databaseSpec, databaseSecret, nil
}

func (d *RDSMultitenantDatabase) getVPCID(store model.InstallationDatabaseStoreInterface, logger log.FieldLogger) (string, error) {
	clusterInstallations, err := store.GetClusterInstallations(&model.ClusterInstallationFilter{
		PerPage:        model.AllPerPage,
		InstallationID: d.installationID,
	})
	if err != nil {
		return "", errors.Wrapf(err, "unable to lookup cluster installations for installation %s", d.installationID)
	}

	clusterInstallationCount := len(clusterInstallations)
	if clusterInstallationCount == 0 {
		return "", fmt.Errorf("no cluster installations found for %s", d.installationID)
	}
	if clusterInstallationCount != 1 {
		return "", fmt.Errorf("RDS provisioning is not currently supported for multiple cluster installations (found %d)", clusterInstallationCount)
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
		return "", err
	}
	if len(vpcs) != 1 {
		return "", fmt.Errorf("expected 1 VPC for cluster %s (found %d)", clusterID, len(vpcs))
	}

	return *vpcs[0].VpcId, nil
}

// Provision completes all the steps necessary to provision a multi-tenant RDS database.
func (d *RDSMultitenantDatabase) Provision(store model.InstallationDatabaseStoreInterface, logger log.FieldLogger) error {
	vpcID, err := d.getVPCID(store, logger)
	if err != nil {
		return err
	}

	rdsDBCluster, err := d.findAvailableDBCluster(vpcID)
	if err != nil {
		return err
	}

	result, err := d.client.Service().secretsManager.GetSecretValue(&secretsmanager.GetSecretValueInput{
		SecretId: aws.String(RDSMultitenantClusterSecretName(vpcID)),
	})
	if err != nil && !IsErrorCode(err, secretsmanager.ErrCodeResourceNotFoundException) {
		return err
	}

	masterSecret, err := unmarshalSecretPayload(*result.SecretString)
	if err != nil {
		return err
	}

	err = d.connectDBCluster(rdsMySQLSchemaInformationDatabase, *rdsDBCluster.Endpoint, masterSecret.MasterUsername, masterSecret.MasterPassword)
	defer d.closeDBConnection(logger)
	if err != nil {
		return err
	}

	ctx, cancel := context.WithDeadline(context.Background(), time.Now().Add(MySQLDefaultContextTimeout*time.Second))
	defer cancel()

	err = d.createDatabaseIfNotExist(ctx, MattermostRDSDatabaseName(d.installationID))
	if err != nil {
		return err
	}

	result, err = d.client.Service().secretsManager.GetSecretValue(&secretsmanager.GetSecretValueInput{
		SecretId: aws.String(RDSMultitenantSecretName(d.installationID)),
	})
	if err != nil && !IsErrorCode(err, secretsmanager.ErrCodeResourceNotFoundException) {
		return err
	}

	var installationSecret *RDSSecret
	if result != nil && result.SecretString != nil {
		installationSecret, err = unmarshalSecretPayload(*result.SecretString)
		if err != nil {
			return err
		}
	} else {
		secreteName := RDSMultitenantSecretName(d.installationID)
		description := fmt.Sprintf("Multitenant database for RDS cluster %s", *rdsDBCluster.DBClusterIdentifier)
		installationSecret, err = d.createSecret(secreteName, d.installationID, description, nil, logger)
		if err != nil {
			return err
		}
	}

	err = d.createUserIfNotExist(ctx, installationSecret.MasterUsername, installationSecret.MasterPassword)
	if err != nil {
		return err
	}

	err = d.grantUserFullPermissions(ctx, MattermostRDSDatabaseName(d.installationID), installationSecret.MasterUsername)
	if err != nil {
		return err
	}

	databaseCluster, err := store.GetDatabaseCluster(*rdsDBCluster.DBClusterIdentifier)
	if err != nil {
		return err
	}

	if databaseCluster == nil {
		databaseCluster = &model.DatabaseCluster{
			ID: *rdsDBCluster.DBClusterIdentifier,
		}

		err := store.CreateDatabaseCluster(databaseCluster)
		if err != nil {
			return err
		}
	}

	locked, err := store.LockDatabaseCluster(*rdsDBCluster.DBClusterIdentifier, d.installationID)
	if err != nil {
		return err
	}
	if !locked {
		return errors.Errorf("could not acquire lock for database cluster %s", *rdsDBCluster.DBClusterIdentifier)
	}
	defer store.UnlockDatabaseCluster(*rdsDBCluster.DBClusterIdentifier, d.installationID, true)

	dbInstallations, err := databaseCluster.GetInstallations()
	if err != nil {
		return err
	}

	if !dbInstallations.Contains(d.installationID) {
		dbInstallations.Add(d.installationID)

		err = databaseCluster.SetInstallations(dbInstallations)
		if err != nil {
			return err
		}

		err = store.UpdateDatabaseCluster(databaseCluster)
		if err != nil {
			return err
		}
	}

	err = d.updateRDSClusterCounterTag(rdsDBCluster.DBClusterArn, dbInstallations.Size())
	if err != nil {
		return err
	}

	return nil
}

func (d *RDSMultitenantDatabase) updateRDSClusterCounterTag(resourceARN *string, counter int) error {
	_, err := d.client.Service().rds.AddTagsToResource(&rds.AddTagsToResourceInput{
		ResourceName: resourceARN,
		Tags: []*rds.Tag{
			{
				Key:   aws.String(trimTagPrefix(rdsMultitenantDatabaseCounterTagKey)),
				Value: aws.String(fmt.Sprintf("%d", counter)),
			},
		},
	})
	if err != nil {
		return err
	}

	return nil
}

func (d *RDSMultitenantDatabase) secretsManagerGetRDSSecret(secretName string, logger log.FieldLogger) (*RDSSecret, error) {
	result, err := d.client.Service().secretsManager.GetSecretValue(&secretsmanager.GetSecretValueInput{
		SecretId: aws.String(secretName),
	})
	if err != nil {
		return nil, errors.Wrapf(err, "unable to get secret %s", secretName)
	}

	var rdsSecret *RDSSecret
	err = json.Unmarshal([]byte(*result.SecretString), &rdsSecret)
	if err != nil {
		return nil, errors.Wrap(err, "unable to marshal secrets manager payload")
	}

	err = rdsSecret.Validate()
	if err != nil {
		return nil, err
	}

	return rdsSecret, nil
}

func (d *RDSMultitenantDatabase) createSecret(secreteName, username, description string, tags []*secretsmanager.Tag, logger log.FieldLogger) (*RDSSecret, error) {
	rdsSecretPayload := RDSSecret{
		MasterUsername: username,
		MasterPassword: newRandomPassword(40),
	}
	err := rdsSecretPayload.Validate()
	if err != nil {
		return nil, err
	}

	b, err := json.Marshal(&rdsSecretPayload)
	if err != nil {
		return nil, errors.Wrap(err, "unable to marshal secrets manager payload")
	}

	_, err = d.client.Service().secretsManager.CreateSecret(&secretsmanager.CreateSecretInput{
		Name:         aws.String(secreteName),
		Description:  aws.String(description),
		Tags:         tags,
		SecretString: aws.String(string(b)),
	})
	if err != nil {
		return nil, errors.Wrap(err, "unable to create secrets manager secret")
	}

	logger.WithField("secret-name", secreteName).Debug("Secret created for multitenant RDS DB cluster")

	return &rdsSecretPayload, nil
}

func (d *RDSMultitenantDatabase) findAvailableDBCluster(vpcID string) (*rds.DBCluster, error) {
	resourceNames, err := d.client.resourceTaggingGetAllResources(gt.GetResourcesInput{
		TagFilters: []*gt.TagFilter{
			// {
			// 	Key:    aws.String(trimTagPrefix(rdsMultitenantPurposeTagKey)),
			// 	Values: []*string{aws.String(rdsMultitenantPurposeTagValueProvisioner)},
			// },
			// {
			// 	Key:    aws.String(trimTagPrefix(rdsMultitenantStateTagKey)),
			// 	Values: []*string{aws.String(tagValueTrue)},
			// },
			// {
			// 	Key:    aws.String(trimTagPrefix(rdsMultitenantOwnerTagKey)),
			// 	Values: []*string{aws.String(rdsMultitenantOwnerTagValueCloudTeam)},
			// },
			// {
			// 	Key:    aws.String(trimTagPrefix(rdsMultitenantEnvironmentTagKey)),
			// 	Values: []*string{aws.String(rdsMultitenantPurposeTagValueProvisioner)},
			// },
			{
				Key:    aws.String(trimTagPrefix(rdsMultitenantVPCIDTagKey)),
				Values: []*string{&vpcID},
			},
		},
	})
	if err != nil {
		return nil, err
	}

	for _, resourceName := range resourceNames {
		var dbClusterID *string
		for _, tag := range resourceName.Tags {
			if *tag.Key == trimTagPrefix(rdsMultitenantDBClusterIDTagKey) {
				dbClusterID = tag.Value
			}
		}
		if dbClusterID != nil {
			dbClusterOutput, err := d.client.Service().rds.DescribeDBClusters(&rds.DescribeDBClustersInput{
				Filters: []*rds.Filter{
					{
						Name:   aws.String("db-cluster-id"),
						Values: []*string{dbClusterID},
					},
				},
			})
			if err != nil {
				return nil, err
			}
			if len(dbClusterOutput.DBClusters) != 1 {
				return nil, errors.Errorf("expected exactly one db cluster (found %d)", len(dbClusterOutput.DBClusters))
			}
			if *dbClusterOutput.DBClusters[0].Status != "available" {
				return nil, errors.Errorf("expected db cluster to be available (is %s)", *dbClusterOutput.DBClusters[0].Status)
			}

			return dbClusterOutput.DBClusters[0], nil
		}
	}

	return nil, errors.New("not enough db clusters")
}

func (d *RDSMultitenantDatabase) connectDBCluster(schema, endpoint, username, password string) error {
	db, err := sql.Open("mysql", RDSMySQLConnString(schema, endpoint, username, password))
	if err != nil {
		return errors.Wrap(err, "connecting to RDS DB cluster")
	}
	d.db = db

	return nil
}

func (d *RDSMultitenantDatabase) createUserIfNotExist(ctx context.Context, username, password string) error {
	query := fmt.Sprintf("CREATE USER IF NOT EXISTS '%s'@'%s' IDENTIFIED BY '%s'", username, "%", password)
	_, err := d.db.QueryContext(ctx, query)
	if err != nil {
		return errors.Wrapf(err, "creating user database user %s", username)
	}

	return nil
}

func (d *RDSMultitenantDatabase) createDatabaseIfNotExist(ctx context.Context, databaseName string) error {
	query := fmt.Sprintf("CREATE DATABASE IF NOT EXISTS %s CHARACTER SET 'utf8mb4'", databaseName)
	_, err := d.db.QueryContext(ctx, query)
	if err != nil {
		return errors.Wrapf(err, "creating database name %s", databaseName)
	}

	return nil
}

func (d *RDSMultitenantDatabase) closeDBConnection(logger log.FieldLogger) {
	err := d.db.Close()
	if err != nil {
		logger.WithError(err).Errorf("failed to close database connection with %s", CloudID(d.installationID))
	}
}

func (d *RDSMultitenantDatabase) grantUserFullPermissions(ctx context.Context, databaseName, username string) error {
	query := fmt.Sprintf("GRANT ALL PRIVILEGES ON %s.* TO '%s'@'%s'", databaseName, username, "%")
	_, err := d.db.QueryContext(ctx, query)
	if err != nil {
		return errors.Wrapf(err, "granting permissions to user %s", username)
	}

	return nil
}

func (d *RDSMultitenantDatabase) grantUserReadOnlyPermissions(ctx context.Context, databaseName, username string) error {
	query := fmt.Sprintf("GRANT USAGE ON %s.* TO '%s'@'%s' WITH GRANT OPTION", databaseName, username, "%")
	_, err := d.db.QueryContext(ctx, query)
	if err != nil {
		return errors.Wrapf(err, "granting permissions to user %s", username)
	}

	return nil
}

func (d *RDSMultitenantDatabase) isDatabase(ctx context.Context, databaseName string) (bool, error) {
	query := "SELECT COUNT(*) FROM information_schema.SCHEMATA WHERE SCHEMA_NAME=?"
	rows, err := d.db.QueryContext(ctx, query, databaseName)

	if err != nil {
		return false, err
	}

	count := int(0)
	for rows.Next() {
		err = rows.Scan(&count)
		if err != nil {
			return false, err
		}
	}

	return count > 0, nil
}

func (d *RDSMultitenantDatabase) countDatabases(ctx context.Context) (int, error) {
	query := "SELECT COUNT(*) FROM information_schema.SCHEMATA WHERE SCHEMA_NAME LIKE ?"
	param := fmt.Sprintf("%s%s", rdsDatabaseNamePrefix, "%")

	rows, err := d.db.QueryContext(ctx, query, param)
	if err != nil {
		return 0, err
	}

	count := int(0)
	for rows.Next() {
		err = rows.Scan(&count)
		if err != nil {
			return count, err
		}
	}

	return count, nil
}

// CreateDBClusterRDS ...
func (d *RDSMultitenantDatabase) CreateDBClusterRDS(vpcID string, logger log.FieldLogger) error {
	secreteName := RDSMultitenantClusterSecretName(vpcID)
	dbClusterID := RDSMultitenantDBClusterID(time.Now().Nanosecond())
	description := fmt.Sprintf("RDS multitenant database for cluster id %s", vpcID)

	result, err := d.client.Service().secretsManager.GetSecretValue(&secretsmanager.GetSecretValueInput{
		SecretId: aws.String(secreteName),
	})
	if err != nil && !IsErrorCode(err, secretsmanager.ErrCodeResourceNotFoundException) {
		return err
	}

	var secret *RDSSecret
	if result != nil && result.SecretString != nil {
		secret, err = unmarshalSecretPayload(*result.SecretString)
		if err != nil {
			return err
		}
	} else {
		secret, err = d.createSecret(secreteName, "mmcloud", description, nil, logger)
		if err != nil {
			return err
		}
	}

	kmsResourceNames, err := d.getKMSResourceNames(dbClusterID)
	if err != nil {
		return err
	}

	var keyMetadata *kms.KeyMetadata
	if len(kmsResourceNames) > 0 {
		enabledKeys, err := d.getEnabledEncryptionKeys(kmsResourceNames)
		if err != nil {
			return errors.Wrapf(err, "failed to get encryption keys for db cluster %s", dbClusterID)
		}

		if len(enabledKeys) != 1 {
			return errors.Errorf("db cluster %s should have exactly one enabled/active encryption key (found %d)", dbClusterID, len(enabledKeys))
		}

		keyMetadata = enabledKeys[0]
	} else {
		keyMetadata, err = d.client.kmsCreateSymmetricKey(KMSKeyDescriptionRDS(dbClusterID), []*kms.Tag{
			{
				TagKey:   aws.String(DefaultRDSEncryptionTagKey),
				TagValue: aws.String(dbClusterID),
			},
		})
		if err != nil {
			return errors.Wrapf(err, "unable to create an encryption key for db cluster %s", dbClusterID)
		}
	}

	err = d.client.createRDSMultiDatabaseDBCluster(dbClusterID, vpcID, secret.MasterUsername, secret.MasterPassword, *keyMetadata.KeyId, logger)
	if err != nil {
		return err
	}

	err = d.client.rdsEnsureDBClusterInstanceCreated(dbClusterID, fmt.Sprintf("%s-master", dbClusterID), logger)
	if err != nil {
		return err
	}

	return nil
}

func (d *RDSMultitenantDatabase) getKMSResourceNames(dbClusterID string) ([]*string, error) {
	kmsResources, err := d.client.resourceTaggingGetAllResources(resourcegroupstaggingapi.GetResourcesInput{
		TagFilters: []*resourcegroupstaggingapi.TagFilter{
			{
				Key:    aws.String(DefaultRDSEncryptionTagKey),
				Values: []*string{aws.String(dbClusterID)},
			},
		},
	})
	if err != nil {
		return nil, errors.Wrapf(err, "failed to get KMS resources with tag %s:%s", DefaultRDSEncryptionTagKey, dbClusterID)
	}

	resourceNameList := make([]*string, len(kmsResources))
	for i, resource := range kmsResources {
		resourceNameList[i] = resource.ResourceARN
	}

	return resourceNameList, nil
}

func (d *RDSMultitenantDatabase) getEnabledEncryptionKeys(resourceNameList []*string) ([]*kms.KeyMetadata, error) {
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
