package main

import "github.com/spf13/cobra"

type clusterInstallationGetFlags struct {
	clusterFlags
	clusterInstallationID string
}

func (flags *clusterInstallationGetFlags) addFlags(command *cobra.Command) {
	command.Flags().StringVar(&flags.clusterInstallationID, "cluster-installation", "", "The id of the cluster installation to be fetched.")
	_ = command.MarkFlagRequired("cluster-installation")
}

type clusterInstallationListFlags struct {
	clusterFlags
	pagingFlags
	tableOptions
	cluster      string
	installation string
}

func (flags *clusterInstallationListFlags) addFlags(command *cobra.Command) {
	flags.pagingFlags.addFlags(command)
	flags.tableOptions.addFlags(command)

	command.Flags().StringVar(&flags.cluster, "cluster", "", "The cluster by which to filter cluster installations.")
	command.Flags().StringVar(&flags.installation, "installation", "", "The installation by which to filter cluster installations.")
}

type clusterInstallationConfigGetFlags struct {
	clusterFlags
	clusterInstallationID string
}

func (flags *clusterInstallationConfigGetFlags) addFlags(command *cobra.Command) {
	command.Flags().StringVar(&flags.clusterInstallationID, "cluster-installation", "", "The id of the cluster installation.")
	_ = command.MarkFlagRequired("cluster-installation")
}

type clusterInstallationConfigSetFlags struct {
	clusterFlags
	clusterInstallationID string
	key                   string
	val                   string
}

func (flags *clusterInstallationConfigSetFlags) addFlags(command *cobra.Command) {
	command.Flags().StringVar(&flags.clusterInstallationID, "cluster-installation", "", "The id of the cluster installation.")
	command.Flags().StringVar(&flags.key, "key", "", "The configuration key to update (e.g. ServiceSettings.SiteURL).")
	command.Flags().StringVar(&flags.val, "value", "", "The value to write to the config.")

	_ = command.MarkFlagRequired("cluster-installation")
	_ = command.MarkFlagRequired("key")
	_ = command.MarkFlagRequired("value")
}

type clusterInstallationMMCTLFlags struct {
	clusterFlags
	clusterInstallationID string
	subcommand            string
}

func (flags *clusterInstallationMMCTLFlags) addFlags(command *cobra.Command) {
	command.Flags().StringVar(&flags.clusterInstallationID, "cluster-installation", "", "The id of the cluster installation.")
	command.Flags().StringVar(&flags.subcommand, "command", "", "The mmctl subcommand to run.")

	_ = command.MarkFlagRequired("cluster-installation")
	_ = command.MarkFlagRequired("command")
}

type clusterInstallationMattermostCLIFlags struct {
	clusterFlags
	clusterInstallationID string
	subcommand            string
}

func (flags *clusterInstallationMattermostCLIFlags) addFlags(command *cobra.Command) {

	command.Flags().StringVar(&flags.clusterInstallationID, "cluster-installation", "", "The id of the cluster installation.")
	command.Flags().StringVar(&flags.subcommand, "command", "", "The Mattermost CLI subcommand to run.")

	_ = command.MarkFlagRequired("cluster-installation")
	_ = command.MarkFlagRequired("command")
}

type clusterInstallationMigrationFlags struct {
	clusterFlags
	installation  string
	sourceCluster string
	targetCluster string
}

func (flags *clusterInstallationMigrationFlags) addFlags(command *cobra.Command) {
	command.Flags().StringVar(&flags.installation, "installation", "", "The specific installation ID to migrate from source cluster, default is ALL.")
	command.Flags().StringVar(&flags.sourceCluster, "source-cluster", "", "The source cluster for the migration to migrate cluster installations from.")
	command.Flags().StringVar(&flags.targetCluster, "target-cluster", "", "The target cluster for the migration to migrate cluster installation to.")

	_ = command.MarkFlagRequired("source-cluster")
	_ = command.MarkFlagRequired("target-cluster")
}

type clusterInstallationDNSMigrationFlags struct {
	clusterFlags
	installation     string
	sourceCluster    string
	targetCluster    string
	lockInstallation bool
}

func (flags *clusterInstallationDNSMigrationFlags) addFlags(command *cobra.Command) {
	command.Flags().StringVar(&flags.installation, "installation", "", "The specific installation ID to migrate from source cluster, default is ALL.")
	command.Flags().StringVar(&flags.sourceCluster, "source-cluster", "", "The source cluster for the migration to switch CNAME(s) from.")
	command.Flags().StringVar(&flags.targetCluster, "target-cluster", "", "The target cluster for the migration to switch CNAME to.")
	command.Flags().BoolVar(&flags.lockInstallation, "lock-installation", true, "The installation's lock flag during DNS migration process.")

	_ = command.MarkFlagRequired("source-cluster")
	_ = command.MarkFlagRequired("target-cluster")
}

type inActiveClusterInstallationDeleteFlags struct {
	clusterFlags
	cluster               string
	clusterInstallationID string
}

func (flags *inActiveClusterInstallationDeleteFlags) addFlags(command *cobra.Command) {
	command.Flags().StringVar(&flags.cluster, "cluster", "", "The cluster ID to delete stale cluster installations from.")
	command.Flags().StringVar(&flags.clusterInstallationID, "cluster-installation", "", "The id of the cluster installation.")
	_ = command.MarkFlagRequired("cluster")
}

type clusterRolesPostMigrationSwitchFlags struct {
	clusterFlags
	switchRole    string
	sourceCluster string
	targetCluster string
}

func (flags *clusterRolesPostMigrationSwitchFlags) addFlags(command *cobra.Command) {
	command.Flags().StringVar(&flags.sourceCluster, "source-cluster", "", "The source cluster to be mark as secondary cluster.")
	command.Flags().StringVar(&flags.targetCluster, "target-cluster", "", "The target cluster to be mark as primary cluster.")

	_ = command.MarkFlagRequired("source-cluster")
	_ = command.MarkFlagRequired("target-cluster")
}
