package main

import "github.com/spf13/cobra"

type databaseMultiTenantListFlag struct {
	clusterFlags
	pagingFlags
	tableOptions

	vpcID        string
	databaseType string
}

func (flags *databaseMultiTenantListFlag) addFlags(cmd *cobra.Command) {
	flags.pagingFlags.addFlags(cmd)
	flags.tableOptions.addFlags(cmd)
	cmd.Flags().StringVar(&flags.vpcID, "vpc-id", "", "The VPC ID by which to filter multitenant databases.")
	cmd.Flags().StringVar(&flags.databaseType, "database-type", "", "The database type by which to filter multitenant databases.")
}

type databaseMultiTenantGetFlag struct {
	clusterFlags
	multitenantDatabaseID string
}

func (flags *databaseMultiTenantGetFlag) addFlags(cmd *cobra.Command) {
	cmd.Flags().StringVar(&flags.multitenantDatabaseID, "multitenant-database", "", "The id of the multitenant database to be fetched.")
	_ = cmd.MarkFlagRequired("multitenant-database")
}

type databaseMultiTenantUpdateFlagChanged struct {
	isMaxInstallationsChanged bool
}

func (flags *databaseMultiTenantUpdateFlagChanged) addFlags(cmd *cobra.Command) {
	flags.isMaxInstallationsChanged = cmd.Flags().Changed("max-installations-per-logical-db")
}

type databaseMultiTenantUpdateFlag struct {
	clusterFlags
	databaseMultiTenantUpdateFlagChanged
	multitenantDatabaseID             string
	maxInstallations                  int64
	isMaxInstallationsPerLogicalDBSet bool
}

func (flags *databaseMultiTenantUpdateFlag) addFlags(cmd *cobra.Command) {
	cmd.Flags().StringVar(&flags.multitenantDatabaseID, "multitenant-database", "", "The id of the multitenant database to be updated.")
	cmd.Flags().Int64Var(&flags.maxInstallations, "max-installations-per-logical-db", 10, "The maximum number of installations permitted in a single logical database (only applies to proxy databases).")
	_ = cmd.MarkFlagRequired("multitenant-database")
}

type databaseMultiTenantDeleteFlag struct {
	clusterFlags
	multitenantDatabaseID string
	force                 bool
}

func (flags *databaseMultiTenantDeleteFlag) addFlags(cmd *cobra.Command) {
	cmd.Flags().StringVar(&flags.multitenantDatabaseID, "multitenant-database", "", "The id of the multitenant database to delete.")
	cmd.Flags().BoolVar(&flags.force, "force", false, "Specifies whether to delete record even if database cluster exists.")
	_ = cmd.MarkFlagRequired("multitenant-database")
}

type databaseMultiTenantReportFlag struct {
	clusterFlags
	multitenantDatabaseID string
}

func (flags *databaseMultiTenantReportFlag) addFlags(cmd *cobra.Command) {
	cmd.Flags().StringVar(&flags.multitenantDatabaseID, "multitenant-database", "", "The id of the multitenant database to be fetched.")
	_ = cmd.MarkFlagRequired("multitenant-database")
}

type databaseLogicalListFlag struct {
	clusterFlags
	pagingFlags
	tableOptions
	multitenantDatabaseID string
}

func (flags *databaseLogicalListFlag) addFlags(cmd *cobra.Command) {
	flags.pagingFlags.addFlags(cmd)
	flags.tableOptions.addFlags(cmd)
	cmd.Flags().StringVar(&flags.multitenantDatabaseID, "multitenant-database-id", "", "The multitenant database ID by which to filter logical databases.")
}

type databaseLogicalGetFlag struct {
	clusterFlags
	logicalDatabaseID string
}

func (flags *databaseLogicalGetFlag) addFlags(cmd *cobra.Command) {
	cmd.Flags().StringVar(&flags.logicalDatabaseID, "logical-database", "", "The id of the logical database to be fetched.")
	_ = cmd.MarkFlagRequired("logical-database")
}

type databaseSchemaListFlag struct {
	clusterFlags
	pagingFlags
	tableOptions
	logicalDatabaseID string
	installationID    string
}

func (flags *databaseSchemaListFlag) addFlags(cmd *cobra.Command) {
	flags.pagingFlags.addFlags(cmd)
	flags.tableOptions.addFlags(cmd)
	cmd.Flags().StringVar(&flags.logicalDatabaseID, "logical-database-id", "", "The logical database ID by which to filter database schemas.")
	cmd.Flags().StringVar(&flags.installationID, "installation-id", "", "The installation ID by which to filter database schemas.")
}

type databaseSchemaGetFlag struct {
	clusterFlags
	databaseSchemaID string
}

func (flags *databaseSchemaGetFlag) addFlags(cmd *cobra.Command) {
	cmd.Flags().StringVar(&flags.databaseSchemaID, "database-schema", "", "The id of the database schema to be fetched.")
	_ = cmd.MarkFlagRequired("database-schema")
}
