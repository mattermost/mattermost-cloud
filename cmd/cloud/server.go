package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"time"

	"github.com/gorilla/mux"
	"github.com/mattermost/mattermost-cloud/internal/api"
	"github.com/mattermost/mattermost-cloud/internal/model"
	"github.com/mattermost/mattermost-cloud/internal/provisioner"
	"github.com/mattermost/mattermost-cloud/internal/store"
	"github.com/mattermost/mattermost-cloud/internal/supervisor"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"golang.org/x/sync/semaphore"
)

// clusterRootDir is the local directory that contains cluster configuration.
const clusterRootDir = "clusters"

var instanceID string

func init() {
	instanceID = model.NewID()

	serverCmd.PersistentFlags().String("database", "sqlite://cloud.db", "The database backing the provisioning server.")
	serverCmd.PersistentFlags().String("listen", ":8075", "The interface and port on which to listen.")
	serverCmd.PersistentFlags().String("state-store", "dev.cloud.mattermost.com", "The S3 bucket used to store cluster state.")
	serverCmd.PersistentFlags().Int("jobs", 1, "The maximum number of background jobs to allow.")
	serverCmd.PersistentFlags().Int("poll", 30, "The interval in seconds to poll for background work.")
}

var serverCmd = &cobra.Command{
	Use:   "server",
	Short: "Run the provisioning server.",
	RunE: func(command *cobra.Command, args []string) error {
		command.SilenceUsage = true

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

		s3StateStore, _ := command.Flags().GetString("state-store")
		logger.Infof("Using state store %s", s3StateStore)

		// Limit the number of concurrent background jobs within this instance.
		jobs, _ := command.Flags().GetInt("jobs")
		workers := semaphore.NewWeighted(int64(jobs))
		defer func() {
			if err := workers.Acquire(context.Background(), int64(jobs)); err != nil {
				logger.WithError(err).Error("failed to shut down worker pool")
			}
		}()

		// Setup the provisioner for actually effecting changes to clusters.
		kopsProvisioner := provisioner.NewKopsProvisioner(
			clusterRootDir,
			s3StateStore,
			logger,
		)

		// Setup the supervisor to effect any requested changes. It is wrapped in a
		// scheduler to trigger it periodically in addition to being poked by the API
		// layer.
		poll, _ := command.Flags().GetInt("poll")
		if jobs == 0 || poll == 0 {
			logger.WithFields(map[string]interface{}{
				"poll": poll,
				"jobs": jobs,
			}).Info("Scheduler is disabled")
		}
		supervisor := supervisor.NewScheduler(
			supervisor.MultiDoer{
				supervisor.NewClusterSupervisor(sqlStore, kopsProvisioner, workers, logger),
			},
			time.Duration(poll)*time.Second,
		)
		defer supervisor.Close()

		router := mux.NewRouter()

		api.Register(router, &api.Context{
			Store:      sqlStore,
			Supervisor: supervisor,
			Logger:     logger,
		})

		listen, _ := command.Flags().GetString("listen")
		srv := &http.Server{
			Addr:           listen,
			Handler:        router,
			ReadTimeout:    10 * time.Second,
			WriteTimeout:   10 * time.Second,
			IdleTimeout:    time.Second * 60,
			MaxHeaderBytes: 1 << 20,
			ErrorLog:       log.New(&logrusWriter{logger}, "", 0),
		}

		go func() {
			logger.WithField("addr", srv.Addr).Info("Listening")
			err := srv.ListenAndServe()
			if err != nil && err != http.ErrServerClosed {
				logger.WithField("err", err).Error("Failed to listen and serve")
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
