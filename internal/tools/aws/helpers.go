package aws

import (
	"fmt"
	"math/rand"
	"strings"
	"time"
)

// CloudID returns the standard ID used for AWS resource names. This ID is used
// to correlate installations to AWS resources.
func CloudID(id string) string {
	return cloudIDPrefix + id
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

// RDSMasterInstanceID formats the name used for RDS database instances.
func RDSMasterInstanceID(installationID string) string {
	return fmt.Sprintf("%s-master", CloudID(installationID))
}
