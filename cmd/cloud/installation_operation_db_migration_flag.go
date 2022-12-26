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

func (flags *installationDBMigrationRequestFlags) addFlags(cmd *cobra.Command) {
	cmd.Flags().StringVar(&flags.installationID, "installation", "", "The id of the installation to be migrated.")
	cmd.Flags().StringVar(&flags.destinationDB, "destination-db", model.InstallationDatabaseMultiTenantRDSPostgres, "The destination database type.")
	cmd.Flags().StringVar(&flags.multiTenantDBID, "multi-tenant-db", "", "The id of the destination multi tenant db.")
	_ = cmd.MarkFlagRequired("installation")
	_ = cmd.MarkFlagRequired("multi-tenant-db")
}

type installationDBMigrationsListFlags struct {
	clusterFlags
	pagingFlags
	tableOptions
	installationID string
	state          string
}

func (flags *installationDBMigrationsListFlags) addFlags(cmd *cobra.Command) {
	cmd.Flags().StringVar(&flags.installationID, "installation", "", "The id of the installation to query operations.")
	cmd.Flags().StringVar(&flags.state, "state", "", "The state to filter operations by.")
	flags.pagingFlags.addFlags(cmd)
	flags.tableOptions.addFlags(cmd)
}

type installationDBMigrationGetFlags struct {
	clusterFlags
	dbMigrationID string
}

func (flags *installationDBMigrationGetFlags) addFlags(cmd *cobra.Command) {
	cmd.Flags().StringVar(&flags.dbMigrationID, "db-migration", "", "The id of the installation db migration operation.")
	_ = cmd.MarkFlagRequired("db-migration")
}

type installationDBMigrationCommitFlags struct {
	clusterFlags
	dbMigrationID string
}

func (flags *installationDBMigrationCommitFlags) addFlags(cmd *cobra.Command) {
	cmd.Flags().StringVar(&flags.dbMigrationID, "db-migration", "", "The id of the installation db migration operation.")
	_ = cmd.MarkFlagRequired("db-migration")
}

type installationDBMigrationRollbackFlags struct {
	clusterFlags
	dbMigrationID string
}

func (flags *installationDBMigrationRollbackFlags) addFlags(cmd *cobra.Command) {
	cmd.Flags().StringVar(&flags.dbMigrationID, "db-migration", "", "The id of the installation db migration operation.")
	_ = cmd.MarkFlagRequired("db-migration")
}
