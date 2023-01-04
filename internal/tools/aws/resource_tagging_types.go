// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.
//

package aws

import (
	"context"

	gt "github.com/aws/aws-sdk-go-v2/service/resourcegroupstaggingapi"
)

// ResourceGroupsTaggingAPIAPI represents the series of calls we require from the AWS SDK v2 ResourceGroupsTaggingAPI Client
type ResourceGroupsTaggingAPIAPI interface {
	GetResources(ctx context.Context, params *gt.GetResourcesInput, optFns ...func(*gt.Options)) (*gt.GetResourcesOutput, error)
}
