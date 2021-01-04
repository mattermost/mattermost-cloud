// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.
//

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
	"github.com/aws/aws-sdk-go/service/rds"
	gt "github.com/aws/aws-sdk-go/service/resourcegroupstaggingapi"
	"github.com/aws/aws-sdk-go/service/secretsmanager"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/mattermost/mattermost-cloud/model"
	mmv1alpha1 "github.com/mattermost/mattermost-operator/apis/mattermost/v1alpha1"

	// Database drivers
	_ "github.com/go-sql-driver/mysql"
	_ "github.com/lib/pq"
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
	databaseType   string
	installationID string
	instanceID     string
	db             SQLDatabaseManager
	client         *Client
}

// NewRDSMultitenantDatabase returns a new instance of RDSMultitenantDatabase that implements database interface.
func NewRDSMultitenantDatabase(databaseType, instanceID, installationID string, client *Client) *RDSMultitenantDatabase {
	return &RDSMultitenantDatabase{
		databaseType:   databaseType,
		instanceID:     instanceID,
		installationID: installationID,
		client:         client,
	}
}

// IsValid returns if the given RDSMultitenantDatabase configuration is valid.
func (d *RDSMultitenantDatabase) IsValid() error {
	if len(d.installationID) == 0 {
		return errors.New("installation ID is not set")
	}

	switch d.databaseType {
	case model.DatabaseEngineTypeMySQL,
		model.DatabaseEngineTypePostgres:
	default:
		return errors.Errorf("invalid database type %s", d.databaseType)
	}

	return nil
}

// DatabaseTypeTagValue returns the tag value used for filtering RDS cluster
// resources based on database type.
func (d *RDSMultitenantDatabase) DatabaseTypeTagValue() string {
	if d.databaseType == model.DatabaseEngineTypeMySQL {
		return DatabaseTypeMySQLAurora
	}

	return DatabaseTypePostgresSQLAurora
}

// MaxSupportedDatabases returns the maximum number of databases supported on
// one RDS cluster for this database type.
func (d *RDSMultitenantDatabase) MaxSupportedDatabases() int {
	if d.databaseType == model.DatabaseEngineTypeMySQL {
		return DefaultRDSMultitenantDatabaseMySQLCountLimit
	}

	return DefaultRDSMultitenantDatabasePostgresCountLimit
}

// Provision claims a multitenant RDS cluster and creates a database schema for
// the installation.
func (d *RDSMultitenantDatabase) Provision(store model.InstallationDatabaseStoreInterface, logger log.FieldLogger) error {
	err := d.IsValid()
	if err != nil {
		return errors.Wrap(err, "multitenant database configuration is invalid")
	}

	installationDatabaseName := MattermostRDSDatabaseName(d.installationID)

	logger = logger.WithFields(log.Fields{
		"multitenant-rds-database": installationDatabaseName,
		"database-type":            d.databaseType,
	})
	logger.Info("Provisioning Multitenant AWS RDS database")

	vpc, err := getVPCForInstallation(d.installationID, store, d.client)
	if err != nil {
		return errors.Wrap(err, "failed to find cluster installation VPC")
	}

	database, unlockFn, err := d.getAndLockAssignedMultitenantDatabase(store, logger)
	if err != nil {
		return errors.Wrap(err, "failed to get and lock assigned database")
	}
	if database == nil {
		logger.Debug("Assigning installation to multitenant database")
		database, unlockFn, err = d.assignInstallationToMultitenantDatabaseAndLock(*vpc.VpcId, store, logger)
		if err != nil {
			return errors.Wrap(err, "failed to assign installation to a multitenant database")
		}
	}
	defer unlockFn()
	logger = logger.WithField("assigned-database", database.ID)

	rdsCluster, err := d.describeRDSCluster(database.ID)
	if err != nil {
		return errors.Wrapf(err, "failed to describe the multitenant RDS cluster ID %s", database.ID)
	}
	if *rdsCluster.Status != DefaultRDSStatusAvailable {
		return errors.Errorf("multitenant RDS cluster ID %s is not available (status: %s)", database.ID, *rdsCluster.Status)
	}

	rdsID := *rdsCluster.DBClusterIdentifier
	logger = logger.WithField("rds-cluster-id", rdsID)

	err = d.runProvisionSQLCommands(installationDatabaseName, *vpc.VpcId, rdsCluster, logger)
	if err != nil {
		return errors.Wrap(err, "failed to run provisioning sql commands")
	}

	err = d.updateCounterTag(rdsCluster.DBClusterArn, database.Installations.Count())
	if err != nil {
		return errors.Wrapf(err, "failed to update tag:counter in RDS cluster ID %s", *rdsCluster.DBClusterIdentifier)
	}
	logger.Debugf("Multitenant database %s counter value updated to %d", database.ID, database.Installations.Count())

	logger.Infof("Installation %s assigned to multitenant database", d.installationID)

	return nil
}

// Snapshot creates a snapshot of single RDS multitenant database.
func (d *RDSMultitenantDatabase) Snapshot(store model.InstallationDatabaseStoreInterface, logger log.FieldLogger) error {
	return errors.New("not implemented")
}

// GenerateDatabaseSpecAndSecret creates the k8s database spec and secret for
// accessing a single database inside a RDS multitenant cluster.
func (d *RDSMultitenantDatabase) GenerateDatabaseSpecAndSecret(store model.InstallationDatabaseStoreInterface, logger log.FieldLogger) (*mmv1alpha1.Database, *corev1.Secret, error) {
	err := d.IsValid()
	if err != nil {
		return nil, nil, errors.Wrap(err, "multitenant database configuration is invalid")
	}

	installationDatabaseName := MattermostRDSDatabaseName(d.installationID)

	logger = logger.WithFields(log.Fields{
		"multitenant-rds-database": installationDatabaseName,
		"database-type":            d.databaseType,
	})

	multitenantDatabase, err := store.GetMultitenantDatabaseForInstallationID(d.installationID)
	if err != nil {
		return nil, nil, errors.Wrap(err, "failed to query for the multitenant database")
	}

	// TODO: probably split this up.
	unlock, err := d.lockMultitenantDatabase(multitenantDatabase.ID, store, logger)
	if err != nil {
		return nil, nil, errors.Wrap(err, "failed to lock multitenant database")
	}
	defer unlock()

	rdsCluster, err := d.describeRDSCluster(multitenantDatabase.ID)
	if err != nil {
		return nil, nil, errors.Wrap(err, "failed to describe RDS cluster")
	}

	logger = logger.WithField("rds-cluster-id", *rdsCluster.DBClusterIdentifier)

	installationSecretName := RDSMultitenantSecretName(d.installationID)

	result, err := d.client.Service().secretsManager.GetSecretValue(&secretsmanager.GetSecretValueInput{
		SecretId: &installationSecretName,
	})
	if err != nil {
		return nil, nil, errors.Wrap(err, "failed to get secret value for database")
	}

	installationSecret, err := unmarshalSecretPayload(*result.SecretString)
	if err != nil {
		return nil, nil, errors.Wrap(err, "failed to unmarshal secret payload")
	}

	var databaseConnectionString, databaseReadReplicasString, databaseConnectionCheck string
	if d.databaseType == model.DatabaseEngineTypeMySQL {
		databaseConnectionString, databaseReadReplicasString =
			MattermostMySQLConnStrings(
				installationDatabaseName,
				installationSecret.MasterUsername,
				installationSecret.MasterPassword,
				rdsCluster,
			)
		databaseConnectionCheck = fmt.Sprintf("http://%s:3306", *rdsCluster.Endpoint)
	} else {
		databaseConnectionString, databaseReadReplicasString =
			MattermostPostgresConnStrings(
				installationDatabaseName,
				installationSecret.MasterUsername,
				installationSecret.MasterPassword,
				rdsCluster,
			)
		databaseConnectionCheck = databaseConnectionString
	}
	secretStringData := map[string]string{
		"DB_CONNECTION_STRING":              databaseConnectionString,
		"MM_SQLSETTINGS_DATASOURCEREPLICAS": databaseReadReplicasString,
	}
	if len(databaseConnectionCheck) != 0 {
		secretStringData["DB_CONNECTION_CHECK_URL"] = databaseConnectionCheck
	}

	databaseSecret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name: installationSecretName,
		},
		StringData: secretStringData,
	}

	databaseSpec := &mmv1alpha1.Database{
		Secret: installationSecretName,
	}

	logger.Debug("AWS RDS multitenant database configuration generated for cluster installation")

	return databaseSpec, databaseSecret, nil
}

// Teardown removes all AWS resources related to a RDS multitenant database.
func (d *RDSMultitenantDatabase) Teardown(store model.InstallationDatabaseStoreInterface, keepData bool, logger log.FieldLogger) error {
	logger = logger.WithField("rds-multitenant-database", MattermostRDSDatabaseName(d.installationID))

	logger.Info("Tearing down RDS multitenant database")

	err := d.IsValid()
	if err != nil {
		return errors.Wrap(err, "multitenant database configuration is invalid")
	}

	if keepData {
		logger.Warn("Keepdata is set to true on this server, but this is not yet supported for RDS multitenant databases")
	}

	database, unlockFn, err := d.getAndLockAssignedMultitenantDatabase(store, logger)
	if err != nil {
		return errors.Wrap(err, "failed to get assigned multitenant database")
	}
	if database != nil {
		defer unlockFn()
		err = d.removeInstallationFromMultitenantDatabase(database, store, logger)
		if err != nil {
			return errors.Wrap(err, "failed to remove installation database")
		}
	} else {
		logger.Debug("No multitenant databases found for this installation; skipping...")
	}

	logger.Info("Multitenant RDS database teardown complete")

	return nil
}

// Helpers

// getAssignedMultitenantDatabaseResources returns the assigned multitenant
// database if there is one or nil if there is not. An error is returned if the
// installation is assigned to more than one database.
func (d *RDSMultitenantDatabase) getAndLockAssignedMultitenantDatabase(store model.InstallationDatabaseStoreInterface, logger log.FieldLogger) (*model.MultitenantDatabase, func(), error) {
	multitenantDatabases, err := store.GetMultitenantDatabases(&model.MultitenantDatabaseFilter{
		InstallationID:        d.installationID,
		MaxInstallationsLimit: model.NoInstallationsLimit,
		PerPage:               model.AllPerPage,
	})
	if err != nil {
		return nil, nil, errors.Wrap(err, "failed to query for multitenant databases")
	}
	if len(multitenantDatabases) > 1 {
		return nil, nil, errors.Errorf("expected no more than 1 assigned database for installation, but found %d", len(multitenantDatabases))
	}
	if len(multitenantDatabases) == 0 {
		return nil, nil, nil
	}

	unlockFn, err := d.lockMultitenantDatabase(multitenantDatabases[0].ID, store, logger)
	if err != nil {
		return nil, nil, errors.Wrap(err, "failed to lock multitenant database")
	}
	// Take no chances that the stored multitenant database was updated between
	// retrieving it and locking it. We know this installation is assigned to
	// exactly one multitenant database at this point so we can use the store
	// function to directly retrieve it.
	database, err := store.GetMultitenantDatabaseForInstallationID(d.installationID)
	if err != nil {
		unlockFn()
		return nil, nil, errors.Wrap(err, "failed to refresh multitenant database after lock")
	}

	return database, unlockFn, nil
}

// This helper method finds a multitenant RDS cluster that is ready for receiving a database installation. The lookup
// for multitenant databases will happen in order:
//	1. fetch a multitenant database by installation ID.
//	2. fetch all multitenant databases in the store which are under the max number of installations limit.
//	3. fetch all multitenant databases in the RDS cluster that are under the max number of installations limit.
func (d *RDSMultitenantDatabase) assignInstallationToMultitenantDatabaseAndLock(vpcID string, store model.InstallationDatabaseStoreInterface, logger log.FieldLogger) (*model.MultitenantDatabase, func(), error) {
	multitenantDatabases, err := store.GetMultitenantDatabases(&model.MultitenantDatabaseFilter{
		DatabaseType:          d.databaseType,
		MaxInstallationsLimit: d.MaxSupportedDatabases(),
		VpcID:                 vpcID,
		PerPage:               model.AllPerPage,
	})
	if err != nil {
		return nil, nil, errors.Wrap(err, "failed to get available multitenant databases")
	}

	if len(multitenantDatabases) == 0 {
		logger.Infof("No %s multitenant databases with less than %d installations found in the datastore; fetching all available resources from AWS", d.databaseType, d.MaxSupportedDatabases())

		multitenantDatabases, err = d.getMultitenantDatabasesFromResourceTags(vpcID, store, logger)
		if err != nil {
			return nil, nil, errors.Wrap(err, "failed to fetch new multitenant databases from AWS")
		}
	}

	if len(multitenantDatabases) == 0 {
		return nil, nil, errors.New("no multitenant databases are currently available for new installations")
	}

	// We want to be smart about how we assign the installation to a database.
	// Find the database with the most installations on it to keep utilization
	// as close to maximim efficiency as possible.
	// TODO: we haven't aquired a lock yet on any of these databases so this
	// could open up small race conditions.
	selectedDatabase := &model.MultitenantDatabase{}
	for _, multitenantDatabase := range multitenantDatabases {
		if multitenantDatabase.Installations.Count() >= selectedDatabase.Installations.Count() {
			selectedDatabase = multitenantDatabase
		}
	}

	unlockFn, err := d.lockMultitenantDatabase(selectedDatabase.ID, store, logger)
	if err != nil {
		return nil, nil, errors.Wrap(err, "failed to lock selected database")
	}
	// Now that we have selected one and have a lock, ensure the database hasn't
	// been updated.
	selectedDatabase, err = store.GetMultitenantDatabase(selectedDatabase.ID)
	if err != nil {
		unlockFn()
		return nil, nil, errors.Wrap(err, "failed to refresh multitenant database after lock")
	}

	// Finish assigning the installation.
	selectedDatabase.Installations.Add(d.installationID)
	err = store.UpdateMultitenantDatabase(selectedDatabase)
	if err != nil {
		unlockFn()
		return nil, nil, errors.Wrap(err, "failed save installation to selected database")
	}

	return selectedDatabase, unlockFn, nil
}

func (d *RDSMultitenantDatabase) getMultitenantDatabasesFromResourceTags(vpcID string, store model.InstallationDatabaseStoreInterface, logger log.FieldLogger) ([]*model.MultitenantDatabase, error) {
	databaseType := d.DatabaseTypeTagValue()

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
				Key:    aws.String(trimTagPrefix(VpcIDTagKey)),
				Values: []*string{&vpcID},
			},
			{
				Key:    aws.String(trimTagPrefix(CloudInstallationDatabaseTagKey)),
				Values: []*string{&databaseType},
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
			return nil, errors.Wrap(err, "failed to get a multitenant RDS cluster ID from AWS resource tags")
		}

		if rdsClusterID != nil {
			multitenantDatabase := model.MultitenantDatabase{
				ID:           *rdsClusterID,
				VpcID:        vpcID,
				DatabaseType: d.databaseType,
			}

			ready, err := d.isRDSClusterEndpointsReady(*rdsClusterID)
			if err != nil {
				logger.WithError(err).Errorf("Failed to check RDS cluster status. Skipping RDS cluster ID %s", *rdsClusterID)
				continue
			}
			if !ready {
				continue
			}

			err = store.CreateMultitenantDatabase(&multitenantDatabase)
			if err != nil {
				logger.WithError(err).Errorf("Failed to create a multitenant database. Skipping RDS cluster ID %s", *rdsClusterID)
				continue
			}

			logger.Debugf("Added multitenant database %s to the datastore", multitenantDatabase.ID)

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

			if counter < d.MaxSupportedDatabases() {
				return rdsClusterID, nil
			}
		}
	}

	return nil, nil
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
		return nil, errors.Wrap(err, "RDS secret failed validation")
	}

	b, err := json.Marshal(&rdsSecretPayload)
	if err != nil {
		return nil, errors.Wrap(err, "failed to marshal secrets manager payload")
	}

	_, err = d.client.Service().secretsManager.CreateSecret(&secretsmanager.CreateSecretInput{
		Name:         aws.String(secretName),
		Description:  aws.String(description),
		Tags:         tags,
		SecretString: aws.String(string(b)),
	})
	if err != nil {
		return nil, errors.Wrap(err, "failed to create secret")
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

// TODO: refactor the two separate locking flows for MT databases into one.
func (d *RDSMultitenantDatabase) lockMultitenantDatabase(multitenantDatabaseID string, store model.InstallationDatabaseStoreInterface, logger log.FieldLogger) (func(), error) {
	locked, err := store.LockMultitenantDatabase(multitenantDatabaseID, d.instanceID)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to lock multitenant database %s", multitenantDatabaseID)
	}
	if !locked {
		return nil, errors.Errorf("failed to acquire lock for multitenant database %s", multitenantDatabaseID)
	}

	unlockFN := func() {
		unlocked, err := store.UnlockMultitenantDatabase(multitenantDatabaseID, d.instanceID, true)
		if err != nil {
			logger.WithError(err).Error("failed to unlock multitenant database")
		}
		if !unlocked {
			logger.Warn("failed to release lock for multitenant database")
		}
	}

	return unlockFN, nil
}

func (d *RDSMultitenantDatabase) validateMultitenantDatabaseInstallations(multitenantDatabaseID string, installations model.MultitenantDatabaseInstallations, store model.InstallationDatabaseStoreInterface) error {
	multitenantDatabase, err := store.GetMultitenantDatabase(multitenantDatabaseID)
	if err != nil {
		return errors.Wrap(err, "failed to query for multitenant database")
	}
	if multitenantDatabase == nil {
		return errors.Errorf("failed to find a multitenant database with ID %s", multitenantDatabaseID)
	}

	if installations.Count() != multitenantDatabase.Installations.Count() {
		return errors.Errorf("supplied %d installations, but multitenant database ID %s has %d", installations.Count(), multitenantDatabase.ID, multitenantDatabase.Installations.Count())
	}

	for _, installation := range installations {
		if !multitenantDatabase.Installations.Contains(installation) {
			return errors.Errorf("failed to find installation ID %s in the multitenant database ID %s", installation, multitenantDatabase.ID)
		}
	}

	return nil
}

// removeInstallationFromMultitenantDatabase performs the work necessary to
// remove a single installation database from a multitenant RDS cluster.
func (d *RDSMultitenantDatabase) removeInstallationFromMultitenantDatabase(database *model.MultitenantDatabase, store model.InstallationDatabaseStoreInterface, logger log.FieldLogger) error {
	rdsCluster, err := d.describeRDSCluster(database.ID)
	if err != nil {
		return errors.Wrap(err, "failed to describe multitenant database")
	}

	logger = logger.WithField("rds-cluster-id", *rdsCluster.DBClusterIdentifier)

	// TODO: split this up.
	err = d.dropDatabaseAndDeleteSecret(database.ID, *rdsCluster.Endpoint, store, logger)
	if err != nil {
		return errors.Wrap(err, "failed to drop database or delete secret")
	}

	numInstallations := database.Installations.Count()

	err = d.updateCounterTag(rdsCluster.DBClusterArn, numInstallations-1)
	if err != nil {
		return errors.Wrap(err, "failed to update counter tag")
	}

	// We need to update the tag before removing the installation ID from the
	// datastore. However, if this operation fails, tag:counter in RDS needs to
	// return to the original value.
	// TODO: improve handling this.
	database.Installations.Remove(d.installationID)
	err = store.UpdateMultitenantDatabase(database)
	if err != nil {
		logger.WithError(err).Warnf("Failed to remove multitenant database from datastore. Rolling tag:counter value back to %d", numInstallations)
		updateTagErr := d.updateCounterTag(rdsCluster.DBClusterArn, numInstallations)
		if updateTagErr != nil {
			logger.WithError(err).Errorf("Failed to roll back tag:counter. Value is still %d", numInstallations-1)
		}
		return errors.Wrapf(err, "failed to remove installation ID %s from multitenant datastore", d.installationID)
	}

	return nil
}

func (d *RDSMultitenantDatabase) dropDatabaseAndDeleteSecret(rdsClusterID, rdsClusterendpoint string, store model.InstallationDatabaseStoreInterface, logger log.FieldLogger) error {
	databaseName := MattermostRDSDatabaseName(d.installationID)

	masterSecretValue, err := d.client.Service().secretsManager.GetSecretValue(&secretsmanager.GetSecretValueInput{
		SecretId: aws.String(rdsClusterID),
	})
	if err != nil {
		return errors.Wrapf(err, "failed to get master secret by ID %s", rdsClusterID)
	}

	close, err := d.connectRDSCluster(rdsClusterendpoint, DefaultMattermostDatabaseUsername, *masterSecretValue.SecretString)
	if err != nil {
		return errors.Wrapf(err, "failed to connect to multitenant RDS cluster ID %s", rdsClusterID)
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
				Key:   aws.String(trimTagPrefix(VpcIDTagKey)),
				Value: VpcID,
			},
			{
				Key:   aws.String(trimTagPrefix(DefaultMattermostInstallationIDTagKey)),
				Value: aws.String(d.installationID),
			},
		}

		// PostgreSQL username can't start with integers, so prepend something
		// valid just in case. Name can't be longer than 32 characters for MySQL
		// databases though.
		username := fmt.Sprintf("user_%s", d.installationID)
		installationSecret, err = d.createInstallationSecret(installationSecretName, username, description, tags)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to create a multitenant RDS database secret %s", installationSecretName)
		}
	}

	return installationSecret, nil
}

func (d *RDSMultitenantDatabase) isRDSClusterEndpointsReady(rdsClusterID string) (bool, error) {
	output, err := d.client.service.rds.DescribeDBClusterEndpoints(&rds.DescribeDBClusterEndpointsInput{
		DBClusterIdentifier: aws.String(rdsClusterID),
	})
	if err != nil {
		return false, errors.Wrap(err, "failed to describe RDS cluster endpoint")
	}

	for _, endpoint := range output.DBClusterEndpoints {
		if *endpoint.Status != DefaultRDSStatusAvailable {
			return false, nil
		}
	}

	return true, nil
}

func (d *RDSMultitenantDatabase) runProvisionSQLCommands(installationDatabaseName, vpcID string, rdsCluster *rds.DBCluster, logger log.FieldLogger) error {
	rdsID := *rdsCluster.DBClusterIdentifier

	masterSecretValue, err := d.client.Service().secretsManager.GetSecretValue(&secretsmanager.GetSecretValueInput{
		SecretId: rdsCluster.DBClusterIdentifier,
	})
	if err != nil {
		return errors.Wrapf(err, "failed to find the master secret for the multitenant RDS cluster %s", rdsID)
	}

	close, err := d.connectRDSCluster(*rdsCluster.Endpoint, DefaultMattermostDatabaseUsername, *masterSecretValue.SecretString)
	if err != nil {
		return errors.Wrapf(err, "failed to connect to the multitenant RDS cluster %s", rdsID)
	}
	defer close(logger)

	ctx, cancel := context.WithDeadline(context.Background(), time.Now().Add(DefaultMySQLContextTimeSeconds*time.Second))
	defer cancel()

	err = d.ensureDatabaseIsCreated(ctx, installationDatabaseName)
	if err != nil {
		return errors.Wrapf(err, "failed to create schema in multitenant RDS cluster %s", rdsID)
	}

	installationSecret, err := d.ensureMultitenantDatabaseSecretIsCreated(rdsCluster.DBClusterIdentifier, &vpcID)
	if err != nil {
		return errors.Wrap(err, "failed to get a secret for installation")
	}

	err = d.ensureDatabaseUserIsCreated(ctx, installationSecret.MasterUsername, installationSecret.MasterPassword)
	if err != nil {
		return errors.Wrap(err, "failed to create Mattermost database user")
	}

	err = d.ensureDatabaseUserHasFullPermissions(ctx, installationDatabaseName, installationSecret.MasterUsername)
	if err != nil {
		return errors.Wrap(err, "failed to grant permissions to Mattermost database user")
	}

	return nil
}

func (d *RDSMultitenantDatabase) connectRDSCluster(endpoint, username, password string) (func(logger log.FieldLogger), error) {
	if d.db == nil {
		var db SQLDatabaseManager
		var err error
		switch d.databaseType {
		case model.DatabaseEngineTypeMySQL:
			db, err = sql.Open("mysql", RDSMySQLConnString(rdsMySQLDefaultSchema, endpoint, username, password))
			if err != nil {
				return nil, errors.Wrapf(err, "failed to connect multitenant RDS cluster endpoint %s", endpoint)
			}
		case model.DatabaseEngineTypePostgres:
			db, err = sql.Open("postgres", RDSPostgresConnString(rdsPostgresDefaultSchema, endpoint, username, password))
			if err != nil {
				return nil, errors.Wrap(err, "failed to connect to postgres database")
			}
		}

		d.db = db
	}

	closeFunc := func(logger log.FieldLogger) {
		err := d.db.Close()
		if err != nil {
			logger.WithError(err).Errorf("Failed to close the connection with multitenant RDS cluster endpoint %s", endpoint)
		}
	}

	return closeFunc, nil
}

func (d *RDSMultitenantDatabase) ensureDatabaseIsCreated(ctx context.Context, databaseName string) error {
	if d.databaseType == model.DatabaseEngineTypeMySQL {
		// Query placeholders don't seem to work with argument database.
		// See https://github.com/mattermost/mattermost-cloud/pull/209#discussion_r422533477
		query := fmt.Sprintf("CREATE DATABASE IF NOT EXISTS %s CHARACTER SET ?", databaseName)
		_, err := d.db.QueryContext(ctx, query, "utf8mb4")
		if err != nil {
			return errors.Wrap(err, "failed to run create database SQL command")
		}
	} else {
		query := fmt.Sprintf(`SELECT datname FROM pg_catalog.pg_database WHERE lower(datname) = lower('%s');`, databaseName)
		rows, err := d.db.QueryContext(ctx, query)
		if err != nil {
			return errors.Wrap(err, "failed to run create database SQL command")
		}
		if rows.Next() {
			return nil
		}

		query = fmt.Sprintf(`CREATE DATABASE %s`, databaseName)
		_, err = d.db.QueryContext(ctx, query)
		if err != nil {
			return errors.Wrap(err, "failed to run create database SQL command")
		}
	}

	return nil
}

func (d *RDSMultitenantDatabase) ensureDatabaseUserIsCreated(ctx context.Context, username, password string) error {
	if d.databaseType == model.DatabaseEngineTypeMySQL {
		_, err := d.db.QueryContext(ctx, "CREATE USER IF NOT EXISTS ?@? IDENTIFIED BY ? REQUIRE SSL", username, "%", password)
		if err != nil {
			return errors.Wrap(err, "failed to run create user SQL command")
		}
	} else {
		query := fmt.Sprintf("SELECT 1 FROM pg_roles WHERE rolname='%s'", username)
		rows, err := d.db.QueryContext(ctx, query)
		if err != nil {
			return errors.Wrap(err, "failed to run original user cleanup SQL command")
		}
		if rows.Next() {
			return nil
		}

		// Due to not being able use parameters here, we have to do something
		// a bit gross to ensure the password is not leaked into logs.
		// https://github.com/lib/pq/issues/694#issuecomment-356180769
		query = fmt.Sprintf("CREATE USER %s WITH PASSWORD '%s'", username, password)
		_, err = d.db.QueryContext(ctx, query)
		if err != nil {
			return errors.New("failed to run create user SQL command: error suppressed")
		}
	}

	return nil
}

func (d *RDSMultitenantDatabase) ensureDatabaseUserHasFullPermissions(ctx context.Context, databaseName, username string) error {
	if d.databaseType == model.DatabaseEngineTypeMySQL {
		// Query placeholders don't seem to work with argument database.
		// See https://github.com/mattermost/mattermost-cloud/pull/209#discussion_r422533477
		query := fmt.Sprintf("GRANT ALL PRIVILEGES ON %s.* TO ?@?", databaseName)
		_, err := d.db.QueryContext(ctx, query, username, "%")
		if err != nil {
			return errors.Wrap(err, "failed to run privilege grant SQL command")
		}
	} else {
		query := fmt.Sprintf("GRANT ALL PRIVILEGES ON DATABASE %s TO %s", databaseName, username)
		_, err := d.db.QueryContext(ctx, query)
		if err != nil {
			return errors.Wrap(err, "failed to run privilege grant SQL command")
		}
	}

	return nil
}

func (d *RDSMultitenantDatabase) dropDatabaseIfExists(ctx context.Context, databaseName string) error {
	// Query placeholders don't seem to work with argument database.
	// See https://github.com/mattermost/mattermost-cloud/pull/209#discussion_r422533477
	query := fmt.Sprintf("DROP DATABASE IF EXISTS %s", databaseName)

	_, err := d.db.QueryContext(ctx, query)
	if err != nil {
		return errors.Wrap(err, "failed to run drop database SQL command")
	}

	return nil
}
