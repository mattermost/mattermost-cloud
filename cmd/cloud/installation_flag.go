package main

import (
	"github.com/mattermost/mattermost-cloud/model"
	"github.com/spf13/cobra"
)

type installationCreateRequestOptions struct {
	name                      string
	ownerID                   string
	groupID                   string
	version                   string
	image                     string
	size                      string
	license                   string
	affinity                  string
	database                  string
	filestore                 string
	mattermostEnv             []string
	priorityEnv               []string
	annotations               []string
	groupSelectionAnnotations []string
}

func (flags *installationCreateRequestOptions) addFlags(cmd *cobra.Command) {
	cmd.Flags().StringVar(&flags.name, "name", "", "Unique human-readable installation name. It should be the same as first segment of domain name.")
	cmd.Flags().StringVar(&flags.ownerID, "owner", "", "An opaque identifier describing the owner of the installation.")
	cmd.Flags().StringVar(&flags.groupID, "group", "", "The id of the group to join")
	cmd.Flags().StringVar(&flags.version, "version", "stable", "The Mattermost version to install.")
	cmd.Flags().StringVar(&flags.image, "image", "mattermost/mattermost-enterprise-edition", "The Mattermost container image to use.")
	cmd.Flags().StringVar(&flags.size, "size", model.InstallationDefaultSize, "The size of the installation. Accepts 100users, 1000users, 5000users, 10000users, 25000users, miniSingleton, or miniHA. Defaults to 100users.")
	cmd.Flags().StringVar(&flags.license, "license", "", "The Mattermost License to use in the server.")
	cmd.Flags().StringVar(&flags.affinity, "affinity", model.InstallationAffinityIsolated, "How other installations may be co-located in the same cluster.")
	cmd.Flags().StringVar(&flags.database, "database", model.InstallationDatabaseMysqlOperator, "The Mattermost server database type. Accepts mysql-operator, aws-rds, aws-rds-postgres, aws-multitenant-rds, or aws-multitenant-rds-postgres")
	cmd.Flags().StringVar(&flags.filestore, "filestore", model.InstallationFilestoreMinioOperator, "The Mattermost server filestore type. Accepts minio-operator, aws-s3, bifrost, or aws-multitenant-s3")
	cmd.Flags().StringArrayVar(&flags.mattermostEnv, "mattermost-env", []string{}, "Env vars to add to the Mattermost App. Accepts format: KEY_NAME=VALUE. Use the flag multiple times to set multiple env vars.")
	cmd.Flags().StringArrayVar(&flags.priorityEnv, "priority-env", []string{}, "Env vars to add to the Mattermost App that take priority over group config. Accepts format: KEY_NAME=VALUE. Use the flag multiple times to set multiple env vars.")
	cmd.Flags().StringArrayVar(&flags.annotations, "annotation", []string{}, "Additional annotations for the installation. Accepts multiple values, for example: '... --annotation abc --annotation def'")
	cmd.Flags().StringArrayVar(&flags.groupSelectionAnnotations, "group-selection-annotation", []string{}, "Annotations for automatic group selection. Accepts multiple values, for example: '... --group-selection-annotation abc --group-selection-annotation def'")

	_ = cmd.MarkFlagRequired("owner")
}

type rdsOptions struct {
	rdsPrimaryInstance string
	rdsReplicaInstance string
	rdsReplicasCount   int
}

func (flags *rdsOptions) addFlags(cmd *cobra.Command) {
	cmd.Flags().StringVar(&flags.rdsPrimaryInstance, "rds-primary-instance", "", "The machine instance type used for primary replica of database cluster. Works only with single tenant RDS databases.")
	cmd.Flags().StringVar(&flags.rdsReplicaInstance, "rds-replica-instance", "", "The machine instance type used for reader replicas of database cluster. Works only with single tenant RDS databases.")
	cmd.Flags().IntVar(&flags.rdsReplicasCount, "rds-replicas-count", 0, "The number of reader replicas of database cluster. Min: 0, Max: 15. Works only with single tenant RDS databases.")
}

type installationCreateFlags struct {
	clusterFlags
	installationCreateRequestOptions
	rdsOptions
	dns                        []string
	externalDatabaseSecretName string
}

func (flags *installationCreateFlags) addFlags(cmd *cobra.Command) {
	flags.installationCreateRequestOptions.addFlags(cmd)
	flags.rdsOptions.addFlags(cmd)

	cmd.Flags().StringSliceVar(&flags.dns, "dns", []string{}, "URLs at which the Mattermost server will be available.")
	cmd.Flags().StringVar(&flags.externalDatabaseSecretName, "external-database-secret-name", "", "The AWS secret name where the external database DSN is stored. Works only with external databases.")

	_ = cmd.MarkFlagRequired("dns")
}

type installationPatchRequestOptions struct {
	ownerID            string
	version            string
	image              string
	size               string
	license            string
	mattermostEnv      []string
	mattermostEnvClear bool
}

func (flags *installationPatchRequestOptions) addFlags(cmd *cobra.Command) {
	cmd.Flags().StringVar(&flags.ownerID, "owner", "", "The new owner value of this installation.")
	cmd.Flags().StringVar(&flags.version, "version", "stable", "The Mattermost version to target.")
	cmd.Flags().StringVar(&flags.image, "image", "mattermost/mattermost-enterprise-edition", "The Mattermost container image to use.")
	cmd.Flags().StringVar(&flags.size, "size", model.InstallationDefaultSize, "The size of the installation. Accepts 100users, 1000users, 5000users, 10000users, 25000users, miniSingleton, or miniHA. Defaults to 100users.")
	cmd.Flags().StringVar(&flags.license, "license", "", "The Mattermost License to use in the server.")

	cmd.Flags().StringArrayVar(&flags.mattermostEnv, "mattermost-env", []string{}, "Env vars to add to the Mattermost App. Accepts format: KEY_NAME=VALUE. Use the flag multiple times to set multiple env vars.")
	cmd.Flags().BoolVar(&flags.mattermostEnvClear, "mattermost-env-clear", false, "Clears all env var data.")

}

type installationUpdateFlags struct {
	clusterFlags
	installationPatchRequestOptions
	priorityEnv      []string
	priorityEnvClear bool
	installationID   string
}

func (flags *installationUpdateFlags) addFlags(cmd *cobra.Command) {
	flags.installationPatchRequestOptions.addFlags(cmd)

	cmd.Flags().StringVar(&flags.installationID, "installation", "", "The id of the installation to be updated.")
	cmd.Flags().StringArrayVar(&flags.priorityEnv, "priority-env", []string{}, "Env vars to add to the Mattermost App that take priority over group config. Accepts format: KEY_NAME=VALUE. Use the flag multiple times to set multiple env vars.")
	cmd.Flags().BoolVar(&flags.priorityEnvClear, "priority-env-clear", false, "Clears all priority env var data.")

	_ = cmd.MarkFlagRequired("installation")
}

type installationDeleteFlags struct {
	clusterFlags
	installationID string
}

func (flags *installationDeleteFlags) addFlags(cmd *cobra.Command) {
	cmd.Flags().StringVar(&flags.installationID, "installation", "", "The id of the installation to be deleted.")
	_ = cmd.MarkFlagRequired("installation")
}

type installationCancelDeletionFlags struct {
	clusterFlags
	installationID string
}

func (flags *installationCancelDeletionFlags) addFlags(cmd *cobra.Command) {
	cmd.Flags().StringVar(&flags.installationID, "installation", "", "The id of the installation to cancel pending deletion for.")
	_ = cmd.MarkFlagRequired("installation")
}

type installationHibernateFlags struct {
	clusterFlags
	installationID string
}

func (flags *installationHibernateFlags) addFlags(cmd *cobra.Command) {
	cmd.Flags().StringVar(&flags.installationID, "installation", "", "The id of the installation to put into hibernation.")
	_ = cmd.MarkFlagRequired("installation")
}

type installationWakeupFlags struct {
	clusterFlags
	installationPatchRequestOptions
	installationID string
}

func (flags *installationWakeupFlags) addFlags(cmd *cobra.Command) {
	flags.installationPatchRequestOptions.addFlags(cmd)

	cmd.Flags().StringVar(&flags.installationID, "installation", "", "The id of the installation to wake up from hibernation.")
	_ = cmd.MarkFlagRequired("installation")
}

type installationGetFlags struct {
	clusterFlags
	installationID              string
	includeGroupConfig          bool
	includeGroupConfigOverrides bool
	hideLicense                 bool
	hideEnv                     bool
}

func (flags *installationGetFlags) addFlags(cmd *cobra.Command) {
	cmd.Flags().StringVar(&flags.installationID, "installation", "", "The id of the installation to be fetched.")
	cmd.Flags().BoolVar(&flags.includeGroupConfig, "include-group-config", true, "Whether to include group configuration in the installation or not.")
	cmd.Flags().BoolVar(&flags.includeGroupConfigOverrides, "include-group-config-overrides", true, "Whether to include a group configuration override summary in the installation or not.")
	cmd.Flags().BoolVar(&flags.hideLicense, "hide-license", true, "Whether to hide the license value in the output or not.")
	cmd.Flags().BoolVar(&flags.hideEnv, "hide-env", true, "Whether to hide env vars in the output or not.")

	_ = cmd.MarkFlagRequired("installation")
}

type installationGetRequestOptions struct {
	owner                       string
	group                       string
	state                       string
	dns                         string
	includeGroupConfig          bool
	includeGroupConfigOverrides bool
}

func (flags *installationGetRequestOptions) addFlags(cmd *cobra.Command) {
	cmd.Flags().StringVar(&flags.owner, "owner", "", "The owner ID to filter installations by.")
	cmd.Flags().StringVar(&flags.group, "group", "", "The group ID to filter installations.")
	cmd.Flags().StringVar(&flags.state, "state", "", "The state to filter installations by.")
	cmd.Flags().StringVar(&flags.dns, "dns", "", "The dns name to filter installations by.")
	cmd.Flags().BoolVar(&flags.includeGroupConfig, "include-group-config", true, "Whether to include group configuration in the installations or not.")
	cmd.Flags().BoolVar(&flags.includeGroupConfigOverrides, "include-group-config-overrides", true, "Whether to include a group configuration override summary in the installations or not.")

}

type installationListFlags struct {
	clusterFlags
	installationGetRequestOptions
	pagingFlags
	tableOptions
	hideLicense bool
	hideEnv     bool
}

func (flags *installationListFlags) addFlags(cmd *cobra.Command) {
	flags.installationGetRequestOptions.addFlags(cmd)
	flags.pagingFlags.addFlags(cmd)
	flags.tableOptions.addFlags(cmd)

	cmd.Flags().BoolVar(&flags.hideLicense, "hide-license", true, "Whether to hide the license value in the output or not.")
	cmd.Flags().BoolVar(&flags.hideEnv, "hide-env", true, "Whether to hide env vars in the output or not.")
}

type installationRecoveryFlags struct {
	clusterFlags
	installationID string
	databaseID     string
	database       string
}

func (flags *installationRecoveryFlags) addFlags(cmd *cobra.Command) {
	cmd.Flags().StringVar(&flags.installationID, "installation", "", "The id of the installation to be recovered.")
	cmd.Flags().StringVar(&flags.databaseID, "installation-database", "", "The original multitenant database id of the installation to be recovered.")
	cmd.Flags().StringVar(&flags.database, "database", "sqlite://cloud.db", "The database backing the provisioning server.")

	_ = cmd.MarkFlagRequired("installation")
	_ = cmd.MarkFlagRequired("installation-database")
}

type installationDeploymentReportFlags struct {
	clusterFlags
	installationID string
	eventCount     int
}

func (flags *installationDeploymentReportFlags) addFlags(cmd *cobra.Command) {
	cmd.Flags().StringVar(&flags.installationID, "installation", "", "The id of the installation to report on.")
	cmd.Flags().IntVar(&flags.eventCount, "event-count", 10, "The number of recent installation events to include in the report.")

	_ = cmd.MarkFlagRequired("installation")
}
