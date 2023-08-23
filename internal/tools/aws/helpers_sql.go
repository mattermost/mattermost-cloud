// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.
//

package aws

import (
	"context"
	"database/sql"
	"fmt"

	rdsTypes "github.com/aws/aws-sdk-go-v2/service/rds/types"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
)

// SQLDatabaseManager is an interface that describes operations to query and to
// close connection with a database. It's used mainly to implement a client that
// needs to perform non-complex queries in a SQL database instance.
type SQLDatabaseManager interface {
	QueryContext(ctx context.Context, query string, args ...interface{}) (*sql.Rows, error)
	Close() error
}

// MattermostMySQLConnStrings formats the connection string used for accessing a
// Mattermost database.
func MattermostMySQLConnStrings(schema, username, password string, dbCluster *rdsTypes.DBCluster) (string, string) {
	dbConnection := fmt.Sprintf("mysql://%s:%s@tcp(%s:3306)/%s?charset=utf8mb4%%2Cutf8&readTimeout=30s&writeTimeout=30s&tls=skip-verify",
		username, password, *dbCluster.Endpoint, schema)
	readReplicas := fmt.Sprintf("%s:%s@tcp(%s:3306)/%s?charset=utf8mb4%%2Cutf8&readTimeout=30s&writeTimeout=30s&tls=skip-verify",
		username, password, *dbCluster.ReaderEndpoint, schema)

	return dbConnection, readReplicas
}

// RDSMySQLConnString formats the connection string used by the provisioner for
// accessing a MySQL RDS cluster.
func RDSMySQLConnString(schema, endpoint, username, password string) string {
	return fmt.Sprintf("%s:%s@tcp(%s:3306)/%s?interpolateParams=true&charset=utf8mb4,utf8&readTimeout=30s&writeTimeout=30s&tls=skip-verify",
		username, password, endpoint, schema)
}

// MattermostPostgresConnStrings formats the connection strings used by Mattermost
// servers to access a PostgreSQL database.
func MattermostPostgresConnStrings(schema, username, password string, dbCluster *rdsTypes.DBCluster) (string, string) {
	dbConnection := fmt.Sprintf("postgres://%s:%s@%s:5432/%s?connect_timeout=10",
		username, password, *dbCluster.Endpoint, schema)
	readReplicas := fmt.Sprintf("postgres://%s:%s@%s:5432/%s?connect_timeout=10",
		username, password, *dbCluster.ReaderEndpoint, schema)

	return dbConnection, readReplicas
}

// MattermostPostgresPGBouncerConnStrings formats the connection strings used by
// Mattermost servers to access a PostgreSQL database with a PGBouncer proxy.
//
// Regarding binary_parameters:
// https://blog.bullgare.com/2019/06/pgbouncer-and-prepared-statements
func MattermostPostgresPGBouncerConnStrings(username, password, database string) (string, string, string) {
	dbConnection := fmt.Sprintf("postgres://%s:%s@pgbouncer.pgbouncer.svc.cluster.local:5432/%s?connect_timeout=10&sslmode=disable&binary_parameters=yes",
		username, password, database)
	readReplicas := fmt.Sprintf("postgres://%s:%s@pgbouncer.pgbouncer.svc.cluster.local:5432/%s-ro?connect_timeout=10&sslmode=disable&binary_parameters=yes",
		username, password, database)
	connectionCheck := fmt.Sprintf("postgres://%s:%s@pgbouncer.pgbouncer.svc.cluster.local:5432/%s?connect_timeout=10&sslmode=disable",
		username, password, database)

	return dbConnection, readReplicas, connectionCheck
}

// MattermostPerseusConnStrings formats the connection strings used by
// Mattermost servers to access a PostgreSQL database with a Perseus proxy.
//
// Regarding binary_parameters:
// https://blog.bullgare.com/2019/06/pgbouncer-and-prepared-statements
func MattermostPerseusConnStrings(username, password, database string) (string, string, string) {
	dbConnection := fmt.Sprintf("postgres://%s:%s@perseus.perseus.svc.cluster.local:5432/%s?schema_search_path=%s&connect_timeout=10&sslmode=disable&binary_parameters=yes",
		username, password, database, username)
	readReplicas := fmt.Sprintf("postgres://%s:%s@perseus.perseus.svc.cluster.local:5432/%s-ro?schema_search_path=%s&connect_timeout=10&sslmode=disable&binary_parameters=yes",
		username, password, database, username)
	connectionCheck := fmt.Sprintf("postgres://%s:%s@perseus.perseus.svc.cluster.local:5432/%s?connect_timeout=10&sslmode=disable",
		username, password, database)

	return dbConnection, readReplicas, connectionCheck
}

// RDSPostgresConnString formats the connection string used by the provisioner
// for accessing a Postgres RDS cluster.
func RDSPostgresConnString(schema, endpoint, username, password string) string {
	return fmt.Sprintf("postgres://%s:%s@%s:5432/%s?connect_timeout=10",
		username, password, endpoint, schema)
}

func connectToPostgresRDSCluster(database, endpoint, username, password string) (SQLDatabaseManager, func(logger log.FieldLogger), error) {
	db, err := sql.Open("postgres", RDSPostgresConnString(database, endpoint, username, password))
	if err != nil {
		return nil, nil, errors.Wrap(err, "failed to connect to postgres database")
	}

	closeFunc := func(logger log.FieldLogger) {
		err := db.Close()
		if err != nil {
			logger.WithError(err).Errorf("Failed to close the connection with RDS cluster endpoint %s", endpoint)
		}
	}

	return db, closeFunc, nil
}

func ensureDatabaseUserIsCreated(ctx context.Context, db SQLDatabaseManager, username, password string) error {
	query := fmt.Sprintf("SELECT 1 FROM pg_roles WHERE rolname='%s'", username)
	rows, err := db.QueryContext(ctx, query)
	if err != nil {
		return errors.Wrap(err, "failed to run database user check SQL command")
	}
	if rows.Next() {
		return nil
	}

	query = fmt.Sprintf("CREATE USER %s WITH PASSWORD '%s'", username, password)
	_, err = db.QueryContext(ctx, query)
	if err != nil {
		return errors.New("failed to run create user SQL command: error suppressed")
	}

	return nil
}

func ensureDatabaseIsCreated(ctx context.Context, db SQLDatabaseManager, databaseName string) error {
	query := fmt.Sprintf(`SELECT datname FROM pg_catalog.pg_database WHERE lower(datname) = lower('%s');`, databaseName)
	rows, err := db.QueryContext(ctx, query)
	if err != nil {
		return errors.Wrap(err, "failed to run database exists SQL command")
	}
	if rows.Next() {
		return nil
	}

	query = fmt.Sprintf(`CREATE DATABASE %s`, databaseName)
	_, err = db.QueryContext(ctx, query)
	if err != nil {
		return errors.Wrap(err, "failed to run create database SQL command")
	}

	return nil
}

func dropDatabaseIfExists(ctx context.Context, db SQLDatabaseManager, databaseName string) error {
	query := fmt.Sprintf("DROP DATABASE IF EXISTS %s", databaseName)
	_, err := db.QueryContext(ctx, query)
	if err != nil {
		return errors.Wrap(err, "failed to run drop database SQL command")
	}

	return nil
}

func createSchemaIfNotExists(ctx context.Context, db SQLDatabaseManager, schemaName, username string) error {
	query := fmt.Sprintf("CREATE SCHEMA IF NOT EXISTS %s AUTHORIZATION %s", schemaName, username)
	_, err := db.QueryContext(ctx, query)
	if err != nil {
		return errors.Wrap(err, "failed to run create schema SQL command")
	}

	return nil
}

func dropUserIfExists(ctx context.Context, db SQLDatabaseManager, username string) error {
	query := fmt.Sprintf("DROP USER IF EXISTS %s", username)
	_, err := db.QueryContext(ctx, query)
	if err != nil {
		return errors.Wrap(err, "failed to run drop user SQL command")
	}

	return nil
}

func dropSchemaIfExists(ctx context.Context, db SQLDatabaseManager, schemaName string) error {
	query := fmt.Sprintf("DROP SCHEMA IF EXISTS %s CASCADE", schemaName)
	_, err := db.QueryContext(ctx, query)
	if err != nil {
		return errors.Wrap(err, "failed to run drop schema SQL command")
	}

	return nil
}

func ensureDefaultTextSearchConfig(ctx context.Context, db SQLDatabaseManager, databaseName string) error {
	query := fmt.Sprintf(`ALTER DATABASE %s SET default_text_search_config TO "pg_catalog.english";`, databaseName)
	_, err := db.QueryContext(ctx, query)
	if err != nil {
		return errors.Wrap(err, "failed to run SQL command to set default_text_search_config to pg_catalog.english")
	}

	return nil
}
