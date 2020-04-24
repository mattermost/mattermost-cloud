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
func (d *RDSMultitenantDatabase) Teardown(keepData bool, logger log.FieldLogger) error {
	return nil
}

// Snapshot creates a snapshot of the multi-tenant multi-tenant RDS database..
func (d *RDSMultitenantDatabase) Snapshot(logger log.FieldLogger) error {
	return nil
}

// GenerateDatabaseSpecAndSecret creates the k8s database spec and secret for
// accessing the multi-tenant RDS database cluster.
func (d *RDSMultitenantDatabase) GenerateDatabaseSpecAndSecret(logger log.FieldLogger) (*mmv1alpha1.Database, *corev1.Secret, error) {
	awsID := CloudID(d.installationID)
	databaseSecretName := RDSMultitenantSecretName(awsID)

	rdsSecret, err := d.secretsManagerGetRDSSecret(databaseSecretName, logger)
	if err != nil {
		return nil, nil, err
	}

	resourceNames, err := d.client.resourceTaggingGetAllResources(gt.GetResourcesInput{
		// We create secrets with the following tags:
		// {
		// 	Key:   aws.String(trimTagPrefix(rdsMultitenantDBCloudIDTagKey)),
		// 	Value: aws.String(cloudID),
		// },
		// {
		// 	Key:   aws.String(trimTagPrefix(rdsMultitenantDBClusterIDTagKey)),
		// 	Value: aws.String(dbClusterID),
		// },
		TagFilters: []*gt.TagFilter{
			{
				Key:    aws.String(trimTagPrefix(rdsMultitenantDBCloudIDTagKey)),
				Values: []*string{aws.String(awsID)},
			},
			{
				Key: aws.String(trimTagPrefix(rdsMultitenantDBClusterIDTagKey)),
			},
		},
	})
	if err != nil {
		return nil, nil, errors.Wrapf(err, "failed to get RDS secret resource that belong to cloud installation %s", awsID)
	}

	if len(resourceNames) != 1 {
		return nil, nil, errors.Errorf("cloud installation %s should have one RDS secret (found %d)", awsID, len(resourceNames))
	}

	var rdsDBClusterID *string
	for _, tag := range resourceNames[0].Tags {
		if *tag.Key == trimTagPrefix(rdsMultitenantDBClusterIDTagKey) {
			rdsDBClusterID = tag.Value
			break
		}
	}

	if rdsDBClusterID == nil {
		return nil, nil, errors.Errorf("could not found a RDS DB cluster for cloud installation %s", awsID)
	}

	result, err := d.client.Service().rds.DescribeDBClusters(&rds.DescribeDBClustersInput{
		DBClusterIdentifier: rdsDBClusterID,
	})
	if err != nil {
		return nil, nil, err
	}

	var rdsDBClusters []*rds.DBCluster
	for _, cluster := range result.DBClusters {
		if *cluster.Status == "available" && *cluster.DBClusterIdentifier == *rdsDBClusterID {
			rdsDBClusters = append(rdsDBClusters, cluster)
		}
	}

	if len(rdsDBClusters) != 1 {
		return nil, nil, fmt.Errorf("expected one multitenant RDS cluster for installation %s (found %d): %v", awsID, len(result.DBClusters), result.DBClusters)
	}

	connString := RDSMySQLConnString(MattermostRDSDatabaseName(d.installationID), *rdsDBClusters[0].Endpoint, rdsSecret.MasterUsername, rdsSecret.MasterPassword)

	logger.Debugf(">>>>> CONNECTION STRING: %s", connString)

	databaseSecret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name: databaseSecretName,
		},
		StringData: map[string]string{
			"DB_CONNECTION_STRING": connString,
		},
	}

	databaseSpec := &mmv1alpha1.Database{
		Secret: databaseSecretName,
	}

	logger.Debug("Cluster installation configured to use an AWS RDS Database")

	return databaseSpec, databaseSecret, nil
}

// Provision completes all the steps necessary to provision a multi-tenant RDS database.
func (d *RDSMultitenantDatabase) Provision(store model.InstallationDatabaseStoreInterface, logger log.FieldLogger) error {
	ctx, cancel := context.WithDeadline(context.Background(), time.Now().Add(MySQLDefaultContextTimeout*time.Second))
	defer cancel()

	rdsDBCluster, err := d.findAvailableDBCluster()
	if err != nil {
		return err
	}

	// Missing checking for the endpoint status

	masterSecret, err := d.client.secretsManagerGetRDSSecret("rds-db-cluster-master-secret", logger)
	if err != nil {
		return err
	}

	err = d.connectDBCluster(rdsMySQLSchemaInformationDatabase, *rdsDBCluster.Endpoint, masterSecret.MasterUsername, masterSecret.MasterPassword)
	if err != nil {
		return err
	}

	isDatabase, err := d.isDatabase(ctx, MattermostRDSDatabaseName(d.installationID))
	if err != nil {
		return err
	}

	if !isDatabase {
		err = d.createDatabaseIfNotExist(ctx, MattermostRDSDatabaseName(d.installationID))
		if err != nil {
			return err
		}
	}

	err = d.closeDBConnection()
	if err != nil {
		return err
	}

	err = d.connectDBCluster(MattermostRDSDatabaseName(d.installationID), *rdsDBCluster.Endpoint, masterSecret.MasterUsername, masterSecret.MasterPassword)
	if err != nil {
		return err
	}

	installationSecret, err := d.secretsManagerEnsureRDSSecretCreated(*rdsDBCluster.DBClusterIdentifier, logger)
	if err != nil {
		return err
	}

	err = d.createUserIfNotExist(ctx, installationSecret.MasterUsername, installationSecret.MasterPassword)
	if err != nil {
		return err
	}

	// err = d.grantUserReadOnlyPermissions(ctx, rdsMySQLSchemaInformationDatabase, installationSecret.MasterUsername)
	// if err != nil {
	// 	return err
	// }

	err = d.grantUserFullPermissions(ctx, MattermostRDSDatabaseName(d.installationID), installationSecret.MasterUsername)
	if err != nil {
		return err
	}

	err = d.closeDBConnection()
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
		return nil, errors.Wrap(err, "unable to get secrets manager secret")
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

func (d *RDSMultitenantDatabase) secretsManagerEnsureRDSSecretCreated(dbClusterID string, logger log.FieldLogger) (*RDSSecret, error) {
	cloudID := CloudID(d.installationID)
	rdsSecretName := RDSMultitenantSecretName(cloudID)
	rdsSecretPayload := RDSSecret{}

	// Check if we already have an RDS secret for this installation.
	result, err := d.client.Service().secretsManager.GetSecretValue(&secretsmanager.GetSecretValueInput{
		SecretId: aws.String(rdsSecretName),
	})
	if err == nil {
		if result != nil {
			logger.WithField("secret-name", rdsSecretName).Debug("AWS RDS secret already created")
			err = json.Unmarshal([]byte(*result.SecretString), &rdsSecretPayload)
			if err != nil {
				return nil, errors.Wrap(err, "unable to marshal secrets manager payload")
			}

			err := rdsSecretPayload.Validate()
			if err != nil {
				return nil, err
			}

			return &rdsSecretPayload, nil
		}

		return nil, err
	}

	// There is no existing secret, so we will create a new one with a strong
	// random username and password.
	rdsSecretPayload.MasterUsername = cloudID
	rdsSecretPayload.MasterPassword = newRandomPassword(40)
	err = rdsSecretPayload.Validate()
	if err != nil {
		return nil, err
	}

	b, err := json.Marshal(&rdsSecretPayload)
	if err != nil {
		return nil, errors.Wrap(err, "unable to marshal secrets manager payload")
	}

	_, err = d.client.Service().secretsManager.CreateSecret(&secretsmanager.CreateSecretInput{
		Name:        aws.String(rdsSecretName),
		Description: aws.String(fmt.Sprintf("RDS multitenant configuration for %s database", cloudID)),
		Tags: []*secretsmanager.Tag{
			{
				Key:   aws.String(trimTagPrefix(rdsMultitenantDBCloudIDTagKey)),
				Value: aws.String(cloudID),
			},
			{
				Key:   aws.String(trimTagPrefix(rdsMultitenantDBClusterIDTagKey)),
				Value: aws.String(dbClusterID),
			},
		},
		SecretString: aws.String(string(b)),
	})
	if err != nil {
		return nil, errors.Wrap(err, "unable to create secrets manager secret")
	}

	logger.WithField("secret-name", rdsSecretName).Debug("Secret created for multitenant RDS DB cluster")

	return &rdsSecretPayload, nil
}

func (d *RDSMultitenantDatabase) findAvailableDBCluster() (*rds.DBCluster, error) {
	resourceNames, err := d.client.resourceTaggingGetAllResources(gt.GetResourcesInput{
		TagFilters: []*gt.TagFilter{
			{
				Key:    aws.String(trimTagPrefix(rdsMultitenantDBClusterStatusTagKey)),
				Values: []*string{aws.String(rdsMultitenantDBClusterStatusAvailable)},
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
	connString := fmt.Sprintf("%s:%s@tcp(%s:3306)/%s?charset=utf8mb4,utf8&readTimeout=30s&writeTimeout=30s",
		username, password, endpoint, schema)
	db, err := sql.Open("mysql", connString)
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

func (d *RDSMultitenantDatabase) closeDBConnection() error {
	err := d.db.Close()
	if err != nil {
		return errors.Wrap(err, "closing connection")
	}

	return nil
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

func (d *RDSMultitenantDatabase) createDBClusterRDS(logger log.FieldLogger) error {
	// To properly provision the database we need a SQL client to lookup which
	// cluster(s) the installation is running on.
	if !d.client.HasSQLStore() {
		return errors.New("the provided AWS client does not have SQL store access")
	}

	clusterInstallations, err := d.client.store.GetClusterInstallations(&model.ClusterInstallationFilter{
		PerPage:        model.AllPerPage,
		InstallationID: d.installationID,
	})
	if err != nil {
		return errors.Wrapf(err, "unable to lookup cluster installations for installation %s", d.installationID)
	}

	clusterInstallationCount := len(clusterInstallations)
	if clusterInstallationCount == 0 {
		return fmt.Errorf("no cluster installations found for %s", d.installationID)
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

	// OK, we go the VPC which the installation belongs too. Let's create a db cluster then.
	dbMasterSecret := "rds-db-cluster-master-secret"
	rdsSecret, err := d.client.secretsManagerEnsureRDSSecretCreated(dbMasterSecret, logger)
	if err != nil {
		return err
	}

	dbMasterEncryptionKey := "rds-db-cluster-master-encryption-key"
	kmsResourceNames, err := d.getKMSResourceNames(dbMasterEncryptionKey)
	if err != nil {
		return err
	}

	dbClusterID := fmt.Sprintf("rds-%s-%d", *vpcs[0].VpcId, time.Now().Nanosecond())
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
		keyMetadata, err = d.client.kmsCreateSymmetricKey(KMSKeyDescriptionRDS(dbMasterEncryptionKey), []*kms.Tag{
			{
				TagKey:   aws.String(DefaultRDSEncryptionTagKey),
				TagValue: aws.String(dbClusterID),
			},
		})
		if err != nil {
			return errors.Wrapf(err, "unable to create an encryption key for db cluster %s", dbClusterID)
		}
	}

	logger.Infof("Encrypting RDS database with key %s", *keyMetadata.Arn)

	err = d.client.createRDSMultiDatabaseDBCluster(dbClusterID, *vpcs[0].VpcId, rdsSecret.MasterUsername, rdsSecret.MasterPassword, *keyMetadata.KeyId, logger)
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

// func (d *RDSMultitenantDatabase) getConnectionString(filter []*gt.TagFilter) (string, error) {
// 	resourceNames, err := d.client.resourceTaggingGetAllResources(gt.GetResourcesInput{
// 		TagFilters: filter,
// 	})
// 	if err != nil {
// 		return "", err
// 	}

// 	if len(resourceNames) < 1 {
// 		return "", nil
// 	}

// 	for _, resource := range resourceNames {
// 		secret, err := d.client.Service().secretsManager.DescribeSecret(&secretsmanager.DescribeSecretInput{
// 			SecretId: resource.ResourceARN,
// 		})
// 		if err != nil {
// 			return "", err
// 		}
// 		if secret.DeletedDate == nil {
// 			result, err := d.client.Service().secretsManager.GetSecretValue(&secretsmanager.GetSecretValueInput{
// 				SecretId: secret.ARN,
// 			})
// 			if err != nil {
// 				return "", err
// 			}

// 			var rdsSecret *RDSSecret
// 			err = json.Unmarshal([]byte(*result.SecretString), &rdsSecret)
// 			if err != nil {
// 				return "", errors.Wrap(err, "unable to marshal secrets manager payload")
// 			}

// 			err = rdsSecret.Validate()
// 			if err != nil {
// 				return "", err
// 			}

// 			for _, tag := range secret.Tags {
// 				if *tag.Key == trimTagPrefix(rdsMultitenantDBClusterIDTagKey) {
// 					out, err := d.client.Service().rds.DescribeDBClusters(&rds.DescribeDBClustersInput{
// 						DBClusterIdentifier: tag.Value,
// 					})
// 					if err != nil {
// 						return "", err
// 					}
// 					if len(out.DBClusters) != 1 {
// 						return "", fmt.Errorf("expected 1 DB cluster, but got %d", len(out.DBClusters))
// 					}

// 					return RDSMySQLConnString(rdsMySQLSchemaInformationDatabase, *out.DBClusters[0].Endpoint, *rdsSecret), nil
// 				}
// 			}
// 		}

// 	}

// 	return "", nil
// }
