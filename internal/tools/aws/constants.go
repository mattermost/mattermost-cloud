// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.
//

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

	// VpcNameTagKey is the tag key used to store name of the VPC.
	VpcNameTagKey = "tag:Name"

	// VpcClusterOwnerValueNone is the tag value for VpcClusterOwnerKey when
	// there is no cluster running in the VPC.
	VpcClusterOwnerValueNone = "none"

	// VpcClusterIDTagValueNone is the tag value for VpcClusterIDTagKey when
	// there is no cluster running in the VPC.
	VpcClusterIDTagValueNone = "none"

	// DefaultDatabaseMySQLVersion is the default version of MySQL used when
	// creating databases.
	DefaultDatabaseMySQLVersion = "5.7"

	// DefaultDatabasePostgresVersion is the default version of PostgreSQL used
	// when creating databases.
	DefaultDatabasePostgresVersion = "11.7"

	// DefaultDBSubnetGroupName is the default DB subnet group name used when
	// creating DB clusters. This group name is defined by the owner of the AWS
	// accounts and can be the same across all accounts.
	// Note: This needs to be manually created before RDS databases can be used.
	DefaultDBSubnetGroupName = "mattermost-databases"

	// DatabaseTypeMySQLAurora is a MySQL database running on AWS RDS Aurora.
	DatabaseTypeMySQLAurora = "MySQL/Aurora"

	// DatabaseTypePostgresSQLAurora is a PostgreSQL database running on AWS
	// RDS Aurora.
	DatabaseTypePostgresSQLAurora = "PostgreSQL/Aurora"

	// CloudInstallationDatabaseTagKey is the common tag key for determing
	// database type.
	CloudInstallationDatabaseTagKey = "tag:MattermostCloudInstallationDatabase"

	// DefaultDBSecurityGroupTagKey is the default DB security group tag key
	// that is used to find security groups to use in configuration of the RDS
	// database.
	// Note: This needs to be manually created before RDS databases can be used.
	DefaultDBSecurityGroupTagKey = "tag:MattermostCloudInstallationDatabase"

	// DefaultDBSecurityGroupTagMySQLValue is the default DB security group tag
	// value that is used to find MySQL security groups to use in configuration
	// of the RDS database.
	// Note: This needs to be manually created before MySQL RDS databases can be
	// used.
	DefaultDBSecurityGroupTagMySQLValue = DatabaseTypeMySQLAurora

	// DefaultDBSecurityGroupTagPostgresValue is the default DB security group
	// tag value that is used to find Postgres security groups to use in
	// configuration of the RDS database.
	// Note: This needs to be manually created before MySQL RDS databases can be
	// used.
	DefaultDBSecurityGroupTagPostgresValue = DatabaseTypePostgresSQLAurora

	// DefaultDBSubnetGroupTagKey is the default DB subnet group tag key that is
	// used to find subnet groups to use in configuration of the RDS database.
	// Note: This needs to be manually created before RDS databases can be used.
	DefaultDBSubnetGroupTagKey = "tag:MattermostCloudInstallationDatabase"

	// DefaultDBSubnetGroupTagValue is the default DB subnet group tag value
	// that is used to find subnet groups to use in configuration of the RDS
	// database.
	// Note: This needs to be manually created before RDS databases can be used.
	DefaultDBSubnetGroupTagValue = DatabaseTypeMySQLAurora

	// DefaultInstallPrivateCertificatesTagKey is the default key used to find the private
	// TLS certificate ARN.
	DefaultInstallPrivateCertificatesTagKey = "tag:MattermostCloudPrivateCertificates"

	// DefaultInstallPrivateCertificatesTagValue is the default value used to find the private
	// TLS certificate ARN.
	DefaultInstallPrivateCertificatesTagValue = "true"

	// DefaultInstallCertificatesTagKey is the default key used to find the server
	// TLS certificate ARN.
	DefaultInstallCertificatesTagKey = "tag:MattermostCloudInstallationCertificates"

	// DefaultInstallCertificatesTagValue is the default value used to find the server
	// TLS certificate ARN.
	DefaultInstallCertificatesTagValue = "true"

	// DefaultCloudDNSTagKey is the default key used to find private and public hosted
	// zone IDs in AWS Route53.
	DefaultCloudDNSTagKey = "tag:MattermostCloudDNS"

	// DefaultAuditLogsCoreSecurityTagKey is the default key used to find its value which
	// has the format URL:port in which we send audit logs for each environment.
	// This URL is in Core Account and port is different for each environment
	//This tag exists in the Route53 Private hosted zones
	DefaultAuditLogsCoreSecurityTagKey = "tag:AuditLogsCoreSecurity"

	// DefaultPrivateCloudDNSTagValue is the default value used to find private hosted
	// zone ID in AWS Route53.
	DefaultPrivateCloudDNSTagValue = "private"

	// DefaultPublicCloudDNSTagValue is the default value used to find public hosted
	// zone ID in AWS Route53.
	DefaultPublicCloudDNSTagValue = "public"

	// HibernatingInstallationResourceRecordIDPrefix is a prefix given to AWS
	// route53 resource records when the installation it points to is hibernating.
	HibernatingInstallationResourceRecordIDPrefix = "[hibernating] "

	// CustomNodePolicyName is the name of the custom IAM policy that will be
	// attached in Kops Instance Profile.
	CustomNodePolicyName = "cloud-provisioning-node-policy"

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

	// rdsMySQLDefaultSchema is the default schema given to a new RDS MySQL
	// database. This is used to connect to multitenant RDS clusters to set up
	// new installation databases as needed.
	rdsMySQLDefaultSchema = "information_schema"

	// rdsPostgresDefaultSchema is the default schema given to a new RDS
	// Postgres database. This is used to connect to multitenant RDS clusters
	// to set up new installation databases as needed.
	rdsPostgresDefaultSchema = "postgres"

	// rdsDatabaseNamePrefix is the prefix value used when creating Mattermost
	// RDS database schemas.
	// Warning:
	// changing this value will break the connection to AWS resources for
	// existing installations.
	rdsDatabaseNamePrefix = "cloud_"

	// DefaultMultitenantDatabaseCounterTagKey is the default key used to
	// identify the counter tag used in RDS multitenant database clusters.
	DefaultMultitenantDatabaseCounterTagKey = "tag:Counter"

	// DefaultClusterInstallationSnapshotTagKey is used for tagging snapshots
	// of a cluster installation.
	DefaultClusterInstallationSnapshotTagKey = "tag:ClusterInstallationSnapshot"

	// DefaultAWSClientRetries supplies how many time the AWS client will
	// retry a failed call.
	DefaultAWSClientRetries = 3

	// KMSMaxTimeEncryptionKeyDeletion is the maximum number of days that
	// AWS will take to delete an encryption key.
	KMSMaxTimeEncryptionKeyDeletion = 30

	// DefaultMySQLContextTimeSeconds is the number of seconds that a SQL
	// client will take before cancel a call to the database.
	DefaultMySQLContextTimeSeconds = 15

	// DefaultRDSMultitenantDatabaseMySQLCountLimit is the maximum number of
	// schemas allowed in a MySQL multitenant RDS database cluster.
	DefaultRDSMultitenantDatabaseMySQLCountLimit = 10

	// DefaultRDSMultitenantDatabasePostgresCountLimit is the maximum number of
	// schemas allowed in a Posgres multitenant RDS database cluster.
	DefaultRDSMultitenantDatabasePostgresCountLimit = 300

	// DefaultRDSMultitenantDatabasePostgresProxySchemaLimit is the maximum number of
	// schemas created in each logical database of a proxied DB cluster.
	DefaultRDSMultitenantDatabasePostgresProxySchemaLimit = 10

	// RDSMultitenantDBClusterResourceNamePrefix identifies the prefix
	// used for naming multitenant RDS DB cluster resources.
	// For example: "rds-cluster-multitenant-00000000000000000-a0000000"
	// Warning:
	// changing this value may cause the provisioner to not find some AWS resources.
	RDSMultitenantDBClusterResourceNamePrefix = "rds-cluster-multitenant"

	// DefaultMattermostInstallationIDTagKey is the default name used for
	// tagging resources with an installation ID.
	DefaultMattermostInstallationIDTagKey = "tag:InstallationId"

	// DefaultMattermostDatabaseUsername is the default username used for
	// connecting to a Mattermost database.
	// Warning:
	// changing this value may break the connection to existing installations.
	DefaultMattermostDatabaseUsername = "mmcloud"

	// DefaultPGBouncerAuthUsername is the default username used for authorizing
	// pgbouncer connections to a shared database.
	// Warning:
	// changing this value may break the connection to existing databases.
	DefaultPGBouncerAuthUsername = "pgbouncer"

	// DefaultResourceTypeClusterRDS is the default resource type used by
	// AWS to identify an RDS cluster.
	DefaultResourceTypeClusterRDS = "rds:cluster"

	// DefaultRDSStatusAvailable identify that a RDS cluster is in available
	// state.
	DefaultRDSStatusAvailable = "available"

	// DefaultRDSEncryptionTagKey in the default tag key used for tagging
	// RDS encryption keys
	// Warning:
	// changing this value will break the connection to AWS resources for existing installations.
	DefaultRDSEncryptionTagKey = "rds-encryption-key"

	// VpcIDTagKey is the key used to identify resources belonging to a given
	// VPC.
	// Warning:
	// changing this value will break the connection to AWS resources for existing installations.
	VpcIDTagKey = "tag:VpcID"

	// FilestoreMultitenantS3TagKey is the key used to identify S3 buckets that
	// provide multitenant filestores.
	// Warning:
	// changing this value will break the connection to AWS resources for existing installations.
	FilestoreMultitenantS3TagKey = "tag:Filestore"

	// FilestoreMultitenantS3TagValue is FilestoreMultitenantS3TagKey value for
	// S3 multitenant databases.
	// Warning:
	// changing this value will break the connection to AWS resources for existing installations.
	FilestoreMultitenantS3TagValue = "Multitenant"

	// DefaultRDSMultitenantDatabaseIDTagKey is the key used to identify a
	// multitenant RDS database clusters.
	// Warning:
	// changing this value will break the connection to AWS resources for existing installations.
	DefaultRDSMultitenantDatabaseIDTagKey = "tag:MultitenantDatabaseID"

	// DefaultRDSMultitenantDatabaseTypeTagKey is the key used to identify a
	// multitenant RDS database clusters.
	// Warning:
	// changing this value will break the connection to AWS resources for existing installations.
	DefaultRDSMultitenantDatabaseTypeTagKey = "tag:DatabaseType"

	// DefaultRDSMultitenantDatabaseTypeTagValue key used to identify a
	// multitenant database cluster of type multitenant-rds.
	// Warning:
	// changing this value will break the connection to AWS resources for existing installations.
	DefaultRDSMultitenantDatabaseTypeTagValue = "multitenant-rds"

	// DefaultRDSMultitenantDatabaseDBProxyTypeTagValue key used to identify a
	// multitenant database cluster with pooled connections of type
	// multitenant-rds-dbproxy.
	// Warning:
	// changing this value will break the connection to AWS resources for existing installations.
	DefaultRDSMultitenantDatabaseDBProxyTypeTagValue = "multitenant-rds-dbproxy"

	// RDSMultitenantPurposeTagKey is the key used to identify the purpose
	// of an RDS cluster.
	// Warning:
	// changing this value will break the connection to AWS resources for existing installations.
	RDSMultitenantPurposeTagKey = "tag:Purpose"

	// RDSMultitenantPurposeTagValueProvisioning is one of the purposes of
	// an RDS cluster.
	// Warning:
	// changing this value will break the connection to AWS resources for existing installations.
	RDSMultitenantPurposeTagValueProvisioning = "provisioning"

	// RDSMultitenantOwnerTagKey identifies who owns the RDS cluster.
	// Warning:
	// changing this value will break the connection to AWS resources for existing installations.
	RDSMultitenantOwnerTagKey = "tag:Owner"

	// RDSMultitenantInstallationCounterTagKey identifies the number of
	// installations in the RDS cluster.
	// Warning:
	// changing this value will break the connection to AWS resources for existing installations.
	RDSMultitenantInstallationCounterTagKey = "tag:Counter"

	// RDSMultitenantOwnerTagValueCloudTeam identifies that cloud team
	// owns the RDS cluster.
	// Warning:
	// changing this value will break the connection to AWS resources for existing installations.
	RDSMultitenantOwnerTagValueCloudTeam = "cloud-team"

	// DefaultAWSTerraformProvisionedKey identifies wether or not a AWS
	// resource has been provisioned via Terraform.
	// Warning:
	// changing this value will break the connection to AWS resources for existing installations.
	DefaultAWSTerraformProvisionedKey = "Terraform"

	// DefaultAWSTerraformProvisionedValueTrue indicates that the AWS
	// resource has been provisioned via Terraform.
	// Warning:
	// changing this value will break the connection to AWS resources for existing installations.
	DefaultAWSTerraformProvisionedValueTrue = "true"

	// VpcSecondaryClusterIDTagKey is the tag key used to store the secondary cluster ID of the
	// cluster running in that VPC.
	VpcSecondaryClusterIDTagKey = "tag:CloudSecondaryClusterID"
)
