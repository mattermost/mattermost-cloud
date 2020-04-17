package aws

import (
	gt "github.com/aws/aws-sdk-go/service/resourcegroupstaggingapi"
)

func (a *Client) resourceTaggingGetAllResources(input gt.GetResourcesInput) ([]*gt.ResourceTagMapping, error) {
	var resources []*gt.ResourceTagMapping
	var next *string

	for {
		input.PaginationToken = next
		output, err := a.Service().resourceGroupsTagging.GetResources(&input)
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
