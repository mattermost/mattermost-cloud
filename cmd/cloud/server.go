package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"time"

	"github.com/gorilla/mux"
	"github.com/mattermost/mattermost-cloud/internal/api"
	"github.com/mattermost/mattermost-cloud/internal/provisioner"
	"github.com/mattermost/mattermost-cloud/internal/store"
	"github.com/mattermost/mattermost-cloud/internal/supervisor"
	"github.com/mattermost/mattermost-cloud/internal/tools/aws"
	"github.com/mattermost/mattermost-cloud/model"
	"github.com/pkg/errors"
	logrus "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

// clusterRootDir is the local directory that contains cluster configuration.
const clusterRootDir = "clusters"

var instanceID string

func init() {
	instanceID = model.NewID()

	serverCmd.PersistentFlags().String("database", "sqlite://cloud.db", "The database backing the provisioning server.")
	serverCmd.PersistentFlags().String("listen", ":8075", "The interface and port on which to listen.")
	serverCmd.PersistentFlags().Bool("cluster-supervisor", true, "Whether this server will run a cluster supervisor or not.")
	serverCmd.PersistentFlags().Bool("installation-supervisor", true, "Whether this server will run an installation supervisor or not.")
	serverCmd.PersistentFlags().Bool("cluster-installation-supervisor", true, "Whether this server will run a cluster installation supervisor or not.")
	serverCmd.PersistentFlags().String("state-store", "dev.cloud.mattermost.com", "The S3 bucket used to store cluster state.")
	serverCmd.PersistentFlags().String("certificate-aws-arn", "", "The certificate ARN from AWS. Generated in the certificate manager console.")
	serverCmd.PersistentFlags().String("route53-id", "", "The route 53 hosted zone ID used for mattermost DNS records.")
	serverCmd.PersistentFlags().String("private-route53-id", "", "The route 53 hosted zone ID used for mattermost private DNS records.")
	serverCmd.PersistentFlags().String("private-dns", "", "The DNS used for mattermost private Route53 records.")
	serverCmd.PersistentFlags().Int("poll", 30, "The interval in seconds to poll for background work.")
	serverCmd.PersistentFlags().Int("cluster-resource-threshold", 80, "The percent threshold where new installations won't be scheduled on a multi-tenant cluster.")
	serverCmd.PersistentFlags().Bool("use-existing-aws-resources", true, "Whether to use existing AWS resources (VPCs, subnets, etc.) or not.")
	serverCmd.PersistentFlags().Bool("keep-database-data", true, "Whether to preserve database data after installation deletion or not.")
	serverCmd.PersistentFlags().Bool("keep-filestore-data", true, "Whether to preserve filestore data after installation deletion or not.")
	serverCmd.PersistentFlags().Bool("debug", false, "Whether to output debug logs.")
	serverCmd.MarkPersistentFlagRequired("route53-id")
	serverCmd.MarkPersistentFlagRequired("private-route53-id")
	serverCmd.MarkPersistentFlagRequired("private-dns")
	serverCmd.MarkPersistentFlagRequired("certificate-aws-arn")
}

var serverCmd = &cobra.Command{
	Use:   "server",
	Short: "Run the provisioning server.",
	RunE: func(command *cobra.Command, args []string) error {
		command.SilenceUsage = true

		debug, _ := command.Flags().GetBool("debug")
		if debug {
			logger.SetLevel(logrus.DebugLevel)
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

		clusterResourceThreshold, _ := command.Flags().GetInt("cluster-resource-threshold")
		if clusterResourceThreshold < 10 || clusterResourceThreshold > 100 {
			return fmt.Errorf("cluster-resource-threshold (%d) must be set between 10 and 100", clusterResourceThreshold)
		}

		clusterSupervisor, _ := command.Flags().GetBool("cluster-supervisor")
		installationSupervisor, _ := command.Flags().GetBool("installation-supervisor")
		clusterInstallationSupervisor, _ := command.Flags().GetBool("cluster-installation-supervisor")
		if !clusterSupervisor && !installationSupervisor && !clusterInstallationSupervisor {
			logger.Warn("Server will be running with no supervisors. Only API functionality will work.")
		}

		s3StateStore, _ := command.Flags().GetString("state-store")
		certificateSslARN, _ := command.Flags().GetString("certificate-aws-arn")
		route53ZoneID, _ := command.Flags().GetString("route53-id")
		privateRoute53ZoneID, _ := command.Flags().GetString("private-route53-id")
		privateDNS, _ := command.Flags().GetString("private-dns")
		useExistingResources, _ := command.Flags().GetBool("use-existing-aws-resources")
		keepDatabaseData, _ := command.Flags().GetBool("keep-database-data")
		keepFilestoreData, _ := command.Flags().GetBool("keep-filestore-data")

		wd, err := os.Getwd()
		if err != nil {
			wd = "error getting working directory"
			logger.WithError(err).Error("Unable to get current working directory")
		}

		logger.WithFields(logrus.Fields{
			"cluster-supervisor":              clusterSupervisor,
			"installation-supervisor":         installationSupervisor,
			"cluster-installation-supervisor": clusterInstallationSupervisor,
			"store-version":                   currentVersion,
			"state-store":                     s3StateStore,
			"aws-arn":                         certificateSslARN,
			"working-directory":               wd,
			"route53-id":                      route53ZoneID,
			"private-route53-id":              privateRoute53ZoneID,
			"private-dns":                     privateDNS,
			"cluster-resource-threshold":      clusterResourceThreshold,
			"use-existing-aws-resources":      useExistingResources,
			"keep-database-data":              keepDatabaseData,
			"keep-filestore-data":             keepFilestoreData,
			"debug":                           debug,
		}).Info("Starting Mattermost Provisioning Server")

		// Setup the provisioner for actually effecting changes to clusters.
		kopsProvisioner := provisioner.NewKopsProvisioner(
			clusterRootDir,
			s3StateStore,
			certificateSslARN,
			privateDNS,
			useExistingResources,
			logger,
		)

		var multiDoer supervisor.MultiDoer
		if clusterSupervisor {
			multiDoer = append(multiDoer, supervisor.NewClusterSupervisor(sqlStore, kopsProvisioner, aws.New(privateRoute53ZoneID), instanceID, logger))
		}
		if installationSupervisor {
			multiDoer = append(multiDoer, supervisor.NewInstallationSupervisor(sqlStore, kopsProvisioner, aws.New(route53ZoneID), instanceID, clusterResourceThreshold, keepDatabaseData, keepFilestoreData, logger))
		}
		if clusterInstallationSupervisor {
			multiDoer = append(multiDoer, supervisor.NewClusterInstallationSupervisor(sqlStore, kopsProvisioner, aws.New(route53ZoneID), instanceID, logger))
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
		// We'll accept graceful shutdowns when quit via SIGINT (Ctrl+C)
		// SIGKILL, SIGQUIT or SIGTERM (Ctrl+/) will not be caught.
		signal.Notify(c, os.Interrupt)

		// Block until we receive our signal.
		<-c
		logger.Info("Shutting down")

		ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
		defer cancel()
		srv.Shutdown(ctx)

		return nil
	},
}
