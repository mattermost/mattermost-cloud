// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.
//

package main

import (
	"context"
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
	"github.com/gorilla/mux"
	"github.com/mattermost/mattermost-cloud/internal/api"
	"github.com/mattermost/mattermost-cloud/internal/metrics"
	"github.com/mattermost/mattermost-cloud/internal/provisioner"
	"github.com/mattermost/mattermost-cloud/internal/store"
	"github.com/mattermost/mattermost-cloud/internal/supervisor"
	toolsAWS "github.com/mattermost/mattermost-cloud/internal/tools/aws"
	"github.com/mattermost/mattermost-cloud/internal/tools/utils"
	"github.com/mattermost/mattermost-cloud/model"
	"github.com/pkg/errors"
	logrus "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

const (
	defaultLocalServerAPI = "http://localhost:8075"
)

var instanceID string

func init() {
	instanceID = model.NewID()

	// General
	serverCmd.PersistentFlags().String("database", "sqlite://cloud.db", "The database backing the provisioning server.")
	serverCmd.PersistentFlags().String("listen", ":8075", "The interface and port on which to listen.")
	serverCmd.PersistentFlags().String("state-store", "dev.cloud.mattermost.com", "The S3 bucket used to store cluster state.")
	serverCmd.PersistentFlags().StringSlice("allow-list-cidr-range", []string{"0.0.0.0/0"}, "The list of CIDRs to allow communication with the private ingress.")
	serverCmd.PersistentFlags().StringSlice("vpn-list-cidr", []string{"0.0.0.0/0"}, "The list of VPN CIDRs to allow communication with the clusters.")
	serverCmd.PersistentFlags().Bool("debug", false, "Whether to output debug logs.")
	serverCmd.PersistentFlags().Bool("machine-readable-logs", false, "Output the logs in machine readable format.")
	serverCmd.PersistentFlags().Bool("dev", false, "Set sane defaults for development")

	// Supervisors
	serverCmd.PersistentFlags().Int("poll", 30, "The interval in seconds to poll for background work.")
	serverCmd.PersistentFlags().Bool("cluster-supervisor", true, "Whether this server will run a cluster supervisor or not.")
	serverCmd.PersistentFlags().Bool("group-supervisor", false, "Whether this server will run an installation group supervisor or not.")
	serverCmd.PersistentFlags().Bool("installation-supervisor", true, "Whether this server will run an installation supervisor or not.")
	serverCmd.PersistentFlags().Bool("cluster-installation-supervisor", true, "Whether this server will run a cluster installation supervisor or not.")

	// Scheduling and installation options
	serverCmd.PersistentFlags().Bool("balanced-installation-scheduling", false, "Whether to schedule installations on the cluster with the greatest percentage of available resources or not. (slows down scheduling speed as cluster count increases)")
	serverCmd.PersistentFlags().Int("cluster-resource-threshold", 80, "The percent threshold where new installations won't be scheduled on a multi-tenant cluster.")
	serverCmd.PersistentFlags().Int("cluster-resource-threshold-scale-value", 0, "The number of worker nodes to scale up by when the threshold is passed. Set to 0 for no scaling. Scaling will never exceed the cluster max worker configuration value.")
	serverCmd.PersistentFlags().Bool("use-existing-aws-resources", true, "Whether to use existing AWS resources (VPCs, subnets, etc.) or not.")
	serverCmd.PersistentFlags().Bool("keep-database-data", true, "Whether to preserve database data after installation deletion or not.")
	serverCmd.PersistentFlags().Bool("keep-filestore-data", true, "Whether to preserve filestore data after installation deletion or not.")
	serverCmd.PersistentFlags().Bool("require-annotated-installations", false, "Require new installations to have at least one annotation.")
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

		machineLogs, _ := command.Flags().GetBool("machine-readable-logs")
		if machineLogs {
			logger.SetFormatter(&logrus.JSONFormatter{})
		}

		requireAnnotatedInstallations, _ := command.Flags().GetBool("require-annotated-installations")
		model.SetRequireAnnotatedInstallations(requireAnnotatedInstallations)

		allowListCIDRRange, _ := command.Flags().GetStringSlice("allow-list-cidr-range")
		if len(allowListCIDRRange) == 0 {
			return errors.New("allow-list-cidr-range must have at least one value")
		}

		vpnListCIDR, _ := command.Flags().GetStringSlice("vpn-list-cidr")
		if len(vpnListCIDR) == 0 {
			return errors.New("vpn-list-cidr must have at least one value")
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
		clusterResourceThreshold, _ := command.Flags().GetInt("cluster-resource-threshold")
		if clusterResourceThreshold < 10 || clusterResourceThreshold > 100 {
			return errors.Errorf("cluster-resource-threshold (%d) must be set between 10 and 100", clusterResourceThreshold)
		}
		clusterResourceThresholdScaleValue, _ := command.Flags().GetInt("cluster-resource-threshold-scale-value")
		if clusterResourceThresholdScaleValue < 0 || clusterResourceThresholdScaleValue > 10 {
			return errors.Errorf("cluster-resource-threshold-scale-value (%d) must be set between 0 and 10", clusterResourceThresholdScaleValue)
		}

		clusterSupervisor, _ := command.Flags().GetBool("cluster-supervisor")
		groupSupervisor, _ := command.Flags().GetBool("group-supervisor")
		installationSupervisor, _ := command.Flags().GetBool("installation-supervisor")
		clusterInstallationSupervisor, _ := command.Flags().GetBool("cluster-installation-supervisor")
		if !clusterSupervisor && !installationSupervisor && !clusterInstallationSupervisor && !groupSupervisor {
			logger.Warn("Server will be running with no supervisors. Only API functionality will work.")
		}

		s3StateStore, _ := command.Flags().GetString("state-store")
		keepDatabaseData, _ := command.Flags().GetBool("keep-database-data")
		keepFilestoreData, _ := command.Flags().GetBool("keep-filestore-data")
		useExistingResources, _ := command.Flags().GetBool("use-existing-aws-resources")
		balancedInstallationScheduling, _ := command.Flags().GetBool("balanced-installation-scheduling")

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

		logger.WithFields(logrus.Fields{
			"build-hash":                             model.BuildHash,
			"cluster-supervisor":                     clusterSupervisor,
			"group-supervisor":                       groupSupervisor,
			"installation-supervisor":                installationSupervisor,
			"cluster-installation-supervisor":        clusterInstallationSupervisor,
			"store-version":                          currentVersion,
			"state-store":                            s3StateStore,
			"working-directory":                      wd,
			"balanced-installation-scheduling":       balancedInstallationScheduling,
			"cluster-resource-threshold":             clusterResourceThreshold,
			"cluster-resource-threshold-scale-value": clusterResourceThresholdScaleValue,
			"use-existing-aws-resources":             useExistingResources,
			"keep-database-data":                     keepDatabaseData,
			"keep-filestore-data":                    keepFilestoreData,
			"debug":                                  debugMode,
			"dev-mode":                               devMode,
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
		awsClient := toolsAWS.NewAWSClientWithConfig(awsConfig, logger)

		err = checkRequirements(awsConfig, s3StateStore)
		if err != nil {
			return errors.Wrap(err, "failed health check")
		}

		resourceUtil := utils.NewResourceUtil(instanceID, awsClient)

		// Setup the provisioner for actually effecting changes to clusters.
		kopsProvisioner := provisioner.NewKopsProvisioner(
			s3StateStore,
			owner,
			useExistingResources,
			allowListCIDRRange,
			vpnListCIDR,
			resourceUtil,
			logger,
			sqlStore,
		)

		cloudMetrics := metrics.New()

		var multiDoer supervisor.MultiDoer
		if clusterSupervisor {
			multiDoer = append(multiDoer, supervisor.NewClusterSupervisor(sqlStore, kopsProvisioner, awsClient, instanceID, logger))
		}
		if groupSupervisor {
			multiDoer = append(multiDoer, supervisor.NewGroupSupervisor(sqlStore, instanceID, logger))
		}
		if installationSupervisor {
			scheduling := supervisor.NewInstallationSupervisorSchedulingOptions(balancedInstallationScheduling, clusterResourceThreshold, clusterResourceThresholdScaleValue)
			multiDoer = append(multiDoer, supervisor.NewInstallationSupervisor(sqlStore, kopsProvisioner, awsClient, instanceID, keepDatabaseData, keepFilestoreData, scheduling, resourceUtil, logger, cloudMetrics))
		}
		if clusterInstallationSupervisor {
			multiDoer = append(multiDoer, supervisor.NewClusterInstallationSupervisor(sqlStore, kopsProvisioner, awsClient, instanceID, logger))
		}

		// Setup the supervisor to effect any requested changes. It is wrapped in a
		// scheduler to trigger it periodically in addition to being poked by the API
		// layer.
		poll, _ := command.Flags().GetInt("poll")
		if poll == 0 {
			logger.WithField("poll", poll).Info("Scheduler is disabled")
		}

		supervisor := supervisor.NewScheduler(multiDoer, time.Duration(poll)*time.Second)
		defer supervisor.Close()

		router := mux.NewRouter()

		api.Register(router, &api.Context{
			Store:       sqlStore,
			Supervisor:  supervisor,
			Provisioner: kopsProvisioner,
			Logger:      logger,
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
			logger.WithField("addr", srv.Addr).Info("Listening")
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

func checkRequirements(awsConfig *sdkAWS.Config, s3StateStore string) error {
	utilities := []string{
		"terraform",
		"kops",
		"kubectl",
		"helm",
	}

	for _, requiredUtility := range utilities {
		_, err := exec.LookPath(requiredUtility)
		if err != nil {
			return errors.Errorf("failed to find %s on the PATH", requiredUtility)
		}
	}

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
	client := toolsAWS.NewAWSClientWithConfig(awsConfig, logger)
	_, err = client.GetCloudEnvironmentName()
	if err != nil {
		return errors.Wrap(err, "failed to establish a connection with AWS")
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
