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
	var sf serverFlags

	cmd := &cobra.Command{
		Use:   "server",
		Short: "Run the provisioning server.",
		RunE: func(command *cobra.Command, args []string) error {
			return executeServerCmd(sf)
		},
		PreRun: func(command *cobra.Command, args []string) {
			command.SilenceUsage = true
			sf.serverFlagChanged.addFlags(command) // To populate flag change variables.
			deprecationWarnings(logger, command)
		},
	}
	sf.addFlags(cmd)

	return cmd
}

func executeServerCmd(sf serverFlags) error {

	debugMode := sf.debug || (sf.devMode && !sf.isDebugChanged)
	if debugMode {
		logger.SetLevel(logrus.DebugLevel)
	}

	helm.SetVerboseHelmLogging(sf.debugHelm)

	if err := model.SetDefaultProxyDatabaseMaxInstallationsPerLogicalDatabase(sf.maxSchemas); err != nil {
		return err
	}

	pgbouncerConfig := provisioner.NewPGBouncerConfig(
		sf.minPoolSize, sf.defaultPoolSize, sf.reservePoolSize,
		sf.maxClientConnections, sf.maxDatabaseConnectionsPerPool,
		sf.serverIdleTimeout, sf.serverLifetime, sf.serverResetQueryAlways,
	)

	if err := pgbouncerConfig.Validate(); err != nil {
		return errors.Wrap(err, "pgbouncer config failed validation")
	}

	gitlabOAuthToken := sf.gitlabOAuthToken
	if len(gitlabOAuthToken) == 0 {
		gitlabOAuthToken = os.Getenv(model.GitlabOAuthTokenKey)
	}
	model.SetGitlabToken(gitlabOAuthToken)
	if len(model.GetGitlabToken()) == 0 {
		logger.Warnf("The gitlab-oauth flag and %s were empty; using local helm charts", model.GitlabOAuthTokenKey)
	}

	if sf.machineLogs {
		logger.SetFormatter(&logrus.JSONFormatter{})
	}

	model.SetRequireAnnotatedInstallations(sf.requireAnnotatedInstallations)

	if len(sf.allowListCIDRRange) == 0 {
		return errors.New("allow-list-cidr-range must have at least one value")
	}

	if len(sf.vpnListCIDR) == 0 {
		return errors.New("vpn-list-cidr must have at least one value")
	}

	if sf.mattermostWebHook != "" {
		_ = os.Setenv(model.MattermostWebhook, sf.mattermostWebHook)
	}

	if sf.mattermostChannel != "" {
		_ = os.Setenv(model.MattermostChannel, sf.mattermostChannel)
	}

	if sf.utilitiesGitURL == "" {
		return errors.New("utilities-git-url must be set")
	}
	model.SetUtilityDefaults(sf.utilitiesGitURL)

	if sf.kubecostToken != "" {
		_ = os.Setenv(model.KubecostToken, sf.kubecostToken)
	}

	logger := logger.WithField("instance", instanceID)

	sqlStore, err := sqlStore(sf.database)
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
		sf.balancedInstallationScheduling,
		sf.clusterResourceThreshold,
		sf.thresholdCPUOverride,
		sf.thresholdMemoryOverride,
		sf.thresholdPodCountOverride,
		sf.clusterResourceThresholdScaleValue,
	)

	if err := installationScheduling.Validate(); err != nil {
		return errors.Wrap(err, "invalid installation scheduling options")
	}

	supervisorsEnabled := sf.supervisorOptions
	if sf.disableAllSupervisors {
		supervisorsEnabled = supervisorOptions{} // reset to zero
	}

	if supervisorsEnabled == (supervisorOptions{}) {
		logger.Warn("Server will be running with no supervisors. Only API functionality will work.")
	}

	model.SetDeployOperators(sf.deployMySQLOperator, sf.deployMinioOperator)

		etcdQuotaBackendBytes, _ := command.Flags().GetInt("etcd-quota-backend-bytes")
		etcdListenMetricsURL, _ := command.Flags().GetString("etcd-listen-metrics-url")

		etcdManagerEnv := map[string]string{
			"ETCD_QUOTA_BACKEND_BYTES": fmt.Sprintf("%v", etcdQuotaBackendBytes),
			"ETCD_LISTEN_METRICS_URLS": etcdListenMetricsURL,
		}

		wd, err := os.Getwd()
		if err != nil {
			wd = "error getting working directory"
			logger.WithError(err).Error("Unable to get current working directory")
		}

	keepDatabaseData := sf.keepDatabaseData
	keepFileStoreData := sf.keepFileStoreData
	if sf.devMode {
		if !sf.isKeepDatabaseDataChanged {
			keepDatabaseData = false
		}
		if !sf.isKeepFileStoreDataChanged {
			keepFileStoreData = false
		}
	}

	provisionerFlag := sf.provisioner

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
		"state-store":                                   sf.s3StateStore,
		"working-directory":                             wd,
		"installation-deletion-pending-time":            sf.installationDeletionPendingTime,
		"installation-deletion-max-updating":            sf.installationDeletionMaxUpdating,
		"balanced-installation-scheduling":              sf.balancedInstallationScheduling,
		"cluster-resource-threshold":                    sf.clusterResourceThreshold,
		"cluster-resource-threshold-cpu-override":       sf.thresholdCPUOverride,
		"cluster-resource-threshold-memory-override":    sf.thresholdMemoryOverride,
		"cluster-resource-threshold-pod-count-override": sf.thresholdPodCountOverride,
		"cluster-resource-threshold-scale-value":        sf.clusterResourceThresholdScaleValue,
		"use-existing-aws-resources":                    sf.useExistingResources,
		"keep-database-data":                            keepDatabaseData,
		"keep-filestore-data":                           keepFileStoreData,
		"force-cr-upgrade":                              sf.forceCRUpgrade,
		"backup-restore-tool-image":                     sf.backupRestoreToolImage,
		"backup-job-ttl-seconds":                        sf.backupJobTTL,
		"debug":                                         debugMode,
		"dev-mode":                                      sf.devMode,
		"deploy-mysql-operator":                         sf.deployMySQLOperator,
		"deploy-minio-operator":                         sf.deployMinioOperator,
		"ndots-value":                                   sf.ndotsDefaultValue,
		"maxDatabaseConnectionsPerPool":                 sf.maxDatabaseConnectionsPerPool,
		"defaultPoolSize":                               sf.defaultPoolSize,
		"minPoolSize":                                   sf.minPoolSize,
		"maxClientConnections":                          sf.maxClientConnections,
		"disable-db-init-check":                         sf.disableDBInitCheck,
		"enable-route53":                                sf.enableRoute53,
		"disable-dns-updates":                           sf.disableDNSUpdates,
		"provisioner":                                   provisionerFlag,
	}).Info("Starting Mattermost Provisioning Server")

	// Warn on settings we consider to be non-production.
	if !sf.useExistingResources {
		logger.Warn("[DEV] Server is configured to not use cluster VPC claim functionality")
	}

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

	if err := checkRequirements(logger); err != nil {
		return errors.Wrap(err, "failed health check")
	}

	// best-effort attempt to tag the VPC with a human's identity for dev purposes
	owner := getHumanReadableID()

	provisioningParams := provisioner.ProvisioningParams{
		S3StateStore:            sf.s3StateStore,
		AllowCIDRRangeList:      sf.allowListCIDRRange,
		VpnCIDRList:             sf.vpnListCIDR,
		Owner:                   owner,
		UseExistingAWSResources: sf.useExistingResources,
		DeployMysqlOperator:     sf.deployMySQLOperator,
		DeployMinioOperator:     sf.deployMinioOperator,
		NdotsValue:              sf.ndotsDefaultValue,
		PGBouncerConfig:         pgbouncerConfig,
		SLOInstallationGroups:   sf.sloInstallationGroups,
		EtcdManagerEnv:          etcdManagerEnv,
	}

	resourceUtil := utils.NewResourceUtil(instanceID, awsClient, dbClusterUtilizationSettingsFromFlags(sf), sf.disableDBInitCheck)

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
			provisioner.NewBackupOperator(sf.backupRestoreToolImage, awsRegion, sf.backupJobTTL),
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
	if sf.enableRoute53 {
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

	if err := dnsManager.IsValid(); err != nil {
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
		multiDoer = append(multiDoer, supervisor.NewInstallationSupervisor(sqlStore, clusterProvisioner, awsClient, instanceID, keepDatabaseData, keepFileStoreData, installationScheduling, resourceUtil, logger, cloudMetrics, eventsProducer, sf.forceCRUpgrade, dnsManager, sf.disableDNSUpdates))
	}
	if supervisorsEnabled.clusterInstallationSupervisor {
		multiDoer = append(multiDoer, supervisor.NewClusterInstallationSupervisor(sqlStore, clusterProvisioner, awsClient, eventsProducer, instanceID, logger, cloudMetrics))
	}
	if supervisorsEnabled.backupSupervisor {
		multiDoer = append(multiDoer, supervisor.NewBackupSupervisor(sqlStore, clusterProvisioner, awsClient, instanceID, logger))
	}
	if supervisorsEnabled.importSupervisor {
		if sf.awatAddress == "" {
			return errors.New("--awat flag must be provided when --import-supervisor flag is provided")
		}
		multiDoer = append(multiDoer, supervisor.NewImportSupervisor(awsClient, awat.NewClient(sf.awatAddress), sqlStore, clusterProvisioner, eventsProducer, logger))
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
	if sf.poll == 0 {
		logger.WithField("poll", sf.poll).Info("Scheduler is disabled")
	}

	standardSupervisor := supervisor.NewScheduler(multiDoer, time.Duration(sf.poll)*time.Second)
	defer standardSupervisor.Close()

	if sf.slowPoll == 0 {
		logger.WithField("slow-poll", sf.slowPoll).Info("Slow scheduler is disabled")
	}
	if supervisorsEnabled.installationDeletionSupervisor {
		var slowMultiDoer supervisor.MultiDoer
		slowMultiDoer = append(slowMultiDoer, supervisor.NewInstallationDeletionSupervisor(instanceID, sf.installationDeletionPendingTime, sf.installationDeletionMaxUpdating, sqlStore, eventsProducer, logger))
		slowSupervisor := supervisor.NewScheduler(slowMultiDoer, time.Duration(sf.slowPoll)*time.Second)
		defer slowSupervisor.Close()
	}

	metricsRouter := mux.NewRouter()
	metricsRouter.Handle("/metrics", promhttp.Handler())

	metricsServer := &http.Server{
		Addr:           fmt.Sprintf(":%d", sf.metricsPort),
		Handler:        metricsRouter,
		ReadTimeout:    180 * time.Second,
		WriteTimeout:   180 * time.Second,
		IdleTimeout:    time.Second * 180,
		MaxHeaderBytes: 1 << 20,
		ErrorLog:       log.New(&logrusWriter{logger: logger}, "", 0),
	}

	go func() {
		logger.WithField("addr", metricsServer.Addr).Info("Metrics server listening")
		if err := metricsServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
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
		Addr:           sf.listen,
		Handler:        router,
		ReadTimeout:    180 * time.Second,
		WriteTimeout:   180 * time.Second,
		IdleTimeout:    time.Second * 180,
		MaxHeaderBytes: 1 << 20,
		ErrorLog:       log.New(&logrusWriter{logger}, "", 0),
	}

	go func() {
		logger.WithField("addr", srv.Addr).Info("API server listening")
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
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
	if err := srv.Shutdown(ctx); err != nil {
		logger.WithField("err", err.Error()).Error("error shutting down server")
	}

	return nil
}

func dbClusterUtilizationSettingsFromFlags(sf serverFlags) utils.DBClusterUtilizationSettings {
	return utils.DBClusterUtilizationSettings{
		MaxInstallationsRDSPostgresPGBouncer: sf.pgbouncer,
		MaxInstallationsRDSPostgres:          sf.postgres,
		MaxInstallationsRDSMySQL:             sf.mysql,
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
