// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.
//

package main

import (
	"github.com/mattermost/mattermost-cloud/model"
	"github.com/spf13/cobra"
)

type installationDBMigrationRequestFlags struct {
	clusterFlags
	installationID  string
	destinationDB   string
	multiTenantDBID string
}

func (flags *installationDBMigrationRequestFlags) addFlags(command *cobra.Command) {
	command.Flags().StringVar(&flags.installationID, "installation", "", "The id of the installation to be migrated.")
	command.Flags().StringVar(&flags.destinationDB, "destination-db", model.InstallationDatabaseMultiTenantRDSPostgres, "The destination database type.")
	command.Flags().StringVar(&flags.multiTenantDBID, "multi-tenant-db", "", "The id of the destination multi tenant db.")
	_ = command.MarkFlagRequired("installation")
	_ = command.MarkFlagRequired("multi-tenant-db")
}

type installationDBMigrationsListFlags struct {
	clusterFlags
	pagingFlags
	tableOptions
	installationID string
	state          string
}

func (flags *installationDBMigrationsListFlags) addFlags(command *cobra.Command) {
	command.Flags().StringVar(&flags.installationID, "installation", "", "The id of the installation to query operations.")
	command.Flags().StringVar(&flags.state, "state", "", "The state to filter operations by.")
	flags.pagingFlags.addFlags(command)
	flags.tableOptions.addFlags(command)
}

type installationDBMigrationGetFlags struct {
	clusterFlags
	dbMigrationID string
}

func (flags *installationDBMigrationGetFlags) addFlags(command *cobra.Command) {
	command.Flags().StringVar(&flags.dbMigrationID, "db-migration", "", "The id of the installation db migration operation.")
	_ = command.MarkFlagRequired("db-migration")
}

type installationDBMigrationCommitFlags struct {
	clusterFlags
	dbMigrationID string
}

func (flags *installationDBMigrationCommitFlags) addFlags(command *cobra.Command) {
	command.Flags().StringVar(&flags.dbMigrationID, "db-migration", "", "The id of the installation db migration operation.")
	_ = command.MarkFlagRequired("db-migration")
}

type installationDBMigrationRollbackFlags struct {
	clusterFlags
	dbMigrationID string
}

func (flags *installationDBMigrationRollbackFlags) addFlags(command *cobra.Command) {
	command.Flags().StringVar(&flags.dbMigrationID, "db-migration", "", "The id of the installation db migration operation.")
	_ = command.MarkFlagRequired("db-migration")
}
