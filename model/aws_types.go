// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.
//

package model

// Certificate represents an AWS ACM certificate
type Certificate struct {
	ARN *string
}

type LaunchTemplateData struct {
	Name           string
	ClusterName    string
	AMI            string
	MaxPodsPerNode int64
	SecurityGroups []string
}
