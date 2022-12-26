package main

import "github.com/spf13/cobra"

type clusterInstallationGetFlags struct {
	clusterFlags
	clusterInstallationID string
}

func (flags *clusterInstallationGetFlags) addFlags(cmd *cobra.Command) {
	cmd.Flags().StringVar(&flags.clusterInstallationID, "cluster-installation", "", "The id of the cluster installation to be fetched.")
	_ = cmd.MarkFlagRequired("cluster-installation")
}

type clusterInstallationListFlags struct {
	clusterFlags
	pagingFlags
	tableOptions
	cluster      string
	installation string
}

func (flags *clusterInstallationListFlags) addFlags(cmd *cobra.Command) {
	flags.pagingFlags.addFlags(cmd)
	flags.tableOptions.addFlags(cmd)

	cmd.Flags().StringVar(&flags.cluster, "cluster", "", "The cluster by which to filter cluster installations.")
	cmd.Flags().StringVar(&flags.installation, "installation", "", "The installation by which to filter cluster installations.")
}

type clusterInstallationConfigGetFlags struct {
	clusterFlags
	clusterInstallationID string
}

func (flags *clusterInstallationConfigGetFlags) addFlags(cmd *cobra.Command) {
	cmd.Flags().StringVar(&flags.clusterInstallationID, "cluster-installation", "", "The id of the cluster installation.")
	_ = cmd.MarkFlagRequired("cluster-installation")
}

type clusterInstallationConfigSetFlags struct {
	clusterFlags
	clusterInstallationID string
	key                   string
	val                   string
}

func (flags *clusterInstallationConfigSetFlags) addFlags(cmd *cobra.Command) {
	cmd.Flags().StringVar(&flags.clusterInstallationID, "cluster-installation", "", "The id of the cluster installation.")
	cmd.Flags().StringVar(&flags.key, "key", "", "The configuration key to update (e.g. ServiceSettings.SiteURL).")
	cmd.Flags().StringVar(&flags.val, "value", "", "The value to write to the config.")

	_ = cmd.MarkFlagRequired("cluster-installation")
	_ = cmd.MarkFlagRequired("key")
	_ = cmd.MarkFlagRequired("value")
}

type clusterInstallationMMCTLFlags struct {
	clusterFlags
	clusterInstallationID string
	subcommand            string
}

func (flags *clusterInstallationMMCTLFlags) addFlags(cmd *cobra.Command) {
	cmd.Flags().StringVar(&flags.clusterInstallationID, "cluster-installation", "", "The id of the cluster installation.")
	cmd.Flags().StringVar(&flags.subcommand, "command", "", "The mmctl subcommand to run.")

	_ = cmd.MarkFlagRequired("cluster-installation")
	_ = cmd.MarkFlagRequired("command")
}

type clusterInstallationMattermostCLIFlags struct {
	clusterFlags
	clusterInstallationID string
	subcommand            string
}

func (flags *clusterInstallationMattermostCLIFlags) addFlags(cmd *cobra.Command) {

	cmd.Flags().StringVar(&flags.clusterInstallationID, "cluster-installation", "", "The id of the cluster installation.")
	cmd.Flags().StringVar(&flags.subcommand, "command", "", "The Mattermost CLI subcommand to run.")

	_ = cmd.MarkFlagRequired("cluster-installation")
	_ = cmd.MarkFlagRequired("command")
}

type clusterInstallationMigrationFlags struct {
	clusterFlags
	installation  string
	sourceCluster string
	targetCluster string
}

func (flags *clusterInstallationMigrationFlags) addFlags(cmd *cobra.Command) {
	cmd.Flags().StringVar(&flags.installation, "installation", "", "The specific installation ID to migrate from source cluster, default is ALL.")
	cmd.Flags().StringVar(&flags.sourceCluster, "source-cluster", "", "The source cluster for the migration to migrate cluster installations from.")
	cmd.Flags().StringVar(&flags.targetCluster, "target-cluster", "", "The target cluster for the migration to migrate cluster installation to.")

	_ = cmd.MarkFlagRequired("source-cluster")
	_ = cmd.MarkFlagRequired("target-cluster")
}

type clusterInstallationDNSMigrationFlags struct {
	clusterFlags
	installation     string
	sourceCluster    string
	targetCluster    string
	lockInstallation bool
}

func (flags *clusterInstallationDNSMigrationFlags) addFlags(cmd *cobra.Command) {
	cmd.Flags().StringVar(&flags.installation, "installation", "", "The specific installation ID to migrate from source cluster, default is ALL.")
	cmd.Flags().StringVar(&flags.sourceCluster, "source-cluster", "", "The source cluster for the migration to switch CNAME(s) from.")
	cmd.Flags().StringVar(&flags.targetCluster, "target-cluster", "", "The target cluster for the migration to switch CNAME to.")
	cmd.Flags().BoolVar(&flags.lockInstallation, "lock-installation", true, "The installation's lock flag during DNS migration process.")

	_ = cmd.MarkFlagRequired("source-cluster")
	_ = cmd.MarkFlagRequired("target-cluster")
}

type inActiveClusterInstallationDeleteFlags struct {
	clusterFlags
	cluster               string
	clusterInstallationID string
}

func (flags *inActiveClusterInstallationDeleteFlags) addFlags(cmd *cobra.Command) {
	cmd.Flags().StringVar(&flags.cluster, "cluster", "", "The cluster ID to delete stale cluster installations from.")
	cmd.Flags().StringVar(&flags.clusterInstallationID, "cluster-installation", "", "The id of the cluster installation.")
	_ = cmd.MarkFlagRequired("cluster")
}

type clusterRolesPostMigrationSwitchFlags struct {
	clusterFlags
	switchRole    string
	sourceCluster string
	targetCluster string
}

func (flags *clusterRolesPostMigrationSwitchFlags) addFlags(cmd *cobra.Command) {
	cmd.Flags().StringVar(&flags.sourceCluster, "source-cluster", "", "The source cluster to be mark as secondary cluster.")
	cmd.Flags().StringVar(&flags.targetCluster, "target-cluster", "", "The target cluster to be mark as primary cluster.")

	_ = cmd.MarkFlagRequired("source-cluster")
	_ = cmd.MarkFlagRequired("target-cluster")
}
