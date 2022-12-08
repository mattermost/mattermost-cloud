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

func (flags *supervisorOptions) addFlags(command *cobra.Command) {
	command.PersistentFlags().BoolVar(&flags.disableAllSupervisors, "disable-all-supervisors", false, "disable all supervisors (API-only functionality)")

	command.PersistentFlags().BoolVar(&flags.clusterSupervisor, "cluster-supervisor", true, "Whether this server will run a cluster supervisor or not.")
	command.PersistentFlags().BoolVar(&flags.groupSupervisor, "group-supervisor", false, "Whether this server will run an installation group supervisor or not.")
	command.PersistentFlags().BoolVar(&flags.installationSupervisor, "installation-supervisor", true, "Whether this server will run an installation supervisor or not.")
	command.PersistentFlags().BoolVar(&flags.installationDeletionSupervisor, "installation-deletion-supervisor", true, "Whether this server will run a installation deletion supervisor or not. (slow-poll supervisor)")
	command.PersistentFlags().BoolVar(&flags.clusterInstallationSupervisor, "cluster-installation-supervisor", true, "Whether this server will run a cluster installation supervisor or not.")
	command.PersistentFlags().BoolVar(&flags.backupSupervisor, "backup-supervisor", false, "Whether this server will run a backup supervisor or not.")
	command.PersistentFlags().BoolVar(&flags.importSupervisor, "import-supervisor", false, "Whether this server will run a workspace import supervisor or not.")
	command.PersistentFlags().BoolVar(&flags.installationDBRestorationSupervisor, "installation-db-restoration-supervisor", false, "Whether this server will run an installation db restoration supervisor or not.")
	command.PersistentFlags().BoolVar(&flags.installationDBMigrationSupervisor, "installation-db-migration-supervisor", false, "Whether this server will run an installation db migration supervisor or not.")

	command.PersistentFlags().DurationVar(&flags.installationDeletionPendingTime, "installation-deletion-pending-time", 3*time.Minute, "The amount of time that installations will stay in the deletion queue before they are actually deleted. Set to 0 for immediate deletion.")
	command.PersistentFlags().Int64Var(&flags.installationDeletionMaxUpdating, "installation-deletion-max-updating", 25, "A soft limit on the number of installations that the provisioner will delete at one time from the group of deletion-pending installations.")
	command.PersistentFlags().BoolVar(&flags.disableDNSUpdates, "disable-dns-updates", false, "If set to true DNS updates will be disabled when updating Installations.")
	command.PersistentFlags().StringVar(&flags.awatAddress, "awat", "http://localhost:8077", "The location of the Automatic Workspace Archive Translator if the import supervisor is being used.")
}

type schedulingOptions struct {
	balancedInstallationScheduling     bool
	clusterResourceThresholdScaleValue int
	clusterResourceThreshold           int
	thresholdCPUOverride               int
	thresholdMemoryOverride            int
	thresholdPodCountOverride          int
}

func (flags *schedulingOptions) addFlags(command *cobra.Command) {
	command.PersistentFlags().BoolVar(&flags.balancedInstallationScheduling, "balanced-installation-scheduling", false, "Whether to schedule installations on the cluster with the greatest percentage of available resources or not. (slows down scheduling speed as cluster count increases)")
	command.PersistentFlags().IntVar(&flags.clusterResourceThresholdScaleValue, "cluster-resource-threshold-scale-value", 0, "The number of worker nodes to scale up by when the threshold is passed. Set to 0 for no scaling. Scaling will never exceed the cluster max worker configuration value.")
	command.PersistentFlags().IntVar(&flags.clusterResourceThreshold, "cluster-resource-threshold", 80, "The percent threshold where new installations won't be scheduled on a multi-tenant cluster.")
	command.PersistentFlags().IntVar(&flags.thresholdCPUOverride, "cluster-resource-threshold-cpu-override", 0, "The cluster-resource-threshold override value for CPU resources only")
	command.PersistentFlags().IntVar(&flags.thresholdMemoryOverride, "cluster-resource-threshold-memory-override", 0, "The cluster-resource-threshold override value for memory resources only")
	command.PersistentFlags().IntVar(&flags.thresholdPodCountOverride, "cluster-resource-threshold-pod-count-override", 0, "The cluster-resource-threshold override value for pod count only")
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

func (flags *provisioningParams) addFlags(command *cobra.Command) {
	command.PersistentFlags().StringVar(&flags.provisioner, "provisioner", "kops", "Specifies which provisioner to use, one of: kops, eks.")
	command.PersistentFlags().StringVar(&flags.s3StateStore, "state-store", "dev.cloud.mattermost.com", "The S3 bucket used to store cluster state.")
	command.PersistentFlags().StringSliceVar(&flags.allowListCIDRRange, "allow-list-cidr-range", []string{"0.0.0.0/0"}, "The list of CIDRs to allow communication with the private ingress.")
	command.PersistentFlags().StringSliceVar(&flags.sloInstallationGroups, "slo-installation-groups", []string{}, "The list of installation group ids to create dedicated SLOs for.")
	command.PersistentFlags().StringSliceVar(&flags.vpnListCIDR, "vpn-list-cidr", []string{"0.0.0.0/0"}, "The list of VPN CIDRs to allow communication with the clusters.")
	command.PersistentFlags().BoolVar(&flags.useExistingResources, "use-existing-aws-resources", true, "Whether to use existing AWS resources (VPCs, subnets, etc.) or not.")
	command.PersistentFlags().BoolVar(&flags.deployMySQLOperator, "deploy-mysql-operator", true, "Whether to deploy the mysql operator.")
	command.PersistentFlags().BoolVar(&flags.deployMinioOperator, "deploy-minio-operator", true, "Whether to deploy the minio operator.")
	command.PersistentFlags().StringVar(&flags.ndotsDefaultValue, "ndots-value", "5", "The default ndots value for installations.")

	command.PersistentFlags().Int32Var(&flags.backupJobTTL, "backup-job-ttl-seconds", 3600, "Number of seconds after which finished backup jobs will be cleaned up. Set to negative value to not cleanup or 0 to cleanup immediately.")
	command.PersistentFlags().StringVar(&flags.backupRestoreToolImage, "backup-restore-tool-image", "mattermost/backup-restore-tool:latest", "Image of Backup Restore Tool to use.")

	command.PersistentFlags().IntVar(&flags.etcdQuotaBackendBytes, "etcd-quota-backend-bytes", 4294967296, "Raise alarms by cluster when backend size exceeds the given quota")
	command.PersistentFlags().StringVar(&flags.etcdListenMetricsURL, "etcd-listen-metrics-urls", "http://0.0.0.0:8081", "List of additional URL to listen for metrics")

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

func (flags *pgBouncerConfig) addFlags(command *cobra.Command) {
	command.PersistentFlags().IntVar(&flags.minPoolSize, "min-proxy-db-pool-size", 1, "The db proxy min pool size.")
	command.PersistentFlags().IntVar(&flags.defaultPoolSize, "default-proxy-db-pool-size", 5, "The db proxy default pool size per user.")
	command.PersistentFlags().IntVar(&flags.reservePoolSize, "reserve-proxy-db-pool-size", 10, "The db proxy reserve pool size per logical database.")
	command.PersistentFlags().IntVar(&flags.maxClientConnections, "max-client-connections", 20000, "The db proxy max client connections.")
	command.PersistentFlags().IntVar(&flags.maxDatabaseConnectionsPerPool, "max-proxy-db-connections-per-pool", 20, "The maximum number of proxy database connections per pool (logical database).")
	command.PersistentFlags().IntVar(&flags.serverIdleTimeout, "server-idle-timeout", 30, "The server idle timeout.")
	command.PersistentFlags().IntVar(&flags.serverLifetime, "server-lifetime", 300, "The server lifetime.")
	command.PersistentFlags().IntVar(&flags.serverResetQueryAlways, "server-reset-query-always", 0, "Whether server_reset_query should be run in all pooling modes.")
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

func (flags *installationOptions) addFlags(command *cobra.Command) {
	command.PersistentFlags().BoolVar(&flags.keepDatabaseData, "keep-database-data", true, "Whether to preserve database data after installation deletion or not.")
	command.PersistentFlags().BoolVar(&flags.keepFileStoreData, "keep-filestore-data", true, "Whether to preserve filestore data after installation deletion or not.")
	command.PersistentFlags().BoolVar(&flags.requireAnnotatedInstallations, "require-annotated-installations", false, "Require new installations to have at least one annotation.")
	command.PersistentFlags().StringVar(&flags.gitlabOAuthToken, "gitlab-oauth", "", "If Helm charts are stored in a Gitlab instance that requires authentication, provide the token here and it will be automatically set in the environment.")
	command.PersistentFlags().BoolVar(&flags.forceCRUpgrade, "force-cr-upgrade", false, "If specified installation CRVersions will be updated to the latest version when supervised.")
	command.PersistentFlags().StringVar(&flags.mattermostWebHook, "mattermost-webhook", "", "Set to use a Mattermost webhook for spot instances termination notifications")
	command.PersistentFlags().StringVar(&flags.mattermostChannel, "mattermost-channel", "", "Set a mattermost channel for spot instances termination notifications")
	command.PersistentFlags().StringVar(&flags.utilitiesGitURL, "utilities-git-url", "", "The private git domain to use for utilities. For example https://gitlab.com")
}

type dbUtilizationSettings struct {
	pgbouncer          int
	postgres           int
	mysql              int
	disableDBInitCheck bool
}

func (flags *dbUtilizationSettings) addFlags(command *cobra.Command) {
	command.PersistentFlags().IntVar(&flags.pgbouncer, "max-installations-rds-postgres-pgbouncer", toolsAWS.DefaultRDSMultitenantPGBouncerDatabasePostgresCountLimit, "Max installations per DB cluster of type RDS Postgres PGbouncer")
	command.PersistentFlags().IntVar(&flags.postgres, "max-installations-rds-postgres", toolsAWS.DefaultRDSMultitenantDatabasePostgresCountLimit, "Max installations per DB cluster of type RDS Postgres")
	command.PersistentFlags().IntVar(&flags.mysql, "max-installations-rds-mysql", toolsAWS.DefaultRDSMultitenantDatabaseMySQLCountLimit, "Max installations per DB cluster of type RDS MySQL")
	command.PersistentFlags().BoolVar(&flags.disableDBInitCheck, "disable-db-init-check", false, "Whether to disable init container with database check.")
}

type serverFlagChanged struct {
	isDebugChanged             bool
	isKeepDatabaseDataChanged  bool
	isKeepFileStoreDataChanged bool
}

func (flags *serverFlagChanged) addFlags(command *cobra.Command) {
	flags.isDebugChanged = command.Flags().Changed("debug")
	flags.isKeepDatabaseDataChanged = command.PersistentFlags().Changed("keep-database-data")
	flags.isKeepFileStoreDataChanged = command.PersistentFlags().Changed("keep-filestore-data")
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
}

func (flags *serverFlags) addFlags(command *cobra.Command) {
	flags.supervisorOptions.addFlags(command)
	flags.schedulingOptions.addFlags(command)
	flags.provisioningParams.addFlags(command)
	flags.pgBouncerConfig.addFlags(command)
	flags.installationOptions.addFlags(command)
	flags.dbUtilizationSettings.addFlags(command)

	command.PersistentFlags().StringVar(&flags.listen, "listen", ":8075", "The interface and port on which to listen.")
	command.PersistentFlags().IntVar(&flags.metricsPort, "metrics-port", 8076, "Port on which the metrics server should be listening.")

	command.PersistentFlags().BoolVar(&flags.debug, "debug", false, "Whether to output debug logs.")
	command.PersistentFlags().BoolVar(&flags.debugHelm, "debug-helm", false, "Whether to include Helm output in debug logs.")
	command.PersistentFlags().BoolVar(&flags.devMode, "dev", false, "Set sane defaults for development")
	command.PersistentFlags().BoolVar(&flags.machineLogs, "machine-readable-logs", false, "Output the logs in machine readable format.")

	command.PersistentFlags().StringVar(&flags.database, "database", "sqlite://cloud.db", "The database backing the provisioning server.")
	command.PersistentFlags().Int64Var(&flags.maxSchemas, "default-max-schemas-per-logical-database", 10, "When importing and creating new proxy multitenant databases, this value is used for MaxInstallationsPerLogicalDatabase.")
	command.PersistentFlags().BoolVar(&flags.enableRoute53, "installation-enable-route53", false, "Specifies whether CNAME records for Installation should be created in Route53 as well.")
	command.PersistentFlags().StringVar(&flags.kubecostToken, "kubecost-token", "", "Set a kubecost token")

	command.PersistentFlags().IntVar(&flags.poll, "poll", 30, "The interval in seconds to poll for background work.")
	command.PersistentFlags().IntVar(&flags.slowPoll, "slow-poll", 60, "The interval in seconds to poll for background work for supervisors that are not time sensitive (slow-poll supervisors).")
}
