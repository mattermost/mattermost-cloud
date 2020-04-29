package aws

import (
	"fmt"
	"math/rand"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go/aws/awserr"
)

// MattermostMySQLConnString ...
func MattermostMySQLConnString(schema, endpoint, username, password string) string {
	return fmt.Sprintf("mysql://%s", RDSMySQLConnString(username, password, endpoint, schema))
}

// RDSMySQLConnString ...
func RDSMySQLConnString(schema, endpoint, username, password string) string {
	return fmt.Sprintf("%s:%s@tcp(%s:3306)/%s?charset=utf8mb4,utf8&readTimeout=30s&writeTimeout=30s",
		username, password, endpoint, schema)
}

// CloudID returns the standard ID used for AWS resource names. This ID is used
// to correlate installations to AWS resources.
func CloudID(id string) string {
	return cloudIDPrefix + id
}

// MattermostRDSDatabaseName formats the name of a Mattermost RDS database schema.
func MattermostRDSDatabaseName(installationID string) string {
	return fmt.Sprintf("%s%s", rdsDatabaseNamePrefix, installationID)
}

// RDSSnapshotTagValue returns the value for tagging a RDS snapshot.
func RDSSnapshotTagValue(cloudID string) string {
	return fmt.Sprintf("rds-snapshot-%s", cloudID)
}

// IAMSecretName returns the IAM Access Key secret name for a given Cloud ID.
func IAMSecretName(cloudID string) string {
	return cloudID + iamSuffix
}

// RDSSecretName returns the RDS secret name for a given Cloud ID.
func RDSSecretName(cloudID string) string {
	return cloudID + rdsSuffix
}

func trimTagPrefix(tag string) string {
	return strings.TrimLeft(tag, "tag:")
}

const passwordBytes = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ1234567890"

func newRandomPassword(length int) string {
	rand.Seed(time.Now().UnixNano())

	b := make([]byte, length)
	for i := range b {
		b[i] = passwordBytes[rand.Intn(len(passwordBytes))]
	}

	return string(b)
}

// DBSubnetGroupName formats the subnet group name used for RDS databases.
func DBSubnetGroupName(vpcID string) string {
	return fmt.Sprintf("mattermost-provisioner-db-%s", vpcID)
}

// KMSAliasNameRDS formats the alias name associated with a KMS encryption key
// used specifically for RDS databases.
func KMSAliasNameRDS(awsID string) string {
	return fmt.Sprintf("alias/%s-rds", awsID)
}

// KMSKeyDescriptionRDS formats the description of an KMS key used for encrypting a RDS cluster.
func KMSKeyDescriptionRDS(awsID string) string {
	return fmt.Sprintf("Key used for encrypting databases in the RDS cluster %v", awsID)
}

// RDSMasterInstanceID formats the name used for RDS database instances.
func RDSMasterInstanceID(installationID string) string {
	return fmt.Sprintf("%s-master", CloudID(installationID))
}

// RDSMigrationInstanceID formats the name used for migrated RDS database instances.
func RDSMigrationInstanceID(installationID string) string {
	return fmt.Sprintf("%s-migration", CloudID(installationID))
}

// IsErrorCode asserts that an AWS error has a certain code.
func IsErrorCode(err error, code string) bool {
	if err != nil {
		awsErr, ok := err.(awserr.Error)
		if ok {
			return awsErr.Code() == code
		}
	}
	return false
}

// USED FROM HERE

// RDSMultitenantClusterSecretName ...
func RDSMultitenantClusterSecretName(id string) string {
	return fmt.Sprintf("rds-multitenant-%s-cluster", id)
}

// RDSMultitenantSecretName ...
func RDSMultitenantSecretName(id string) string {
	return fmt.Sprintf("rds-multitenant-%s", id)
}

// RDSMultitenantMasterUsername ...
func RDSMultitenantMasterUsername(id string) string {
	return fmt.Sprintf("", id)
}

// RDSMultitenantUsername ...
func RDSMultitenantUsername(id string) string {
	return fmt.Sprintf("%s", id)
}

// RDSMultitenantDBClusterID ..
func RDSMultitenantDBClusterID(randomID int) string {
	return fmt.Sprintf("rds-multitenant-%d", randomID)
}
