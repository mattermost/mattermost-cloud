// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.
//

package main

import "github.com/spf13/cobra"

type databaseMultiTenantListFlag struct {
	clusterFlags
	pagingFlags
	tableOptions

	vpcID        string
	databaseType string
}

func (flags *databaseMultiTenantListFlag) addFlags(command *cobra.Command) {
	flags.pagingFlags.addFlags(command)
	flags.tableOptions.addFlags(command)
	command.Flags().StringVar(&flags.vpcID, "vpc-id", "", "The VPC ID by which to filter multitenant databases.")
	command.Flags().StringVar(&flags.databaseType, "database-type", "", "The database type by which to filter multitenant databases.")
}

type databaseMultiTenantGetFlag struct {
	clusterFlags
	multitenantDatabaseID string
}

func (flags *databaseMultiTenantGetFlag) addFlags(command *cobra.Command) {
	command.Flags().StringVar(&flags.multitenantDatabaseID, "multitenant-database", "", "The id of the multitenant database to be fetched.")
	_ = command.MarkFlagRequired("multitenant-database")
}

type databaseMultiTenantUpdateFlagChanged struct {
	isMaxInstallationsChanged bool
}

func (flags *databaseMultiTenantUpdateFlagChanged) addFlags(command *cobra.Command) {
	flags.isMaxInstallationsChanged = command.Flags().Changed("max-installations-per-logical-db")
}

type databaseMultiTenantUpdateFlag struct {
	clusterFlags
	databaseMultiTenantUpdateFlagChanged
	multitenantDatabaseID string
	maxInstallations      int64
}

func (flags *databaseMultiTenantUpdateFlag) addFlags(command *cobra.Command) {
	command.Flags().StringVar(&flags.multitenantDatabaseID, "multitenant-database", "", "The id of the multitenant database to be updated.")
	command.Flags().Int64Var(&flags.maxInstallations, "max-installations-per-logical-db", 10, "The maximum number of installations permitted in a single logical database (only applies to proxy databases).")
	_ = command.MarkFlagRequired("multitenant-database")
}

type databaseMultiTenantDeleteFlag struct {
	clusterFlags
	multitenantDatabaseID string
	force                 bool
}

func (flags *databaseMultiTenantDeleteFlag) addFlags(command *cobra.Command) {
	command.Flags().StringVar(&flags.multitenantDatabaseID, "multitenant-database", "", "The id of the multitenant database to delete.")
	command.Flags().BoolVar(&flags.force, "force", false, "Specifies whether to delete record even if database cluster exists.")
	_ = command.MarkFlagRequired("multitenant-database")
}

type databaseMultiTenantReportFlag struct {
	clusterFlags
	multitenantDatabaseID string
	includeSchemaCounts   bool
}

func (flags *databaseMultiTenantReportFlag) addFlags(command *cobra.Command) {
	command.Flags().StringVar(&flags.multitenantDatabaseID, "multitenant-database", "", "The id of the multitenant database to be fetched.")
	command.Flags().BoolVar(&flags.includeSchemaCounts, "include-schema-counts", false, "Whether to include schema counts for each logical database or not.")
	_ = command.MarkFlagRequired("multitenant-database")
}

type databaseLogicalListFlag struct {
	clusterFlags
	pagingFlags
	tableOptions
	multitenantDatabaseID string
}

func (flags *databaseLogicalListFlag) addFlags(command *cobra.Command) {
	flags.pagingFlags.addFlags(command)
	flags.tableOptions.addFlags(command)
	command.Flags().StringVar(&flags.multitenantDatabaseID, "multitenant-database-id", "", "The multitenant database ID by which to filter logical databases.")
}

type databaseLogicalGetFlag struct {
	clusterFlags
	logicalDatabaseID string
}

func (flags *databaseLogicalGetFlag) addFlags(command *cobra.Command) {
	command.Flags().StringVar(&flags.logicalDatabaseID, "logical-database", "", "The id of the logical database to be fetched.")
	_ = command.MarkFlagRequired("logical-database")
}

type databaseLogicalDeleteFlag struct {
	clusterFlags
	logicalDatabaseID string
}

func (flags *databaseLogicalDeleteFlag) addFlags(command *cobra.Command) {
	command.Flags().StringVar(&flags.logicalDatabaseID, "logical-database", "", "The id of the logical database to delete.").Required()
}

type databaseSchemaListFlag struct {
	clusterFlags
	pagingFlags
	tableOptions
	logicalDatabaseID string
	installationID    string
}

func (flags *databaseSchemaListFlag) addFlags(command *cobra.Command) {
	flags.pagingFlags.addFlags(command)
	flags.tableOptions.addFlags(command)
	command.Flags().StringVar(&flags.logicalDatabaseID, "logical-database-id", "", "The logical database ID by which to filter database schemas.")
	command.Flags().StringVar(&flags.installationID, "installation-id", "", "The installation ID by which to filter database schemas.")
}

type databaseSchemaGetFlag struct {
	clusterFlags
	databaseSchemaID string
}

func (flags *databaseSchemaGetFlag) addFlags(command *cobra.Command) {
	command.Flags().StringVar(&flags.databaseSchemaID, "database-schema", "", "The id of the database schema to be fetched.")
	_ = command.MarkFlagRequired("database-schema")
}
