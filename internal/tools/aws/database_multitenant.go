package aws

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/arn"
	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/aws/aws-sdk-go/service/rds"
	gt "github.com/aws/aws-sdk-go/service/resourcegroupstaggingapi"
	"github.com/aws/aws-sdk-go/service/secretsmanager"

	"github.com/mattermost/mattermost-cloud/model"
	mmv1alpha1 "github.com/mattermost/mattermost-operator/pkg/apis/mattermost/v1alpha1"

	_ "github.com/go-sql-driver/mysql"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// SQLDatabaseManager is an interface that describes operations to execute a SQL commands and close the
// the connection with a database.
type SQLDatabaseManager interface {
	QueryContext(ctx context.Context, query string, args ...interface{}) (*sql.Rows, error)
	Close() error
}

// RDSMultitenantDatabase is a database backed by AWS RDS that supports multi tenancy.
type RDSMultitenantDatabase struct {
	client         *Client
	db             SQLDatabaseManager
	installationID string
}

// NewRDSMultitenantDatabase returns a new instance of RDSMultitenantDatabase that implements database interface.
func NewRDSMultitenantDatabase(installationID string, client *Client) *RDSMultitenantDatabase {
	return &RDSMultitenantDatabase{
		client:         client,
		installationID: installationID,
	}
}

// Teardown removes all AWS resources related to a RDS multitenant database.
func (d *RDSMultitenantDatabase) Teardown(store model.InstallationDatabaseStoreInterface, keepData bool, logger log.FieldLogger) error {
	databaseName := MattermostRDSDatabaseName(d.installationID)

	logger = logger.WithField("rds-multitenant-database", databaseName)

	if keepData {
		logger.Infof("Keep data was requested - leaving RDS database for installation ID %s intacted", d.installationID)

		return nil
	}

	logger.Info("Tearing down RDS database and database secret")

	multitenantDatabases, err := store.GetMultitenantDatabases(&model.MultitenantDatabaseFilter{
		InstallationID: d.installationID,
	})
	if err != nil {
		return errors.Wrapf(err, "cannot teardown RDS cluster for installation ID %s", d.installationID)
	}
	if len(multitenantDatabases) != 1 {
		return errors.Errorf("expect exactly one RDS cluster for installation ID %s (found %d)", d.installationID, len(multitenantDatabases))
	}

	err = d.removeRDSDatabase(multitenantDatabases[0], databaseName, store, logger)
	if err != nil {
		return errors.Wrapf(err, "cannot teardown RDS cluster for installation ID %s", d.installationID)
	}

	_, err = d.client.Service().secretsManager.DeleteSecret(&secretsmanager.DeleteSecretInput{
		SecretId: aws.String(RDSMultitenantSecretName(d.installationID)),
	})
	if err != nil && !IsErrorCode(err, secretsmanager.ErrCodeResourceNotFoundException) {
		return errors.Wrapf(err, "failed to delete RDS database secret name %s", RDSMultitenantSecretName(d.installationID))
	}

	logger.Infof("RDS multitenant database teardown successfully completed")

	return nil
}

// Snapshot creates a snapshot of single RDS multitenant database.
func (d *RDSMultitenantDatabase) Snapshot(store model.InstallationDatabaseStoreInterface, logger log.FieldLogger) error {
	return errors.New("not implemented")
}

// GenerateDatabaseSpecAndSecret creates the k8s database spec and secret for accessing a single database inside
// a RDS multitenant cluster.
func (d *RDSMultitenantDatabase) GenerateDatabaseSpecAndSecret(store model.InstallationDatabaseStoreInterface, logger log.FieldLogger) (*mmv1alpha1.Database, *corev1.Secret, error) {
	databaseName := MattermostRDSDatabaseName(d.installationID)

	logger = logger.WithField("rds-multitenant-database", databaseName)

	multitenantDatabases, err := store.GetMultitenantDatabases(&model.MultitenantDatabaseFilter{
		InstallationID: d.installationID,
	})
	if err != nil {
		return nil, nil, errors.Wrapf(err, "could not get installation id %", d.installationID)
	}
	if len(multitenantDatabases) != 1 {
		return nil, nil, errors.Errorf("expect exactly one RDS cluster for installation ID %s (found %d)", d.installationID, len(multitenantDatabases))
	}

	unlocked, err := d.acquireLock(multitenantDatabases[0].ID, store)
	if err != nil {
		return nil, nil, errors.Wrapf(err, "failed to lock RDS database cluster %s", multitenantDatabases[0].ID)
	}
	defer unlocked(logger)

	rdsCluster, err := d.describeRDSCluster(multitenantDatabases[0].ID)
	if err != nil {
		return nil, nil, errors.Wrap(err, "failed get database cluster information from AWS RDS service")
	}

	installationSecretName := RDSMultitenantSecretName(d.installationID)

	result, err := d.client.Service().secretsManager.GetSecretValue(&secretsmanager.GetSecretValueInput{
		SecretId: &installationSecretName,
	})
	if err != nil {
		return nil, nil, errors.Wrap(err, "failed to get database spec and secret from AWS Secrets Manager Service")
	}

	installationSecret, err := unmarshalSecretPayload(*result.SecretString)
	if err != nil {
		return nil, nil, errors.Wrap(err, "failed to generate database spec and secret")
	}

	installationDatabaseName := MattermostRDSDatabaseName(d.installationID)
	installationDatabaseConn := MattermostMySQLConnString(installationDatabaseName, *rdsCluster.Endpoint, installationSecret.MasterUsername, installationSecret.MasterPassword)

	databaseSecret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name: installationSecretName,
		},
		StringData: map[string]string{
			"DB_CONNECTION_STRING": installationDatabaseConn,
		},
	}

	databaseSpec := &mmv1alpha1.Database{
		Secret: installationSecretName,
	}

	logger.Infof("Finished to set up spec and secret for RDS multitenant database %s", installationDatabaseName)

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
	if err != nil {
		return errors.Wrapf(err, "unable to find the master secret for RDS cluster ID %s", *lockedCluster.cluster.DBClusterIdentifier)
	}

	close, err := d.connectRDSCluster(rdsMySQLSchemaInformationDatabase, *lockedCluster.cluster.Endpoint, DefaultMattermostDatabaseUsername, *masterSecretValue.SecretString)
	defer close(logger)
	if err != nil {
		return errors.Wrapf(err, "unable to connect to RDS cluster ID %s", *lockedCluster.cluster.DBClusterIdentifier)
	}

	ctx, cancel := context.WithDeadline(context.Background(), time.Now().Add(DefaultMySQLContextTimeSeconds*time.Second))
	defer cancel()

	err = d.createDatabaseIfNotExist(ctx, databaseName)
	if err != nil {
		return errors.Wrapf(err, "unable to create schema in RDS cluster ID %s", *lockedCluster.cluster.DBClusterIdentifier)
	}

	installationSecret, err := d.ensureMultitenantDatabaseSecretIsCreated(lockedCluster.cluster.DBClusterIdentifier, vpc.VpcId)
	if err != nil {
		return errors.Wrapf(err, "failed to get a database secrete for installation ID %s", d.installationID)
	}

	err = d.createUserIfNotExist(ctx, installationSecret.MasterUsername, installationSecret.MasterPassword)
	if err != nil {
		return errors.Wrapf(err, "failed to create a Mattermost database schema for installation ID %s", d.installationID)
	}

	err = d.grantUserFullPermissions(ctx, databaseName, installationSecret.MasterUsername)
	if err != nil {
		return errors.Wrapf(err, "failed to grant permissions to Mattermost database user for installation ID %s", d.installationID)
	}

	databaseInstallationIDs, err := store.AddMultitenantDatabaseInstallationID(*lockedCluster.cluster.DBClusterIdentifier, d.installationID)
	if err != nil {
		return errors.Wrapf(err, "failed to database installation ID %s to the datastore", d.installationID)
	}

	err = d.updateCounterTag(lockedCluster.cluster.DBClusterArn, len(databaseInstallationIDs))
	if err != nil {
		return errors.Wrapf(err, "failed to set update counter tag for RDS cluster ID %s", *lockedCluster.cluster.DBClusterIdentifier)
	}

	logger.Infof("Multitenant RDS cluster %s has %d installations", *lockedCluster.cluster.DBClusterIdentifier, len(databaseInstallationIDs))

	return nil
}

// Helpers

type lockRDSClusterOutput struct {
	unlock  func(log.FieldLogger)
	cluster *rds.DBCluster
}

func (d *RDSMultitenantDatabase) lockRDSCluster(vpcID string, store model.InstallationDatabaseStoreInterface, logger log.FieldLogger) (*lockRDSClusterOutput, error) {
	// We get clusters with filter. Found it? Lock and return, otherwise get all clusters and see if there is one available and lockable.
	// If not, got to RDS and find one that is not in the datastore and has less databases than the limit.
	multitenantDatabases, err := store.GetMultitenantDatabases(&model.MultitenantDatabaseFilter{
		InstallationID: d.installationID,
	})
	if err != nil {
		return nil, err
	}
	if multitenantDatabases != nil && len(multitenantDatabases) == 1 {
		unlockFn, err := d.acquireLock(multitenantDatabases[0].ID, store)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to lock cluster ID %s", multitenantDatabases[0].ID)
		}

		rdsCluster, err := d.describeRDSCluster(multitenantDatabases[0].ID)
		if err != nil {
			unlockFn(logger)
			return nil, errors.Wrapf(err, "failed to get RDS cluster %s", multitenantDatabases[0].ID)
		}

		if *rdsCluster.Status != "available" {
			unlockFn(logger)
			return nil, errors.Wrapf(err, "expected RDS cluster %s to be 'available', but is '%s'", multitenantDatabases[0].ID, *rdsCluster.Status)
		}

		return &lockRDSClusterOutput{
			unlock:  unlockFn,
			cluster: rdsCluster,
		}, nil
	}

	// OK, no installation ID yet. Let's get all clusters.
	// 3 situations:
	// - no clusters at all (empty database)
	// - there are clusters but they all exceeded capacity.
	// - there are still clusters that didn't exceeded the cap, so let's try to lock one.
	// validateCLusters := func() string, error {
	multitenantDatabases, err = store.GetMultitenantDatabases(&model.MultitenantDatabaseFilter{
		NumOfInstallationsLimit: DefaultRDSMultitenantDatabaseCountLimit,
	})
	if err != nil {
		return nil, err
	}

	if len(multitenantDatabases) < 1 {
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
					Key:    aws.String(DefaultAWSTerraformProvisionedKey),
					Values: []*string{aws.String(DefaultAWSTerraformProvisionedValueTrue)},
				},
				{
					Key:    aws.String(trimTagPrefix(DefaultRDSMultitenantVPCIDTagKey)),
					Values: []*string{&vpcID},
				},
				{
					Key: aws.String(trimTagPrefix(RDSMultitenantInstallationCounterTagKey)),
				},
			},

			// TODO(gsagula): create a constant for this ARN type.
			ResourceTypeFilters: []*string{aws.String("rds:cluster")},
		})
		if err != nil {
			return nil, errors.Wrap(err, "failed to get available RDS cluster resources")
		}

		getClusterID := func(tags []*gt.Tag) (*string, error) {
			var rdsClusterID *string
			var installationCounter *string
			for _, tag := range tags {
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
						return rdsClusterID, nil
					}
				}
			}
			return nil, nil
		}

		for _, resource := range resourceNames {
			resourceARN, err := arn.Parse(*resource.ResourceARN)
			if err != nil {
				return nil, err
			}
			if !strings.Contains(resourceARN.Resource, RDSMultitenantDBClusterResourceNamePrefix) {
				continue
			}

			id, err := getClusterID(resource.Tags)
			if err != nil {
				return nil, err
			}

			if id != nil {
				multitenantDatabase := model.MultitenantDatabase{
					ID: *id,
				}

				err := store.CreateMultitenantDatabase(&multitenantDatabase)
				if err != nil {
					logger.WithError(err).Errorf("failed to create a multitenant database for installation ID %s", d.installationID)
					continue
				}

				multitenantDatabases = append(multitenantDatabases, &multitenantDatabase)
			}
		}
	}

	for _, multitenantDatabase := range multitenantDatabases {
		installations, err := multitenantDatabase.GetInstallationIDs()
		if err != nil {
			return nil, errors.Wrapf(err, "failed to get installations for multitenant database ID %s", multitenantDatabase.ID)
		}

		if len(installations) < DefaultRDSMultitenantDatabaseCountLimit {
			unlockFn, err := d.acquireLock(multitenantDatabase.ID, store)
			if err != nil {
				logger.WithError(err).Errorf("failed to lock rds cluster %s", multitenantDatabase.ID)
				continue
			}

			multitenantDatabase, err = store.GetMultitenantDatabase(multitenantDatabase.ID)
			if err != nil {
				logger.WithError(err).Errorf("failed to get database cluster ID %s", multitenantDatabase.ID)
				continue
			}

			// Since clusters can be sourced from either the datastore or AWS resources,
			// we make sure that nothing has changed prior to the locking has been acquired.
			err = d.validateDatabaseClusterInstallations(installations, multitenantDatabase)
			if err != nil {
				unlockFn(logger)
				logger.WithError(err).Error("multitenant database validation failed")
				continue
			}

			rdsCluster, err := d.describeRDSCluster(multitenantDatabase.ID)
			if err != nil {
				unlockFn(logger)
				logger.WithError(err).Errorf("failed to get RDS cluster %s", multitenantDatabase.ID)
				continue
			}

			if *rdsCluster.Status != "available" {
				unlockFn(logger)
				logger.WithError(err).Errorf("failed to get RDS cluster %s", multitenantDatabase.ID)
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

func (d *RDSMultitenantDatabase) acquireLock(multitenantDatabaseID string, store model.InstallationDatabaseStoreInterface) (func(logger log.FieldLogger), error) {
	locked, err := store.LockMultitenantDatabase(multitenantDatabaseID, d.installationID)
	if err != nil {
		return nil, errors.Wrapf(err, "could not acquire lock for multitenant database %s", multitenantDatabaseID)
	}
	if !locked {
		return nil, errors.Errorf("could not acquire lock for multitenant database %s", multitenantDatabaseID)
	}
	unlockFN := func(logger log.FieldLogger) {
		unlocked, err := store.UnlockMultitenantDatabase(multitenantDatabaseID, d.installationID, true)
		if err != nil {
			logger.WithError(err).Errorf("provisioner datastore failed to release locker for multitenant database ID %s", multitenantDatabaseID)
		}
		if !unlocked {
			logger.Warnf("multitenant database ID %s and installation ID %s is still locked", multitenantDatabaseID, d.installationID)
		}
	}

	return unlockFN, nil
}

func (d *RDSMultitenantDatabase) validateDatabaseClusterInstallations(installations model.MultitenantDatabaseInstallationIDs, multitenantDatabase *model.MultitenantDatabase) error {
	expectedInstallations, err := multitenantDatabase.GetInstallationIDs()
	if err != nil {
		return errors.Errorf("failed to get installations from multitenant database ID %s", multitenantDatabase.ID)
	}

	if len(installations) != len(expectedInstallations) {
		return errors.Errorf("supplied %s installations, but the multitenant database ID %s has %s", len(installations), len(expectedInstallations), multitenantDatabase.ID)
	}

	for _, installation := range installations {
		if !expectedInstallations.Contains(installation) {
			return errors.Errorf("unable to find installation ID %s in the multitenant database ID %s", installation, multitenantDatabase.ID)
		}
	}

	return nil
}

func (d *RDSMultitenantDatabase) removeRDSDatabase(rdsDatabaseCluster *model.MultitenantDatabase, rdsDatabaseName string, store model.InstallationDatabaseStoreInterface, logger log.FieldLogger) error {
	logger = logger.WithField("rds-cluster-id", rdsDatabaseCluster.ID)

	unlocked, err := d.acquireLock(rdsDatabaseCluster.ID, store)
	if err != nil {
		return errors.Wrapf(err, "failed to lock RDS database cluster %s", rdsDatabaseCluster.ID)
	}
	defer unlocked(logger)

	rdsCluster, err := d.describeRDSCluster(rdsDatabaseCluster.ID)
	if err != nil {
		return errors.Wrap(err, "failed get database cluster information from AWS RDS service")
	}

	masterSecretValue, err := d.client.Service().secretsManager.GetSecretValue(&secretsmanager.GetSecretValueInput{
		SecretId: &rdsDatabaseCluster.ID,
	})
	if err != nil {
		return errors.Wrapf(err, "failed to get master secret by ID %s", rdsDatabaseCluster.ID)
	}

	close, err := d.connectRDSCluster(rdsMySQLSchemaInformationDatabase, *rdsCluster.Endpoint, DefaultMattermostDatabaseUsername, *masterSecretValue.SecretString)
	if err != nil {
		return errors.Wrapf(err, "unable to connect to RDS database cluster %s", rdsDatabaseCluster.ID)
	}
	defer close(logger)

	ctx, cancel := context.WithDeadline(context.Background(), time.Now().Add(DefaultMySQLContextTimeSeconds*time.Second))
	defer cancel()

	err = d.dropDatabaseIfExists(ctx, rdsDatabaseName)
	if err != nil {
		return errors.Wrapf(err, "failed to drop RDS database name %s", rdsDatabaseName)
	}

	databaseInstallationIDs, err := store.RemoveMultitenantDatabaseInstallationID(rdsDatabaseCluster.ID, d.installationID)
	if err != nil {
		return errors.Wrapf(err, "failed to remove installation ID %s from datasore", d.installationID)
	}

	err = d.updateCounterTag(rdsCluster.DBClusterArn, len(databaseInstallationIDs))
	if err != nil {
		return errors.Wrapf(err, "failed to update counter tag for RDS cluster %s", rdsDatabaseCluster.ID)
	}

	return nil
}

func (d *RDSMultitenantDatabase) ensureMultitenantDatabaseSecretIsCreated(rdsClusterID, VpcID *string) (*RDSSecret, error) {
	installationSecretName := RDSMultitenantSecretName(d.installationID)

	installationSecretValue, err := d.client.Service().secretsManager.GetSecretValue(&secretsmanager.GetSecretValueInput{
		SecretId: aws.String(installationSecretName),
	})
	if err != nil && !IsErrorCode(err, secretsmanager.ErrCodeResourceNotFoundException) {
		return nil, errors.Wrapf(err, "failed to get multitenant RDS database secret %s", installationSecretName)
	}

	var installationSecret *RDSSecret
	if installationSecretValue != nil && installationSecretValue.SecretString != nil {
		installationSecret, err = unmarshalSecretPayload(*installationSecretValue.SecretString)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to unmarshal multitenant RDS database secret %s", installationSecretName)
		}
	} else {
		description := RDSMultitenantClusterSecretDescription(d.installationID, *rdsClusterID)
		tags := []*secretsmanager.Tag{
			{
				Key:   aws.String(trimTagPrefix(DefaultRDSMultitenantDBClusterIDTagKey)),
				Value: rdsClusterID,
			},
			{
				Key:   aws.String(trimTagPrefix(DefaultRDSMultitenantVPCIDTagKey)),
				Value: VpcID,
			},
			{
				Key:   aws.String(trimTagPrefix(DefaultMattermostInstallationIDTagKey)),
				Value: aws.String(d.installationID),
			},
		}
		installationSecret, err = d.createInstallationSecret(installationSecretName, d.installationID, description, tags)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to create a RDS database secret %s", installationSecretName)
		}
	}

	return installationSecret, nil
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
	_, err := d.db.QueryContext(ctx, "CREATE USER IF NOT EXISTS ?@? IDENTIFIED BY ?", username, "%", password)
	if err != nil {
		return errors.Wrapf(err, "creating user %s", username)
	}

	return nil
}

func (d *RDSMultitenantDatabase) createDatabaseIfNotExist(ctx context.Context, databaseName string) error {
	query := fmt.Sprintf("CREATE DATABASE IF NOT EXISTS %s CHARACTER SET ?", databaseName)

	_, err := d.db.QueryContext(ctx, query, "utf8mb4")
	if err != nil {
		return errors.Wrapf(err, "creating database name %s", databaseName)
	}

	return nil
}

func (d *RDSMultitenantDatabase) grantUserFullPermissions(ctx context.Context, databaseName, username string) error {
	query := fmt.Sprintf("GRANT ALL PRIVILEGES ON %s.* TO ?@?", databaseName)

	_, err := d.db.QueryContext(ctx, query, username, "%")
	if err != nil {
		return errors.Wrapf(err, "granting permissions to user %s", username)
	}

	// TODO(gsagula): need to flush privileges.

	return nil
}

func (d *RDSMultitenantDatabase) dropDatabaseIfExists(ctx context.Context, databaseName string) error {
	query := fmt.Sprintf("DROP DATABASE IF EXISTS %s", databaseName)

	_, err := d.db.QueryContext(ctx, query)
	if err != nil {
		return errors.Wrapf(err, "failed to drop database %s", databaseName)
	}

	return nil
}
