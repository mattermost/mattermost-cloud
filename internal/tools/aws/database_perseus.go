// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.
//

package aws

import (
	"context"
	"encoding/base64"
	"fmt"
	"strings"
	"time"

	scrypt "github.com/agnivade/easy-scrypt"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/aws/arn"
	ec2Types "github.com/aws/aws-sdk-go-v2/service/ec2/types"
	rdsTypes "github.com/aws/aws-sdk-go-v2/service/rds/types"
	gt "github.com/aws/aws-sdk-go-v2/service/resourcegroupstaggingapi"
	gtTypes "github.com/aws/aws-sdk-go-v2/service/resourcegroupstaggingapi/types"
	"github.com/aws/aws-sdk-go-v2/service/secretsmanager"
	smTypes "github.com/aws/aws-sdk-go-v2/service/secretsmanager/types"
	"github.com/mattermost/mattermost-cloud/model"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	// Database drivers
	_ "github.com/go-sql-driver/mysql"
	_ "github.com/lib/pq"
)

// PerseusDatabase is a database backed by RDS that supports multi-tenancy and
// pooled connections via Perseus.
type PerseusDatabase struct {
	databaseType              string
	installationID            string
	instanceID                string
	db                        SQLDatabaseManager
	client                    *Client
	maxSupportedInstallations int
	disableDBCheck            bool
}

// NewPerseusDatabase returns a new instance of PerseusDatabase that implements
// the database interface.
func NewPerseusDatabase(databaseType, instanceID, installationID string, client *Client, installationsLimit int, disableDBCheck bool) *PerseusDatabase {
	return &PerseusDatabase{
		databaseType:              databaseType,
		instanceID:                instanceID,
		installationID:            installationID,
		client:                    client,
		maxSupportedInstallations: valueOrDefault(installationsLimit, DefaultRDSMultitenantPerseusDatabasePostgresCountLimit),
		disableDBCheck:            disableDBCheck,
	}
}

// Validate validates the configuration of a PerseusDatabase.
func (d *PerseusDatabase) Validate() error {
	if len(d.installationID) == 0 {
		return errors.New("installation ID is not set")
	}

	switch d.databaseType {
	case model.DatabaseEngineTypePostgresProxyPerseus:
	default:
		return errors.Errorf("invalid perseus database type %s", d.databaseType)
	}

	return nil
}

// DatabaseEngineTypeTagValue returns the tag value used for filtering RDS cluster
// resources based on database engine type.
func (d *PerseusDatabase) DatabaseEngineTypeTagValue() string {
	return DatabaseTypePostgresSQLAurora
}

// MaxSupportedDatabases returns the maximum number of databases supported on
// one RDS cluster for this database type.
func (d *PerseusDatabase) MaxSupportedDatabases() int {
	return d.maxSupportedInstallations
}

// Provision claims a multitenant RDS cluster and creates a database schema for
// the installation.
func (d *PerseusDatabase) Provision(store model.InstallationDatabaseStoreInterface, logger log.FieldLogger) error {
	logger = logger.WithField("database", model.InstallationDatabasePerseus)
	logger.Info("Provisioning Perseus database")

	err := d.Validate()
	if err != nil {
		return errors.Wrap(err, "perseus database configuration is invalid")
	}

	vpc, err := getVPCForInstallation(d.installationID, store, d.client)
	if err != nil {
		return errors.Wrap(err, "failed to find cluster installation VPC")
	}

	dbResources, unlockFn, err := d.getAndLockAssignedProxiedDatabase(store, logger)
	if err != nil {
		return errors.Wrap(err, "failed to get and lock assigned database")
	}
	if dbResources == nil {
		logger.Debug("Assigning installation to perseus multitenant database")
		dbResources, unlockFn, err = d.assignInstallationToProxiedDatabaseAndLock(*vpc.VpcId, store, logger)
		if err != nil {
			return errors.Wrap(err, "failed to assign installation to a perseus multitenant database")
		}
	}
	defer unlockFn()

	logger = logger.WithFields(log.Fields{
		"multitenant-database":      dbResources.MultitenantDatabase.ID,
		"multitenant-database-type": dbResources.MultitenantDatabase.DatabaseType,
		"rds-cluster-id":            dbResources.MultitenantDatabase.RdsClusterID,
		"logical-database":          dbResources.LogicalDatabase.ID,
		"logical-database-name":     dbResources.LogicalDatabase.Name,
		"database-schema":           dbResources.DatabaseSchema.ID,
	})

	rdsCluster, err := describeRDSCluster(dbResources.MultitenantDatabase.RdsClusterID, d.client)
	if err != nil {
		return errors.Wrapf(err, "failed to describe the multitenant RDS cluster ID %s", dbResources.MultitenantDatabase.ID)
	}
	if *rdsCluster.Status != DefaultRDSStatusAvailable {
		return errors.Errorf("multitenant RDS cluster ID %s is not available (status: %s)", dbResources.MultitenantDatabase.ID, *rdsCluster.Status)
	}

	rdsID := *rdsCluster.DBClusterIdentifier
	logger = logger.WithField("rds-cluster-id", rdsID)

	if dbResources.MultitenantDatabase.State == model.DatabaseStateProvisioningRequested {
		err = d.provisionPerseusDatabase(*vpc.VpcId, rdsCluster, logger)
		if err != nil {
			return errors.Wrap(err, "failed to provision perseus database")
		}

		dbResources.MultitenantDatabase.State = model.DatabaseStateStable
		err = store.UpdateMultitenantDatabase(dbResources.MultitenantDatabase)
		if err != nil {
			return errors.Wrap(err, "failed to update state on provisioned perseus database")
		}
	}

	err = d.ensureLogicalDatabaseExists(dbResources.LogicalDatabase.Name, rdsCluster, logger)
	if err != nil {
		return errors.Wrap(err, "failed to ensure the logical database is created")
	}

	installationSecret, err := d.ensureLogicalDatabaseSetup(dbResources.LogicalDatabase.Name, *vpc.VpcId, rdsCluster, logger)
	if err != nil {
		return errors.Wrap(err, "failed to ensure the logical database is setup")
	}

	err = d.ensurePerseusAuthDatabaseEntriesCreated(dbResources, installationSecret, logger)
	if err != nil {
		return errors.Wrap(err, "failed to create perseus auth database entries")
	}

	err = updateCounterTagWithCurrentWeight(dbResources.MultitenantDatabase, rdsCluster, store, d.client, logger)
	if err != nil {
		return errors.Wrap(err, "failed to update counter tag with current weight")
	}

	logger.Infof("Installation %s assigned to perseus multitenant database", d.installationID)

	return nil
}

func (d *PerseusDatabase) ensurePerseusAuthDatabaseEntriesCreated(dbResources *model.DatabaseResourceGrouping, installationSecret *RDSSecret, logger log.FieldLogger) error {
	// Generate source password.
	hashBytes, err := scrypt.DerivePassphrase(installationSecret.MasterPassword, 32)
	if err != nil {
		return errors.Wrap(err, "failed to hash source password")
	}
	hashedSourcePassword := base64.StdEncoding.EncodeToString(hashBytes)

	// Generate destination password.
	perseusUserSecretName := PerseusDatabaseUserSecretName(dbResources.MultitenantDatabase.RdsClusterID)
	perseusSecret, err := d.client.secretsManagerGetRDSSecret(perseusUserSecretName)
	if err != nil {
		return errors.Wrap(err, "failed to get perseus database user secret")
	}
	key, err := d.client.kmsGetSymmetricKey(PerseusKMSAliasName(dbResources.MultitenantDatabase.VpcID))
	if err != nil {
		return errors.Wrapf(err, "failed to get KMS key for perseus")
	}
	encBytes, err := d.client.kmsEncrypt(*key.Arn, perseusSecret.MasterPassword)
	if err != nil {
		return errors.Wrap(err, "failed to encrypt destination password")
	}
	encryptedDestinationPassword := base64.StdEncoding.EncodeToString(encBytes)

	// Create Perseus auth entries.
	closeDB, err := d.connectToPerseusAuthDatabase(dbResources.MultitenantDatabase.VpcID, logger)
	if err != nil {
		return errors.Wrap(err, "failed to connect to perseus auth database")
	}
	defer closeDB(logger)

	ctx, cancel := context.WithDeadline(context.Background(), time.Now().Add(DefaultPostgresContextTimeSeconds*time.Second))
	defer cancel()

	writerEntryQuery := buildPerseusAuthEntryQuery(
		dbResources.LogicalDatabase.Name,
		installationSecret.MasterUsername,
		hashedSourcePassword,
		dbResources.LogicalDatabase.Name,
		perseusSecret.MasterUsername,
		encryptedDestinationPassword,
		dbResources.MultitenantDatabase.WriterEndpoint,
	)
	readerEntryQuery := buildPerseusAuthEntryQuery(
		dbResources.LogicalDatabase.Name+"-ro",
		installationSecret.MasterUsername,
		hashedSourcePassword,
		dbResources.LogicalDatabase.Name,
		perseusSecret.MasterUsername,
		encryptedDestinationPassword,
		dbResources.MultitenantDatabase.ReaderEndpoint,
	)

	// Combine the two queries to run at the same time.
	_, err = d.db.QueryContext(ctx, writerEntryQuery+readerEntryQuery)
	if err != nil {
		return errors.Wrap(err, "failed to run perseus auth entry SQL command")
	}

	return nil
}

func buildPerseusAuthEntryQuery(sourceDatabase, sourceUsername, sourcePassword, destinationDatabase, destinationUsername, destinationPassword, endpoint string) string {
	return fmt.Sprintf(`INSERT INTO perseus_auth (
		source_db,
		source_schema,
		source_user,
		source_pass_hashed,
		dest_host,
		dest_db,
		dest_user,
		dest_pass_enc)
		values ('%s', '%s','%s', '%s', '%s', '%s', '%s', '%s');`,
		sourceDatabase,
		sourceUsername,
		sourceUsername,
		sourcePassword,
		endpoint,
		destinationDatabase,
		destinationUsername,
		destinationPassword,
	)
}

// This helper method finds a multitenant RDS cluster that is ready for receiving a database installation. The lookup
// for multitenant databases will happen in order:
//  1. fetch a multitenant database by installation ID.
//  2. fetch all multitenant databases in the store which are under the max number of installations limit.
//  3. fetch all multitenant databases in the RDS cluster that are under the max number of installations limit.
func (d *PerseusDatabase) assignInstallationToProxiedDatabaseAndLock(vpcID string, store model.InstallationDatabaseStoreInterface, logger log.FieldLogger) (*model.DatabaseResourceGrouping, func(), error) {
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
		logger.Infof("No multitenant databases with less than %d installations found in the datastore; fetching all available resources from AWS", d.MaxSupportedDatabases())

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
	// as close to maximum efficiency as possible.
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

	// Finish assigning the installation.
	databaseResources, err := store.GetOrCreateProxyDatabaseResourcesForInstallation(d.installationID, selectedDatabase.ID)
	if err != nil {
		unlockFn()
		return nil, nil, errors.Wrap(err, "failed to save installation to selected database")
	}

	return databaseResources, unlockFn, nil
}

func (d *PerseusDatabase) getMultitenantDatabasesFromResourceTags(vpcID string, store model.InstallationDatabaseStoreInterface, logger log.FieldLogger) ([]*model.MultitenantDatabase, error) {
	databaseEngineType := d.DatabaseEngineTypeTagValue()

	resourceNames, err := d.client.resourceTaggingGetAllResources(gt.GetResourcesInput{
		TagFilters: standardMultitenantDatabaseTagFilters(
			DefaultRDSMultitenantDatabasePerseusTypeTagValue,
			databaseEngineType,
			vpcID,
		),
		ResourceTypeFilters: []string{DefaultResourceTypeClusterRDS},
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
			RdsClusterID:                       *rdsClusterID,
			VpcID:                              vpcID,
			DatabaseType:                       d.databaseType,
			State:                              model.DatabaseStateProvisioningRequested,
			WriterEndpoint:                     *rdsCluster.Endpoint,
			ReaderEndpoint:                     *rdsCluster.ReaderEndpoint,
			MaxInstallationsPerLogicalDatabase: model.GetDefaultProxyDatabaseMaxInstallationsPerLogicalDatabase(),
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
func (d *PerseusDatabase) getAndLockAssignedProxiedDatabase(store model.InstallationDatabaseStoreInterface, logger log.FieldLogger) (*model.DatabaseResourceGrouping, func(), error) {
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
	databaseResources, err := store.GetOrCreateProxyDatabaseResourcesForInstallation(d.installationID, multitenantDatabases[0].ID)
	if err != nil {
		unlockFn()
		return nil, nil, errors.Wrap(err, "failed to get database resources")
	}

	return databaseResources, unlockFn, nil
}

func (d *PerseusDatabase) provisionPerseusDatabase(vpcID string, rdsCluster *rdsTypes.DBCluster, logger log.FieldLogger) error {
	rdsID := *rdsCluster.DBClusterIdentifier

	logger.Infof("Provisioning Perseus database %s", rdsID)

	masterDBSecret, err := d.client.Service().secretsManager.GetSecretValue(
		context.TODO(),
		&secretsmanager.GetSecretValueInput{
			SecretId: rdsCluster.DBClusterIdentifier,
		})
	if err != nil {
		return errors.Wrapf(err, "failed to find the master secret for the multitenant proxy cluster %s", rdsID)
	}

	perseusUser, err := d.ensurePerseusDatabaseUserSecretIsCreated(&rdsID, &vpcID)
	if err != nil {
		return errors.Wrapf(err, "failed to ensure the perseus database user secret was created for %s", rdsID)
	}

	pgbouncerUser, err := d.ensurePGBouncerAuthUserSecretIsCreated(&rdsID, &vpcID)
	if err != nil {
		return errors.Wrapf(err, "failed to ensure the pgbouncer auth user secret was created for %s", rdsID)
	}

	closeDB, err := d.connectToRDSCluster(rdsPostgresDefaultSchema, *rdsCluster.Endpoint, DefaultMattermostDatabaseUsername, *masterDBSecret.SecretString)
	if err != nil {
		return errors.Wrapf(err, "failed to connect to the multitenant proxy cluster %s", rdsID)
	}
	defer closeDB(logger)

	ctx, cancel := context.WithDeadline(context.Background(), time.Now().Add(DefaultPostgresContextTimeSeconds*time.Second))
	defer cancel()

	err = ensureDatabaseUserIsCreated(ctx, d.db, perseusUser.MasterUsername, perseusUser.MasterPassword)
	if err != nil {
		return errors.Wrapf(err, "failed to ensure the perseus database user was created for %s", rdsID)
	}

	err = ensureDatabaseUserIsCreated(ctx, d.db, pgbouncerUser.MasterUsername, pgbouncerUser.MasterPassword)
	if err != nil {
		return errors.Wrapf(err, "failed to ensure the pgbouncer database user was created for %s", rdsID)
	}

	return nil
}

func (d *PerseusDatabase) ensurePerseusDatabaseUserSecretIsCreated(rdsClusterID, VpcID *string) (*RDSSecret, error) {
	perseusUserSecretName := PerseusDatabaseUserSecretName(*rdsClusterID)

	secret, err := d.client.secretsManagerGetRDSSecret(perseusUserSecretName)
	if err != nil {
		var awsErr *smTypes.ResourceNotFoundException
		if !errors.As(err, &awsErr) {
			return nil, errors.Wrapf(err, "failed to get perseus database user secret %s", perseusUserSecretName)
		}
	}

	if secret == nil {
		description := RDSMultitenantPerseusClusterSecretDescription(*rdsClusterID)
		tags := []smTypes.Tag{
			{
				Key:   aws.String(trimTagPrefix(DefaultRDSMultitenantDatabaseIDTagKey)),
				Value: rdsClusterID,
			},
			{
				Key:   aws.String(trimTagPrefix(VpcIDTagKey)),
				Value: VpcID,
			},
		}

		secret, err = createDatabaseUserSecret(perseusUserSecretName, DefaultPerseusDatabaseUsername, description, tags, d.client)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to create perseus database user secret %s", perseusUserSecretName)
		}
	}

	return secret, nil
}

func (d *PerseusDatabase) ensurePGBouncerAuthUserSecretIsCreated(rdsClusterID, VpcID *string) (*RDSSecret, error) {
	authUserSecretName := PGBouncerAuthUserSecretName(*VpcID)

	secret, err := d.client.secretsManagerGetRDSSecret(authUserSecretName)
	if err != nil {
		var awsErr *smTypes.ResourceNotFoundException
		if !errors.As(err, &awsErr) {
			return nil, errors.Wrapf(err, "failed to get pgbouncer auth user secret %s", authUserSecretName)
		}
	}

	if secret == nil {
		description := RDSMultitenantPGBouncerClusterSecretDescription(*VpcID)
		tags := []smTypes.Tag{
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

func (d *PerseusDatabase) ensureLogicalDatabaseExists(databaseName string, rdsCluster *rdsTypes.DBCluster, logger log.FieldLogger) error {
	rdsID := *rdsCluster.DBClusterIdentifier

	masterSecretValue, err := d.client.Service().secretsManager.GetSecretValue(
		context.TODO(),
		&secretsmanager.GetSecretValueInput{
			SecretId: rdsCluster.DBClusterIdentifier,
		})
	if err != nil {
		return errors.Wrapf(err, "failed to find the master secret for the multitenant proxy cluster %s", rdsID)
	}

	closeDB, err := d.connectToRDSCluster(rdsPostgresDefaultSchema, *rdsCluster.Endpoint, DefaultMattermostDatabaseUsername, *masterSecretValue.SecretString)
	if err != nil {
		return errors.Wrapf(err, "failed to connect to the multitenant proxy cluster %s", rdsID)
	}
	defer closeDB(logger)

	ctx, cancel := context.WithDeadline(context.Background(), time.Now().Add(DefaultPostgresContextTimeSeconds*time.Second))
	defer cancel()

	err = ensureDatabaseIsCreated(ctx, d.db, databaseName)
	if err != nil {
		return errors.Wrapf(err, "failed to create database in multitenant proxy cluster %s", rdsID)
	}

	return nil
}

func (d *PerseusDatabase) ensureLogicalDatabaseSetup(databaseName, vpcID string, rdsCluster *rdsTypes.DBCluster, logger log.FieldLogger) (*RDSSecret, error) {
	rdsID := *rdsCluster.DBClusterIdentifier

	masterSecretValue, err := d.client.Service().secretsManager.GetSecretValue(
		context.TODO(),
		&secretsmanager.GetSecretValueInput{
			SecretId: rdsCluster.DBClusterIdentifier,
		})
	if err != nil {
		return nil, errors.Wrapf(err, "failed to find the master secret for the multitenant proxy cluster %s", rdsID)
	}

	closeDB, err := d.connectToRDSCluster(databaseName, *rdsCluster.Endpoint, DefaultMattermostDatabaseUsername, *masterSecretValue.SecretString)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to connect to the multitenant proxy cluster %s", rdsID)
	}
	defer closeDB(logger)

	ctx, cancel := context.WithDeadline(context.Background(), time.Now().Add(DefaultPostgresContextTimeSeconds*time.Second))
	defer cancel()

	err = d.ensureBaselineDatabasePrep(ctx, databaseName)
	if err != nil {
		return nil, errors.Wrap(err, "failed to perform basic database setup")
	}

	// Run PGBouncer prep to provide a clean path for migrations to and from
	// Perseus.
	err = d.ensurePGBouncerDatabasePrep(ctx, databaseName, DefaultPGBouncerAuthUsername)
	if err != nil {
		return nil, errors.Wrap(err, "failed to perform pgbouncer setup")
	}

	installationSecret, err := d.ensureMultitenantDatabaseSecretIsCreated(rdsCluster.DBClusterIdentifier, &vpcID)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get secret for installation")
	}

	err = ensureDatabaseUserIsCreated(ctx, d.db, installationSecret.MasterUsername, installationSecret.MasterPassword)
	if err != nil {
		return nil, errors.Wrap(err, "failed to create Mattermost database user")
	}

	err = d.ensurePerseusUserHasRole(ctx, installationSecret.MasterUsername)
	if err != nil {
		return nil, errors.Wrap(err, "failed to grant permissions to perseus database user")
	}

	err = d.ensureMasterUserHasRole(ctx, DefaultPerseusDatabaseUsername)
	if err != nil {
		return nil, errors.Wrap(err, "failed to ensure master user has role")
	}

	err = d.ensureSchemaIsCreated(ctx, installationSecret.MasterUsername, DefaultPerseusDatabaseUsername)
	if err != nil {
		return nil, errors.Wrap(err, "failed to create installation schema")
	}

	query := fmt.Sprintf(`GRANT ALL PRIVILEGES ON DATABASE %s TO %s`, databaseName, DefaultPerseusDatabaseUsername)
	_, err = d.db.QueryContext(ctx, query)
	if err != nil {
		return nil, errors.Wrap(err, "failed to grant perseus previleges")
	}

	return installationSecret, nil
}

func (d *PerseusDatabase) ensureBaselineDatabasePrep(ctx context.Context, databaseName string) error {
	query := fmt.Sprintf(`ALTER DATABASE %s SET search_path = "$user"`, databaseName)
	_, err := d.db.QueryContext(ctx, query)
	if err != nil {
		return errors.Wrap(err, "failed to run database search path set SQL command")
	}

	err = ensureDefaultTextSearchConfig(ctx, d.db, databaseName)
	if err != nil {
		return errors.Wrap(err, "failed to ensure default text search config")
	}

	return nil
}

func (d *PerseusDatabase) ensurePGBouncerDatabasePrep(ctx context.Context, databaseName, pgbouncerUsername string) error {
	err := d.ensureMasterUserHasRole(ctx, pgbouncerUsername)
	if err != nil {
		return errors.Wrap(err, "failed to run master user has pgbouncer role SQL command")
	}

	err = d.ensureSchemaIsCreated(ctx, pgbouncerUsername, DefaultMattermostDatabaseUsername)
	if err != nil {
		return errors.Wrap(err, "failed to run pgbouncer schema is created SQL command")
	}

	query := fmt.Sprintf(`
	CREATE TABLE IF NOT EXISTS %s.pgbouncer_users(
	usename NAME PRIMARY KEY,
	passwd TEXT NOT NULL
	)`, pgbouncerUsername)
	_, err = d.db.QueryContext(ctx, query)
	if err != nil {
		return errors.Wrap(err, "failed to run pgbouncer user table creation SQL command")
	}

	query = fmt.Sprintf("GRANT SELECT ON ALL TABLES IN SCHEMA %s TO %s;", pgbouncerUsername, pgbouncerUsername)
	_, err = d.db.QueryContext(ctx, query)
	if err != nil {
		return errors.Wrap(err, "failed to run pgbouncer select grant SQL command")
	}

	return nil
}

func (d *PerseusDatabase) ensureSchemaIsCreated(ctx context.Context, schemaName, username string) error {
	return createSchemaIfNotExists(ctx, d.db, schemaName, username)
}

func (d *PerseusDatabase) ensureMasterUserHasRole(ctx context.Context, role string) error {
	query := fmt.Sprintf("GRANT %s TO %s;", role, DefaultMattermostDatabaseUsername)
	_, err := d.db.QueryContext(ctx, query)
	if err != nil {
		return errors.Wrap(err, "failed to run grant role SQL command")
	}

	return nil
}

func (d *PerseusDatabase) ensurePerseusUserHasRole(ctx context.Context, role string) error {
	query := fmt.Sprintf("GRANT %s TO %s;", role, DefaultPerseusDatabaseUsername)
	_, err := d.db.QueryContext(ctx, query)
	if err != nil {
		return errors.Wrap(err, "failed to run grant role SQL command")
	}

	return nil
}

func (d *PerseusDatabase) connectToRDSCluster(database, endpoint, username, password string) (func(logger log.FieldLogger), error) {
	db, closeFunc, err := connectToPostgresRDSCluster(database, endpoint, username, password)
	if err != nil {
		return nil, errors.Wrap(err, "failed to connect to postgres database")
	}

	d.db = db

	return closeFunc, nil
}

func (d *PerseusDatabase) connectToPerseusAuthDatabase(vpcID string, logger log.FieldLogger) (func(logger log.FieldLogger), error) {
	authCluster, err := getPerseusAuthDatabaseCluster(vpcID, d.client, logger)
	if err != nil {
		return nil, errors.Wrap(err, "failed to find perseus auth cluster")
	}

	masterDBSecret, err := d.client.Service().secretsManager.GetSecretValue(
		context.TODO(),
		&secretsmanager.GetSecretValueInput{
			SecretId: authCluster.DBClusterIdentifier,
		})
	if err != nil {
		return nil, errors.Wrap(err, "failed to get database master secret")
	}

	db, closeFunc, err := connectToPostgresRDSCluster(DefaultPerseusAuthDatabaseName, *authCluster.Endpoint, DefaultMattermostDatabaseUsername, *masterDBSecret.SecretString)
	if err != nil {
		return nil, errors.Wrap(err, "failed to connect to postgres database")
	}

	d.db = db

	return closeFunc, nil
}

func (d *PerseusDatabase) ensureMultitenantDatabaseSecretIsCreated(rdsClusterID, vpcID *string) (*RDSSecret, error) {
	installationSecretName := PerseusInstallationSecretName(d.installationID)

	installationSecret, err := d.client.secretsManagerGetRDSSecret(installationSecretName)
	var awsErr *smTypes.ResourceNotFoundException
	if err != nil && !errors.As(err, &awsErr) {
		return nil, errors.Wrapf(err, "failed to get multitenant RDS database secret %s", installationSecretName)
	}
	if installationSecret != nil {
		if err != nil {
			return nil, errors.Wrapf(err, "failed to unmarshal multitenant RDS database secret %s", installationSecretName)
		}

		return installationSecret, nil
	}

	description := RDSMultitenantClusterSecretDescription(d.installationID, *rdsClusterID)
	tags := []smTypes.Tag{
		{
			Key:   aws.String(trimTagPrefix(DefaultRDSMultitenantDatabaseIDTagKey)),
			Value: rdsClusterID,
		},
		{
			Key:   aws.String(trimTagPrefix(VpcIDTagKey)),
			Value: vpcID,
		},
		{
			Key:   aws.String(trimTagPrefix(DefaultMattermostInstallationIDTagKey)),
			Value: aws.String(d.installationID),
		},
	}

	username := MattermostPerseusDatabaseUsername(d.installationID)
	installationSecret, err = createDatabaseUserSecret(installationSecretName, username, description, tags, d.client)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to create a multitenant RDS database secret %s", installationSecretName)
	}

	return installationSecret, nil
}

// GenerateDatabaseSecret creates the k8s database spec and secret for
// accessing a single schema inside a RDS multitenant cluster with a Perseus
// proxy.
func (d *PerseusDatabase) GenerateDatabaseSecret(store model.InstallationDatabaseStoreInterface, logger log.FieldLogger) (*corev1.Secret, error) {
	err := d.Validate()
	if err != nil {
		return nil, errors.Wrap(err, "perseus database configuration is invalid")
	}

	dbResources, err := store.GetProxyDatabaseResourcesForInstallation(d.installationID)
	if err != nil {
		return nil, errors.Wrap(err, "failed to query for database resources")
	}
	if dbResources == nil {
		return nil, errors.New("no database resources found for this installation; it potentially has not been assigned yet")
	}

	unlock, err := lockMultitenantDatabase(dbResources.MultitenantDatabase.ID, d.instanceID, store, logger)
	if err != nil {
		return nil, errors.Wrap(err, "failed to lock multitenant database")
	}
	defer unlock()

	logger = logger.WithFields(log.Fields{
		"multitenant-database":      dbResources.MultitenantDatabase.ID,
		"multitenant-database-type": dbResources.MultitenantDatabase.DatabaseType,
		"rds-cluster-id":            dbResources.MultitenantDatabase.RdsClusterID,
		"logical-database":          dbResources.LogicalDatabase.ID,
		"logical-database-name":     dbResources.LogicalDatabase.Name,
		"database-schema":           dbResources.DatabaseSchema.ID,
	})

	installationSecretName := PerseusInstallationSecretName(d.installationID)

	installationSecret, err := d.client.secretsManagerGetRDSSecret(installationSecretName)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get secret value for installation")
	}

	databaseConnectionString, databaseReadReplicasString, databaseConnectionCheck :=
		MattermostPerseusConnStrings(
			installationSecret.MasterUsername,
			installationSecret.MasterPassword,
			dbResources.LogicalDatabase.Name,
		)

	secret := InstallationDBSecret{
		InstallationSecretName: installationSecretName,
		ConnectionString:       databaseConnectionString,
		DBCheckURL:             databaseConnectionCheck,
		ReadReplicasURL:        databaseReadReplicasString,
	}

	logger.Debug("Perseus database configuration generated for cluster installation")

	return secret.ToK8sSecret(d.disableDBCheck), nil
}

// Teardown removes all AWS resources related to a Perseus database.
func (d *PerseusDatabase) Teardown(store model.InstallationDatabaseStoreInterface, keepData bool, logger log.FieldLogger) error {
	logger = logger.WithField("database", model.InstallationDatabasePerseus)
	logger.Info("Tearing down Perseus database")

	err := d.Validate()
	if err != nil {
		return errors.Wrap(err, "perseus database configuration is invalid")
	}

	if keepData {
		logger.Warn("Keepdata is set to true on this server, but this is not yet supported for Perseus databases")
	}

	dbResources, unlockFn, err := d.getAndLockAssignedProxiedDatabase(store, logger)
	if err != nil {
		return errors.Wrap(err, "failed to get assigned multitenant database")
	}
	if dbResources != nil {
		defer unlockFn()
		err = d.removeInstallationPerseusResources(dbResources, store, logger)
		if err != nil {
			return errors.Wrap(err, "failed to remove installation database")
		}
	} else {
		logger.Debug("No Perseus database found for this installation; skipping...")
	}

	logger.Info("Perseus database teardown complete")

	return nil
}

// removeInstallationPerseusResources performs the work necessary to remove a
// single installation schema from a Perseus RDS cluster.
func (d *PerseusDatabase) removeInstallationPerseusResources(dbResources *model.DatabaseResourceGrouping, store model.InstallationDatabaseStoreInterface, logger log.FieldLogger) error {
	rdsCluster, err := describeRDSCluster(dbResources.MultitenantDatabase.RdsClusterID, d.client)
	if err != nil {
		return errors.Wrap(err, "failed to describe rds cluster")
	}

	logger = logger.WithField("rds-cluster-id", *rdsCluster.DBClusterIdentifier)

	err = d.client.secretsManagerEnsureSecretDeleted(PerseusInstallationSecretName(d.installationID), false, logger)
	if err != nil {
		return errors.Wrap(err, "failed to delete multitenant database secret")
	}

	username := MattermostPerseusDatabaseUsername(d.installationID)

	err = d.cleanupDatabase(dbResources.MultitenantDatabase.VpcID, *rdsCluster.DBClusterIdentifier, *rdsCluster.Endpoint, dbResources.LogicalDatabase.Name, username, logger)
	if err != nil {
		return errors.Wrap(err, "failed to cleanup perseus database")
	}

	dbResources.MultitenantDatabase.Installations.Remove(d.installationID)
	err = updateCounterTagWithCurrentWeight(dbResources.MultitenantDatabase, rdsCluster, store, d.client, logger)
	if err != nil {
		return errors.Wrap(err, "failed to update counter tag with current weight")
	}

	err = store.DeleteInstallationProxyDatabaseResources(dbResources.MultitenantDatabase, dbResources.DatabaseSchema)
	if err != nil {
		return errors.Wrapf(err, "failed to remove installation ID %s from multitenant datastore", d.installationID)
	}

	return nil
}

func (d *PerseusDatabase) cleanupDatabase(vpcID, rdsClusterID, rdsClusterendpoint, databaseName, installationUsername string, logger log.FieldLogger) error {
	err := d.cleanupPerseusMultitenantDatabase(rdsClusterID, rdsClusterendpoint, databaseName, installationUsername, logger)
	if err != nil {
		return errors.Wrap(err, "failed to cleanup multitenant database")
	}

	err = d.cleanupPerseusAuthDatabase(vpcID, installationUsername, logger)
	if err != nil {
		return errors.Wrap(err, "failed to cleanup authentication database")
	}

	return nil
}

func (d *PerseusDatabase) cleanupPerseusMultitenantDatabase(rdsClusterID, rdsClusterendpoint, databaseName, installationUsername string, logger log.FieldLogger) error {
	masterDBSecret, err := d.client.Service().secretsManager.GetSecretValue(
		context.TODO(),
		&secretsmanager.GetSecretValueInput{
			SecretId: aws.String(rdsClusterID),
		})
	if err != nil {
		return errors.Wrapf(err, "failed to get master secret by ID %s", rdsClusterID)
	}

	closeDB, err := d.connectToRDSCluster(databaseName, rdsClusterendpoint, DefaultMattermostDatabaseUsername, *masterDBSecret.SecretString)
	if err != nil {
		return errors.Wrapf(err, "failed to connect to multitenant RDS cluster ID %s", rdsClusterID)
	}
	defer closeDB(logger)

	ctx, cancel := context.WithDeadline(context.Background(), time.Now().Add(DefaultPostgresContextTimeSeconds*time.Second))
	defer cancel()

	err = dropSchemaIfExists(ctx, d.db, installationUsername)
	if err != nil {
		return errors.Wrap(err, "failed to delete installation schema")
	}

	err = dropUserIfExists(ctx, d.db, installationUsername)
	if err != nil {
		return errors.Wrap(err, "failed to delete installation database user")
	}

	return nil
}

func (d *PerseusDatabase) cleanupPerseusAuthDatabase(vpcID, username string, logger log.FieldLogger) error {
	closeDB, err := d.connectToPerseusAuthDatabase(vpcID, logger)
	if err != nil {
		return errors.Wrap(err, "failed to connect to perseus auth database")
	}
	defer closeDB(logger)

	ctx, cancel := context.WithDeadline(context.Background(), time.Now().Add(DefaultPostgresContextTimeSeconds*time.Second))
	defer cancel()

	query := fmt.Sprintf(`DELETE FROM perseus_auth WHERE source_user = '%s';`, username)
	_, err = d.db.QueryContext(ctx, query)
	if err != nil {
		return errors.Wrap(err, "failed to run perseus auth cleanup SQL command")
	}

	return nil
}

// GeneratePerseusUtilitySecret provisions Perseus resources and returns the k8s
// secret needed by the perseus service to perform secure authentication tasks.
func (a *Client) GeneratePerseusUtilitySecret(clusterID string, logger log.FieldLogger) (*corev1.Secret, error) {
	vpc, err := getVPCForCluster(clusterID, a)
	if err != nil {
		return nil, errors.Wrap(err, "failed to find cluster VPC")
	}

	perseusAuthDatabase, err := getPerseusAuthDatabaseCluster(*vpc.VpcId, a, logger)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get perseus auth database DB cluster")
	}

	authUserCredentials, err := ensurePerseusAuthDatabaseProvisioned(a, perseusAuthDatabase, vpc, logger)
	if err != nil {
		return nil, errors.Wrap(err, "failed to ensure perseus auth database was provisioned")
	}

	authDSN := fmt.Sprintf("postgres://%s:%s@%s:5432/perseus?connect_timeout=10",
		authUserCredentials.MasterUsername, authUserCredentials.MasterPassword, *perseusAuthDatabase.Endpoint,
	)

	perseusIAMCreds, err := a.secretsManagerGetIAMAccessKeyFromSecretName(PerseusIAMUserSecretName(*vpc.VpcId))
	if err != nil {
		return nil, errors.Wrapf(err, "failed to get IAM user credential secret for perseus")
	}

	kms, err := a.kmsGetSymmetricKey(PerseusKMSAliasName(*vpc.VpcId))
	if err != nil {
		return nil, errors.Wrapf(err, "failed to get KMS ARN for perseus")
	}

	return &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name: "perseus",
		},
		StringData: map[string]string{
			"AuthDBSettings_AuthDBDSN":    authDSN,                // DSN to database with perseus connection data
			"AWSSettings_AccessKeyId":     perseusIAMCreds.ID,     // IAM ID to access KMS encryption/decryption secret
			"AWSSettings_SecretAccessKey": perseusIAMCreds.Secret, // IAM Secret to access KMS encryption/decryption secret
			"AWSSettings_KMSKeyARN":       *kms.Arn,               // KMS secret to use for encryption/decryption
		},
	}, nil
}

func ensurePerseusAuthDatabaseProvisioned(a *Client, rdsCluster *rdsTypes.DBCluster, vpc *ec2Types.Vpc, logger log.FieldLogger) (*RDSSecret, error) {
	rdsID := *rdsCluster.DBClusterIdentifier

	masterSecretValue, err := a.Service().secretsManager.GetSecretValue(
		context.TODO(),
		&secretsmanager.GetSecretValueInput{
			SecretId: rdsCluster.DBClusterIdentifier,
		})
	if err != nil {
		return nil, errors.Wrapf(err, "failed to find the master secret for %s", rdsID)
	}

	authUserCredentials, err := ensurePerseusAuthUserSecretIsCreated(a, rdsCluster.DBClusterIdentifier, vpc.VpcId)
	if err != nil {
		return nil, errors.Wrap(err, "failed to ensure perseus auth user secret was created")
	}

	err = func() error {
		db, closeDB, errInner := connectToPostgresRDSCluster(rdsPostgresDefaultSchema, *rdsCluster.Endpoint, DefaultMattermostDatabaseUsername, *masterSecretValue.SecretString)
		if errInner != nil {
			return errors.Wrapf(err, "failed to connect to RDS cluster %s", rdsID)
		}
		defer closeDB(logger)

		ctx, cancel := context.WithDeadline(context.Background(), time.Now().Add(DefaultPostgresContextTimeSeconds*time.Second))
		defer cancel()

		errInner = ensureDatabaseUserIsCreated(ctx, db, authUserCredentials.MasterUsername, authUserCredentials.MasterPassword)
		if errInner != nil {
			return errors.Wrap(err, "failed to ensure perseus auth database user was created")
		}

		errInner = ensureDatabaseIsCreated(ctx, db, DefaultPerseusAuthDatabaseName)
		if errInner != nil {
			return errors.Wrap(err, "failed to ensure perseus auth logical database was created")
		}

		return nil
	}()
	if err != nil {
		return nil, err
	}

	db, closeDB, err := connectToPostgresRDSCluster(DefaultPerseusAuthDatabaseName, *rdsCluster.Endpoint, DefaultMattermostDatabaseUsername, *masterSecretValue.SecretString)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to connect to RDS cluster %s", rdsID)
	}
	defer closeDB(logger)

	ctx, cancel := context.WithDeadline(context.Background(), time.Now().Add(DefaultPostgresContextTimeSeconds*time.Second))
	defer cancel()

	query := `CREATE TABLE IF NOT EXISTS perseus_auth (
		id serial primary key,
		source_db character varying(64),
		source_schema character varying(64),
		source_user character varying(64),
		source_pass_hashed character varying(1024),
		dest_host character varying(1024),
		dest_db character varying(64),
		dest_user character varying(64),
		dest_pass_enc character varying(1024),
		unique (source_db, source_schema)
	);`
	_, err = db.QueryContext(ctx, query)
	if err != nil {
		return nil, errors.Wrap(err, "failed to create perseus_auth table")
	}

	_, err = db.QueryContext(ctx, fmt.Sprintf(`GRANT SELECT ON perseus_auth TO %s;`, authUserCredentials.MasterUsername))
	if err != nil {
		return nil, errors.Wrap(err, "failed to grant permissions to perseus auth user")
	}

	return authUserCredentials, nil
}

func ensurePerseusAuthUserSecretIsCreated(a *Client, rdsClusterID, VpcID *string) (*RDSSecret, error) {
	authUserSecretName := PerseusAuthUserSecretName(*VpcID)

	secret, err := a.secretsManagerGetRDSSecret(authUserSecretName)
	if err != nil {
		var awsErr *smTypes.ResourceNotFoundException
		if !errors.As(err, &awsErr) {
			return nil, errors.Wrapf(err, "failed to get perseus auth user secret %s", authUserSecretName)
		}
	}

	if secret == nil {
		description := RDSMultitenantPerseusAuthSecretDescription(*VpcID)
		tags := []smTypes.Tag{
			{
				Key:   aws.String(trimTagPrefix(DefaultPerseusAuthDatabaseIDTagKey)),
				Value: rdsClusterID,
			},
			{
				Key:   aws.String(trimTagPrefix(VpcIDTagKey)),
				Value: VpcID,
			},
		}

		secret, err = createDatabaseUserSecret(authUserSecretName, DefaultPerseusAuthUsername, description, tags, a)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to create perseus auth user secret %s", authUserSecretName)
		}
	}

	return secret, nil
}

func getPerseusAuthDatabaseCluster(vpcID string, a *Client, logger log.FieldLogger) (*rdsTypes.DBCluster, error) {
	resourceNames, err := a.resourceTaggingGetAllResources(gt.GetResourcesInput{
		TagFilters: []gtTypes.TagFilter{
			{
				Key:    aws.String(trimTagPrefix(RDSMultitenantPurposeTagKey)),
				Values: []string{RDSMultitenantPurposeTagValueProvisioning},
			},
			{
				Key:    aws.String(trimTagPrefix(RDSMultitenantOwnerTagKey)),
				Values: []string{RDSMultitenantOwnerTagValueCloudTeam},
			},
			{
				Key:    aws.String(DefaultAWSTerraformProvisionedKey),
				Values: []string{DefaultAWSTerraformProvisionedValueTrue},
			},
			{
				Key:    aws.String(trimTagPrefix(DefaultPerseusAuthDatabaseTagKey)),
				Values: []string{DefaultPerseusAuthDatabaseTagValue},
			},
			{
				Key:    aws.String(trimTagPrefix(VpcIDTagKey)),
				Values: []string{vpcID},
			},
		},
		ResourceTypeFilters: []string{DefaultResourceTypeClusterRDS},
	})
	if err != nil {
		return nil, errors.Wrap(err, "failed to query for resources with perseus auth tags")
	}

	// TODO: AWS returns extra cluster resources that have an identical
	// tag match to the actual cluster. Improve this logic when a better method
	// is found.
	dbClusterARNs := []string{}
	var resourceARN arn.ARN
	for _, resource := range resourceNames {
		resourceARN, err = arn.Parse(*resource.ResourceARN)
		if err != nil {
			return nil, err
		}
		if strings.HasPrefix(resourceARN.Resource, "cluster:cluster-") {
			logger.Debugf("Skipping duplicate RDS resource (%s) with 'cluster' prefix", resourceARN.Resource)
			continue
		}
		dbClusterARNs = append(dbClusterARNs, *resource.ResourceARN)
	}

	if len(dbClusterARNs) != 1 {
		return nil, errors.Errorf("expected exactly 1 perseus auth database, but found %d", len(dbClusterARNs))
	}

	rdsCluster, err := describeRDSCluster(dbClusterARNs[0], a)
	if err != nil {
		return nil, errors.Wrap(err, "failed to describe RDS cluster")
	}

	return rdsCluster, nil
}

// Unsupported Methods

// Snapshot creates a snapshot of single RDS multitenant database.
func (d *PerseusDatabase) Snapshot(store model.InstallationDatabaseStoreInterface, logger log.FieldLogger) error {
	return errors.New("not implemented")
}

// MigrateOut migrating out of MySQL Operator managed database is not supported.
func (d *PerseusDatabase) MigrateOut(store model.InstallationDatabaseStoreInterface, dbMigration *model.InstallationDBMigrationOperation, logger log.FieldLogger) error {
	return errors.New("database migration is not supported for Perseus databases")
}

// MigrateTo migration to MySQL Operator managed database is not supported.
func (d *PerseusDatabase) MigrateTo(store model.InstallationDatabaseStoreInterface, dbMigration *model.InstallationDBMigrationOperation, logger log.FieldLogger) error {
	return errors.New("database migration is not supported for Perseus databases")
}

// TeardownMigrated tearing down migrated databases is not supported for MySQL Operator managed database.
func (d *PerseusDatabase) TeardownMigrated(store model.InstallationDatabaseStoreInterface, migrationOp *model.InstallationDBMigrationOperation, logger log.FieldLogger) error {
	return errors.New("tearing down migrated installations is not supported for Perseus databases")
}

// RollbackMigration rolling back migration is not supported for MySQL Operator managed database.
func (d *PerseusDatabase) RollbackMigration(store model.InstallationDatabaseStoreInterface, dbMigration *model.InstallationDBMigrationOperation, logger log.FieldLogger) error {
	return errors.New("rolling back db migration is not supported for Perseus databases")
}

// RefreshResourceMetadata ensures various operator database resource's metadata
// are correct.
func (d *PerseusDatabase) RefreshResourceMetadata(store model.InstallationDatabaseStoreInterface, logger log.FieldLogger) error {
	return nil
}
