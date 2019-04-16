package main

import (
	"strings"

	"github.com/spf13/cobra"

	"github.com/mattermost/mattermost-cloud/internal/provisioner"
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

	clusterCmd.AddCommand(clusterCreateCmd)
	clusterCmd.AddCommand(clusterUpgradeCmd)
	clusterCmd.AddCommand(clusterDeleteCmd)
}

var clusterCmd = &cobra.Command{
	Use:   "cluster",
	Short: "Manipulate clusters managed by the provisioning server.",
}

var clusterCreateCmd = &cobra.Command{
	Use:   "create",
	Short: "Create a cluster.",
	RunE: func(command *cobra.Command, args []string) error {
		provider, _ := command.Flags().GetString("provider")
		s3StateStore, _ := command.Flags().GetString("state-store")
		size, _ := command.Flags().GetString("size")
		zones, _ := command.Flags().GetString("zones")

		splitZones := strings.Split(zones, ",")

		command.SilenceUsage = true

		return provisioner.CreateCluster(provider, s3StateStore, size, splitZones, logger)
	},
}

var clusterUpgradeCmd = &cobra.Command{
	Use:   "upgrade",
	Short: "Upgrade a cluster.",
	RunE: func(command *cobra.Command, args []string) error {
		clusterId, _ := command.Flags().GetString("cluster")
		s3StateStore, _ := command.Flags().GetString("state-store")

		command.SilenceUsage = true

		return provisioner.UpgradeCluster(clusterId, s3StateStore, logger)
	},
}

var clusterDeleteCmd = &cobra.Command{
	Use:   "delete",
	Short: "Delete a cluster.",
	RunE: func(command *cobra.Command, args []string) error {
		clusterId, _ := command.Flags().GetString("cluster")
		s3StateStore, _ := command.Flags().GetString("state-store")

		command.SilenceUsage = true

		return provisioner.DeleteCluster(clusterId, s3StateStore, logger)
	},
}
