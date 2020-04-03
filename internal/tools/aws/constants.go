package aws

const (
	// S3URL is the S3 URL for making bucket API calls.
	S3URL = "s3.amazonaws.com"

	// DefaultAWSRegion is the default AWS region for AWS resources.
	DefaultAWSRegion = "us-east-1"

	// VpcAvailableTagKey is the tag key to determine if a VPC is currently in
	// use by a cluster or not.
	VpcAvailableTagKey = "tag:Available"

	// VpcAvailableTagValueTrue is the tag value for VpcAvailableTagKey when the
	// VPC is currently not in use by a cluster and can be claimed.
	VpcAvailableTagValueTrue = "true"

	// VpcAvailableTagValueFalse is the tag value for VpcAvailableTagKey when the
	// VPC is currently in use by a cluster and cannot be claimed.
	VpcAvailableTagValueFalse = "false"

	// VpcClusterIDTagKey is the tag key used to store the cluster ID of the
	// cluster running in that VPC.
	VpcClusterIDTagKey = "tag:CloudClusterID"

	// VpcClusterOwnerKey is the tag key  used to store the owner of the
	// cluster's human name so that the VPC's owner can be identified
	VpcClusterOwnerKey = "tag:CloudClusterOwner"

	// VpcClusterOwnerValueNone is the tag value for VpcClusterOwnerKey when
	// there is no cluster running in the VPC.
	VpcClusterOwnerValueNone = "none"

	// VpcClusterIDTagValueNone is the tag value for VpcClusterIDTagKey when
	// there is no cluster running in the VPC.
	VpcClusterIDTagValueNone = "none"

	// DefaultDBSubnetGroupName is the default DB subnet group name used when
	// creating DB clusters. This group name is defined by the owner of the AWS
	// accounts and can be the same across all accounts.
	// Note: This needs to be manually created before RDS databases can be used.
	DefaultDBSubnetGroupName = "mattermost-databases"

	// DefaultDBSecurityGroupTagKey is the default DB security group tag key
	// that is used to find security groups to use in configuration of the RDS
	// database.
	// Note: This needs to be manually created before RDS databases can be used.
	DefaultDBSecurityGroupTagKey = "tag:MattermostCloudInstallationDatabase"

	// DefaultDBSecurityGroupTagValue is the default DB security group tag value
	// that is used to find security groups to use in configuration of the RDS
	// database.
	// Note: This needs to be manually created before RDS databases can be used.
	DefaultDBSecurityGroupTagValue = "MYSQL/Aurora"

	// DefaultDBSubnetGroupTagKey is the default DB subnet group tag key that is
	// used to find subnet groups to use in configuration of the RDS database.
	// Note: This needs to be manually created before RDS databases can be used.
	DefaultDBSubnetGroupTagKey = "tag:MattermostCloudInstallationDatabase"

	// DefaultDBSubnetGroupTagValue is the default DB subnet group tag value
	// that is used to find subnet groups to use in configuration of the RDS
	// database.
	// Note: This needs to be manually created before RDS databases can be used.
	DefaultDBSubnetGroupTagValue = "MYSQL/Aurora"

	// DefaultInstallCertificatesTagKey is the default key used to find the server
	// TLS certificate ARN.
	DefaultInstallCertificatesTagKey = "tag:MattermostCloudInstallationCertificates"

	// DefaultInstallCertificatesTagValue is the default value used to find the server
	// TLS certificate ARN.
	DefaultInstallCertificatesTagValue = "true"

	// DefaultCloudDNSTagKey is the default key used to find private and public hosted
	// zone IDs in AWS Route53.
	DefaultCloudDNSTagKey = "tag:MattermostCloudDNS"

	// DefaultPrivateCloudDNSTagValue is the default value used to find private hosted zone ID
	// in AWS Route53.
	DefaultPrivateCloudDNSTagValue = "private"

	// DefaultPublicCloudDNSTagValue is the default value used to find public hosted zone ID
	// in AWS Route53.
	DefaultPublicCloudDNSTagValue = "public"

	// cloudIDPrefix is the prefix value used when creating AWS resource names.
	// Warning:
	// changing this value will break the connection to AWS resources for
	// existing installations.
	cloudIDPrefix = "cloud-"

	// iamSuffix is the suffix value used when referencing an AWS IAM secret.
	// Warning:
	// changing this value will break the connection to AWS resources for
	// existing installations.
	iamSuffix = "-iam"

	// rdsSuffix is the suffix value used when referencing an AWS RDS secret.
	// Warning:
	// changing this value will break the connection to AWS resources for
	// existing installations.
	rdsSuffix = "-rds"

	// DefaultDatabaseName is the default name used for Mattermost database.
	DefaultDatabaseName = "mattermost"

	// DefaultMySQLPort represents the default TCP port used for connecting to a MySQL database.
	DefaultMySQLPort = 3306

	// DefaultClusterInstallationSnapshotTagKey is used for tagging snapshots of a cluster installation.
	DefaultClusterInstallationSnapshotTagKey = "tag:ClusterInstallationSnapshot"

	// DefaultAWSClientRetries supplies how many time the AWS client will retry a failed call.
	DefaultAWSClientRetries = 3

	// RDSDefaultSnapshotType represents the default RDS snapshot type.
	RDSDefaultSnapshotType = "manual"

	// RDSStatusAvailable represents the status of an RDS cluster, instance or snapshot that is ready to be used.
	RDSStatusAvailable = "available"

	// RDSStatusDeleting represents the status of an RDS cluster, instance or snapshot that is being deleted.
	RDSStatusDeleting = "deleting"

	// RDSStatusCreating represents the status of an RDS cluster, instance or snapshot that is being created.
	RDSStatusCreating = "creating"

	// RDSStatusModifying represents the status of an RDS cluster, instance or snapshot that is being modified.
	RDSStatusModifying = "modifying"

	// RDSAuroraMySQLEngineName ..
	RDSAuroraMySQLEngineName = "aurora-mysql"

	// RDSAuroraDefaultMySQLVersion represents the MySQL database engine version managed by RDS.
	RDSAuroraDefaultMySQLVersion = "5.7"

	// RDSCustomParamGroupClusterName represents a custom MySQL parameter group name used by the RDS DB cluster.
	RDSCustomParamGroupClusterName = "mattermost-provisioner-rds-cluster-pg"

	// RDSCustomParamGroupName represents a custom MySQL parameter group name used by the RDS.
	RDSCustomParamGroupName = "mattermost-provisioner-rds-pg"

	// RDSDefaultInstanceClass represents the default RDS instance class used by the provisioner.
	RDSDefaultInstanceClass = "db.r5.large"

	// RDSDefaultEngineMode represents the default RDS database mode used by the provisioner.
	RDSDefaultEngineMode = "provisioned"
)
