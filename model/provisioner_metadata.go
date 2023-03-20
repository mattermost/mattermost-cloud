// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.
//

package model

type ProvisionerMetadata struct {
	Name             string
	Version          string
	AMI              string
	NodeInstanceType string
	NodeMinCount     int64
	NodeMaxCount     int64
	MaxPodsPerNode   int64
	VPC              string
	Networking       string
}
