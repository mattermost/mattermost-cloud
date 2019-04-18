package main

import (
	"encoding/json"
	"os"
	"strings"

	"github.com/pkg/errors"
	"github.com/spf13/cobra"

	"github.com/mattermost/mattermost-cloud/internal/provisioner"
	"github.com/mattermost/mattermost-cloud/internal/store"
)

func init() {
	clusterCmd.PersistentFlags().String("state-store", "dev.cloud.mattermost.com", "The S3 bucket used to store cluster state.")

	clusterCreateCmd.Flags().String("provider", "aws", "Cloud provider hosting the cluster.")
	clusterCreateCmd.Flags().String("size", "SizeAlef500", "The size constant describing the cluster.")
	clusterCreateCmd.Flags().String("zones", "us-east-1a", "The zones where the cluster will be deployed. Use commas to separate multiple zones.")
	clusterCreateCmd.MarkFlagRequired("size")

	clusterUpgradeCmd.Flags().String("cluster", "", "The id of the cluster to be upgraded.")
	clusterUpgradeCmd.MarkFlagRequired("cluster")

	clusterDeleteCmd.Flags().String("cluster", "", "The id of the cluster to be deleted.")
	clusterDeleteCmd.MarkFlagRequired("cluster")

	clusterGetCmd.Flags().String("cluster", "", "The id of the cluster to be fetched.")
	clusterGetCmd.MarkFlagRequired("cluster")

	clusterListCmd.Flags().Int("page", 0, "The page of clusters to fetch, starting at 0.")
	clusterListCmd.Flags().Int("per-page", 100, "The number of clusters to fetch per page.")
	clusterListCmd.Flags().Bool("include-deleted", false, "Whether to include deleted clusters.")

	clusterCmd.AddCommand(clusterCreateCmd)
	clusterCmd.AddCommand(clusterUpgradeCmd)
	clusterCmd.AddCommand(clusterDeleteCmd)
	clusterCmd.AddCommand(clusterGetCmd)
	clusterCmd.AddCommand(clusterListCmd)
}

var clusterCmd = &cobra.Command{
	Use:   "cluster",
	Short: "Manipulate clusters managed by the provisioning server.",
}

var clusterCreateCmd = &cobra.Command{
	Use:   "create",
	Short: "Create a cluster.",
	RunE: func(command *cobra.Command, args []string) error {
		command.SilenceUsage = true

		sqlStore, err := sqlStore(command)
		if err != nil {
			return err
		}

		provider, _ := command.Flags().GetString("provider")
		s3StateStore, _ := command.Flags().GetString("state-store")
		size, _ := command.Flags().GetString("size")
		zones, _ := command.Flags().GetString("zones")

		splitZones := strings.Split(zones, ",")

		return provisioner.CreateCluster(sqlStore, provider, s3StateStore, size, splitZones, logger)
	},
}

var clusterUpgradeCmd = &cobra.Command{
	Use:   "upgrade",
	Short: "Upgrade k8s on a cluster.",
	RunE: func(command *cobra.Command, args []string) error {
		command.SilenceUsage = true

		sqlStore, err := sqlStore(command)
		if err != nil {
			return err
		}

		clusterID, _ := command.Flags().GetString("cluster")
		s3StateStore, _ := command.Flags().GetString("state-store")

		return provisioner.UpgradeCluster(sqlStore, clusterID, s3StateStore, logger)
	},
}

var clusterDeleteCmd = &cobra.Command{
	Use:   "delete",
	Short: "Delete a cluster.",
	RunE: func(command *cobra.Command, args []string) error {
		command.SilenceUsage = true

		sqlStore, err := sqlStore(command)
		if err != nil {
			return err
		}

		clusterID, _ := command.Flags().GetString("cluster")
		s3StateStore, _ := command.Flags().GetString("state-store")

		return provisioner.DeleteCluster(sqlStore, clusterID, s3StateStore, logger)
	},
}

var clusterGetCmd = &cobra.Command{
	Use:   "get",
	Short: "Get a particular cluster.",
	RunE: func(command *cobra.Command, args []string) error {
		command.SilenceUsage = true

		sqlStore, err := sqlStore(command)
		if err != nil {
			return err
		}

		clusterID, _ := command.Flags().GetString("cluster")

		cluster, err := sqlStore.GetCluster(clusterID)
		if err != nil {
			return errors.Wrap(err, "failed to query cluster")
		}
		if cluster == nil {
			return nil
		}

		encoder := json.NewEncoder(os.Stdout)
		encoder.SetIndent("", "    ")
		err = encoder.Encode(cluster)
		if err != nil {
			return err
		}

		return nil
	},
}

var clusterListCmd = &cobra.Command{
	Use:   "list",
	Short: "List created clusters.",
	RunE: func(command *cobra.Command, args []string) error {
		command.SilenceUsage = true

		sqlStore, err := sqlStore(command)
		if err != nil {
			return err
		}

		page, _ := command.Flags().GetInt("page")
		perPage, _ := command.Flags().GetInt("per-page")
		includeDeleted, _ := command.Flags().GetBool("include-deleted")

		clusters, err := sqlStore.GetClusters(page, perPage, includeDeleted)
		if err != nil {
			return errors.Wrap(err, "failed to query clusters")
		}

		if clusters == nil {
			clusters = []*store.Cluster{}
		}

		results := struct {
			Clusters []*store.Cluster
			Page     int
			PerPage  int
		}{
			clusters,
			page,
			perPage,
		}

		encoder := json.NewEncoder(os.Stdout)
		encoder.SetIndent("", "    ")
		err = encoder.Encode(results)
		if err != nil {
			return err
		}

		return nil
	},
}
