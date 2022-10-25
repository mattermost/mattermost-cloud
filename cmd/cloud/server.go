// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.
//

package main

import (
	"context"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"os/exec"
	"os/signal"
	"path"
	"strings"
	"syscall"
	"time"

	cf "github.com/cloudflare/cloudflare-go"

	"github.com/mattermost/mattermost-cloud/internal/tools/cloudflare"

	"github.com/prometheus/client_golang/prometheus/promhttp"

	sdkAWS "github.com/aws/aws-sdk-go/aws"
	"github.com/gorilla/mux"
	awat "github.com/mattermost/awat/model"
	"github.com/mattermost/mattermost-cloud/internal/api"
	"github.com/mattermost/mattermost-cloud/internal/events"
	"github.com/mattermost/mattermost-cloud/internal/metrics"
	"github.com/mattermost/mattermost-cloud/internal/provisioner"
	"github.com/mattermost/mattermost-cloud/internal/store"
	"github.com/mattermost/mattermost-cloud/internal/supervisor"
	toolsAWS "github.com/mattermost/mattermost-cloud/internal/tools/aws"
	"github.com/mattermost/mattermost-cloud/internal/tools/helm"
	"github.com/mattermost/mattermost-cloud/internal/tools/kops"
	"github.com/mattermost/mattermost-cloud/internal/tools/terraform"
	"github.com/mattermost/mattermost-cloud/internal/tools/utils"
	"github.com/mattermost/mattermost-cloud/model"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

const defaultLocalServerAPI = "http://localhost:8075"

var instanceID string

func init() {
	instanceID = model.NewID()

	// General
	serverCmd.PersistentFlags().String("database", "sqlite://cloud.db", "The database backing the provisioning server.")
	serverCmd.PersistentFlags().String("listen", ":8075", "The interface and port on which to listen.")
	serverCmd.PersistentFlags().Int("metrics-port", 8076, "Port on which the metrics server should be listening.")
	serverCmd.PersistentFlags().String("state-store", "dev.cloud.mattermost.com", "The S3 bucket used to store cluster state.")
	serverCmd.PersistentFlags().StringSlice("allow-list-cidr-range", []string{"0.0.0.0/0"}, "The list of CIDRs to allow communication with the private ingress.")
	serverCmd.PersistentFlags().StringSlice("vpn-list-cidr", []string{"0.0.0.0/0"}, "The list of VPN CIDRs to allow communication with the clusters.")
	serverCmd.PersistentFlags().Bool("debug", false, "Whether to output debug logs.")
	serverCmd.PersistentFlags().Bool("debug-helm", false, "Whether to include Helm output in debug logs.")
	serverCmd.PersistentFlags().Bool("machine-readable-logs", false, "Output the logs in machine readable format.")
	serverCmd.PersistentFlags().Bool("dev", false, "Set sane defaults for development")
	serverCmd.PersistentFlags().String("backup-restore-tool-image", "mattermost/backup-restore-tool:latest", "Image of Backup Restore Tool to use.")
	serverCmd.PersistentFlags().Int32("backup-job-ttl-seconds", 3600, "Number of seconds after which finished backup jobs will be cleaned up. Set to negative value to not cleanup or 0 to cleanup immediately.")
	serverCmd.PersistentFlags().Bool("deploy-mysql-operator", true, "Whether to deploy the mysql operator.")
	serverCmd.PersistentFlags().Bool("deploy-minio-operator", true, "Whether to deploy the minio operator.")
	serverCmd.PersistentFlags().Int64("default-max-schemas-per-logical-database", 10, "When importing and creating new proxy multitenant databases, this value is used for MaxInstallationsPerLogicalDatabase.")
	serverCmd.PersistentFlags().String("provisioner", "kops", "Specifies which provisioner to use, one of: kops, eks.")
	serverCmd.PersistentFlags().Bool("disable-all-supervisors", false, "disable all provisioners and enable API only funcionality")

	// Supervisors
	serverCmd.PersistentFlags().Int("poll", 30, "The interval in seconds to poll for background work.")
	serverCmd.PersistentFlags().Int("slow-poll", 60, "The interval in seconds to poll for background work for supervisors that are not time sensitive (slow-poll supervisors).")
	serverCmd.PersistentFlags().Bool("cluster-supervisor", true, "Whether this server will run a cluster supervisor or not.")
	serverCmd.PersistentFlags().Bool("group-supervisor", false, "Whether this server will run an installation group supervisor or not.")
	serverCmd.PersistentFlags().Bool("installation-supervisor", true, "Whether this server will run an installation supervisor or not.")
	serverCmd.PersistentFlags().Bool("installation-db-restoration-supervisor", false, "Whether this server will run an installation db restoration supervisor or not.")
	serverCmd.PersistentFlags().Bool("installation-db-migration-supervisor", false, "Whether this server will run an installation db migration supervisor or not.")
	serverCmd.PersistentFlags().Bool("installation-deletion-supervisor", true, "Whether this server will run a installation deletion supervisor or not. (slow-poll supervisor)")
	serverCmd.PersistentFlags().Bool("cluster-installation-supervisor", true, "Whether this server will run a cluster installation supervisor or not.")
	serverCmd.PersistentFlags().Bool("backup-supervisor", false, "Whether this server will run a backup supervisor or not.")
	serverCmd.PersistentFlags().Bool("import-supervisor", false, "Whether this server will run a workspace import supervisor or not.")
	serverCmd.PersistentFlags().String("awat", "http://localhost:8077", "The location of the Automatic Workspace Archive Translator if the import supervisor is being used.")

	// Scheduling options
	serverCmd.PersistentFlags().Bool("balanced-installation-scheduling", false, "Whether to schedule installations on the cluster with the greatest percentage of available resources or not. (slows down scheduling speed as cluster count increases)")
	serverCmd.PersistentFlags().Int("cluster-resource-threshold-scale-value", 0, "The number of worker nodes to scale up by when the threshold is passed. Set to 0 for no scaling. Scaling will never exceed the cluster max worker configuration value.")
	serverCmd.PersistentFlags().Int("cluster-resource-threshold", 80, "The percent threshold where new installations won't be scheduled on a multi-tenant cluster.")
	serverCmd.PersistentFlags().Int("cluster-resource-threshold-cpu-override", 0, "The cluster-resource-threshold override value for CPU resources only")
	serverCmd.PersistentFlags().Int("cluster-resource-threshold-memory-override", 0, "The cluster-resource-threshold override value for memory resources only")
	serverCmd.PersistentFlags().Int("cluster-resource-threshold-pod-count-override", 0, "The cluster-resource-threshold override value for pod count only")

	// Installation options
	serverCmd.PersistentFlags().Bool("use-existing-aws-resources", true, "Whether to use existing AWS resources (VPCs, subnets, etc.) or not.")
	serverCmd.PersistentFlags().Bool("keep-database-data", true, "Whether to preserve database data after installation deletion or not.")
	serverCmd.PersistentFlags().Bool("keep-filestore-data", true, "Whether to preserve filestore data after installation deletion or not.")
	serverCmd.PersistentFlags().Bool("require-annotated-installations", false, "Require new installations to have at least one annotation.")
	serverCmd.PersistentFlags().String("gitlab-oauth", "", "If Helm charts are stored in a Gitlab instance that requires authentication, provide the token here and it will be automatically set in the environment.")
	serverCmd.PersistentFlags().Bool("force-cr-upgrade", false, "If specified installation CRVersions will be updated to the latest version when supervised.")
	serverCmd.PersistentFlags().String("mattermost-webhook", "", "Set to use a Mattermost webhook for spot instances termination notifications")
	serverCmd.PersistentFlags().String("mattermost-channel", "", "Set a mattermost channel for spot instances termination notifications")
	serverCmd.PersistentFlags().String("utilities-git-url", "", "The private git domain to use for utilities. For example https://gitlab.com")
	serverCmd.PersistentFlags().Int("max-proxy-db-connections-per-pool", 20, "The maximum number of proxy database connections per pool (logical database).")
	serverCmd.PersistentFlags().Int("default-proxy-db-pool-size", 5, "The db proxy default pool size per user.")
	serverCmd.PersistentFlags().Int("reserve-proxy-db-pool-size", 10, "The db proxy reserve pool size per logical database.")
	serverCmd.PersistentFlags().Int("min-proxy-db-pool-size", 1, "The db proxy min pool size.")
	serverCmd.PersistentFlags().Int("max-client-connections", 20000, "The db proxy max client connections.")
	serverCmd.PersistentFlags().Int("server-idle-timeout", 30, "The server idle timeout.")
	serverCmd.PersistentFlags().Int("server-lifetime", 300, "The server lifetime.")
	serverCmd.PersistentFlags().Int("server-reset-query-always", 0, "Whether server_reset_query should be run in all pooling modes.")

	serverCmd.PersistentFlags().String("kubecost-token", "", "Set a kubecost token")
	serverCmd.PersistentFlags().String("ndots-value", "5", "The default ndots value for installations.")
	serverCmd.PersistentFlags().Bool("disable-db-init-check", false, "Whether to disable init container with database check.")
	serverCmd.PersistentFlags().Bool("installation-enable-route53", false, "Specifies whether CNAME records for Installation should be created in Route53 as well.")
	serverCmd.PersistentFlags().Bool("disable-dns-updates", false, "If set to true DNS updates will be disabled when updating Installations.")
	serverCmd.PersistentFlags().Duration("installation-deletion-pending-time", 3*time.Minute, "The amount of time that installations will stay in the deletion queue before they are actually deleted. Set to 0 for immediate deletion.")
	serverCmd.PersistentFlags().Int64("installation-deletion-max-updating", 25, "A soft limit on the number of installations that the provisioner will delete at one time from the group of deletion-pending installations.")

	// DB clusters utilization configuration
	serverCmd.PersistentFlags().Int("max-installations-rds-postgres-pgbouncer", toolsAWS.DefaultRDSMultitenantPGBouncerDatabasePostgresCountLimit, "Max installations per DB cluster of type RDS Postgres PGbouncer")
	serverCmd.PersistentFlags().Int("max-installations-rds-postgres", toolsAWS.DefaultRDSMultitenantDatabasePostgresCountLimit, "Max installations per DB cluster of type RDS Postgres")
	serverCmd.PersistentFlags().Int("max-installations-rds-mysql", toolsAWS.DefaultRDSMultitenantDatabaseMySQLCountLimit, "Max installations per DB cluster of type RDS MySQL")
}

// Provisioner is an interface for different types of provisioners.
type Provisioner interface {
	api.Provisioner
	supervisor.Provisioner
}

var serverCmd = &cobra.Command{
	Use:   "server",
	Short: "Run the provisioning server.",
	RunE: func(command *cobra.Command, args []string) error {
		command.SilenceUsage = true

		devMode, _ := command.Flags().GetBool("dev")

		debug, _ := command.Flags().GetBool("debug")
		debugMode := debug || (devMode && flagIsUnset(command, "debug"))
		if debugMode {
			logger.SetLevel(logrus.DebugLevel)
		}

		debugHelm, _ := command.Flags().GetBool("debug-helm")
		helm.SetVerboseHelmLogging(debugHelm)

		maxSchemas, _ := command.Flags().GetInt64("default-max-schemas-per-logical-database")
		err := model.SetDefaultProxyDatabaseMaxInstallationsPerLogicalDatabase(maxSchemas)
		if err != nil {
			return err
		}

		// Generate PGBouncer Config
		minPoolSize, _ := command.Flags().GetInt("min-proxy-db-pool-size")
		defaultPoolSize, _ := command.Flags().GetInt("default-proxy-db-pool-size")
		reservePoolSize, _ := command.Flags().GetInt("reserve-proxy-db-pool-size")
		maxClientConnections, _ := command.Flags().GetInt("max-client-connections")
		maxDatabaseConnectionsPerPool, _ := command.Flags().GetInt("max-proxy-db-connections-per-pool")
		serverIdleTimeout, _ := command.Flags().GetInt("server-idle-timeout")
		serverLifetime, _ := command.Flags().GetInt("server-lifetime")
		serverResetQueryAlways, _ := command.Flags().GetInt("server-reset-query-always")
		pgbouncerConfig := provisioner.NewPGBouncerConfig(
			minPoolSize, defaultPoolSize, reservePoolSize,
			maxClientConnections, maxDatabaseConnectionsPerPool,
			serverIdleTimeout, serverLifetime, serverResetQueryAlways,
		)
		err = pgbouncerConfig.Validate()
		if err != nil {
			return errors.Wrap(err, "pgbouncer config failed validation")
		}

		gitlabOAuthToken, _ := command.Flags().GetString("gitlab-oauth")
		if len(gitlabOAuthToken) == 0 {
			gitlabOAuthToken = os.Getenv(model.GitlabOAuthTokenKey)
		}
		model.SetGitlabToken(gitlabOAuthToken)
		if len(model.GetGitlabToken()) == 0 {
			logger.Warnf("The gitlab-oauth flag and %s were empty; using local helm charts", model.GitlabOAuthTokenKey)
		}

		machineLogs, _ := command.Flags().GetBool("machine-readable-logs")
		if machineLogs {
			logger.SetFormatter(&logrus.JSONFormatter{})
		}

		requireAnnotatedInstallations, _ := command.Flags().GetBool("require-annotated-installations")
		model.SetRequireAnnotatedInstallations(requireAnnotatedInstallations)

		forceCRUpgrade, _ := command.Flags().GetBool("force-cr-upgrade")

		allowListCIDRRange, _ := command.Flags().GetStringSlice("allow-list-cidr-range")
		if len(allowListCIDRRange) == 0 {
			return errors.New("allow-list-cidr-range must have at least one value")
		}

		vpnListCIDR, _ := command.Flags().GetStringSlice("vpn-list-cidr")
		if len(vpnListCIDR) == 0 {
			return errors.New("vpn-list-cidr must have at least one value")
		}

		mattermostWebHook, _ := command.Flags().GetString("mattermost-webhook")
		if mattermostWebHook != "" {
			os.Setenv(model.MattermostWebhook, mattermostWebHook)
		}

		mattermostChannel, _ := command.Flags().GetString("mattermost-channel")
		if mattermostChannel != "" {
			os.Setenv(model.MattermostChannel, mattermostChannel)
		}

		utilitiesGitURL, _ := command.Flags().GetString("utilities-git-url")
		if utilitiesGitURL == "" {
			return errors.New("utilities-git-url must be set")
		}
		model.SetUtilityDefaults(utilitiesGitURL)

		kubecostToken, _ := command.Flags().GetString("kubecost-token")
		if kubecostToken != "" {
			os.Setenv(model.KubecostToken, kubecostToken)
		}

		logger := logger.WithField("instance", instanceID)

		sqlStore, err := sqlStore(command)
		if err != nil {
			return err
		}

		currentVersion, err := sqlStore.GetCurrentVersion()
		if err != nil {
			return err
		}
		serverVersion := store.LatestVersion()

		// Require the schema to be at least the server version, and also the same major
		// version.
		if currentVersion.LT(serverVersion) || currentVersion.Major != serverVersion.Major {
			return errors.Errorf("server requires at least schema %s, current is %s", serverVersion, currentVersion)
		}

		// TODO: move these cluster threshold values to cluster configuration.
		balancedInstallationScheduling, _ := command.Flags().GetBool("balanced-installation-scheduling")
		clusterResourceThreshold, _ := command.Flags().GetInt("cluster-resource-threshold")
		thresholdCPUOverride, _ := command.Flags().GetInt("cluster-resource-threshold-cpu-override")
		thresholdMemoryOverride, _ := command.Flags().GetInt("cluster-resource-threshold-memory-override")
		thresholdPodCountOverride, _ := command.Flags().GetInt("cluster-resource-threshold-pod-count-override")
		clusterResourceThresholdScaleValue, _ := command.Flags().GetInt("cluster-resource-threshold-scale-value")
		installationScheduling := supervisor.NewInstallationSupervisorSchedulingOptions(balancedInstallationScheduling, clusterResourceThreshold, thresholdCPUOverride, thresholdMemoryOverride, thresholdPodCountOverride, clusterResourceThresholdScaleValue)
		err = installationScheduling.Validate()
		if err != nil {
			return errors.Wrap(err, "invalid installation scheduling options")
		}

		disableAllSupervisors, _ := command.Flags().GetBool("disable-all-supervisors")
		supervisorsEnabled := map[string]bool{
			"clusterSupervisor":                   false,
			"groupSupervisor":                     false,
			"installationSupervisor":              false,
			"installationDeletionSupervisor":      false,
			"clusterInstallationSupervisor":       false,
			"backupSupervisor":                    false,
			"importSupervisor":                    false,
			"installationDBRestorationSupervisor": false,
			"installationDBMigrationSupervisor":   false,
		}

		if !disableAllSupervisors {
			supervisorsEnabled["clusterSupervisor"], _ = command.Flags().GetBool("cluster-supervisor")
			supervisorsEnabled["groupSupervisor"], _ = command.Flags().GetBool("group-supervisor")
			supervisorsEnabled["installationSupervisor"], _ = command.Flags().GetBool("installation-supervisor")
			supervisorsEnabled["installationDeletionSupervisor"], _ = command.Flags().GetBool("installation-deletion-supervisor")
			supervisorsEnabled["clusterInstallationSupervisor"], _ = command.Flags().GetBool("cluster-installation-supervisor")
			supervisorsEnabled["backupSupervisor"], _ = command.Flags().GetBool("backup-supervisor")
			supervisorsEnabled["importSupervisor"], _ = command.Flags().GetBool("import-supervisor")
			supervisorsEnabled["installationDBRestorationSupervisor"], _ = command.Flags().GetBool("installation-db-restoration-supervisor")
			supervisorsEnabled["installationDBMigrationSupervisor"], _ = command.Flags().GetBool("installation-db-migration-supervisor")
		}

		if !isAnyMapValue(supervisorsEnabled) {
			logger.Warn("Server will be running with no supervisors. Only API functionality will work.")
		}

		s3StateStore, _ := command.Flags().GetString("state-store")
		keepDatabaseData, _ := command.Flags().GetBool("keep-database-data")
		keepFilestoreData, _ := command.Flags().GetBool("keep-filestore-data")
		useExistingResources, _ := command.Flags().GetBool("use-existing-aws-resources")
		backupRestoreToolImage, _ := command.Flags().GetString("backup-restore-tool-image")
		backupJobTTL, _ := command.Flags().GetInt32("backup-job-ttl-seconds")
		installationDeletionPendingTime, _ := command.Flags().GetDuration("installation-deletion-pending-time")
		installationDeletionMaxUpdating, _ := command.Flags().GetInt64("installation-deletion-max-updating")

		deployMySQLOperator, _ := command.Flags().GetBool("deploy-mysql-operator")
		deployMinioOperator, _ := command.Flags().GetBool("deploy-minio-operator")
		model.SetDeployOperators(deployMySQLOperator, deployMinioOperator)

		ndotsDefaultValue, _ := command.Flags().GetString("ndots-value")
		disableDBInitCheck, _ := command.Flags().GetBool("disable-db-init-check")
		enableRoute53, _ := command.Flags().GetBool("installation-enable-route53")
		disableDNSUpdates, _ := command.Flags().GetBool("disable-dns-updates")

		wd, err := os.Getwd()
		if err != nil {
			wd = "error getting working directory"
			logger.WithError(err).Error("Unable to get current working directory")
		}

		if devMode {
			if flagIsUnset(command, "keep-database-data") {
				keepDatabaseData = false
			}
			if flagIsUnset(command, "keep-filestore-data") {
				keepFilestoreData = false
			}
		}

		provisionerFlag, _ := command.Flags().GetString("provisioner")

		logger.WithFields(logrus.Fields{
			"build-hash":                                    model.BuildHash,
			"cluster-supervisor":                            supervisorsEnabled["clusterSupervisor"],
			"group-supervisor":                              supervisorsEnabled["groupSupervisor"],
			"installation-supervisor":                       supervisorsEnabled["installationSupervisor"],
			"installation-deletion-supervisor":              supervisorsEnabled["installationDeletionSupervisor"],
			"cluster-installation-supervisor":               supervisorsEnabled["clusterInstallationSupervisor"],
			"backup-supervisor":                             supervisorsEnabled["backupSupervisor"],
			"import-supervisor":                             supervisorsEnabled["importSupervisor"],
			"installation-db-restoration-supervisor":        supervisorsEnabled["installationDBRestorationSupervisor"],
			"installation-db-migration-supervisor":          supervisorsEnabled["installationDBMigrationSupervisor"],
			"store-version":                                 currentVersion,
			"state-store":                                   s3StateStore,
			"working-directory":                             wd,
			"installation-deletion-pending-time":            installationDeletionPendingTime,
			"installation-deletion-max-updating":            installationDeletionMaxUpdating,
			"balanced-installation-scheduling":              balancedInstallationScheduling,
			"cluster-resource-threshold":                    clusterResourceThreshold,
			"cluster-resource-threshold-cpu-override":       thresholdCPUOverride,
			"cluster-resource-threshold-memory-override":    thresholdMemoryOverride,
			"cluster-resource-threshold-pod-count-override": thresholdPodCountOverride,
			"cluster-resource-threshold-scale-value":        clusterResourceThresholdScaleValue,
			"use-existing-aws-resources":                    useExistingResources,
			"keep-database-data":                            keepDatabaseData,
			"keep-filestore-data":                           keepFilestoreData,
			"force-cr-upgrade":                              forceCRUpgrade,
			"backup-restore-tool-image":                     backupRestoreToolImage,
			"backup-job-ttl-seconds":                        backupJobTTL,
			"debug":                                         debugMode,
			"dev-mode":                                      devMode,
			"deploy-mysql-operator":                         deployMySQLOperator,
			"deploy-minio-operator":                         deployMinioOperator,
			"ndots-value":                                   ndotsDefaultValue,
			"maxDatabaseConnectionsPerPool":                 maxDatabaseConnectionsPerPool,
			"defaultPoolSize":                               defaultPoolSize,
			"minPoolSize":                                   minPoolSize,
			"maxClientConnections":                          maxClientConnections,
			"disable-db-init-check":                         disableDBInitCheck,
			"enable-route53":                                enableRoute53,
			"disable-dns-updates":                           disableDNSUpdates,
			"provisioner":                                   provisionerFlag,
		}).Info("Starting Mattermost Provisioning Server")

		deprecationWarnings(logger, command)

		// Warn on settings we consider to be non-production.
		if !useExistingResources {
			logger.Warn("[DEV] Server is configured to not use cluster VPC claim functionality")
		}

		// best-effort attempt to tag the VPC with a human's identity for dev purposes
		owner := getHumanReadableID()

		awsRegion := os.Getenv("AWS_REGION")
		if awsRegion == "" {
			awsRegion = toolsAWS.DefaultAWSRegion
		}
		awsConfig := &sdkAWS.Config{
			Region: sdkAWS.String(awsRegion),
			// TODO: we should use Retryer for a more robust retry strategy.
			// https://github.com/aws/aws-sdk-go/blob/99cd35c8c7d369ba8c32c46ed306f6c88d24cfd7/aws/request/retryer.go#L20
			MaxRetries: sdkAWS.Int(toolsAWS.DefaultAWSClientRetries),
		}
		awsClient, err := toolsAWS.NewAWSClientWithConfig(awsConfig, logger)
		if err != nil {
			return errors.Wrap(err, "failed to build AWS client")
		}

		err = checkRequirements(logger)
		if err != nil {
			return errors.Wrap(err, "failed health check")
		}

		resourceUtil := utils.NewResourceUtil(instanceID, awsClient, dbClusterUtilizationSettingsFromFlags(command), disableDBInitCheck)

		provisioningParams := provisioner.ProvisioningParams{
			S3StateStore:            s3StateStore,
			AllowCIDRRangeList:      allowListCIDRRange,
			VpnCIDRList:             vpnListCIDR,
			Owner:                   owner,
			UseExistingAWSResources: useExistingResources,
			DeployMysqlOperator:     deployMySQLOperator,
			DeployMinioOperator:     deployMinioOperator,
			NdotsValue:              ndotsDefaultValue,
			PGBouncerConfig:         pgbouncerConfig,
		}

		// TODO: In the future we can support both provisioners running
		// at the same time, and the correct one should be chosen based
		// on request. For now for simplicity we configure it with a
		// flag.
		var clusterProvisioner Provisioner
		switch provisionerFlag {
		case provisioner.KopsProvisionerType:
			kopsProvisioner := provisioner.NewKopsProvisioner(
				provisioningParams,
				resourceUtil,
				logger,
				sqlStore,
				provisioner.NewBackupOperator(backupRestoreToolImage, awsRegion, backupJobTTL),
			)
			defer kopsProvisioner.Teardown()
			clusterProvisioner = kopsProvisioner
		case provisioner.EKSProvisionerType:
			eksProvisioner := provisioner.NewEKSProvisioner(sqlStore,
				sqlStore,
				provisioningParams,
				resourceUtil,
				awsClient,
				logger)

			clusterProvisioner = eksProvisioner
		default:
			return errors.Errorf("invalid value for provisioner flag %q, expected one of: kops, eks", provisionerFlag)
		}

		cloudMetrics := metrics.New()

		delivererCfg := events.DelivererConfig{
			RetryWorkers:    2,
			UpToDateWorkers: 2,
			MaxBurstWorkers: 100,
		}
		deliveryCtx, deliveryCancel := context.WithCancel(context.Background())
		eventsDeliverer := events.NewDeliverer(deliveryCtx, sqlStore, instanceID, logger, delivererCfg)
		defer deliveryCancel()

		eventsProducer := events.NewProducer(sqlStore, eventsDeliverer, awsClient.GetCloudEnvironmentName(), logger)

		// DNS configuration
		dnsManager := supervisor.NewDNSManager()
		if enableRoute53 {
			dnsManager.AddProvider(supervisor.NewRoute53DNSProvider(awsClient))
		} else {
			logger.Warn("Route53 disabled for Installation, Route53 CNAME records will not be created")
		}

		if cloudflareToken := os.Getenv("CLOUDFLARE_API_TOKEN"); cloudflareToken != "" {
			cfClient, err := cf.NewWithAPIToken(cloudflareToken)
			if err != nil {
				return errors.Wrap(err, "failed to initialize cloudflare client using API token")
			}
			dnsManager.AddProvider(cloudflare.NewClientWithToken(cfClient, awsClient))
		} else {
			logger.Warn("Cloudflare token not provided, Cloudflare records registration will be skipped")
		}

		err = dnsManager.IsValid()
		if err != nil {
			return errors.Wrap(err, "invalid DNS providers configuration")
		}

		var multiDoer supervisor.MultiDoer
		if supervisorsEnabled["clusterSupervisor"] {
			multiDoer = append(multiDoer, supervisor.NewClusterSupervisor(sqlStore, clusterProvisioner, awsClient, eventsProducer, instanceID, logger))
		}
		if supervisorsEnabled["groupSupervisor"] {
			multiDoer = append(multiDoer, supervisor.NewGroupSupervisor(sqlStore, eventsProducer, instanceID, logger))
		}
		if supervisorsEnabled["installationSupervisor"] {
			multiDoer = append(multiDoer, supervisor.NewInstallationSupervisor(sqlStore, clusterProvisioner, awsClient, instanceID, keepDatabaseData, keepFilestoreData, installationScheduling, resourceUtil, logger, cloudMetrics, eventsProducer, forceCRUpgrade, dnsManager, disableDNSUpdates))
		}
		if supervisorsEnabled["clusterInstallationSupervisor"] {
			multiDoer = append(multiDoer, supervisor.NewClusterInstallationSupervisor(sqlStore, clusterProvisioner, awsClient, eventsProducer, instanceID, logger, cloudMetrics))
		}
		if supervisorsEnabled["backupSupervisor"] {
			multiDoer = append(multiDoer, supervisor.NewBackupSupervisor(sqlStore, clusterProvisioner, awsClient, instanceID, logger))
		}
		if supervisorsEnabled["importSupervisor"] {
			awatAddress, _ := command.Flags().GetString("awat")
			if awatAddress == "" {
				return errors.New("--awat flag must be provided when --import-supervisor flag is provided")
			}
			multiDoer = append(multiDoer, supervisor.NewImportSupervisor(awsClient, awat.NewClient(awatAddress), sqlStore, clusterProvisioner, eventsProducer, logger))
		}
		if supervisorsEnabled["installationDBRestorationSupervisor"] {
			multiDoer = append(multiDoer, supervisor.NewInstallationDBRestorationSupervisor(sqlStore, awsClient, clusterProvisioner, eventsProducer, instanceID, logger))
		}
		if supervisorsEnabled["installationDBMigrationSupervisor"] {
			multiDoer = append(multiDoer, supervisor.NewInstallationDBMigrationSupervisor(sqlStore, awsClient, resourceUtil, instanceID, clusterProvisioner, eventsProducer, logger))
		}

		// Setup the supervisor to effect any requested changes. It is wrapped in a
		// scheduler to trigger it periodically in addition to being poked by the API
		// layer.
		poll, _ := command.Flags().GetInt("poll")
		if poll == 0 {
			logger.WithField("poll", poll).Info("Scheduler is disabled")
		}

		standardSupervisor := supervisor.NewScheduler(multiDoer, time.Duration(poll)*time.Second)
		defer standardSupervisor.Close()

		slowPoll, _ := command.Flags().GetInt("slow-poll")
		if slowPoll == 0 {
			logger.WithField("slow-poll", slowPoll).Info("Slow scheduler is disabled")
		}
		if supervisorsEnabled["installationDeletionSupervisor"] {
			var slowMultiDoer supervisor.MultiDoer
			slowMultiDoer = append(slowMultiDoer, supervisor.NewInstallationDeletionSupervisor(instanceID, installationDeletionPendingTime, installationDeletionMaxUpdating, sqlStore, eventsProducer, logger))
			slowSupervisor := supervisor.NewScheduler(slowMultiDoer, time.Duration(slowPoll)*time.Second)
			defer slowSupervisor.Close()
		}

		metricsPort, _ := command.Flags().GetInt("metrics-port")
		metricsRouter := mux.NewRouter()
		metricsRouter.Handle("/metrics", promhttp.Handler())

		metricsServer := &http.Server{
			Addr:           fmt.Sprintf(":%d", metricsPort),
			Handler:        metricsRouter,
			ReadTimeout:    180 * time.Second,
			WriteTimeout:   180 * time.Second,
			IdleTimeout:    time.Second * 180,
			MaxHeaderBytes: 1 << 20,
			ErrorLog:       log.New(&logrusWriter{logger: logger}, "", 0),
		}

		go func() {
			logger.WithField("addr", metricsServer.Addr).Info("Metrics server listening")
			err := metricsServer.ListenAndServe()
			if err != nil && err != http.ErrServerClosed {
				logger.WithError(err).Error("Failed to listen and serve metrics")
			}
		}()

		router := mux.NewRouter()

		api.Register(router, &api.Context{
			Store:         sqlStore,
			Supervisor:    standardSupervisor,
			Provisioner:   clusterProvisioner,
			DBProvider:    resourceUtil,
			EventProducer: eventsProducer,
			Environment:   awsClient.GetCloudEnvironmentName(),
			AwsClient:     awsClient,
			Metrics:       cloudMetrics,
			Logger:        logger,
		})

		listen, _ := command.Flags().GetString("listen")
		srv := &http.Server{
			Addr:           listen,
			Handler:        router,
			ReadTimeout:    180 * time.Second,
			WriteTimeout:   180 * time.Second,
			IdleTimeout:    time.Second * 180,
			MaxHeaderBytes: 1 << 20,
			ErrorLog:       log.New(&logrusWriter{logger}, "", 0),
		}

		go func() {
			logger.WithField("addr", srv.Addr).Info("API server listening")
			err := srv.ListenAndServe()
			if err != nil && err != http.ErrServerClosed {
				logger.WithError(err).Error("Failed to listen and serve")
			}
		}()

		c := make(chan os.Signal, 1)
		// We'll accept graceful shutdowns when quit via:
		//  - SIGINT (Ctrl+C)
		//  - SIGTERM (Ctrl+/) (Kubernetes pod rolling termination)
		// SIGKILL and SIGQUIT will not be caught.
		signal.Notify(c, os.Interrupt, syscall.SIGTERM)
		// Important:
		// There are long-lived serial processes in the supervisors (especially
		// the cluster supervisor). It is quite possible that these will still
		// be terminated before completion if the k8s rolling grace period is
		// too low. Handling this will require further improvements.

		// Block until we receive a valid signal.
		sig := <-c
		logger.WithField("shutdown-signal", sig.String()).Info("Shutting down")

		ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
		defer cancel()
		srv.Shutdown(ctx)

		return nil
	},
}

func dbClusterUtilizationSettingsFromFlags(command *cobra.Command) utils.DBClusterUtilizationSettings {
	pgbouncer, _ := command.Flags().GetInt("max-installations-rds-postgres-pgbouncer")
	postgres, _ := command.Flags().GetInt("max-installations-rds-postgres")
	mysql, _ := command.Flags().GetInt("max-installations-rds-mysql")

	return utils.DBClusterUtilizationSettings{
		MaxInstallationsRDSPostgresPGBouncer: pgbouncer,
		MaxInstallationsRDSPostgres:          postgres,
		MaxInstallationsRDSMySQL:             mysql,
	}
}

func checkRequirements(logger logrus.FieldLogger) error {
	// Check for required tool binaries.
	silentLogger := logrus.New()
	silentLogger.Out = ioutil.Discard

	terraformClient, err := terraform.New(".", "dummy-remote-state", silentLogger)
	if err != nil {
		return errors.Wrap(err, "failed terraform client health check")
	}
	version, err := terraformClient.Version(true)
	if err != nil {
		return errors.Wrap(err, "failed to get terraform version")
	}
	logger.Infof("[startup-check] Using terraform: %s", version)

	kopsClient, err := kops.New("dummy-state-store", silentLogger)
	if err != nil {
		return errors.Wrap(err, "failed kops client health check")
	}
	version, err = kopsClient.Version()
	if err != nil {
		return errors.Wrap(err, "failed to get kops version")
	}
	logger.Infof("[startup-check] Using kops: %s", version)

	helmClient, err := helm.New(silentLogger)
	if err != nil {
		return errors.Wrap(err, "failed helm client health check")
	}
	version, err = helmClient.Version()
	if err != nil {
		return errors.Wrap(err, "failed to get helm version")
	}
	logger.Infof("[startup-check] Using helm: %s", version)

	// Check for extra tools that don't have a wrapper, but are still required.
	extraTools := []string{
		"kubectl",
	}
	for _, extraTool := range extraTools {
		_, err := exec.LookPath(extraTool)
		if err != nil {
			return errors.Errorf("failed to find %s on the PATH", extraTool)
		}
	}

	// Check for SSH keys.
	homedir, err := os.UserHomeDir()
	if err != nil {
		return errors.Wrap(err, "failed to determine the current user's home directory")
	}
	sshDir := path.Join(homedir, ".ssh")
	possibleKeys, err := ioutil.ReadDir(sshDir)
	if err != nil {
		return errors.Wrapf(err, "failed to find a SSH key in %s", sshDir)

	}
	hasKeys := func() bool {
		for _, k := range possibleKeys {
			if k.IsDir() {
				continue
			}
			keyFile, err := os.Open(path.Join(sshDir, k.Name()))
			if err != nil {
				continue
			}
			prefix := make([]byte, 3)
			read, err := keyFile.Read(prefix)
			if read == 0 || err != nil {
				continue
			}
			if string(prefix) == "ssh" {
				return true
			}
		}
		return false
	}()
	if !hasKeys {
		return errors.Errorf("failed to find an SSH key in %s", homedir)
	}

	return nil
}

// deprecationWarnings performs all checks for deprecated settings and warns if
// any are found.
func deprecationWarnings(logger logrus.FieldLogger, cmd *cobra.Command) {
	// Add deprecation logic here.
}

// getHumanReadableID  represents  a  best  effort  attempt  to  retrieve  an
// identifiable  human to  associate with  a given  provisioner. Since
// this is for dev workflows, any  error causes it to merely return a
// generic string.
func getHumanReadableID() string {
	envVar := os.Getenv("CLOUD_SERVER_OWNER")
	if envVar != "" {
		return envVar
	}

	cmd := exec.Command("git", "config",
		"--get", "user.email")

	output, err := cmd.Output()
	if err != nil {
		logger.Debugf("Couldn't determine username of developer with which to tag infrastructure due to error: %s", err.Error())
		if len(output) != 0 {
			logger.Debugf("Command output was: %s", string(output))
		}
		return "SRETeam"
	}

	return strings.TrimSpace(string(output))
}

func flagIsUnset(cmd *cobra.Command, flagName string) bool {
	return !cmd.Flags().Changed(flagName)
}

func isAny(conditions []bool) bool {
	for _, b := range conditions {
		if b {
			return true
		}
	}
	return false
}

func isAnyMapValue(conditions map[string]bool) bool {
	for _, b := range conditions {
		if b {
			return true
		}
	}
	return false
}
