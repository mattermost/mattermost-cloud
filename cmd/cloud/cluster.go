// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.
//

package main

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/olekukonko/tablewriter"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"

	"github.com/mattermost/mattermost-cloud/clusterdictionary"
	"github.com/mattermost/mattermost-cloud/model"
)

func init() {
	clusterCmd.PersistentFlags().String("server", defaultLocalServerAPI, "The provisioning server whose API will be queried.")
	clusterCmd.PersistentFlags().Bool("dry-run", false, "When set to true, only print the API request without sending it.")

	clusterCreateCmd.Flags().String("provider", "aws", "Cloud provider hosting the cluster.")
	clusterCreateCmd.Flags().String("version", "latest", "The Kubernetes version to target. Use 'latest' or versions such as '1.16.10'.")
	clusterCreateCmd.Flags().String("kops-ami", "", "The AMI to use for the cluster hosts. Leave empty for the default kops image.")
	clusterCreateCmd.Flags().String("size", "SizeAlef500", "The size constant describing the cluster")
	clusterCreateCmd.Flags().String("size-master-instance-type", "", "The instance type describing the k8s master nodes. Overwrites value from 'size'.")
	clusterCreateCmd.Flags().Int64("size-master-count", 0, "The number of k8s master nodes. Overwrites value from 'size'.")
	clusterCreateCmd.Flags().String("size-node-instance-type", "", "The instance type describing the k8s worker nodes. Overwrites value from 'size'.")
	clusterCreateCmd.Flags().Int64("size-node-count", 0, "The number of k8s worker nodes. Overwrites value from 'size'.")
	clusterCreateCmd.Flags().String("zones", "us-east-1a", "The zones where the cluster will be deployed. Use commas to separate multiple zones.")
	clusterCreateCmd.Flags().Bool("allow-installations", true, "Whether the cluster will allow for new installations to be scheduled.")
	clusterCreateCmd.Flags().String("prometheus-version", model.PrometheusDefaultVersion, "The version of Prometheus to provision. Use 'stable' to provision the latest stable version published upstream.")
	clusterCreateCmd.Flags().String("prometheus-operator-version", model.PrometheusOperatorDefaultVersion, "The version of Prometheus Operator to provision. Use 'stable' to provision the latest stable version published upstream.")
	clusterCreateCmd.Flags().String("fluentbit-version", model.FluentbitDefaultVersion, "The version of Fluentbit to provision. Use 'stable' to provision the latest stable version published upstream.")
	clusterCreateCmd.Flags().String("nginx-version", model.NginxDefaultVersion, "The version of Nginx to provision. Use 'stable' to provision the latest stable version published upstream.")
	clusterCreateCmd.Flags().String("teleport-version", model.TeleportDefaultVersion, "The version of Teleport to provision. Use 'stable' to provision the latest stable version published upstream.")

	clusterProvisionCmd.Flags().String("cluster", "", "The id of the cluster to be provisioned.")
	clusterProvisionCmd.Flags().String("prometheus-version", "", "The version of Prometheus to provision, no change if omitted. Use \"stable\" as an argument to this command to indicate that you wish to remove the pinned version and return the utility to tracking the latest version.")
	clusterProvisionCmd.Flags().String("prometheus-operator-version", "", "The version of Prometheus Operator to provision, no change if omitted. Use \"stable\" as an argument to this command to indicate that you wish to remove the pinned version and return the utility to tracking the latest version.")
	clusterProvisionCmd.Flags().String("fluentbit-version", "", "The version of Fluentbit to provision, no change if omitted. Use \"stable\" as an argument to this command to indicate that you wish to remove the pinned version and return the utility to tracking the latest version.")
	clusterProvisionCmd.Flags().String("nginx-version", "", "The version of Nginx to provision, no change if omitted. Use \"stable\" as an argument to this command to indicate that you wish to remove the pinned version and return the utility to tracking the latest version.")
	clusterProvisionCmd.Flags().String("teleport-version", "", "The version of Teleport to provision, no change if omitted. Use \"stable\" as an argument to this command to indicate that you wish to remove the pinned version and return the utility to tracking the latest version.")
	clusterProvisionCmd.MarkFlagRequired("cluster")

	clusterUpdateCmd.Flags().String("cluster", "", "The id of the cluster to be updated.")
	clusterUpdateCmd.Flags().Bool("allow-installations", true, "Whether the cluster will allow for new installations to be scheduled.")
	clusterUpdateCmd.MarkFlagRequired("cluster")

	clusterUpgradeCmd.Flags().String("cluster", "", "The id of the cluster to be upgraded.")
	clusterUpgradeCmd.Flags().String("version", "", "The Kubernetes version to target. Use 'latest' or versions such as '1.16.10'.")
	clusterUpgradeCmd.Flags().String("kops-ami", "", "The AMI to use for the cluster hosts. Use 'latest' for the default kops image.")
	clusterUpgradeCmd.MarkFlagRequired("cluster")

	clusterResizeCmd.Flags().String("cluster", "", "The id of the cluster to be resized.")
	clusterResizeCmd.Flags().String("size", "", "The size constant describing the cluster")
	clusterResizeCmd.Flags().String("size-node-instance-type", "", "The instance type describing the k8s worker nodes. Overwrites value from 'size'.")
	clusterResizeCmd.Flags().Int64("size-node-min-count", 0, "The minimum number of k8s worker nodes. Overwrites value from 'size'.")
	clusterResizeCmd.Flags().Int64("size-node-max-count", 0, "The maximum number of k8s worker nodes. Overwrites value from 'size'.")
	clusterResizeCmd.MarkFlagRequired("cluster")

	clusterDeleteCmd.Flags().String("cluster", "", "The id of the cluster to be deleted.")
	clusterDeleteCmd.MarkFlagRequired("cluster")

	clusterGetCmd.Flags().String("cluster", "", "The id of the cluster to be fetched.")
	clusterGetCmd.MarkFlagRequired("cluster")

	clusterListCmd.Flags().Int("page", 0, "The page of clusters to fetch, starting at 0.")
	clusterListCmd.Flags().Int("per-page", 100, "The number of clusters to fetch per page.")
	clusterListCmd.Flags().Bool("include-deleted", false, "Whether to include deleted clusters.")
	clusterListCmd.Flags().Bool("table", false, "Whether to display the returned cluster list in a table or not")

	clusterUtilitiesCmd.Flags().String("cluster", "", "The id of the cluster whose utilities are to be fetched.")
	clusterUtilitiesCmd.MarkFlagRequired("cluster")

	clusterCmd.AddCommand(clusterCreateCmd)
	clusterCmd.AddCommand(clusterProvisionCmd)
	clusterCmd.AddCommand(clusterUpdateCmd)
	clusterCmd.AddCommand(clusterUpgradeCmd)
	clusterCmd.AddCommand(clusterResizeCmd)
	clusterCmd.AddCommand(clusterDeleteCmd)
	clusterCmd.AddCommand(clusterGetCmd)
	clusterCmd.AddCommand(clusterListCmd)
	clusterCmd.AddCommand(clusterInstallationCmd)
	clusterCmd.AddCommand(clusterShowStateReport)
	clusterCmd.AddCommand(clusterUtilitiesCmd)
	clusterCmd.AddCommand(clusterShowSizeDictionary)
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
		version, _ := command.Flags().GetString("version")
		kopsAMI, _ := command.Flags().GetString("kops-ami")
		zones, _ := command.Flags().GetString("zones")
		allowInstallations, _ := command.Flags().GetBool("allow-installations")

		request := &model.CreateClusterRequest{
			Provider:               provider,
			Version:                version,
			KopsAMI:                kopsAMI,
			Zones:                  strings.Split(zones, ","),
			AllowInstallations:     allowInstallations,
			DesiredUtilityVersions: processUtilityFlags(command),
		}

		size, _ := command.Flags().GetString("size")
		err := clusterdictionary.ApplyToCreateClusterRequest(size, request)
		if err != nil {
			return errors.Wrap(err, "failed to apply size values")
		}
		masterInstanceType, _ := command.Flags().GetString("size-master-instance-type")
		if len(masterInstanceType) != 0 {
			request.MasterInstanceType = masterInstanceType
		}
		masterCount, _ := command.Flags().GetInt64("size-master-count")
		if masterCount != 0 {
			request.MasterCount = masterCount
		}
		nodeInstanceType, _ := command.Flags().GetString("size-node-instance-type")
		if len(nodeInstanceType) != 0 {
			request.NodeInstanceType = nodeInstanceType
		}
		nodeCount, _ := command.Flags().GetInt64("size-node-count")
		if nodeCount != 0 {
			// Setting different min and max counts in currently not supported
			// with the kops create cluster flag.
			request.NodeMinCount = nodeCount
			request.NodeMaxCount = nodeCount
		}

		dryRun, _ := command.Flags().GetBool("dry-run")
		if dryRun {
			err = printJSON(request)
			if err != nil {
				return errors.Wrap(err, "failed to print API request")
			}

			return nil
		}

		cluster, err := client.CreateCluster(request)
		if err != nil {
			return errors.Wrap(err, "failed to create cluster")
		}

		err = printJSON(cluster)
		if err != nil {
			return errors.Wrap(err, "failed to print cluster response")
		}

		return nil
	},
}

var clusterProvisionCmd = &cobra.Command{
	Use:   "provision",
	Short: "Provision/Reprovision a cluster's k8s resources.",
	RunE: func(command *cobra.Command, args []string) error {
		command.SilenceUsage = true

		serverAddress, _ := command.Flags().GetString("server")
		client := model.NewClient(serverAddress)
		clusterID, _ := command.Flags().GetString("cluster")

		var request *model.ProvisionClusterRequest = nil
		if desiredUtilityVersions := processUtilityFlags(command); len(desiredUtilityVersions) > 0 {
			request = &model.ProvisionClusterRequest{
				DesiredUtilityVersions: desiredUtilityVersions,
			}
		}

		dryRun, _ := command.Flags().GetBool("dry-run")
		if dryRun {
			err := printJSON(request)
			if err != nil {
				return errors.Wrap(err, "failed to print API request")
			}

			return nil
		}

		cluster, err := client.ProvisionCluster(clusterID, request)
		if err != nil {
			return errors.Wrap(err, "failed to provision cluster")
		}

		err = printJSON(cluster)
		if err != nil {
			return errors.Wrap(err, "failed to print cluster response")
		}

		return nil
	},
}

var clusterUpdateCmd = &cobra.Command{
	Use:   "update",
	Short: "Updates a cluster's configuration.",
	RunE: func(command *cobra.Command, args []string) error {
		command.SilenceUsage = true

		serverAddress, _ := command.Flags().GetString("server")
		client := model.NewClient(serverAddress)

		clusterID, _ := command.Flags().GetString("cluster")
		allowInstallations, _ := command.Flags().GetBool("allow-installations")

		request := &model.UpdateClusterRequest{
			AllowInstallations: allowInstallations,
		}

		dryRun, _ := command.Flags().GetBool("dry-run")
		if dryRun {
			err := printJSON(request)
			if err != nil {
				return errors.Wrap(err, "failed to print API request")
			}

			return nil
		}

		cluster, err := client.UpdateCluster(clusterID, request)
		if err != nil {
			return errors.Wrap(err, "failed to update cluster")
		}

		err = printJSON(cluster)
		if err != nil {
			return errors.Wrap(err, "failed to print cluster response")
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

		request := &model.PatchUpgradeClusterRequest{
			Version: getStringFlagPointer(command, "version"),
			KopsAMI: getStringFlagPointer(command, "kops-ami"),
		}

		dryRun, _ := command.Flags().GetBool("dry-run")
		if dryRun {
			err := printJSON(request)
			if err != nil {
				return errors.Wrap(err, "failed to print API request")
			}

			return nil
		}

		cluster, err := client.UpgradeCluster(clusterID, request)
		if err != nil {
			return errors.Wrap(err, "failed to upgrade cluster")
		}

		err = printJSON(cluster)
		if err != nil {
			return errors.Wrap(err, "failed to print cluster response")
		}

		return nil
	},
}

var clusterResizeCmd = &cobra.Command{
	Use:   "resize",
	Short: "Resize a k8s cluster",
	RunE: func(command *cobra.Command, args []string) error {
		command.SilenceUsage = true

		serverAddress, _ := command.Flags().GetString("server")
		client := model.NewClient(serverAddress)

		clusterID, _ := command.Flags().GetString("cluster")

		// Apply values from 'size' constant and then apply overrides.
		request := &model.PatchClusterSizeRequest{}
		size, _ := command.Flags().GetString("size")
		err := clusterdictionary.ApplyToPatchClusterSizeRequest(size, request)
		if err != nil {
			return errors.Wrap(err, "failed to apply size values")
		}
		nodeInstanceType, _ := command.Flags().GetString("size-node-instance-type")
		if len(nodeInstanceType) != 0 {
			request.NodeInstanceType = &nodeInstanceType
		}
		nodeMinCount, _ := command.Flags().GetInt64("size-node-min-count")
		if nodeMinCount != 0 {
			request.NodeMinCount = &nodeMinCount
		}
		nodeMaxCount, _ := command.Flags().GetInt64("size-node-max-count")
		if nodeMaxCount != 0 {
			request.NodeMaxCount = &nodeMaxCount
		}

		dryRun, _ := command.Flags().GetBool("dry-run")
		if dryRun {
			err = printJSON(request)
			if err != nil {
				return errors.Wrap(err, "failed to print API request")
			}

			return nil
		}

		cluster, err := client.ResizeCluster(clusterID, request)
		if err != nil {
			return errors.Wrap(err, "failed to resize cluster")
		}

		err = printJSON(cluster)
		if err != nil {
			return errors.Wrap(err, "failed to print cluster response")
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
			return errors.Wrap(err, "failed to print cluster response")
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

		outputToTable, _ := command.Flags().GetBool("table")
		if outputToTable {
			table := tablewriter.NewWriter(os.Stdout)
			table.SetAlignment(tablewriter.ALIGN_LEFT)
			table.SetHeader([]string{"ID", "STATE", "VERSION", "MASTER NODES", "WORKER NODES"})

			for _, cluster := range clusters {
				table.Append([]string{
					cluster.ID,
					cluster.State,
					cluster.ProvisionerMetadataKops.Version,
					fmt.Sprintf("%d x %s", cluster.ProvisionerMetadataKops.MasterCount, cluster.ProvisionerMetadataKops.MasterInstanceType),
					fmt.Sprintf("%d x %s (max %d)", cluster.ProvisionerMetadataKops.NodeMinCount, cluster.ProvisionerMetadataKops.NodeInstanceType, cluster.ProvisionerMetadataKops.NodeMaxCount),
				})
			}
			table.Render()

			return nil
		}

		err = printJSON(clusters)
		if err != nil {
			return errors.Wrap(err, "failed to print cluster response")
		}

		return nil
	},
}

var clusterUtilitiesCmd = &cobra.Command{
	Use:   "utilities",
	Short: "Show metadata regarding utility services running in a cluster.",
	RunE: func(command *cobra.Command, args []string) error {
		command.SilenceUsage = true

		serverAddress, _ := command.Flags().GetString("server")
		client := model.NewClient(serverAddress)
		clusterID, err := command.Flags().GetString("cluster")
		if err != nil {
			return err
		}

		metadata, err := client.GetClusterUtilities(clusterID)
		if err != nil {
			return err
		}

		err = printJSON(metadata)
		if err != nil {
			return err
		}

		return nil
	},
}

var clusterShowSizeDictionary = &cobra.Command{
	Use:   "dictionary",
	Short: "Shows predefined cluster size templates.",
	RunE: func(command *cobra.Command, args []string) error {
		command.SilenceUsage = true

		err := printJSON(clusterdictionary.ValidSizes)
		if err != nil {
			return errors.Wrap(err, "failed to print cluster dictionary")
		}

		return nil
	},
}

// TODO:
// Instead of showing the state data from the model of the CLI binary, add a new
// API endpoint to return the server's state model.
var clusterShowStateReport = &cobra.Command{
	Use:   "state-report",
	Short: "Shows information regarding changing cluster state.",
	RunE: func(command *cobra.Command, args []string) error {
		command.SilenceUsage = true

		err := printJSON(model.GetClusterRequestStateReport())
		if err != nil {
			return err
		}

		return nil
	},
}

func processUtilityFlags(command *cobra.Command) map[string]string {
	prometheusVersion, _ := command.Flags().GetString("prometheus-version")
	prometheusOperatorVersion, _ := command.Flags().GetString("prometheus-operator-version")
	fluentbitVersion, _ := command.Flags().GetString("fluentbit-version")
	nginxVersion, _ := command.Flags().GetString("nginx-version")
	teleportVersion, _ := command.Flags().GetString("teleport-version")

	utilityVersions := make(map[string]string)

	if prometheusVersion != "" {
		utilityVersions[model.PrometheusCanonicalName] = prometheusVersion
	}

	if prometheusOperatorVersion != "" {
		utilityVersions[model.PrometheusOperatorCanonicalName] = prometheusOperatorVersion
	}

	if fluentbitVersion != "" {
		utilityVersions[model.FluentbitCanonicalName] = fluentbitVersion
	}

	if nginxVersion != "" {
		utilityVersions[model.NginxCanonicalName] = nginxVersion
	}

	if teleportVersion != "" {
		utilityVersions[model.TeleportCanonicalName] = teleportVersion
	}

	return utilityVersions
}
