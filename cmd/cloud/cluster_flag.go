package main

import (
	"github.com/mattermost/mattermost-cloud/model"
	"github.com/spf13/cobra"
)

func setClusterFlags(command *cobra.Command) {
	command.PersistentFlags().String("server", defaultLocalServerAPI, "The provisioning server whose API will be queried.")
	command.PersistentFlags().StringToString("header", nil, "The extra headers to send in every API call towards the provisioning server. Accepts format: HEADER_KEY=HEADER_VALUE. Use the flag multiple times to set multiple headers.")
	command.PersistentFlags().Bool("dry-run", false, "When set to true, only print the API request without sending it.")
}

type clusterFlags struct {
	serverAddress string
	headers       map[string]string
	dryRun        bool
}

func (flags *clusterFlags) addFlags(command *cobra.Command) {
	flags.serverAddress, _ = command.Flags().GetString("server")
	flags.headers, _ = command.Flags().GetStringToString("header")
	flags.dryRun, _ = command.Flags().GetBool("dry-run")
}

type createRequestOptions struct {
	provider                    string
	version                     string
	ami                         string
	zones                       string
	allowInstallations          bool
	annotations                 []string
	networking                  string
	vpc                         string
	clusterRoleARN              string
	nodeRoleARN                 string
	useEKS                      bool
	additionalNodegroups        map[string]string
	nodegroupsWithPublicSubnet  []string
	nodegroupsWithSecurityGroup []string
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
	command.Flags().StringToStringVar(&flags.additionalNodegroups, "additional-nodegroups", nil, "Additional nodegroups to create. The key is the name of the nodegroup and the value is the size constant.")
	command.Flags().StringSliceVar(&flags.nodegroupsWithPublicSubnet, "nodegroups-with-public-subnet", nil, "Nodegroups to create with public subnet. The value is the name of the nodegroup.")
	command.Flags().StringSliceVar(&flags.nodegroupsWithSecurityGroup, "nodegroups-with-sg", nil, "Nodegroups to create with dedicated security group. The value is the name of the nodegroup.")
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
	argocdRegister             map[string]string
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
	command.Flags().StringToStringVarP(&flags.argocdRegister, "argocd-register", "", nil, "Register a cluster into Argocd format: cluster-type=customer,force=true")
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
	pgBouncerConfigOptions
	cluster string
}

func (flags *clusterCreateFlags) addFlags(command *cobra.Command) {
	flags.createRequestOptions.addFlags(command)
	flags.utilityFlags.addFlags(command)
	flags.sizeOptions.addFlags(command)
	flags.pgBouncerConfigOptions.addFlags(command)

	command.Flags().StringVar(&flags.cluster, "cluster", "", "The id of the cluster. If provided and the cluster exists the creation will be retried ignoring other parameters.")
}

type importClusterRequestOptions struct {
	secretName         string
	vpcID              string
	annotations        []string
	allowInstallations bool
}

func (flags *importClusterRequestOptions) addFlags(command *cobra.Command) {
	command.Flags().StringVar(&flags.secretName, "secret-name", "", "The name of the AWS Secret Manager secret containing the cluster kubeconfig.")
	command.Flags().StringVar(&flags.vpcID, "vpc-id", "", "Optional VPC ID for clusters running in our managed VPCs. This must be provided for allowing installations to be scheduled with our managed backends (ex. pgbouncer and bifrost).")
	command.Flags().BoolVar(&flags.allowInstallations, "allow-installations", true, "Whether the cluster will allow for new installations to be scheduled.")
	command.Flags().StringArrayVar(&flags.annotations, "annotation", []string{}, "Additional annotations for the cluster. Accepts multiple values, for example: '... --annotation abc --annotation def'")
}

type clusterImportFlags struct {
	clusterFlags
	importClusterRequestOptions
}

func (flags *clusterImportFlags) addFlags(command *cobra.Command) {
	flags.importClusterRequestOptions.addFlags(command)

	_ = command.MarkFlagRequired("secret-name")
}

type clusterProvisionFlags struct {
	clusterFlags
	utilityFlags
	pgBouncerConfigOptions
	cluster                 string
	reprovisionAllUtilities bool
}

func (flags *clusterProvisionFlags) addFlags(command *cobra.Command) {
	flags.utilityFlags.addFlags(command)
	flags.pgBouncerConfigOptions.addFlags(command)

	command.Flags().StringVar(&flags.cluster, "cluster", "", "The id of the cluster to be provisioned.")
	command.Flags().BoolVar(&flags.reprovisionAllUtilities, "reprovision-all-utilities", false, "Set to true if all utilities should be reprovisioned and not just ones with new versions")

	_ = command.MarkFlagRequired("cluster")
}

type pgBouncerConfigChanges struct {
	minPoolSizeChanged                   bool
	defaultPoolSizeChanged               bool
	reservePoolSizeChanged               bool
	maxClientConnectionsChanged          bool
	maxDatabaseConnectionsPerPoolChanged bool
	serverIdleTimeoutChanged             bool
	serverLifetimeChanged                bool
	serverResetQueryAlwaysChanged        bool
}

func (flags *pgBouncerConfigChanges) addFlags(command *cobra.Command) {
	flags.minPoolSizeChanged = command.Flags().Changed("pgbouncer-min-pool-size")
	flags.defaultPoolSizeChanged = command.Flags().Changed("pgbouncer-default-pool-size")
	flags.reservePoolSizeChanged = command.Flags().Changed("pgbouncer-reserve-pool-size")
	flags.maxClientConnectionsChanged = command.Flags().Changed("pgbouncer-max-client-connections")
	flags.maxDatabaseConnectionsPerPoolChanged = command.Flags().Changed("pgbouncer-max-connections-per-pool")
	flags.serverIdleTimeoutChanged = command.Flags().Changed("pgbouncer-server-idle-timeout")
	flags.serverLifetimeChanged = command.Flags().Changed("pgbouncer-server-lifetime")
	flags.serverResetQueryAlwaysChanged = command.Flags().Changed("pgbouncer-server-reset-query-always")
}

func (flags *pgBouncerConfigOptions) addFlags(command *cobra.Command) {
	command.Flags().Int64Var(&flags.minPoolSize, "pgbouncer-min-pool-size", model.PgBouncerDefaultMinPoolSize, "The PgBouncer config for min pool size.")
	command.Flags().Int64Var(&flags.defaultPoolSize, "pgbouncer-default-pool-size", model.PgBouncerDefaultDefaultPoolSize, "The PgBouncer config for default pool size per user.")
	command.Flags().Int64Var(&flags.reservePoolSize, "pgbouncer-reserve-pool-size", model.PgBouncerDefaultReservePoolSize, "The PgBouncer config for reserve pool size per logical database.")
	command.Flags().Int64Var(&flags.maxClientConnections, "pgbouncer-max-client-connections", model.PgBouncerDefaultMaxClientConnections, "The PgBouncer config for max client connections.")
	command.Flags().Int64Var(&flags.maxDatabaseConnectionsPerPool, "pgbouncer-max-connections-per-pool", model.PgBouncerDefaultMaxDatabaseConnectionsPerPool, "The PgBouncer config for maximum number of proxy database connections per pool (logical database).")
	command.Flags().Int64Var(&flags.serverIdleTimeout, "pgbouncer-server-idle-timeout", model.PgBouncerDefaultServerIdleTimeout, "The PgBouncer config for server idle timeout.")
	command.Flags().Int64Var(&flags.serverLifetime, "pgbouncer-server-lifetime", model.PgBouncerDefaultServerLifetime, "The PgBouncer config for server lifetime.")
	command.Flags().Int64Var(&flags.serverResetQueryAlways, "pgbouncer-server-reset-query-always", model.PgBouncerDefaultServerResetQueryAlways, "The PgBouncer config for whether server_reset_query should be run in all pooling modes.")
}

type pgBouncerConfigOptions struct {
	pgBouncerConfigChanges
	minPoolSize                   int64
	defaultPoolSize               int64
	reservePoolSize               int64
	maxClientConnections          int64
	maxDatabaseConnectionsPerPool int64
	serverIdleTimeout             int64
	serverLifetime                int64
	serverResetQueryAlways        int64
}

func (flags *pgBouncerConfigOptions) GetPgBouncerConfig() *model.PgBouncerConfig {
	return &model.PgBouncerConfig{
		MinPoolSize:                   flags.minPoolSize,
		DefaultPoolSize:               flags.defaultPoolSize,
		ReservePoolSize:               flags.reservePoolSize,
		MaxClientConnections:          flags.maxClientConnections,
		MaxDatabaseConnectionsPerPool: 20,
		ServerIdleTimeout:             30,
		ServerLifetime:                300,
		ServerResetQueryAlways:        0,
	}
}

func (flags *pgBouncerConfigOptions) GetPatchPgBouncerConfig() *model.PatchPgBouncerConfig {
	request := model.PatchPgBouncerConfig{}

	if flags.minPoolSizeChanged {
		request.MinPoolSize = &flags.minPoolSize
	}
	if flags.defaultPoolSizeChanged {
		request.DefaultPoolSize = &flags.defaultPoolSize
	}
	if flags.reservePoolSizeChanged {
		request.ReservePoolSize = &flags.reservePoolSize
	}
	if flags.maxClientConnectionsChanged {
		request.MaxClientConnections = &flags.maxClientConnections
	}
	if flags.maxDatabaseConnectionsPerPoolChanged {
		request.MaxDatabaseConnectionsPerPool = &flags.maxDatabaseConnectionsPerPool
	}
	if flags.serverIdleTimeoutChanged {
		request.ServerIdleTimeout = &flags.serverIdleTimeout
	}
	if flags.serverLifetimeChanged {
		request.ServerLifetime = &flags.serverLifetime
	}
	if flags.serverResetQueryAlwaysChanged {
		request.ServerResetQueryAlways = &flags.serverResetQueryAlways
	}

	return &request
}

func (flags *clusterPatchRequestChanges) addFlags(command *cobra.Command) {
	flags.nameChanged = command.Flags().Changed("name")
	flags.allowInstallationsChanged = command.Flags().Changed("allow-installations")
}

type clusterPatchRequestChanges struct {
	nameChanged               bool
	allowInstallationsChanged bool
}

type clusterUpdateFlags struct {
	clusterFlags
	clusterPatchRequestChanges
	cluster            string
	name               string
	allowInstallations bool
}

func (flags *clusterUpdateFlags) addFlags(command *cobra.Command) {
	command.Flags().StringVar(&flags.cluster, "cluster", "", "The id of the cluster to be updated.")
	command.Flags().StringVar(&flags.name, "name", "", "An optional name value to identify the cluster.")
	command.Flags().BoolVar(&flags.allowInstallations, "allow-installations", true, "Whether the cluster will allow for new installations to be scheduled.")

	_ = command.MarkFlagRequired("cluster")
}

func (flags *clusterUpdateFlags) GetPatchClusterRequest() *model.UpdateClusterRequest {
	request := model.UpdateClusterRequest{}

	if flags.nameChanged {
		request.Name = &flags.name
	}
	if flags.allowInstallationsChanged {
		request.AllowInstallations = &flags.allowInstallations
	}

	return &request
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
	isKMSkeyChanged         bool
}

func (flags *clusterUpgradeFlagChanged) addFlags(command *cobra.Command) {
	flags.isVersionChanged = command.Flags().Changed("version")
	flags.isAmiChanged = command.Flags().Changed("ami")
	flags.isMaxPodsPerNodeChanged = command.Flags().Changed("max-pods-per-node")
	flags.isKMSkeyChanged = command.Flags().Changed("kms-key-id")
}

type clusterUpgradeFlags struct {
	clusterFlags
	rotatorConfig
	clusterUpgradeFlagChanged
	cluster        string
	version        string
	ami            string
	maxPodsPerNode int64
	kmsKeyId       string
}

func (flags *clusterUpgradeFlags) addFlags(command *cobra.Command) {
	flags.rotatorConfig.addFlags(command)

	command.Flags().StringVar(&flags.cluster, "cluster", "", "The id of the cluster to be upgraded.")
	command.Flags().StringVar(&flags.version, "version", "", "The Kubernetes version to target. Use 'latest' or versions such as '1.16.10'.")
	command.Flags().StringVar(&flags.ami, "ami", "", "The AMI Name to use for the cluster hosts. Note: AMI ID is still supported for backwards compatibility, but fails in cases you have ARM nodegroups.")
	command.Flags().Int64Var(&flags.maxPodsPerNode, "max-pods-per-node", 0, "The maximum number of pods that can run on a single worker node.")
	command.Flags().StringVar(&flags.kmsKeyId, "kms-key-id", "", "Custom KMS key for enterprise customers.")

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
	nodegroups       []string
	sizeOptions
}

func (flags *clusterResizeFlags) addFlags(command *cobra.Command) {
	flags.rotatorConfig.addFlags(command)

	command.Flags().StringVar(&flags.cluster, "cluster", "", "The id of the cluster to be resized.")
	command.Flags().StringVar(&flags.size, "size", "", "The size constant describing the cluster")
	command.Flags().StringVar(&flags.nodeInstanceType, "size-node-instance-type", "", "The instance type describing the k8s worker nodes. Overwrites value from 'size'.")
	command.Flags().Int64Var(&flags.nodeMinCount, "size-node-min-count", 0, "The minimum number of k8s worker nodes. Overwrites value from 'size'.")
	command.Flags().Int64Var(&flags.nodeMaxCount, "size-node-max-count", 0, "The maximum number of k8s worker nodes. Overwrites value from 'size'.")
	command.Flags().StringSliceVar(&flags.nodegroups, "nodegroups", nil, "The list of nodegroups to resize. Must specify if the cluster has multiple nodegroups.")
	command.Flags().StringVar(&flags.masterInstanceType, "size-master-instance-type", "", "The instance type describing the k8s master nodes. Overwrites value from 'size'.")

	_ = command.MarkFlagRequired("cluster")
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
