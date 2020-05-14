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

	// MySQL implementation of database/sql
	_ "github.com/go-sql-driver/mysql"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// SQLDatabaseManager is an interface that describes operations to query and to
// close connection with a database. It's used mainly to implement a client that
// needs to perform non-complex queries in a SQL database instance.
type SQLDatabaseManager interface {
	QueryContext(ctx context.Context, query string, args ...interface{}) (*sql.Rows, error)
	Close() error
}

// RDSMultitenantDatabase is a database backed by RDS that supports multi-tenancy.
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

	logger.Info("Tearing down RDS database and database secret")

	multitenantDatabase, err := store.GetMultitenantDatabaseForInstallationID(d.installationID)
	if err != nil {
		return errors.Wrapf(err, "unable to find multitenant database for installation ID %s", d.installationID)
	}

	unlocked, err := d.lockMultitenantDatabase(multitenantDatabase.ID, store)
	if err != nil {
		return errors.Wrapf(err, "failed to lock multitenant database ID %s", multitenantDatabase.ID)
	}
	defer unlocked(logger)

	rdsCluster, err := d.describeRDSCluster(multitenantDatabase.ID)
	if err != nil {
		return errors.Wrapf(err, "failed to describe multitenant database ID %s", multitenantDatabase.ID)
	}

	logger = logger.WithField("rds-cluster-id", *rdsCluster.DBClusterIdentifier)

	err = d.dropDatabaseAndDeleteSecret(multitenantDatabase.ID, *rdsCluster.Endpoint, databaseName, store, logger)
	if err != nil {
		return errors.Wrap(err, "unable to finish teardown of multitenant database")
	}

	err = d.updateTagCounterAndRemoveInstallationID(rdsCluster.DBClusterArn, multitenantDatabase, store, logger)
	if err != nil {
		return errors.Wrap(err, "unable to finish teardown of multitenant database")
	}

	logger.Infof("Multitenant RDS database teardown complete")

	return nil
}

// Snapshot creates a snapshot of single RDS multitenant database.
func (d *RDSMultitenantDatabase) Snapshot(store model.InstallationDatabaseStoreInterface, logger log.FieldLogger) error {
	return errors.New("not implemented")
}

// GenerateDatabaseSpecAndSecret creates the k8s database spec and secret for accessing a single database inside
// a RDS multitenant cluster.
func (d *RDSMultitenantDatabase) GenerateDatabaseSpecAndSecret(store model.InstallationDatabaseStoreInterface, logger log.FieldLogger) (*mmv1alpha1.Database, *corev1.Secret, error) {
	installationDatabaseName := MattermostRDSDatabaseName(d.installationID)

	logger = logger.WithField("rds-multitenant-database", installationDatabaseName)

	multitenantDatabase, err := store.GetMultitenantDatabaseForInstallationID(d.installationID)
	if err != nil {
		return nil, nil, errors.Wrapf(err, "unable to find multitenant database for installation ID %s", d.installationID)
	}

	unlocked, err := d.lockMultitenantDatabase(multitenantDatabase.ID, store)
	if err != nil {
		return nil, nil, errors.Wrapf(err, "failed to lock multitenant database ID %s", multitenantDatabase.ID)
	}
	defer unlocked(logger)

	rdsCluster, err := d.describeRDSCluster(multitenantDatabase.ID)
	if err != nil {
		return nil, nil, errors.Wrapf(err, "failed to describe RDS cluster ID %s", multitenantDatabase.ID)
	}

	logger = logger.WithField("rds-cluster-id", *rdsCluster.DBClusterIdentifier)

	installationSecretName := RDSMultitenantSecretName(d.installationID)

	result, err := d.client.Service().secretsManager.GetSecretValue(&secretsmanager.GetSecretValueInput{
		SecretId: &installationSecretName,
	})
	if err != nil {
		return nil, nil, errors.Wrap(err, "failed to get multitenant database spec and secret from AWS Secrets Manager Service")
	}

	installationSecret, err := unmarshalSecretPayload(*result.SecretString)
	if err != nil {
		return nil, nil, errors.Wrap(err, "failed to generate multitenant database spec and secret")
	}

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

	logger.Infof("Finished to set up spec and secret for multitenant database name %s", installationDatabaseName)

	return databaseSpec, databaseSecret, nil
}

// Provision claims a multitenant RDS cluster and creates a database schema for the installation.
func (d *RDSMultitenantDatabase) Provision(store model.InstallationDatabaseStoreInterface, logger log.FieldLogger) error {
	installationDatabaseName := MattermostRDSDatabaseName(d.installationID)

	logger = logger.WithField("multitenant-rds-database", installationDatabaseName)

	vpc, err := d.getClusterInstallationVPC(store)
	if err != nil {
		return errors.Wrapf(err, "unable to find a VPC that hosts the installation ID %s", d.installationID)
	}

	lockedRDSCluster, err := d.findRDSClusterForInstallation(*vpc.VpcId, store, logger)
	if err != nil {
		return errors.Wrapf(err, "unable to lock a multitenant database for installation ID %s", d.installationID)
	}
	defer lockedRDSCluster.unlock(logger)

	logger = logger.WithField("rds-cluster-id", *lockedRDSCluster.cluster.DBClusterIdentifier)

	masterSecretValue, err := d.client.Service().secretsManager.GetSecretValue(&secretsmanager.GetSecretValueInput{
		SecretId: lockedRDSCluster.cluster.DBClusterIdentifier,
	})
	if err != nil {
		return errors.Wrapf(err, "unable to find the master secret for the multitenant RDS cluster ID %s", *lockedRDSCluster.cluster.DBClusterIdentifier)
	}

	close, err := d.connectRDSCluster(rdsMySQLSchemaInformationDatabase, *lockedRDSCluster.cluster.Endpoint, DefaultMattermostDatabaseUsername, *masterSecretValue.SecretString)
	if err != nil {
		return errors.Wrapf(err, "unable to connect to the multitenant RDS cluster ID %s", *lockedRDSCluster.cluster.DBClusterIdentifier)
	}
	defer close(logger)

	ctx, cancel := context.WithDeadline(context.Background(), time.Now().Add(DefaultMySQLContextTimeSeconds*time.Second))
	defer cancel()

	err = d.createDatabaseIfNotExist(ctx, installationDatabaseName)
	if err != nil {
		return errors.Wrapf(err, "unable to create schema in multitenant RDS cluster ID %s", *lockedRDSCluster.cluster.DBClusterIdentifier)
	}

	installationSecret, err := d.ensureMultitenantDatabaseSecretIsCreated(lockedRDSCluster.cluster.DBClusterIdentifier, vpc.VpcId)
	if err != nil {
		return errors.Wrapf(err, "failed to get a database secrete for installation ID %s", d.installationID)
	}

	err = d.createUserIfNotExist(ctx, installationSecret.MasterUsername, installationSecret.MasterPassword)
	if err != nil {
		return errors.Wrapf(err, "failed to create a Mattermost database schema for installation ID %s", d.installationID)
	}

	err = d.grantUserFullPermissions(ctx, installationDatabaseName, installationSecret.MasterUsername)
	if err != nil {
		return errors.Wrapf(err, "failed to grant permissions to Mattermost database user for installation ID %s", d.installationID)
	}

	databaseInstallationIDs, err := store.AddMultitenantDatabaseInstallationID(*lockedRDSCluster.cluster.DBClusterIdentifier, d.installationID)
	if err != nil {
		return errors.Wrapf(err, "failed to add the installation ID %s to multitenant database ID %s", d.installationID, *lockedRDSCluster.cluster.DBClusterIdentifier)
	}

	err = d.updateCounterTag(lockedRDSCluster.cluster.DBClusterArn, len(databaseInstallationIDs))
	if err != nil {
		return errors.Wrapf(err, "failed to update tag:counter in RDS cluster ID %s", *lockedRDSCluster.cluster.DBClusterIdentifier)
	}

	logger.Infof("Multitenant database ID %s has %d installations", *lockedRDSCluster.cluster.DBClusterIdentifier, len(databaseInstallationIDs))

	return nil
}

// Helpers

// An instance of this object holds a locked multitenant RDS cluster and an unlock function. Locked RDS clusters
// are ready to receive a new database installation.
type rdsClusterOutput struct {
	unlock  func(log.FieldLogger)
	cluster *rds.DBCluster
}

// Ensure that an RDS cluster is in available state and validates cluster attributes before and after locking
// the resource for an installation.
func (d *RDSMultitenantDatabase) validateAndLockRDSCluster(multitenantDatabases []*model.MultitenantDatabase, store model.InstallationDatabaseStoreInterface, logger log.FieldLogger) (*rdsClusterOutput, error) {
	for _, multitenantDatabase := range multitenantDatabases {
		installationIDs, err := multitenantDatabase.GetInstallationIDs()
		if err != nil {
			return nil, errors.Wrapf(err, "failed to get installations for multitenant database ID %s", multitenantDatabase.ID)
		}

		if len(installationIDs) < DefaultRDSMultitenantDatabaseCountLimit || installationIDs.Contains(d.installationID) {
			rdsClusterOutput, err := d.multitenantDatabaseToRDSClusterLock(multitenantDatabase.ID, installationIDs, store, logger)
			if err != nil {
				logger.WithError(err).Errorf("Failed to lock multitenant database ID %s", multitenantDatabase.ID)
				continue
			}

			return rdsClusterOutput, nil
		}
	}

	return nil, errors.New("unable to find and lock a available multitenant database")
}

// This helper method finds a multitenant RDS cluster that is ready for receiving a database installation. The lookup
// for multitenant databases will happen in order:
// 	1. a multitenant databases by installation ID.
// 	2. all multitenant databases in the store that are under the max number of installations allowed.
// 	3. all multitenant databases in the RDS cluster that are under the max number of installations allowed.
func (d *RDSMultitenantDatabase) findRDSClusterForInstallation(vpcID string, store model.InstallationDatabaseStoreInterface, logger log.FieldLogger) (*rdsClusterOutput, error) {
	multitenantDatabases, err := store.GetMultitenantDatabases(&model.MultitenantDatabaseFilter{
		InstallationID: d.installationID,
		PerPage:        model.AllPerPage,
	})
	if err != nil {
		return nil, errors.Wrapf(err, "unable get multitenant database for installation ID %s", d.installationID)
	}

	logger.Infof("Could not find a multitenant database for installation ID %s. Fetching all available resources in the datastore.")

	if len(multitenantDatabases) == 0 {
		multitenantDatabases, err = store.GetMultitenantDatabases(&model.MultitenantDatabaseFilter{
			NumOfInstallationsLimit: DefaultRDSMultitenantDatabaseCountLimit,
			PerPage:                 model.AllPerPage,
		})
		if err != nil {
			return nil, err
		}
	}

	logger.Infof("Could not find any multitenant database with less than %d installations in the datastore. Fetching all available resources in AWS.", DefaultRDSMultitenantDatabaseCountLimit)

	if len(multitenantDatabases) == 0 {
		multitenantDatabases, err = d.getMultitenantDatabasesFromResourceTags(vpcID, store, logger)
		if err != nil {
			return nil, err
		}
	}

	lockedRDSCluster, err := d.validateAndLockRDSCluster(multitenantDatabases, store, logger)
	if err != nil {
		return nil, errors.Wrapf(err, "could not find a multitenant RDS cluster available in vpc %s", vpcID)
	}

	return lockedRDSCluster, nil
}

func (d *RDSMultitenantDatabase) multitenantDatabaseToRDSClusterLock(multitenantDatabaseID string, databaseInstallationIDs model.MultitenantDatabaseInstallationIDs, store model.InstallationDatabaseStoreInterface, logger log.FieldLogger) (*rdsClusterOutput, error) {
	unlockFn, err := d.lockMultitenantDatabase(multitenantDatabaseID, store)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to lock multitenant database ID %s", multitenantDatabaseID)
	}

	err = d.validateMultitenantDatabaseInstallations(multitenantDatabaseID, databaseInstallationIDs, store)
	if err != nil {
		unlockFn(logger)
		return nil, errors.Wrapf(err, "multitenant database ID %s validation failed", multitenantDatabaseID)
	}

	rdsCluster, err := d.describeRDSCluster(multitenantDatabaseID)
	if err != nil {
		unlockFn(logger)
		return nil, errors.Wrapf(err, "failed to describe the multitenant RDS cluster ID %s", multitenantDatabaseID)
	}

	if *rdsCluster.Status != DefaultRDSStatusAvailable {
		unlockFn(logger)
		return nil, errors.Wrapf(err, "multitenant RDS cluster ID %s is not available (status: %s)", multitenantDatabaseID, *rdsCluster.Status)
	}

	rdsClusterOutput := rdsClusterOutput{
		unlock:  unlockFn,
		cluster: rdsCluster,
	}

	return &rdsClusterOutput, nil
}

func (d *RDSMultitenantDatabase) getMultitenantDatabasesFromResourceTags(vpcID string, store model.InstallationDatabaseStoreInterface, logger log.FieldLogger) ([]*model.MultitenantDatabase, error) {
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
				Key:    aws.String(trimTagPrefix(DefaultRDSMultitenantDatabaseTypeTagKey)),
				Values: []*string{aws.String(DefaultRDSMultitenantDatabaseTypeTagValue)},
			},
			{
				Key:    aws.String(trimTagPrefix(DefaultRDSMultitenantVPCIDTagKey)),
				Values: []*string{&vpcID},
			},
			{
				Key: aws.String(trimTagPrefix(RDSMultitenantInstallationCounterTagKey)),
			},
			{
				Key: aws.String(trimTagPrefix(DefaultRDSMultitenantDatabaseIDTagKey)),
			},
		},
		ResourceTypeFilters: []*string{aws.String(DefaultResourceTypeClusterRDS)},
	})
	if err != nil {
		return nil, errors.Wrap(err, "failed to get available multitenant RDS resources")
	}

	var multitenantDatabases []*model.MultitenantDatabase

	for _, resource := range resourceNames {
		resourceARN, err := arn.Parse(*resource.ResourceARN)
		if err != nil {
			return nil, err
		}
		if !strings.Contains(resourceARN.Resource, RDSMultitenantDBClusterResourceNamePrefix) {
			logger.Warnf("Provisioner skipped RDS resource (%s) because name does not have a correct multitenant database prefix (%s)", resourceARN.Resource, RDSMultitenantDBClusterResourceNamePrefix)
			continue
		}

		rdsClusterID, err := d.getRDSClusterIDFromResourceTags(resource.Tags)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to get a multitenant RDS cluster ID from AWS resource tags")
		}

		if rdsClusterID != nil {
			multitenantDatabase := model.MultitenantDatabase{
				ID: *rdsClusterID,
			}

			err := store.CreateMultitenantDatabase(&multitenantDatabase)
			if err != nil {
				logger.WithError(err).Errorf("Failed to create a multitenant database for installation ID %s. Skipping RDS cluster ID %s", d.installationID, *rdsClusterID)
				continue
			}

			multitenantDatabases = append(multitenantDatabases, &multitenantDatabase)
		}
	}

	return multitenantDatabases, nil
}

func (d *RDSMultitenantDatabase) getRDSClusterIDFromResourceTags(resourceTags []*gt.Tag) (*string, error) {
	var rdsClusterID *string
	var installationCounter *string

	for _, tag := range resourceTags {
		if *tag.Key == trimTagPrefix(RDSMultitenantInstallationCounterTagKey) && tag.Value != nil {
			installationCounter = tag.Value
		}

		if *tag.Key == trimTagPrefix(DefaultRDSMultitenantDatabaseIDTagKey) && tag.Value != nil {
			rdsClusterID = tag.Value
		}

		if rdsClusterID != nil && installationCounter != nil {
			counter, err := strconv.Atoi(*installationCounter)
			if err != nil {
				return nil, errors.Wrap(err, "failed to parse string tag:counter to integer")
			}

			if counter < DefaultRDSMultitenantDatabaseCountLimit {
				return rdsClusterID, nil
			}
		}
	}

	return nil, nil
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
		return nil, fmt.Errorf("Multitenant RDS provisioning is not currently supported for more than one cluster installation (found %d)", clusterInstallationCount)
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
		return nil, errors.Wrapf(err, "unable to lookup the VPC for installation ID %s", d.installationID)
	}
	if len(vpcs) != 1 {
		return nil, fmt.Errorf("expected 1 VPC for multitenant RDS cluster ID %s (found %d)", clusterInstallations[0].ClusterID, len(vpcs))
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
		return errors.Wrapf(err, "failed to update %s for the multitenant RDS cluster %s", DefaultMultitenantDatabaseCounterTagKey, *resourceARN)
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
		return nil, errors.Wrapf(err, "failed to get multitenant RDS cluster id %s", dbClusterID)
	}
	if len(dbClusterOutput.DBClusters) != 1 {
		return nil, fmt.Errorf("expected exactly one multitenant RDS cluster for installation id %s (found %d)", d.installationID, len(dbClusterOutput.DBClusters))
	}

	return dbClusterOutput.DBClusters[0], nil
}

func (d *RDSMultitenantDatabase) lockMultitenantDatabase(multitenantDatabaseID string, store model.InstallationDatabaseStoreInterface) (func(logger log.FieldLogger), error) {
	locked, err := store.LockMultitenantDatabase(multitenantDatabaseID, d.installationID)
	if err != nil {
		return nil, errors.Wrapf(err, "could not acquire lock for multitenant database ID %s", multitenantDatabaseID)
	}
	if !locked {
		return nil, errors.Errorf("could not acquire lock for multitenant database ID %s", multitenantDatabaseID)
	}

	unlockFN := func(logger log.FieldLogger) {
		unlocked, err := store.UnlockMultitenantDatabase(multitenantDatabaseID, d.installationID, true)
		if err != nil {
			logger.WithError(err).Errorf("provisioner datastore failed to release locker for multitenant database ID %s", multitenantDatabaseID)
		}
		if !unlocked {
			logger.Warnf("multitenant database ID %s and installation ID %s are still locked", multitenantDatabaseID, d.installationID)
		}
	}

	return unlockFN, nil
}

func (d *RDSMultitenantDatabase) validateMultitenantDatabaseInstallations(multitenantDatabaseID string, installations model.MultitenantDatabaseInstallationIDs, store model.InstallationDatabaseStoreInterface) error {
	multitenantDatabase, err := store.GetMultitenantDatabase(multitenantDatabaseID)
	if err != nil {
		return errors.Wrapf(err, "failed to get multitenant database ID %s", multitenantDatabaseID)
	}

	expectedInstallations, err := multitenantDatabase.GetInstallationIDs()
	if err != nil {
		return errors.Errorf("failed to get installations from multitenant database ID %s", multitenantDatabase.ID)
	}

	if len(installations) != len(expectedInstallations) {
		return errors.Errorf("supplied %d installations, but multitenant database ID %s has %d", len(installations), multitenantDatabase.ID, len(expectedInstallations))
	}

	for _, installation := range installations {
		if !expectedInstallations.Contains(installation) {
			return errors.Errorf("unable to find installation ID %s in the multitenant database ID %s", installation, multitenantDatabase.ID)
		}
	}

	return nil
}

func (d *RDSMultitenantDatabase) dropDatabaseAndDeleteSecret(rdsClusterID, rdsClusterendpoint, databaseName string, store model.InstallationDatabaseStoreInterface, logger log.FieldLogger) error {
	masterSecretValue, err := d.client.Service().secretsManager.GetSecretValue(&secretsmanager.GetSecretValueInput{
		SecretId: aws.String(rdsClusterID),
	})
	if err != nil {
		return errors.Wrapf(err, "failed to get master secret by ID %s", rdsClusterID)
	}

	close, err := d.connectRDSCluster(rdsMySQLSchemaInformationDatabase, rdsClusterendpoint, DefaultMattermostDatabaseUsername, *masterSecretValue.SecretString)
	if err != nil {
		return errors.Wrapf(err, "unable to connect to multitenant RDS cluster ID %s", rdsClusterID)
	}
	defer close(logger)

	ctx, cancel := context.WithDeadline(context.Background(), time.Now().Add(DefaultMySQLContextTimeSeconds*time.Second))
	defer cancel()

	err = d.dropDatabaseIfExists(ctx, databaseName)
	if err != nil {
		return errors.Wrapf(err, "failed to drop multitenant RDS database name %s", databaseName)
	}

	multitenantDatabaseSecretName := RDSMultitenantSecretName(d.installationID)

	_, err = d.client.Service().secretsManager.DeleteSecret(&secretsmanager.DeleteSecretInput{
		SecretId: aws.String(multitenantDatabaseSecretName),
	})
	if err != nil && !IsErrorCode(err, secretsmanager.ErrCodeResourceNotFoundException) {
		return errors.Wrapf(err, "failed to delete multitenant database secret name %s", multitenantDatabaseSecretName)
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
				Key:   aws.String(trimTagPrefix(DefaultRDSMultitenantDatabaseIDTagKey)),
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
			return nil, errors.Wrapf(err, "failed to create a multitenant RDS database secret %s", installationSecretName)
		}
	}

	return installationSecret, nil
}

func (d *RDSMultitenantDatabase) updateTagCounterAndRemoveInstallationID(dbClusterArn *string, multitenantDatabase *model.MultitenantDatabase, store model.InstallationDatabaseStoreInterface, logger log.FieldLogger) error {
	installationIDs, err := multitenantDatabase.GetInstallationIDs()
	if err != nil {
		return errors.Wrapf(err, "failed to get installations from multitenant database ID %s", multitenantDatabase.ID)
	}

	numOfInstallations := len(installationIDs)

	err = d.updateCounterTag(dbClusterArn, numOfInstallations-1)
	if err != nil {
		return errors.Wrap(err, "failed to update counter tag for the multitenant")
	}

	// We need to update the tag before removing the installation ID from the datastore. However, if this
	// operation fails, tag:counter in RDS needs to return to the original value.
	_, err = store.RemoveMultitenantDatabaseInstallationID(multitenantDatabase.ID, d.installationID)
	if err != nil {
		logger.WithError(err).Warnf("Failed to remove multitenant database from datastore. Rolling tag:counter value back to %d", numOfInstallations)
		updateTagErr := d.updateCounterTag(dbClusterArn, numOfInstallations)
		if updateTagErr != nil {
			logger.WithError(err).Warnf("Failed to roll back tag:counter. Value is still %d", numOfInstallations-1)
		}
		return errors.Wrapf(err, "failed to remove installation ID %s from datasore", d.installationID)
	}

	return nil
}

func (d *RDSMultitenantDatabase) connectRDSCluster(schema, endpoint, username, password string) (func(logger log.FieldLogger), error) {
	// This condition allows injecting a mocked implementation of SQLDatabaseManager interface.
	if d.db == nil {
		db, err := sql.Open("mysql", RDSMySQLConnString(schema, endpoint, username, password))
		if err != nil {
			return nil, errors.Wrapf(err, "connecting to multitenant RDS cluster endpoint %s", endpoint)
		}

		d.db = db
	}

	closeFunc := func(logger log.FieldLogger) {
		err := d.db.Close()
		if err != nil {
			logger.WithError(err).Errorf("failed to close the connection with multitenant RDS cluster endpoint %s", endpoint)
		}
	}

	return closeFunc, nil
}

func (d *RDSMultitenantDatabase) createUserIfNotExist(ctx context.Context, username, password string) error {
	_, err := d.db.QueryContext(ctx, "CREATE USER IF NOT EXISTS ?@? IDENTIFIED BY ?", username, "%", password)
	if err != nil {
		return errors.Wrapf(err, "creating user %s", username)
	}

	return nil
}

func (d *RDSMultitenantDatabase) createDatabaseIfNotExist(ctx context.Context, databaseName string) error {
	// Query placeholders don't seem to work with argument database.
	// See https://github.com/mattermost/mattermost-cloud/pull/209#discussion_r422533477
	query := fmt.Sprintf("CREATE DATABASE IF NOT EXISTS %s CHARACTER SET ?", databaseName)

	_, err := d.db.QueryContext(ctx, query, "utf8mb4")
	if err != nil {
		return errors.Wrapf(err, "creating database name %s", databaseName)
	}

	return nil
}

func (d *RDSMultitenantDatabase) grantUserFullPermissions(ctx context.Context, databaseName, username string) error {
	// Query placeholders don't seem to work with argument database.
	// See https://github.com/mattermost/mattermost-cloud/pull/209#discussion_r422533477
	query := fmt.Sprintf("GRANT ALL PRIVILEGES ON %s.* TO ?@?", databaseName)

	_, err := d.db.QueryContext(ctx, query, username, "%")
	if err != nil {
		return errors.Wrapf(err, "granting permissions to user %s", username)
	}

	return nil
}

func (d *RDSMultitenantDatabase) dropDatabaseIfExists(ctx context.Context, databaseName string) error {
	// Query placeholders don't seem to work with argument database.
	// See https://github.com/mattermost/mattermost-cloud/pull/209#discussion_r422533477
	query := fmt.Sprintf("DROP DATABASE IF EXISTS %s", databaseName)

	_, err := d.db.QueryContext(ctx, query)
	if err != nil {
		return errors.Wrapf(err, "failed to drop database %s", databaseName)
	}

	return nil
}
