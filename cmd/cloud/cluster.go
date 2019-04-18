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
	clusterCreateCmd.Flags().Bool("cluster-ready", true, "Controls if the command should wait for k8s to become fully ready before exiting")
	clusterCreateCmd.MarkFlagRequired("size")

	clusterUpgradeCmd.Flags().String("cluster", "", "The id of the cluster to be upgraded.")
	clusterUpgradeCmd.Flags().Bool("cluster-ready", true, "Controls if the command should wait for k8s to become fully ready before exiting")
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
		waitForReady, _ := command.Flags().GetBool("cluster-ready")

		splitZones := strings.Split(zones, ",")

		command.SilenceUsage = true

		return provisioner.CreateCluster(provider, s3StateStore, size, splitZones, waitForReady, logger)
	},
}

var clusterUpgradeCmd = &cobra.Command{
	Use:   "upgrade",
	Short: "Upgrade k8s on a cluster.",
	RunE: func(command *cobra.Command, args []string) error {
		clusterID, _ := command.Flags().GetString("cluster")
		s3StateStore, _ := command.Flags().GetString("state-store")
		waitForReady, _ := command.Flags().GetBool("cluster-ready")

		command.SilenceUsage = true

		return provisioner.UpgradeCluster(clusterID, s3StateStore, waitForReady, logger)
	},
}

var clusterDeleteCmd = &cobra.Command{
	Use:   "delete",
	Short: "Delete a cluster.",
	RunE: func(command *cobra.Command, args []string) error {
		clusterID, _ := command.Flags().GetString("cluster")
		s3StateStore, _ := command.Flags().GetString("state-store")

		command.SilenceUsage = true

		return provisioner.DeleteCluster(clusterID, s3StateStore, logger)
	},
}
