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
	_ "github.com/go-sql-driver/mysql"
	"github.com/mattermost/mattermost-cloud/model"
	mmv1alpha1 "github.com/mattermost/mattermost-operator/pkg/apis/mattermost/v1alpha1"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// SQLDatabaseManager is an interface that describes operations to execute a SQL commands and close the
// the connection with a database.
type SQLDatabaseManager interface {
	ExecContext(ctx context.Context, query string, args ...interface{}) (sql.Result, error)
	Close() error
}

// RDSMultitenantDatabase is a database backed by AWS RDS that supports multi tenancy.
type RDSMultitenantDatabase struct {
	client         *Client
	db             SQLDatabaseManager
	installationID string
}

// NewRDSMultitenantDatabase returns a new RDSDatabase interface.
func NewRDSMultitenantDatabase(installationID string, client *Client) *RDSMultitenantDatabase {
	return &RDSMultitenantDatabase{
		client:         client,
		installationID: installationID,
	}
}

// Teardown removes all AWS resources related to a multitenant RDS database.
func (d *RDSMultitenantDatabase) Teardown(store model.InstallationDatabaseStoreInterface, keepData bool, logger log.FieldLogger) error {
	logger.Info("Tearing down multitenant RDS database cluster")
	logger = logger.WithField("rds-multitenant-database", MattermostRDSDatabaseName(d.installationID))

	// TODO(gsagula): We currently keep the database intact and delete the local data store record. In the future we
	// may prefer to snapshot database to S3 and then also delete the schema from the RDS cluster.

	databaseClusters, err := store.GetDatabaseClusters()
	if err != nil {
		return errors.Wrap(err, "unable to fetch database clusters from data store")
	}
	if len(databaseClusters) < 1 {
		return errors.Errorf("data store has no database clusters")
	}

	for _, databaseCluster := range databaseClusters {
		dbInstallations, err := databaseCluster.GetInstallations()
		if err != nil {
			return errors.Wrapf(err, "could not get installations from database cluster ID %s", databaseCluster.ID)
		}

		if dbInstallations.Contains(d.installationID) {
			unlocked, err := d.acquireLock(databaseCluster.ID, store)
			if err != nil {
				return errors.Wrapf(err, "could not lock database cluster ID %s", databaseCluster.ID)
			}
			defer unlocked(logger)

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

			rdsCluster, err := d.describeRDSCluster(databaseCluster.ID)
			if err != nil {
				return errors.Wrap(err, "could not describe RDS cluster")
			}

			err = d.updateCounterTag(rdsCluster.DBClusterArn, dbInstallations.Size())
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
	if err != nil && !IsErrorCode(err, secretsmanager.ErrCodeResourceNotFoundException) {
		return errors.Wrapf(err, "failed to delete RDS database secret name %s", databaseSecretName)
	}

	logger.Info("RDS database installation teardown complete")

	return nil
}

// Snapshot creates a snapshot of single database.
func (d *RDSMultitenantDatabase) Snapshot(store model.InstallationDatabaseStoreInterface, logger log.FieldLogger) error {
	return errors.New("not implemented")
}

// GenerateDatabaseSpecAndSecret creates the k8s database spec and secret for accessing a single database inside
// a multitenant RDS cluster.
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
			unlocked, err := d.acquireLock(databaseCluster.ID, store)
			if err != nil {
				return nil, nil, errors.Wrapf(err, "could not lock database installation id %", d.installationID)
			}
			defer unlocked(logger)

			rdsCluster, err = d.describeRDSCluster(databaseCluster.ID)
			if err != nil {
				return nil, nil, errors.Wrap(err, "could not describe RDS cluster")
			}

			break
		}
	}

	if rdsCluster == nil {
		return nil, nil, errors.Errorf("could not find an RDS endpoint for installation id %s", d.installationID)
	}

	installationSecreteName := RDSMultitenantSecretName(d.installationID)

	result, err := d.client.Service().secretsManager.GetSecretValue(&secretsmanager.GetSecretValueInput{
		SecretId: &installationSecreteName,
	})
	if err != nil {
		return nil, nil, errors.Wrap(err, "unable to generate databa spec and secret")
	}

	installationSecret, err := unmarshalSecretPayload(*result.SecretString)
	if err != nil {
		return nil, nil, errors.Wrap(err, "unable to generate databa spec and secret")
	}

	installationDBName := MattermostRDSDatabaseName(d.installationID)
	installationDBConnString := MattermostMySQLConnString(installationDBName, *rdsCluster.Endpoint, installationSecret.MasterUsername, installationSecret.MasterPassword)

	databaseSecret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name: installationSecreteName,
		},
		StringData: map[string]string{
			"DB_CONNECTION_STRING": installationDBConnString,
		},
	}

	databaseSpec := &mmv1alpha1.Database{
		Secret: installationSecreteName,
	}

	logger.Infof("Finish to set up spec and secret for database %s", installationDBName)
	logger.Debugf("RDS DB cluster %s configured to use connection string %s", *rdsCluster.DBClusterIdentifier, installationDBConnString)

	return databaseSpec, databaseSecret, nil
}

// Provision claims a multitenant RDS cluster and creates a database schema for the installation.
func (d *RDSMultitenantDatabase) Provision(store model.InstallationDatabaseStoreInterface, logger log.FieldLogger) error {
	databaseName := MattermostRDSDatabaseName(d.installationID)
	logger = logger.WithField("multitenant-rds-database", databaseName)

	vpc, err := d.getClusterInstallationVPC(store)
	if err != nil {
		return errors.Wrapf(err, "unable to find the VPC for installation ID %s", d.installationID)
	}

	lockedCluster, err := d.lockRDSCluster(*vpc.VpcId, store, logger)
	if err != nil {
		return errors.Wrapf(err, "unable to lock a RDS cluster for installation ID %s", d.installationID)
	}
	defer lockedCluster.unlock(logger)

	masterSecretValue, err := d.client.Service().secretsManager.GetSecretValue(&secretsmanager.GetSecretValueInput{
		SecretId: lockedCluster.cluster.DBClusterIdentifier,
	})
	if err != nil && !IsErrorCode(err, secretsmanager.ErrCodeResourceNotFoundException) {
		return errors.Wrapf(err, "unable to find the master secret for RDS cluster ID %s", *lockedCluster.cluster.DBClusterIdentifier)
	}

	close, err := d.connectRDSCluster(rdsMySQLSchemaInformationDatabase, *lockedCluster.cluster.Endpoint, DefaultMattermostDatabaseUsername, *masterSecretValue.SecretString)
	defer close(logger)
	if err != nil {
		return errors.Wrapf(err, "unable to connect to RDS cluster ID %s", *lockedCluster.cluster.DBClusterIdentifier)
	}

	ctx, cancel := context.WithDeadline(context.Background(), time.Now().Add(DefaultMySQLDefaultContextTimeout*time.Second))
	defer cancel()

	err = d.createDatabaseIfNotExist(ctx, databaseName)
	if err != nil {
		return errors.Wrapf(err, "unable to create schema in RDS cluster ID %s", *lockedCluster.cluster.DBClusterIdentifier)
	}

	installationSecreteName := RDSMultitenantSecretName(d.installationID)

	installationSecretValue, err := d.client.Service().secretsManager.GetSecretValue(&secretsmanager.GetSecretValueInput{
		SecretId: aws.String(installationSecreteName),
	})
	if err != nil && !IsErrorCode(err, secretsmanager.ErrCodeResourceNotFoundException) {
		return errors.Wrapf(err, "failed to get RDS database secret for installation ID %s", d.installationID)
	}

	var installationSecret *RDSSecret
	if installationSecretValue != nil && installationSecretValue.SecretString != nil {
		installationSecret, err = unmarshalSecretPayload(*installationSecretValue.SecretString)
		if err != nil {
			return errors.Wrapf(err, "failed to unmarshal RDS database secret for installation ID %s", d.installationID)
		}
	} else {
		description := RDSMultitenantClusterSecretDescription(d.installationID, *lockedCluster.cluster.DBClusterIdentifier)
		tags := []*secretsmanager.Tag{
			{
				Key:   aws.String(trimTagPrefix(DefaultRDSMultitenantDBClusterIDTagKey)),
				Value: lockedCluster.cluster.DBClusterIdentifier,
			},
			{
				Key:   aws.String(trimTagPrefix(DefaultRDSMultitenantVPCIDTagKey)),
				Value: aws.String(*vpc.VpcId),
			},
			{
				Key:   aws.String(trimTagPrefix(DefaultMattermostInstallationIDTagKey)),
				Value: aws.String(d.installationID),
			},
		}
		installationSecret, err = d.createInstallationSecret(installationSecreteName, d.installationID, description, tags)
		if err != nil {
			return errors.Wrapf(err, "failed to create a RDS database secret for installation ID %s", d.installationID)
		}
	}

	err = d.createUserIfNotExist(ctx, installationSecret.MasterUsername, installationSecret.MasterPassword)
	if err != nil {
		return errors.Wrapf(err, "failed to create a Mattermost database schema for installation ID %s", d.installationID)
	}

	err = d.grantUserFullPermissions(ctx, databaseName, installationSecret.MasterUsername)
	if err != nil {
		return errors.Wrapf(err, "failed to grant permissions to Mattermost database user for installation ID %s", d.installationID)
	}

	databaseCluster, err := store.GetDatabaseCluster(*lockedCluster.cluster.DBClusterIdentifier)
	if err != nil {
		return errors.Wrapf(err, "failed to get database cluster for RDS cluster ID %s", *lockedCluster.cluster.DBClusterIdentifier)
	}

	databaseClusterInstallations, err := databaseCluster.GetInstallations()
	if err != nil {
		return errors.Wrapf(err, "failed to get database cluster installations for RDS cluster ID %s", *lockedCluster.cluster.DBClusterIdentifier)
	}

	if !databaseClusterInstallations.Contains(d.installationID) {
		databaseClusterInstallations.Add(d.installationID)

		err = databaseCluster.SetInstallations(databaseClusterInstallations)
		if err != nil {
			return errors.Wrapf(err, "failed to set installations in database cluster ID %s", *lockedCluster.cluster.DBClusterIdentifier)
		}

		err = store.UpdateDatabaseCluster(databaseCluster)
		if err != nil {
			return errors.Wrapf(err, "failed to update database cluster ID %s", *lockedCluster.cluster.DBClusterIdentifier)
		}
	}

	err = d.updateCounterTag(lockedCluster.cluster.DBClusterArn, databaseClusterInstallations.Size())
	if err != nil {
		return errors.Wrapf(err, "failed to set update counter tag for RDS cluster ID %s", *lockedCluster.cluster.DBClusterIdentifier)
	}

	logger.Infof("Updated counter tag multitenant RDS cluster to %d", databaseClusterInstallations.Size())

	return nil
}

// Helpers

type lockRDSClusterOutput struct {
	unlock  func(log.FieldLogger)
	cluster *rds.DBCluster
}

func (d *RDSMultitenantDatabase) lockRDSCluster(vpcID string, store model.InstallationDatabaseStoreInterface, logger log.FieldLogger) (*lockRDSClusterOutput, error) {
	clusters, err := store.GetDatabaseClusters()
	if err != nil {
		return nil, err
	}

	if len(clusters) < 1 {
		resourceNames, err := d.client.resourceTaggingGetAllResources(gt.GetResourcesInput{
			TagFilters: []*gt.TagFilter{
				{
					Key:    aws.String(trimTagPrefix(RDSMultitenantPurposeTagKey)),
					Values: []*string{aws.String(RDSMultitenantPurposeTagValueProvisioning)},
				},
				{
					Key:    aws.String(trimTagPrefix(RDSMultitenantOwnerTagKey)),
					Values: []*string{aws.String(RDSMultitenantOwnerTagValueCloudTeam)},
				},
				{
					Key:    aws.String(trimTagPrefix(DefaultRDSMultitenantVPCIDTagKey)),
					Values: []*string{&vpcID},
				},
				{
					Key: aws.String(trimTagPrefix(RDSMultitenantInstallationCounterTagKey)),
				},
			},
		})
		if err != nil {
			return nil, errors.Wrap(err, "failed to get available RDS cluster resources")
		}

		var rdsClusterID *string
		var installationCounter *string
		for _, resource := range resourceNames {
			for _, tag := range resource.Tags {
				if *tag.Key == trimTagPrefix(RDSMultitenantInstallationCounterTagKey) && tag.Value != nil {
					installationCounter = tag.Value
				}

				if *tag.Key == trimTagPrefix(DefaultRDSMultitenantDBClusterIDTagKey) && tag.Value != nil {
					rdsClusterID = tag.Value
				}

				if rdsClusterID != nil && installationCounter != nil {
					counter, err := strconv.Atoi(*installationCounter)
					if err != nil {
						return nil, err
					}

					if counter < DefaultRDSMultitenantDatabaseCountLimit {
						cluster := model.DatabaseCluster{
							ID: *rdsClusterID,
						}

						err := store.CreateDatabaseCluster(&cluster)
						if err != nil {
							logger.WithError(err).Errorf("failed to create database cluster for installation ID %s", d.installationID)
							continue
						}

						clusters = append(clusters, &cluster)
					}

					break
				}
			}
		}
	}

	for _, cluster := range clusters {
		installations, err := cluster.GetInstallations()
		if err != nil {
			return nil, errors.Wrapf(err, "failed to get installations for database cluster ID %s", cluster.ID)
		}

		if installations.Size() < DefaultRDSMultitenantDatabaseCountLimit {
			unlockFn, err := d.acquireLock(cluster.ID, store)
			if err != nil {
				logger.WithError(err).Errorf("failed to lock rds cluster %s", cluster.ID)
				continue
			}

			cluster, err = store.GetDatabaseCluster(cluster.ID)
			if err != nil {
				logger.WithError(err).Errorf("failed to get database cluster ID %s", cluster.ID)
				continue
			}

			// Since clusters can be sourced from either the datastore or AWS resources,
			// we make sure that nothing has changed prior to the locking has been acquired.
			err = d.validateDatabaseClusterInstallations(installations, cluster)
			if err != nil {
				unlockFn(logger)
				logger.WithError(err).Error("database cluster validation failed")
				continue
			}

			rdsCluster, err := d.describeRDSCluster(cluster.ID)
			if err != nil {
				unlockFn(logger)
				logger.WithError(err).Errorf("failed to get RDS cluster %s", cluster.ID)
				continue
			}

			if *rdsCluster.Status != "available" {
				unlockFn(logger)
				logger.WithError(err).Errorf("failed to get RDS cluster %s", cluster.ID)
				continue
			}

			return &lockRDSClusterOutput{
				unlock:  unlockFn,
				cluster: rdsCluster,
			}, nil
		}
	}

	return nil, errors.Errorf("could not find a RDS cluster available in VPC ID %s", vpcID)
}

func (d *RDSMultitenantDatabase) getClusterInstallationVPC(store model.InstallationDatabaseStoreInterface) (*ec2.Vpc, error) {
	clusterInstallations, err := store.GetClusterInstallations(&model.ClusterInstallationFilter{
		PerPage:        model.AllPerPage,
		InstallationID: d.installationID,
	})
	if err != nil {
		return nil, errors.Wrapf(err, "unable to lookup cluster installations for installation ID %s", d.installationID)
	}

	clusterInstallationCount := len(clusterInstallations)
	if clusterInstallationCount == 0 {
		return nil, fmt.Errorf("no cluster installations found for installation ID %s", d.installationID)
	}
	if clusterInstallationCount != 1 {
		return nil, fmt.Errorf("RDS provisioning is not currently supported for multiple cluster installations (found %d)", clusterInstallationCount)
	}

	vpcs, err := d.client.GetVpcsWithFilters([]*ec2.Filter{
		{
			Name:   aws.String(VpcClusterIDTagKey),
			Values: []*string{aws.String(clusterInstallations[0].ClusterID)},
		},
		{
			Name:   aws.String(VpcAvailableTagKey),
			Values: []*string{aws.String(VpcAvailableTagValueFalse)},
		},
	})
	if err != nil {
		return nil, errors.Wrapf(err, "unable to lookup VPCs for installation ID %s", d.installationID)
	}
	if len(vpcs) != 1 {
		return nil, fmt.Errorf("expected 1 VPC for RDS cluster %s (found %d)", clusterInstallations[0].ClusterID, len(vpcs))
	}

	return vpcs[0], nil
}

func (d *RDSMultitenantDatabase) updateCounterTag(resourceARN *string, counter int) error {
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
		return errors.Wrap(err, "failed to update RDS cluster counter's tag")
	}

	return nil
}

func (d *RDSMultitenantDatabase) createInstallationSecret(secretName, username, description string, tags []*secretsmanager.Tag) (*RDSSecret, error) {
	rdsSecretPayload := RDSSecret{
		MasterUsername: username,
		MasterPassword: newRandomPassword(40),
	}
	err := rdsSecretPayload.Validate()
	if err != nil {
		return nil, errors.Wrapf(err, "secret %s has error(s)", secretName)
	}

	b, err := json.Marshal(&rdsSecretPayload)
	if err != nil {
		return nil, errors.Wrap(err, "unable to marshal secrets manager payload")
	}

	_, err = d.client.Service().secretsManager.CreateSecret(&secretsmanager.CreateSecretInput{
		Name:         aws.String(secretName),
		Description:  aws.String(description),
		Tags:         tags,
		SecretString: aws.String(string(b)),
	})
	if err != nil {
		return nil, errors.Wrapf(err, "unable to create secret %s", secretName)
	}

	return &rdsSecretPayload, nil
}

func (d *RDSMultitenantDatabase) describeRDSCluster(dbClusterID string) (*rds.DBCluster, error) {
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

func (d *RDSMultitenantDatabase) acquireLock(dbClusterID string, store model.InstallationDatabaseStoreInterface) (func(logger log.FieldLogger), error) {
	locked, err := store.LockDatabaseCluster(dbClusterID, d.installationID)
	if err != nil {
		return nil, errors.Wrapf(err, "could not acquire lock for database cluster %s", dbClusterID)
	}
	if !locked {
		return nil, errors.Errorf("could not acquire lock for database cluster %s", dbClusterID)
	}
	unlockFN := func(logger log.FieldLogger) {
		unlocked, err := store.UnlockDatabaseCluster(dbClusterID, d.installationID, true)
		if err != nil {
			logger.WithError(err).Errorf("provisioner datastore failed to release locker for database cluster ID %s", dbClusterID)
		}
		if !unlocked {
			logger.Warnf("database ID %s and locker ID %s is still locked", dbClusterID, d.installationID)
		}
	}

	return unlockFN, nil
}

func (d *RDSMultitenantDatabase) validateDatabaseClusterInstallations(installations model.DatabaseClusterInstallations, databaseCluster *model.DatabaseCluster) error {
	expectedInstallations, err := databaseCluster.GetInstallations()
	if err != nil {
		return errors.Errorf("failed to get installations from database cluster ID %s", databaseCluster.ID)
	}

	if installations.Size() != expectedInstallations.Size() {
		return errors.Errorf("supplied %s installations, but the cluster ID %s has %s", installations.Size(), expectedInstallations.Size(), databaseCluster.ID)
	}

	for _, installation := range installations {
		if !expectedInstallations.Contains(installation) {
			return errors.Errorf("unable to find installation ID %s in cluster ID %s", installation, databaseCluster.ID)
		}
	}

	return nil
}

func (d *RDSMultitenantDatabase) connectRDSCluster(schema, endpoint, username, password string) (func(logger log.FieldLogger), error) {
	// This condition allows injecting our own driver for testing.
	if d.db == nil {
		db, err := sql.Open("mysql", RDSMySQLConnString(schema, endpoint, username, password))
		if err != nil {
			return nil, errors.Wrap(err, "connecting to RDS DB cluster")
		}

		d.db = db
	}

	close := func(logger log.FieldLogger) {
		err := d.db.Close()
		if err != nil {
			logger.WithError(err).Errorf("failed to close database connection to %s", endpoint)
		}
	}

	return close, nil
}

func (d *RDSMultitenantDatabase) createUserIfNotExist(ctx context.Context, username, password string) error {
	query := fmt.Sprintf("CREATE USER IF NOT EXISTS '%s'@'%s' IDENTIFIED BY '%s'", username, "%", password)

	_, err := d.db.ExecContext(ctx, query)
	if err != nil {
		return errors.Wrapf(err, "creating user database user %s", username)
	}

	return nil
}

func (d *RDSMultitenantDatabase) createDatabaseIfNotExist(ctx context.Context, databaseName string) error {
	query := fmt.Sprintf("CREATE DATABASE IF NOT EXISTS %s CHARACTER SET 'utf8mb4'", databaseName)

	_, err := d.db.ExecContext(ctx, query)
	if err != nil {
		return errors.Wrapf(err, "creating database name %s", databaseName)
	}

	return nil
}

func (d *RDSMultitenantDatabase) grantUserFullPermissions(ctx context.Context, databaseName, username string) error {
	query := fmt.Sprintf("GRANT ALL PRIVILEGES ON %s.* TO '%s'@'%s'", databaseName, username, "%")

	_, err := d.db.ExecContext(ctx, query)
	if err != nil {
		return errors.Wrapf(err, "granting permissions to user %s", username)
	}

	return nil
}
