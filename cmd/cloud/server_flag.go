// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.
//

package main

import (
	"time"

	toolsAWS "github.com/mattermost/mattermost-cloud/internal/tools/aws"
	"github.com/spf13/cobra"
)

type supervisorOptions struct {
	disableAllSupervisors               bool
	clusterSupervisor                   bool
	groupSupervisor                     bool
	installationSupervisor              bool
	installationDeletionSupervisor      bool
	clusterInstallationSupervisor       bool
	backupSupervisor                    bool
	importSupervisor                    bool
	installationDBRestorationSupervisor bool
	installationDBMigrationSupervisor   bool

	installationDeletionPendingTime time.Duration
	installationDeletionMaxUpdating int64

	disableDNSUpdates bool
	awatAddress       string
}

func (flags *supervisorOptions) addFlags(cmd *cobra.Command) {
	cmd.PersistentFlags().BoolVar(&flags.disableAllSupervisors, "disable-all-supervisors", false, "disable all supervisors (API-only functionality)")

	cmd.PersistentFlags().BoolVar(&flags.clusterSupervisor, "cluster-supervisor", true, "Whether this server will run a cluster supervisor or not.")
	cmd.PersistentFlags().BoolVar(&flags.groupSupervisor, "group-supervisor", false, "Whether this server will run an installation group supervisor or not.")
	cmd.PersistentFlags().BoolVar(&flags.installationSupervisor, "installation-supervisor", true, "Whether this server will run an installation supervisor or not.")
	cmd.PersistentFlags().BoolVar(&flags.installationDeletionSupervisor, "installation-deletion-supervisor", true, "Whether this server will run a installation deletion supervisor or not. (slow-poll supervisor)")
	cmd.PersistentFlags().BoolVar(&flags.clusterInstallationSupervisor, "cluster-installation-supervisor", true, "Whether this server will run a cluster installation supervisor or not.")
	cmd.PersistentFlags().BoolVar(&flags.backupSupervisor, "backup-supervisor", false, "Whether this server will run a backup supervisor or not.")
	cmd.PersistentFlags().BoolVar(&flags.importSupervisor, "import-supervisor", false, "Whether this server will run a workspace import supervisor or not.")
	cmd.PersistentFlags().BoolVar(&flags.installationDBRestorationSupervisor, "installation-db-restoration-supervisor", false, "Whether this server will run an installation db restoration supervisor or not.")
	cmd.PersistentFlags().BoolVar(&flags.installationDBMigrationSupervisor, "installation-db-migration-supervisor", false, "Whether this server will run an installation db migration supervisor or not.")

	cmd.PersistentFlags().DurationVar(&flags.installationDeletionPendingTime, "installation-deletion-pending-time", 3*time.Minute, "The amount of time that installations will stay in the deletion queue before they are actually deleted. Set to 0 for immediate deletion.")
	cmd.PersistentFlags().Int64Var(&flags.installationDeletionMaxUpdating, "installation-deletion-max-updating", 25, "A soft limit on the number of installations that the provisioner will delete at one time from the group of deletion-pending installations.")
	cmd.PersistentFlags().BoolVar(&flags.disableDNSUpdates, "disable-dns-updates", false, "If set to true DNS updates will be disabled when updating Installations.")
	cmd.PersistentFlags().StringVar(&flags.awatAddress, "awat", "http://localhost:8077", "The location of the Automatic Workspace Archive Translator if the import supervisor is being used.")
}

type schedulingOptions struct {
	balancedInstallationScheduling     bool
	clusterResourceThresholdScaleValue int
	clusterResourceThreshold           int
	thresholdCPUOverride               int
	thresholdMemoryOverride            int
	thresholdPodCountOverride          int
}

func (flags *schedulingOptions) addFlags(cmd *cobra.Command) {
	cmd.PersistentFlags().BoolVar(&flags.balancedInstallationScheduling, "balanced-installation-scheduling", false, "Whether to schedule installations on the cluster with the greatest percentage of available resources or not. (slows down scheduling speed as cluster count increases)")
	cmd.PersistentFlags().IntVar(&flags.clusterResourceThresholdScaleValue, "cluster-resource-threshold-scale-value", 0, "The number of worker nodes to scale up by when the threshold is passed. Set to 0 for no scaling. Scaling will never exceed the cluster max worker configuration value.")
	cmd.PersistentFlags().IntVar(&flags.clusterResourceThreshold, "cluster-resource-threshold", 80, "The percent threshold where new installations won't be scheduled on a multi-tenant cluster.")
	cmd.PersistentFlags().IntVar(&flags.thresholdCPUOverride, "cluster-resource-threshold-cpu-override", 0, "The cluster-resource-threshold override value for CPU resources only")
	cmd.PersistentFlags().IntVar(&flags.thresholdMemoryOverride, "cluster-resource-threshold-memory-override", 0, "The cluster-resource-threshold override value for memory resources only")
	cmd.PersistentFlags().IntVar(&flags.thresholdPodCountOverride, "cluster-resource-threshold-pod-count-override", 0, "The cluster-resource-threshold override value for pod count only")
}

type provisioningParams struct {
	provisioner           string
	s3StateStore          string
	allowListCIDRRange    []string
	sloInstallationGroups []string
	vpnListCIDR           []string
	useExistingResources  bool
	deployMySQLOperator   bool
	deployMinioOperator   bool
	ndotsDefaultValue     string

	backupJobTTL           int32
	backupRestoreToolImage string

	etcdQuotaBackendBytes int
	etcdListenMetricsURL  string
}

func (flags *provisioningParams) addFlags(cmd *cobra.Command) {
	cmd.PersistentFlags().StringVar(&flags.provisioner, "provisioner", "kops", "Specifies which provisioner to use, one of: kops, eks.")
	cmd.PersistentFlags().StringVar(&flags.s3StateStore, "state-store", "dev.cloud.mattermost.com", "The S3 bucket used to store cluster state.")
	cmd.PersistentFlags().StringSliceVar(&flags.allowListCIDRRange, "allow-list-cidr-range", []string{"0.0.0.0/0"}, "The list of CIDRs to allow communication with the private ingress.")
	cmd.PersistentFlags().StringSliceVar(&flags.sloInstallationGroups, "slo-installation-groups", []string{}, "The list of installation group ids to create dedicated SLOs for.")
	cmd.PersistentFlags().StringSliceVar(&flags.vpnListCIDR, "vpn-list-cidr", []string{"0.0.0.0/0"}, "The list of VPN CIDRs to allow communication with the clusters.")
	cmd.PersistentFlags().BoolVar(&flags.useExistingResources, "use-existing-aws-resources", true, "Whether to use existing AWS resources (VPCs, subnets, etc.) or not.")
	cmd.PersistentFlags().BoolVar(&flags.deployMySQLOperator, "deploy-mysql-operator", true, "Whether to deploy the mysql operator.")
	cmd.PersistentFlags().BoolVar(&flags.deployMinioOperator, "deploy-minio-operator", true, "Whether to deploy the minio operator.")
	cmd.PersistentFlags().StringVar(&flags.ndotsDefaultValue, "ndots-value", "5", "The default ndots value for installations.")

	cmd.PersistentFlags().Int32Var(&flags.backupJobTTL, "backup-job-ttl-seconds", 3600, "Number of seconds after which finished backup jobs will be cleaned up. Set to negative value to not cleanup or 0 to cleanup immediately.")
	cmd.PersistentFlags().StringVar(&flags.backupRestoreToolImage, "backup-restore-tool-image", "mattermost/backup-restore-tool:latest", "Image of Backup Restore Tool to use.")

	cmd.PersistentFlags().IntVar(&flags.etcdQuotaBackendBytes, "etcd-quota-backend-bytes", 4294967296, "Raise alarms by cluster when backend size exceeds the given quota")
	cmd.PersistentFlags().StringVar(&flags.etcdListenMetricsURL, "etcd-listen-metrics-urls", "http://0.0.0.0:8081", "List of additional URL to listen for metrics")

}

type pgBouncerConfig struct {
	minPoolSize                   int
	defaultPoolSize               int
	reservePoolSize               int
	maxClientConnections          int
	maxDatabaseConnectionsPerPool int
	serverIdleTimeout             int
	serverLifetime                int
	serverResetQueryAlways        int
}

func (flags *pgBouncerConfig) addFlags(cmd *cobra.Command) {
	cmd.PersistentFlags().IntVar(&flags.minPoolSize, "min-proxy-db-pool-size", 1, "The db proxy min pool size.")
	cmd.PersistentFlags().IntVar(&flags.defaultPoolSize, "default-proxy-db-pool-size", 5, "The db proxy default pool size per user.")
	cmd.PersistentFlags().IntVar(&flags.reservePoolSize, "reserve-proxy-db-pool-size", 10, "The db proxy reserve pool size per logical database.")
	cmd.PersistentFlags().IntVar(&flags.maxClientConnections, "max-client-connections", 20000, "The db proxy max client connections.")
	cmd.PersistentFlags().IntVar(&flags.maxDatabaseConnectionsPerPool, "max-proxy-db-connections-per-pool", 20, "The maximum number of proxy database connections per pool (logical database).")
	cmd.PersistentFlags().IntVar(&flags.serverIdleTimeout, "server-idle-timeout", 30, "The server idle timeout.")
	cmd.PersistentFlags().IntVar(&flags.serverLifetime, "server-lifetime", 300, "The server lifetime.")
	cmd.PersistentFlags().IntVar(&flags.serverResetQueryAlways, "server-reset-query-always", 0, "Whether server_reset_query should be run in all pooling modes.")
}

type installationOptions struct {
	keepDatabaseData              bool
	keepFileStoreData             bool
	requireAnnotatedInstallations bool
	gitlabOAuthToken              string
	forceCRUpgrade                bool
	mattermostWebHook             string
	mattermostChannel             string
	utilitiesGitURL               string
}

func (flags *installationOptions) addFlags(cmd *cobra.Command) {
	cmd.PersistentFlags().BoolVar(&flags.keepDatabaseData, "keep-database-data", true, "Whether to preserve database data after installation deletion or not.")
	cmd.PersistentFlags().BoolVar(&flags.keepFileStoreData, "keep-filestore-data", true, "Whether to preserve filestore data after installation deletion or not.")
	cmd.PersistentFlags().BoolVar(&flags.requireAnnotatedInstallations, "require-annotated-installations", false, "Require new installations to have at least one annotation.")
	cmd.PersistentFlags().StringVar(&flags.gitlabOAuthToken, "gitlab-oauth", "", "If Helm charts are stored in a Gitlab instance that requires authentication, provide the token here and it will be automatically set in the environment.")
	cmd.PersistentFlags().BoolVar(&flags.forceCRUpgrade, "force-cr-upgrade", false, "If specified installation CRVersions will be updated to the latest version when supervised.")
	cmd.PersistentFlags().StringVar(&flags.mattermostWebHook, "mattermost-webhook", "", "Set to use a Mattermost webhook for spot instances termination notifications")
	cmd.PersistentFlags().StringVar(&flags.mattermostChannel, "mattermost-channel", "", "Set a mattermost channel for spot instances termination notifications")
	cmd.PersistentFlags().StringVar(&flags.utilitiesGitURL, "utilities-git-url", "", "The private git domain to use for utilities. For example https://gitlab.com")
}

type dbUtilizationSettings struct {
	pgbouncer          int
	postgres           int
	mysql              int
	disableDBInitCheck bool
}

func (flags *dbUtilizationSettings) addFlags(cmd *cobra.Command) {
	cmd.PersistentFlags().IntVar(&flags.pgbouncer, "max-installations-rds-postgres-pgbouncer", toolsAWS.DefaultRDSMultitenantPGBouncerDatabasePostgresCountLimit, "Max installations per DB cluster of type RDS Postgres PGbouncer")
	cmd.PersistentFlags().IntVar(&flags.postgres, "max-installations-rds-postgres", toolsAWS.DefaultRDSMultitenantDatabasePostgresCountLimit, "Max installations per DB cluster of type RDS Postgres")
	cmd.PersistentFlags().IntVar(&flags.mysql, "max-installations-rds-mysql", toolsAWS.DefaultRDSMultitenantDatabaseMySQLCountLimit, "Max installations per DB cluster of type RDS MySQL")
	cmd.PersistentFlags().BoolVar(&flags.disableDBInitCheck, "disable-db-init-check", false, "Whether to disable init container with database check.")
}

type serverFlagChanged struct {
	isDebugChanged             bool
	isKeepDatabaseDataChanged  bool
	isKeepFileStoreDataChanged bool
}

func (flags *serverFlagChanged) addFlags(cmd *cobra.Command) {
	flags.isDebugChanged = cmd.Flags().Changed("debug")
	flags.isKeepDatabaseDataChanged = cmd.PersistentFlags().Changed("keep-database-data")
	flags.isKeepFileStoreDataChanged = cmd.PersistentFlags().Changed("keep-filestore-data")
}

type serverFlags struct {
	supervisorOptions
	schedulingOptions
	provisioningParams
	pgBouncerConfig
	installationOptions
	dbUtilizationSettings
	serverFlagChanged

	listen      string
	metricsPort int

	debug       bool
	debugHelm   bool
	devMode     bool
	machineLogs bool

	database      string
	maxSchemas    int64
	enableRoute53 bool
	kubecostToken string

	poll     int
	slowPoll int

	sloTargetAvailability float64
}

func (flags *serverFlags) addFlags(cmd *cobra.Command) {
	flags.supervisorOptions.addFlags(cmd)
	flags.schedulingOptions.addFlags(cmd)
	flags.provisioningParams.addFlags(cmd)
	flags.pgBouncerConfig.addFlags(cmd)
	flags.installationOptions.addFlags(cmd)
	flags.dbUtilizationSettings.addFlags(cmd)

	cmd.PersistentFlags().StringVar(&flags.listen, "listen", ":8075", "The interface and port on which to listen.")
	cmd.PersistentFlags().IntVar(&flags.metricsPort, "metrics-port", 8076, "Port on which the metrics server should be listening.")

	cmd.PersistentFlags().BoolVar(&flags.debug, "debug", false, "Whether to output debug logs.")
	cmd.PersistentFlags().BoolVar(&flags.debugHelm, "debug-helm", false, "Whether to include Helm output in debug logs.")
	cmd.PersistentFlags().BoolVar(&flags.devMode, "dev", false, "Set sane defaults for development")
	cmd.PersistentFlags().BoolVar(&flags.machineLogs, "machine-readable-logs", false, "Output the logs in machine readable format.")

	cmd.PersistentFlags().StringVar(&flags.database, "database", "sqlite://cloud.db", "The database backing the provisioning server.")
	cmd.PersistentFlags().Int64Var(&flags.maxSchemas, "default-max-schemas-per-logical-database", 10, "When importing and creating new proxy multitenant databases, this value is used for MaxInstallationsPerLogicalDatabase.")
	cmd.PersistentFlags().BoolVar(&flags.enableRoute53, "installation-enable-route53", false, "Specifies whether CNAME records for Installation should be created in Route53 as well.")
	cmd.PersistentFlags().StringVar(&flags.kubecostToken, "kubecost-token", "", "Set a kubecost token")

	cmd.PersistentFlags().IntVar(&flags.poll, "poll", 30, "The interval in seconds to poll for background work.")
	cmd.PersistentFlags().IntVar(&flags.slowPoll, "slow-poll", 60, "The interval in seconds to poll for background work for supervisors that are not time sensitive (slow-poll supervisors).")

	cmd.PersistentFlags().Float64Var(&flags.sloTargetAvailability, "slo-target-availability", 99.5, "The default SLOs availability when provisioning clusters")
}
