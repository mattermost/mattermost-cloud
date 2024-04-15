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
	waitBetweenRotationsDefault    = 120  // seconds
	waitBetweenDrainsDefault       = 900  // seconds
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
	cmd.AddCommand(newCmdClusterNodegroup())
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
			flags.pgBouncerConfigChanges.addFlags(cmd)
		},
	}
	flags.addFlags(cmd)

	return cmd
}

func executeClusterCreateCmd(flags clusterCreateFlags) error {
	client := createClient(flags.clusterFlags)

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
		AMI:                    flags.ami,
		Zones:                  strings.Split(flags.zones, ","),
		AllowInstallations:     flags.allowInstallations,
		DesiredUtilityVersions: processUtilityFlags(flags.utilityFlags),
		Annotations:            flags.annotations,
		Networking:             flags.networking,
		VPC:                    flags.vpc,
		Provisioner:            model.ProvisionerKops,
		ArgocdClusterRegister:  flags.argocdRegister,
		PgBouncerConfig:        flags.GetPgBouncerConfig(),
	}

	if flags.useEKS {
		request.Provisioner = model.ProvisionerEKS
		request.ClusterRoleARN = flags.clusterRoleARN
		request.NodeRoleARN = flags.nodeRoleARN
		request.NodeGroupWithPublicSubnet = flags.nodegroupsWithPublicSubnet
		request.NodeGroupWithSecurityGroup = flags.nodegroupsWithSecurityGroup
	}

	if flags.additionalNodegroups != nil {
		if _, f := flags.additionalNodegroups[model.NodeGroupWorker]; f {
			return errors.New("worker nodegroup can only be specified with --size flag")
		}
	}

	for _, ng := range flags.nodegroupsWithPublicSubnet {
		if ng == model.NodeGroupWorker {
			continue
		}
		if _, f := flags.additionalNodegroups[ng]; !f {
			return fmt.Errorf("nodegroup %s not provided as additional nodegroups", ng)
		}
	}

	for _, ng := range flags.nodegroupsWithSecurityGroup {
		if ng == model.NodeGroupWorker {
			continue
		}
		if _, f := flags.additionalNodegroups[ng]; !f {
			return fmt.Errorf("nodegroup %s not provided as additional nodegroups", ng)
		}
	}

	err := clusterdictionary.ApplyToCreateClusterRequest(flags.size, request)
	if err != nil {
		return errors.Wrap(err, "failed to apply size values")
	}

	err = clusterdictionary.AddToCreateClusterRequest(flags.additionalNodegroups, request)
	if err != nil {
		return errors.Wrap(err, "failed to apply size values for additional nodegroups")
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
			flags.pgBouncerConfigChanges.addFlags(cmd)
		},
	}
	flags.addFlags(cmd)

	return cmd
}

func executeClusterProvisionCmd(flags clusterProvisionFlags) error {
	client := createClient(flags.clusterFlags)

	request := &model.ProvisionClusterRequest{
		Force:                  flags.reprovisionAllUtilities,
		DesiredUtilityVersions: processUtilityFlags(flags.utilityFlags),
		ArgocdClusterRegister:  flags.argocdRegister,
		PgBouncerConfig:        flags.GetPatchPgBouncerConfig(),
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
		},
	}
	flags.addFlags(cmd)

	return cmd
}

func executeClusterUpdateCmd(flags clusterUpdateFlags) error {

	client := createClient(flags.clusterFlags)

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
		},
	}
	flags.addFlags(cmd)

	return cmd
}

func executeClusterUpgradeCmd(flags clusterUpgradeFlags) error {
	client := createClient(flags.clusterFlags)

	rotatorConfig := getRotatorConfigFromFlags(flags.rotatorConfig)

	request := &model.PatchUpgradeClusterRequest{
		RotatorConfig: &rotatorConfig,
	}

	if flags.isVersionChanged {
		request.Version = &flags.version
	}
	if flags.isAmiChanged {
		request.AMI = &flags.ami
	}
	if flags.isMaxPodsPerNodeChanged {
		request.MaxPodsPerNode = &flags.maxPodsPerNode
	}

	if flags.isKMSkeyChanged {
		request.KmsKeyId = &flags.kmsKeyId
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
		},
	}
	flags.addFlags(cmd)

	return cmd
}

func executeClusterResizeCmd(flags clusterResizeFlags) error {
	client := createClient(flags.clusterFlags)

	rotatorConfig := getRotatorConfigFromFlags(flags.rotatorConfig)

	request := &model.PatchClusterSizeRequest{
		RotatorConfig: &rotatorConfig,
		NodeGroups:    flags.nodegroups,
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

	if len(flags.masterInstanceType) != 0 {
		request.MasterInstanceType = &flags.masterInstanceType
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
		},
	}
	flags.addFlags(cmd)

	return cmd
}

func executeClusterDeleteCmd(flags clusterDeleteFlags) error {
	client := createClient(flags.clusterFlags)

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
		},
	}
	flags.addFlags(cmd)

	return cmd
}

func executeClusterGetCmd(flags clusterGetFlags) error {
	client := createClient(flags.clusterFlags)

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
		},
	}
	flags.addFlags(cmd)

	return cmd
}

func executeClusterListCmd(flags clusterListFlags) error {
	client := createClient(flags.clusterFlags)

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

		var provisionerMetadata model.ProvisionerMetadata
		var masterCount int64
		var masterInstanceType string
		if cluster.Provisioner == model.ProvisionerKops && cluster.ProvisionerMetadataKops != nil {
			provisionerMetadata = cluster.ProvisionerMetadataKops.GetCommonMetadata()
			masterCount = cluster.ProvisionerMetadataKops.MasterCount
			masterInstanceType = cluster.ProvisionerMetadataKops.MasterInstanceType
		} else if cluster.Provisioner == model.ProvisionerEKS && cluster.ProvisionerMetadataEKS != nil {
			provisionerMetadata = cluster.ProvisionerMetadataEKS.GetCommonMetadata()
			masterCount = 1
			masterInstanceType = "-"
		}

		values = append(values, []string{
			cluster.ID,
			cluster.State,
			provisionerMetadata.Version,
			fmt.Sprintf("%d x %s", masterCount, masterInstanceType),
			fmt.Sprintf("%d x %s (max %d)", provisionerMetadata.NodeMinCount, provisionerMetadata.NodeInstanceType, provisionerMetadata.NodeMaxCount),
			provisionerMetadata.AMI,
			provisionerMetadata.Networking,
			provisionerMetadata.VPC,
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
			client := createClient(flags.clusterFlags)

			metadata, err := client.GetClusterUtilities(flags.cluster)
			if err != nil {
				return err
			}

			return printJSON(metadata)
		},
		PreRun: func(cmd *cobra.Command, args []string) {
			flags.clusterFlags.addFlags(cmd)
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
