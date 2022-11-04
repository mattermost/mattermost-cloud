// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.
//

package awsv2

const (
	True  = "true"
	False = "false"

	// awsTagFilterPrefix is the prefix used by AWS when filtering by tag keys
	awsTagFilterPrefix = "tag:"

	//
	// RDS
	//

	// awsRDSSuffix the suffix added to RDS resources
	awsRDSSuffix = "-rds"

	// cloudIDPrefix is the prefix value used when creating AWS resource names.
	// ⚠️ Warning:
	// changing this value will break the connection to AWS resources for
	// existing installations.
	cloudIDPrefix = "cloud-"

	// DefaultInstallPrivateCertificatesTagKey is the default key used to find the private
	// TLS certificate ARN.
	DefaultInstallPrivateCertificatesTagKey = "MattermostCloudPrivateCertificates"

	// DefaultInstallPrivateCertificatesTagValue is the default value used to find the private
	// TLS certificate ARN.
	DefaultInstallPrivateCertificatesTagValue = "true"

	// VpcClusterIDTagKey is the tag key used to store the cluster ID of the
	// cluster running in that VPC.
	VpcClusterIDTagKey = "CloudClusterID"

	// VpcAvailableTagKey is the tag key to determine if a VPC is currently in
	// use by a cluster or not.
	VpcAvailableTagKey = "Available"
)
