// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.
//

package aws

import (
	"context"
	"crypto/md5"
	"database/sql"
	"fmt"
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
	// Database drivers
	_ "github.com/go-sql-driver/mysql"
	_ "github.com/lib/pq"
)

// RDSMultitenantPGBouncerDatabase is a database backed by RDS that supports
// multi-tenancy and pooled connections.
type RDSMultitenantPGBouncerDatabase struct {
	databaseType   string
	installationID string
	instanceID     string
	db             SQLDatabaseManager
	client         *Client
}

// NewRDSMultitenantPGBouncerDatabase returns a new instance of
// RDSMultitenantPGBouncerDatabase that implements database interface.
func NewRDSMultitenantPGBouncerDatabase(databaseType, instanceID, installationID string, client *Client) *RDSMultitenantPGBouncerDatabase {
	return &RDSMultitenantPGBouncerDatabase{
		databaseType:   databaseType,
		instanceID:     instanceID,
		installationID: installationID,
		client:         client,
	}
}

// IsValid returns if the given RDSMultitenantDatabase configuration is valid.
func (d *RDSMultitenantPGBouncerDatabase) IsValid() error {
	if len(d.installationID) == 0 {
		return errors.New("installation ID is not set")
	}

	switch d.databaseType {
	case model.DatabaseEngineTypePostgresProxy:
	default:
		return errors.Errorf("invalid pgbouncer database type %s", d.databaseType)
	}

	return nil
}

// DatabaseTypeTagValue returns the tag value used for filtering RDS cluster
// resources based on database type.
func (d *RDSMultitenantPGBouncerDatabase) DatabaseTypeTagValue() string {
	return DatabaseTypePostgresSQLAurora
}

// MaxSupportedDatabases returns the maximum number of databases supported on
// one RDS cluster for this database type.
func (d *RDSMultitenantPGBouncerDatabase) MaxSupportedDatabases() int {
	return DefaultRDSMultitenantPGBouncerDatabasePostgresCountLimit
}

// Provision claims a multitenant RDS cluster and creates a database schema for
// the installation.
func (d *RDSMultitenantPGBouncerDatabase) Provision(store model.InstallationDatabaseStoreInterface, logger log.FieldLogger) error {
	err := d.IsValid()
	if err != nil {
		return errors.Wrap(err, "pgbouncer database configuration is invalid")
	}

	logger = logger.WithField("database-type", d.databaseType)
	logger.Info("Provisioning Multitenant AWS RDS PGBouncer database")

	vpc, err := getVPCForInstallation(d.installationID, store, d.client)
	if err != nil {
		return errors.Wrap(err, "failed to find cluster installation VPC")
	}

	database, unlockFn, err := d.getAndLockAssignedProxiedDatabase(store, logger)
	if err != nil {
		return errors.Wrap(err, "failed to get and lock assigned database")
	}
	if database == nil {
		logger.Debug("Assigning installation to multitenant proxy database")
		database, unlockFn, err = d.assignInstallationToProxiedDatabaseAndLock(*vpc.VpcId, store, logger)
		if err != nil {
			return errors.Wrap(err, "failed to assign installation to a multitenant proxy database")
		}
	}
	defer unlockFn()

	databaseName := database.SharedLogicalDatabaseMappings.GetLogicalDatabaseName(d.installationID)
	logger = logger.WithFields(log.Fields{
		"assigned-database": database.ID,
		"logical-database":  databaseName,
	})

	rdsCluster, err := describeRDSCluster(database.ID, d.client)
	if err != nil {
		return errors.Wrapf(err, "failed to describe the multitenant RDS cluster ID %s", database.ID)
	}
	if *rdsCluster.Status != DefaultRDSStatusAvailable {
		return errors.Errorf("multitenant RDS cluster ID %s is not available (status: %s)", database.ID, *rdsCluster.Status)
	}

	rdsID := *rdsCluster.DBClusterIdentifier
	logger = logger.WithField("rds-cluster-id", rdsID)

	if database.State == model.DatabaseStateProvisioningRequested {
		err = d.provisionPGBouncerDatabase(*vpc.VpcId, rdsCluster, logger)
		if err != nil {
			return errors.Wrap(err, "failed to provision pgbouncer database")
		}

		database.State = model.DatabaseStateStable
		err = store.UpdateMultitenantDatabase(database)
		if err != nil {
			return errors.Wrap(err, "failed to update state on provisioned pgbouncer database")
		}
	}

	err = d.ensureLogicalDatabaseExists(databaseName, rdsCluster, logger)
	if err != nil {
		return errors.Wrap(err, "failed to ensure the logical database is created")
	}

	err = d.ensureLogicalDatabaseSetup(databaseName, *vpc.VpcId, rdsCluster, logger)
	if err != nil {
		return errors.Wrap(err, "failed to ensure the logical database is setup")
	}

	err = updateCounterTagWithCurrentWeight(database, rdsCluster, store, d.client, logger)
	if err != nil {
		return errors.Wrap(err, "failed to update counter tag with current weight")
	}

	logger.Infof("Installation %s assigned to pgbouncer multitenant database", d.installationID)

	return nil
}

// This helper method finds a multitenant RDS cluster that is ready for receiving a database installation. The lookup
// for multitenant databases will happen in order:
//	1. fetch a multitenant database by installation ID.
//	2. fetch all multitenant databases in the store which are under the max number of installations limit.
//	3. fetch all multitenant databases in the RDS cluster that are under the max number of installations limit.
func (d *RDSMultitenantPGBouncerDatabase) assignInstallationToProxiedDatabaseAndLock(vpcID string, store model.InstallationDatabaseStoreInterface, logger log.FieldLogger) (*model.MultitenantDatabase, func(), error) {
	multitenantDatabases, err := store.GetMultitenantDatabases(&model.MultitenantDatabaseFilter{
		DatabaseType:          d.databaseType,
		MaxInstallationsLimit: d.MaxSupportedDatabases(),
		VpcID:                 vpcID,
		Paging:                model.AllPagesNotDeleted(),
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
		return nil, nil, errors.New("no multitenant proxy databases are currently available for new installations")
	}

	// We want to be smart about how we assign the installation to a database.
	// Find the database with the most installations on it to keep utilization
	// as close to maximim efficiency as possible.
	selectedDatabase := &model.MultitenantDatabase{}
	for _, multitenantDatabase := range multitenantDatabases {
		if multitenantDatabase.Installations.Count() >= selectedDatabase.Installations.Count() {
			selectedDatabase = multitenantDatabase
		}
	}

	unlockFn, err := lockMultitenantDatabase(selectedDatabase.ID, d.instanceID, store, logger)
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
	selectedDatabase.AddInstallationToLogicalDatabaseMapping(d.installationID)
	err = store.UpdateMultitenantDatabase(selectedDatabase)
	if err != nil {
		unlockFn()
		return nil, nil, errors.Wrap(err, "failed to save installation to selected database")
	}

	return selectedDatabase, unlockFn, nil
}

func (d *RDSMultitenantPGBouncerDatabase) getMultitenantDatabasesFromResourceTags(vpcID string, store model.InstallationDatabaseStoreInterface, logger log.FieldLogger) ([]*model.MultitenantDatabase, error) {
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
				Values: []*string{aws.String(DefaultRDSMultitenantDatabaseDBProxyTypeTagValue)},
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

		rdsClusterID, err := getRDSClusterIDFromResourceTags(d.MaxSupportedDatabases(), resource.Tags)
		if err != nil {
			return nil, errors.Wrap(err, "failed to get a multitenant RDS cluster ID from AWS resource tags")
		}
		if rdsClusterID == nil {
			continue
		}

		ready, err := isRDSClusterEndpointsReady(*rdsClusterID, d.client)
		if err != nil {
			logger.WithError(err).Errorf("Failed to check RDS cluster status. Skipping RDS cluster ID %s", *rdsClusterID)
			continue
		}
		if !ready {
			continue
		}

		rdsCluster, err := describeRDSCluster(*rdsClusterID, d.client)
		if err != nil {
			logger.WithError(err).Errorf("Failed to describe the multitenant RDS cluster ID %s", *rdsClusterID)
			continue
		}

		multitenantDatabase := model.MultitenantDatabase{
			ID:                                 *rdsClusterID,
			VpcID:                              vpcID,
			DatabaseType:                       d.databaseType,
			State:                              model.DatabaseStateProvisioningRequested,
			WriterEndpoint:                     *rdsCluster.Endpoint,
			ReaderEndpoint:                     *rdsCluster.ReaderEndpoint,
			MaxInstallationsPerLogicalDatabase: 10,
		}

		err = store.CreateMultitenantDatabase(&multitenantDatabase)
		if err != nil {
			logger.WithError(err).Errorf("Failed to create a multitenant database. Skipping RDS cluster ID %s", *rdsClusterID)
			continue
		}

		logger.Debugf("Added multitenant database %s to the datastore", multitenantDatabase.ID)

		multitenantDatabases = append(multitenantDatabases, &multitenantDatabase)
	}

	return multitenantDatabases, nil
}

// getAssignedMultitenantDatabaseResources returns the assigned multitenant
// database if there is one or nil if there is not. An error is returned if the
// installation is assigned to more than one database.
func (d *RDSMultitenantPGBouncerDatabase) getAndLockAssignedProxiedDatabase(store model.InstallationDatabaseStoreInterface, logger log.FieldLogger) (*model.MultitenantDatabase, func(), error) {
	multitenantDatabases, err := store.GetMultitenantDatabases(&model.MultitenantDatabaseFilter{
		DatabaseType:          d.databaseType,
		InstallationID:        d.installationID,
		MaxInstallationsLimit: model.NoInstallationsLimit,
		Paging:                model.AllPagesNotDeleted(),
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

	unlockFn, err := lockMultitenantDatabase(multitenantDatabases[0].ID, d.instanceID, store, logger)
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

func (d *RDSMultitenantPGBouncerDatabase) provisionPGBouncerDatabase(vpcID string, rdsCluster *rds.DBCluster, logger log.FieldLogger) error {
	rdsID := *rdsCluster.DBClusterIdentifier

	logger.Infof("Provisioning PGBouncer database %s", rdsID)

	masterSecretValue, err := d.client.Service().secretsManager.GetSecretValue(&secretsmanager.GetSecretValueInput{
		SecretId: rdsCluster.DBClusterIdentifier,
	})
	if err != nil {
		return errors.Wrapf(err, "failed to find the master secret for the multitenant proxy cluster %s", rdsID)
	}

	authUserSecret, err := d.ensurePGBouncerAuthUserSecretIsCreated(&rdsID, &vpcID)
	if err != nil {
		return errors.Wrapf(err, "failed to ensure the pgbouncer auth user secret was created for %s", rdsID)
	}

	close, err := d.connectRDSCluster(rdsPostgresDefaultSchema, *rdsCluster.Endpoint, DefaultMattermostDatabaseUsername, *masterSecretValue.SecretString)
	if err != nil {
		return errors.Wrapf(err, "failed to connect to the multitenant proxy cluster %s", rdsID)
	}
	defer close(logger)

	ctx, cancel := context.WithDeadline(context.Background(), time.Now().Add(DefaultMySQLContextTimeSeconds*time.Second))
	defer cancel()

	query := fmt.Sprintf("SELECT 1 FROM pg_roles WHERE rolname='%s'", authUserSecret.MasterUsername)
	rows, err := d.db.QueryContext(ctx, query)
	if err != nil {
		return errors.Wrap(err, "failed to run database user check SQL command")
	}
	if rows.Next() {
		// User already exists.
		return nil
	}

	query = fmt.Sprintf("CREATE USER %s WITH PASSWORD '%s'", authUserSecret.MasterUsername, authUserSecret.MasterPassword)
	_, err = d.db.QueryContext(ctx, query)
	if err != nil {
		return errors.New("failed to run create pgbouncer auth user SQL command: error suppressed")
	}

	return nil
}

func (d *RDSMultitenantPGBouncerDatabase) ensurePGBouncerAuthUserSecretIsCreated(rdsClusterID, VpcID *string) (*RDSSecret, error) {
	authUserSecretName := PGBouncerAuthUserSecretName(*VpcID)

	secretValue, err := d.client.Service().secretsManager.GetSecretValue(&secretsmanager.GetSecretValueInput{
		SecretId: aws.String(authUserSecretName),
	})
	if err != nil && !IsErrorCode(err, secretsmanager.ErrCodeResourceNotFoundException) {
		return nil, errors.Wrapf(err, "failed to get pgbouncer auth user secret %s", authUserSecretName)
	}

	var secret *RDSSecret
	if secretValue != nil && secretValue.SecretString != nil {
		secret, err = unmarshalSecretPayload(*secretValue.SecretString)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to unmarshal pgbouncer auth user secret %s", authUserSecretName)
		}
	} else {
		description := RDSMultitenantPGBouncerClusterSecretDescription(*VpcID)
		tags := []*secretsmanager.Tag{
			{
				Key:   aws.String(trimTagPrefix(DefaultRDSMultitenantDatabaseIDTagKey)),
				Value: rdsClusterID,
			},
			{
				Key:   aws.String(trimTagPrefix(VpcIDTagKey)),
				Value: VpcID,
			},
		}

		secret, err = createDatabaseUserSecret(authUserSecretName, DefaultPGBouncerAuthUsername, description, tags, d.client)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to create a multitenant RDS database secret %s", authUserSecretName)
		}
	}

	return secret, nil
}

func (d *RDSMultitenantPGBouncerDatabase) ensureLogicalDatabaseExists(databaseName string, rdsCluster *rds.DBCluster, logger log.FieldLogger) error {
	rdsID := *rdsCluster.DBClusterIdentifier

	masterSecretValue, err := d.client.Service().secretsManager.GetSecretValue(&secretsmanager.GetSecretValueInput{
		SecretId: rdsCluster.DBClusterIdentifier,
	})
	if err != nil {
		return errors.Wrapf(err, "failed to find the master secret for the multitenant proxy cluster %s", rdsID)
	}

	close, err := d.connectRDSCluster(rdsPostgresDefaultSchema, *rdsCluster.Endpoint, DefaultMattermostDatabaseUsername, *masterSecretValue.SecretString)
	if err != nil {
		return errors.Wrapf(err, "failed to connect to the multitenant proxy cluster %s", rdsID)
	}
	defer close(logger)

	ctx, cancel := context.WithDeadline(context.Background(), time.Now().Add(DefaultMySQLContextTimeSeconds*time.Second))
	defer cancel()

	err = d.ensureDatabaseIsCreated(ctx, databaseName)
	if err != nil {
		return errors.Wrapf(err, "failed to create database in multitenant proxy cluster %s", rdsID)
	}

	return nil
}

func (d *RDSMultitenantPGBouncerDatabase) ensureLogicalDatabaseSetup(databaseName, vpcID string, rdsCluster *rds.DBCluster, logger log.FieldLogger) error {
	rdsID := *rdsCluster.DBClusterIdentifier

	masterSecretValue, err := d.client.Service().secretsManager.GetSecretValue(&secretsmanager.GetSecretValueInput{
		SecretId: rdsCluster.DBClusterIdentifier,
	})
	if err != nil {
		return errors.Wrapf(err, "failed to find the master secret for the multitenant proxy cluster %s", rdsID)
	}

	close, err := d.connectRDSCluster(databaseName, *rdsCluster.Endpoint, DefaultMattermostDatabaseUsername, *masterSecretValue.SecretString)
	if err != nil {
		return errors.Wrapf(err, "failed to connect to the multitenant proxy cluster %s", rdsID)
	}
	defer close(logger)

	ctx, cancel := context.WithDeadline(context.Background(), time.Now().Add(DefaultMySQLContextTimeSeconds*time.Second))
	defer cancel()

	err = d.ensurePGBouncerDatabasePrep(ctx, databaseName, DefaultPGBouncerAuthUsername)
	if err != nil {
		return errors.Wrap(err, "failed to perform pgbouncer setup")
	}

	installationSecret, err := d.ensureMultitenantDatabaseSecretIsCreated(rdsCluster.DBClusterIdentifier, &vpcID)
	if err != nil {
		return errors.Wrap(err, "failed to get a secret for installation")
	}

	err = d.ensureDatabaseUserIsCreated(ctx, installationSecret.MasterUsername, installationSecret.MasterPassword)
	if err != nil {
		return errors.Wrap(err, "failed to create Mattermost database user")
	}

	err = d.ensureInstallationUserAddedToUsersTable(ctx, installationSecret.MasterUsername, installationSecret.MasterPassword)
	if err != nil {
		return errors.Wrap(err, "failed to create Mattermost user entry for PGBouncer")
	}

	err = d.ensureMasterUserHasRole(ctx, installationSecret.MasterUsername)
	if err != nil {
		return errors.Wrap(err, "failed to ensure master user has installation role")
	}

	err = d.ensureSchemaIsCreated(ctx, installationSecret.MasterUsername)
	if err != nil {
		return errors.Wrap(err, "failed to grant permissions to Mattermost database user")
	}

	return nil
}

func (d *RDSMultitenantPGBouncerDatabase) ensureDatabaseIsCreated(ctx context.Context, databaseName string) error {
	query := fmt.Sprintf(`SELECT datname FROM pg_catalog.pg_database WHERE lower(datname) = lower('%s');`, databaseName)
	rows, err := d.db.QueryContext(ctx, query)
	if err != nil {
		return errors.Wrap(err, "failed to run database exists SQL command")
	}
	if rows.Next() {
		return nil
	}

	query = fmt.Sprintf(`CREATE DATABASE %s`, databaseName)
	_, err = d.db.QueryContext(ctx, query)
	if err != nil {
		return errors.Wrap(err, "failed to run create database SQL command")
	}

	return nil
}

func (d *RDSMultitenantPGBouncerDatabase) ensureInstallationUserAddedToUsersTable(ctx context.Context, username, password string) error {
	query := fmt.Sprintf("SELECT usename FROM pgbouncer.pgbouncer_users WHERE usename = '%s';", username)
	rows, err := d.db.QueryContext(ctx, query)
	if err != nil {
		return errors.Wrap(err, "failed to run installation pgbouncer user exists SQL command")
	}
	if rows.Next() {
		return nil
	}

	query = fmt.Sprintf(`INSERT INTO pgbouncer.pgbouncer_users (usename, passwd) VALUES ('%s', 'md5%x')`, username, md5.Sum([]byte(password+username)))
	_, err = d.db.QueryContext(ctx, query)
	if err != nil {
		return errors.Wrap(err, "failed to run create pgbouncer installation user SQL command")
	}

	return nil
}

func (d *RDSMultitenantPGBouncerDatabase) ensurePGBouncerDatabasePrep(ctx context.Context, databaseName, pgbouncerUsername string) error {
	err := d.ensureMasterUserHasRole(ctx, pgbouncerUsername)
	if err != nil {
		return errors.Wrap(err, "failed to run master user has pgbouncer role SQL command")
	}

	err = d.ensureSchemaIsCreated(ctx, pgbouncerUsername)
	if err != nil {
		return errors.Wrap(err, "failed to run pgbouncer schema is created SQL command")
	}

	query := fmt.Sprintf(`ALTER DATABASE %s SET search_path = "$user"`, databaseName)
	_, err = d.db.QueryContext(ctx, query)
	if err != nil {
		return errors.Wrap(err, "failed to run database search path set SQL command")
	}

	query = fmt.Sprintf(`
	CREATE TABLE IF NOT EXISTS %s.pgbouncer_users(
	usename NAME PRIMARY KEY,
	passwd TEXT NOT NULL
	)`, pgbouncerUsername)
	_, err = d.db.QueryContext(ctx, query)
	if err != nil {
		return errors.Wrap(err, "failed to run pgbouncer user table creation SQL command")
	}

	query = fmt.Sprintf(`
	CREATE OR REPLACE FUNCTION %s.get_auth(uname TEXT) RETURNS TABLE (usename name, passwd text) as
	$$
	  SELECT usename, passwd FROM pgbouncer.pgbouncer_users WHERE usename=$1;
	$$
	LANGUAGE sql SECURITY DEFINER;`,
		pgbouncerUsername)
	_, err = d.db.QueryContext(ctx, query)
	if err != nil {
		return errors.Wrap(err, "failed to run auth user lookup query SQL command")
	}

	return nil
}

func (d *RDSMultitenantPGBouncerDatabase) ensureDatabaseUserIsCreated(ctx context.Context, username, password string) error {
	query := fmt.Sprintf("SELECT 1 FROM pg_roles WHERE rolname='%s'", username)
	rows, err := d.db.QueryContext(ctx, query)
	if err != nil {
		return errors.Wrap(err, "failed to run database user check SQL command")
	}
	if rows.Next() {
		return nil
	}

	query = fmt.Sprintf("CREATE USER %s WITH PASSWORD '%s'", username, password)
	_, err = d.db.QueryContext(ctx, query)
	if err != nil {
		return errors.New("failed to run create user SQL command: error suppressed")
	}

	return nil
}

func (d *RDSMultitenantPGBouncerDatabase) ensureMasterUserHasRole(ctx context.Context, role string) error {
	query := fmt.Sprintf("GRANT %s TO %s;", role, DefaultMattermostDatabaseUsername)
	_, err := d.db.QueryContext(ctx, query)
	if err != nil {
		return errors.Wrap(err, "failed to run grant role SQL command")
	}

	return nil
}

func (d *RDSMultitenantPGBouncerDatabase) ensureSchemaIsCreated(ctx context.Context, username string) error {
	query := fmt.Sprintf("CREATE SCHEMA IF NOT EXISTS AUTHORIZATION %s", username)
	_, err := d.db.QueryContext(ctx, query)
	if err != nil {
		return errors.Wrap(err, "failed to run create schema SQL command")
	}

	return nil
}

func (d *RDSMultitenantPGBouncerDatabase) connectRDSCluster(database, endpoint, username, password string) (func(logger log.FieldLogger), error) {
	db, err := sql.Open("postgres", RDSPostgresConnString(database, endpoint, username, password))
	if err != nil {
		return nil, errors.Wrap(err, "failed to connect to postgres database")
	}

	d.db = db

	closeFunc := func(logger log.FieldLogger) {
		err := d.db.Close()
		if err != nil {
			logger.WithError(err).Errorf("Failed to close the connection with multitenant RDS cluster endpoint %s", endpoint)
		}
	}

	return closeFunc, nil
}

func (d *RDSMultitenantPGBouncerDatabase) ensureMultitenantDatabaseSecretIsCreated(rdsClusterID, VpcID *string) (*RDSSecret, error) {
	installationSecretName := RDSMultitenantPGBouncerSecretName(d.installationID)

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

		username := MattermostPGBouncerDatabaseUsername(d.installationID)
		installationSecret, err = createDatabaseUserSecret(installationSecretName, username, description, tags, d.client)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to create a multitenant RDS database secret %s", installationSecretName)
		}
	}

	return installationSecret, nil
}

// GenerateDatabaseSecret creates the k8s database spec and secret for
// accessing a single schema inside a RDS multitenant cluster with a PGBouncer
// proxy.
func (d *RDSMultitenantPGBouncerDatabase) GenerateDatabaseSecret(store model.InstallationDatabaseStoreInterface, logger log.FieldLogger) (*corev1.Secret, error) {
	err := d.IsValid()
	if err != nil {
		return nil, errors.Wrap(err, "pgbouncer database configuration is invalid")
	}

	multitenantDatabase, err := store.GetMultitenantDatabaseForInstallationID(d.installationID)
	if err != nil {
		return nil, errors.Wrap(err, "failed to query for the multitenant database")
	}

	unlock, err := lockMultitenantDatabase(multitenantDatabase.ID, d.instanceID, store, logger)
	if err != nil {
		return nil, errors.Wrap(err, "failed to lock multitenant database")
	}
	defer unlock()

	installationDatabaseName := multitenantDatabase.SharedLogicalDatabaseMappings.GetLogicalDatabaseName(d.installationID)
	logger = logger.WithFields(log.Fields{
		"multitenant-rds-database": installationDatabaseName,
		"database-type":            d.databaseType,
	})

	installationSecretName := RDSMultitenantPGBouncerSecretName(d.installationID)

	result, err := d.client.Service().secretsManager.GetSecretValue(&secretsmanager.GetSecretValueInput{
		SecretId: &installationSecretName,
	})
	if err != nil {
		return nil, errors.Wrap(err, "failed to get secret value for database")
	}

	installationSecret, err := unmarshalSecretPayload(*result.SecretString)
	if err != nil {
		return nil, errors.Wrap(err, "failed to unmarshal secret payload")
	}

	databaseConnectionString, databaseReadReplicasString :=
		MattermostPostgresPGBouncerConnStrings(
			installationSecret.MasterUsername,
			installationSecret.MasterPassword,
			installationDatabaseName,
		)
	databaseConnectionCheck := databaseConnectionString

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

	logger.Debug("AWS RDS multitenant PGBouncer database configuration generated for cluster installation")

	return databaseSecret, nil
}

// Teardown removes all AWS resources related to a RDS multitenant database.
func (d *RDSMultitenantPGBouncerDatabase) Teardown(store model.InstallationDatabaseStoreInterface, keepData bool, logger log.FieldLogger) error {
	logger.Info("Tearing down RDS multitenant PGBouncer database")

	err := d.IsValid()
	if err != nil {
		return errors.Wrap(err, "multitenant database configuration is invalid")
	}

	if keepData {
		logger.Warn("Keepdata is set to true on this server, but this is not yet supported for RDS multitenant PGBouncer databases")
	}

	database, unlockFn, err := d.getAndLockAssignedProxiedDatabase(store, logger)
	if err != nil {
		return errors.Wrap(err, "failed to get assigned multitenant database")
	}
	if database != nil {
		defer unlockFn()
		err = d.removeInstallationPGBouncerResources(database, store, logger)
		if err != nil {
			return errors.Wrap(err, "failed to remove installation database")
		}
	} else {
		logger.Debug("No multitenant databases found for this installation; skipping...")
	}

	logger.Info("Multitenant RDS PGBouncer database teardown complete")

	return nil
}

// removeInstallationFromPGBouncerDatabase performs the work necessary to
// remove a single installation schema from a multitenant PGBouncer RDS cluster.
func (d *RDSMultitenantPGBouncerDatabase) removeInstallationPGBouncerResources(database *model.MultitenantDatabase, store model.InstallationDatabaseStoreInterface, logger log.FieldLogger) error {
	rdsCluster, err := describeRDSCluster(database.ID, d.client)
	if err != nil {
		return errors.Wrap(err, "failed to describe multitenant database")
	}

	logger = logger.WithField("rds-cluster-id", *rdsCluster.DBClusterIdentifier)

	err = ensureSecretDeleted(RDSMultitenantSecretName(d.installationID), d.client)
	if err != nil {
		return errors.Wrap(err, "failed to delete multitenant database secret")
	}

	databaseName := database.SharedLogicalDatabaseMappings.GetLogicalDatabaseName(d.installationID)
	username := MattermostPGBouncerDatabaseUsername(d.installationID)

	err = d.cleanupDatabase(*rdsCluster.DBClusterIdentifier, *rdsCluster.Endpoint, databaseName, username, logger)
	if err != nil {
		return errors.Wrap(err, "failed to cleanup pgbouncer database")
	}

	database.Installations.Remove(d.installationID)
	database.SharedLogicalDatabaseMappings.RemoveInstallation(d.installationID)
	err = updateCounterTagWithCurrentWeight(database, rdsCluster, store, d.client, logger)
	if err != nil {
		return errors.Wrap(err, "failed to update counter tag with current weight")
	}

	err = store.UpdateMultitenantDatabase(database)
	if err != nil {
		return errors.Wrapf(err, "failed to remove installation ID %s from multitenant datastore", d.installationID)
	}

	return nil
}

func (d *RDSMultitenantPGBouncerDatabase) cleanupDatabase(rdsClusterID, rdsClusterendpoint, databaseName, installationUsername string, logger log.FieldLogger) error {
	masterSecretValue, err := d.client.Service().secretsManager.GetSecretValue(&secretsmanager.GetSecretValueInput{
		SecretId: aws.String(rdsClusterID),
	})
	if err != nil {
		return errors.Wrapf(err, "failed to get master secret by ID %s", rdsClusterID)
	}

	close, err := d.connectRDSCluster(databaseName, rdsClusterendpoint, DefaultMattermostDatabaseUsername, *masterSecretValue.SecretString)
	if err != nil {
		return errors.Wrapf(err, "failed to connect to multitenant RDS cluster ID %s", rdsClusterID)
	}
	defer close(logger)

	ctx, cancel := context.WithDeadline(context.Background(), time.Now().Add(DefaultMySQLContextTimeSeconds*time.Second))
	defer cancel()

	err = dropSchemaIfExists(ctx, d.db, installationUsername)
	if err != nil {
		return errors.Wrap(err, "failed to delete installation schema")
	}

	err = dropUserIfExists(ctx, d.db, installationUsername)
	if err != nil {
		return errors.Wrap(err, "failed to delete installation database user")
	}

	err = d.deleteInstallationUsernameEntry(ctx, installationUsername)
	if err != nil {
		return errors.Wrap(err, "failed to remove installation database user from pgbouncer table")
	}

	return nil
}

func (d *RDSMultitenantPGBouncerDatabase) deleteInstallationUsernameEntry(ctx context.Context, username string) error {
	query := fmt.Sprintf(`DELETE FROM  pgbouncer.pgbouncer_users WHERE usename = '%s'`, username)
	_, err := d.db.QueryContext(ctx, query)
	if err != nil {
		return errors.Wrap(err, "failed to run delete installation user SQL command")
	}

	return nil
}

// Unsupported Methods

// Snapshot creates a snapshot of single RDS multitenant database.
func (d *RDSMultitenantPGBouncerDatabase) Snapshot(store model.InstallationDatabaseStoreInterface, logger log.FieldLogger) error {
	return errors.New("not implemented")
}

// MigrateOut migrating out of MySQL Operator managed database is not supported.
func (d *RDSMultitenantPGBouncerDatabase) MigrateOut(store model.InstallationDatabaseStoreInterface, dbMigration *model.InstallationDBMigrationOperation, logger log.FieldLogger) error {
	return errors.New("database migration is not supported for PGBouncer database")
}

// MigrateTo migration to MySQL Operator managed database is not supported.
func (d *RDSMultitenantPGBouncerDatabase) MigrateTo(store model.InstallationDatabaseStoreInterface, dbMigration *model.InstallationDBMigrationOperation, logger log.FieldLogger) error {
	return errors.New("database migration is not supported for PGBouncer database")
}

// TeardownMigrated tearing down migrated databases is not supported for MySQL Operator managed database.
func (d *RDSMultitenantPGBouncerDatabase) TeardownMigrated(store model.InstallationDatabaseStoreInterface, migrationOp *model.InstallationDBMigrationOperation, logger log.FieldLogger) error {
	return errors.New("tearing down migrated installations is not supported for PGBouncer database")
}

// RollbackMigration rolling back migration is not supported for MySQL Operator managed database.
func (d *RDSMultitenantPGBouncerDatabase) RollbackMigration(store model.InstallationDatabaseStoreInterface, dbMigration *model.InstallationDBMigrationOperation, logger log.FieldLogger) error {
	return errors.New("rolling back db migration is not supported for PGBouncer database")
}

// RefreshResourceMetadata ensures various operator database resource's metadata
// are correct.
func (d *RDSMultitenantPGBouncerDatabase) RefreshResourceMetadata(store model.InstallationDatabaseStoreInterface, logger log.FieldLogger) error {
	return nil
}
