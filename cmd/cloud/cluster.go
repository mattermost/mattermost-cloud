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
	clusterCreateCmd.Flags().String("prometheus-operator-version", model.PrometheusOperatorDefaultVersion.Version(), "The version of Prometheus Operator to provision. Use 'stable' to provision the latest stable version published upstream.")
	clusterCreateCmd.Flags().String("thanos-version", model.ThanosDefaultVersion.Version(), "The version of Thanos to provision. Use 'stable' to provision the latest stable version published upstream.")
	clusterCreateCmd.Flags().String("fluentbit-version", model.FluentbitDefaultVersion.Version(), "The version of Fluentbit to provision. Use 'stable' to provision the latest stable version published upstream.")
	clusterCreateCmd.Flags().String("nginx-version", model.NginxDefaultVersion.Version(), "The version of Nginx Internal to provision. Use 'stable' to provision the latest stable version published upstream.")
	clusterCreateCmd.Flags().String("nginx-internal-version", model.NginxInternalDefaultVersion.Version(), "The version of Nginx to provision. Use 'stable' to provision the latest stable version published upstream.")
	clusterCreateCmd.Flags().String("teleport-version", model.TeleportDefaultVersion.Version(), "The version of Teleport to provision. Use 'stable' to provision the latest stable version published upstream.")
	clusterCreateCmd.Flags().String("pgbouncer-version", model.PgbouncerDefaultVersion.Version(), "The version of Pgbouncer to provision. Use 'stable' to provision the latest stable version published upstream.")
	clusterCreateCmd.Flags().String("prometheus-operator-values", model.PrometheusOperatorDefaultVersion.Values(), "The branch name of the desired chart value file's version for Prometheus Operator")
	clusterCreateCmd.Flags().String("thanos-values", model.ThanosDefaultVersion.Values(), "The branch name of the desired chart value file's version for Thanos")
	clusterCreateCmd.Flags().String("fluentbit-values", model.FluentbitDefaultVersion.Values(), "The branch name of the desired chart value file's version for Fluent-Bit")
	clusterCreateCmd.Flags().String("nginx-values", model.NginxDefaultVersion.Values(), "The branch name of the desired chart value file's version for NGINX")
	clusterCreateCmd.Flags().String("nginx-internal-values", model.NginxInternalDefaultVersion.Values(), "The branch name of the desired chart value file's version for NGINX Internal")
	clusterCreateCmd.Flags().String("teleport-values", model.TeleportDefaultVersion.Values(), "The branch name of the desired chart value file's version for Teleport")
	clusterCreateCmd.Flags().String("pgbouncer-values", model.PgbouncerDefaultVersion.Values(), "The branch name of the desired chart value file's version for Pgbouncer")
	clusterCreateCmd.Flags().String("networking", "amazon-vpc-routed-eni", "Networking mode to use, for example: weave, calico, canal, amazon-vpc-routed-eni")
	clusterCreateCmd.Flags().String("vpc", "", "Set to use a shared VPC")

	clusterCreateCmd.Flags().StringArray("annotation", []string{}, "Additional annotations for the cluster. Accepts multiple values, for example: '... --annotation abc --annotation def'")

	clusterProvisionCmd.Flags().String("cluster", "", "The id of the cluster to be provisioned.")
	clusterProvisionCmd.Flags().String("prometheus-operator-version", "", "The version of the Prometheus Operator Helm chart")
	clusterProvisionCmd.Flags().String("thanos-version", "", "The version of the Thanos Helm chart")
	clusterProvisionCmd.Flags().String("fluentbit-version", "", "The version of the Fluent-Bit Helm chart")
	clusterProvisionCmd.Flags().String("nginx-version", "", "The version of the NGINX Helm chart")
	clusterProvisionCmd.Flags().String("nginx-internal-version", "", "The version of the internal NGINX Helm chart")
	clusterProvisionCmd.Flags().String("teleport-version", "", "The version of the Teleport Helm chart")
	clusterProvisionCmd.Flags().String("pgbouncer-version", "", "The version of the Pgbouncer Helm chart")

	clusterProvisionCmd.Flags().String("prometheus-operator-values", "", "The branch name of the desired chart value file's version for Prometheus Operator")
	clusterProvisionCmd.Flags().String("thanos-values", "", "The branch name of the desired chart value file's version for Thanos")
	clusterProvisionCmd.Flags().String("fluentbit-values", "", "The branch name of the desired chart value file's version for Fluent-Bit")
	clusterProvisionCmd.Flags().String("nginx-values", "", "The branch name of the desired chart value file's version for NGINX")
	clusterProvisionCmd.Flags().String("nginx-internal-values", "", "The branch name of the desired chart value file's version for NGINX Internal")
	clusterProvisionCmd.Flags().String("teleport-values", "", "The branch name of the desired chart value file's version for Teleport")
	clusterProvisionCmd.Flags().String("pgbouncer-values", "", "The branch name of the desired chart value file's version for Pgbouncer")

	clusterProvisionCmd.MarkFlagRequired("cluster")

	clusterUpdateCmd.Flags().String("cluster", "", "The id of the cluster to be updated.")
	clusterUpdateCmd.Flags().Bool("allow-installations", true, "Whether the cluster will allow for new installations to be scheduled.")
	clusterUpdateCmd.MarkFlagRequired("cluster")

	clusterUpgradeCmd.Flags().String("cluster", "", "The id of the cluster to be upgraded.")
	clusterUpgradeCmd.Flags().String("version", "", "The Kubernetes version to target. Use 'latest' or versions such as '1.16.10'.")
	clusterUpgradeCmd.Flags().String("kops-ami", "", "The AMI to use for the cluster hosts. Use 'latest' for the default kops image.")
	clusterUpgradeCmd.Flags().Bool("use-rotator", true, "Whether the cluster will be upgraded using the node rotator.")
	clusterUpgradeCmd.Flags().Int("max-scaling", 5, "The maximum number of nodes to rotate every time. If the number is bigger than the number of nodes, then the number of nodes will be the maximum number.")
	clusterUpgradeCmd.Flags().Int("max-drain-retries", 10, "The number of times to retry a node drain.")
	clusterUpgradeCmd.Flags().Int("evict-grace-period", 600, "The pod eviction grace period when draining in seconds.")
	clusterUpgradeCmd.Flags().Int("wait-between-rotations", 60, "Τhe time to wait between each rotation of a group of nodes.")
	clusterUpgradeCmd.Flags().Int("wait-between-drains", 60, "The time to wait between each node drain in a group of nodes.")
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

	registerPagingFlags(clusterListCmd)
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
	clusterCmd.AddCommand(clusterAnnotationCmd)
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
		annotations, _ := command.Flags().GetStringArray("annotation")
		networking, _ := command.Flags().GetString("networking")
		vpc, _ := command.Flags().GetString("vpc")

		request := &model.CreateClusterRequest{
			Provider:               provider,
			Version:                version,
			KopsAMI:                kopsAMI,
			Zones:                  strings.Split(zones, ","),
			AllowInstallations:     allowInstallations,
			DesiredUtilityVersions: processUtilityFlags(command),
			Annotations:            annotations,
			Networking:             networking,
			VPC:                    vpc,
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
		useRotator, _ := command.Flags().GetBool("use-rotator")
		maxScaling, _ := command.Flags().GetInt("max-scaling")
		maxDrainRetries, _ := command.Flags().GetInt("max-drain-retries")
		evictGracePeriod, _ := command.Flags().GetInt("evict-grace-period")
		waitBetweenRotations, _ := command.Flags().GetInt("wait-between-rotations")
		waitBetweenDrains, _ := command.Flags().GetInt("wait-between-drains")

		rotatorConfig := model.RotatorConfig{
			UseRotator:           &useRotator,
			MaxScaling:           &maxScaling,
			MaxDrainRetries:      &maxDrainRetries,
			EvictGracePeriod:     &evictGracePeriod,
			WaitBetweenRotations: &waitBetweenRotations,
			WaitBetweenDrains:    &waitBetweenDrains,
		}

		request := &model.PatchUpgradeClusterRequest{
			Version:       getStringFlagPointer(command, "version"),
			KopsAMI:       getStringFlagPointer(command, "kops-ami"),
			RotatorConfig: &rotatorConfig,
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

		paging := parsePagingFlags(command)

		clusters, err := client.GetClusters(&model.GetClustersRequest{
			Paging: paging,
		})
		if err != nil {
			return errors.Wrap(err, "failed to query clusters")
		}

		outputToTable, _ := command.Flags().GetBool("table")
		if outputToTable {
			table := tablewriter.NewWriter(os.Stdout)
			table.SetAlignment(tablewriter.ALIGN_LEFT)
			table.SetHeader([]string{"ID", "STATE", "VERSION", "MASTER NODES", "WORKER NODES", "NETWORKING", "VPC"})

			for _, cluster := range clusters {
				table.Append([]string{
					cluster.ID,
					cluster.State,
					cluster.ProvisionerMetadataKops.Version,
					fmt.Sprintf("%d x %s", cluster.ProvisionerMetadataKops.MasterCount, cluster.ProvisionerMetadataKops.MasterInstanceType),
					fmt.Sprintf("%d x %s (max %d)", cluster.ProvisionerMetadataKops.NodeMinCount, cluster.ProvisionerMetadataKops.NodeInstanceType, cluster.ProvisionerMetadataKops.NodeMaxCount),
					cluster.ProvisionerMetadataKops.Networking,
					cluster.ProvisionerMetadataKops.VPC,
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

func processUtilityFlags(command *cobra.Command) map[string]*model.HelmUtilityVersion {
	prometheusOperatorVersion, _ := command.Flags().GetString("prometheus-operator-version")
	thanosVersion, _ := command.Flags().GetString("thanos-version")
	fluentbitVersion, _ := command.Flags().GetString("fluentbit-version")
	nginxVersion, _ := command.Flags().GetString("nginx-version")
	nginxInternalVersion, _ := command.Flags().GetString("nginx-internal-version")
	teleportVersion, _ := command.Flags().GetString("teleport-version")
	pgbouncerVersion, _ := command.Flags().GetString("pgbouncer-version")

	prometheusOperatorValues, _ := command.Flags().GetString("prometheus-operator-values")
	thanosValues, _ := command.Flags().GetString("thanos-values")
	fluentbitValues, _ := command.Flags().GetString("fluentbit-values")
	nginxValues, _ := command.Flags().GetString("nginx-values")
	nginxInternalValues, _ := command.Flags().GetString("nginx-internal-values")
	teleportValues, _ := command.Flags().GetString("teleport-values")
	pgbouncerValues, _ := command.Flags().GetString("pgbouncer-values")

	utilityVersions := make(map[string]*model.HelmUtilityVersion)

	if prometheusOperatorVersion != "" && prometheusOperatorValues != "" {
		utilityVersions[model.PrometheusOperatorCanonicalName] = &model.HelmUtilityVersion{Chart: prometheusOperatorVersion, ValuesPath: prometheusOperatorValues}
	}

	if thanosVersion != "" && thanosValues != "" {
		utilityVersions[model.ThanosCanonicalName] = &model.HelmUtilityVersion{Chart: thanosVersion, ValuesPath: thanosValues}
	}

	if fluentbitVersion != "" && fluentbitValues != "" {
		utilityVersions[model.FluentbitCanonicalName] = &model.HelmUtilityVersion{Chart: fluentbitVersion, ValuesPath: fluentbitValues}
	}

	if nginxVersion != "" && nginxValues != "" {
		utilityVersions[model.NginxCanonicalName] = &model.HelmUtilityVersion{Chart: nginxVersion, ValuesPath: nginxValues}
	}

	if nginxInternalVersion != "" && nginxInternalValues != "" {
		utilityVersions[model.NginxInternalCanonicalName] = &model.HelmUtilityVersion{Chart: nginxInternalVersion, ValuesPath: nginxInternalValues}
	}

	if teleportVersion != "" && teleportValues != "" {
		utilityVersions[model.TeleportCanonicalName] = &model.HelmUtilityVersion{Chart: teleportVersion, ValuesPath: teleportValues}
	}

	if pgbouncerVersion != "" && pgbouncerValues != "" {
		utilityVersions[model.PgbouncerCanonicalName] = &model.HelmUtilityVersion{Chart: pgbouncerVersion, ValuesPath: pgbouncerValues}
	}

	return utilityVersions
}
