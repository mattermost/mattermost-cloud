package aws

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"strconv"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/aws/aws-sdk-go/service/rds"
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
	logger.Info("Tearing down multitenant RDS database cluster")
	logger = logger.WithField("rds-multitenant-database", d.installationID)

	// TODO(gsagula): We currently keep the database intact and delete the local data store record. In the future we
	// may prefer to snapshot database to S3 and then also delete the schema from the RDS cluster.

	databaseClusters, err := store.GetDatabaseClusters()
	if err != nil {
		return errors.Wrapf(err, "could not get installation id %", d.installationID)
	}
	if len(databaseClusters) < 1 {
		return errors.Errorf("could not find installation id %", d.installationID)
	}

	for _, databaseCluster := range databaseClusters {
		dbInstallations, err := databaseCluster.GetInstallations()
		if err != nil {
			return errors.Wrapf(err, "could not get installation id %", d.installationID)
		}

		if dbInstallations.Contains(d.installationID) {
			unlocked, err := d.lockDatabaseStore(databaseCluster.ID, store, logger)
			defer unlocked()
			if err != nil {
				return errors.Wrapf(err, "could not lock database for tearing down installation id %", d.installationID)
			}

			if !dbInstallations.Remove(d.installationID) {
				return errors.Errorf("could not remove installation id %s from database cluster installations", d.installationID)
			}

			err = databaseCluster.SetInstallations(dbInstallations)
			if err != nil {
				return errors.Wrapf(err, "could not set installations in database cluster %s", databaseCluster.ID)
			}

			err = store.UpdateDatabaseCluster(databaseCluster)
			if err != nil {
				return errors.Wrapf(err, "could not set update database cluster %s", databaseCluster.ID)
			}

			rdsCluster, err := d.getRDSCluster(databaseCluster.ID)
			if err != nil {
				return errors.Wrap(err, "could not describe RDS cluster")
			}

			err = d.updateRDSClusterCounterTag(rdsCluster.DBClusterArn, dbInstallations.Size())
			if err != nil {
				return errors.Wrapf(err, "failed to update counter tag in RDS cluster id %s", databaseCluster.ID)
			}

			return nil
		}
	}

	databaseSecretName := RDSMultitenantSecretName(d.installationID)

	_, err = d.client.Service().secretsManager.DeleteSecret(&secretsmanager.DeleteSecretInput{
		SecretId: aws.String(databaseSecretName),
	})
	if err != nil {
		return errors.Wrapf(err, "failed to delete RDS database secret name %s", databaseSecretName)
	}

	logger.Info("RDS database installation teardown complete")

	return nil
}

// Snapshot creates a snapshot of the multi-tenant multi-tenant RDS database..
func (d *RDSMultitenantDatabase) Snapshot(store model.InstallationDatabaseStoreInterface, logger log.FieldLogger) error {
	return errors.New("not implemented")
}

// GenerateDatabaseSpecAndSecret creates the k8s database spec and secret for
// accessing the multi-tenant RDS database cluster.
func (d *RDSMultitenantDatabase) GenerateDatabaseSpecAndSecret(store model.InstallationDatabaseStoreInterface, logger log.FieldLogger) (*mmv1alpha1.Database, *corev1.Secret, error) {
	logger.Info("Setting up database spec and secret")
	logger = logger.WithField("rds-multitenant-database", d.installationID)

	databaseClusters, err := store.GetDatabaseClusters()
	if err != nil {
		return nil, nil, errors.Wrapf(err, "could not get installation id %", d.installationID)
	}
	if len(databaseClusters) < 1 {
		return nil, nil, errors.Errorf("could not find installation id %", d.installationID)
	}

	var rdsCluster *rds.DBCluster
	for _, databaseCluster := range databaseClusters {
		dbInstallations, err := databaseCluster.GetInstallations()
		if err != nil {
			return nil, nil, errors.Wrapf(err, "could not get installation id %", d.installationID)
		}

		if dbInstallations.Contains(d.installationID) {
			unlocked, err := d.lockDatabaseStore(databaseCluster.ID, store, logger)
			defer unlocked()
			if err != nil {
				return nil, nil, errors.Wrapf(err, "could not lock database installation id %", d.installationID)
			}

			rdsCluster, err = d.getRDSCluster(databaseCluster.ID)
			if err != nil {
				return nil, nil, errors.Wrap(err, "could not describe RDS cluster")
			}

			break
		}
	}

	if rdsCluster == nil {
		return nil, nil, errors.Errorf("could not find an RDS endpoint for installation id %s", d.installationID)
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
	databaseConnString := MattermostMySQLConnString(databaseName,
		*rdsCluster.Endpoint, installationSecret.MasterUsername, installationSecret.MasterPassword)

	databaseSecret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name: databaseSecretName,
		},
		StringData: map[string]string{
			"DB_CONNECTION_STRING": databaseConnString,
		},
	}

	databaseSpec := &mmv1alpha1.Database{
		Secret: databaseSecretName,
	}

	logger.Infof("Finish to set up spec and secret for database %s", databaseName)
	logger.Debugf("RDS DB cluster %s configured to use connection string %s", *rdsCluster.DBClusterIdentifier, databaseConnString)

	return databaseSpec, databaseSecret, nil
}

// Provision completes all the steps necessary to provision a multi-tenant RDS database.
func (d *RDSMultitenantDatabase) Provision(store model.InstallationDatabaseStoreInterface, logger log.FieldLogger) error {
	databaseName := MattermostRDSDatabaseName(d.installationID)
	logger = logger.WithField("multitenant-rds-database", databaseName)

	vpcID, err := d.getVPCID(store, logger)
	if err != nil {
		return errors.Wrapf(err, "unable to get VPC ID when provisioning database %s", databaseName)
	}

	rdsDBCluster, err := d.findAvailableDBCluster(vpcID, logger)
	if err != nil {
		return errors.Wrapf(err, "unable to find an available RDS cluster to provision database %s", databaseName)
	}

	result, err := d.client.Service().secretsManager.GetSecretValue(&secretsmanager.GetSecretValueInput{
		SecretId: aws.String(RDSMultitenantClusterSecretName(vpcID)),
	})
	if err != nil && !IsErrorCode(err, secretsmanager.ErrCodeResourceNotFoundException) {
		return errors.Wrapf(err, "unable to get a RDS cluster master secret in order to provision database %s", databaseName)
	}

	masterSecret, err := unmarshalSecretPayload(*result.SecretString)
	if err != nil {
		return errors.Wrapf(err, "failed to unmarshal master secret payload in order to provision database %s", databaseName)
	}

	close, err := d.connectDBCluster(rdsMySQLSchemaInformationDatabase, *rdsDBCluster.Endpoint, masterSecret.MasterUsername, masterSecret.MasterPassword, logger)
	defer close()
	if err != nil {
		return errors.Wrapf(err, "unable to connect to the RDS cluster in order to provision database %s", databaseName)
	}

	ctx, cancel := context.WithDeadline(context.Background(), time.Now().Add(DefaultMySQLDefaultContextTimeout*time.Second))
	defer cancel()

	err = d.createDatabaseIfNotExist(ctx, databaseName)
	if err != nil {
		return errors.Wrapf(err, "unable to create schema in the RDS cluster  in order to provision database %s", databaseName)
	}

	result, err = d.client.Service().secretsManager.GetSecretValue(&secretsmanager.GetSecretValueInput{
		SecretId: aws.String(RDSMultitenantSecretName(d.installationID)),
	})
	if err != nil && !IsErrorCode(err, secretsmanager.ErrCodeResourceNotFoundException) {
		return errors.Wrapf(err, "failed to get a RDS cluster installation secret in order to provision database %s", databaseName)
	}

	var installationSecret *RDSSecret
	if result != nil && result.SecretString != nil {
		installationSecret, err = unmarshalSecretPayload(*result.SecretString)
		if err != nil {
			return errors.Wrapf(err, "failed to unmarshal secret payload in order to provision database %s", databaseName)
		}
	} else {
		secreteName := RDSMultitenantSecretName(d.installationID)
		description := fmt.Sprintf("Multitenant database for RDS cluster %s", *rdsDBCluster.DBClusterIdentifier)
		installationSecret, err = d.createSecret(secreteName, d.installationID, description, nil, logger)
		if err != nil {
			return errors.Wrapf(err, "failed to create an installation secret in order to provision database %s", databaseName)
		}
	}

	err = d.createUserIfNotExist(ctx, installationSecret.MasterUsername, installationSecret.MasterPassword)
	if err != nil {
		return errors.Wrapf(err, "failed to create a Mattermost installation user for database %s", databaseName)
	}

	err = d.grantUserFullPermissions(ctx, databaseName, installationSecret.MasterUsername)
	if err != nil {
		return errors.Wrapf(err, "failed to grant permissions to Mattermost user for database %s", databaseName)
	}

	databaseCluster, err := store.GetDatabaseCluster(*rdsDBCluster.DBClusterIdentifier)
	if err != nil {
		return errors.Wrapf(err, "failed to get cluster database schema from the store in order to register database %s", databaseName)
	}

	if databaseCluster == nil {
		databaseCluster = &model.DatabaseCluster{
			ID: *rdsDBCluster.DBClusterIdentifier,
		}

		err := store.CreateDatabaseCluster(databaseCluster)
		if err != nil {
			return errors.Wrapf(err, "failed to create cluster database schema strore in order to register database %s", databaseName)
		}
	}

	unlocked, err := d.lockDatabaseStore(databaseCluster.ID, store, logger)
	defer unlocked()
	if err != nil {
		return errors.Wrapf(err, "could not lock cluster database schema in order to register database %s", databaseName)
	}

	dbInstallations, err := databaseCluster.GetInstallations()
	if err != nil {
		return errors.Wrapf(err, "could not get installations from cluster database schema in order to register database %s", databaseName)
	}

	if !dbInstallations.Contains(d.installationID) {
		dbInstallations.Add(d.installationID)

		err = databaseCluster.SetInstallations(dbInstallations)
		if err != nil {
			return errors.Wrapf(err, "could not set installations in cluster database schema in order to register database %s", databaseName)
		}

		err = store.UpdateDatabaseCluster(databaseCluster)
		if err != nil {
			return errors.Wrapf(err, "could not update cluster database schema in order to register database %s", databaseName)
		}
	}

	err = d.updateRDSClusterCounterTag(rdsDBCluster.DBClusterArn, dbInstallations.Size())
	if err != nil {
		return errors.Wrapf(err, "could not update RDS cluster counter tag to register database %s", databaseName)
	}

	logger.Infof("Updated multitenant RDS counter tag to %d", dbInstallations.Size())

	return nil
}

// Helpers

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
		return "", fmt.Errorf("no cluster installations found for installation id %s", d.installationID)
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

func (d *RDSMultitenantDatabase) updateRDSClusterCounterTag(resourceARN *string, counter int) error {
	_, err := d.client.Service().rds.AddTagsToResource(&rds.AddTagsToResourceInput{
		ResourceName: resourceARN,
		Tags: []*rds.Tag{
			{
				Key:   aws.String(trimTagPrefix(DefaultMultitenantDatabaseCounterTagKey)),
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

func (d *RDSMultitenantDatabase) findAvailableDBCluster(vpcID string, logger log.FieldLogger) (*rds.DBCluster, error) {
	resourceNames, err := d.client.resourceTaggingGetAllResources(gt.GetResourcesInput{
		TagFilters: []*gt.TagFilter{
			{
				Key:    aws.String(trimTagPrefix(RDSMultitenantPurposeTagKey)),
				Values: []*string{aws.String(RDSMultitenantPurposeTagValueProvisioning)},
			},
			{
				Key:    aws.String(trimTagPrefix(RDSMultitenantAcceptingInstallationsTagKey)),
				Values: []*string{aws.String(RDSMultitenantAcceptingInstallationsTagValueTrue)},
			},
			{
				Key:    aws.String(trimTagPrefix(RDSMultitenantOwnerTagKey)),
				Values: []*string{aws.String(RDSMultitenantOwnerTagValueCloudTeam)},
			},
			{
				Key:    aws.String(trimTagPrefix(DefaultRDSMultitenantVPCIDTagKey)),
				Values: []*string{&vpcID},
			},
		},
	})
	if err != nil {
		return nil, errors.Wrapf(err, "failed to get available RDS cluster resources for VPC ID %s", vpcID)
	}

	for _, resourceName := range resourceNames {
		var dbClusterID *string
		var databaseCounter *int

		for _, tag := range resourceName.Tags {
			if *tag.Key == trimTagPrefix(DefaultRDSMultitenantDBClusterIDTagKey) && tag.Value != nil {
				dbClusterID = tag.Value
			}

			if *tag.Key == trimTagPrefix(DefaultMultitenantDatabaseCounterTagKey) && tag.Value != nil {
				value, err := strconv.Atoi(*tag.Value)
				if err == nil {
					databaseCounter = &value
				}
			}

			if dbClusterID != nil && databaseCounter != nil && *databaseCounter < DefaultRDSMultitenantDatabaseCountLimit {
				dbCluster, err := d.getRDSCluster(*dbClusterID)
				if err != nil {
					logger.WithError(err).Errorf("failed to get RDS DB cluster")
					continue
				}
				if *dbCluster.Status != "available" {
					logger.WithError(err).Errorf("expected db cluster to be available (is %s)", *dbCluster.Status)
					continue
				}

				return dbCluster, nil
			}
		}
	}

	return nil, errors.Errorf("could not find a RDS cluster available in the vpc ID %s", vpcID)
}

func (d *RDSMultitenantDatabase) connectDBCluster(schema, endpoint, username, password string, logger log.FieldLogger) (func(), error) {
	db, err := sql.Open("mysql", RDSMySQLConnString(schema, endpoint, username, password))
	if err != nil {
		return nil, errors.Wrap(err, "connecting to RDS DB cluster")
	}

	close := func() {
		err = db.Close()
		if err != nil {
			logger.WithError(err).Errorf("failed to close database connection with %s", endpoint)
		}
	}

	d.db = db

	return close, nil
}

func (d *RDSMultitenantDatabase) createUserIfNotExist(ctx context.Context, username, password string) error {
	// TODO(gsagula): replace string format with proper query.
	query := fmt.Sprintf("CREATE USER IF NOT EXISTS '%s'@'%s' IDENTIFIED BY '%s'", username, "%", password)
	_, err := d.db.QueryContext(ctx, query)
	if err != nil {
		return errors.Wrapf(err, "creating user database user %s", username)
	}

	return nil
}

func (d *RDSMultitenantDatabase) createDatabaseIfNotExist(ctx context.Context, databaseName string) error {
	// TODO(gsagula): replace string format with proper query.
	query := fmt.Sprintf("CREATE DATABASE IF NOT EXISTS %s CHARACTER SET 'utf8mb4'", databaseName)
	_, err := d.db.QueryContext(ctx, query)
	if err != nil {
		return errors.Wrapf(err, "creating database name %s", databaseName)
	}

	return nil
}

func (d *RDSMultitenantDatabase) grantUserFullPermissions(ctx context.Context, databaseName, username string) error {
	// TODO(gsagula): replace string format with proper query.
	query := fmt.Sprintf("GRANT ALL PRIVILEGES ON %s.* TO '%s'@'%s'", databaseName, username, "%")
	_, err := d.db.QueryContext(ctx, query)
	if err != nil {
		return errors.Wrapf(err, "granting permissions to user %s", username)
	}

	return nil
}

func (d *RDSMultitenantDatabase) grantUserReadOnlyPermissions(ctx context.Context, databaseName, username string) error {
	// TODO(gsagula): replace string format with proper query.
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

func (d *RDSMultitenantDatabase) lockDatabaseStore(dbClusterID string, store model.InstallationDatabaseStoreInterface, logger log.FieldLogger) (func(), error) {
	locked, err := store.LockDatabaseCluster(dbClusterID, d.installationID)
	if err != nil {
		return nil, errors.Wrapf(err, "could not acquire lock for database cluster %s", dbClusterID)
	}
	if !locked {
		return nil, errors.Errorf("could not acquire lock for database cluster %s", dbClusterID)
	}
	unlockFN := func() {
		unlocked, err := store.UnlockDatabaseCluster(dbClusterID, d.installationID, true)
		if err != nil {
			logger.WithError(err).Errorf("provisioner store failed to release locker for database id %s", dbClusterID)
		}
		if !unlocked {
			logger.Warnf("database id %s and locker id %s is still locked", dbClusterID, d.installationID)
		}
	}

	return unlockFN, nil
}

func (d *RDSMultitenantDatabase) getRDSCluster(dbClusterID string) (*rds.DBCluster, error) {
	dbClusterOutput, err := d.client.Service().rds.DescribeDBClusters(&rds.DescribeDBClustersInput{
		Filters: []*rds.Filter{
			{
				Name:   aws.String("db-cluster-id"),
				Values: []*string{aws.String(dbClusterID)},
			},
		},
	})
	if err != nil {
		return nil, errors.Wrapf(err, "failed to get RDS cluster id %s", dbClusterID)
	}
	if len(dbClusterOutput.DBClusters) != 1 {
		return nil, fmt.Errorf("expected exactly one RDS cluster for installation id %s (found %d)", d.installationID, len(dbClusterOutput.DBClusters))
	}

	return dbClusterOutput.DBClusters[0], nil
}
