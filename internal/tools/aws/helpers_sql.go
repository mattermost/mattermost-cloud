// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.
//

package aws

import (
	"context"
	"fmt"

	"github.com/aws/aws-sdk-go/service/rds"
	"github.com/pkg/errors"
)

// MattermostMySQLConnStrings formats the connection string used for accessing a
// Mattermost database.
func MattermostMySQLConnStrings(schema, username, password string, dbCluster *rds.DBCluster) (string, string) {
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
//
// Regarding binary_parameters:
// https://blog.bullgare.com/2019/06/pgbouncer-and-prepared-statements
func MattermostPostgresConnStrings(schema, username, password string, dbCluster *rds.DBCluster) (string, string) {
	dbConnection := fmt.Sprintf("postgres://%s:%s@%s:5432/%s?connect_timeout=10&binary_parameters=yes",
		username, password, *dbCluster.Endpoint, schema)
	readReplicas := fmt.Sprintf("postgres://%s:%s@%s:5432/%s?connect_timeout=10&binary_parameters=yes",
		username, password, *dbCluster.ReaderEndpoint, schema)

	return dbConnection, readReplicas
}

// MattermostPostgresPGBouncerConnStrings formats the connection strings used by
// Mattermost servers to access a PostgreSQL database with a PGBouncer proxy.
//
// Regarding binary_parameters:
// https://blog.bullgare.com/2019/06/pgbouncer-and-prepared-statements
func MattermostPostgresPGBouncerConnStrings(username, password, database string) (string, string, string) {
	dbConnection := fmt.Sprintf("postgres://%s:%s@pgbouncer.pgbouncer:5432/%s?connect_timeout=10&sslmode=disable&binary_parameters=yes",
		username, password, database)
	readReplicas := fmt.Sprintf("postgres://%s:%s@pgbouncer.pgbouncer:5432/%s-ro?connect_timeout=10&sslmode=disable&binary_parameters=yes",
		username, password, database)
	connectionCheck := fmt.Sprintf("postgres://%s:%s@pgbouncer.pgbouncer:5432/%s?connect_timeout=10&sslmode=disable",
		username, password, database)

	return dbConnection, readReplicas, connectionCheck
}

// RDSPostgresConnString formats the connection string used by the provisioner
// for accessing a Postgres RDS cluster.
func RDSPostgresConnString(schema, endpoint, username, password string) string {
	return fmt.Sprintf("postgres://%s:%s@%s:5432/%s?connect_timeout=10",
		username, password, endpoint, schema)
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
