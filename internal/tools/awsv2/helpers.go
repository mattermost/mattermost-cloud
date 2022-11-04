// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.
//

package awsv2

// formatCloudResourceID returns the standard ID used for AWS resource names. This ID is used
// to correlate installations to AWS resources.
func formatCloudResourceID(id string) string {
	return cloudIDPrefix + id
}

// formatAsTagFilter returns the provided string as a tag filter name
func formatAsTagFilter(str string) string {
	return awsTagFilterPrefix + str
}

// formatRDSResource formats the RDS resource name with the appropriate suffix
func formatRDSResource(name string) string {
	return name + awsRDSSuffix
}
