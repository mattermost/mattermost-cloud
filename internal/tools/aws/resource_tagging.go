// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.
//

package aws

import (
	"context"

	gt "github.com/aws/aws-sdk-go-v2/service/resourcegroupstaggingapi"
	"github.com/aws/aws-sdk-go-v2/service/resourcegroupstaggingapi/types"
)

func (a *Client) resourceTaggingGetAllResources(input gt.GetResourcesInput) ([]types.ResourceTagMapping, error) {
	var resources []types.ResourceTagMapping
	var next *string

	for {
		input.PaginationToken = next
		output, err := a.Service().resourceGroupsTagging.GetResources(context.TODO(), &input)
		if err != nil {
			return nil, err
		}

		if output.ResourceTagMappingList != nil {
			resources = append(resources, output.ResourceTagMappingList...)
		}

		if output.PaginationToken != nil && len(*output.PaginationToken) > 0 {
			next = output.PaginationToken
			continue
		}

		return resources, nil
	}
}
