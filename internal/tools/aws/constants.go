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

	// rdsMySQLSchemaInformationDatabase is the schema the name given to a MySQL database information's table.
	rdsMySQLSchemaInformationDatabase = "information_schema"

	// rdsDatabaseNamePrefix is the prefix value used when creating Mattermost RDS database schemas.
	// Warning:
	// changing this value will break the connection to AWS resources for existing installations.
	rdsDatabaseNamePrefix = "cloud_"

	// DefaultMultitenantDatabaseCounterTagKey is the default key used to identify the counter tag used
	// in RDS multitenant database clusters.
	DefaultMultitenantDatabaseCounterTagKey = "tag:Counter"

	// DefaultClusterInstallationSnapshotTagKey is used for tagging snapshots of a cluster installation.
	DefaultClusterInstallationSnapshotTagKey = "tag:ClusterInstallationSnapshot"

	// DefaultAWSClientRetries supplies how many time the AWS client will retry a failed call.
	DefaultAWSClientRetries = 3

	// KMSMaxTimeEncryptionKeyDeletion is the maximum number of days that AWS will take to delete an encryption key.
	KMSMaxTimeEncryptionKeyDeletion = 30

	// DefaultMySQLDefaultContextTimeout is the number of seconds that a SQL client will take before cancel a
	// call to the database.
	DefaultMySQLDefaultContextTimeout = 30

	// DefaultRDSMultitenantDatabaseCountLimit is the maximum number of schemas allowed in a multitenant RDS database
	// cluster.
	DefaultRDSMultitenantDatabaseCountLimit = 20

	// DefaultRDSEncryptionTagKey in the default tag key used for tagging RDS encryption keys
	DefaultRDSEncryptionTagKey = "rds-encryption-key"

	// DefaultRDSMultitenantVPCIDTagKey is the key used to identify the VPC ID used for multitenant RDS
	// database clusters.
	DefaultRDSMultitenantVPCIDTagKey = "tag:RDSMultitenantVPCID"

	// DefaultRDSMultitenantDBClusterIDTagKey is the key used to identify the ID used for a multitenant RDS
	// database clusters.
	DefaultRDSMultitenantDBClusterIDTagKey = "tag:RDSMultitenantDBClusterID"

	// RDSMultitenantPurposeTagKey is the key used to identify the purpose of an RDS cluster.
	RDSMultitenantPurposeTagKey = "tag:Purpose"

	// RDSMultitenantPurposeTagValueProvisioning is one of the purposes of an RDS cluster.
	RDSMultitenantPurposeTagValueProvisioning = "provisioning"

	// RDSMultitenantAcceptingInstallationsTagKey is the key used to indicate the state of an RDS cluster.
	RDSMultitenantAcceptingInstallationsTagKey = "tag:AcceptingInstallations"

	// RDSMultitenantAcceptingInstallationsTagValueTrue indicates that the RDS cluster is taking new databases.
	RDSMultitenantAcceptingInstallationsTagValueTrue = "true"

	// RDSMultitenantOwnerTagKey identifies who owns the RDS cluster.
	RDSMultitenantOwnerTagKey = "Owner"

	// RDSMultitenantOwnerTagValueCloudTeam identifies that cloud team owns the RDS cluster.
	RDSMultitenantOwnerTagValueCloudTeam = "cloud team"
)
