package aws

const (
	// S3URL is the S3 URL for making bucket API calls.
	S3URL = "s3.amazonaws.com"

	// DefaultAWSRegion is the default AWS region for AWS resources.
	DefaultAWSRegion = "us-east-1"

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
)
