// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.
//

package main

import (
	"time"

	toolsAWS "github.com/mattermost/mattermost-cloud/internal/tools/aws"
	"github.com/mattermost/mattermost-cloud/model"
	"github.com/spf13/cobra"
	corev1 "k8s.io/api/core/v1"
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
	command.Flags().BoolVar(&flags.disableAllSupervisors, "disable-all-supervisors", false, "disable all supervisors (API-only functionality)")

	command.Flags().BoolVar(&flags.clusterSupervisor, "cluster-supervisor", true, "Whether this server will run a cluster supervisor or not.")
	command.Flags().BoolVar(&flags.groupSupervisor, "group-supervisor", false, "Whether this server will run an installation group supervisor or not.")
	command.Flags().BoolVar(&flags.installationSupervisor, "installation-supervisor", true, "Whether this server will run an installation supervisor or not.")
	command.Flags().BoolVar(&flags.installationDeletionSupervisor, "installation-deletion-supervisor", true, "Whether this server will run a installation deletion supervisor or not. (slow-poll supervisor)")
	command.Flags().BoolVar(&flags.clusterInstallationSupervisor, "cluster-installation-supervisor", true, "Whether this server will run a cluster installation supervisor or not.")
	command.Flags().BoolVar(&flags.backupSupervisor, "backup-supervisor", false, "Whether this server will run a backup supervisor or not.")
	command.Flags().BoolVar(&flags.importSupervisor, "import-supervisor", false, "Whether this server will run a workspace import supervisor or not.")
	command.Flags().BoolVar(&flags.installationDBRestorationSupervisor, "installation-db-restoration-supervisor", false, "Whether this server will run an installation db restoration supervisor or not.")
	command.Flags().BoolVar(&flags.installationDBMigrationSupervisor, "installation-db-migration-supervisor", false, "Whether this server will run an installation db migration supervisor or not.")

	command.Flags().DurationVar(&flags.installationDeletionPendingTime, "installation-deletion-pending-time", 3*time.Minute, "The amount of time that installations will stay in the deletion queue before they are actually deleted. Set to 0 for immediate deletion.")
	command.Flags().Int64Var(&flags.installationDeletionMaxUpdating, "installation-deletion-max-updating", 25, "A soft limit on the number of installations that the provisioner will delete at one time from the group of deletion-pending installations.")
	command.Flags().BoolVar(&flags.disableDNSUpdates, "disable-dns-updates", false, "If set to true DNS updates will be disabled when updating Installations.")
	command.Flags().StringVar(&flags.awatAddress, "awat", "http://localhost:8077", "The location of the Automatic Workspace Archive Translator if the import supervisor is being used.")
}

type schedulingOptions struct {
	balancedInstallationScheduling     bool
	preferScheduleOnStableClusters     bool
	alwaysScheduleExternalClusters     bool
	clusterResourceThresholdScaleValue int
	clusterResourceThreshold           int
	thresholdCPUOverride               int
	thresholdMemoryOverride            int
	thresholdPodCountOverride          int
}

func (flags *schedulingOptions) addFlags(command *cobra.Command) {
	command.Flags().BoolVar(&flags.balancedInstallationScheduling, "balanced-installation-scheduling", true, "Whether to schedule installations on the cluster with the greatest percentage of available resources or not. (slows down scheduling speed as cluster count increases)")
	command.Flags().BoolVar(&flags.preferScheduleOnStableClusters, "prefer-stable-cluster-installation-scheduling", false, "Whether to prioritize scheduling installations on the clusters in the stable state or not. (can slow scheduling speed as cluster count increases)")
	command.Flags().BoolVar(&flags.alwaysScheduleExternalClusters, "always-schedule-external-clusters", false, "Whether to always schedule installations on the clusters that are externally managed by ignoring resource thresholds.")
	command.Flags().IntVar(&flags.clusterResourceThresholdScaleValue, "cluster-resource-threshold-scale-value", 0, "The number of worker nodes to scale up by when the threshold is passed. Set to 0 for no scaling. Scaling will never exceed the cluster max worker configuration value.")
	command.Flags().IntVar(&flags.clusterResourceThreshold, "cluster-resource-threshold", 80, "The percent threshold where new installations won't be scheduled on a multi-tenant cluster.")
	command.Flags().IntVar(&flags.thresholdCPUOverride, "cluster-resource-threshold-cpu-override", 0, "The cluster-resource-threshold override value for CPU resources only")
	command.Flags().IntVar(&flags.thresholdMemoryOverride, "cluster-resource-threshold-memory-override", 0, "The cluster-resource-threshold override value for memory resources only")
	command.Flags().IntVar(&flags.thresholdPodCountOverride, "cluster-resource-threshold-pod-count-override", 0, "The cluster-resource-threshold override value for pod count only")
}

type provisioningParams struct {
	s3StateStore              string
	allowListCIDRRange        []string
	sloInstallationGroups     []string
	sloEnterpriseGroups       []string
	vpnListCIDR               []string
	useExistingResources      bool
	deployMySQLOperator       bool
	deployMinioOperator       bool
	mattermostOperatorHelmDir string
	ndotsDefaultValue         string

	backupJobTTL           int32
	backupRestoreToolImage string

	etcdQuotaBackendBytes int
	etcdListenMetricsURL  string

	probeLivenessFailureThreshold    int32
	probeLivenessSuccessThreshold    int32
	probeLivenessInitialDelaySeconds int32
	probeLivenessPeriodSeconds       int32
	probeLivenessTimeoutSeconds      int32

	probeReadinessFailureThreshold    int32
	probeReadinessSuccessThreshold    int32
	probeReadinessInitialDelaySeconds int32
	probeReadinessPeriodSeconds       int32
	probeReadinessTimeoutSeconds      int32
}

func (flags *provisioningParams) addFlags(command *cobra.Command) {
	command.Flags().StringVar(&flags.s3StateStore, "state-store", "dev.cloud.mattermost.com", "The S3 bucket used to store cluster state.")
	command.Flags().StringSliceVar(&flags.allowListCIDRRange, "allow-list-cidr-range", []string{"0.0.0.0/0"}, "The list of CIDRs to allow communication with the private ingress.")
	command.Flags().StringSliceVar(&flags.sloInstallationGroups, "slo-installation-groups", []string{}, "The list of installation group ids to create dedicated SLOs for.")
	command.Flags().StringSliceVar(&flags.sloEnterpriseGroups, "slo-enterprise-groups", []string{}, "The list of enterprise group ids to create dedicated Nginx SLOs for.")
	command.Flags().StringSliceVar(&flags.vpnListCIDR, "vpn-list-cidr", []string{"0.0.0.0/0"}, "The list of VPN CIDRs to allow communication with the clusters.")
	command.Flags().BoolVar(&flags.useExistingResources, "use-existing-aws-resources", true, "Whether to use existing AWS resources (VPCs, subnets, etc.) or not.")
	command.Flags().BoolVar(&flags.deployMySQLOperator, "deploy-mysql-operator", false, "Whether to deploy the mysql operator.")
	command.Flags().BoolVar(&flags.deployMinioOperator, "deploy-minio-operator", false, "Whether to deploy the minio operator.")
	command.Flags().StringVar(&flags.mattermostOperatorHelmDir, "mattermost-operator-helm-dir", "", "Provide a directory location where a local mattermost operator helm chart will be deployed instead of from the standard repo")
	command.Flags().StringVar(&flags.ndotsDefaultValue, "ndots-value", "5", "The default ndots value for installations.")

	command.Flags().Int32Var(&flags.backupJobTTL, "backup-job-ttl-seconds", 3600, "Number of seconds after which finished backup jobs will be cleaned up. Set to negative value to not cleanup or 0 to cleanup immediately.")
	command.Flags().StringVar(&flags.backupRestoreToolImage, "backup-restore-tool-image", "mattermost/backup-restore-tool:latest", "Image of Backup Restore Tool to use.")

	command.Flags().IntVar(&flags.etcdQuotaBackendBytes, "etcd-quota-backend-bytes", 4294967296, "Raise alarms by cluster when backend size exceeds the given quota")
	command.Flags().StringVar(&flags.etcdListenMetricsURL, "etcd-listen-metrics-urls", "http://0.0.0.0:8081", "List of additional URL to listen for metrics")

	command.Flags().Int32Var(&flags.probeLivenessFailureThreshold, "probe-liveness-failure-threshold", 10, "An optional override for the liveness probe failure threshold.")
	command.Flags().Int32Var(&flags.probeLivenessSuccessThreshold, "probe-liveness-success-threshold", 1, "An optional override for the liveness probe success threshold.")
	command.Flags().Int32Var(&flags.probeLivenessInitialDelaySeconds, "probe-liveness-initial-delay-seconds", 30, "An optional override for the liveness probe initial delay seconds.")
	command.Flags().Int32Var(&flags.probeLivenessPeriodSeconds, "probe-liveness-period-seconds", 10, "An optional override for the liveness probe period seconds.")
	command.Flags().Int32Var(&flags.probeLivenessTimeoutSeconds, "probe-liveness-timeout-seconds", 5, "An optional override for the liveness probe timeout seconds.")

	command.Flags().Int32Var(&flags.probeReadinessFailureThreshold, "probe-readiness-failure-threshold", 10, "An optional override for the readiness probe failure threshold.")
	command.Flags().Int32Var(&flags.probeReadinessSuccessThreshold, "probe-readiness-success-threshold", 1, "An optional override for the readiness probe success threshold.")
	command.Flags().Int32Var(&flags.probeReadinessInitialDelaySeconds, "probe-readiness-initial-delay-seconds", 30, "An optional override for the readiness probe initial delay seconds.")
	command.Flags().Int32Var(&flags.probeReadinessPeriodSeconds, "probe-readiness-period-seconds", 10, "An optional override for the readiness probe period seconds.")
	command.Flags().Int32Var(&flags.probeReadinessTimeoutSeconds, "probe-readiness-timeout-seconds", 5, "An optional override for the readiness probe timeout seconds.")
}

func (flags *serverFlags) generateProbeOverrides() model.PodProbeOverrides {
	probeOverrides := model.PodProbeOverrides{}

	livenessOverride := corev1.Probe{}
	var livenessChanged bool
	if flags.probeLivenessFailureThresholdChanged {
		livenessOverride.FailureThreshold = flags.probeLivenessFailureThreshold
		livenessChanged = true
	}
	if flags.probeLivenessSuccessThresholdChanged {
		livenessOverride.SuccessThreshold = flags.probeLivenessSuccessThreshold
		livenessChanged = true
	}
	if flags.probeLivenessInitialDelaySecondsChanged {
		livenessOverride.InitialDelaySeconds = flags.probeLivenessInitialDelaySeconds
		livenessChanged = true
	}
	if flags.probeLivenessPeriodSecondsChanged {
		livenessOverride.PeriodSeconds = flags.probeLivenessPeriodSeconds
		livenessChanged = true
	}
	if flags.probeLivenessTimeoutSecondsChanged {
		livenessOverride.TimeoutSeconds = flags.probeLivenessTimeoutSeconds
		livenessChanged = true
	}
	if livenessChanged {
		probeOverrides.LivenessProbeOverride = &livenessOverride
	}

	readinessOverride := corev1.Probe{}
	var readinessChanged bool
	if flags.probeReadinessFailureThresholdChanged {
		readinessOverride.FailureThreshold = flags.probeReadinessFailureThreshold
		readinessChanged = true
	}
	if flags.probeReadinessSuccessThresholdChanged {
		readinessOverride.SuccessThreshold = flags.probeReadinessSuccessThreshold
		readinessChanged = true
	}
	if flags.probeReadinessInitialDelaySecondsChanged {
		readinessOverride.InitialDelaySeconds = flags.probeReadinessInitialDelaySeconds
		readinessChanged = true
	}
	if flags.probeReadinessPeriodSecondsChanged {
		readinessOverride.PeriodSeconds = flags.probeReadinessPeriodSeconds
		readinessChanged = true
	}
	if flags.probeReadinessTimeoutSecondsChanged {
		readinessOverride.TimeoutSeconds = flags.probeReadinessTimeoutSeconds
		readinessChanged = true
	}
	if readinessChanged {
		probeOverrides.ReadinessProbeOverride = &readinessOverride
	}

	return probeOverrides
}

type installationOptions struct {
	keepDatabaseData              bool
	keepFileStoreData             bool
	requireAnnotatedInstallations bool
	gitlabOAuthToken              string
	argocdApiToken                string
	argocdServerApi               string
	forceCRUpgrade                bool
	utilitiesGitURL               string
	utilitiesGitPath              string
	enableS3Versioning            bool
	internalIPRanges              []string
}

func (flags *installationOptions) addFlags(command *cobra.Command) {
	command.Flags().BoolVar(&flags.keepDatabaseData, "keep-database-data", false, "Whether to preserve database data after installation deletion or not.")
	command.Flags().BoolVar(&flags.keepFileStoreData, "keep-filestore-data", false, "Whether to preserve filestore data after installation deletion or not.")
	command.Flags().BoolVar(&flags.requireAnnotatedInstallations, "require-annotated-installations", false, "Require new installations to have at least one annotation.")
	command.Flags().StringVar(&flags.gitlabOAuthToken, "gitlab-oauth", "", "If Helm charts are stored in a Gitlab instance that requires authentication, provide the token here and it will be automatically set in the environment.")
	command.Flags().StringVar(&flags.argocdServerApi, "argocd-server-api", "", "The API endpoint to use for ArgoCD.")
	command.Flags().StringVar(&flags.argocdApiToken, "argocd-api-token", "", "The API token to use for ArgoCD.")
	command.Flags().BoolVar(&flags.forceCRUpgrade, "force-cr-upgrade", false, "If specified installation CRVersions will be updated to the latest version when supervised.")
	command.Flags().StringVar(&flags.utilitiesGitURL, "utilities-git-url", "", "The private git domain to use for utilities. For example https://gitlab.com")
	command.Flags().StringVar(&flags.utilitiesGitPath, "utilities-git-path", "", "The git path to use for utilities. For example /gitops/gitops.git")
	command.Flags().BoolVar(&flags.enableS3Versioning, "enable-s3-versioning", false, "Whether to enable S3 versioning for the installation bucket or not")
	command.Flags().StringSliceVar(&flags.internalIPRanges, "internal-ip-ranges", []string{}, "Some ranges that needed to be allowed for operational reasons")
}

type dbUtilizationSettings struct {
	perseus            int
	pgbouncer          int
	postgres           int
	mysql              int
	disableDBInitCheck bool
}

func (flags *dbUtilizationSettings) addFlags(command *cobra.Command) {
	command.Flags().IntVar(&flags.perseus, "max-installations-perseus", toolsAWS.DefaultRDSMultitenantPerseusDatabasePostgresCountLimit, "Max installations per DB cluster of type Perseus")
	command.Flags().IntVar(&flags.pgbouncer, "max-installations-rds-postgres-pgbouncer", toolsAWS.DefaultRDSMultitenantPGBouncerDatabasePostgresCountLimit, "Max installations per DB cluster of type RDS Postgres PGbouncer")
	command.Flags().IntVar(&flags.postgres, "max-installations-rds-postgres", toolsAWS.DefaultRDSMultitenantDatabasePostgresCountLimit, "Max installations per DB cluster of type RDS Postgres")
	command.Flags().IntVar(&flags.mysql, "max-installations-rds-mysql", toolsAWS.DefaultRDSMultitenantDatabaseMySQLCountLimit, "Max installations per DB cluster of type RDS MySQL")
	command.Flags().BoolVar(&flags.disableDBInitCheck, "disable-db-init-check", false, "Whether to disable init container with database check.")
}

type serverFlagChanged struct {
	isDebugChanged bool

	isKeepDatabaseDataChanged  bool
	isKeepFileStoreDataChanged bool

	probeLivenessFailureThresholdChanged    bool
	probeLivenessSuccessThresholdChanged    bool
	probeLivenessInitialDelaySecondsChanged bool
	probeLivenessPeriodSecondsChanged       bool
	probeLivenessTimeoutSecondsChanged      bool

	probeReadinessFailureThresholdChanged    bool
	probeReadinessSuccessThresholdChanged    bool
	probeReadinessInitialDelaySecondsChanged bool
	probeReadinessPeriodSecondsChanged       bool
	probeReadinessTimeoutSecondsChanged      bool
}

type serverAuthFlags struct {
	Issuer                               string
	Audience                             string
	TokenEndpoint                        string
	RestrictedClientIDs                  []string
	RestrictedClientAllowedEndpointsList []string
	JWKSURL                              string
}

func (flags *serverAuthFlags) addFlags(command *cobra.Command) {
	command.Flags().StringVar(&flags.Issuer, "issuer", "", "The OAuth2 issuer URL.")
	command.Flags().StringVar(&flags.Audience, "audience", "", "The OAuth2 audience.")
	command.Flags().StringVar(&flags.TokenEndpoint, "token-endpoint", "", "The OAuth2 token endpoint.")
	command.Flags().StringSliceVar(&flags.RestrictedClientIDs, "restricted-client-ids", []string{}, "The list of restricted client IDs.")
	command.Flags().StringSliceVar(&flags.RestrictedClientAllowedEndpointsList, "restricted-client-allowed-endpoints", []string{}, "The list of restricted endpoints.")
	command.Flags().StringVar(&flags.JWKSURL, "jwks-url", "", "The URL of the JWKS endpoint.")
}

func (flags *serverFlagChanged) addFlags(command *cobra.Command) {
	flags.isDebugChanged = command.Flags().Changed("debug")

	flags.isKeepDatabaseDataChanged = command.Flags().Changed("keep-database-data")
	flags.isKeepFileStoreDataChanged = command.Flags().Changed("keep-filestore-data")

	flags.probeLivenessFailureThresholdChanged = command.Flags().Changed("probe-liveness-failure-threshold")
	flags.probeLivenessSuccessThresholdChanged = command.Flags().Changed("probe-liveness-success-threshold")
	flags.probeLivenessInitialDelaySecondsChanged = command.Flags().Changed("probe-liveness-initial-delay-seconds")
	flags.probeLivenessPeriodSecondsChanged = command.Flags().Changed("probe-liveness-period-seconds")
	flags.probeLivenessTimeoutSecondsChanged = command.Flags().Changed("probe-liveness-timeout-seconds")

	flags.probeReadinessFailureThresholdChanged = command.Flags().Changed("probe-readiness-failure-threshold")
	flags.probeReadinessSuccessThresholdChanged = command.Flags().Changed("probe-readiness-success-threshold")
	flags.probeReadinessInitialDelaySecondsChanged = command.Flags().Changed("probe-readiness-initial-delay-seconds")
	flags.probeReadinessPeriodSecondsChanged = command.Flags().Changed("probe-readiness-period-seconds")
	flags.probeReadinessTimeoutSecondsChanged = command.Flags().Changed("probe-readiness-timeout-seconds")
}

type serverFlags struct {
	supervisorOptions
	schedulingOptions
	provisioningParams
	installationOptions
	dbUtilizationSettings
	serverFlagChanged
	serverAuthFlags

	listen      string
	metricsPort int

	debug                    bool
	devMode                  bool
	machineLogs              bool
	enableLogStacktrace      bool
	enableLogFilesPerCluster bool
	logFilesPerClusterPath   string
	grafanaURL               string
	grafanaTokens            []string

	database      string
	maxSchemas    int64
	enableRoute53 bool

	poll     int
	slowPoll int
}

func (flags *serverFlags) addFlags(command *cobra.Command) {
	flags.supervisorOptions.addFlags(command)
	flags.schedulingOptions.addFlags(command)
	flags.provisioningParams.addFlags(command)
	flags.installationOptions.addFlags(command)
	flags.dbUtilizationSettings.addFlags(command)
	flags.serverAuthFlags.addFlags(command)

	command.Flags().StringVar(&flags.listen, "listen", ":8075", "The interface and port on which to listen.")
	command.Flags().IntVar(&flags.metricsPort, "metrics-port", 8076, "Port on which the metrics server should be listening.")

	command.Flags().BoolVar(&flags.debug, "debug", false, "Whether to output debug logs.")
	command.Flags().BoolVar(&flags.devMode, "dev", false, "Set sane defaults for development")
	command.Flags().BoolVar(&flags.machineLogs, "machine-readable-logs", false, "Output the logs in machine readable format.")
	command.Flags().BoolVar(&flags.enableLogStacktrace, "enable-log-stacktrace", false, "Add stacktrace in error logs.")
	command.Flags().BoolVar(&flags.enableLogFilesPerCluster, "enable-log-files-per-cluster", false, "Store individual log files per cluster.")
	command.Flags().StringVar(&flags.logFilesPerClusterPath, "log-files-per-cluster-path", "", "Where to store the cluster log files.")
	command.Flags().StringVar(&flags.grafanaURL, "grafana-url", "", "The URL of a Grafana endpoint to send annotations to.")
	command.Flags().StringSliceVar(&flags.grafanaTokens, "grafana-tokens", []string{""}, "Grafana tokens which will be used with the provided URL")
	command.MarkFlagsRequiredTogether("enable-log-files-per-cluster", "log-files-per-cluster-path")

	command.Flags().StringVar(&flags.database, "database", "", "The database backing the provisioning server.")
	command.Flags().Int64Var(&flags.maxSchemas, "default-max-schemas-per-logical-database", 10, "When importing and creating new proxy multitenant databases, this value is used for MaxInstallationsPerLogicalDatabase.")
	command.Flags().BoolVar(&flags.enableRoute53, "installation-enable-route53", false, "Specifies whether CNAME records for Installation should be created in Route53 as well.")

	command.Flags().IntVar(&flags.poll, "poll", 30, "The interval in seconds to poll for background work.")
	command.Flags().IntVar(&flags.slowPoll, "slow-poll", 60, "The interval in seconds to poll for background work for supervisors that are not time sensitive (slow-poll supervisors).")
}
