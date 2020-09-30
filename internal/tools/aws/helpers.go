// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.
//

package aws

import (
	"fmt"
	"math/rand"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/mattermost/mattermost-cloud/model"
	"github.com/pkg/errors"
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

// RDSMultitenantSecretName formats the name of a secret used in a multitenant RDS database.
func RDSMultitenantSecretName(id string) string {
	return fmt.Sprintf("rds-multitenant-%s", id)
}

// MattermostMultitenantS3Name formats the name of a Mattermost S3 multitenant
// filestore bucket name.
func MattermostMultitenantS3Name(environmentName, vpcID string) string {
	return fmt.Sprintf("mattermost-cloud-%s-provisioning-%s", environmentName, vpcID)
}

// MattermostRDSDatabaseName formats the name of a Mattermost RDS database schema.
func MattermostRDSDatabaseName(installationID string) string {
	return fmt.Sprintf("%s%s", rdsDatabaseNamePrefix, installationID)
}

// MattermostMySQLConnStrings formats the connection string used for accessing a
// Mattermost database.
func MattermostMySQLConnStrings(schema, endpoint, username, password string) (string, string) {
	dbConnection := fmt.Sprintf("mysql://%s:%s@tcp(%s:3306)/%s?charset=utf8mb4,utf8&readTimeout=30s&writeTimeout=30s&tls=skip-verify",
		username, password, endpoint, schema)
	readReplicas := fmt.Sprintf("%s:%s@tcp(%s:3306)/%s?charset=utf8mb4,utf8&readTimeout=30s&writeTimeout=30s&tls=skip-verify",
		username, password, endpoint, schema)

	return dbConnection, readReplicas
}

// RDSMySQLConnString formats the connection string used by the provisioner for
// accessing a MySQL RDS cluster.
func RDSMySQLConnString(schema, endpoint, username, password string) string {
	return fmt.Sprintf("%s:%s@tcp(%s:3306)/%s?interpolateParams=true&charset=utf8mb4,utf8&readTimeout=30s&writeTimeout=30s&tls=skip-verify",
		username, password, endpoint, schema)
}

// MattermostPostgresConnStrings formats the connection strings used by Mattermost
// servers to access a PostgreSQL database.
func MattermostPostgresConnStrings(schema, endpoint, username, password string) (string, string) {
	dbConnection := fmt.Sprintf("postgres://%s:%s@%s:5432/%s?connect_timeout=10",
		username, password, endpoint, schema)
	readReplicas := fmt.Sprintf("postgres://%s:%s@%s:5432/%s?connect_timeout=10",
		username, password, endpoint, schema)

	return dbConnection, readReplicas
}

// RDSPostgresConnString formats the connection string used by the provisioner
// for accessing a Postgres RDS cluster.
func RDSPostgresConnString(schema, endpoint, username, password string) string {
	return fmt.Sprintf("postgres://%s:%s@%s:5432/%s?connect_timeout=10",
		username, password, endpoint, schema)
}

// RDSMultitenantClusterSecretDescription formats the text used for the describing a multitenant database's secret key.
func RDSMultitenantClusterSecretDescription(installationID, rdsClusterID string) string {
	return fmt.Sprintf("Used for accessing installation ID: %s database managed by RDS cluster ID: %s", installationID, rdsClusterID)
}

// getVPCForInstallation returns a single VPC that the cluster installation of
// the provided installation resides in. Installations with multiple cluster
// installations are currently not supported.
func getVPCForInstallation(installationID string, store model.InstallationDatabaseStoreInterface, client *Client) (*ec2.Vpc, error) {
	clusterInstallations, err := store.GetClusterInstallations(&model.ClusterInstallationFilter{
		PerPage:        model.AllPerPage,
		InstallationID: installationID,
		IncludeDeleted: true,
	})
	if err != nil {
		return nil, errors.Wrap(err, "failed to query cluster installations")
	}

	clusterInstallationCount := len(clusterInstallations)
	if clusterInstallationCount == 0 {
		return nil, errors.Errorf("no cluster installations found for installation ID %s", installationID)
	}
	if clusterInstallationCount != 1 {
		return nil, errors.Errorf("VPC lookups for installations with more than one cluster installation are currently not supported (found %d)", clusterInstallationCount)
	}

	vpcs, err := client.GetVpcsWithFilters([]*ec2.Filter{
		{
			Name:   aws.String(VpcClusterIDTagKey),
			Values: []*string{aws.String(clusterInstallations[0].ClusterID)},
		},
		{
			Name:   aws.String(VpcAvailableTagKey),
			Values: []*string{aws.String(VpcAvailableTagValueFalse)},
		},
	})
	if err != nil {
		return nil, errors.Wrapf(err, "failed to lookup VPC for installation %s", installationID)
	}
	if len(vpcs) != 1 {
		return nil, errors.Errorf("expected 1 VPC for cluster installation %s, but found %d", clusterInstallations[0].ClusterID, len(vpcs))
	}

	return vpcs[0], nil
}
