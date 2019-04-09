package main

import (
	"github.com/spf13/cobra"

	"github.com/mattermost/mattermost-cloud/internal/provisioner"
)

func init() {
	clusterCreateCmd.Flags().String("provider", "aws", "Cloud provider hosting the cluster.")
	clusterCreateCmd.Flags().String("size", "", "The size constant describing the cluster.")
	clusterCreateCmd.MarkFlagRequired("size")

	clusterDeleteCmd.Flags().String("cluster", "", "The id of the cluster to be deleted.")
	clusterDeleteCmd.MarkFlagRequired("cluster")

	clusterCmd.AddCommand(clusterCreateCmd)
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
		size, _ := command.Flags().GetString("size")

		command.SilenceUsage = true

		return provisioner.CreateCluster(provider, size, logger)
	},
}

var clusterDeleteCmd = &cobra.Command{
	Use:   "delete",
	Short: "Delete a cluster",
	RunE: func(command *cobra.Command, args []string) error {
		clusterId, _ := command.Flags().GetString("cluster")

		command.SilenceUsage = true

		return provisioner.DeleteCluster(clusterId, logger)
	},
}
