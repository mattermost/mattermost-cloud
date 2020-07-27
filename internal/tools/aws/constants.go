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

	// rdsMySQLSchemaInformationDatabase is the schema the name given to a
	// MySQL database information's table.
	rdsMySQLSchemaInformationDatabase = "information_schema"

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

	// DefaultRDSMultitenantDatabaseCountLimit is the maximum number of
	// schemas allowed in a multitenant RDS database cluster.
	DefaultRDSMultitenantDatabaseCountLimit = 10

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
	// connectting to a Mattermost database.
	// Warning:
	// changing this value may break the connection to existing installations.
	DefaultMattermostDatabaseUsername = "mmcloud"

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

	// DefaultRDSMultitenantVPCIDTagKey is the key used to identify the VPC ID
	// used for multitenant RDS
	// database clusters.
	// Warning:
	// changing this value will break the connection to AWS resources for existing installations.
	DefaultRDSMultitenantVPCIDTagKey = "tag:VpcID"

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
)
