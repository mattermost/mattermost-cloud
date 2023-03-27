package main

import (
	"github.com/spf13/cobra"
)

func setClusterFlags(command *cobra.Command) {
	command.PersistentFlags().String("server", defaultLocalServerAPI, "The provisioning server whose API will be queried.")
	command.PersistentFlags().Bool("dry-run", false, "When set to true, only print the API request without sending it.")
}

type clusterFlags struct {
	serverAddress string
	dryRun        bool
}

func (flags *clusterFlags) addFlags(command *cobra.Command) {
	flags.serverAddress, _ = command.Flags().GetString("server")
	flags.dryRun, _ = command.Flags().GetBool("dry-run")
}

type createRequestOptions struct {
	provider                  string
	version                   string
	ami                       string
	zones                     string
	allowInstallations        bool
	annotations               []string
	networking                string
	vpc                       string
	clusterRoleARN            string
	nodeRoleARN               string
	useEKS                    bool
	additionalNodeGroups      map[string]string
	nodegroupWithPublicSubnet []string
}

func (flags *createRequestOptions) addFlags(command *cobra.Command) {
	command.Flags().StringVar(&flags.provider, "provider", "aws", "Cloud provider hosting the cluster.")
	command.Flags().StringVar(&flags.version, "version", "latest", "The Kubernetes version to target. Use 'latest' or versions such as '1.16.10'.")
	command.Flags().StringVar(&flags.ami, "ami", "", "The AMI to use for the cluster hosts.")
	command.Flags().StringVar(&flags.zones, "zones", "us-east-1a", "The zones where the cluster will be deployed. Use commas to separate multiple zones.")
	command.Flags().BoolVar(&flags.allowInstallations, "allow-installations", true, "Whether the cluster will allow for new installations to be scheduled.")
	command.Flags().StringArrayVar(&flags.annotations, "annotation", []string{}, "Additional annotations for the cluster. Accepts multiple values, for example: '... --annotation abc --annotation def'")
	command.Flags().StringVar(&flags.networking, "networking", "calico", "Networking mode to use, for example: weave, calico, canal, amazon-vpc-routed-eni")
	command.Flags().StringVar(&flags.vpc, "vpc", "", "Set to use a shared VPC")

	command.Flags().StringVar(&flags.clusterRoleARN, "cluster-role-arn", "", "AWS role ARN for cluster.")
	command.Flags().StringVar(&flags.nodeRoleARN, "node-role-arn", "", "AWS role ARN for node.")
	command.Flags().BoolVar(&flags.useEKS, "eks", false, "Create EKS cluster.")
	command.Flags().StringToStringVar(&flags.additionalNodeGroups, "additional-node-groups", nil, "Additional nodegroups to create. The key is the name of the nodegroup and the value is the size constant.")
	command.Flags().StringSliceVar(&flags.nodegroupWithPublicSubnet, "nodegroup-with-public-subnet", nil, "Nodegroups to create with public subnet. The value is the name of the nodegroup.")
}

type utilityFlags struct {
	prometheusOperatorVersion  string
	prometheusOperatorValues   string
	thanosVersion              string
	thanosValues               string
	fluentbitVersion           string
	fluentbitValues            string
	nginxVersion               string
	nginxValues                string
	nginxInternalVersion       string
	nginxInternalValues        string
	teleportVersion            string
	teleportValues             string
	pgbouncerVersion           string
	pgbouncerValues            string
	rtcdVersion                string
	rtcdValues                 string
	promtailVersion            string
	promtailValues             string
	nodeProblemDetectorVersion string
	nodeProblemDetectorValues  string
	metricsServerVersion       string
	metricsServerValues        string
	veleroVersion              string
	veleroValues               string
	cloudproberVersion         string
	cloudproberValues          string
}

func (flags *utilityFlags) addFlags(command *cobra.Command) {
	command.Flags().StringVar(&flags.prometheusOperatorVersion, "prometheus-operator-version", "", "The version of Prometheus Operator to provision. Use 'stable' to provision the latest stable version published upstream.")
	command.Flags().StringVar(&flags.prometheusOperatorValues, "prometheus-operator-values", "", "The full Git URL of the desired chart values for Prometheus Operator")
	command.Flags().StringVar(&flags.thanosVersion, "thanos-version", "", "The version of Thanos to provision. Use 'stable' to provision the latest stable version published upstream.")
	command.Flags().StringVar(&flags.thanosValues, "thanos-values", "", "The full Git URL of the desired chart values for Thanos")
	command.Flags().StringVar(&flags.fluentbitVersion, "fluentbit-version", "", "The version of Fluentbit to provision. Use 'stable' to provision the latest stable version published upstream.")
	command.Flags().StringVar(&flags.fluentbitValues, "fluentbit-values", "", "The full Git URL of the desired chart values for FluentBit")
	command.Flags().StringVar(&flags.nginxVersion, "nginx-version", "", "The version of Nginx Internal to provision. Use 'stable' to provision the latest stable version published upstream.")
	command.Flags().StringVar(&flags.nginxValues, "nginx-values", "", "The full Git URL of the desired chart values for Nginx")
	command.Flags().StringVar(&flags.nginxInternalVersion, "nginx-internal-version", "", "The version of Nginx to provision. Use 'stable' to provision the latest stable version published upstream.")
	command.Flags().StringVar(&flags.nginxInternalValues, "nginx-internal-values", "", "The full Git URL of the desired chart values for Nginx Internal")
	command.Flags().StringVar(&flags.teleportVersion, "teleport-version", "", "The version of Teleport to provision. Use 'stable' to provision the latest stable version published upstream.")
	command.Flags().StringVar(&flags.teleportValues, "teleport-values", "", "The full Git URL of the desired chart values for Teleport")
	command.Flags().StringVar(&flags.pgbouncerVersion, "pgbouncer-version", "", "The version of Pgbouncer to provision. Use 'stable' to provision the latest stable version published upstream.")
	command.Flags().StringVar(&flags.pgbouncerValues, "pgbouncer-values", "", "The full Git URL of the desired chart values for PGBouncer")
	command.Flags().StringVar(&flags.rtcdVersion, "rtcd-version", "", "The version of RTCD to provision. Use 'stable' to provision the latest stable version published upstream.")
	command.Flags().StringVar(&flags.rtcdValues, "rtcd-values", "", "The full Git URL of the desired chart values for RTCD")
	command.Flags().StringVar(&flags.promtailVersion, "promtail-version", "", "The version of Promtail to provision. Use 'stable' to provision the latest stable version published upstream.")
	command.Flags().StringVar(&flags.promtailValues, "promtail-values", "", "The full Git URL of the desired chart values for Promtail")
	command.Flags().StringVar(&flags.nodeProblemDetectorVersion, "node-problem-detector-version", "", "The version of Node Problem Detector. Use 'stable' to provision the latest stable version published upstream.")
	command.Flags().StringVar(&flags.nodeProblemDetectorValues, "node-problem-detector-values", "", "The full Git URL of the desired chart values for Node Problem Detector")
	command.Flags().StringVar(&flags.metricsServerVersion, "metrics-server-version", "", "The version of Metrics Server. Use 'stable' to provision the latest stable version published upstream.")
	command.Flags().StringVar(&flags.metricsServerValues, "metrics-server-values", "", "The full Git URL of the desired chart values for Metrics Server")
	command.Flags().StringVar(&flags.veleroVersion, "velero-version", "", "The version of Velero. Use 'stable' to provision the latest stable version published upstream.")
	command.Flags().StringVar(&flags.veleroValues, "velero-values", "", "The full Git URL of the desired chart value file's version for Velero")
	command.Flags().StringVar(&flags.cloudproberVersion, "cloudprober-version", "", "The version of Cloudprober. Use 'stable' to provision the latest stable version published upstream.")
	command.Flags().StringVar(&flags.cloudproberValues, "cloudprober-values", "", "The full Git URL of the desired chart value file's version for Cloudprober")
}

type sizeOptions struct {
	size               string
	masterInstanceType string
	masterCount        int64
	nodeInstanceType   string
	nodeCount          int64
	maxPodsPerNode     int64
}

func (flags *sizeOptions) addFlags(command *cobra.Command) {
	command.Flags().StringVar(&flags.size, "size", "SizeAlef500", "The size constant describing the master & worker nodegroups.")
	command.Flags().StringVar(&flags.masterInstanceType, "size-master-instance-type", "", "The instance type describing the k8s master nodes. Overwrites value from 'size'.")
	command.Flags().Int64Var(&flags.masterCount, "size-master-count", 0, "The number of k8s master nodes. Overwrites value from 'size'.")
	command.Flags().StringVar(&flags.nodeInstanceType, "size-node-instance-type", "", "The instance type describing the k8s worker nodes. Overwrites value from 'size'.")
	command.Flags().Int64Var(&flags.nodeCount, "size-node-count", 0, "The number of k8s worker nodes. Overwrites value from 'size'.")
	command.Flags().Int64Var(&flags.maxPodsPerNode, "max-pods-per-node", 0, "The maximum number of pods that can run on a single worker node.")
}

type clusterCreateFlags struct {
	clusterFlags
	createRequestOptions
	utilityFlags
	sizeOptions
	cluster string
}

func (flags *clusterCreateFlags) addFlags(command *cobra.Command) {
	flags.createRequestOptions.addFlags(command)
	flags.utilityFlags.addFlags(command)
	flags.sizeOptions.addFlags(command)

	command.Flags().StringVar(&flags.cluster, "cluster", "", "The id of the cluster. If provided and the cluster exists the creation will be retried ignoring other parameters.")
}

type clusterProvisionFlags struct {
	clusterFlags
	utilityFlags
	cluster                 string
	reprovisionAllUtilities bool
}

func (flags *clusterProvisionFlags) addFlags(command *cobra.Command) {
	flags.utilityFlags.addFlags(command)

	command.Flags().StringVar(&flags.cluster, "cluster", "", "The id of the cluster to be provisioned.")
	command.Flags().BoolVar(&flags.reprovisionAllUtilities, "reprovision-all-utilities", false, "Set to true if all utilities should be reprovisioned and not just ones with new versions")

	_ = command.MarkFlagRequired("cluster")
}

type clusterUpdateFlags struct {
	clusterFlags
	cluster            string
	allowInstallations bool
}

func (flags *clusterUpdateFlags) addFlags(command *cobra.Command) {
	command.Flags().StringVar(&flags.cluster, "cluster", "", "The id of the cluster to be updated.")
	command.Flags().BoolVar(&flags.allowInstallations, "allow-installations", true, "Whether the cluster will allow for new installations to be scheduled.")

	_ = command.MarkFlagRequired("cluster")
}

type rotatorConfig struct {
	useRotator              bool
	maxScaling              int
	maxDrainRetries         int
	evictGracePeriod        int
	waitBetweenRotations    int
	waitBetweenDrains       int
	waitBetweenPodEvictions int
}

func (flags *rotatorConfig) addFlags(command *cobra.Command) {
	command.Flags().BoolVar(&flags.useRotator, "use-rotator", useRotatorDefault, "Whether the cluster will be upgraded using the node rotator.")
	command.Flags().IntVar(&flags.maxScaling, "max-scaling", maxScalingDefault, "The maximum number of nodes to rotate every time. If the number is bigger than the number of nodes, then the number of nodes will be the maximum number.")
	command.Flags().IntVar(&flags.maxDrainRetries, "max-drain-retries", maxDrainRetriesDefault, "The number of times to retry a node drain.")
	command.Flags().IntVar(&flags.evictGracePeriod, "evict-grace-period", evictGracePeriodDefault, "The pod eviction grace period when draining in seconds.")
	command.Flags().IntVar(&flags.waitBetweenRotations, "wait-between-rotations", waitBetweenRotationsDefault, "Τhe time in seconds to wait between each rotation of a group of nodes.")
	command.Flags().IntVar(&flags.waitBetweenDrains, "wait-between-drains", waitBetweenDrainsDefault, "The time in seconds to wait between each node drain in a group of nodes.")
	command.Flags().IntVar(&flags.waitBetweenPodEvictions, "wait-between-pod-evictions", waitBetweenPodEvictionsDefault, "The time in seconds to wait between each pod eviction in a node drain.")
}

type clusterUpgradeFlagChanged struct {
	isVersionChanged        bool
	isAmiChanged            bool
	isMaxPodsPerNodeChanged bool
}

func (flags *clusterUpgradeFlagChanged) addFlags(command *cobra.Command) {
	flags.isVersionChanged = command.Flags().Changed("version")
	flags.isAmiChanged = command.Flags().Changed("ami")
	flags.isMaxPodsPerNodeChanged = command.Flags().Changed("max-pods-per-node")
}

type clusterUpgradeFlags struct {
	clusterFlags
	rotatorConfig
	clusterUpgradeFlagChanged
	cluster        string
	version        string
	ami            string
	maxPodsPerNode int64
}

func (flags *clusterUpgradeFlags) addFlags(command *cobra.Command) {
	flags.rotatorConfig.addFlags(command)

	command.Flags().StringVar(&flags.cluster, "cluster", "", "The id of the cluster to be upgraded.")
	command.Flags().StringVar(&flags.version, "version", "", "The Kubernetes version to target. Use 'latest' or versions such as '1.16.10'.")
	command.Flags().StringVar(&flags.ami, "ami", "", "The AMI to use for the cluster hosts.")
	command.Flags().Int64Var(&flags.maxPodsPerNode, "max-pods-per-node", 0, "The maximum number of pods that can run on a single worker node.")

	_ = command.MarkFlagRequired("cluster")
}

type clusterResizeFlags struct {
	clusterFlags
	rotatorConfig
	cluster          string
	size             string
	nodeInstanceType string
	nodeMinCount     int64
	nodeMaxCount     int64
	nodeGroups       []string
}

func (flags *clusterResizeFlags) addFlags(command *cobra.Command) {
	flags.rotatorConfig.addFlags(command)

	command.Flags().StringVar(&flags.cluster, "cluster", "", "The id of the cluster to be resized.")
	command.Flags().StringVar(&flags.size, "size", "", "The size constant describing the cluster")
	command.Flags().StringVar(&flags.nodeInstanceType, "size-node-instance-type", "", "The instance type describing the k8s worker nodes. Overwrites value from 'size'.")
	command.Flags().Int64Var(&flags.nodeMinCount, "size-node-min-count", 0, "The minimum number of k8s worker nodes. Overwrites value from 'size'.")
	command.Flags().Int64Var(&flags.nodeMaxCount, "size-node-max-count", 0, "The maximum number of k8s worker nodes. Overwrites value from 'size'.")
	command.Flags().StringSliceVar(&flags.nodeGroups, "node-groups", nil, "The list of nodegroups to resize.")

	_ = command.MarkFlagRequired("cluster")
	_ = command.MarkFlagRequired("node-groups")
}

type clusterDeleteFlags struct {
	clusterFlags
	cluster string
}

func (flags *clusterDeleteFlags) addFlags(command *cobra.Command) {
	command.Flags().StringVar(&flags.cluster, "cluster", "", "The id of the cluster to be deleted.")
	_ = command.MarkFlagRequired("cluster")
}

type clusterGetFlags struct {
	clusterFlags
	cluster string
}

func (flags *clusterGetFlags) addFlags(command *cobra.Command) {
	command.Flags().StringVar(&flags.cluster, "cluster", "", "The id of the cluster to be fetched.")
	_ = command.MarkFlagRequired("cluster")
}

type pagingFlags struct {
	page           int
	perPage        int
	includeDeleted bool
}

func (flags *pagingFlags) addFlags(command *cobra.Command) {
	command.Flags().IntVar(&flags.page, "page", 0, "The page to fetch, starting at 0.")
	command.Flags().IntVar(&flags.perPage, "per-page", 100, "The number of objects to fetch per page.")
	command.Flags().BoolVar(&flags.includeDeleted, "include-deleted", false, "Whether to include deleted objects.")
}

type tableOptions struct {
	outputToTable bool
	customCols    []string
}

func (flags *tableOptions) addFlags(command *cobra.Command) {
	command.Flags().BoolVar(&flags.outputToTable, "table", false, "Whether to display the returned output list as a table or not.")
	command.Flags().StringSliceVar(&flags.customCols, "custom-columns", []string{}, "Custom columns for table output specified with jsonpath in form <column_name>:<jsonpath>. Example: --custom-columns=ID:.ID,State:.State,VPC:.ProvisionerMetadataKops.VPC")
}

type clusterListFlags struct {
	clusterFlags
	pagingFlags
	tableOptions
	showTags bool
}

func (flags *clusterListFlags) addFlags(command *cobra.Command) {
	flags.pagingFlags.addFlags(command)
	flags.tableOptions.addFlags(command)
	command.Flags().BoolVar(&flags.showTags, "show-tags", false, "When printing, show all tags as the last column")
}

type clusterUtilitiesFlags struct {
	clusterFlags
	cluster string
}

func (flags *clusterUtilitiesFlags) addFlags(command *cobra.Command) {
	command.Flags().StringVar(&flags.cluster, "cluster", "", "The id of the cluster whose utilities are to be fetched.")
	_ = command.MarkFlagRequired("cluster")
}
