package main

import (
	"encoding/json"
	"os"
	"strings"

	"github.com/pkg/errors"
	"github.com/spf13/cobra"

	"github.com/mattermost/mattermost-cloud/model"
)

func init() {
	clusterCmd.PersistentFlags().String("server", "http://localhost:8075", "The provisioning server whose API will be queried.")

	clusterCreateCmd.Flags().String("provider", "aws", "Cloud provider hosting the cluster.")
	clusterCreateCmd.Flags().String("size", "SizeAlef500", "The size constant describing the cluster.")
	clusterCreateCmd.Flags().String("zones", "us-east-1a", "The zones where the cluster will be deployed. Use commas to separate multiple zones.")
	clusterCreateCmd.Flags().Int("wait", 600, "The amount of seconds to wait for k8s to become fully ready before exiting. Set to 0 to exit immediately.")

	clusterProvisionCmd.Flags().String("cluster", "", "The id of the cluster to be provisioned.")
	clusterProvisionCmd.MarkFlagRequired("cluster")

	clusterUpgradeCmd.Flags().String("cluster", "", "The id of the cluster to be upgraded.")
	clusterUpgradeCmd.Flags().String("version", "latest", "The Kubernetes version to target.")
	clusterUpgradeCmd.Flags().Int("wait", 600, "The amount of seconds to wait for k8s to become fully ready before exiting. Set to 0 to exit immediately.")
	clusterUpgradeCmd.MarkFlagRequired("cluster")

	clusterDeleteCmd.Flags().String("cluster", "", "The id of the cluster to be deleted.")
	clusterDeleteCmd.MarkFlagRequired("cluster")

	clusterGetCmd.Flags().String("cluster", "", "The id of the cluster to be fetched.")
	clusterGetCmd.MarkFlagRequired("cluster")

	clusterListCmd.Flags().Int("page", 0, "The page of clusters to fetch, starting at 0.")
	clusterListCmd.Flags().Int("per-page", 100, "The number of clusters to fetch per page.")
	clusterListCmd.Flags().Bool("include-deleted", false, "Whether to include deleted clusters.")

	clusterCmd.AddCommand(clusterCreateCmd)
	clusterCmd.AddCommand(clusterProvisionCmd)
	clusterCmd.AddCommand(clusterUpgradeCmd)
	clusterCmd.AddCommand(clusterDeleteCmd)
	clusterCmd.AddCommand(clusterGetCmd)
	clusterCmd.AddCommand(clusterListCmd)
	clusterCmd.AddCommand(clusterInstallationCmd)
}

var clusterCmd = &cobra.Command{
	Use:   "cluster",
	Short: "Manipulate clusters managed by the provisioning server.",
}

func printJSON(data interface{}) error {
	encoder := json.NewEncoder(os.Stdout)
	encoder.SetIndent("", "    ")
	return encoder.Encode(data)
}

var clusterCreateCmd = &cobra.Command{
	Use:   "create",
	Short: "Create a cluster.",
	RunE: func(command *cobra.Command, args []string) error {
		command.SilenceUsage = true

		serverAddress, _ := command.Flags().GetString("server")
		client := model.NewClient(serverAddress)

		provider, _ := command.Flags().GetString("provider")
		size, _ := command.Flags().GetString("size")
		zones, _ := command.Flags().GetString("zones")

		cluster, err := client.CreateCluster(&model.CreateClusterRequest{
			Provider: provider,
			Size:     size,
			Zones:    strings.Split(zones, ","),
		})
		if err != nil {
			return errors.Wrap(err, "failed to create cluster")
		}

		// TODO --wait for the cluster to be ready.
		// wait, _ := command.Flags().GetInt("wait")

		err = printJSON(cluster)
		if err != nil {
			return err
		}

		return nil
	},
}

var clusterProvisionCmd = &cobra.Command{
	Use:   "provision",
	Short: "Provision/Reprovision a cluster's k8s operators.",
	RunE: func(command *cobra.Command, args []string) error {
		command.SilenceUsage = true

		serverAddress, _ := command.Flags().GetString("server")
		client := model.NewClient(serverAddress)

		clusterID, _ := command.Flags().GetString("cluster")
		err := client.ProvisionCluster(clusterID)
		if err != nil {
			return errors.Wrap(err, "failed to provision cluster")
		}

		return nil
	},
}

var clusterUpgradeCmd = &cobra.Command{
	Use:   "upgrade",
	Short: "Upgrade k8s on a cluster.",
	RunE: func(command *cobra.Command, args []string) error {
		command.SilenceUsage = true

		serverAddress, _ := command.Flags().GetString("server")
		client := model.NewClient(serverAddress)

		clusterID, _ := command.Flags().GetString("cluster")
		version, _ := command.Flags().GetString("version")

		err := client.UpgradeCluster(clusterID, version)
		if err != nil {
			return errors.Wrap(err, "failed to upgrade cluster")
		}

		return nil
	},
}

var clusterDeleteCmd = &cobra.Command{
	Use:   "delete",
	Short: "Delete a cluster.",
	RunE: func(command *cobra.Command, args []string) error {
		command.SilenceUsage = true

		serverAddress, _ := command.Flags().GetString("server")
		client := model.NewClient(serverAddress)

		clusterID, _ := command.Flags().GetString("cluster")

		err := client.DeleteCluster(clusterID)
		if err != nil {
			return errors.Wrap(err, "failed to delete cluster")
		}

		return nil
	},
}

var clusterGetCmd = &cobra.Command{
	Use:   "get",
	Short: "Get a particular cluster.",
	RunE: func(command *cobra.Command, args []string) error {
		command.SilenceUsage = true

		serverAddress, _ := command.Flags().GetString("server")
		client := model.NewClient(serverAddress)

		clusterID, _ := command.Flags().GetString("cluster")
		cluster, err := client.GetCluster(clusterID)
		if err != nil {
			return errors.Wrap(err, "failed to query cluster")
		}
		if cluster == nil {
			return nil
		}

		err = printJSON(cluster)
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

		serverAddress, _ := command.Flags().GetString("server")
		client := model.NewClient(serverAddress)

		page, _ := command.Flags().GetInt("page")
		perPage, _ := command.Flags().GetInt("per-page")
		includeDeleted, _ := command.Flags().GetBool("include-deleted")
		clusters, err := client.GetClusters(&model.GetClustersRequest{
			Page:           page,
			PerPage:        perPage,
			IncludeDeleted: includeDeleted,
		})
		if err != nil {
			return errors.Wrap(err, "failed to query clusters")
		}

		err = printJSON(clusters)
		if err != nil {
			return err
		}

		return nil
	},
}
