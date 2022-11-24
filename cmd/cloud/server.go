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

	sdkAWS "github.com/aws/aws-sdk-go/aws"
	cf "github.com/cloudflare/cloudflare-go"
	"github.com/gorilla/mux"
	awat "github.com/mattermost/awat/model"
	"github.com/mattermost/mattermost-cloud/internal/api"
	"github.com/mattermost/mattermost-cloud/internal/events"
	"github.com/mattermost/mattermost-cloud/internal/metrics"
	"github.com/mattermost/mattermost-cloud/internal/provisioner"
	"github.com/mattermost/mattermost-cloud/internal/store"
	"github.com/mattermost/mattermost-cloud/internal/supervisor"
	toolsAWS "github.com/mattermost/mattermost-cloud/internal/tools/aws"
	"github.com/mattermost/mattermost-cloud/internal/tools/cloudflare"
	"github.com/mattermost/mattermost-cloud/internal/tools/helm"
	"github.com/mattermost/mattermost-cloud/internal/tools/kops"
	"github.com/mattermost/mattermost-cloud/internal/tools/terraform"
	"github.com/mattermost/mattermost-cloud/internal/tools/utils"
	"github.com/mattermost/mattermost-cloud/model"
	"github.com/pkg/errors"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

const defaultLocalServerAPI = "http://localhost:8075"

// Provisioner is an interface for different types of provisioners.
type Provisioner interface {
	api.Provisioner
	supervisor.Provisioner
}

func newCmdServer() *cobra.Command {
	var serverFlags ServerFlags

	cmd := &cobra.Command{
		Use:   "server",
		Short: "Run the provisioning server.",
		RunE: func(command *cobra.Command, args []string) error {
			return executeServerCmd(serverFlags)
		},
		PreRun: func(command *cobra.Command, args []string) {
			command.SilenceUsage = true
			serverFlags.serverFlagChanged.AddFlags(command)
			deprecationWarnings(logger, command)
		},
	}
	serverFlags.AddFlags(cmd)

	return cmd
}

func executeServerCmd(serverFlags ServerFlags) error {

	devMode := serverFlags.devMode
	debugMode := serverFlags.debug || (devMode && !serverFlags.isDebugChanged)
	if debugMode {
		logger.SetLevel(logrus.DebugLevel)
	}

	helm.SetVerboseHelmLogging(serverFlags.debugHelm)

	err := model.SetDefaultProxyDatabaseMaxInstallationsPerLogicalDatabase(serverFlags.maxSchemas)
	if err != nil {
		return err
	}

	pgbouncerConfig := provisioner.NewPGBouncerConfig(
		serverFlags.minPoolSize, serverFlags.defaultPoolSize, serverFlags.reservePoolSize,
		serverFlags.maxClientConnections, serverFlags.maxDatabaseConnectionsPerPool,
		serverFlags.serverIdleTimeout, serverFlags.serverLifetime, serverFlags.serverResetQueryAlways,
	)
	err = pgbouncerConfig.Validate()
	if err != nil {
		return errors.Wrap(err, "pgbouncer config failed validation")
	}

	gitlabOAuthToken := serverFlags.gitlabOAuthToken
	if len(gitlabOAuthToken) == 0 {
		gitlabOAuthToken = os.Getenv(model.GitlabOAuthTokenKey)
	}
	model.SetGitlabToken(gitlabOAuthToken)
	if len(model.GetGitlabToken()) == 0 {
		logger.Warnf("The gitlab-oauth flag and %s were empty; using local helm charts", model.GitlabOAuthTokenKey)
	}

	if serverFlags.machineLogs {
		logger.SetFormatter(&logrus.JSONFormatter{})
	}

	model.SetRequireAnnotatedInstallations(serverFlags.requireAnnotatedInstallations)

	if len(serverFlags.allowListCIDRRange) == 0 {
		return errors.New("allow-list-cidr-range must have at least one value")
	}

	if len(serverFlags.vpnListCIDR) == 0 {
		return errors.New("vpn-list-cidr must have at least one value")
	}

	if serverFlags.mattermostWebHook != "" {
		_ = os.Setenv(model.MattermostWebhook, serverFlags.mattermostWebHook)
	}

	if serverFlags.mattermostChannel != "" {
		_ = os.Setenv(model.MattermostChannel, serverFlags.mattermostChannel)
	}

	if serverFlags.utilitiesGitURL == "" {
		return errors.New("utilities-git-url must be set")
	}
	model.SetUtilityDefaults(serverFlags.utilitiesGitURL)

	if serverFlags.kubeCostToken != "" {
		_ = os.Setenv(model.KubecostToken, serverFlags.kubeCostToken)
	}

	logger := logger.WithField("instance", instanceID)

	sqlStore, err := sqlStore(serverFlags.database)
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
	installationScheduling := supervisor.NewInstallationSupervisorSchedulingOptions(
		serverFlags.balancedInstallationScheduling,
		serverFlags.clusterResourceThreshold,
		serverFlags.thresholdCPUOverride,
		serverFlags.thresholdMemoryOverride,
		serverFlags.thresholdPodCountOverride,
		serverFlags.clusterResourceThresholdScaleValue,
	)
	err = installationScheduling.Validate()
	if err != nil {
		return errors.Wrap(err, "invalid installation scheduling options")
	}

	supervisorsEnabled := serverFlags.supervisorOption
	if serverFlags.disableAllSupervisors {
		supervisorsEnabled = supervisorOption{} // reset to zero
	}

	if supervisorsEnabled == (supervisorOption{}) {
		logger.Warn("Server will be running with no supervisors. Only API functionality will work.")
	}

	model.SetDeployOperators(serverFlags.deployMySQLOperator, serverFlags.deployMinioOperator)

	wd, err := os.Getwd()
	if err != nil {
		wd = "error getting working directory"
		logger.WithError(err).Error("Unable to get current working directory")
	}

	keepDatabaseData := serverFlags.keepDatabaseData
	keepFileStoreData := serverFlags.keepFileStoreData
	if devMode {
		if !serverFlags.isKeepDatabaseDataChanged {
			keepDatabaseData = false
		}
		if !serverFlags.isKeepFileStoreDataChanged {
			keepFileStoreData = false
		}
	}

	provisionerFlag := serverFlags.provisioner
	logger.WithFields(logrus.Fields{
		"build-hash":                                    model.BuildHash,
		"cluster-supervisor":                            supervisorsEnabled.clusterSupervisor,
		"group-supervisor":                              supervisorsEnabled.groupSupervisor,
		"installation-supervisor":                       supervisorsEnabled.installationSupervisor,
		"installation-deletion-supervisor":              supervisorsEnabled.installationDeletionSupervisor,
		"cluster-installation-supervisor":               supervisorsEnabled.clusterInstallationSupervisor,
		"backup-supervisor":                             supervisorsEnabled.backupSupervisor,
		"import-supervisor":                             supervisorsEnabled.importSupervisor,
		"installation-db-restoration-supervisor":        supervisorsEnabled.installationDBRestorationSupervisor,
		"installation-db-migration-supervisor":          supervisorsEnabled.installationDBMigrationSupervisor,
		"store-version":                                 currentVersion,
		"state-store":                                   serverFlags.s3StateStore,
		"working-directory":                             wd,
		"installation-deletion-pending-time":            serverFlags.installationDeletionPendingTime,
		"installation-deletion-max-updating":            serverFlags.installationDeletionMaxUpdating,
		"balanced-installation-scheduling":              serverFlags.balancedInstallationScheduling,
		"cluster-resource-threshold":                    serverFlags.clusterResourceThreshold,
		"cluster-resource-threshold-cpu-override":       serverFlags.thresholdCPUOverride,
		"cluster-resource-threshold-memory-override":    serverFlags.thresholdMemoryOverride,
		"cluster-resource-threshold-pod-count-override": serverFlags.thresholdPodCountOverride,
		"cluster-resource-threshold-scale-value":        serverFlags.clusterResourceThresholdScaleValue,
		"use-existing-aws-resources":                    serverFlags.useExistingResources,
		"keep-database-data":                            keepDatabaseData,
		"keep-filestore-data":                           keepFileStoreData,
		"force-cr-upgrade":                              serverFlags.forceCRUpgrade,
		"backup-restore-tool-image":                     serverFlags.backupRestoreToolImage,
		"backup-job-ttl-seconds":                        serverFlags.backupJobTTL,
		"debug":                                         debugMode,
		"dev-mode":                                      serverFlags.devMode,
		"deploy-mysql-operator":                         serverFlags.deployMySQLOperator,
		"deploy-minio-operator":                         serverFlags.deployMinioOperator,
		"ndots-value":                                   serverFlags.ndotsDefaultValue,
		"maxDatabaseConnectionsPerPool":                 serverFlags.maxDatabaseConnectionsPerPool,
		"defaultPoolSize":                               serverFlags.defaultPoolSize,
		"minPoolSize":                                   serverFlags.minPoolSize,
		"maxClientConnections":                          serverFlags.maxClientConnections,
		"disable-db-init-check":                         serverFlags.disableDBInitCheck,
		"enable-route53":                                serverFlags.enableRoute53,
		"disable-dns-updates":                           serverFlags.disableDNSUpdates,
		"provisioner":                                   provisionerFlag,
	}).Info("Starting Mattermost Provisioning Server")

	// Warn on settings we consider to be non-production.
	if !serverFlags.useExistingResources {
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

	resourceUtil := utils.NewResourceUtil(instanceID, awsClient, dbClusterUtilizationSettingsFromFlags(serverFlags), serverFlags.disableDBInitCheck)

	provisioningParams := provisioner.ProvisioningParams{
		S3StateStore:            serverFlags.s3StateStore,
		AllowCIDRRangeList:      serverFlags.allowListCIDRRange,
		VpnCIDRList:             serverFlags.vpnListCIDR,
		Owner:                   owner,
		UseExistingAWSResources: serverFlags.useExistingResources,
		DeployMysqlOperator:     serverFlags.deployMySQLOperator,
		DeployMinioOperator:     serverFlags.deployMinioOperator,
		NdotsValue:              serverFlags.ndotsDefaultValue,
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
			provisioner.NewBackupOperator(serverFlags.backupRestoreToolImage, awsRegion, serverFlags.backupJobTTL),
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
	if serverFlags.enableRoute53 {
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
	if supervisorsEnabled.clusterSupervisor {
		multiDoer = append(multiDoer, supervisor.NewClusterSupervisor(sqlStore, clusterProvisioner, awsClient, eventsProducer, instanceID, logger))
	}
	if supervisorsEnabled.groupSupervisor {
		multiDoer = append(multiDoer, supervisor.NewGroupSupervisor(sqlStore, eventsProducer, instanceID, logger))
	}
	if supervisorsEnabled.installationSupervisor {
		multiDoer = append(multiDoer, supervisor.NewInstallationSupervisor(sqlStore, clusterProvisioner, awsClient, instanceID, keepDatabaseData, keepFileStoreData, installationScheduling, resourceUtil, logger, cloudMetrics, eventsProducer, serverFlags.forceCRUpgrade, dnsManager, serverFlags.disableDNSUpdates))
	}
	if supervisorsEnabled.clusterInstallationSupervisor {
		multiDoer = append(multiDoer, supervisor.NewClusterInstallationSupervisor(sqlStore, clusterProvisioner, awsClient, eventsProducer, instanceID, logger, cloudMetrics))
	}
	if supervisorsEnabled.backupSupervisor {
		multiDoer = append(multiDoer, supervisor.NewBackupSupervisor(sqlStore, clusterProvisioner, awsClient, instanceID, logger))
	}
	if supervisorsEnabled.importSupervisor {
		awatAddress := serverFlags.awatAddress
		if awatAddress == "" {
			return errors.New("--awat flag must be provided when --import-supervisor flag is provided")
		}
		multiDoer = append(multiDoer, supervisor.NewImportSupervisor(awsClient, awat.NewClient(awatAddress), sqlStore, clusterProvisioner, eventsProducer, logger))
	}
	if supervisorsEnabled.installationDBRestorationSupervisor {
		multiDoer = append(multiDoer, supervisor.NewInstallationDBRestorationSupervisor(sqlStore, awsClient, clusterProvisioner, eventsProducer, instanceID, logger))
	}
	if supervisorsEnabled.installationDBMigrationSupervisor {
		multiDoer = append(multiDoer, supervisor.NewInstallationDBMigrationSupervisor(sqlStore, awsClient, resourceUtil, instanceID, clusterProvisioner, eventsProducer, logger))
	}

	// Setup the supervisor to effect any requested changes. It is wrapped in a
	// scheduler to trigger it periodically in addition to being poked by the API
	// layer.
	poll := serverFlags.poll
	if poll == 0 {
		logger.WithField("poll", poll).Info("Scheduler is disabled")
	}

	standardSupervisor := supervisor.NewScheduler(multiDoer, time.Duration(poll)*time.Second)
	defer standardSupervisor.Close()

	slowPoll := serverFlags.slowPoll
	if slowPoll == 0 {
		logger.WithField("slow-poll", slowPoll).Info("Slow scheduler is disabled")
	}
	if supervisorsEnabled.installationDeletionSupervisor {
		var slowMultiDoer supervisor.MultiDoer
		slowMultiDoer = append(slowMultiDoer, supervisor.NewInstallationDeletionSupervisor(instanceID, serverFlags.installationDeletionPendingTime, serverFlags.installationDeletionMaxUpdating, sqlStore, eventsProducer, logger))
		slowSupervisor := supervisor.NewScheduler(slowMultiDoer, time.Duration(slowPoll)*time.Second)
		defer slowSupervisor.Close()
	}

	metricsRouter := mux.NewRouter()
	metricsRouter.Handle("/metrics", promhttp.Handler())

	metricsServer := &http.Server{
		Addr:           fmt.Sprintf(":%d", serverFlags.metricsPort),
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

	srv := &http.Server{
		Addr:           serverFlags.listen,
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
	_ = srv.Shutdown(ctx)

	return nil
}

func dbClusterUtilizationSettingsFromFlags(serverFlags ServerFlags) utils.DBClusterUtilizationSettings {
	return utils.DBClusterUtilizationSettings{
		MaxInstallationsRDSPostgresPGBouncer: serverFlags.pgbouncer,
		MaxInstallationsRDSPostgres:          serverFlags.postgres,
		MaxInstallationsRDSMySQL:             serverFlags.mysql,
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
