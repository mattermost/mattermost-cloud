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
	var cf clusterFlags

	cmd := &cobra.Command{
		Use:   "cluster",
		Short: "Manipulate clusters managed by the provisioning server.",
	}
	cf.addFlags(cmd)

	cmd.AddCommand(newCmdClusterCreate(cf))
	cmd.AddCommand(newCmdClusterProvision(cf))
	cmd.AddCommand(newCmdClusterUpdate(cf))
	cmd.AddCommand(newCmdClusterUpgrade(cf))
	cmd.AddCommand(newCmdClusterResize(cf))
	cmd.AddCommand(newCmdClusterDelete(cf))
	cmd.AddCommand(newCmdClusterGet(cf))
	cmd.AddCommand(newCmdClusterList(cf))
	cmd.AddCommand(newCmdClusterUtilities(cf))

	cmd.AddCommand(newCmdClusterSizeDictionary())
	cmd.AddCommand(newCmdClusterShowStateReport())

	cmd.AddCommand(clusterAnnotationCmd)
	cmd.AddCommand(clusterInstallationCmd)

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

func newCmdClusterCreate(globalFlags clusterFlags) *cobra.Command {
	cf := newClusterCreateFlags(globalFlags)

	cmd := &cobra.Command{
		Use:   "create",
		Short: "Create a cluster.",
		RunE: func(command *cobra.Command, args []string) error {
			return executeClusterCreateCmd(cf)
		},
		PreRun: func(cmd *cobra.Command, args []string) {
			cmd.SilenceUsage = true
			return
		},
	}
	cf.addFlags(cmd)

	return cmd
}

func executeClusterCreateCmd(cf clusterCreateFlags) error {
	client := model.NewClient(cf.serverAddress)

	if cf.cluster != "" {
		err := client.RetryCreateCluster(cf.cluster)
		if err != nil {
			return errors.Wrap(err, "failed to retry cluster creation")
		}
		return nil
	}

	request := &model.CreateClusterRequest{
		Provider:               cf.provider,
		Version:                cf.version,
		KopsAMI:                cf.kopsAMI,
		Zones:                  strings.Split(cf.zones, ","),
		AllowInstallations:     cf.allowInstallations,
		DesiredUtilityVersions: processUtilityFlags(cf.utilityFlags),
		Annotations:            cf.annotations,
		Networking:             cf.networking,
		VPC:                    cf.vpc,
	}

	if cf.useEKS {
		nodeGroupsConfigRaw, err := os.ReadFile(cf.eksNodeGroupsConfig)
		if err != nil {
			return errors.Wrap(err, "failed to read node groups config")
		}
		var nodeGroupsConfig model.EKSNodeGroups
		err = json.Unmarshal(nodeGroupsConfigRaw, &nodeGroupsConfig)
		if err != nil {
			return errors.Wrap(err, "failed to unmarshal node groups config")
		}
		request.EKSConfig = &model.EKSConfig{
			ClusterRoleARN: &cf.eksRoleArn,
			NodeGroups:     nodeGroupsConfig,
		}
	}

	err := clusterdictionary.ApplyToCreateClusterRequest(cf.size, request)
	if err != nil {
		return errors.Wrap(err, "failed to apply size values")
	}

	if len(cf.masterInstanceType) != 0 {
		request.MasterInstanceType = cf.masterInstanceType
	}

	if cf.masterCount != 0 {
		request.MasterCount = cf.masterCount
	}

	if len(cf.nodeInstanceType) != 0 {
		request.NodeInstanceType = cf.nodeInstanceType
	}

	if cf.nodeCount != 0 {
		// Setting different min and max counts in currently not supported
		// with the kops create cluster flag.
		request.NodeMinCount = cf.nodeCount
		request.NodeMaxCount = cf.nodeCount
	}

	if cf.maxPodsPerNode != 0 {
		request.MaxPodsPerNode = cf.maxPodsPerNode
	}

	if cf.dryRun {
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

}

func newCmdClusterProvision(globalFlags clusterFlags) *cobra.Command {
	pf := newClusterProvisionFlags(globalFlags)

	cmd := &cobra.Command{
		Use:   "provision",
		Short: "Provision/Re-provision a cluster's k8s resources.",
		RunE: func(command *cobra.Command, args []string) error {
			return executeClusterProvisionCmd(pf)
		},
		PreRun: func(cmd *cobra.Command, args []string) {
			cmd.SilenceUsage = true
			return
		},
	}
	pf.addFlags(cmd)

	return cmd
}

func executeClusterProvisionCmd(pf clusterProvisionFlags) error {
	client := model.NewClient(pf.serverAddress)

	request := &model.ProvisionClusterRequest{
		Force:                  pf.reprovisionAllUtilities,
		DesiredUtilityVersions: processUtilityFlags(pf.utilityFlags),
	}

	if pf.dryRun {
		err := printJSON(request)
		if err != nil {
			return errors.Wrap(err, "failed to print API request")
		}

		return nil
	}

	cluster, err := client.ProvisionCluster(pf.cluster, request)
	if err != nil {
		return errors.Wrap(err, "failed to provision cluster")
	}

	err = printJSON(cluster)
	if err != nil {
		return errors.Wrap(err, "failed to print cluster response")
	}

	return nil

}

func newCmdClusterUpdate(globalFlags clusterFlags) *cobra.Command {
	uf := newClusterUpdateFlags(globalFlags)

	cmd := &cobra.Command{
		Use:   "update",
		Short: "Updates a cluster's configuration.",
		RunE: func(command *cobra.Command, args []string) error {
			return executeClusterUpdateCmd(uf)
		},
		PreRun: func(cmd *cobra.Command, args []string) {
			cmd.SilenceUsage = true
			return
		},
	}
	uf.addFlags(cmd)

	return cmd
}

func executeClusterUpdateCmd(uf clusterUpdateFlags) error {

	client := model.NewClient(uf.serverAddress)

	request := &model.UpdateClusterRequest{
		AllowInstallations: uf.allowInstallations,
	}

	if uf.dryRun {
		err := printJSON(request)
		if err != nil {
			return errors.Wrap(err, "failed to print API request")
		}

		return nil
	}

	cluster, err := client.UpdateCluster(uf.cluster, request)
	if err != nil {
		return errors.Wrap(err, "failed to update cluster")
	}

	err = printJSON(cluster)
	if err != nil {
		return errors.Wrap(err, "failed to print cluster response")
	}

	return nil

}

func newCmdClusterUpgrade(globalFlags clusterFlags) *cobra.Command {
	uf := newClusterUpgradeFlags(globalFlags)

	cmd := &cobra.Command{
		Use:   "upgrade",
		Short: "Upgrade k8s on a cluster.",
		RunE: func(command *cobra.Command, args []string) error {
			return executeClusterUpgradeCmd(uf)
		},
		PreRun: func(cmd *cobra.Command, args []string) {
			cmd.SilenceUsage = true
			uf.clusterUpgradeFlagChanged.addFlags(cmd)
			return
		},
	}
	uf.addFlags(cmd)

	return cmd
}

func executeClusterUpgradeCmd(uf clusterUpgradeFlags) error {
	client := model.NewClient(uf.serverAddress)

	rotatorConfig := getRotatorConfigFromFlags(uf.rotatorConfig)

	request := &model.PatchUpgradeClusterRequest{
		RotatorConfig: &rotatorConfig,
	}

	if uf.isVersionChanged {
		request.Version = &uf.version
	}
	if uf.isKopsAmiChanged {
		request.KopsAMI = &uf.kopsAMI
	}
	if uf.isMaxPodsPerNodeChanged {
		request.MaxPodsPerNode = &uf.maxPodsPerNode
	}
	if uf.dryRun {
		err := printJSON(request)
		if err != nil {
			return errors.Wrap(err, "failed to print API request")
		}

		return nil
	}

	cluster, err := client.UpgradeCluster(uf.cluster, request)
	if err != nil {
		return errors.Wrap(err, "failed to upgrade cluster")
	}

	err = printJSON(cluster)
	if err != nil {
		return errors.Wrap(err, "failed to print cluster response")
	}

	return nil

}

func newCmdClusterResize(globalFlags clusterFlags) *cobra.Command {
	rf := newClusterResizeFlags(globalFlags)

	cmd := &cobra.Command{
		Use:   "resize",
		Short: "Resize a k8s cluster",
		RunE: func(command *cobra.Command, args []string) error {
			return executeClusterResizeCmd(rf)
		},
		PreRun: func(cmd *cobra.Command, args []string) {
			cmd.SilenceUsage = true
			return
		},
	}
	rf.addFlags(cmd)

	return cmd
}

func executeClusterResizeCmd(rf clusterResizeFlags) error {
	client := model.NewClient(rf.serverAddress)

	rotatorConfig := getRotatorConfigFromFlags(rf.rotatorConfig)

	request := &model.PatchClusterSizeRequest{
		RotatorConfig: &rotatorConfig,
	}

	// Apply values from 'size' constant and then apply overrides.
	err := clusterdictionary.ApplyToPatchClusterSizeRequest(rf.size, request)
	if err != nil {
		return errors.Wrap(err, "failed to apply size values")
	}

	if len(rf.nodeInstanceType) != 0 {
		request.NodeInstanceType = &rf.nodeInstanceType
	}

	if rf.nodeMinCount != 0 {
		request.NodeMinCount = &rf.nodeMinCount
	}

	if rf.nodeMaxCount != 0 {
		request.NodeMaxCount = &rf.nodeMaxCount
	}

	if rf.dryRun {
		err = printJSON(request)
		if err != nil {
			return errors.Wrap(err, "failed to print API request")
		}

		return nil
	}

	cluster, err := client.ResizeCluster(rf.cluster, request)
	if err != nil {
		return errors.Wrap(err, "failed to resize cluster")
	}

	err = printJSON(cluster)
	if err != nil {
		return errors.Wrap(err, "failed to print cluster response")
	}

	return nil

}

func newCmdClusterDelete(globalFlags clusterFlags) *cobra.Command {
	df := newClusterDeleteFlags(globalFlags)

	cmd := &cobra.Command{
		Use:   "delete",
		Short: "Delete a cluster.",
		RunE: func(command *cobra.Command, args []string) error {
			return executeClusterDeleteCmd(df)
		},
		PreRun: func(cmd *cobra.Command, args []string) {
			cmd.SilenceUsage = true
			return
		},
	}
	df.addFlags(cmd)

	return cmd
}

func executeClusterDeleteCmd(df clusterDeleteFlags) error {
	client := model.NewClient(df.serverAddress)

	err := client.DeleteCluster(df.cluster)
	if err != nil {
		return errors.Wrap(err, "failed to delete cluster")
	}

	return nil
}

func newCmdClusterGet(globalFlags clusterFlags) *cobra.Command {
	gf := newClusterGetFlags(globalFlags)

	cmd := &cobra.Command{
		Use:   "get",
		Short: "Get a particular cluster.",
		RunE: func(command *cobra.Command, args []string) error {
			return executeClusterGetCmd(gf)
		},
		PreRun: func(cmd *cobra.Command, args []string) {
			cmd.SilenceUsage = true
			return
		},
	}
	gf.addFlags(cmd)

	return cmd
}

func executeClusterGetCmd(gf clusterGetFlags) error {
	client := model.NewClient(gf.serverAddress)

	cluster, err := client.GetCluster(gf.cluster)
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
}

func newCmdClusterList(globalFlags clusterFlags) *cobra.Command {
	lf := newClusterListFlags(globalFlags)

	cmd := &cobra.Command{
		Use:   "list",
		Short: "List created clusters.",
		RunE: func(command *cobra.Command, args []string) error {
			return executeClusterListCmd(lf)
		},
		PreRun: func(cmd *cobra.Command, args []string) {
			cmd.SilenceUsage = true
			return
		},
	}
	lf.addFlags(cmd)

	return cmd
}

func executeClusterListCmd(lf clusterListFlags) error {
	client := model.NewClient(lf.serverAddress)

	paging := getPagingModel(lf.pagingFlags)

	clusters, err := client.GetClusters(&model.GetClustersRequest{
		Paging: paging,
	})
	if err != nil {
		return errors.Wrap(err, "failed to query clusters")
	}

	if enabled, customCols := getTableOutputOption(lf.tableOptions); enabled {
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

func newCmdClusterUtilities(globalFlags clusterFlags) *cobra.Command {
	uf := newClusterUtilitiesFlags(globalFlags)

	cmd := &cobra.Command{
		Use:   "utilities",
		Short: "Show metadata regarding utility services running in a cluster.",
		RunE: func(command *cobra.Command, args []string) error {
			client := model.NewClient(uf.serverAddress)

			metadata, err := client.GetClusterUtilities(uf.cluster)
			if err != nil {
				return err
			}

			if err := printJSON(metadata); err != nil {
				return err
			}

			return nil
		},
		PreRun: func(cmd *cobra.Command, args []string) {
			cmd.SilenceUsage = true
			return
		},
	}
	uf.addFlags(cmd)

	return cmd
}

func newCmdClusterSizeDictionary() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "dictionary",
		Short: "Shows predefined cluster size templates.",
		RunE: func(command *cobra.Command, args []string) error {
			if err := printJSON(clusterdictionary.ValidSizes); err != nil {
				return errors.Wrap(err, "failed to print cluster dictionary")
			}
			return nil
		},
		PreRun: func(cmd *cobra.Command, args []string) {
			cmd.SilenceUsage = true
			return
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
			if err := printJSON(model.GetClusterRequestStateReport()); err != nil {
				return err
			}
			return nil
		},
		PreRun: func(cmd *cobra.Command, args []string) {
			cmd.SilenceUsage = true
			return
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
