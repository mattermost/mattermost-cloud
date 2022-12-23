// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.
//

package main

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/mattermost/mattermost-cloud/clusterdictionary"
	"github.com/mattermost/mattermost-cloud/model"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
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

func newCmdCluster() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "cluster",
		Short: "Manipulate clusters managed by the provisioning server.",
	}

	setClusterFlags(cmd)

	cmd.AddCommand(newCmdClusterCreate())
	cmd.AddCommand(newCmdClusterProvision())
	cmd.AddCommand(newCmdClusterUpdate())
	cmd.AddCommand(newCmdClusterUpgrade())
	cmd.AddCommand(newCmdClusterResize())
	cmd.AddCommand(newCmdClusterDelete())
	cmd.AddCommand(newCmdClusterGet())
	cmd.AddCommand(newCmdClusterList())
	cmd.AddCommand(newCmdClusterUtilities())

	cmd.AddCommand(newCmdClusterSizeDictionary())
	cmd.AddCommand(newCmdClusterShowStateReport())

	cmd.AddCommand(newCmdClusterAnnotation())
	cmd.AddCommand(newCmdClusterInstallation())

	return cmd
}

func printJSON(data interface{}) error {
	encoder := json.NewEncoder(os.Stdout)
	encoder.SetIndent("", "    ")
	return encoder.Encode(data)
}

// getRotatorConfigFromFlags creates a new RotatorConfig with the flags provided to the command
func getRotatorConfigFromFlags(rc rotatorConfig) model.RotatorConfig {
	return model.RotatorConfig{
		UseRotator:              &rc.useRotator,
		MaxScaling:              &rc.maxScaling,
		MaxDrainRetries:         &rc.maxDrainRetries,
		EvictGracePeriod:        &rc.evictGracePeriod,
		WaitBetweenRotations:    &rc.waitBetweenRotations,
		WaitBetweenDrains:       &rc.waitBetweenDrains,
		WaitBetweenPodEvictions: &rc.waitBetweenPodEvictions,
	}
}

func newCmdClusterCreate() *cobra.Command {
	var flags clusterCreateFlags

	cmd := &cobra.Command{
		Use:   "create",
		Short: "Create a cluster.",
		RunE: func(command *cobra.Command, args []string) error {
			command.SilenceUsage = true
			return executeClusterCreateCmd(flags)
		},
		PreRun: func(cmd *cobra.Command, args []string) {
			flags.clusterFlags.addFlags(cmd)
			return
		},
	}
	flags.addFlags(cmd)

	return cmd
}

func executeClusterCreateCmd(flags clusterCreateFlags) error {
	client := model.NewClient(flags.serverAddress)

	if flags.cluster != "" {
		err := client.RetryCreateCluster(flags.cluster)
		if err != nil {
			return errors.Wrap(err, "failed to retry cluster creation")
		}
		return nil
	}

	request := &model.CreateClusterRequest{
		Provider:               flags.provider,
		Version:                flags.version,
		KopsAMI:                flags.kopsAMI,
		Zones:                  strings.Split(flags.zones, ","),
		AllowInstallations:     flags.allowInstallations,
		DesiredUtilityVersions: processUtilityFlags(flags.utilityFlags),
		Annotations:            flags.annotations,
		Networking:             flags.networking,
		VPC:                    flags.vpc,
	}

	if flags.useEKS {
		nodeGroupsConfigRaw, err := os.ReadFile(flags.eksNodeGroupsConfig)
		if err != nil {
			return errors.Wrap(err, "failed to read node groups config")
		}
		var nodeGroupsConfig model.EKSNodeGroups
		err = json.Unmarshal(nodeGroupsConfigRaw, &nodeGroupsConfig)
		if err != nil {
			return errors.Wrap(err, "failed to unmarshal node groups config")
		}
		request.EKSConfig = &model.EKSConfig{
			ClusterRoleARN: &flags.eksRoleArn,
			NodeGroups:     nodeGroupsConfig,
		}
	}

	err := clusterdictionary.ApplyToCreateClusterRequest(flags.size, request)
	if err != nil {
		return errors.Wrap(err, "failed to apply size values")
	}

	if len(flags.masterInstanceType) != 0 {
		request.MasterInstanceType = flags.masterInstanceType
	}

	if flags.masterCount != 0 {
		request.MasterCount = flags.masterCount
	}

	if len(flags.nodeInstanceType) != 0 {
		request.NodeInstanceType = flags.nodeInstanceType
	}

	if flags.nodeCount != 0 {
		// Setting different min and max counts in currently not supported
		// with the kops create cluster flag.
		request.NodeMinCount = flags.nodeCount
		request.NodeMaxCount = flags.nodeCount
	}

	if flags.maxPodsPerNode != 0 {
		request.MaxPodsPerNode = flags.maxPodsPerNode
	}

	if flags.dryRun {
		return runDryRun(request)
	}

	cluster, err := client.CreateCluster(request)
	if err != nil {
		return errors.Wrap(err, "failed to create cluster")
	}

	if err = printJSON(cluster); err != nil {
		return errors.Wrap(err, "failed to print cluster response")
	}

	return nil

}

func newCmdClusterProvision() *cobra.Command {
	var flags clusterProvisionFlags

	cmd := &cobra.Command{
		Use:   "provision",
		Short: "Provision/Re-provision a cluster's k8s resources.",
		RunE: func(command *cobra.Command, args []string) error {
			command.SilenceUsage = true
			return executeClusterProvisionCmd(flags)
		},
		PreRun: func(cmd *cobra.Command, args []string) {
			flags.clusterFlags.addFlags(cmd)
			return
		},
	}
	flags.addFlags(cmd)

	return cmd
}

func executeClusterProvisionCmd(flags clusterProvisionFlags) error {
	client := model.NewClient(flags.serverAddress)

	request := &model.ProvisionClusterRequest{
		Force:                  flags.reprovisionAllUtilities,
		DesiredUtilityVersions: processUtilityFlags(flags.utilityFlags),
	}

	if flags.dryRun {
		return runDryRun(request)
	}

	cluster, err := client.ProvisionCluster(flags.cluster, request)
	if err != nil {
		return errors.Wrap(err, "failed to provision cluster")
	}

	if err = printJSON(cluster); err != nil {
		return errors.Wrap(err, "failed to print cluster response")
	}

	return nil

}

func newCmdClusterUpdate() *cobra.Command {
	var flags clusterUpdateFlags

	cmd := &cobra.Command{
		Use:   "update",
		Short: "Updates a cluster's configuration.",
		RunE: func(command *cobra.Command, args []string) error {
			command.SilenceUsage = true
			return executeClusterUpdateCmd(flags)
		},
		PreRun: func(cmd *cobra.Command, args []string) {
			flags.clusterFlags.addFlags(cmd)
			return
		},
	}
	flags.addFlags(cmd)

	return cmd
}

func executeClusterUpdateCmd(flags clusterUpdateFlags) error {

	client := model.NewClient(flags.serverAddress)

	request := &model.UpdateClusterRequest{
		AllowInstallations: flags.allowInstallations,
	}

	if flags.dryRun {
		return runDryRun(request)
	}

	cluster, err := client.UpdateCluster(flags.cluster, request)
	if err != nil {
		return errors.Wrap(err, "failed to update cluster")
	}

	if err = printJSON(cluster); err != nil {
		return errors.Wrap(err, "failed to print cluster response")
	}

	return nil

}

func newCmdClusterUpgrade() *cobra.Command {
	var flags clusterUpgradeFlags

	cmd := &cobra.Command{
		Use:   "upgrade",
		Short: "Upgrade k8s on a cluster.",
		RunE: func(command *cobra.Command, args []string) error {
			command.SilenceUsage = true
			return executeClusterUpgradeCmd(flags)
		},
		PreRun: func(cmd *cobra.Command, args []string) {
			flags.clusterFlags.addFlags(cmd)
			flags.clusterUpgradeFlagChanged.addFlags(cmd)
			return
		},
	}
	flags.addFlags(cmd)

	return cmd
}

func executeClusterUpgradeCmd(flags clusterUpgradeFlags) error {
	client := model.NewClient(flags.serverAddress)

	rotatorConfig := getRotatorConfigFromFlags(flags.rotatorConfig)

	request := &model.PatchUpgradeClusterRequest{
		RotatorConfig: &rotatorConfig,
	}

	if flags.isVersionChanged {
		request.Version = &flags.version
	}
	if flags.isKopsAmiChanged {
		request.KopsAMI = &flags.kopsAMI
	}
	if flags.isMaxPodsPerNodeChanged {
		request.MaxPodsPerNode = &flags.maxPodsPerNode
	}
	if flags.dryRun {
		return runDryRun(request)
	}

	cluster, err := client.UpgradeCluster(flags.cluster, request)
	if err != nil {
		return errors.Wrap(err, "failed to upgrade cluster")
	}

	if err = printJSON(cluster); err != nil {
		return errors.Wrap(err, "failed to print cluster response")
	}

	return nil

}

func newCmdClusterResize() *cobra.Command {
	var flags clusterResizeFlags

	cmd := &cobra.Command{
		Use:   "resize",
		Short: "Resize a k8s cluster",
		RunE: func(command *cobra.Command, args []string) error {
			command.SilenceUsage = true
			return executeClusterResizeCmd(flags)
		},
		PreRun: func(cmd *cobra.Command, args []string) {
			flags.clusterFlags.addFlags(cmd)
			return
		},
	}
	flags.addFlags(cmd)

	return cmd
}

func executeClusterResizeCmd(flags clusterResizeFlags) error {
	client := model.NewClient(flags.serverAddress)

	rotatorConfig := getRotatorConfigFromFlags(flags.rotatorConfig)

	request := &model.PatchClusterSizeRequest{
		RotatorConfig: &rotatorConfig,
	}

	// Apply values from 'size' constant and then apply overrides.
	err := clusterdictionary.ApplyToPatchClusterSizeRequest(flags.size, request)
	if err != nil {
		return errors.Wrap(err, "failed to apply size values")
	}

	if len(flags.nodeInstanceType) != 0 {
		request.NodeInstanceType = &flags.nodeInstanceType
	}

	if flags.nodeMinCount != 0 {
		request.NodeMinCount = &flags.nodeMinCount
	}

	if flags.nodeMaxCount != 0 {
		request.NodeMaxCount = &flags.nodeMaxCount
	}

	if flags.dryRun {
		return runDryRun(request)
	}

	cluster, err := client.ResizeCluster(flags.cluster, request)
	if err != nil {
		return errors.Wrap(err, "failed to resize cluster")
	}

	if err = printJSON(cluster); err != nil {
		return errors.Wrap(err, "failed to print cluster response")
	}

	return nil

}

func newCmdClusterDelete() *cobra.Command {
	var flags clusterDeleteFlags

	cmd := &cobra.Command{
		Use:   "delete",
		Short: "Delete a cluster.",
		RunE: func(command *cobra.Command, args []string) error {
			command.SilenceUsage = true
			return executeClusterDeleteCmd(flags)
		},
		PreRun: func(cmd *cobra.Command, args []string) {
			flags.clusterFlags.addFlags(cmd)
			return
		},
	}
	flags.addFlags(cmd)

	return cmd
}

func executeClusterDeleteCmd(flags clusterDeleteFlags) error {
	client := model.NewClient(flags.serverAddress)

	err := client.DeleteCluster(flags.cluster)
	if err != nil {
		return errors.Wrap(err, "failed to delete cluster")
	}

	return nil
}

func newCmdClusterGet() *cobra.Command {
	var flags clusterGetFlags

	cmd := &cobra.Command{
		Use:   "get",
		Short: "Get a particular cluster.",
		RunE: func(command *cobra.Command, args []string) error {
			command.SilenceUsage = true
			return executeClusterGetCmd(flags)
		},
		PreRun: func(cmd *cobra.Command, args []string) {
			flags.clusterFlags.addFlags(cmd)
			return
		},
	}
	flags.addFlags(cmd)

	return cmd
}

func executeClusterGetCmd(flags clusterGetFlags) error {
	client := model.NewClient(flags.serverAddress)

	cluster, err := client.GetCluster(flags.cluster)
	if err != nil {
		return errors.Wrap(err, "failed to query cluster")
	}
	if cluster == nil {
		return nil
	}

	if err = printJSON(cluster); err != nil {
		return errors.Wrap(err, "failed to print cluster response")
	}
	return nil
}

func newCmdClusterList() *cobra.Command {
	var flags clusterListFlags

	cmd := &cobra.Command{
		Use:   "list",
		Short: "List created clusters.",
		RunE: func(command *cobra.Command, args []string) error {
			command.SilenceUsage = true
			return executeClusterListCmd(flags)
		},
		PreRun: func(cmd *cobra.Command, args []string) {
			flags.clusterFlags.addFlags(cmd)
			return
		},
	}
	flags.addFlags(cmd)

	return cmd
}

func executeClusterListCmd(flags clusterListFlags) error {
	client := model.NewClient(flags.serverAddress)

	paging := getPaging(flags.pagingFlags)

	clusters, err := client.GetClusters(&model.GetClustersRequest{
		Paging: paging,
	})
	if err != nil {
		return errors.Wrap(err, "failed to query clusters")
	}

	if enabled, customCols := getTableOutputOption(flags.tableOptions); enabled {
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

		if flags.showTags {
			keys, vals = enhanceTableWithAnnotations(clusters, keys, vals)
		}

		printTable(keys, vals)
		return nil
	}

	if err = printJSON(clusters); err != nil {
		return errors.Wrap(err, "failed to print cluster response")
	}

	return nil
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

func enhanceTableWithAnnotations(clusters []*model.ClusterDTO, keys []string, vals [][]string) ([]string, [][]string) {
	var tags [][]string
	for _, cluster := range clusters {
		var list []string
		for _, annotation := range cluster.Annotations {
			list = append(list, annotation.Name)
		}
		tags = append(tags, list)
	}
	keys = append(keys, "TAG")
	for i, v := range vals {
		v = append(v, strings.Join(tags[i], ","))
		vals[i] = v
	}

	return keys, vals
}

func newCmdClusterUtilities() *cobra.Command {
	var flags clusterUtilitiesFlags

	cmd := &cobra.Command{
		Use:   "utilities",
		Short: "Show metadata regarding utility services running in a cluster.",
		RunE: func(command *cobra.Command, args []string) error {
			command.SilenceUsage = true
			client := model.NewClient(flags.serverAddress)

			metadata, err := client.GetClusterUtilities(flags.cluster)
			if err != nil {
				return err
			}

			return printJSON(metadata)
		},
		PreRun: func(cmd *cobra.Command, args []string) {
			flags.clusterFlags.addFlags(cmd)
			return
		},
	}
	flags.addFlags(cmd)

	return cmd
}

func newCmdClusterSizeDictionary() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "dictionary",
		Short: "Shows predefined cluster size templates.",
		RunE: func(command *cobra.Command, args []string) error {
			command.SilenceUsage = true
			if err := printJSON(clusterdictionary.ValidSizes); err != nil {
				return errors.Wrap(err, "failed to print cluster dictionary")
			}
			return nil
		},
	}

	return cmd
}

// TODO:
// Instead of showing the state data from the model of the CLI binary, add a new
// API endpoint to return the server's state model.
func newCmdClusterShowStateReport() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "state-report",
		Short: "Shows information regarding changing cluster state.",
		RunE: func(command *cobra.Command, args []string) error {
			command.SilenceUsage = true
			if err := printJSON(model.GetClusterRequestStateReport()); err != nil {
				return err
			}
			return nil
		},
	}

	return cmd
}

// processUtilityFlags handles processing the arguments passed for all
// of the utilities, for cloud cluster create & cloud cluster
// provision.
func processUtilityFlags(utilityFlags utilityFlags) map[string]*model.HelmUtilityVersion {
	return map[string]*model.HelmUtilityVersion{
		model.PrometheusOperatorCanonicalName: {
			Chart:      utilityFlags.prometheusOperatorVersion,
			ValuesPath: utilityFlags.prometheusOperatorValues,
		},
		model.ThanosCanonicalName: {
			Chart:      utilityFlags.thanosVersion,
			ValuesPath: utilityFlags.thanosValues,
		},
		model.FluentbitCanonicalName: {
			Chart:      utilityFlags.fluentbitVersion,
			ValuesPath: utilityFlags.fluentbitValues,
		},
		model.NginxCanonicalName: {
			Chart:      utilityFlags.nginxVersion,
			ValuesPath: utilityFlags.nginxValues,
		},
		model.NginxInternalCanonicalName: {
			Chart:      utilityFlags.nginxInternalVersion,
			ValuesPath: utilityFlags.nginxInternalValues,
		},
		model.TeleportCanonicalName: {
			Chart:      utilityFlags.teleportVersion,
			ValuesPath: utilityFlags.teleportValues,
		},
		model.PgbouncerCanonicalName: {
			Chart:      utilityFlags.pgbouncerVersion,
			ValuesPath: utilityFlags.pgbouncerValues,
		},
		model.PromtailCanonicalName: {
			Chart:      utilityFlags.promtailVersion,
			ValuesPath: utilityFlags.promtailValues,
		},
		model.RtcdCanonicalName: {
			Chart:      utilityFlags.rtcdVersion,
			ValuesPath: utilityFlags.rtcdValues,
		},
		model.KubecostCanonicalName: {
			Chart:      utilityFlags.kubecostVersion,
			ValuesPath: utilityFlags.kubecostValues,
		},
		model.NodeProblemDetectorCanonicalName: {
			Chart:      utilityFlags.nodeProblemDetectorVersion,
			ValuesPath: utilityFlags.nodeProblemDetectorValues,
		},
		model.MetricsServerCanonicalName: {
			Chart:      utilityFlags.metricsServerVersion,
			ValuesPath: utilityFlags.metricsServerValues,
		},
		model.VeleroCanonicalName: {
			Chart:      utilityFlags.veleroVersion,
			ValuesPath: utilityFlags.veleroValues,
		},
		model.CloudproberCanonicalName: {
			Chart:      utilityFlags.cloudproberVersion,
			ValuesPath: utilityFlags.cloudproberValues,
		},
	}
}
