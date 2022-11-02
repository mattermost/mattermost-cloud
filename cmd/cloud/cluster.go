// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.
//

package main

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/pkg/errors"
	"github.com/spf13/cobra"

	"github.com/mattermost/mattermost-cloud/clusterdictionary"
	"github.com/mattermost/mattermost-cloud/model"
)

// Defaults
const (
	useRotatorDefault              = true
	maxScalingDefault              = 5
	maxDrainRetriesDefault         = 10
	evictGracePeriodDefault        = 3600 // seconds
	waitBetweenRotationsDefault    = 180  // seconds
	waitBetweenDrainsDefault       = 1800 // seconds
	waitBetweenPodEvictionsDefault = 5    // seconds
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
	clusterCreateCmd.Flags().Int64("max-pods-per-node", 0, "The maximum number of pods that can run on a single worker node.")
	clusterCreateCmd.Flags().String("zones", "us-east-1a", "The zones where the cluster will be deployed. Use commas to separate multiple zones.")
	clusterCreateCmd.Flags().Bool("allow-installations", true, "Whether the cluster will allow for new installations to be scheduled.")
	clusterCreateCmd.Flags().String("prometheus-operator-version", "", "The version of Prometheus Operator to provision. Use 'stable' to provision the latest stable version published upstream.")
	clusterCreateCmd.Flags().String("thanos-version", "", "The version of Thanos to provision. Use 'stable' to provision the latest stable version published upstream.")
	clusterCreateCmd.Flags().String("fluentbit-version", "", "The version of Fluentbit to provision. Use 'stable' to provision the latest stable version published upstream.")
	clusterCreateCmd.Flags().String("nginx-version", "", "The version of Nginx Internal to provision. Use 'stable' to provision the latest stable version published upstream.")
	clusterCreateCmd.Flags().String("nginx-internal-version", "", "The version of Nginx to provision. Use 'stable' to provision the latest stable version published upstream.")
	clusterCreateCmd.Flags().String("teleport-version", "", "The version of Teleport to provision. Use 'stable' to provision the latest stable version published upstream.")
	clusterCreateCmd.Flags().String("pgbouncer-version", "", "The version of Pgbouncer to provision. Use 'stable' to provision the latest stable version published upstream.")
	clusterCreateCmd.Flags().String("rtcd-version", "", "The version of RTCD to provision. Use 'stable' to provision the latest stable version published upstream.")
	clusterCreateCmd.Flags().String("promtail-version", "", "The version of Promtail to provision. Use 'stable' to provision the latest stable version published upstream.")
	clusterCreateCmd.Flags().String("kubecost-version", "", "The version of Kubecost. Use 'stable' to provision the latest stable version published upstream.")
	clusterCreateCmd.Flags().String("node-problem-detector-version", "", "The version of Node Problem Detector. Use 'stable' to provision the latest stable version published upstream.")
	clusterCreateCmd.Flags().String("metrics-server-version", "", "The version of Metrics Server. Use 'stable' to provision the latest stable version published upstream.")
	clusterCreateCmd.Flags().String("velero-version", "", "The version of Velero. Use 'stable' to provision the latest stable version published upstream.")
	clusterCreateCmd.Flags().String("cloudprober-version", "", "The version of Cloudprober. Use 'stable' to provision the latest stable version published upstream.")
	clusterCreateCmd.Flags().String("prometheus-operator-values", "", "The full Git URL of the desired chart value file's version for Prometheus Operator")
	clusterCreateCmd.Flags().String("thanos-values", "", "The full Git URL of the desired chart value file's version for Thanos")
	clusterCreateCmd.Flags().String("fluentbit-values", "", "The full Git URL of the desired chart value file's version for Fluent-Bit")
	clusterCreateCmd.Flags().String("nginx-values", "", "The full Git URL of the desired chart value file's version for NGINX")
	clusterCreateCmd.Flags().String("nginx-internal-values", "", "The full Git URL of the desired chart value file's version for NGINX Internal")
	clusterCreateCmd.Flags().String("teleport-values", "", "The full Git URL of the desired chart value file's version for Teleport")
	clusterCreateCmd.Flags().String("pgbouncer-values", "", "The full Git URL of the desired chart value file's version for Pgbouncer")
	clusterCreateCmd.Flags().String("rtcd-values", "", "The full Git URL of the desired chart value file's version for RTCD")
	clusterCreateCmd.Flags().String("promtail-values", "", "The full Git URL of the desired chart value file's version for Promtail")
	clusterCreateCmd.Flags().String("kubecost-values", "", "The full Git URL of the desired chart value file's version for Kubecost")
	clusterCreateCmd.Flags().String("node-problem-detector-values", "", "The full Git URL of the desired chart value file's version for Node Problem Detector")
	clusterCreateCmd.Flags().String("metrics-server-values", "", "The full Git URL of the desired chart value file's version for Metrics Server")
	clusterCreateCmd.Flags().String("velero-values", "", "The full Git URL of the desired chart value file's version for Velero")
	clusterCreateCmd.Flags().String("cloudprober-values", "", "The full Git URL of the desired chart value file's version for Cloudprober")
	clusterCreateCmd.Flags().String("networking", "calico", "Networking mode to use, for example: weave, calico, canal, amazon-vpc-routed-eni")
	clusterCreateCmd.Flags().String("vpc", "", "Set to use a shared VPC")
	clusterCreateCmd.Flags().String("cluster", "", "The id of the cluster. If provided and the cluster exists the creation will be retried ignoring other parameters.")

	clusterCreateCmd.Flags().StringArray("annotation", []string{}, "Additional annotations for the cluster. Accepts multiple values, for example: '... --annotation abc --annotation def'")

	// EKS flags
	clusterCreateCmd.Flags().Bool("eks", false, "Create EKS cluster.")
	clusterCreateCmd.Flags().String("eks-role-arn", "", "EKS role ARN.")
	clusterCreateCmd.Flags().String("eks-node-groups-config", "", "Path to node groups configuration in JSON format.")

	clusterProvisionCmd.Flags().String("cluster", "", "The id of the cluster to be provisioned.")
	clusterProvisionCmd.Flags().String("prometheus-operator-version", "", "The version of the Prometheus Operator Helm chart")
	clusterProvisionCmd.Flags().String("thanos-version", "", "The version of the Thanos Helm chart")
	clusterProvisionCmd.Flags().String("fluentbit-version", "", "The version of the Fluent-Bit Helm chart")
	clusterProvisionCmd.Flags().String("nginx-version", "", "The version of the NGINX Helm chart")
	clusterProvisionCmd.Flags().String("nginx-internal-version", "", "The version of the internal NGINX Helm chart")
	clusterProvisionCmd.Flags().String("teleport-version", "", "The version of the Teleport Helm chart")
	clusterProvisionCmd.Flags().String("pgbouncer-version", "", "The version of the Pgbouncer Helm chart")
	clusterProvisionCmd.Flags().String("rtcd-version", "", "The version of the RTCD Helm chart")
	clusterProvisionCmd.Flags().String("promtail-version", "", "The version of the Promtail Helm chart")
	clusterProvisionCmd.Flags().String("kubecost-version", "", "The version of the Kubecost Helm chart")
	clusterProvisionCmd.Flags().String("node-problem-detector-version", "", "The version of the Node Problem Detector Helm chart")
	clusterProvisionCmd.Flags().String("metrics-server-version", "", "The version of the Metrics Server Helm chart")
	clusterProvisionCmd.Flags().String("velero-version", "", "The version of Velero. Use 'stable' to provision the latest stable version published upstream.")
	clusterProvisionCmd.Flags().String("cloudprober-version", "", "The version of Cloudprober. Use 'stable' to provision the latest stable version published upstream.")

	clusterProvisionCmd.Flags().String("prometheus-operator-values", "", "The full Git URL of the desired chart values for Prometheus Operator")
	clusterProvisionCmd.Flags().String("thanos-values", "", "The full Git URL of the desired chart values for Thanos")
	clusterProvisionCmd.Flags().String("fluentbit-values", "", "The full Git URL of the desired chart values for Fluentbit")
	clusterProvisionCmd.Flags().String("nginx-values", "", "The full Git URL of the desired chart values for Nginx")
	clusterProvisionCmd.Flags().String("nginx-internal-values", "", "The full Git URL of the desired chart values for Nginx Internal")
	clusterProvisionCmd.Flags().String("teleport-values", "", "The full Git URL of the desired chart values for Teleport")
	clusterProvisionCmd.Flags().String("pgbouncer-values", "", "The full Git URL of the desired chart values for PGBouncer")
	clusterProvisionCmd.Flags().String("rtcd-values", "", "The full Git URL of the desired chart values for RTCD")
	clusterProvisionCmd.Flags().String("promtail-values", "", "The full Git URL of the desired chart values for Promtail")
	clusterProvisionCmd.Flags().String("kubecost-values", "", "The full Git URL of the desired Kubecost chart")
	clusterProvisionCmd.Flags().String("node-problem-detector-values", "", "The full Git URL of the desired chart values for the Node Problem Detector")
	clusterProvisionCmd.Flags().String("metrics-server-values", "", "The full Git URL of the desired chart values for the Metrics Server")
	clusterProvisionCmd.Flags().String("velero-values", "", "The full Git URL of the desired chart value file's version for Velero")
	clusterProvisionCmd.Flags().String("cloudprober-values", "", "The full Git URL of the desired chart value file's version for Cloudprober")
	clusterProvisionCmd.Flags().Bool("reprovision-all-utilities", false, "Set to true if all utilities should be reprovisioned and not just ones with new versions")

	clusterProvisionCmd.MarkFlagRequired("cluster")

	clusterUpdateCmd.Flags().String("cluster", "", "The id of the cluster to be updated.")
	clusterUpdateCmd.Flags().Bool("allow-installations", true, "Whether the cluster will allow for new installations to be scheduled.")
	clusterUpdateCmd.MarkFlagRequired("cluster")

	clusterUpgradeCmd.Flags().String("cluster", "", "The id of the cluster to be upgraded.")
	clusterUpgradeCmd.Flags().String("version", "", "The Kubernetes version to target. Use 'latest' or versions such as '1.16.10'.")
	clusterUpgradeCmd.Flags().String("kops-ami", "", "The AMI to use for the cluster hosts. Use 'latest' for the default kops image.")
	clusterUpgradeCmd.Flags().Int64("max-pods-per-node", 0, "The maximum number of pods that can run on a single worker node.")
	clusterUpgradeCmd.Flags().Bool("use-rotator", useRotatorDefault, "Whether the cluster will be upgraded using the node rotator.")
	clusterUpgradeCmd.Flags().Int("max-scaling", maxScalingDefault, "The maximum number of nodes to rotate every time. If the number is bigger than the number of nodes, then the number of nodes will be the maximum number.")
	clusterUpgradeCmd.Flags().Int("max-drain-retries", maxDrainRetriesDefault, "The number of times to retry a node drain.")
	clusterUpgradeCmd.Flags().Int("evict-grace-period", evictGracePeriodDefault, "The pod eviction grace period when draining in seconds.")
	clusterUpgradeCmd.Flags().Int("wait-between-rotations", waitBetweenRotationsDefault, "Τhe time in seconds to wait between each rotation of a group of nodes.")
	clusterUpgradeCmd.Flags().Int("wait-between-drains", waitBetweenDrainsDefault, "The time in seconds to wait between each node drain in a group of nodes.")
	clusterUpgradeCmd.Flags().Int("wait-between-pod-evictions", waitBetweenPodEvictionsDefault, "The time in seconds to wait between each pod eviction in a node drain.")
	clusterUpgradeCmd.MarkFlagRequired("cluster")

	clusterResizeCmd.Flags().String("cluster", "", "The id of the cluster to be resized.")
	clusterResizeCmd.Flags().String("size", "", "The size constant describing the cluster")
	clusterResizeCmd.Flags().String("size-node-instance-type", "", "The instance type describing the k8s worker nodes. Overwrites value from 'size'.")
	clusterResizeCmd.Flags().Int64("size-node-min-count", 0, "The minimum number of k8s worker nodes. Overwrites value from 'size'.")
	clusterResizeCmd.Flags().Int64("size-node-max-count", 0, "The maximum number of k8s worker nodes. Overwrites value from 'size'.")
	clusterResizeCmd.Flags().Bool("use-rotator", useRotatorDefault, "Whether the cluster will be upgraded using the node rotator.")
	clusterResizeCmd.Flags().Int("max-scaling", maxScalingDefault, "The maximum number of nodes to rotate every time. If the number is bigger than the number of nodes, then the number of nodes will be the maximum number.")
	clusterResizeCmd.Flags().Int("max-drain-retries", maxDrainRetriesDefault, "The number of times to retry a node drain.")
	clusterResizeCmd.Flags().Int("evict-grace-period", evictGracePeriodDefault, "The pod eviction grace period when draining in seconds.")
	clusterResizeCmd.Flags().Int("wait-between-rotations", waitBetweenRotationsDefault, "Τhe time in seconds to wait between each rotation of a group of nodes.")
	clusterResizeCmd.Flags().Int("wait-between-drains", waitBetweenDrainsDefault, "The time in seconds to wait between each node drain in a group of nodes.")
	clusterResizeCmd.Flags().Int("wait-between-pod-evictions", waitBetweenPodEvictionsDefault, "The time in seconds to wait between each pod eviction in a node drain.")
	clusterResizeCmd.MarkFlagRequired("cluster")

	clusterDeleteCmd.Flags().String("cluster", "", "The id of the cluster to be deleted.")
	clusterDeleteCmd.MarkFlagRequired("cluster")

	clusterGetCmd.Flags().String("cluster", "", "The id of the cluster to be fetched.")
	clusterGetCmd.MarkFlagRequired("cluster")

	registerPagingFlags(clusterListCmd)
	registerTableOutputFlags(clusterListCmd)

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

// getRotatorConfigFromFlags creates a new RotatorConfig with the flags provided to the command
func getRotatorConfigFromFlags(command *cobra.Command) model.RotatorConfig {
	useRotator, _ := command.Flags().GetBool("use-rotator")
	maxScaling, _ := command.Flags().GetInt("max-scaling")
	maxDrainRetries, _ := command.Flags().GetInt("max-drain-retries")
	evictGracePeriod, _ := command.Flags().GetInt("evict-grace-period")
	waitBetweenRotations, _ := command.Flags().GetInt("wait-between-rotations")
	waitBetweenDrains, _ := command.Flags().GetInt("wait-between-drains")
	waitBetweenPodEvictions, _ := command.Flags().GetInt("wait-between-pod-evictions")

	return model.RotatorConfig{
		UseRotator:              &useRotator,
		MaxScaling:              &maxScaling,
		MaxDrainRetries:         &maxDrainRetries,
		EvictGracePeriod:        &evictGracePeriod,
		WaitBetweenRotations:    &waitBetweenRotations,
		WaitBetweenDrains:       &waitBetweenDrains,
		WaitBetweenPodEvictions: &waitBetweenPodEvictions,
	}
}

var clusterCreateCmd = &cobra.Command{
	Use:   "create",
	Short: "Create a cluster.",
	RunE: func(command *cobra.Command, args []string) error {
		command.SilenceUsage = true

		serverAddress, _ := command.Flags().GetString("server")
		client := model.NewClient(serverAddress)

		clusterID, _ := command.Flags().GetString("cluster")
		if clusterID != "" {
			err := client.RetryCreateCluster(clusterID)
			if err != nil {
				return errors.Wrap(err, "failed to retry cluster creation")
			}
			return nil
		}

		provider, _ := command.Flags().GetString("provider")
		version, _ := command.Flags().GetString("version")
		kopsAMI, _ := command.Flags().GetString("kops-ami")
		zones, _ := command.Flags().GetString("zones")
		allowInstallations, _ := command.Flags().GetBool("allow-installations")
		annotations, _ := command.Flags().GetStringArray("annotation")
		networking, _ := command.Flags().GetString("networking")
		vpc, _ := command.Flags().GetString("vpc")

		useEKS, _ := command.Flags().GetBool("eks")
		eksRoleArn, _ := command.Flags().GetString("eks-role-arn")
		eksNodeGroupsConfig, _ := command.Flags().GetString("eks-node-groups-config")

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

		if useEKS {
			nodeGroupsConfigRaw, err := os.ReadFile(eksNodeGroupsConfig)
			if err != nil {
				return errors.Wrap(err, "failed to read node groups config")
			}
			var nodeGroupsConfig model.EKSNodeGroups
			err = json.Unmarshal(nodeGroupsConfigRaw, &nodeGroupsConfig)
			if err != nil {
				return errors.Wrap(err, "failed to unmarshal node groups config")
			}
			request.EKSConfig = &model.EKSConfig{
				ClusterRoleARN: &eksRoleArn,
				NodeGroups:     nodeGroupsConfig,
			}
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
		maxPodsPerNode, _ := command.Flags().GetInt64("max-pods-per-node")
		if maxPodsPerNode != 0 {
			request.MaxPodsPerNode = maxPodsPerNode
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
	Short: "Provision/Re-provision a cluster's k8s resources.",
	RunE: func(command *cobra.Command, args []string) error {
		command.SilenceUsage = true

		serverAddress, _ := command.Flags().GetString("server")
		client := model.NewClient(serverAddress)
		clusterID, _ := command.Flags().GetString("cluster")

		var request *model.ProvisionClusterRequest = new(model.ProvisionClusterRequest)
		request.Force, _ = command.Flags().GetBool("reprovision-all-utilities")

		if desiredUtilityVersions := processUtilityFlags(command); len(desiredUtilityVersions) > 0 {
			request.DesiredUtilityVersions = desiredUtilityVersions
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
		rotatorConfig := getRotatorConfigFromFlags(command)

		request := &model.PatchUpgradeClusterRequest{
			Version:        getStringFlagPointer(command, "version"),
			KopsAMI:        getStringFlagPointer(command, "kops-ami"),
			MaxPodsPerNode: getInt64FlagPointer(command, "max-pods-per-node"),
			RotatorConfig:  &rotatorConfig,
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
		rotatorConfig := getRotatorConfigFromFlags(command)

		request := &model.PatchClusterSizeRequest{
			RotatorConfig: &rotatorConfig,
		}

		// Apply values from 'size' constant and then apply overrides.
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

		if enabled, customCols := tableOutputEnabled(command); enabled {
			var keys []string
			var vals [][]string

			if len(customCols) > 0 {
				data := make([]interface{}, 0, len(clusters))
				for _, cluster := range clusters {
					data = append(data, cluster)
				}
				keys, vals, err = prepareTableData(customCols, data)
				if err != nil {
					return errors.Wrap(err, "failed to prepare table output")
				}
			} else {
				keys, vals = defaultClustersTableData(clusters)
			}

			printTable(keys, vals)
			return nil
		}

		err = printJSON(clusters)
		if err != nil {
			return errors.Wrap(err, "failed to print cluster response")
		}

		return nil
	},
}

func defaultClustersTableData(clusters []*model.ClusterDTO) ([]string, [][]string) {
	keys := []string{"ID", "STATE", "VERSION", "MASTER NODES", "WORKER NODES", "AMI ID", "NETWORKING", "VPC", "STATUS"}
	values := make([][]string, 0, len(clusters))

	for _, cluster := range clusters {
		status := "offline"
		if cluster.AllowInstallations {
			status = "online"
		}
		values = append(values, []string{
			cluster.ID,
			cluster.State,
			cluster.ProvisionerMetadataKops.Version,
			fmt.Sprintf("%d x %s", cluster.ProvisionerMetadataKops.MasterCount, cluster.ProvisionerMetadataKops.MasterInstanceType),
			fmt.Sprintf("%d x %s (max %d)", cluster.ProvisionerMetadataKops.NodeMinCount, cluster.ProvisionerMetadataKops.NodeInstanceType, cluster.ProvisionerMetadataKops.NodeMaxCount),
			cluster.ProvisionerMetadataKops.AMI,
			cluster.ProvisionerMetadataKops.Networking,
			cluster.ProvisionerMetadataKops.VPC,
			status,
		})
	}
	return keys, values
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

// MustGetString works like Cobra Commander's GetString but panicks on
// error, similar to regexp.MustCompile vs regexp.Compile
func MustGetString(key string, command *cobra.Command) string {
	field, err := command.Flags().GetString(key)
	if err != nil {
		panic(err)
	}
	return field
}

// processUtilityFlags handles processing the arguments passed for all
// of the utilities, for cloud cluster create & cloud cluster
// provision.
func processUtilityFlags(command *cobra.Command) map[string]*model.HelmUtilityVersion {
	return map[string]*model.HelmUtilityVersion{
		model.PrometheusOperatorCanonicalName: {
			Chart:      MustGetString("prometheus-operator-version", command),
			ValuesPath: MustGetString("prometheus-operator-values", command)},
		model.ThanosCanonicalName: {
			Chart:      MustGetString("thanos-version", command),
			ValuesPath: MustGetString("thanos-values", command)},
		model.FluentbitCanonicalName: {
			Chart:      MustGetString("fluentbit-version", command),
			ValuesPath: MustGetString("fluentbit-values", command)},
		model.NginxCanonicalName: {
			Chart:      MustGetString("nginx-version", command),
			ValuesPath: MustGetString("nginx-values", command)},
		model.NginxInternalCanonicalName: {
			Chart:      MustGetString("nginx-internal-version", command),
			ValuesPath: MustGetString("nginx-internal-values", command)},
		model.TeleportCanonicalName: {
			Chart:      MustGetString("teleport-version", command),
			ValuesPath: MustGetString("teleport-values", command)},
		model.PgbouncerCanonicalName: {
			Chart:      MustGetString("pgbouncer-version", command),
			ValuesPath: MustGetString("pgbouncer-values", command)},
		model.PromtailCanonicalName: {
			Chart:      MustGetString("promtail-version", command),
			ValuesPath: MustGetString("promtail-values", command)},
		model.RtcdCanonicalName: {
			Chart:      MustGetString("rtcd-version", command),
			ValuesPath: MustGetString("rtcd-values", command)},
		model.KubecostCanonicalName: {
			Chart:      MustGetString("kubecost-version", command),
			ValuesPath: MustGetString("kubecost-values", command)},
		model.NodeProblemDetectorCanonicalName: {
			Chart:      MustGetString("node-problem-detector-version", command),
			ValuesPath: MustGetString("node-problem-detector-values", command)},
		model.MetricsServerCanonicalName: {
			Chart:      MustGetString("metrics-server-version", command),
			ValuesPath: MustGetString("metrics-server-values", command)},
		model.VeleroCanonicalName: {
			Chart:      MustGetString("velero-version", command),
			ValuesPath: MustGetString("velero-values", command)},
		model.CloudproberCanonicalName: {
			Chart:      MustGetString("cloudprober-version", command),
			ValuesPath: MustGetString("cloudprober-values", command)},
	}
}
